package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"WeMediaSpider/backend/pkg/logger"

	"golang.org/x/sys/windows/registry"
)

// Manager 自启动管理器
type Manager struct {
	appName string
	exePath string
}

// NewManager 创建自启动管理器
func NewManager() (*Manager, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// 解析符号链接
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	return &Manager{
		appName: "WeMediaSpider",
		exePath: exePath,
	}, nil
}

// IsEnabled 检查是否已启用自启动
func (m *Manager) IsEnabled() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	value, _, err := key.GetStringValue(m.appName)
	if err != nil {
		return false
	}

	return value == m.exePath || value == fmt.Sprintf(`"%s"`, m.exePath)
}

// Enable 启用自启动
func (m *Manager) Enable(silent bool) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("autostart is only supported on Windows")
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close()

	// 如果是静默启动，添加 --silent 参数
	value := m.exePath
	if silent {
		value = fmt.Sprintf(`"%s" --silent`, m.exePath)
	} else {
		value = fmt.Sprintf(`"%s"`, m.exePath)
	}

	if err := key.SetStringValue(m.appName, value); err != nil {
		return fmt.Errorf("failed to set registry value: %w", err)
	}

	logger.Infof("Autostart enabled (silent: %v)", silent)
	return nil
}

// Disable 禁用自启动
func (m *Manager) Disable() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("autostart is only supported on Windows")
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close()

	if err := key.DeleteValue(m.appName); err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("failed to delete registry value: %w", err)
	}

	logger.Info("Autostart disabled")
	return nil
}

// IsSilentMode 检查是否为静默启动模式
func (m *Manager) IsSilentMode() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	value, _, err := key.GetStringValue(m.appName)
	if err != nil {
		return false
	}

	// 检查是否包含 --silent 参数
	return len(value) > 0 && (value[len(value)-8:] == "--silent" || value[len(value)-9:] == "--silent\"")
}
