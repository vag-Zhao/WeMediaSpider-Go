package models

// Account 公众号结构
type Account struct {
	Name        string `json:"name"`        // 公众号名称
	Fakeid      string `json:"fakeid"`      // 公众号 fakeid
	Alias       string `json:"alias"`       // 公众号别名
	Signature   string `json:"signature"`   // 公众号签名
	Avatar      string `json:"avatar"`      // 头像
	QRCode      string `json:"qrCode"`      // 二维码
	ServiceType int    `json:"serviceType"` // 服务类型
}

// SearchResult 搜索结果
type SearchResult struct {
	Total    int       `json:"total"`
	Accounts []Account `json:"accounts"`
}
