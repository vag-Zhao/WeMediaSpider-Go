package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"WeMediaSpider/backend/internal/models"
	"WeMediaSpider/backend/pkg/logger"
)

// Manager 数据存储管理器
type Manager struct {
	dataDir string
}

// NewManager 创建存储管理器
func NewManager() *Manager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Errorf("Failed to get home directory: %v", err)
		homeDir = "."
	}

	// 创建数据目录 ~/.wemediaspider/data
	dataDir := filepath.Join(homeDir, ".wemediaspider", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Errorf("Failed to create data directory: %v", err)
	}

	logger.Infof("数据存储目录: %s", dataDir)

	return &Manager{
		dataDir: dataDir,
	}
}

// GetDataDir 获取数据目录路径
func (m *Manager) GetDataDir() string {
	return m.dataDir
}

// SavedData 保存的数据结构
type SavedData struct {
	Articles   []models.Article `json:"articles"`
	SaveTime   string           `json:"saveTime"`
	TotalCount int              `json:"totalCount"`
	Accounts   []string         `json:"accounts"`
}

// AutoSave 自动保存爬取的数据
func (m *Manager) AutoSave(articles []models.Article) (string, error) {
	if len(articles) == 0 {
		return "", fmt.Errorf("没有数据需要保存")
	}

	// 生成文件名：scrape_YYYYMMDD_HHmmss.json
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("scrape_%s.json", timestamp)
	filepath := filepath.Join(m.dataDir, filename)

	// 统计公众号
	accountSet := make(map[string]bool)
	for _, article := range articles {
		accountSet[article.AccountName] = true
	}
	accounts := make([]string, 0, len(accountSet))
	for name := range accountSet {
		accounts = append(accounts, name)
	}
	sort.Strings(accounts)

	// 构建保存数据
	data := SavedData{
		Articles:   articles,
		SaveTime:   time.Now().Format("2006-01-02 15:04:05"),
		TotalCount: len(articles),
		Accounts:   accounts,
	}

	// 序列化为 JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化数据失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return "", fmt.Errorf("保存文件失败: %w", err)
	}

	logger.Infof("数据已自动保存到: %s (共 %d 篇文章)", filepath, len(articles))
	return filepath, nil
}

// DataFileInfo 数据文件信息
type DataFileInfo struct {
	Filename   string   `json:"filename"`
	FilePath   string   `json:"filepath"`
	SaveTime   string   `json:"saveTime"`
	TotalCount int      `json:"totalCount"`
	Accounts   []string `json:"accounts"`
	FileSize   int64    `json:"fileSize"`
}

// ListDataFiles 列出所有保存的数据文件
func (m *Manager) ListDataFiles() ([]DataFileInfo, error) {
	files, err := os.ReadDir(m.dataDir)
	if err != nil {
		return nil, fmt.Errorf("读取数据目录失败: %w", err)
	}

	var dataFiles []DataFileInfo
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(m.dataDir, file.Name())

		// 读取文件获取元数据
		fileData, err := os.ReadFile(filePath)
		if err != nil {
			logger.Warnf("读取文件失败 %s: %v", file.Name(), err)
			continue
		}

		var savedData SavedData
		if err := json.Unmarshal(fileData, &savedData); err != nil {
			logger.Warnf("解析文件失败 %s: %v", file.Name(), err)
			continue
		}

		fileInfo, _ := file.Info()
		dataFiles = append(dataFiles, DataFileInfo{
			Filename:   file.Name(),
			FilePath:   filePath,
			SaveTime:   savedData.SaveTime,
			TotalCount: savedData.TotalCount,
			Accounts:   savedData.Accounts,
			FileSize:   fileInfo.Size(),
		})
	}

	// 按保存时间倒序排序
	sort.Slice(dataFiles, func(i, j int) bool {
		return dataFiles[i].SaveTime > dataFiles[j].SaveTime
	})

	return dataFiles, nil
}

// LoadData 加载指定的数据文件
func (m *Manager) LoadData(filepath string) ([]models.Article, error) {
	fileData, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	var savedData SavedData
	if err := json.Unmarshal(fileData, &savedData); err != nil {
		return nil, fmt.Errorf("解析数据失败: %w", err)
	}

	logger.Infof("成功加载数据: %s (共 %d 篇文章)", filepath, len(savedData.Articles))
	return savedData.Articles, nil
}

// DeleteData 删除指定的数据文件
func (m *Manager) DeleteData(filepath string) error {
	if err := os.Remove(filepath); err != nil {
		return fmt.Errorf("删除文件失败: %w", err)
	}

	logger.Infof("已删除数据文件: %s", filepath)
	return nil
}
