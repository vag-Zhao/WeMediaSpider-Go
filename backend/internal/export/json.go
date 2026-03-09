package export

import (
	"encoding/json"
	"os"

	"WeMediaSpider/backend/internal/models"
	"WeMediaSpider/backend/pkg/logger"
)

// JSONExporter JSON 导出器
type JSONExporter struct{}

// Export 导出为 JSON
func (e *JSONExporter) Export(articles []models.Article, filename string) error {
	logger.Infof("📊 开始导出 JSON 文件: %s (文章数: %d)", filename, len(articles))

	file, err := os.Create(filename)
	if err != nil {
		logger.Errorf("❌ 创建 JSON 文件失败: %v", err)
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	logger.Infof("📝 正在编码 JSON 数据...")

	if err := encoder.Encode(articles); err != nil {
		logger.Errorf("❌ 编码 JSON 数据失败: %v", err)
		return err
	}

	logger.Infof("✅ JSON 文件导出成功: %s", filename)
	return nil
}
