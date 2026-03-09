package export

import (
	"WeMediaSpider/backend/internal/models"
)

// Exporter 导出器接口
type Exporter interface {
	Export(articles []models.Article, filename string) error
}

// GetExporter 获取导出器
func GetExporter(format string) Exporter {
	switch format {
	case "csv":
		return &CSVExporter{}
	case "json":
		return &JSONExporter{}
	case "excel":
		return &ExcelExporter{}
	case "markdown":
		return &MarkdownExporter{}
	default:
		return &JSONExporter{}
	}
}
