package types

import "time"

// Topic represents a conversation topic within an assistant
type Topic struct {
	ID          string    `json:"id"`
	AssistantID string    `json:"assistant_id"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
