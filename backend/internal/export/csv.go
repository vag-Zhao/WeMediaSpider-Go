package export

import (
	"encoding/csv"
	"os"

	"WeMediaSpider/backend/internal/models"
	"WeMediaSpider/backend/pkg/logger"
)

// CSVExporter CSV 导出器
type CSVExporter struct{}

// Export 导出为 CSV
func (e *CSVExporter) Export(articles []models.Article, filename string) error {
	logger.Infof("📊 开始导出 CSV 文件: %s (文章数: %d)", filename, len(articles))

	file, err := os.Create(filename)
	if err != nil {
		logger.Errorf("❌ 创建 CSV 文件失败: %v", err)
		return err
	}
	defer file.Close()

	// 写入 UTF-8 BOM（Excel 兼容）
	file.Write([]byte{0xEF, 0xBB, 0xBF})
	logger.Infof("✅ 已写入 UTF-8 BOM 标记（Excel 兼容）")

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	headers := []string{
		"公众号名称",
		"文章标题",
		"文章链接",
		"发布时间",
		"正文内容",
	}
	if err := writer.Write(headers); err != nil {
		logger.Errorf("❌ 写入 CSV 表头失败: %v", err)
		return err
	}

	logger.Infof("📝 开始写入 %d 篇文章数据...", len(articles))

	// 写入数据
	for i, article := range articles {
		record := []string{
			article.AccountName,
			article.Title,
			article.Link,
			article.PublishTime,
			article.Content,
		}
		if err := writer.Write(record); err != nil {
			logger.Errorf("❌ 写入第 %d 篇文章失败: %v", i+1, err)
			return err
		}

		if (i+1)%10 == 0 || i == len(articles)-1 {
			logger.Infof("  已写入 %d/%d 篇文章", i+1, len(articles))
		}
	}

	logger.Infof("✅ CSV 文件导出成功: %s", filename)
	return nil
}
