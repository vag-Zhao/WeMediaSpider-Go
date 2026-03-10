package models

import "time"

// Account 公众号模型
type Account struct {
	ID        uint      `gorm:"primaryKey"`
	Fakeid    string    `gorm:"uniqueIndex;size:64;not null"`
	Name      string    `gorm:"size:255;not null;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Articles  []Article `gorm:"foreignKey:AccountID"`
}

// TableName 指定表名
func (Account) TableName() string {
	return "accounts"
}
