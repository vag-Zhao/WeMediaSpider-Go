package spider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"WeMediaSpider/backend/internal/models"
	"WeMediaSpider/backend/pkg/logger"
	"WeMediaSpider/backend/pkg/utils"

	"github.com/PuerkitoBio/goquery"
	md "github.com/JohannesKaufmann/html-to-markdown"
)

// Scraper 爬虫
type Scraper struct {
	token       string
	headers     map[string]string
	client      *http.Client
	converter   *md.Converter
	cancelFunc  context.CancelFunc
	rateLimiter *utils.RateLimiter
}

// NewScraper 创建爬虫
func NewScraper(token string, headers map[string]string) *Scraper {
	// 创建智能频率限制器：最小2秒，最大5秒间隔，每分钟最多15个请求
	rateLimiter := utils.NewRateLimiter(2*time.Second, 5*time.Second, 15)

	return &Scraper{
		token:       token,
		headers:     headers,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		converter:   md.NewConverter("", true, nil),
		rateLimiter: rateLimiter,
	}
}

// SearchAccount 搜索公众号
func (s *Scraper) SearchAccount(query string) ([]models.Account, error) {
	// 应用频率限制
	s.rateLimiter.Wait()

	apiURL := "https://mp.weixin.qq.com/cgi-bin/searchbiz"

	params := url.Values{}
	params.Set("action", "search_biz")
	params.Set("token", s.token)
	params.Set("lang", "zh_CN")
	params.Set("f", "json")
	params.Set("ajax", "1")
	params.Set("query", query)
	params.Set("begin", "0")
	params.Set("count", "5")

	req, err := http.NewRequest("GET", apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头，使用随机 User-Agent
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("User-Agent", utils.GetRandomUserAgent())

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		BaseResp struct {
			Ret    int    `json:"ret"`
			ErrMsg string `json:"err_msg"`
		} `json:"base_resp"`
		List []struct {
			Fakeid      string `json:"fakeid"`
			Nickname    string `json:"nickname"`
			Alias       string `json:"alias"`
			Signature   string `json:"signature"`
			RoundHeadImg string `json:"round_head_img"`
			ServiceType int    `json:"service_type"`
		} `json:"list"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.BaseResp.Ret != 0 {
		// 记录失败
		s.rateLimiter.RecordFailure()
		return nil, fmt.Errorf("API error: %s", result.BaseResp.ErrMsg)
	}

	// 记录成功
	s.rateLimiter.RecordSuccess()

	accounts := make([]models.Account, 0, len(result.List))
	for _, item := range result.List {
		accounts = append(accounts, models.Account{
			Name:        item.Nickname,
			Fakeid:      item.Fakeid,
			Alias:       item.Alias,
			Signature:   item.Signature,
			Avatar:      item.RoundHeadImg,
			ServiceType: item.ServiceType,
		})
	}

	return accounts, nil
}

// GetArticlesList 获取文章列表
func (s *Scraper) GetArticlesList(ctx context.Context, fakeid string, page int) ([]models.Article, error) {
	// 应用频率限制
	s.rateLimiter.Wait()

	apiURL := "https://mp.weixin.qq.com/cgi-bin/appmsg"

	params := url.Values{}
	params.Set("action", "list_ex")
	params.Set("token", s.token)
	params.Set("lang", "zh_CN")
	params.Set("f", "json")
	params.Set("ajax", "1")
	params.Set("fakeid", fakeid)
	params.Set("type", "9")
	params.Set("query", "")
	params.Set("begin", fmt.Sprintf("%d", page*5))
	params.Set("count", "5")

	// 安全地截取 token 用于日志
	tokenPreview := s.token
	if len(s.token) > 10 {
		tokenPreview = s.token[:10] + "..."
	}
	logger.Infof("请求文章列表: fakeid=%s page=%d token=%s", fakeid, page, tokenPreview)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头，使用随机 User-Agent
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("User-Agent", utils.GetRandomUserAgent())

	logger.Infof("请求头: %+v", s.headers)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	logger.Infof("响应状态: %d 响应体: %s", resp.StatusCode, string(body))
	if err != nil {
		return nil, err
	}

	var result struct {
		BaseResp struct {
			Ret    int    `json:"ret"`
			ErrMsg string `json:"err_msg"`
		} `json:"base_resp"`
		AppMsgList []struct {
			Aid          string `json:"aid"`
			Title        string `json:"title"`
			Link         string `json:"link"`
			Digest       string `json:"digest"`
			UpdateTime   int64  `json:"update_time"`
			CreateTime   int64  `json:"create_time"`
			Author       string `json:"author"`
			Cover        string `json:"cover"`
		} `json:"app_msg_list"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.BaseResp.Ret != 0 {
		// 记录失败
		s.rateLimiter.RecordFailure()
		return nil, fmt.Errorf("API error: %s", result.BaseResp.ErrMsg)
	}

	// 记录成功
	s.rateLimiter.RecordSuccess()

	articles := make([]models.Article, 0, len(result.AppMsgList))
	for _, item := range result.AppMsgList {
		publishTime := time.Unix(item.UpdateTime, 0)
		articles = append(articles, models.Article{
			ID:               item.Aid,
			Title:            item.Title,
			Link:             item.Link,
			Digest:           item.Digest,
			PublishTimestamp: item.UpdateTime,
			PublishTime:      publishTime.Format("2006-01-02 15:04:05"),
			CreatedAt:        time.Now(),
		})
	}

	return articles, nil
}

// GetArticleContent 获取文章内容（带智能重试机制）
func (s *Scraper) GetArticleContent(ctx context.Context, link string) (string, error) {
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// 应用频率限制
		s.rateLimiter.Wait()

		// 如果是重试,使用较短的退避（因为频率限制器已经在自适应调整）
		if attempt > 0 {
			backoff := time.Duration(2*(attempt+1)) * time.Second
			logger.Infof("重试获取文章内容 (尝试 %d/%d)，等待 %v", attempt+1, maxRetries, backoff)
			time.Sleep(backoff)
		}

		content, err := s.getArticleContentOnce(ctx, link)
		if err == nil && strings.TrimSpace(content) != "" {
			// 成功，记录到频率限制器
			s.rateLimiter.RecordSuccess()
			return content, nil
		}

		// 失败，记录到频率限制器
		s.rateLimiter.RecordFailure()
		lastErr = err
		logger.Warnf("获取文章内容失败 (尝试 %d/%d): %v", attempt+1, maxRetries, err)
	}

	return "", fmt.Errorf("获取文章内容失败，已重试 %d 次: %w", maxRetries, lastErr)
}

// getArticleContentOnce 单次获取文章内容
func (s *Scraper) getArticleContentOnce(ctx context.Context, link string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", link, nil)
	if err != nil {
		return "", err
	}

	// 设置请求头，使用随机 User-Agent
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("User-Agent", utils.GetRandomUserAgent())
	req.Header.Set("Referer", "https://mp.weixin.qq.com/")

	logger.Infof("正在请求文章内容: %s", link)

	resp, err := s.client.Do(req)
	if err != nil {
		logger.Errorf("请求文章失败: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	logger.Infof("文章响应状态码: %d", resp.StatusCode)

	// 读取响应体用于调试
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("读取响应体失败: %v", err)
		return "", err
	}

	// 检查响应内容长度
	logger.Infof("响应体长度: %d bytes", len(bodyBytes))

	// 从字节创建 goquery 文档
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyBytes)))
	if err != nil {
		logger.Errorf("解析HTML失败: %v", err)
		return "", err
	}

	// 尝试多个可能的内容选择器
	var contentHTML string
	var content *goquery.Selection

	// 尝试 #js_content
	content = doc.Find("#js_content")
	if content.Length() > 0 {
		contentHTML, err = content.Html()
		if err == nil && strings.TrimSpace(contentHTML) != "" {
			logger.Infof("使用 #js_content 选择器成功，内容长度: %d", len(contentHTML))
		}
	}

	// 如果 #js_content 为空，尝试其他选择器
	if strings.TrimSpace(contentHTML) == "" {
		logger.Warnf("#js_content 为空，尝试其他选择器...")

		// 尝试 .rich_media_content
		content = doc.Find(".rich_media_content")
		if content.Length() > 0 {
			contentHTML, err = content.Html()
			if err == nil && strings.TrimSpace(contentHTML) != "" {
				logger.Infof("使用 .rich_media_content 选择器成功，内容长度: %d", len(contentHTML))
			}
		}
	}

	// 如果还是为空，尝试 article 标签
	if strings.TrimSpace(contentHTML) == "" {
		logger.Warnf(".rich_media_content 为空，尝试 article 标签...")
		content = doc.Find("article")
		if content.Length() > 0 {
			contentHTML, err = content.Html()
			if err == nil && strings.TrimSpace(contentHTML) != "" {
				logger.Infof("使用 article 选择器成功，内容长度: %d", len(contentHTML))
			}
		}
	}

	// 如果所有选择器都失败，记录HTML片段用于调试
	if strings.TrimSpace(contentHTML) == "" {
		logger.Errorf("所有内容选择器都失败")
		// 记录HTML的前1000个字符用于调试
		htmlPreview := string(bodyBytes)
		if len(htmlPreview) > 1000 {
			htmlPreview = htmlPreview[:1000]
		}
		logger.Errorf("HTML预览: %s", htmlPreview)
		return "", fmt.Errorf("无法提取文章内容，所有选择器都返回空")
	}

	// 处理图片：将 data-src 的值替换到 src 中，以便 Markdown 转换器能正确处理
	imageCount := 0
	content.Find("img").Each(func(i int, img *goquery.Selection) {
		// 优先使用 data-src（微信懒加载图片）
		if dataSrc, exists := img.Attr("data-src"); exists && dataSrc != "" {
			// 清理URL参数，移除 #imgIndex 等
			cleanURL := strings.Split(dataSrc, "#")[0]
			// 将真实图片URL设置到 src 属性
			img.SetAttr("src", cleanURL)
			imageCount++
			logger.Infof("处理图片 %d: %s", imageCount, cleanURL)
		} else if src, exists := img.Attr("src"); exists && strings.HasPrefix(src, "data:") {
			// 如果 src 是占位符（data: 协议），尝试从其他属性获取真实URL
			if originalSrc, exists := img.Attr("data-original-src"); exists && originalSrc != "" {
				cleanURL := strings.Split(originalSrc, "#")[0]
				img.SetAttr("src", cleanURL)
				imageCount++
				logger.Infof("处理图片 %d: %s (来自 data-original-src)", imageCount, cleanURL)
			}
		}
	})

	logger.Infof("共处理 %d 张图片", imageCount)

	// 重新获取处理后的HTML
	contentHTML, err = content.Html()
	if err != nil {
		logger.Errorf("获取处理后的HTML失败: %v", err)
		return "", err
	}

	// 转换为 Markdown
	markdown, err := s.converter.ConvertString(contentHTML)
	if err != nil {
		logger.Errorf("转换为Markdown失败: %v", err)
		return "", err
	}

	logger.Infof("成功转换为Markdown，长度: %d", len(markdown))
	return markdown, nil
}

// FilterArticlesByDate 按日期过滤文章
func (s *Scraper) FilterArticlesByDate(articles []models.Article, startDate, endDate string) []models.Article {
	// 使用本地时区解析日期
	loc := time.Local
	start, _ := time.ParseInLocation("2006-01-02", startDate, loc)
	end, _ := time.ParseInLocation("2006-01-02", endDate, loc)
	end = end.Add(24 * time.Hour) // 包含结束日期当天

	filtered := make([]models.Article, 0)
	for _, article := range articles {
		publishTime := time.Unix(article.PublishTimestamp, 0)
		if (publishTime.Equal(start) || publishTime.After(start)) && publishTime.Before(end) {
			filtered = append(filtered, article)
		}
	}

	return filtered
}

// FilterArticlesByKeyword 按关键词过滤文章（检查标题、摘要和正文内容）
func (s *Scraper) FilterArticlesByKeyword(articles []models.Article, keyword string) []models.Article {
	if keyword == "" {
		return articles
	}

	filtered := make([]models.Article, 0)
	for _, article := range articles {
		// 检查标题、摘要和正文内容
		if strings.Contains(article.Title, keyword) ||
		   strings.Contains(article.Digest, keyword) ||
		   strings.Contains(article.Content, keyword) {
			filtered = append(filtered, article)
		}
	}

	return filtered
}

// Cancel 取消爬取
func (s *Scraper) Cancel() {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
}
