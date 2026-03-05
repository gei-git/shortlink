package models

import (
	"time"

	"gorm.io/gorm"
)

type ShortLink struct {
	ID          uint   `gorm:"primaryKey"`
	ShortCode   string `gorm:"uniqueIndex;size:8;not null"` // 短码，唯一
	OriginalURL string `gorm:"size:2048;not null"`          // 原长链接
	ClickCount  int64  `gorm:"default:0"`
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type ClickLog struct {
	ID        uint   `gorm:"primaryKey"`
	ShortCode string `gorm:"index"`
	IPAddress string
	UserAgent string
	ClickedAt time.Time
}
