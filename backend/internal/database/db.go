package database

import (
	"fmt"
	"path/filepath"
	"time"

	"WeMediaSpider/backend/internal/database/models"
	"WeMediaSpider/backend/pkg/logger"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Database 数据库管理器
type Database struct {
	*gorm.DB
	dbPath string
}

// NewDatabase 创建数据库实例
func NewDatabase(dataDir string) (*Database, error) {
	dbPath := filepath.Join(dataDir, "wemedia.db")
	logger.Infof("初始化数据库: %s", dbPath)

	// 配置 GORM
	config := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	// 打开数据库连接
	db, err := gorm.Open(sqlite.Open(dbPath), config)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 获取底层 SQL DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// 启用 WAL 模式（Write-Ahead Logging）
	db.Exec("PRAGMA journal_mode=WAL")
	// 设置缓存大小（10MB）
	db.Exec("PRAGMA cache_size=-10000")
	// 启用外键约束
	db.Exec("PRAGMA foreign_keys=ON")
	// 同步模式（NORMAL 平衡性能和安全）
	db.Exec("PRAGMA synchronous=NORMAL")

	logger.Info("数据库连接成功")

	return &Database{
		DB:     db,
		dbPath: dbPath,
	}, nil
}

// Close 关闭数据库连接
func (db *Database) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	logger.Info("关闭数据库连接")
	return sqlDB.Close()
}

// AutoMigrate 自动迁移表结构
func (db *Database) AutoMigrate() error {
	logger.Info("开始自动迁移数据库表结构")

	// 迁移表结构
	if err := db.DB.AutoMigrate(
		&models.Account{},
		&models.Article{},
		&models.AppStats{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// 初始化统计记录（如果不存在）
	var count int64
	db.DB.Model(&models.AppStats{}).Count(&count)
	if count == 0 {
		stats := &models.AppStats{ID: 1}
		if err := db.DB.Create(stats).Error; err != nil {
			return fmt.Errorf("failed to initialize app_stats: %w", err)
		}
		logger.Info("初始化应用统计记录")
	}

	logger.Info("数据库表结构迁移完成")
	return nil
}

// GetPath 获取数据库文件路径
func (db *Database) GetPath() string {
	return db.dbPath
}
