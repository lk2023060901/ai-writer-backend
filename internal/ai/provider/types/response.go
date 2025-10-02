package types

import "strings"

// StopReason 停止原因
type StopReason string

const (
	StopReasonEndTurn   StopReason = "end_turn"   // 自然停止
	StopReasonMaxTokens StopReason = "max_tokens" // 达到 token 限制
	StopReasonPauseTurn StopReason = "pause_turn" // 暂停，可继续
	StopReasonStop      StopReason = "stop"       // 遇到停止序列
	StopReasonToolUse   StopReason = "tool_use"   // 工具调用
)

// ContentType 内容类型
type ContentType string

const (
	ContentTypeText     ContentType = "text"      // 普通文本回复
	ContentTypeThinking ContentType = "thinking"  // 思考过程
	ContentTypeToolUse  ContentType = "tool_use"  // 工具调用
	ContentTypeToolResult ContentType = "tool_result" // 工具结果
)

// ChatCompletionResponse 聊天补全响应（OpenAI 标准格式）
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 选择项
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"` // 统一使用 Message 结构
	FinishReason string  `json:"finish_reason"`
}

// Message 消息结构（用于请求和响应）
// 支持多种内容类型：text、thinking、tool_use 等
type Message struct {
	Role    string    `json:"role"`              // system, user, assistant
	Content string    `json:"content,omitempty"` // 简单文本内容（向后兼容）

	// 扩展内容（支持多种类型）
	ContentBlocks []ContentBlock `json:"content_blocks,omitempty"` // 多个内容块
}

// ContentBlock 内容块（支持不同类型）
type ContentBlock struct {
	Type ContentType `json:"type"` // text, thinking, tool_use, tool_result
	Text string      `json:"text,omitempty"` // 文本内容（type=text 或 thinking）

	// 工具相关（type=tool_use）
	ID       string                 `json:"id,omitempty"`    // 工具调用 ID
	Name     string                 `json:"name,omitempty"`  // 工具名称
	Input    map[string]interface{} `json:"input,omitempty"` // 工具输入

	// 工具结果（type=tool_result）
	ToolUseID string `json:"tool_use_id,omitempty"` // 对应的工具调用 ID
	IsError   bool   `json:"is_error,omitempty"`    // 是否为错误结果
}

// Usage Token 使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Done    bool           `json:"done"` // 是否结束
	Usage   *Usage         `json:"usage,omitempty"`
	Error   error          `json:"-"` // 错误（不序列化）
}

// StreamChoice 流式选择项
type StreamChoice struct {
	Index        int          `json:"index"`
	Delta        MessageDelta `json:"delta"`
	FinishReason *string      `json:"finish_reason"`
}

// MessageDelta 消息增量（流式）
type MessageDelta struct {
	Role         string              `json:"role,omitempty"`
	Content      string              `json:"content,omitempty"`      // 简单文本增量
	ContentBlock *ContentBlockDelta  `json:"content_block,omitempty"` // 内容块增量
}

// ContentBlockDelta 内容块增量
type ContentBlockDelta struct {
	Type  ContentType `json:"type"`           // 内容类型
	Index int         `json:"index"`          // 内容块索引
	Text  string      `json:"text,omitempty"` // 文本增量
}

// NeedsContinuation 判断响应是否需要继续（因为达到 max_tokens）
func (r *ChatCompletionResponse) NeedsContinuation() bool {
	if len(r.Choices) == 0 {
		return false
	}
	return r.Choices[0].FinishReason == string(StopReasonMaxTokens) ||
		r.Choices[0].FinishReason == string(StopReasonPauseTurn)
}

// GetTextContent 获取文本内容（优先从 Content 字段，然后从 ContentBlocks）
func (m *Message) GetTextContent() string {
	if m.Content != "" {
		return m.Content
	}

	// 从 ContentBlocks 中提取文本
	var texts []string
	for _, block := range m.ContentBlocks {
		if block.Type == ContentTypeText || block.Type == ContentTypeThinking {
			texts = append(texts, block.Text)
		}
	}

	if len(texts) > 0 {
		return strings.Join(texts, "\n")
	}

	return ""
}

// GetThinkingContent 获取 thinking 内容
func (m *Message) GetThinkingContent() string {
	for _, block := range m.ContentBlocks {
		if block.Type == ContentTypeThinking {
			return block.Text
		}
	}
	return ""
}

// HasToolUse 判断是否包含工具调用
func (m *Message) HasToolUse() bool {
	for _, block := range m.ContentBlocks {
		if block.Type == ContentTypeToolUse {
			return true
		}
	}
	return false
}
