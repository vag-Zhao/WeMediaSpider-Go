package models

import "time"

// Article 文章模型
type Article struct {
	ID               uint      `gorm:"primaryKey"`
	ArticleID        string    `gorm:"uniqueIndex;size:128;not null"`
	AccountID        uint      `gorm:"not null;index"`
	AccountFakeid    string    `gorm:"size:64;not null;index"`
	AccountName      string    `gorm:"size:255;not null"`
	Title            string    `gorm:"type:text;not null"`
	Link             string    `gorm:"type:text;not null"`
	Digest           string    `gorm:"type:text"`
	Content          string    `gorm:"type:text"`
	PublishTime      string    `gorm:"size:32"`
	PublishTimestamp int64     `gorm:"not null;index"`
	CreatedAt        time.Time `gorm:"index"`
	UpdatedAt        time.Time
	Account          Account   `gorm:"foreignKey:AccountID"`
}

// TableName 指定表名
func (Article) TableName() string {
	return "articles"
}
