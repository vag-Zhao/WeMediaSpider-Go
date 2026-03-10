package database

import (
	"time"

	"WeMediaSpider/backend/internal/database/models"
	appmodels "WeMediaSpider/backend/internal/models"
)

// ConvertToDBArticle 将业务模型转换为数据库模型
func ConvertToDBArticle(art *appmodels.Article, accountID uint) *models.Article {
	return &models.Article{
		ArticleID:        art.ID,
		AccountID:        accountID,
		AccountFakeid:    art.AccountFakeid,
		AccountName:      art.AccountName,
		Title:            art.Title,
		Link:             art.Link,
		Digest:           art.Digest,
		Content:          art.Content,
		PublishTime:      art.PublishTime,
		PublishTimestamp: art.PublishTimestamp,
		CreatedAt:        art.CreatedAt,
	}
}

// ConvertToAppArticle 将数据库模型转换为业务模型
func ConvertToAppArticle(dbArt *models.Article) appmodels.Article {
	return appmodels.Article{
		ID:               dbArt.ArticleID,
		AccountName:      dbArt.AccountName,
		AccountFakeid:    dbArt.AccountFakeid,
		Title:            dbArt.Title,
		Link:             dbArt.Link,
		Digest:           dbArt.Digest,
		Content:          dbArt.Content,
		PublishTime:      dbArt.PublishTime,
		PublishTimestamp: dbArt.PublishTimestamp,
		CreatedAt:        dbArt.CreatedAt,
	}
}

// ConvertToAppArticles 批量转换数据库模型为业务模型
func ConvertToAppArticles(dbArticles []*models.Article) []appmodels.Article {
	articles := make([]appmodels.Article, len(dbArticles))
	for i, dbArt := range dbArticles {
		articles[i] = ConvertToAppArticle(dbArt)
	}
	return articles
}

// ConvertToAppData 将数据库统计模型转换为业务模型
func ConvertToAppData(stats *models.AppStats, accountNames []string) appmodels.AppData {
	return appmodels.AppData{
		TotalArticles:  stats.TotalArticles,
		TotalAccounts:  stats.TotalAccounts,
		TotalImages:    stats.TotalImages,
		TotalExports:   stats.TotalExports,
		TodayArticles:  stats.TodayArticles,
		LastScrapeDate: stats.LastScrapeDate,
		LastScrapeTime: stats.LastScrapeTime,
		LastUpdateTime: stats.LastUpdateTime,
		AccountNames:   accountNames,
	}
}

// CalculateTodayArticles 计算今日文章数
func CalculateTodayArticles(articles []appmodels.Article) int {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	todayArticles := 0
	for _, article := range articles {
		publishTime := time.Unix(article.PublishTimestamp, 0)
		if publishTime.After(todayStart) || publishTime.Equal(todayStart) {
			todayArticles++
		}
	}
	return todayArticles
}

// ExtractAccountNames 从文章列表中提取公众号名称
func ExtractAccountNames(articles []appmodels.Article) []string {
	accountSet := make(map[string]bool)
	for _, article := range articles {
		accountSet[article.AccountName] = true
	}

	accountNames := make([]string, 0, len(accountSet))
	for name := range accountSet {
		accountNames = append(accountNames, name)
	}
	return accountNames
}
