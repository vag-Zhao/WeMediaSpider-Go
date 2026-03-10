package models

import "time"

// AppStats 应用统计模型（单例模式）
type AppStats struct {
	ID             uint   `gorm:"primaryKey;check:id=1"`
	TotalArticles  int    `gorm:"default:0"`
	TotalAccounts  int    `gorm:"default:0"`
	TotalImages    int    `gorm:"default:0"`
	TotalExports   int    `gorm:"default:0"`
	TodayArticles  int    `gorm:"default:0"`
	LastScrapeDate string `gorm:"size:32"`
	LastScrapeTime string `gorm:"size:32"`
	LastUpdateTime string `gorm:"size:32"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TableName 指定表名
func (AppStats) TableName() string {
	return "app_stats"
}
