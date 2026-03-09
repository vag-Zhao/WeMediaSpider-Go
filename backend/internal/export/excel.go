package export

import (
	"fmt"
	"path/filepath"
	"strings"

	"WeMediaSpider/backend/internal/models"
	"WeMediaSpider/backend/pkg/logger"

	"github.com/xuri/excelize/v2"
)

// ExcelExporter Excel 导出器
type ExcelExporter struct{}

// Export 导出为 Excel
func (e *ExcelExporter) Export(articles []models.Article, filename string) error {
	logger.Infof("📊 开始导出 Excel 文件: %s (文章数: %d)", filename, len(articles))

	// 确保文件扩展名正确
	if !strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		ext := filepath.Ext(filename)
		if ext != "" {
			filename = strings.TrimSuffix(filename, ext) + ".xlsx"
		} else {
			filename = filename + ".xlsx"
		}
		logger.Infof("📝 修正文件扩展名为: %s", filename)
	}

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "文章列表"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		logger.Errorf("❌ 创建工作表失败: %v", err)
		return err
	}

	logger.Infof("✅ 创建工作表: %s", sheetName)

	// 设置表头
	headers := []string{"公众号名称", "文章标题", "文章链接", "发布时间", "正文内容"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// 设置表头样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 12,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#4472C4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	f.SetCellStyle(sheetName, "A1", fmt.Sprintf("%c1", 'A'+len(headers)-1), headerStyle)

	logger.Infof("📝 开始写入 %d 篇文章数据...", len(articles))

	// 写入数据
	for i, article := range articles {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), article.AccountName)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), article.Title)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), article.Link)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), article.PublishTime)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), article.Content)

		if (i+1)%10 == 0 || i == len(articles)-1 {
			logger.Infof("  已写入 %d/%d 篇文章", i+1, len(articles))
		}
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 20)
	f.SetColWidth(sheetName, "B", "B", 40)
	f.SetColWidth(sheetName, "C", "C", 50)
	f.SetColWidth(sheetName, "D", "D", 20)
	f.SetColWidth(sheetName, "E", "E", 60)

	// 设置活动工作表
	f.SetActiveSheet(index)

	// 删除默认的 Sheet1
	f.DeleteSheet("Sheet1")

	logger.Infof("💾 保存 Excel 文件: %s", filename)

	// 保存文件
	if err := f.SaveAs(filename); err != nil {
		logger.Errorf("❌ 保存 Excel 文件失败: %v", err)
		return err
	}

	logger.Infof("✅ Excel 文件导出成功: %s", filename)
	return nil
}
