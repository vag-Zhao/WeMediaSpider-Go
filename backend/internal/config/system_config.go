package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"WeMediaSpider/backend/pkg/logger"
)

// SystemConfig 系统配置
type SystemConfig struct {
	CloseToTray    bool `json:"closeToTray"`    // 关闭到托盘
	RememberChoice bool `json:"rememberChoice"` // 记住用户选择
}

// SystemConfigManager 系统配置管理器
type SystemConfigManager struct {
	configPath string
}

// NewSystemConfigManager 创建系统配置管理器
func NewSystemConfigManager() (*SystemConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(homeDir, ".wemediaspider")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	return &SystemConfigManager{
		configPath: filepath.Join(configDir, "system_config.json"),
	}, nil
}

// Load 加载系统配置
func (m *SystemConfigManager) Load() (SystemConfig, error) {
	// 默认配置
	defaultConfig := SystemConfig{
		CloseToTray:    false, // 默认禁用关闭到托盘
		RememberChoice: false, // 默认不记住选择
	}

	// 检查文件是否存在
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		// 文件不存在，保存默认配置
		if err := m.Save(defaultConfig); err != nil {
			logger.Warnf("Failed to save default system config: %v", err)
		}
		return defaultConfig, nil
	}

	// 读取文件
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		logger.Warnf("Failed to read system config: %v", err)
		return defaultConfig, nil
	}

	// 解析 JSON
	var config SystemConfig
	if err := json.Unmarshal(data, &config); err != nil {
		logger.Warnf("Failed to parse system config: %v", err)
		return defaultConfig, nil
	}

	logger.Infof("Loaded system config: closeToTray=%v, rememberChoice=%v", config.CloseToTray, config.RememberChoice)
	return config, nil
}

// Save 保存系统配置
func (m *SystemConfigManager) Save(config SystemConfig) error {
	// 确保配置目录存在
	configDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// 序列化为 JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// 写入文件
	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return err
	}

	logger.Infof("Saved system config: closeToTray=%v, rememberChoice=%v", config.CloseToTray, config.RememberChoice)
	return nil
}
