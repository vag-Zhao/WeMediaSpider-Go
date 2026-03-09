package models

// Config 应用配置
type Config struct {
	MaxPages         int    `json:"maxPages"`         // 最大页数
	RequestInterval  int    `json:"requestInterval"`  // 请求间隔（秒）
	MaxWorkers       int    `json:"maxWorkers"`       // 最大并发数
	IncludeContent   bool   `json:"includeContent"`   // 是否获取正文
	CacheExpireHours int    `json:"cacheExpireHours"` // 缓存过期时间
	OutputDir        string `json:"outputDir"`        // 输出目录
}

// ScrapeConfig 爬取配置
type ScrapeConfig struct {
	Accounts        []string `json:"accounts"`        // 公众号列表
	StartDate       string   `json:"startDate"`       // 开始日期 YYYY-MM-DD
	EndDate         string   `json:"endDate"`         // 结束日期 YYYY-MM-DD
	MaxPages        int      `json:"maxPages"`        // 最大页数
	RequestInterval int      `json:"requestInterval"` // 请求间隔
	IncludeContent  bool     `json:"includeContent"`  // 是否获取正文
	KeywordFilter   string   `json:"keywordFilter"`   // 关键词过滤
	MaxWorkers      int      `json:"maxWorkers"`      // 最大并发数
}
