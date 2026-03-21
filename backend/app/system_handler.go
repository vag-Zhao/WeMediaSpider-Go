package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"WeMediaSpider/backend/internal/config"
	"WeMediaSpider/backend/pkg/logger"
	"WeMediaSpider/backend/pkg/timeutil"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
)

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// 启动 NTP 时间同步（对齐中国时间）
	logger.Log.Info("启动 NTP 时间同步")
	timeutil.StartAutoSync()

	// 延迟初始化系统托盘，避免启动时的竞态条件
	// 注意：systray 库在 Windows 上可能会输出 "The operation completed successfully" 错误日志
	// 这是 Windows API 的正常行为，不影响功能，可以忽略
	go func() {
		time.Sleep(500 * time.Millisecond)

		// 优先使用嵌入的图标
		var iconData []byte
		if len(embeddedIcon) > 0 {
			iconData = embeddedIcon
			logger.Log.Info("使用嵌入式托盘图标")
		} else {
			// 回退：尝试从文件系统加载
			iconPaths := []string{
				"build/icon.ico",
				"icon.ico",
				"build/windows/icon.ico",
			}
			for _, path := range iconPaths {
				data, err := os.ReadFile(path)
				if err == nil {
					iconData = data
					logger.Log.Info("从文件加载托盘图标", zap.String("path", path))
					break
				}
			}
			if iconData == nil {
				logger.Log.Warn("托盘图标加载失败，使用默认图标")
			}
		}

		// 传递退出回调函数
		a.trayManager.Setup(ctx, iconData, func() {
			a.ForceQuit()
		})
	}()

	// 输出当前系统配置状态
	logger.Log.Info("应用已启动", zap.Bool("close_to_tray", a.closeToTray), zap.Bool("remember_choice", a.rememberChoice))

	// 启动定时任务管理器
	if a.cronManager != nil {
		a.cronManager.Start()
		a.loadScheduledTasks()
	}

	// 注册任务完成事件回调
	if a.taskScheduler != nil {
		a.taskScheduler.SetOnTaskComplete(func(taskID uint, status string, articles int, errMsg string) {
			runtime.EventsEmit(a.ctx, "task:completed", map[string]interface{}{
				"taskID":   taskID,
				"status":   status,
				"articles": articles,
				"errMsg":   errMsg,
			})
		})
	}
}

// Shutdown 应用关闭时调用
func (a *App) Shutdown(ctx context.Context) {
	// 停止定时任务管理器
	if a.cronManager != nil {
		a.cronManager.Stop()
	}
	// 关闭分析器
	if a.analyzer != nil {
		a.analyzer.Close()
	}
	if a.cacheManager != nil {
		a.cacheManager.Close()
	}
	if a.db != nil {
		a.db.Close()
	}
	logger.Log.Info("应用已关闭")
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
	logger.Log.Info("关闭到托盘设置已更新", zap.Bool("enabled", enabled))

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
	logger.Log.Info("记住选择设置已更新", zap.Bool("remember", remember))

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
	logger.Log.Info("获取更新忽略日期", zap.String("date", a.updateIgnoredDate))
	return a.updateIgnoredDate
}

// SetUpdateIgnoredDate 设置更新忽略日期
func (a *App) SetUpdateIgnoredDate(date string) {
	logger.Log.Info("设置更新忽略日期", zap.String("date", date))
	a.updateIgnoredDate = date

	// 保存到配置文件
	a.saveSystemConfig()

	// 发送配置更新事件到前端
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "system-config-changed", map[string]interface{}{
			"closeToTray":       a.closeToTray,
			"rememberChoice":    a.rememberChoice,
			"updateIgnoredDate": a.updateIgnoredDate,
		})
	} else {
		logger.Log.Warn("ctx 为空，无法发送事件")
	}
}

// saveSystemConfig 保存系统配置到文件
func (a *App) saveSystemConfig() {
	if a.systemConfigManager == nil {
		logger.Log.Warn("系统配置管理器未初始化，无法保存")
		return
	}

	cfg := config.SystemConfig{
		CloseToTray:       a.closeToTray,
		RememberChoice:    a.rememberChoice,
		UpdateIgnoredDate: a.updateIgnoredDate,
	}

	if err := a.systemConfigManager.Save(cfg); err != nil {
		logger.Log.Error("保存系统配置失败", zap.Error(err))
	} else {
		logger.Log.Info("系统配置已保存", zap.Bool("close_to_tray", cfg.CloseToTray), zap.Bool("remember_choice", cfg.RememberChoice), zap.String("update_ignored_date", cfg.UpdateIgnoredDate))
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

func (a *App) GetAppVersion() string {
	return "2.0.0"
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
	logger.Log.Info("开始检查更新", zap.String("current_version", currentVersion))

	// 先检查缓存（24小时内有效）
	cacheFile := filepath.Join(os.TempDir(), "wemediaspider_update_cache.json")
	if cached, ok := a.loadUpdateCache(cacheFile); ok {
		cacheAge := time.Since(cached.CheckedAt)
		if cacheAge < 24*time.Hour {
			logger.Log.Info("使用缓存的更新信息", zap.Duration("cache_age", cacheAge))
			hasUpdate := compareVersions(cached.Version, currentVersion) > 0
			return VersionInfo{
				CurrentVersion: currentVersion,
				LatestVersion:  cached.Version,
				HasUpdate:      hasUpdate,
				UpdateURL:      cached.UpdateURL,
				ReleaseNotes:   cached.ReleaseNotes,
			}, nil
		}
		logger.Log.Info("更新缓存已过期，重新检查")
	}

	// 多源并发检查：同时请求所有源，取最快成功的结果
	type updateResult struct {
		version      string
		updateURL    string
		releaseNotes string
	}

	resultCh := make(chan updateResult, 1)
	sources := []struct {
		name string
		fn   func() (string, string, string, error)
	}{
		{"GitHub API", a.checkUpdateViaGitHubAPI},
		{"jsdelivr CDN", a.checkUpdateViaCDN},
		{"ghproxy 镜像", a.checkUpdateViaGhproxy},
	}

	for _, src := range sources {
		go func(name string, fn func() (string, string, string, error)) {
			ver, url, notes, err := fn()
			if err != nil {
				logger.Log.Warn("更新源检查失败", zap.String("source", name), zap.Error(err))
				return
			}
			logger.Log.Info("更新源检查成功", zap.String("source", name), zap.String("version", ver))
			select {
			case resultCh <- updateResult{ver, url, notes}:
			default:
			}
		}(src.name, src.fn)
	}

	// 等待最快的结果，最多等 15 秒
	select {
	case r := <-resultCh:
		// 保存缓存
		a.saveUpdateCache(cacheFile, updateCache{
			Version:      r.version,
			UpdateURL:    r.updateURL,
			ReleaseNotes: r.releaseNotes,
			CheckedAt:    time.Now(),
		})

		hasUpdate := compareVersions(r.version, currentVersion) > 0
		logger.Log.Info("版本检查完成", zap.String("current", currentVersion), zap.String("latest", r.version), zap.Bool("has_update", hasUpdate))
		return VersionInfo{
			CurrentVersion: currentVersion,
			LatestVersion:  r.version,
			HasUpdate:      hasUpdate,
			UpdateURL:      r.updateURL,
			ReleaseNotes:   r.releaseNotes,
		}, nil

	case <-time.After(15 * time.Second):
		logger.Log.Warn("所有更新源均超时")
		return VersionInfo{
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
			HasUpdate:      false,
		}, nil
	}
}

// checkUpdateViaCDN 通过 jsdelivr CDN 检查更新
func (a *App) checkUpdateViaCDN() (version, updateURL, releaseNotes string, err error) {
	logger.Log.Info("通过 CDN 检查更新")
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 使用 jsdelivr 获取 releases 信息（使用正确的 URL 格式）
	cdnURL := "https://cdn.jsdelivr.net/gh/vag-Zhao/WeMediaSpider-Go@main/version.json"
	logger.Log.Info("CDN 请求地址", zap.String("url", cdnURL))
	req, err := http.NewRequest("GET", cdnURL, nil)
	if err != nil {
		logger.Log.Error("创建 CDN 请求失败", zap.Error(err))
		return "", "", "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Error("CDN 请求失败", zap.Error(err))
		return "", "", "", err
	}
	defer resp.Body.Close()

	logger.Log.Info("CDN 响应状态码", zap.Int("status", resp.StatusCode))
	if resp.StatusCode != 200 {
		return "", "", "", fmt.Errorf("CDN 返回状态码: %d", resp.StatusCode)
	}

	var versionInfo struct {
		Version      string `json:"version"`
		UpdateURL    string `json:"updateUrl"`
		ReleaseNotes string `json:"releaseNotes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&versionInfo); err != nil {
		logger.Log.Error("解析 CDN 响应失败", zap.Error(err))
		return "", "", "", err
	}

	logger.Log.Info("CDN 返回版本信息", zap.String("version", versionInfo.Version), zap.String("update_url", versionInfo.UpdateURL))

	return strings.TrimPrefix(versionInfo.Version, "v"),
		versionInfo.UpdateURL,
		versionInfo.ReleaseNotes,
		nil
}

// checkUpdateViaGitHubAPI 通过 GitHub API 检查更新
func (a *App) checkUpdateViaGitHubAPI() (version, updateURL, releaseNotes string, err error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", "https://api.github.com/repos/vag-Zhao/WeMediaSpider-Go/releases/latest", nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("User-Agent", "WeMediaSpider-Go/"+a.GetAppVersion())
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", "", fmt.Errorf("GitHub API 返回状态码: %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", "", err
	}

	return strings.TrimPrefix(release.TagName, "v"), release.HTMLURL, release.Body, nil
}

// checkUpdateViaGhproxy 通过 ghproxy 镜像检查更新
func (a *App) checkUpdateViaGhproxy() (version, updateURL, releaseNotes string, err error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// ghproxy 镜像 GitHub raw 文件
	mirrorURL := "https://ghp.ci/https://raw.githubusercontent.com/vag-Zhao/WeMediaSpider-Go/main/version.json"
	req, err := http.NewRequest("GET", mirrorURL, nil)
	if err != nil {
		return "", "", "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", "", fmt.Errorf("ghproxy 返回状态码: %d", resp.StatusCode)
	}

	var versionInfo struct {
		Version      string `json:"version"`
		UpdateURL    string `json:"updateUrl"`
		ReleaseNotes string `json:"releaseNotes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&versionInfo); err != nil {
		return "", "", "", err
	}

	return strings.TrimPrefix(versionInfo.Version, "v"), versionInfo.UpdateURL, versionInfo.ReleaseNotes, nil
}

// ClearUpdateCache 清除更新缓存（用于测试）
func (a *App) ClearUpdateCache() error {
	cacheFile := filepath.Join(os.TempDir(), "wemediaspider_update_cache.json")
	err := os.Remove(cacheFile)
	if err != nil && !os.IsNotExist(err) {
		logger.Log.Warn("清除更新缓存失败", zap.Error(err))
		return err
	}
	logger.Log.Info("更新缓存已清除")
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
		logger.Log.Warn("序列化更新缓存失败", zap.Error(err))
		return
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		logger.Log.Warn("写入更新缓存文件失败", zap.Error(err))
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

func (a *App) GetTimeInfo() map[string]interface{} {
	systemTime := time.Now()
	chinaTime := timeutil.Now()

	// 获取系统时区信息
	systemZone, systemOffset := systemTime.Zone()

	return map[string]interface{}{
		"currentTime":    chinaTime.Format("2006-01-02T15:04:05+08:00"), // ISO 格式，明确时区
		"currentDate":    chinaTime.Format("2006-01-02"),                 // 只返回日期
		"systemTime":     systemTime.Format("2006-01-02 15:04:05"),
		"systemZone":     systemZone,
		"systemOffset":   systemOffset / 3600, // 转换为小时
		"chinaZone":      "CST (UTC+8)",
		"timeOffset":     timeutil.GetTimeOffset().String(),
		"lastSyncTime":   timeutil.GetLastSyncTime().Format("2006-01-02 15:04:05"),
		"ntpServer":      timeutil.ChinaNTPServer,
	}
}

// SyncTimeNow 立即同步时间
func (a *App) SyncTimeNow() error {
	return timeutil.SyncTime()
}

// GetRecentLogs 获取最近的日志
func (a *App) GetRecentLogs(count int) []string {
	buffer := logger.GetBuffer()
	if buffer == nil {
		return []string{}
	}
	return buffer.GetRecentLogs(count)
}

// GetAllLogs 获取所有日志
func (a *App) GetAllLogs() []string {
	buffer := logger.GetBuffer()
	if buffer == nil {
		return []string{}
	}
	return buffer.GetLogs()
}

// ClearLogs 清空日志缓冲区
func (a *App) ClearLogs() {
	buffer := logger.GetBuffer()
	if buffer != nil {
		buffer.Clear()
	}
}


