package errors

import "errors"

var (
	// 登录相关错误
	ErrNotLoggedIn  = errors.New("未登录")
	ErrLoginFailed  = errors.New("登录失败")
	ErrTokenExpired = errors.New("Token 已过期")
	ErrInvalidToken = errors.New("无效的 Token")

	// 爬取相关错误
	ErrAccountNotFound = errors.New("未找到公众号")
	ErrNoArticles      = errors.New("没有找到文章")
	ErrScrapeFailed    = errors.New("爬取失败")
	ErrScrapeCancelled = errors.New("爬取已取消")

	// 导出相关错误
	ErrExportFailed  = errors.New("导出失败")
	ErrInvalidFormat = errors.New("不支持的导出格式")

	// 配置相关错误
	ErrConfigNotFound = errors.New("配置文件不存在")
	ErrInvalidConfig  = errors.New("无效的配置")
)

// AppError 应用错误
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

// NewAppError 创建应用错误
func NewAppError(code, message, details string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
}
