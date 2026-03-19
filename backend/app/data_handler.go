package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"WeMediaSpider/backend/internal/database"
	dbmodels "WeMediaSpider/backend/internal/database/models"
	"WeMediaSpider/backend/internal/models"
	"WeMediaSpider/backend/pkg/logger"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetAppData() (models.AppData, error) {
	// 从数据库获取
	if a.db == nil || a.statsRepo == nil {
		return models.AppData{}, fmt.Errorf("database not initialized")
	}

	stats, err := a.statsRepo.Get()
	if err != nil {
		return models.AppData{}, fmt.Errorf("failed to get stats from database: %w", err)
	}

	// 获取公众号列表
	accounts, err := a.accountRepo.List()
	if err != nil {
		logger.Warnf("Failed to list accounts: %v", err)
		accounts = []*dbmodels.Account{}
	}

	accountNames := make([]string, len(accounts))
	for i, acc := range accounts {
		accountNames[i] = acc.Name
	}

	return database.ConvertToAppData(stats, accountNames), nil
}

// UpdateAppData 更新应用数据（已废弃，保留用于兼容）
func (a *App) UpdateAppData(articles []models.Article) error {
	logger.Warn("UpdateAppData is deprecated, stats are updated automatically")
	return nil
}

// ============================================================
// 图片下载相关
// ============================================================

func (a *App) ListDataFiles() ([]models.DataFileInfo, error) {
	// 从数据库获取
	if a.db == nil || a.articleRepo == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	articles, err := a.articleRepo.GetAllArticles()
	if err != nil {
		return nil, fmt.Errorf("failed to get articles from database: %w", err)
	}

	// 按公众号分组
	accountGroups := make(map[string][]models.Article)
	for _, dbArt := range articles {
		art := database.ConvertToAppArticle(dbArt)
		accountGroups[art.AccountFakeid] = append(accountGroups[art.AccountFakeid], art)
	}

	// 构建 DataFileInfo 列表
	var dataFiles []models.DataFileInfo
	for fakeid, arts := range accountGroups {
		if len(arts) == 0 {
			continue
		}

		// 获取公众号名称（使用第一篇文章的公众号名）
		accountName := arts[0].AccountName

		// 获取最早和最晚的发布时间
		var earliestTime, latestTime int64
		for i, art := range arts {
			if i == 0 {
				earliestTime = art.PublishTimestamp
				latestTime = art.PublishTimestamp
			} else {
				if art.PublishTimestamp < earliestTime {
					earliestTime = art.PublishTimestamp
				}
				if art.PublishTimestamp > latestTime {
					latestTime = art.PublishTimestamp
				}
			}
		}

		// 格式化时间范围
		earliestDate := time.Unix(earliestTime, 0).Format("2006-01-02")
		latestDate := time.Unix(latestTime, 0).Format("2006-01-02")
		timeRange := earliestDate
		if earliestDate != latestDate {
			timeRange = fmt.Sprintf("%s ~ %s", earliestDate, latestDate)
		}

		dataFiles = append(dataFiles, models.DataFileInfo{
			Filename:   fmt.Sprintf("%s.db", accountName),
			FilePath:   fakeid, // 使用 fakeid 作为标识
			SaveTime:   timeRange,
			TotalCount: len(arts),
			Accounts:   []string{accountName},
			FileSize:   0, // 数据库模式下不计算文件大小
		})
	}

	// 按文章数量倒序排序
	sort.Slice(dataFiles, func(i, j int) bool {
		return dataFiles[i].TotalCount > dataFiles[j].TotalCount
	})

	return dataFiles, nil
}

// LoadDataFile 加载指定公众号的文章
func (a *App) LoadDataFile(fakeidOrPath string) ([]models.Article, error) {
	// 从数据库加载
	if a.db == nil || a.articleRepo == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// 使用 fakeid 查询文章
	dbArticles, err := a.articleRepo.FindByAccountFakeid(fakeidOrPath, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to load articles from database: %w", err)
	}

	return database.ConvertToAppArticles(dbArticles), nil
}

// DeleteDataFile 删除指定公众号的所有文章
func (a *App) DeleteDataFile(fakeidOrPath string) error {
	// 从数据库删除
	if a.db == nil || a.articleRepo == nil {
		return fmt.Errorf("database not initialized")
	}

	// 查询该公众号的所有文章
	dbArticles, err := a.articleRepo.FindByAccountFakeid(fakeidOrPath, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to find articles: %w", err)
	}

	// 删除所有文章
	for _, article := range dbArticles {
		if err := a.articleRepo.Delete(article.ArticleID); err != nil {
			logger.Warnf("Failed to delete article %s: %v", article.ArticleID, err)
		}
	}

	logger.Infof("Deleted %d articles for account %s", len(dbArticles), fakeidOrPath)

	// 更新统计信息
	totalArticles, _ := a.articleRepo.Count()
	accounts, _ := a.accountRepo.List()
	if err := a.statsRepo.UpdateArticleStats(int(totalArticles), len(accounts), 0, ""); err != nil {
		logger.Warnf("Failed to update stats: %v", err)
	}

	return nil
}

// GetDataDirectory 获取数据目录路径
func (a *App) GetDataDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(homeDir, ".wemediaspider")
}

// OpenDataFileDialog 打开数据文件选择对话框（用于导入 JSON 文件）
func (a *App) OpenDataFileDialog() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	dataDir := filepath.Join(homeDir, ".wemediaspider", "data")

	filepath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "选择 JSON 数据文件导入",
		DefaultDirectory: dataDir,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "JSON 数据文件 (*.json)",
				Pattern:     "*.json",
			},
		},
	})

	if err != nil || filepath == "" {
		return "", fmt.Errorf("用户取消操作")
	}

	return filepath, nil
}

