package types

import "time"

// AssistantFavorite represents a user's favorite assistant (快捷访问列表)
type AssistantFavorite struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	AssistantID string    `json:"assistant_id"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
}

// AssistantFavoriteWithDetails includes assistant details
type AssistantFavoriteWithDetails struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	AssistantID string    `json:"assistant_id"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`

	// Assistant details
	AssistantName  string `json:"assistant_name"`
	AssistantEmoji string `json:"assistant_emoji"`
	AssistantType  string `json:"assistant_type"`
	AssistantTags  string `json:"assistant_tags,omitempty" gorm:"type:text"` // 改为 string，接收 JSON 字符串
}
