package repository

import (
	"fmt"
	"time"

	"WeMediaSpider/backend/internal/database/models"

	"gorm.io/gorm"
)

// ArticleRepository 文章仓储接口
type ArticleRepository interface {
	Create(article *models.Article) error
	BatchCreate(articles []*models.Article) error
	FindByID(articleID string) (*models.Article, error)
	FindByAccountFakeid(fakeid string, limit, offset int) ([]*models.Article, error)
	FindByDateRange(startDate, endDate time.Time, limit, offset int) ([]*models.Article, error)
	Search(keyword string, limit, offset int) ([]*models.Article, error)
	Count() (int64, error)
	CountByAccount(fakeid string) (int64, error)
	Delete(articleID string) error
	GetAllArticles() ([]*models.Article, error)
}

// ArticleRepositoryImpl 文章仓储实现
type ArticleRepositoryImpl struct {
	db *gorm.DB
}

// NewArticleRepository 创建文章仓储
func NewArticleRepository(db *gorm.DB) ArticleRepository {
	return &ArticleRepositoryImpl{db: db}
}

// Create 创建文章
func (r *ArticleRepositoryImpl) Create(article *models.Article) error {
	return r.db.Create(article).Error
}

// BatchCreate 批量创建文章
func (r *ArticleRepositoryImpl) BatchCreate(articles []*models.Article) error {
	if len(articles) == 0 {
		return nil
	}

	// 使用事务批量插入
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 每批 500 条
		batchSize := 500
		for i := 0; i < len(articles); i += batchSize {
			end := i + batchSize
			if end > len(articles) {
				end = len(articles)
			}

			batch := articles[i:end]
			// 使用 Clauses 处理冲突（忽略重复）
			if err := tx.Clauses().CreateInBatches(batch, batchSize).Error; err != nil {
				return fmt.Errorf("failed to batch create articles: %w", err)
			}
		}
		return nil
	})
}

// FindByID 根据文章 ID 查找
func (r *ArticleRepositoryImpl) FindByID(articleID string) (*models.Article, error) {
	var article models.Article
	err := r.db.Where("article_id = ?", articleID).First(&article).Error
	if err != nil {
		return nil, err
	}
	return &article, nil
}

// FindByAccountFakeid 根据公众号 fakeid 查找文章
func (r *ArticleRepositoryImpl) FindByAccountFakeid(fakeid string, limit, offset int) ([]*models.Article, error) {
	var articles []*models.Article
	query := r.db.Where("account_fakeid = ?", fakeid).
		Order("publish_timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&articles).Error
	return articles, err
}

// FindByDateRange 根据日期范围查找文章
func (r *ArticleRepositoryImpl) FindByDateRange(startDate, endDate time.Time, limit, offset int) ([]*models.Article, error) {
	var articles []*models.Article
	startTimestamp := startDate.Unix()
	endTimestamp := endDate.Unix()

	query := r.db.Where("publish_timestamp >= ? AND publish_timestamp < ?", startTimestamp, endTimestamp).
		Order("publish_timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&articles).Error
	return articles, err
}

// Search 搜索文章（标题、摘要、内容）
func (r *ArticleRepositoryImpl) Search(keyword string, limit, offset int) ([]*models.Article, error) {
	var articles []*models.Article
	searchPattern := "%" + keyword + "%"

	query := r.db.Where("title LIKE ? OR digest LIKE ? OR content LIKE ?",
		searchPattern, searchPattern, searchPattern).
		Order("publish_timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&articles).Error
	return articles, err
}

// Count 统计文章总数
func (r *ArticleRepositoryImpl) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.Article{}).Count(&count).Error
	return count, err
}

// CountByAccount 统计指定公众号的文章数
func (r *ArticleRepositoryImpl) CountByAccount(fakeid string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Article{}).Where("account_fakeid = ?", fakeid).Count(&count).Error
	return count, err
}

// Delete 删除文章
func (r *ArticleRepositoryImpl) Delete(articleID string) error {
	return r.db.Where("article_id = ?", articleID).Delete(&models.Article{}).Error
}

// GetAllArticles 获取所有文章
func (r *ArticleRepositoryImpl) GetAllArticles() ([]*models.Article, error) {
	var articles []*models.Article
	err := r.db.Order("publish_timestamp DESC").Find(&articles).Error
	return articles, err
}
