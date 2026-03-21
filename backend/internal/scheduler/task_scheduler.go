package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"WeMediaSpider/backend/internal/database/models"
	appmodels "WeMediaSpider/backend/internal/models"
	"WeMediaSpider/backend/internal/repository"
	"WeMediaSpider/backend/internal/spider"
	"WeMediaSpider/backend/pkg/logger"
	"WeMediaSpider/backend/pkg/timeutil"

	"go.uber.org/zap"
)

// TaskScheduler 任务调度器
type TaskScheduler struct {
	taskRepo       repository.TaskRepository
	articleRepo    repository.ArticleRepository
	accountRepo    repository.AccountRepository
	loginManager   *spider.LoginManager
	runningTasks   map[uint]context.CancelFunc // 正在运行的任务
	mu             sync.RWMutex
	onTaskComplete func(taskID uint, status string, articles int, errMsg string) // 任务完成回调
}

// NewTaskScheduler 创建任务调度器
func NewTaskScheduler(
	taskRepo repository.TaskRepository,
	articleRepo repository.ArticleRepository,
	accountRepo repository.AccountRepository,
	loginManager *spider.LoginManager,
) *TaskScheduler {
	return &TaskScheduler{
		taskRepo:     taskRepo,
		articleRepo:  articleRepo,
		accountRepo:  accountRepo,
		loginManager: loginManager,
		runningTasks: make(map[uint]context.CancelFunc),
	}
}

// SetOnTaskComplete 设置任务完成回调
func (ts *TaskScheduler) SetOnTaskComplete(fn func(taskID uint, status string, articles int, errMsg string)) {
	ts.onTaskComplete = fn
}

// ExecuteTask 执行任务
func (ts *TaskScheduler) ExecuteTask(parentCtx context.Context, taskID uint, triggerType string) {
	// 检查任务是否已在运行
	if ts.isTaskRunning(taskID) {
		logger.Log.Warn("Task is already running, skipping", zap.Uint("taskID", taskID))
		return
	}

	// 获取任务配置
	task, err := ts.taskRepo.FindByID(taskID)
	if err != nil {
		logger.Log.Error("Failed to find task", zap.Uint("taskID", taskID), zap.Error(err))
		return
	}

	if !task.Enabled {
		logger.Log.Info("Task is disabled, skipping", zap.Uint("taskID", taskID))
		return
	}

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(parentCtx)
	ts.setTaskRunning(taskID, cancel)
	defer ts.clearTaskRunning(taskID)

	// 创建执行日志
	execLog := &models.TaskExecutionLog{
		TaskID:      taskID,
		TaskName:    task.Name,
		StartTime:   timeutil.Now(),
		Status:      "running",
		TriggerType: triggerType,
	}

	if err := ts.taskRepo.CreateExecutionLog(execLog); err != nil {
		logger.Log.Error("Failed to create execution log", zap.Error(err))
		return
	}

	// 执行爬取任务
	startTime := timeutil.Now()
	articles, execErr := ts.executeScrape(ctx, task)
	duration := time.Since(startTime).Milliseconds()

	// 更新执行日志
	endTime := timeutil.Now()
	execLog.EndTime = &endTime
	execLog.Duration = duration
	execLog.ArticlesCount = len(articles)

	if execErr != nil {
		execLog.Status = "failed"
		execLog.ErrorMessage = execErr.Error()
		task.LastRunStatus = "failed"
		task.LastRunError = execErr.Error()
		task.FailedRuns++
	} else {
		execLog.Status = "success"
		task.LastRunStatus = "success"
		task.LastRunError = ""
		task.SuccessRuns++
	}

	// 更新任务统计
	task.LastRunTime = &startTime
	task.TotalRuns++

	ts.taskRepo.UpdateExecutionLog(execLog)
	ts.taskRepo.Update(task)

	logger.Log.Info("Task completed", zap.Uint("taskID", taskID), zap.String("status", execLog.Status), zap.Int("articles", execLog.ArticlesCount), zap.Int64("durationMs", duration))

	if ts.onTaskComplete != nil {
		ts.onTaskComplete(taskID, execLog.Status, execLog.ArticlesCount, execLog.ErrorMessage)
	}
}

// executeScrape 执行爬取
func (ts *TaskScheduler) executeScrape(ctx context.Context, task *models.ScheduledTask) ([]appmodels.Article, error) {
	// 解析爬取配置
	var scrapeConfig appmodels.ScrapeConfig
	if err := json.Unmarshal([]byte(task.ScrapeConfig), &scrapeConfig); err != nil {
		return nil, fmt.Errorf("invalid scrape config: %w", err)
	}

	// 如果设置了 RecentDays，动态计算日期范围
	if scrapeConfig.RecentDays > 0 {
		now := timeutil.Now()
		scrapeConfig.EndDate = now.Format("2006-01-02")
		scrapeConfig.StartDate = now.AddDate(0, 0, -scrapeConfig.RecentDays).Format("2006-01-02")
		logger.Log.Info("Task using recentDays", zap.Uint("taskID", task.ID), zap.Int("recentDays", scrapeConfig.RecentDays), zap.String("start", scrapeConfig.StartDate), zap.String("end", scrapeConfig.EndDate))
	}

	// 检查登录状态
	if !ts.loginManager.GetStatus().IsLoggedIn {
		return nil, fmt.Errorf("not logged in")
	}

	// 创建爬虫
	scraper := spider.NewAsyncScraper(
		ts.loginManager.GetToken(),
		ts.loginManager.GetHeaders(),
		scrapeConfig.MaxWorkers,
	)

	// 执行爬取（不发送进度事件）
	articles, err := scraper.BatchScrapeAsync(ctx, scrapeConfig, nil, nil)
	if err != nil {
		return nil, err
	}

	// 保存到数据库
	if len(articles) > 0 {
		if err := ts.saveArticles(articles); err != nil {
			logger.Log.Error("Failed to save articles", zap.Error(err))
		}
	}

	return articles, nil
}

// saveArticles 保存文章到数据库
func (ts *TaskScheduler) saveArticles(articles []appmodels.Article) error {
	// 转换为数据库模型
	dbArticles := make([]*models.Article, 0, len(articles))

	for i := range articles {
		article := &articles[i]

		// 查找或创建公众号
		account, err := ts.accountRepo.FindOrCreate(article.AccountFakeid, article.AccountName)
		if err != nil {
			logger.Log.Warn("Failed to find or create account", zap.String("account", article.AccountName), zap.Error(err))
			continue
		}

		// 转换为数据库模型
		dbArticle := &models.Article{
			ArticleID:        article.ID,
			AccountID:        account.ID,
			AccountFakeid:    article.AccountFakeid,
			AccountName:      article.AccountName,
			Title:            article.Title,
			Link:             article.Link,
			Digest:           article.Digest,
			Content:          article.Content,
			PublishTime:      article.PublishTime,
			PublishTimestamp: article.PublishTimestamp,
		}

		dbArticles = append(dbArticles, dbArticle)
	}

	// 批量插入
	if len(dbArticles) > 0 {
		return ts.articleRepo.BatchCreate(dbArticles)
	}

	return nil
}

// CancelTask 取消正在运行的任务
func (ts *TaskScheduler) CancelTask(taskID uint) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if cancel, exists := ts.runningTasks[taskID]; exists {
		cancel()
		logger.Log.Info("Canceled task", zap.Uint("taskID", taskID))
		return nil
	}

	return fmt.Errorf("task %d is not running", taskID)
}

// isTaskRunning 检查任务是否正在运行
func (ts *TaskScheduler) isTaskRunning(taskID uint) bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	_, exists := ts.runningTasks[taskID]
	return exists
}

// setTaskRunning 设置任务为运行状态
func (ts *TaskScheduler) setTaskRunning(taskID uint, cancel context.CancelFunc) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.runningTasks[taskID] = cancel
}

// clearTaskRunning 清除任务运行状态
func (ts *TaskScheduler) clearTaskRunning(taskID uint) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.runningTasks, taskID)
}
