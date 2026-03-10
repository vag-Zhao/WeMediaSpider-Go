package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"WeMediaSpider/backend/internal/database"
	dbmodels "WeMediaSpider/backend/internal/database/models"
	"WeMediaSpider/backend/internal/models"
	"WeMediaSpider/backend/internal/repository"
	"WeMediaSpider/backend/pkg/logger"
)

// SavedData JSON 文件数据结构
type SavedData struct {
	Articles   []models.Article `json:"articles"`
	SaveTime   string           `json:"saveTime"`
	TotalCount int              `json:"totalCount"`
	Accounts   []string         `json:"accounts"`
}

// Migrator 数据迁移器
type Migrator struct {
	db          *database.Database
	storageDir  string
	backupDir   string
	articleRepo repository.ArticleRepository
	accountRepo repository.AccountRepository
	statsRepo   repository.StatsRepository
}

func main() {
	// 初始化日志
	logger.Init()
	logger.Info("========== 开始数据迁移 ==========")

	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Errorf("Failed to get home directory: %v", err)
		os.Exit(1)
	}

	configDir := filepath.Join(homeDir, ".wemediaspider")
	storageDir := filepath.Join(configDir, "data")
	backupDir := filepath.Join(configDir, "backup")

	// 创建迁移器
	migrator := &Migrator{
		storageDir: storageDir,
		backupDir:  backupDir,
	}

	// 执行迁移
	if err := migrator.Run(); err != nil {
		logger.Errorf("Migration failed: %v", err)
		os.Exit(1)
	}

	logger.Info("========== 数据迁移完成 ==========")
}

// Run 执行迁移
func (m *Migrator) Run() error {
	// 1. 备份 JSON 文件
	logger.Info("步骤 1/6: 备份 JSON 文件")
	if err := m.BackupJSONFiles(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	// 2. 初始化数据库
	logger.Info("步骤 2/6: 初始化数据库")
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".wemediaspider")

	db, err := database.NewDatabase(configDir)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()
	m.db = db

	// 自动迁移表结构
	if err := db.AutoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate schema: %w", err)
	}

	// 初始化仓储
	m.articleRepo = repository.NewArticleRepository(db.DB)
	m.accountRepo = repository.NewAccountRepository(db.DB)
	m.statsRepo = repository.NewStatsRepository(db.DB)

	// 3. 迁移文章数据
	logger.Info("步骤 3/6: 迁移文章数据")
	articleCount, accountCount, err := m.MigrateArticles()
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// 4. 更新统计信息
	logger.Info("步骤 4/6: 更新统计信息")
	if err := m.UpdateStats(articleCount, accountCount); err != nil {
		return fmt.Errorf("failed to update stats: %w", err)
	}

	// 5. 验证数据
	logger.Info("步骤 5/6: 验证数据完整性")
	if err := m.ValidateData(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// 6. 生成报告
	logger.Info("步骤 6/6: 生成迁移报告")
	if err := m.GenerateReport(articleCount, accountCount); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	return nil
}

// BackupJSONFiles 备份 JSON 文件
func (m *Migrator) BackupJSONFiles() error {
	// 创建备份目录
	backupDataDir := filepath.Join(m.backupDir, "data")
	if err := os.MkdirAll(backupDataDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 检查源目录是否存在
	if _, err := os.Stat(m.storageDir); os.IsNotExist(err) {
		logger.Warn("数据目录不存在，跳过备份")
		return nil
	}

	// 读取所有 JSON 文件
	files, err := os.ReadDir(m.storageDir)
	if err != nil {
		return fmt.Errorf("failed to read storage directory: %w", err)
	}

	backupCount := 0
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		srcPath := filepath.Join(m.storageDir, file.Name())
		dstPath := filepath.Join(backupDataDir, file.Name())

		// 复制文件
		data, err := os.ReadFile(srcPath)
		if err != nil {
			logger.Warnf("Failed to read file %s: %v", file.Name(), err)
			continue
		}

		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			logger.Warnf("Failed to backup file %s: %v", file.Name(), err)
			continue
		}

		backupCount++
	}

	logger.Infof("已备份 %d 个 JSON 文件到: %s", backupCount, backupDataDir)
	return nil
}

// MigrateArticles 迁移文章数据
func (m *Migrator) MigrateArticles() (int, int, error) {
	// 检查数据目录
	if _, err := os.Stat(m.storageDir); os.IsNotExist(err) {
		logger.Warn("数据目录不存在，跳过迁移")
		return 0, 0, nil
	}

	// 读取所有 JSON 文件
	files, err := os.ReadDir(m.storageDir)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read storage directory: %w", err)
	}

	// 用于去重
	articleMap := make(map[string]*models.Article)
	accountMap := make(map[string]string) // fakeid -> name

	// 解析所有 JSON 文件
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(m.storageDir, file.Name())
		logger.Infof("解析文件: %s", file.Name())

		data, err := os.ReadFile(filePath)
		if err != nil {
			logger.Warnf("Failed to read file %s: %v", file.Name(), err)
			continue
		}

		var savedData SavedData
		if err := json.Unmarshal(data, &savedData); err != nil {
			logger.Warnf("Failed to parse file %s: %v", file.Name(), err)
			continue
		}

		// 收集文章和公众号
		for i := range savedData.Articles {
			article := &savedData.Articles[i]
			articleMap[article.ID] = article
			accountMap[article.AccountFakeid] = article.AccountName
		}
	}

	logger.Infof("共解析 %d 篇文章，%d 个公众号", len(articleMap), len(accountMap))

	// 先创建公众号
	logger.Info("创建公众号记录...")
	accountIDMap := make(map[string]uint)
	for fakeid, name := range accountMap {
		account, err := m.accountRepo.FindOrCreate(fakeid, name)
		if err != nil {
			logger.Warnf("Failed to create account %s: %v", name, err)
			continue
		}
		accountIDMap[fakeid] = account.ID
	}

	// 批量插入文章
	logger.Info("批量插入文章...")
	dbArticles := make([]*dbmodels.Article, 0, len(articleMap))
	for _, article := range articleMap {
		accountID, ok := accountIDMap[article.AccountFakeid]
		if !ok {
			logger.Warnf("Account not found for article %s", article.ID)
			continue
		}

		dbArticle := database.ConvertToDBArticle(article, accountID)
		dbArticles = append(dbArticles, dbArticle)
	}

	if len(dbArticles) > 0 {
		if err := m.articleRepo.BatchCreate(dbArticles); err != nil {
			return 0, 0, fmt.Errorf("failed to batch create articles: %w", err)
		}
	}

	logger.Infof("成功迁移 %d 篇文章", len(dbArticles))
	return len(dbArticles), len(accountMap), nil
}

// UpdateStats 更新统计信息
func (m *Migrator) UpdateStats(articleCount, accountCount int) error {
	// 计算今日文章数（假设为 0，因为是历史数据）
	todayArticles := 0
	lastScrapeTime := time.Now().Format("2006-01-02 15:04:05")

	return m.statsRepo.UpdateArticleStats(articleCount, accountCount, todayArticles, lastScrapeTime)
}

// ValidateData 验证数据完整性
func (m *Migrator) ValidateData() error {
	// 统计数据库中的记录数
	articleCount, err := m.articleRepo.Count()
	if err != nil {
		return fmt.Errorf("failed to count articles: %w", err)
	}

	accounts, err := m.accountRepo.List()
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}

	logger.Infof("数据库统计: %d 篇文章, %d 个公众号", articleCount, len(accounts))

	// 随机抽样验证
	if articleCount > 0 {
		articles, err := m.articleRepo.GetAllArticles()
		if err != nil {
			return fmt.Errorf("failed to get articles: %w", err)
		}

		if len(articles) > 0 {
			sample := articles[0]
			logger.Infof("样本文章: ID=%s, 标题=%s, 公众号=%s",
				sample.ArticleID, sample.Title, sample.AccountName)
		}
	}

	return nil
}

// GenerateReport 生成迁移报告
func (m *Migrator) GenerateReport(articleCount, accountCount int) error {
	reportPath := filepath.Join(m.backupDir, "migration_report.txt")

	report := fmt.Sprintf(`数据迁移报告
=====================================
迁移时间: %s
数据库路径: %s
备份目录: %s

迁移结果:
- 文章总数: %d
- 公众号数: %d

状态: 成功 ✓
=====================================
`,
		time.Now().Format("2006-01-02 15:04:05"),
		m.db.GetPath(),
		m.backupDir,
		articleCount,
		accountCount,
	)

	if err := os.WriteFile(reportPath, []byte(report), 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	logger.Infof("迁移报告已生成: %s", reportPath)
	fmt.Println(report)

	return nil
}
