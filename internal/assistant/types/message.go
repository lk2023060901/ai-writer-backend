package types

import "time"

// Message represents a message in a conversation topic
type Message struct {
	ID            string         `json:"id"`
	TopicID       string         `json:"topic_id"`
	Role          string         `json:"role"` // user | assistant
	ContentBlocks []ContentBlock `json:"content_blocks"`
	TokenCount    *int           `json:"token_count,omitempty"`
	Provider      string         `json:"provider,omitempty"`      // AI provider (e.g., anthropic, openai)
	Model         string         `json:"model,omitempty"`         // AI model (e.g., claude-sonnet-4-5)
	CreatedAt     time.Time      `json:"created_at"`
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
