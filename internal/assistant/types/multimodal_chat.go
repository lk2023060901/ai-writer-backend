package types

import "time"

// ChatRequest 多模态聊天请求（支持多服务商并发）
type ChatRequest struct {
	// 基础字段
	Message         string   `json:"message" binding:"required"`
	TopicID         string   `json:"topic_id,omitempty"` // 会话 ID，用于上下文
	KnowledgeBaseID string   `json:"knowledge_base_id,omitempty"`
	UserID          string   `json:"-"` // 用户 ID（从认证中间件设置，不从请求 JSON 中读取）

	// 多模态内容
	ContentBlocks   []MessageContentBlock `json:"content_blocks,omitempty"` // 富文本内容块

	// 多服务商配置（核心功能）
	Providers       []ProviderConfig `json:"providers" binding:"required,min=1"` // 至少选择一个服务商

	// 联网搜索
	EnableWebSearch bool             `json:"enable_web_search,omitempty"`
	SearchDepth     string           `json:"search_depth,omitempty"` // basic | advanced

	// 高级选项
	Temperature     *float64         `json:"temperature,omitempty"`
	MaxTokens       *int             `json:"max_tokens,omitempty"`
	SystemPrompt    string           `json:"system_prompt,omitempty"`
}

// MessageContentBlock 消息内容块（支持文本、图片、文件等）
type MessageContentBlock struct {
	Type string `json:"type" binding:"required"` // text | image | file | audio | video | web_search

	// 文本内容
	Text string `json:"text,omitempty"`

	// 图片内容
	ImageURL    string `json:"image_url,omitempty"`
	ImageDetail string `json:"image_detail,omitempty"` // auto | low | high (OpenAI)

	// 文件内容
	FileURL      string `json:"file_url,omitempty"`
	FileName     string `json:"file_name,omitempty"`
	FileMimeType string `json:"file_mime_type,omitempty"`
	FileSize     int64  `json:"file_size,omitempty"`

	// 联网搜索结果
	SearchQuery   string                 `json:"search_query,omitempty"`
	SearchResults []WebSearchResult      `json:"search_results,omitempty"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ProviderConfig 服务商配置
type ProviderConfig struct {
	// 服务商信息
	Provider string `json:"provider" binding:"required"` // openai | anthropic | gemini | grok | deepseek | qwen
	Model    string `json:"model" binding:"required"`    // gpt-4o, claude-3-5-sonnet-20241022, gemini-2.0-flash-exp, grok-2

	// 可选配置（覆盖全局配置）
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`

	// 高级选项（服务商特定）
	Options map[string]interface{} `json:"options,omitempty"`
}

// ChatResponse SSE 流式响应（多服务商并发返回）
type ChatResponse struct {
	// 响应标识
	SessionID  string `json:"session_id"`  // 会话 ID
	Provider   string `json:"provider"`    // 当前响应的服务商
	Model      string `json:"model"`       // 当前使用的模型

	// 响应内容
	EventType string                 `json:"event_type"` // start | token | done | error
	Content   string                 `json:"content,omitempty"`
	Index     int                    `json:"index,omitempty"`

	// 元数据
	TokenCount   *int                   `json:"token_count,omitempty"`
	FinishReason string                 `json:"finish_reason,omitempty"` // stop | length | content_filter
	Metadata     map[string]interface{} `json:"metadata,omitempty"`

	// 错误信息
	Error string `json:"error,omitempty"`

	// 时间戳
	Timestamp time.Time `json:"timestamp"`
}

// WebSearchResult 联网搜索结果
type WebSearchResult struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Snippet     string    `json:"snippet"`
	Content     string    `json:"content,omitempty"` // 完整内容（可选）
	Relevance   float64   `json:"relevance,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}

// MultiProviderResponse 多服务商响应汇总
type MultiProviderResponse struct {
	SessionID string                      `json:"session_id"`
	TopicID   string                      `json:"topic_id,omitempty"`
	Providers []ProviderResponseSummary   `json:"providers"` // 各服务商的响应摘要
	CreatedAt time.Time                   `json:"created_at"`
}

// ProviderResponseSummary 单个服务商的响应摘要
type ProviderResponseSummary struct {
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	Content      string    `json:"content"`
	TokenCount   int       `json:"token_count"`
	FinishReason string    `json:"finish_reason"`
	Duration     float64   `json:"duration"` // 响应耗时（秒）
	Error        string    `json:"error,omitempty"`
	CompletedAt  time.Time `json:"completed_at"`
}

// SupportedProviders 支持的服务商列表
var SupportedProviders = map[string][]string{
	"openai": {
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-3.5-turbo",
		"o1",
		"o1-mini",
	},
	"anthropic": {
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	},
	"gemini": {
		"gemini-2.0-flash-exp",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-1.0-pro",
	},
	"grok": {
		"grok-2-1212",
		"grok-2-vision-1212",
		"grok-beta",
	},
	"deepseek": {
		"deepseek-chat",
		"deepseek-reasoner",
	},
	"qwen": {
		"qwen-max",
		"qwen-plus",
		"qwen-turbo",
	},
}

// ValidateProvider 验证服务商和模型是否支持
func ValidateProvider(provider, model string) bool {
	models, exists := SupportedProviders[provider]
	if !exists {
		return false
	}

	for _, m := range models {
		if m == model {
			return true
		}
	}
	return false
}
