package models

// ProgressType 进度类型
type ProgressType string

const (
	ProgressTypeAccount ProgressType = "account" // 公众号进度
	ProgressTypeArticle ProgressType = "article" // 文章进度
	ProgressTypeContent ProgressType = "content" // 内容进度
)

// Progress 进度信息
type Progress struct {
	Type    ProgressType `json:"type"`    // 进度类型
	Current int          `json:"current"` // 当前值
	Total   int          `json:"total"`   // 总数
	Message string       `json:"message"` // 消息
}

// AccountStatus 公众号状态
type AccountStatus struct {
	AccountName  string `json:"accountName"`  // 公众号名称
	Status       string `json:"status"`       // 状态
	Message      string `json:"message"`      // 消息
	ArticleCount int    `json:"articleCount"` // 文章数
}
