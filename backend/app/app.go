package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"WeMediaSpider/backend/internal/autostart"
	"WeMediaSpider/backend/internal/cache"
	"WeMediaSpider/backend/internal/config"
	"WeMediaSpider/backend/internal/database"
	dbmodels "WeMediaSpider/backend/internal/database/models"
	"WeMediaSpider/backend/internal/export"
	"WeMediaSpider/backend/internal/models"
	"WeMediaSpider/backend/internal/repository"
	"WeMediaSpider/backend/internal/spider"
	"WeMediaSpider/backend/internal/storage"
	"WeMediaSpider/backend/internal/tray"
	"WeMediaSpider/backend/pkg/logger"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 应用结构
type App struct {
	ctx                 context.Context
	loginManager        *spider.LoginManager
	scraper             *spider.AsyncScraper
	configManager       *config.Manager
	systemConfigManager *config.SystemConfigManager
	cacheManager        *cache.Manager
	imageDownloader     *spider.ImageDownloader
	db                  *database.Database
	articleRepo         repository.ArticleRepository
	accountRepo         repository.AccountRepository
	statsRepo           repository.StatsRepository
	trayManager         *tray.Manager
	autostartManager    *autostart.Manager
	closeToTray         bool   // 关闭到托盘
	rememberChoice      bool   // 记住用户选择
	updateIgnoredDate   string // 更新忽略日期
	forceQuit           bool   // 强制退出标志
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

	// 创建自启动管理器
	autostartManager, err := autostart.NewManager()
	if err != nil {
		logger.Errorf("Failed to create autostart manager: %v", err)
	}

	// 创建系统配置管理器
	systemConfigManager, err := config.NewSystemConfigManager()
	if err != nil {
		logger.Errorf("Failed to create system config manager: %v", err)
	}

	// 加载系统配置
	systemConfig := config.SystemConfig{
		CloseToTray:       true,
		RememberChoice:    false,
		UpdateIgnoredDate: "",
	}
	if systemConfigManager != nil {
		loadedConfig, err := systemConfigManager.Load()
		if err == nil {
			systemConfig = loadedConfig
			logger.Infof("[NewApp] Loaded system config: closeToTray=%v, rememberChoice=%v, updateIgnoredDate=%s",
				systemConfig.CloseToTray, systemConfig.RememberChoice, systemConfig.UpdateIgnoredDate)
		} else {
			logger.Warnf("[NewApp] Failed to load system config: %v", err)
		}
	} else {
		logger.Warnf("[NewApp] systemConfigManager is nil")
	}

	// 初始化数据库
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Errorf("Failed to get home directory: %v", err)
		homeDir = "."
	}
	configDir := filepath.Join(homeDir, ".wemediaspider")

	db, err := database.NewDatabase(configDir)
	if err != nil {
		logger.Errorf("Failed to initialize database: %v", err)
	} else {
		// 自动迁移表结构
		if err := db.AutoMigrate(); err != nil {
			logger.Errorf("Failed to migrate database: %v", err)
		}
	}

	// 初始化仓储
	var articleRepo repository.ArticleRepository
	var accountRepo repository.AccountRepository
	var statsRepo repository.StatsRepository
	if db != nil {
		articleRepo = repository.NewArticleRepository(db.DB)
		accountRepo = repository.NewAccountRepository(db.DB)
		statsRepo = repository.NewStatsRepository(db.DB)
	}

	app := &App{
		loginManager:        spider.NewLoginManager(),
		configManager:       config.NewManager(),
		systemConfigManager: systemConfigManager,
		cacheManager:        cacheManager,
		db:                  db,
		articleRepo:         articleRepo,
		accountRepo:         accountRepo,
		statsRepo:           statsRepo,
		trayManager:         tray.NewManager(),
		autostartManager:    autostartManager,
		closeToTray:         systemConfig.CloseToTray,
		rememberChoice:      systemConfig.RememberChoice,
		updateIgnoredDate:   systemConfig.UpdateIgnoredDate,
	}

	logger.Infof("[NewApp] App created with updateIgnoredDate=%s", app.updateIgnoredDate)
	return app
}

// Startup 应用启动时调用
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// 延迟初始化系统托盘，避免启动时的竞态条件
	// 注意：systray 库在 Windows 上可能会输出 "The operation completed successfully" 错误日志
	// 这是 Windows API 的正常行为，不影响功能，可以忽略
	go func() {
		time.Sleep(500 * time.Millisecond)

		// 尝试加载 ICO 图标文件
		iconPaths := []string{
			"icon.ico",
			"build/appicon.png",
			"appicon.png",
		}

		var iconData []byte
		var err error
		var loadedPath string

		for _, path := range iconPaths {
			iconData, err = os.ReadFile(path)
			if err == nil {
				loadedPath = path
				break
			}
		}

		if err != nil {
			logger.Warnf("Failed to load tray icon from all paths, using default")
			iconData = nil
		} else {
			logger.Infof("Loaded tray icon from: %s", loadedPath)
		}

		// 传递退出回调函数
		a.trayManager.Setup(ctx, iconData, func() {
			a.ForceQuit()
		})
	}()

	// 输出当前系统配置状态
	logger.Infof("Application started - CloseToTray: %v, RememberChoice: %v", a.closeToTray, a.rememberChoice)
}

// Shutdown 应用关闭时调用
func (a *App) Shutdown(ctx context.Context) {
	if a.cacheManager != nil {
		a.cacheManager.Close()
	}
	if a.db != nil {
		a.db.Close()
	}
	logger.Info("Application shutdown")
}

// ============================================================
// 托盘和窗口管理
// ============================================================

// HideToTray 隐藏到托盘
func (a *App) HideToTray() {
	a.trayManager.HideToTray()
}

// ShowWindow 显示窗口
func (a *App) ShowWindow() {
	a.trayManager.ShowWindow()
}

// SetCloseToTray 设置关闭到托盘
func (a *App) SetCloseToTray(enabled bool) {
	a.closeToTray = enabled
	logger.Infof("Close to tray: %v", enabled)

	// 保存到配置文件
	a.saveSystemConfig()

	// 发送配置更新事件到前端
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "system-config-changed", map[string]interface{}{
			"closeToTray":    a.closeToTray,
			"rememberChoice": a.rememberChoice,
		})
	}
}

// GetCloseToTray 获取关闭到托盘设置
func (a *App) GetCloseToTray() bool {
	return a.closeToTray
}

// SetRememberChoice 设置是否记住用户选择
func (a *App) SetRememberChoice(remember bool) {
	a.rememberChoice = remember
	logger.Infof("Remember choice: %v", remember)

	// 保存到配置文件
	a.saveSystemConfig()

	// 发送配置更新事件到前端
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "system-config-changed", map[string]interface{}{
			"closeToTray":    a.closeToTray,
			"rememberChoice": a.rememberChoice,
		})
	}
}

// GetRememberChoice 获取是否记住用户选择
func (a *App) GetRememberChoice() bool {
	return a.rememberChoice
}

// GetUpdateIgnoredDate 获取更新忽略日期
func (a *App) GetUpdateIgnoredDate() string {
	logger.Infof("[GetUpdateIgnoredDate] Returning: %s", a.updateIgnoredDate)
	return a.updateIgnoredDate
}

// SetUpdateIgnoredDate 设置更新忽略日期
func (a *App) SetUpdateIgnoredDate(date string) {
	logger.Infof("[SetUpdateIgnoredDate] Called with date: %s", date)
	a.updateIgnoredDate = date
	logger.Infof("[SetUpdateIgnoredDate] Updated field to: %s", a.updateIgnoredDate)

	// 保存到配置文件
	logger.Infof("[SetUpdateIgnoredDate] Calling saveSystemConfig")
	a.saveSystemConfig()
	logger.Infof("[SetUpdateIgnoredDate] saveSystemConfig completed")

	// 发送配置更新事件到前端
	if a.ctx != nil {
		logger.Infof("[SetUpdateIgnoredDate] Emitting system-config-changed event")
		runtime.EventsEmit(a.ctx, "system-config-changed", map[string]interface{}{
			"closeToTray":       a.closeToTray,
			"rememberChoice":    a.rememberChoice,
			"updateIgnoredDate": a.updateIgnoredDate,
		})
	} else {
		logger.Warnf("[SetUpdateIgnoredDate] ctx is nil, cannot emit event")
	}
	logger.Infof("[SetUpdateIgnoredDate] Completed")
}

// saveSystemConfig 保存系统配置到文件
func (a *App) saveSystemConfig() {
	logger.Infof("[saveSystemConfig] Called")
	if a.systemConfigManager == nil {
		logger.Warnf("[saveSystemConfig] systemConfigManager is nil, cannot save")
		return
	}

	config := config.SystemConfig{
		CloseToTray:       a.closeToTray,
		RememberChoice:    a.rememberChoice,
		UpdateIgnoredDate: a.updateIgnoredDate,
	}

	logger.Infof("[saveSystemConfig] Saving config: closeToTray=%v, rememberChoice=%v, updateIgnoredDate=%s",
		config.CloseToTray, config.RememberChoice, config.UpdateIgnoredDate)

	if err := a.systemConfigManager.Save(config); err != nil {
		logger.Errorf("[saveSystemConfig] Failed to save system config: %v", err)
	} else {
		logger.Infof("[saveSystemConfig] Successfully saved system config")
	}
}

// ForceQuit 强制退出程序
func (a *App) ForceQuit() {
	a.forceQuit = true
	runtime.Quit(a.ctx)
}

// ShouldBlockClose 检查是否应该阻止关闭
func (a *App) ShouldBlockClose() bool {
	return !a.forceQuit
}

// ============================================================
// 自启动管理
// ============================================================

// IsAutostartEnabled 检查是否启用自启动
func (a *App) IsAutostartEnabled() bool {
	if a.autostartManager == nil {
		return false
	}
	return a.autostartManager.IsEnabled()
}

// SetAutostart 设置自启动
func (a *App) SetAutostart(enabled bool, silent bool) error {
	if a.autostartManager == nil {
		return fmt.Errorf("autostart manager not initialized")
	}

	if enabled {
		return a.autostartManager.Enable(silent)
	}
	return a.autostartManager.Disable()
}

// IsAutostartSilent 检查是否为静默启动
func (a *App) IsAutostartSilent() bool {
	if a.autostartManager == nil {
		return false
	}
	return a.autostartManager.IsSilentMode()
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
	if err == nil && len(articles) > 0 {
		// 保存到数据库
		if a.db != nil && a.articleRepo != nil {
			logger.Info("保存文章到数据库...")
			dbArticles := make([]*dbmodels.Article, 0, len(articles))

			for i := range articles {
				article := &articles[i]
				// 查找或创建公众号
				account, accErr := a.accountRepo.FindOrCreate(article.AccountFakeid, article.AccountName)
				if accErr != nil {
					logger.Errorf("Failed to find or create account: %v", accErr)
					continue
				}

				dbArticle := database.ConvertToDBArticle(article, account.ID)
				dbArticles = append(dbArticles, dbArticle)
			}

			// 批量保存到数据库
			if len(dbArticles) > 0 {
				if saveErr := a.articleRepo.BatchCreate(dbArticles); saveErr != nil {
					logger.Errorf("Failed to save articles to database: %v", saveErr)
				} else {
					logger.Infof("Successfully saved %d articles to database", len(dbArticles))
				}
			}

			// 更新统计信息
			totalArticles, _ := a.articleRepo.Count()
			accounts, _ := a.accountRepo.List()
			todayArticles := database.CalculateTodayArticles(articles)
			lastScrapeTime := articles[0].PublishTime

			if statsErr := a.statsRepo.UpdateArticleStats(
				int(totalArticles),
				len(accounts),
				todayArticles,
				lastScrapeTime,
			); statsErr != nil {
				logger.Errorf("Failed to update stats: %v", statsErr)
			}
		}

		runtime.EventsEmit(a.ctx, "scrape:completed", map[string]interface{}{
			"total": len(articles),
		})
	} else if err != nil {
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
		if a.statsRepo != nil {
			if err := a.statsRepo.IncrementExports(); err != nil {
				logger.Errorf("Failed to update export stats: %v", err)
			}
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
	return "1.2.0"
}

// VersionInfo 版本信息
type VersionInfo struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	HasUpdate      bool   `json:"hasUpdate"`
	UpdateURL      string `json:"updateUrl"`
	ReleaseNotes   string `json:"releaseNotes"`
}

// updateCache 更新检查缓存
type updateCache struct {
	Version      string    `json:"version"`
	UpdateURL    string    `json:"updateUrl"`
	ReleaseNotes string    `json:"releaseNotes"`
	CheckedAt    time.Time `json:"checkedAt"`
}

// CheckForUpdates 检查更新
func (a *App) CheckForUpdates() (VersionInfo, error) {
	currentVersion := a.GetAppVersion()
	logger.Infof("开始检查更新，当前版本: %s", currentVersion)

	// 暂时禁用缓存，每次都重新检查
	// cacheFile := filepath.Join(os.TempDir(), "wemediaspider_update_cache.json")
	// if cached, ok := a.loadUpdateCache(cacheFile); ok {
	// 	cacheAge := time.Since(cached.CheckedAt)
	// 	logger.Infof("找到缓存: 版本=%s, 缓存时间=%v", cached.Version, cacheAge)
	// 	if cacheAge < 24*time.Hour {
	// 		logger.Info("使用缓存的更新信息")
	// 		hasUpdate := compareVersions(cached.Version, currentVersion) > 0
	// 		logger.Infof("版本比较: %s vs %s, 有更新=%v", cached.Version, currentVersion, hasUpdate)
	// 		return VersionInfo{
	// 			CurrentVersion: currentVersion,
	// 			LatestVersion:  cached.Version,
	// 			HasUpdate:      hasUpdate,
	// 			UpdateURL:      cached.UpdateURL,
	// 			ReleaseNotes:   cached.ReleaseNotes,
	// 		}, nil
	// 	}
	// 	logger.Info("缓存已过期，重新检查")
	// } else {
	// 	logger.Info("未找到缓存，执行首次检查")
	// }

	// 方案1: 尝试使用 jsdelivr CDN（不受速率限制）
	latestVersion, updateURL, releaseNotes, err := a.checkUpdateViaCDN()
	if err == nil {
		// 暂时不保存缓存
		// a.saveUpdateCache(cacheFile, updateCache{
		// 	Version:      latestVersion,
		// 	UpdateURL:    updateURL,
		// 	ReleaseNotes: releaseNotes,
		// 	CheckedAt:    time.Now(),
		// })

		hasUpdate := compareVersions(latestVersion, currentVersion) > 0
		logger.Infof("当前版本: %s, 最新版本: %s, 有更新: %v", currentVersion, latestVersion, hasUpdate)

		return VersionInfo{
			CurrentVersion: currentVersion,
			LatestVersion:  latestVersion,
			HasUpdate:      hasUpdate,
			UpdateURL:      updateURL,
			ReleaseNotes:   releaseNotes,
		}, nil
	}

	logger.Warnf("CDN 检查失败，尝试 GitHub API: %v", err)

	// 方案2: 回退到 GitHub API
	return a.checkUpdateViaGitHubAPI(currentVersion)
}

// checkUpdateViaCDN 通过 jsdelivr CDN 检查更新
func (a *App) checkUpdateViaCDN() (version, updateURL, releaseNotes string, err error) {
	logger.Info("尝试通过 CDN 检查更新")
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 使用 jsdelivr 获取 releases 信息（使用正确的 URL 格式）
	cdnURL := "https://cdn.jsdelivr.net/gh/vag-Zhao/WeMediaSpider-Go@main/version.json"
	logger.Infof("CDN URL: %s", cdnURL)
	req, err := http.NewRequest("GET", cdnURL, nil)
	if err != nil {
		logger.Errorf("创建 CDN 请求失败: %v", err)
		return "", "", "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("CDN 请求失败: %v", err)
		return "", "", "", err
	}
	defer resp.Body.Close()

	logger.Infof("CDN 响应状态码: %d", resp.StatusCode)
	if resp.StatusCode != 200 {
		return "", "", "", fmt.Errorf("CDN 返回状态码: %d", resp.StatusCode)
	}

	var versionInfo struct {
		Version      string `json:"version"`
		UpdateURL    string `json:"updateUrl"`
		ReleaseNotes string `json:"releaseNotes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&versionInfo); err != nil {
		logger.Errorf("解析 CDN 响应失败: %v", err)
		return "", "", "", err
	}

	logger.Infof("CDN 返回版本信息: version=%s, updateUrl=%s", versionInfo.Version, versionInfo.UpdateURL)

	return strings.TrimPrefix(versionInfo.Version, "v"),
		versionInfo.UpdateURL,
		versionInfo.ReleaseNotes,
		nil
}

// checkUpdateViaGitHubAPI 通过 GitHub API 检查更新（回退方案）
func (a *App) checkUpdateViaGitHubAPI(currentVersion string) (VersionInfo, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://api.github.com/repos/vag-Zhao/WeMediaSpider-Go/releases/latest", nil)
	if err != nil {
		logger.Warnf("创建请求失败: %v", err)
		return VersionInfo{
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
			HasUpdate:      false,
		}, nil
	}

	req.Header.Set("User-Agent", "WeMediaSpider-Go/"+currentVersion)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
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
		if resp.StatusCode == 403 {
			logger.Warnf("GitHub API 速率限制，将在 24 小时后重试")
		} else {
			logger.Warnf("检查更新失败，状态码: %d", resp.StatusCode)
		}
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

	latestVersion := strings.TrimPrefix(release.TagName, "v")

	// 暂时不保存缓存
	// a.saveUpdateCache(cacheFile, updateCache{
	// 	Version:      latestVersion,
	// 	UpdateURL:    release.HTMLURL,
	// 	ReleaseNotes: release.Body,
	// 	CheckedAt:    time.Now(),
	// })

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

// ClearUpdateCache 清除更新缓存（用于测试）
func (a *App) ClearUpdateCache() error {
	cacheFile := filepath.Join(os.TempDir(), "wemediaspider_update_cache.json")
	err := os.Remove(cacheFile)
	if err != nil && !os.IsNotExist(err) {
		logger.Warnf("清除更新缓存失败: %v", err)
		return err
	}
	logger.Info("更新缓存已清除")
	return nil
}

// loadUpdateCache 加载更新缓存
func (a *App) loadUpdateCache(cacheFile string) (updateCache, bool) {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return updateCache{}, false
	}

	var cache updateCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return updateCache{}, false
	}

	return cache, true
}

// saveUpdateCache 保存更新缓存
func (a *App) saveUpdateCache(cacheFile string, cache updateCache) {
	data, err := json.Marshal(cache)
	if err != nil {
		logger.Warnf("保存更新缓存失败: %v", err)
		return
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		logger.Warnf("写入更新缓存失败: %v", err)
	}
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
	// 从数据库获取
	if a.db == nil || a.statsRepo == nil {
		return models.AppData{}, fmt.Errorf("database not initialized")
	}

	stats, err := a.statsRepo.Get()
	if err != nil {
		return models.AppData{}, fmt.Errorf("failed to get stats from database: %w", err)
	}

	// 获取公众号列表
	accounts, err := a.accountRepo.List()
	if err != nil {
		logger.Warnf("Failed to list accounts: %v", err)
		accounts = []*dbmodels.Account{}
	}

	accountNames := make([]string, len(accounts))
	for i, acc := range accounts {
		accountNames[i] = acc.Name
	}

	return database.ConvertToAppData(stats, accountNames), nil
}

// UpdateAppData 更新应用数据（已废弃，保留用于兼容）
func (a *App) UpdateAppData(articles []models.Article) error {
	logger.Warn("UpdateAppData is deprecated, stats are updated automatically")
	return nil
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
		if a.statsRepo != nil {
			if err := a.statsRepo.IncrementImages(len(images)); err != nil {
				logger.Errorf("Failed to update image stats: %v", err)
			}
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

// ListDataFiles 列出所有保存的数据文件（按公众号分组）
func (a *App) ListDataFiles() ([]storage.DataFileInfo, error) {
	// 从数据库获取
	if a.db == nil || a.articleRepo == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	articles, err := a.articleRepo.GetAllArticles()
	if err != nil {
		return nil, fmt.Errorf("failed to get articles from database: %w", err)
	}

	// 按公众号分组
	accountGroups := make(map[string][]models.Article)
	for _, dbArt := range articles {
		art := database.ConvertToAppArticle(dbArt)
		accountGroups[art.AccountFakeid] = append(accountGroups[art.AccountFakeid], art)
	}

	// 构建 DataFileInfo 列表
	var dataFiles []storage.DataFileInfo
	for fakeid, arts := range accountGroups {
		if len(arts) == 0 {
			continue
		}

		// 获取公众号名称（使用第一篇文章的公众号名）
		accountName := arts[0].AccountName

		// 获取最早和最晚的发布时间
		var earliestTime, latestTime int64
		for i, art := range arts {
			if i == 0 {
				earliestTime = art.PublishTimestamp
				latestTime = art.PublishTimestamp
			} else {
				if art.PublishTimestamp < earliestTime {
					earliestTime = art.PublishTimestamp
				}
				if art.PublishTimestamp > latestTime {
					latestTime = art.PublishTimestamp
				}
			}
		}

		// 格式化时间范围
		earliestDate := time.Unix(earliestTime, 0).Format("2006-01-02")
		latestDate := time.Unix(latestTime, 0).Format("2006-01-02")
		timeRange := earliestDate
		if earliestDate != latestDate {
			timeRange = fmt.Sprintf("%s ~ %s", earliestDate, latestDate)
		}

		dataFiles = append(dataFiles, storage.DataFileInfo{
			Filename:   fmt.Sprintf("%s.db", accountName),
			FilePath:   fakeid, // 使用 fakeid 作为标识
			SaveTime:   timeRange,
			TotalCount: len(arts),
			Accounts:   []string{accountName},
			FileSize:   0, // 数据库模式下不计算文件大小
		})
	}

	// 按文章数量倒序排序
	sort.Slice(dataFiles, func(i, j int) bool {
		return dataFiles[i].TotalCount > dataFiles[j].TotalCount
	})

	return dataFiles, nil
}

// LoadDataFile 加载指定公众号的文章
func (a *App) LoadDataFile(fakeidOrPath string) ([]models.Article, error) {
	// 从数据库加载
	if a.db == nil || a.articleRepo == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// 使用 fakeid 查询文章
	dbArticles, err := a.articleRepo.FindByAccountFakeid(fakeidOrPath, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to load articles from database: %w", err)
	}

	return database.ConvertToAppArticles(dbArticles), nil
}

// DeleteDataFile 删除指定公众号的所有文章
func (a *App) DeleteDataFile(fakeidOrPath string) error {
	// 从数据库删除
	if a.db == nil || a.articleRepo == nil {
		return fmt.Errorf("database not initialized")
	}

	// 查询该公众号的所有文章
	dbArticles, err := a.articleRepo.FindByAccountFakeid(fakeidOrPath, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to find articles: %w", err)
	}

	// 删除所有文章
	for _, article := range dbArticles {
		if err := a.articleRepo.Delete(article.ArticleID); err != nil {
			logger.Warnf("Failed to delete article %s: %v", article.ArticleID, err)
		}
	}

	logger.Infof("Deleted %d articles for account %s", len(dbArticles), fakeidOrPath)

	// 更新统计信息
	totalArticles, _ := a.articleRepo.Count()
	accounts, _ := a.accountRepo.List()
	if err := a.statsRepo.UpdateArticleStats(int(totalArticles), len(accounts), 0, ""); err != nil {
		logger.Warnf("Failed to update stats: %v", err)
	}

	return nil
}

// GetDataDirectory 获取数据目录路径
func (a *App) GetDataDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(homeDir, ".wemediaspider")
}

// OpenDataFileDialog 打开数据文件选择对话框（用于导入 JSON 文件）
func (a *App) OpenDataFileDialog() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	dataDir := filepath.Join(homeDir, ".wemediaspider", "data")

	filepath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "选择 JSON 数据文件导入",
		DefaultDirectory: dataDir,
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

// ImportJSONFile 导入 JSON 文件到数据库
func (a *App) ImportJSONFile(filePath string) error {
	if a.db == nil || a.articleRepo == nil {
		return fmt.Errorf("database not initialized")
	}

	// 读取 JSON 文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// 解析 JSON
	var savedData struct {
		Articles   []models.Article `json:"articles"`
		SaveTime   string           `json:"saveTime"`
		TotalCount int              `json:"totalCount"`
		Accounts   []string         `json:"accounts"`
	}

	if err := json.Unmarshal(data, &savedData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	logger.Infof("Importing %d articles from %s", len(savedData.Articles), filePath)

	// 转换并保存到数据库
	dbArticles := make([]*dbmodels.Article, 0, len(savedData.Articles))
	for i := range savedData.Articles {
		article := &savedData.Articles[i]

		// 查找或创建公众号
		account, err := a.accountRepo.FindOrCreate(article.AccountFakeid, article.AccountName)
		if err != nil {
			logger.Warnf("Failed to find or create account: %v", err)
			continue
		}

		dbArticle := database.ConvertToDBArticle(article, account.ID)
		dbArticles = append(dbArticles, dbArticle)
	}

	// 批量保存
	if len(dbArticles) > 0 {
		if err := a.articleRepo.BatchCreate(dbArticles); err != nil {
			return fmt.Errorf("failed to save articles: %w", err)
		}
		logger.Infof("Successfully imported %d articles", len(dbArticles))
	}

	// 更新统计信息
	totalArticles, _ := a.articleRepo.Count()
	accounts, _ := a.accountRepo.List()
	todayArticles := database.CalculateTodayArticles(savedData.Articles)
	lastScrapeTime := time.Now().Format("2006-01-02 15:04:05")

	if err := a.statsRepo.UpdateArticleStats(
		int(totalArticles),
		len(accounts),
		todayArticles,
		lastScrapeTime,
	); err != nil {
		logger.Warnf("Failed to update stats: %v", err)
	}

	return nil
}

// ExportToJSON 导出数据库数据到 JSON 文件
func (a *App) ExportToJSON(dateOrPath string) (string, error) {
	if a.db == nil || a.articleRepo == nil {
		return "", fmt.Errorf("database not initialized")
	}

	// 解析日期
	date, err := time.Parse("2006-01-02", dateOrPath)
	if err != nil {
		return "", fmt.Errorf("invalid date format: %w", err)
	}

	// 查询该日期的文章
	startDate := date
	endDate := date.Add(24 * time.Hour)
	dbArticles, err := a.articleRepo.FindByDateRange(startDate, endDate, 0, 0)
	if err != nil {
		return "", fmt.Errorf("failed to load articles: %w", err)
	}

	articles := database.ConvertToAppArticles(dbArticles)

	// 构建保存数据
	accounts := database.ExtractAccountNames(articles)
	savedData := struct {
		Articles   []models.Article `json:"articles"`
		SaveTime   string           `json:"saveTime"`
		TotalCount int              `json:"totalCount"`
		Accounts   []string         `json:"accounts"`
	}{
		Articles:   articles,
		SaveTime:   time.Now().Format("2006-01-02 15:04:05"),
		TotalCount: len(articles),
		Accounts:   accounts,
	}

	// 序列化为 JSON
	jsonData, err := json.MarshalIndent(savedData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// 打开保存对话框
	filename := fmt.Sprintf("export_%s.json", dateOrPath)
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出为 JSON 文件",
		DefaultFilename: filename,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "JSON 文件 (*.json)",
				Pattern:     "*.json",
			},
		},
	})

	if err != nil || savePath == "" {
		return "", fmt.Errorf("用户取消操作")
	}

	// 写入文件
	if err := os.WriteFile(savePath, jsonData, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	logger.Infof("Exported %d articles to %s", len(articles), savePath)
	return savePath, nil
}
