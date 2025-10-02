package types

import "time"

// Assistant represents a user's AI assistant instance
type Assistant struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"` // Owner of this assistant

	// Basic information
	Name   string   `json:"name"`
	Emoji  string   `json:"emoji"`
	Prompt string   `json:"prompt"`
	Type   string   `json:"type"` // "assistant" or "translate"
	Tags   []string `json:"tags"` // User-defined tags

	// Feature toggles
	KnowledgeBaseIDs []string `json:"knowledge_base_ids"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AssistantSettings defines configuration for an assistant
type AssistantSettings struct {
	Temperature      float64                `json:"temperature"`
	TopP             float64                `json:"top_p"`
	MaxTokens        int                    `json:"max_tokens"`
	EnableMaxTokens  bool                   `json:"enable_max_tokens"`
	ContextCount     int                    `json:"context_count"`
	StreamOutput     bool                   `json:"stream_output"`
	ToolUseMode      string                 `json:"tool_use_mode"` // "function" or "prompt"
	CustomParameters map[string]interface{} `json:"custom_parameters"`
}

// AssistantFilter defines filtering options for listing assistants
type AssistantFilter struct {
	Tags    []string `json:"tags"`    // Filter by tags
	Keyword string   `json:"keyword"` // Search by name
}
