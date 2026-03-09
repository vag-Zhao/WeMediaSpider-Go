package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"WeMediaSpider/backend/internal/models"
)

// Manager 配置管理器
type Manager struct {
	configFile string
}

// NewManager 创建配置管理器
func NewManager() *Manager {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".wemediaspider")
	os.MkdirAll(configDir, 0755)

	return &Manager{
		configFile: filepath.Join(configDir, "config.json"),
	}
}

// Load 加载配置
func (m *Manager) Load() (models.Config, error) {
	data, err := os.ReadFile(m.configFile)
	if err != nil {
		// 返回默认配置
		return m.GetDefault(), nil
	}

	var config models.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return m.GetDefault(), err
	}

	return config, nil
}

// Save 保存配置
func (m *Manager) Save(config models.Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.configFile, data, 0644)
}

// GetDefault 获取默认配置
func (m *Manager) GetDefault() models.Config {
	homeDir, _ := os.UserHomeDir()
	outputDir := filepath.Join(homeDir, "Documents", "WeMediaSpider")

	return models.Config{
		MaxPages:         10,
		RequestInterval:  5,   // 正常速度5秒，自适应调整
		MaxWorkers:       3,   // 适中的并发数
		IncludeContent:   false,
		CacheExpireHours: 96,
		OutputDir:        outputDir,
	}
}
