package spider

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"WeMediaSpider/backend/internal/models"
	"WeMediaSpider/backend/pkg/logger"

	"golang.org/x/sync/errgroup"
)

// AsyncScraper 异步爬虫
type AsyncScraper struct {
	*Scraper
	maxWorkers int
	mu         sync.Mutex
	progress   models.Progress
}

// NewAsyncScraper 创建异步爬虫
func NewAsyncScraper(token string, headers map[string]string, maxWorkers int) *AsyncScraper {
	return &AsyncScraper{
		Scraper:    NewScraper(token, headers),
		maxWorkers: maxWorkers,
	}
}

// BatchScrapeAsync 异步批量爬取
func (as *AsyncScraper) BatchScrapeAsync(
	ctx context.Context,
	config models.ScrapeConfig,
	progressChan chan<- models.Progress,
	statusChan chan<- models.AccountStatus,
) ([]models.Article, error) {
	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	as.cancelFunc = cancel
	defer cancel()

	// 创建 errgroup
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(as.maxWorkers)

	// 结果通道
	resultsChan := make(chan []models.Article, len(config.Accounts))

	// 为每个公众号启动 goroutine
	for _, accountName := range config.Accounts {
		accountName := accountName // 捕获循环变量

		g.Go(func() error {
			logger.Infof("🔎 开始处理公众号 [%s]", accountName)

			// 发送状态
			if statusChan != nil {
				statusChan <- models.AccountStatus{
					AccountName: accountName,
					Status:      "searching",
					Message:     "正在搜索公众号...",
				}
			}

			// 搜索公众号
			logger.Infof("🔍 正在搜索公众号 [%s]", accountName)
			accounts, err := as.SearchAccount(accountName)
			if err != nil || len(accounts) == 0 {
				logger.Errorf("❌ 未找到公众号 [%s]: %v", accountName, err)
				if statusChan != nil {
					statusChan <- models.AccountStatus{
						AccountName: accountName,
						Status:      "error",
						Message:     "未找到公众号",
					}
				}
				return nil // 不中断其他任务
			}

			account := accounts[0]
			logger.Infof("✅ 找到公众号 [%s] fakeid=%s alias=%s", accountName, account.Fakeid, account.Alias)

			// 发送状态
			if statusChan != nil {
				statusChan <- models.AccountStatus{
					AccountName: accountName,
					Status:      "fetching",
					Message:     "正在获取文章列表...",
				}
			}

			// 获取文章列表
			var allArticles []models.Article

			// 添加初始随机延迟，避免频率限制（2-5秒）
			initialDelay := time.Duration(2+len(accountName)%4) * time.Second
			logger.Infof("⏳ 等待 %v 后开始获取文章 [%s] (fakeid: %s)", initialDelay, accountName, account.Fakeid)
			time.Sleep(initialDelay)

			logger.Infof("📄 开始获取文章列表 [%s] 最大页数=%d 请求间隔=%ds", accountName, config.MaxPages, config.RequestInterval)

			for page := 0; page < config.MaxPages; page++ {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				logger.Infof("📖 正在获取第 %d/%d 页 [%s]", page+1, config.MaxPages, accountName)

				// 智能重试机制
				var articles []models.Article
				var err error
				maxRetries := 3
				for retry := 0; retry < maxRetries; retry++ {
					articles, err = as.GetArticlesList(ctx, account.Fakeid, page)
					if err == nil {
						break
					}

					// 如果是频率限制错误，等待较长时间
					if strings.Contains(err.Error(), "freq control") || strings.Contains(err.Error(), "频率") {
						waitTime := time.Duration(10*(retry+1)) * time.Second
						logger.Warnf("⚠️  遇到频率限制 [%s] page=%d，等待 %v 后重试 (%d/%d)", accountName, page+1, waitTime, retry+1, maxRetries)
						time.Sleep(waitTime)
						continue
					}

					// 其他错误，短暂等待后重试
					if retry < maxRetries-1 {
						waitTime := time.Duration(3*(retry+1)) * time.Second
						logger.Warnf("⚠️  请求失败 [%s] page=%d: %v，等待 %v 后重试 (%d/%d)", accountName, page+1, err, waitTime, retry+1, maxRetries)
						time.Sleep(waitTime)
						continue
					}

					// 最后一次重试失败
					logger.Errorf("❌ 获取文章列表失败 [%s] page=%d: %v", accountName, page+1, err)
					break
				}

				if err != nil {
					logger.Errorf("❌ 获取文章列表最终失败 [%s] page=%d: %v", accountName, page+1, err)
					break
				}

				logger.Infof("✅ 获取到文章 [%s] page=%d/%d count=%d", accountName, page+1, config.MaxPages, len(articles))

				if len(articles) == 0 {
					logger.Infof("📭 文章列表为空 [%s] page=%d，停止获取", accountName, page+1)
					break
				}

				// 打印文章标题列表
				for i, article := range articles {
					logger.Infof("  %d. %s (发布时间: %s)", i+1, article.Title, article.PublishTime)
				}

				// 设置公众号名称
				for i := range articles {
					articles[i].AccountName = accountName
					articles[i].AccountFakeid = account.Fakeid
				}

				allArticles = append(allArticles, articles...)

				// 智能请求间隔：基础延迟 + 轻微随机抖动
				if page < config.MaxPages-1 {
					baseDelay := time.Duration(config.RequestInterval) * time.Second
					// 轻微随机抖动（±10%）
					jitter := float64(baseDelay) * 0.1 * (rand.Float64() - 0.5)
					totalDelay := baseDelay + time.Duration(jitter)
					// 确保最小延迟2秒
					if totalDelay < 2*time.Second {
						totalDelay = 2 * time.Second
					}
					logger.Infof("⏳ 等待 %v 后继续获取下一页 [%s]", totalDelay, accountName)
					time.Sleep(totalDelay)
				}
			}

			logger.Infof("📊 总共获取文章 [%s] count=%d", accountName, len(allArticles))

			// 打印所有文章链接
			logger.Infof("🔗 文章链接列表 [%s]:", accountName)
			for i, article := range allArticles {
				logger.Infof("  %d. %s", i+1, article.Link)
				logger.Infof("     标题: %s", article.Title)
			}

			// 日期过滤
			if config.StartDate != "" && config.EndDate != "" {
				beforeFilter := len(allArticles)
				allArticles = as.FilterArticlesByDate(allArticles, config.StartDate, config.EndDate)
				logger.Infof("📅 日期过滤 [%s] 范围=%s ~ %s 过滤前=%d 过滤后=%d", accountName, config.StartDate, config.EndDate, beforeFilter, len(allArticles))

				// 如果有文章被过滤掉，显示过滤后的文章列表
				if beforeFilter != len(allArticles) {
					logger.Infof("📋 过滤后的文章列表 [%s]:", accountName)
					for i, article := range allArticles {
						logger.Infof("  %d. %s - %s", i+1, article.Title, article.PublishTime)
					}
				}
			}

			// 发送获取到的文章数
			if statusChan != nil {
				statusChan <- models.AccountStatus{
					AccountName:  accountName,
					Status:       "fetching",
					Message:      fmt.Sprintf("已获取 %d 篇文章", len(allArticles)),
					ArticleCount: len(allArticles),
				}
			}

			// 获取文章内容
			if config.IncludeContent {
				if statusChan != nil {
					statusChan <- models.AccountStatus{
						AccountName:  accountName,
						Status:       "content",
						Message:      fmt.Sprintf("正在获取文章内容 (并发数: %d)...", as.maxWorkers),
						ArticleCount: len(allArticles),
					}
				}

				logger.Infof("开始获取文章内容 [%s] 总数=%d 并发数=%d", accountName, len(allArticles), as.maxWorkers)

				// 使用 errgroup 并发获取文章内容
				contentGroup, contentCtx := errgroup.WithContext(ctx)
				contentGroup.SetLimit(as.maxWorkers)

				// 进度计数器
				var contentMu sync.Mutex
				contentProgress := 0

				for i := range allArticles {
					i := i // 捕获循环变量

					contentGroup.Go(func() error {
						select {
						case <-contentCtx.Done():
							return contentCtx.Err()
						default:
						}

						logger.Infof("正在获取文章内容 [%s] (%d/%d): %s", accountName, i+1, len(allArticles), allArticles[i].Title)

						content, err := as.GetArticleContent(contentCtx, allArticles[i].Link)
						if err != nil {
							logger.Warnf("获取文章内容失败 [%s] (%d/%d): %s - %v", accountName, i+1, len(allArticles), allArticles[i].Title, err)
						} else {
							allArticles[i].Content = content
							logger.Infof("成功获取文章内容 [%s] (%d/%d): %s (长度: %d)", accountName, i+1, len(allArticles), allArticles[i].Title, len(content))
						}

						// 更新进度
						contentMu.Lock()
						contentProgress++
						currentProgress := contentProgress
						contentMu.Unlock()

						// 发送进度
						if progressChan != nil {
							progressChan <- models.Progress{
								Type:    models.ProgressTypeContent,
								Current: currentProgress,
								Total:   len(allArticles),
								Message: fmt.Sprintf("正在获取文章内容 [%s] (%d/%d)", accountName, currentProgress, len(allArticles)),
							}
						}

						// 轻微延迟（频率限制器已经在控制）
						time.Sleep(time.Duration(config.RequestInterval) * time.Second)
						return nil
					})
				}

				// 等待所有内容获取完成
				if err := contentGroup.Wait(); err != nil {
					logger.Errorf("获取文章内容过程中出错 [%s]: %v", accountName, err)
				}

				logger.Infof("文章内容获取完成 [%s] 成功=%d 总数=%d", accountName, contentProgress, len(allArticles))
			}

			// 关键词过滤（在获取正文内容之后进行，以便搜索全文）
			if config.KeywordFilter != "" {
				beforeFilter := len(allArticles)
				allArticles = as.FilterArticlesByKeyword(allArticles, config.KeywordFilter)
				logger.Infof("🔍 关键词过滤 [%s] 关键词='%s' 过滤前=%d 过滤后=%d", accountName, config.KeywordFilter, beforeFilter, len(allArticles))

				// 如果有文章被过滤掉，显示过滤后的文章列表
				if beforeFilter != len(allArticles) {
					logger.Infof("📋 关键词过滤后的文章列表 [%s]:", accountName)
					for i, article := range allArticles {
						logger.Infof("  %d. %s", i+1, article.Title)
					}
				}
			}

			// 发送完成状态
			if statusChan != nil {
				statusChan <- models.AccountStatus{
					AccountName:  accountName,
					Status:       "completed",
					Message:      "爬取完成",
					ArticleCount: len(allArticles),
				}
			}

			logger.Infof("🎉 公众号 [%s] 爬取完成！最终文章数=%d", accountName, len(allArticles))

			resultsChan <- allArticles
			return nil
		})
	}

	// 等待所有任务完成
	go func() {
		g.Wait()
		close(resultsChan)
		logger.Infof("🏁 所有公众号爬取任务已完成")
	}()

	// 收集结果
	logger.Infof("📦 开始收集所有公众号的文章结果...")
	var allResults []models.Article
	accountCount := 0
	for articles := range resultsChan {
		accountCount++
		allResults = append(allResults, articles...)
		logger.Infof("📥 收集第 %d 个公众号的文章，本次=%d 累计=%d", accountCount, len(articles), len(allResults))
	}

	logger.Infof("✨ 所有文章收集完成！总计 %d 个公众号，%d 篇文章", accountCount, len(allResults))

	// 检查错误
	if err := g.Wait(); err != nil {
		logger.Errorf("❌ 爬取过程中出现错误: %v", err)
		return allResults, err
	}

	return allResults, nil
}

// GetProgress 获取进度
func (as *AsyncScraper) GetProgress() models.Progress {
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.progress
}
