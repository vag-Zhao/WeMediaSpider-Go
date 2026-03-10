package repository

import (
	"time"

	"WeMediaSpider/backend/internal/database/models"

	"gorm.io/gorm"
)

// StatsRepository 统计仓储接口
type StatsRepository interface {
	Get() (*models.AppStats, error)
	Update(stats *models.AppStats) error
	IncrementExports() error
	IncrementImages(count int) error
	UpdateArticleStats(totalArticles, totalAccounts, todayArticles int, lastScrapeTime string) error
}

// StatsRepositoryImpl 统计仓储实现
type StatsRepositoryImpl struct {
	db *gorm.DB
}

// NewStatsRepository 创建统计仓储
func NewStatsRepository(db *gorm.DB) StatsRepository {
	return &StatsRepositoryImpl{db: db}
}

// Get 获取统计信息
func (r *StatsRepositoryImpl) Get() (*models.AppStats, error) {
	var stats models.AppStats
	err := r.db.First(&stats, 1).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// Update 更新统计信息
func (r *StatsRepositoryImpl) Update(stats *models.AppStats) error {
	return r.db.Save(stats).Error
}

// IncrementExports 增加导出次数
func (r *StatsRepositoryImpl) IncrementExports() error {
	return r.db.Model(&models.AppStats{}).Where("id = ?", 1).
		UpdateColumn("total_exports", gorm.Expr("total_exports + ?", 1)).Error
}

// IncrementImages 增加图片下载数
func (r *StatsRepositoryImpl) IncrementImages(count int) error {
	return r.db.Model(&models.AppStats{}).Where("id = ?", 1).
		UpdateColumn("total_images", gorm.Expr("total_images + ?", count)).Error
}

// UpdateArticleStats 更新文章统计
func (r *StatsRepositoryImpl) UpdateArticleStats(totalArticles, totalAccounts, todayArticles int, lastScrapeTime string) error {
	now := time.Now()
	today := now.Format("2006-01-02")

	updates := map[string]interface{}{
		"total_articles":   totalArticles,
		"total_accounts":   totalAccounts,
		"today_articles":   todayArticles,
		"last_scrape_date": today,
		"last_scrape_time": lastScrapeTime,
		"last_update_time": lastScrapeTime,
	}

	return r.db.Model(&models.AppStats{}).Where("id = ?", 1).Updates(updates).Error
}
