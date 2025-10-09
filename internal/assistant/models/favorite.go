package models

import (
	"time"
)

// AssistantFavorite is the GORM model for assistant_favorites table
type AssistantFavorite struct {
	ID          string    `gorm:"primaryKey;type:varchar(36)"`
	UserID      string    `gorm:"type:varchar(36);not null;index:idx_user_assistant,unique"`
	AssistantID string    `gorm:"type:varchar(36);not null;index:idx_user_assistant,unique;index"`
	SortOrder   int       `gorm:"not null;default:0"`
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// TableName specifies the table name
func (AssistantFavorite) TableName() string {
	return "assistant_favorites"
}
