package tray

import (
	"context"

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
}

// NewManager 创建托盘管理器
func NewManager() *Manager {
	return &Manager{
		isHidden: false,
	}
}

// Setup 设置托盘
func (m *Manager) Setup(ctx context.Context, iconData []byte) {
	m.ctx = ctx

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
		systray.SetIcon(iconData)
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
		for {
			select {
			case <-mShow.ClickedCh:
				m.ShowWindow()
			case <-mQuit.ClickedCh:
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
	if m.ctx != nil {
		runtime.WindowShow(m.ctx)
		runtime.WindowUnminimise(m.ctx)
		m.isHidden = false
		logger.Info("Window shown from tray")
	}
}

// Quit 退出应用
func (m *Manager) Quit() {
	if m.ctx != nil {
		runtime.Quit(m.ctx)
	}
	systray.Quit()
}

// IsHidden 是否已隐藏
func (m *Manager) IsHidden() bool {
	return m.isHidden
}

