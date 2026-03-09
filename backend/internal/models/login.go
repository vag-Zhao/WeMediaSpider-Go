package models

import "time"

// LoginStatus 登录状态
type LoginStatus struct {
	IsLoggedIn       bool      `json:"isLoggedIn"`
	LoginTime        time.Time `json:"loginTime,omitempty"`
	ExpireTime       time.Time `json:"expireTime,omitempty"`
	HoursSinceLogin  float64   `json:"hoursSinceLogin,omitempty"`
	HoursUntilExpire float64   `json:"hoursUntilExpire,omitempty"`
	Token            string    `json:"token,omitempty"`
	Message          string    `json:"message"`
}

// LoginCache 登录缓存
type LoginCache struct {
	Token     string            `json:"token"`
	Cookies   map[string]string `json:"cookies"`
	Timestamp int64             `json:"timestamp"`
}
