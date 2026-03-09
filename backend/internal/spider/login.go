package spider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"WeMediaSpider/backend/internal/models"
	"WeMediaSpider/backend/pkg/crypto"
	"WeMediaSpider/backend/pkg/errors"
	"WeMediaSpider/backend/pkg/logger"
	"WeMediaSpider/backend/pkg/utils"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// LoginManager 登录管理器
type LoginManager struct {
	token       string
	cookies     map[string]string
	cacheFile   string
	expireHours int
	keyManager  *crypto.KeyManager
	loginTime   int64 // 添加登录时间字段
}

// NewLoginManager 创建登录管理器
func NewLoginManager() *LoginManager {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".wemediaspider")
	os.MkdirAll(cacheDir, 0755)

	keyManager, err := crypto.NewKeyManager()
	if err != nil {
		logger.Errorf("Failed to create key manager: %v", err)
	}

	lm := &LoginManager{
		cacheFile:   filepath.Join(cacheDir, "login_cache.json"),
		expireHours: 96, // 4 days
		keyManager:  keyManager,
	}

	// 尝试加载缓存
	if err := lm.loadCache(); err == nil {
		logger.Info("已从缓存加载登录状态")
	}

	return lm
}

// Login 执行登录
func (lm *LoginManager) Login(ctx context.Context) error {
	logger.Info("开始登录流程")

	// 尝试加载缓存
	if err := lm.loadCache(); err == nil {
		if lm.validateCache() {
			logger.Info("使用缓存登录成功")
			return nil
		}
	}

	// 自动检测浏览器
	logger.Info("正在检测可用浏览器...")
	browserPath := utils.GetDefaultBrowser()

	if browserPath == "" {
		logger.Error("未检测到 Chrome 或 Edge 浏览器，请安装其中之一")
		return fmt.Errorf("未检测到可用的浏览器（Chrome 或 Edge）")
	}

	logger.Info("创建浏览器实例")

	// 创建独立的context，不使用传入的ctx（避免被前端取消）
	// 设置足够长的超时时间（10分钟，给用户充足时间）
	loginCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// 创建 Chrome 选项
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-plugins", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.ExecPath(browserPath), // 使用检测到的浏览器
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(loginCtx, opts...)
	defer allocCancel()

	chromeCtx, chromeCancel := chromedp.NewContext(allocCtx)
	defer chromeCancel()

	logger.Info("打开微信公众平台登录页面")

	// 访问登录页面（不等待二维码，直接开始轮询）
	err := chromedp.Run(chromeCtx,
		chromedp.Navigate("https://mp.weixin.qq.com/"),
	)

	if err != nil {
		logger.Errorf("打开登录页面失败: %v", err)
		return fmt.Errorf("打开登录页面失败: %w", err)
	}

	logger.Info("页面已打开，请在浏览器窗口中扫码登录...")
	logger.Info("等待登录完成（最长等待10分钟）...")

	// 等待页面加载
	time.Sleep(2 * time.Second)

	// 使用goroutine轮询URL，避免阻塞
	var currentURL string
	loginSuccess := false
	done := make(chan bool, 1)
	errChan := make(chan error, 1)

	logger.Info("开始监控登录状态...")

	go func() {
		for i := 0; i < 600; i++ { // 600秒 = 10分钟
			// 检查context是否被取消
			select {
			case <-loginCtx.Done():
				errChan <- fmt.Errorf("登录超时: %w", loginCtx.Err())
				return
			case <-chromeCtx.Done():
				// 浏览器被关闭
				logger.Warn("检测到浏览器已关闭，登录已取消")
				errChan <- fmt.Errorf("登录已取消：浏览器已关闭")
				return
			default:
			}

			// 获取当前 URL
			var url string
			err := chromedp.Run(chromeCtx, chromedp.Location(&url))
			if err != nil {
				// 检查是否是因为浏览器关闭导致的错误
				if chromeCtx.Err() != nil {
					logger.Warn("检测到浏览器已关闭，登录已取消")
					errChan <- fmt.Errorf("登录已取消：浏览器已关闭")
					return
				}
				logger.Debugf("获取 URL 失败: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			currentURL = url

			// 打印当前URL用于调试
			if i == 0 || i%10 == 0 { // 第一次和每10秒打印一次
				logger.Infof("检查登录状态 [%d/600秒] URL: %s", i, currentURL)
			}

			// 检查 URL 是否包含 token
			if containsString(currentURL, "token=") {
				logger.Infof("检测到登录成功！URL: %s", currentURL)
				done <- true
				return
			}

			// 等待 1 秒后继续检查
			time.Sleep(1 * time.Second)
		}
		errChan <- fmt.Errorf("登录超时，未在10分钟内完成扫码")
	}()

	// 等待登录完成或超时
	select {
	case <-done:
		loginSuccess = true
	case err := <-errChan:
		logger.Error(err.Error())
		return err
	}

	if !loginSuccess {
		logger.Error("登录失败")
		return fmt.Errorf("登录失败")
	}

	logger.Info("正在获取登录信息...")

	// 提取 token 和 cookies
	if err := lm.extractTokenAndCookies(chromeCtx, currentURL); err != nil {
		logger.Errorf("提取登录信息失败: %v", err)
		return err
	}

	logger.Infof("Token: %s", lm.token)
	logger.Infof("Cookies数量: %d", len(lm.cookies))

	// 保存缓存
	if err := lm.saveCache(); err != nil {
		logger.Errorf("保存缓存失败: %v", err)
		return err
	}

	logger.Info("登录信息已保存到缓存")

	// 设置登录时间
	lm.loginTime = time.Now().Unix()

	return nil
}

// extractTokenAndCookies 提取 token 和 cookies
func (lm *LoginManager) extractTokenAndCookies(ctx context.Context, url string) error {
	// 从 URL 提取 token
	// URL 格式: https://mp.weixin.qq.com/cgi-bin/home?t=home/index&lang=zh_CN&token=1234567890
	if containsString(url, "token=") {
		// 简单解析
		parts := splitString(url, "token=")
		if len(parts) > 1 {
			tokenPart := parts[1]
			endIdx := indexOfChar(tokenPart, '&')
			if endIdx > 0 {
				lm.token = tokenPart[:endIdx]
			} else {
				lm.token = tokenPart
			}
		}
	}

	// 获取 cookies
	var cookies []*network.Cookie
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		cookies, err = network.GetCookies().Do(ctx)
		return err
	})); err != nil {
		return err
	}

	lm.cookies = make(map[string]string)
	for _, cookie := range cookies {
		lm.cookies[cookie.Name] = cookie.Value
	}

	return nil
}

// Helper functions
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && indexOfString(s, substr) >= 0
}

func indexOfString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func splitString(s, sep string) []string {
	idx := indexOfString(s, sep)
	if idx < 0 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+len(sep):]}
}

func indexOfChar(s string, c rune) int {
	for i, ch := range s {
		if ch == c {
			return i
		}
	}
	return -1
}

// saveCache 保存缓存
func (lm *LoginManager) saveCache() error {
	// 如果没有设置登录时间，使用当前时间
	timestamp := lm.loginTime
	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}

	cache := models.LoginCache{
		Token:     lm.token,
		Cookies:   lm.cookies,
		Timestamp: timestamp,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(lm.cacheFile, data, 0644)
}

// loadCache 加载缓存
func (lm *LoginManager) loadCache() error {
	data, err := os.ReadFile(lm.cacheFile)
	if err != nil {
		return err
	}

	var cache models.LoginCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return err
	}

	lm.token = cache.Token
	lm.cookies = cache.Cookies
	lm.loginTime = cache.Timestamp // 加载登录时间

	// 检查是否过期
	elapsed := time.Since(time.Unix(cache.Timestamp, 0))
	if elapsed.Hours() > float64(lm.expireHours) {
		return errors.ErrTokenExpired
	}

	return nil
}

// validateCache 验证缓存
func (lm *LoginManager) validateCache() bool {
	// TODO: 实现 API 验证
	return lm.token != "" && len(lm.cookies) > 0
}

// ClearCache 清除缓存
func (lm *LoginManager) ClearCache() error {
	lm.token = ""
	lm.cookies = nil
	return os.Remove(lm.cacheFile)
}

// Logout 退出登录
func (lm *LoginManager) Logout() error {
	return lm.ClearCache()
}

// GetStatus 获取登录状态
func (lm *LoginManager) GetStatus() models.LoginStatus {
	if lm.token == "" {
		return models.LoginStatus{
			IsLoggedIn: false,
			Message:    "未登录",
		}
	}

	// 读取缓存时间
	data, err := os.ReadFile(lm.cacheFile)
	if err != nil {
		return models.LoginStatus{
			IsLoggedIn: false,
			Message:    "缓存文件不存在",
		}
	}

	var cache models.LoginCache
	json.Unmarshal(data, &cache)

	loginTime := time.Unix(cache.Timestamp, 0)
	expireTime := loginTime.Add(time.Duration(lm.expireHours) * time.Hour)
	hoursSince := time.Since(loginTime).Hours()
	hoursUntil := time.Until(expireTime).Hours()

	return models.LoginStatus{
		IsLoggedIn:       true,
		LoginTime:        loginTime,
		ExpireTime:       expireTime,
		HoursSinceLogin:  hoursSince,
		HoursUntilExpire: hoursUntil,
		Token:            lm.token,
		Message:          fmt.Sprintf("已登录 %.1f 小时", hoursSince),
	}
}

// GetToken 获取 token
func (lm *LoginManager) GetToken() string {
	return lm.token
}

// GetCookies 获取 cookies
func (lm *LoginManager) GetCookies() map[string]string {
	return lm.cookies
}

// GetHeaders 获取请求头
func (lm *LoginManager) GetHeaders() map[string]string {
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Referer":    "https://mp.weixin.qq.com/",
	}

	// 添加 Cookie
	cookieStr := ""
	for k, v := range lm.cookies {
		if cookieStr != "" {
			cookieStr += "; "
		}
		cookieStr += k + "=" + v
	}
	if cookieStr != "" {
		headers["Cookie"] = cookieStr
	}

	return headers
}

// ExportCredentials 导出加密的登录凭证（自动加密）
func (lm *LoginManager) ExportCredentials() ([]byte, error) {
	if lm.token == "" || len(lm.cookies) == 0 {
		return nil, fmt.Errorf("未登录，无法导出凭证")
	}

	// 创建凭证数据
	cache := models.LoginCache{
		Token:     lm.token,
		Cookies:   lm.cookies,
		Timestamp: time.Now().Unix(),
	}

	// 序列化为 JSON
	data, err := json.Marshal(cache)
	if err != nil {
		return nil, fmt.Errorf("序列化凭证失败: %w", err)
	}

	// 获取主密钥
	masterKey, err := lm.keyManager.GetMasterKey()
	if err != nil {
		return nil, fmt.Errorf("获取密钥失败: %w", err)
	}

	// 加密为 .zgswx 格式
	encrypted, err := crypto.EncryptToZGSWX(data, masterKey)
	if err != nil {
		return nil, fmt.Errorf("加密凭证失败: %w", err)
	}

	return encrypted, nil
}

// ImportCredentials 导入加密的登录凭证（自动解密）
func (lm *LoginManager) ImportCredentials(encryptedData []byte) error {
	// 验证文件格式
	if err := crypto.ValidateZGSWXFormat(encryptedData); err != nil {
		return fmt.Errorf("无效的凭证文件格式: %w", err)
	}

	// 获取主密钥
	masterKey, err := lm.keyManager.GetMasterKey()
	if err != nil {
		return fmt.Errorf("获取密钥失败: %w", err)
	}

	// 解密数据
	data, err := crypto.DecryptFromZGSWX(encryptedData, masterKey)
	if err != nil {
		return fmt.Errorf("解密凭证失败: %w", err)
	}

	// 反序列化 JSON
	var cache models.LoginCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return fmt.Errorf("解析凭证失败: %w", err)
	}

	// 验证凭证
	if cache.Token == "" || len(cache.Cookies) == 0 {
		return fmt.Errorf("凭证数据无效")
	}

	// 检查凭证是否过期
	elapsed := time.Since(time.Unix(cache.Timestamp, 0))
	if elapsed.Hours() > float64(lm.expireHours) {
		return fmt.Errorf("凭证已过期（超过 %d 小时）", lm.expireHours)
	}

	// 应用凭证
	lm.token = cache.Token
	lm.cookies = cache.Cookies
	lm.loginTime = cache.Timestamp // 保留原始登录时间

	// 保存到本地缓存
	if err := lm.saveCache(); err != nil {
		logger.Errorf("保存凭证到缓存失败: %v", err)
		return fmt.Errorf("保存凭证失败: %w", err)
	}

	logger.Info("登录凭证导入成功")
	return nil
}
