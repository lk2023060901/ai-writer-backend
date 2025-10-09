package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Message is the GORM model for messages table
type Message struct {
	ID            string         `gorm:"primaryKey;type:uuid" json:"id"`
	TopicID       string         `gorm:"type:uuid;not null;index" json:"topic_id"`
	Role          string         `gorm:"type:varchar(20);not null" json:"role"` // user | assistant
	ContentBlocks ContentBlocks  `gorm:"type:jsonb;not null" json:"content_blocks"`
	TokenCount    *int           `gorm:"type:integer" json:"token_count,omitempty"`
	Provider      string         `gorm:"type:varchar(50)" json:"provider,omitempty"`      // AI provider
	Model         string         `gorm:"type:varchar(100)" json:"model,omitempty"`        // AI model
	CreatedAt     time.Time      `gorm:"not null" json:"created_at"`
}

// TableName specifies the table name
func (Message) TableName() string {
	return "messages"
}

// ContentBlock represents a single content block in a message
type ContentBlock struct {
	Type      string                 `json:"type"` // text | thinking | tool_use | tool_result
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	ToolUseID string                 `json:"tool_use_id,omitempty"`
	Content   string                 `json:"content,omitempty"`
}

// ContentBlocks is a custom type for []ContentBlock stored as JSONB
type ContentBlocks []ContentBlock

// Scan implements sql.Scanner interface
func (cb *ContentBlocks) Scan(value interface{}) error {
	if value == nil {
		*cb = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, cb)
}

// Value implements driver.Valuer interface
func (cb ContentBlocks) Value() (driver.Value, error) {
	if cb == nil {
		return nil, nil
	}
	return json.Marshal(cb)
}

// GetTextContent extracts plain text from content blocks
func (m *Message) GetTextContent() string {
	var texts []string
	for _, block := range m.ContentBlocks {
		if block.Type == "text" || block.Type == "thinking" {
			if block.Text != "" {
				texts = append(texts, block.Text)
			}
		}
	}

	// Join with newline if multiple blocks
	result := ""
	for i, text := range texts {
		if i > 0 {
			result += "\n"
		}
		result += text
	}
	return result
}
