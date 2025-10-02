package types

import "time"

// Agent represents a pre-configured AI agent template
type Agent struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Emoji       string         `json:"emoji"`
	Prompt      string         `json:"prompt"`       // System prompt template
	Group       []string       `json:"group"`        // Category tags like ["职业", "商业"]
	Settings    *AgentSettings `json:"settings"`     // Default settings
	IsBuiltin   bool           `json:"is_builtin"`   // Whether this is a system builtin agent
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// AgentSettings defines default configuration for an agent
type AgentSettings struct {
	Temperature      float64 `json:"temperature"`        // 0.0-2.0
	MaxTokens        int     `json:"max_tokens"`         // Max tokens in response
	ContextCount     int     `json:"context_count"`      // Number of context messages
	EnableWebSearch  bool    `json:"enable_web_search"`  // Enable web search capability
	ToolUseMode      string  `json:"tool_use_mode"`      // "function" or "prompt"
}

// AgentFilter defines filtering options for listing agents
type AgentFilter struct {
	Group     string `json:"group"`      // Filter by group
	IsBuiltin *bool  `json:"is_builtin"` // Filter by builtin status
	Keyword   string `json:"keyword"`    // Search by name or description
}
