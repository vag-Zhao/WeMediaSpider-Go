package models

import "time"

// Article 文章结构
type Article struct {
	ID               string    `json:"id"`
	AccountName      string    `json:"accountName"`      // 公众号名称
	AccountFakeid    string    `json:"accountFakeid"`    // 公众号 fakeid
	Title            string    `json:"title"`            // 文章标题
	Link             string    `json:"link"`             // 文章链接
	Digest           string    `json:"digest"`           // 文章摘要
	Content          string    `json:"content"`          // 文章正文（Markdown）
	PublishTime      string    `json:"publishTime"`      // 格式化时间
	PublishTimestamp int64     `json:"publishTimestamp"` // 时间戳
	CreatedAt        time.Time `json:"createdAt"`        // 创建时间
}

// ArticleList 文章列表
type ArticleList struct {
	Total    int       `json:"total"`
	Articles []Article `json:"articles"`
}

// ArticleFilter 文章过滤条件
type ArticleFilter struct {
	AccountName string `json:"accountName,omitempty"`
	Keyword     string `json:"keyword,omitempty"`
	StartDate   string `json:"startDate,omitempty"`
	EndDate     string `json:"endDate,omitempty"`
}
