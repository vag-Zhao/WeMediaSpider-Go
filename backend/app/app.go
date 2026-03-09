package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"WeMediaSpider/backend/internal/cache"
	"WeMediaSpider/backend/internal/config"
	"WeMediaSpider/backend/internal/export"
	"WeMediaSpider/backend/internal/models"
	"WeMediaSpider/backend/internal/spider"
	"WeMediaSpider/backend/internal/storage"
	"WeMediaSpider/backend/pkg/logger"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 应用结构
type App struct {
	ctx              context.Context
	loginManager     *spider.LoginManager
	scraper          *spider.AsyncScraper
	configManager    *config.Manager
	cacheManager     *cache.Manager
	imageDownloader  *spider.ImageDownloader
	dataManager      *config.DataManager
	storageManager   *storage.Manager
}

// NewApp 创建应用实例
func NewApp() *App {
	// 初始化日志
	logger.Init()

	// 创建缓存管理器
	cacheManager, err := cache.NewManager(96) // 96小时过期
	if err != nil {
		logger.Errorf("Failed to create cache manager: %v", err)
	}

	return &App{
		loginManager:   spider.NewLoginManager(),
		configManager:  config.NewManager(),
		cacheManager:   cacheManager,
		dataManager:    config.NewDataManager(),
		storageManager: storage.NewManager(),
	}
}

// Startup 应用启动时调用
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	logger.Info("Application started")
}

// Shutdown 应用关闭时调用
func (a *App) Shutdown(ctx context.Context) {
	if a.cacheManager != nil {
		a.cacheManager.Close()
	}
	logger.Info("Application shutdown")
}

// ============================================================
// 登录相关
// ============================================================

// Login 执行登录
func (a *App) Login() error {
	return a.loginManager.Login(a.ctx)
}

// Logout 退出登录
func (a *App) Logout() error {
	return a.loginManager.Logout()
}

// GetLoginStatus 获取登录状态
func (a *App) GetLoginStatus() models.LoginStatus {
	return a.loginManager.GetStatus()
}

// ClearLoginCache 清除登录缓存
func (a *App) ClearLoginCache() error {
	return a.loginManager.ClearCache()
}

// ExportCredentials 导出加密的登录凭证到文件
func (a *App) ExportCredentials() (string, error) {
	// 导出凭证数据
	data, err := a.loginManager.ExportCredentials()
	if err != nil {
		return "", err
	}

	// 打开保存文件对话框
	filepath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出登录凭证",
		DefaultFilename: fmt.Sprintf("wemedia_credentials_%d.zgswx", time.Now().Unix()),
		Filters: []runtime.FileFilter{
			{
				DisplayName: "WeMediaSpider 凭证文件 (*.zgswx)",
				Pattern:     "*.zgswx",
			},
		},
	})

	if err != nil || filepath == "" {
		return "", fmt.Errorf("用户取消操作")
	}

	// 写入文件
	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return "", fmt.Errorf("保存文件失败: %w", err)
	}

	logger.Infof("凭证已导出到: %s", filepath)
	return filepath, nil
}

// ImportCredentials 从文件导入加密的登录凭证
func (a *App) ImportCredentials() error {
	// 打开文件选择对话框
	filepath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "导入登录凭证",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "WeMediaSpider 凭证文件 (*.zgswx)",
				Pattern:     "*.zgswx",
			},
		},
	})

	if err != nil || filepath == "" {
		return fmt.Errorf("用户取消操作")
	}

	// 读取文件
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// 导入凭证
	if err := a.loginManager.ImportCredentials(data); err != nil {
		return err
	}

	logger.Infof("凭证已从文件导入: %s", filepath)
	return nil
}

// ============================================================
// 爬取相关
// ============================================================

// SearchAccount 搜索公众号
func (a *App) SearchAccount(query string) ([]models.Account, error) {
	// 确保已登录
	status := a.loginManager.GetStatus()
	if !status.IsLoggedIn {
		return nil, nil
	}

	// 创建临时爬虫
	scraper := spider.NewScraper(
		a.loginManager.GetToken(),
		a.loginManager.GetHeaders(),
	)

	return scraper.SearchAccount(query)
}

// StartScrape 开始爬取
func (a *App) StartScrape(config models.ScrapeConfig) ([]models.Article, error) {
	// 创建异步爬虫
	a.scraper = spider.NewAsyncScraper(
		a.loginManager.GetToken(),
		a.loginManager.GetHeaders(),
		config.MaxWorkers,
	)

	// 创建进度通道
	progressChan := make(chan models.Progress, 100)
	statusChan := make(chan models.AccountStatus, 100)

	// 启动进度发送协程
	go func() {
		for {
			select {
			case progress, ok := <-progressChan:
				if !ok {
					return
				}
				runtime.EventsEmit(a.ctx, "scrape:progress", progress)
			case status, ok := <-statusChan:
				if !ok {
					return
				}
				runtime.EventsEmit(a.ctx, "scrape:status", status)
			case <-a.ctx.Done():
				return
			}
		}
	}()

	// 执行爬取
	articles, err := a.scraper.BatchScrapeAsync(a.ctx, config, progressChan, statusChan)

	// 关闭通道
	close(progressChan)
	close(statusChan)

	// 发送完成事件
	if err == nil {
		// 自动保存数据到配置目录
		savedPath, saveErr := a.storageManager.AutoSave(articles)
		if saveErr != nil {
			logger.Errorf("自动保存数据失败: %v", saveErr)
		} else {
			logger.Infof("数据已自动保存: %s", savedPath)
		}

		runtime.EventsEmit(a.ctx, "scrape:completed", map[string]interface{}{
			"total":     len(articles),
			"savedPath": savedPath,
		})
		// 保存应用数据
		if len(articles) > 0 {
			if err := a.dataManager.UpdateArticleStats(articles); err != nil {
				logger.Errorf("Failed to update app data: %v", err)
			}
		}
	} else {
		// 检查是否是取消操作导致的错误
		errMsg := err.Error()
		if errMsg != "context canceled" && !strings.Contains(errMsg, "canceled") {
			// 只有非取消的错误才发送错误事件
			runtime.EventsEmit(a.ctx, "scrape:error", map[string]string{
				"error": errMsg,
			})
		}
	}

	return articles, err
}

// CancelScrape 取消爬取
func (a *App) CancelScrape() {
	if a.scraper != nil {
		a.scraper.Cancel()
	}
}

// ============================================================
// 配置相关
// ============================================================

// LoadConfig 加载配置
func (a *App) LoadConfig() (models.Config, error) {
	return a.configManager.Load()
}

// SaveConfig 保存配置
func (a *App) SaveConfig(config models.Config) error {
	return a.configManager.Save(config)
}

// GetDefaultConfig 获取默认配置
func (a *App) GetDefaultConfig() models.Config {
	return a.configManager.GetDefault()
}

// ============================================================
// 文件系统相关
// ============================================================

// SelectDirectory 选择目录
func (a *App) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择输出目录",
	})
}

// SelectSaveFile 选择保存文件
func (a *App) SelectSaveFile(defaultFilename string, filters []runtime.FileFilter) (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "保存文件",
		DefaultFilename: defaultFilename,
		Filters:         filters,
	})
}

// ============================================================
// 导出相关
// ============================================================

// ExportArticles 导出文章
func (a *App) ExportArticles(articles []models.Article, format string, filename string) error {
	logger.Infof("Exporting %d articles to %s format: %s", len(articles), format, filename)
	exporter := export.GetExporter(format)
	err := exporter.Export(articles, filename)
	if err != nil {
		logger.Errorf("Export failed: %v", err)
	} else {
		logger.Infof("Export completed successfully")
		// 保存导出统计
		if err := a.dataManager.IncrementExports(); err != nil {
			logger.Errorf("Failed to update export stats: %v", err)
		}
	}
	return err
}

// ============================================================
// 缓存相关
// ============================================================

// ClearCache 清除缓存
func (a *App) ClearCache() error {
	if a.cacheManager == nil {
		return nil
	}
	logger.Info("Clearing all cache")
	return a.cacheManager.ClearAll()
}

// ClearExpiredCache 清除过期缓存
func (a *App) ClearExpiredCache() error {
	if a.cacheManager == nil {
		return nil
	}
	logger.Info("Clearing expired cache")
	return a.cacheManager.ClearExpired()
}

// GetCacheStats 获取缓存统计
func (a *App) GetCacheStats() (map[string]int, error) {
	if a.cacheManager == nil {
		return map[string]int{}, nil
	}
	return a.cacheManager.GetStats()
}

// ============================================================
// 工具方法
// ============================================================

// GetAppVersion 获取应用版本
func (a *App) GetAppVersion() string {
	return "1.0.0"
}

// VersionInfo 版本信息
type VersionInfo struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	HasUpdate      bool   `json:"hasUpdate"`
	UpdateURL      string `json:"updateUrl"`
	ReleaseNotes   string `json:"releaseNotes"`
}

// CheckForUpdates 检查更新
func (a *App) CheckForUpdates() (VersionInfo, error) {
	currentVersion := a.GetAppVersion()

	// 发起 HTTP 请求获取最新版本
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://api.github.com/repos/yourusername/WeMediaSpider/releases/latest")
	if err != nil {
		logger.Warnf("检查更新失败: %v", err)
		return VersionInfo{
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
			HasUpdate:      false,
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.Warnf("检查更新失败，状态码: %d", resp.StatusCode)
		return VersionInfo{
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
			HasUpdate:      false,
		}, nil
	}

	var release struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		logger.Warnf("解析版本信息失败: %v", err)
		return VersionInfo{
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
			HasUpdate:      false,
		}, nil
	}

	// 移除 tag_name 中的 'v' 前缀
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	// 比较版本号
	hasUpdate := compareVersions(latestVersion, currentVersion) > 0

	logger.Infof("当前版本: %s, 最新版本: %s, 有更新: %v", currentVersion, latestVersion, hasUpdate)

	return VersionInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		HasUpdate:      hasUpdate,
		UpdateURL:      release.HTMLURL,
		ReleaseNotes:   release.Body,
	}, nil
}

// compareVersions 比较两个版本号，返回 1 表示 v1 > v2，-1 表示 v1 < v2，0 表示相等
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			n1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			n2, _ = strconv.Atoi(parts2[i])
		}

		if n1 > n2 {
			return 1
		} else if n1 < n2 {
			return -1
		}
	}

	return 0
}

// ============================================================
// 应用数据相关
// ============================================================

// GetAppData 获取应用数据
func (a *App) GetAppData() (models.AppData, error) {
	return a.dataManager.LoadData()
}

// UpdateAppData 更新应用数据
func (a *App) UpdateAppData(articles []models.Article) error {
	return a.dataManager.UpdateArticleStats(articles)
}

// ============================================================
// 图片下载相关
// ============================================================

// ExtractArticleImages 从文章内容中提取图片
func (a *App) ExtractArticleImages(content string) []spider.ImageInfo {
	downloader := spider.NewImageDownloader(a.loginManager.GetHeaders())
	return downloader.ExtractImages(content)
}

// BatchDownloadImages 批量下载图片
func (a *App) BatchDownloadImages(images []spider.ImageInfo, baseDir string, maxWorkers int) error {
	// 创建新的下载器
	a.imageDownloader = spider.NewImageDownloader(a.loginManager.GetHeaders())

	// 创建进度通道
	progressChan := make(chan spider.ImageDownloadProgress, 100)

	// 启动进度发送协程
	go func() {
		for progress := range progressChan {
			runtime.EventsEmit(a.ctx, "image:progress", progress)
		}
	}()

	// 执行下载
	err := a.imageDownloader.DownloadImagesWithProgress(images, baseDir, maxWorkers, progressChan)

	// 发送完成事件
	if err == nil {
		runtime.EventsEmit(a.ctx, "image:completed", map[string]interface{}{
			"total": len(images),
		})
		// 保存图片下载统计
		if err := a.dataManager.IncrementImages(len(images)); err != nil {
			logger.Errorf("Failed to update image stats: %v", err)
		}
	} else {
		// 检查是否是取消操作导致的错误
		errMsg := err.Error()
		if errMsg != "context canceled" && !strings.Contains(errMsg, "canceled") {
			runtime.EventsEmit(a.ctx, "image:error", map[string]string{
				"error": errMsg,
			})
		}
	}

	return err
}

// CancelImageDownload 取消图片下载
func (a *App) CancelImageDownload() {
	if a.imageDownloader != nil {
		a.imageDownloader.Cancel()
	}
}

// ============================================================
// 数据管理相关
// ============================================================

// ListDataFiles 列出所有保存的数据文件
func (a *App) ListDataFiles() ([]storage.DataFileInfo, error) {
	return a.storageManager.ListDataFiles()
}

// LoadDataFile 加载指定的数据文件
func (a *App) LoadDataFile(filepath string) ([]models.Article, error) {
	return a.storageManager.LoadData(filepath)
}

// DeleteDataFile 删除指定的数据文件
func (a *App) DeleteDataFile(filepath string) error {
	return a.storageManager.DeleteData(filepath)
}

// GetDataDirectory 获取数据目录路径
func (a *App) GetDataDirectory() string {
	return a.storageManager.GetDataDir()
}

// OpenDataFileDialog 打开数据文件选择对话框（自动定位到数据目录）
func (a *App) OpenDataFileDialog() (string, error) {
	dataDir := a.storageManager.GetDataDir()

	filepath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "选择数据文件",
		DefaultDirectory:     dataDir,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "JSON 数据文件 (*.json)",
				Pattern:     "*.json",
			},
		},
	})

	if err != nil || filepath == "" {
		return "", fmt.Errorf("用户取消操作")
	}

	return filepath, nil
}
