package tray

import (
	"context"
	"time"

	"WeMediaSpider/backend/pkg/logger"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Manager 托盘管理器
type Manager struct {
	ctx        context.Context
	onShow     func()
	onQuit     func()
	isHidden   bool
	quitFunc   func() // 退出回调函数
}

// NewManager 创建托盘管理器
func NewManager() *Manager {
	return &Manager{
		isHidden: false,
	}
}

// Setup 设置托盘
func (m *Manager) Setup(ctx context.Context, iconData []byte, quitFunc func()) {
	m.ctx = ctx
	m.quitFunc = quitFunc

	go func() {
		systray.Run(func() {
			m.onReady(iconData)
		}, func() {
			logger.Info("System tray exiting")
		})
	}()

	logger.Info("System tray initialized")
}

// onReady 托盘就绪回调
func (m *Manager) onReady(iconData []byte) {
	// 设置托盘图标
	if len(iconData) > 0 {
		// 忽略 SetIcon 的错误，因为 Windows 会报告误导性的错误
		// "The operation completed successfully" 实际上表示成功
		systray.SetIcon(iconData)
	} else {
		// 如果没有图标数据，使用默认图标（空图标）
		logger.Warn("No icon data provided, using default icon")
	}

	systray.SetTitle("WeMediaSpider")
	systray.SetTooltip("微信公众号爬虫")

	// 显示窗口菜单项
	mShow := systray.AddMenuItem("显示窗口", "显示主窗口")
	systray.AddSeparator()

	// 退出菜单项
	mQuit := systray.AddMenuItem("退出", "退出应用")

	// 监听菜单点击
	go func() {
		logger.Info("Tray menu listener started")
		for {
			select {
			case <-mShow.ClickedCh:
				logger.Info("Show window menu item clicked")
				m.ShowWindow()
			case <-mQuit.ClickedCh:
				logger.Info("Quit menu item clicked")
				m.Quit()
				return
			}
		}
	}()
}

// HideToTray 隐藏到托盘
func (m *Manager) HideToTray() {
	if m.ctx != nil {
		runtime.WindowHide(m.ctx)
		m.isHidden = true
		logger.Info("Window hidden to tray")
	}
}

// ShowWindow 显示窗口
func (m *Manager) ShowWindow() {
	if m.ctx == nil {
		logger.Error("Context is nil, cannot show window")
		return
	}

	runtime.WindowShow(m.ctx)
	runtime.WindowUnminimise(m.ctx)
	runtime.WindowSetAlwaysOnTop(m.ctx, true)

	go func() {
		time.Sleep(200 * time.Millisecond)
		if m.ctx != nil {
			runtime.WindowCenter(m.ctx)
			runtime.WindowSetAlwaysOnTop(m.ctx, false)
		}
	}()

	m.isHidden = false
	logger.Info("Window shown from tray")
}

// Quit 退出应用
func (m *Manager) Quit() {
	logger.Info("Tray quit requested")
	if m.quitFunc != nil {
		m.quitFunc()
	} else {
		// 回退方案
		if m.ctx != nil {
			runtime.Quit(m.ctx)
		}
	}
	systray.Quit()
}

// IsHidden 是否已隐藏
func (m *Manager) IsHidden() bool {
	return m.isHidden
}

