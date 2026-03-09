package models

// AppData 应用数据
type AppData struct {
	TotalArticles   int      `json:"totalArticles"`
	TotalAccounts   int      `json:"totalAccounts"`
	AccountNames    []string `json:"accountNames"`
	LastUpdateTime  string   `json:"lastUpdateTime"`
	TotalImages     int      `json:"totalImages"`
	LastScrapeTime  string   `json:"lastScrapeTime"`
	TotalExports    int      `json:"totalExports"`
	TodayArticles   int      `json:"todayArticles"`
	LastScrapeDate  string   `json:"lastScrapeDate"` // 最后爬取日期，用于判断是否是今天
}
