package types

// Model 表示 AI 模型信息
type Model struct {
	ID          string   `json:"id"`           // 模型 ID
	Object      string   `json:"object"`       // 对象类型，通常为 "model"
	Created     int64    `json:"created"`      // 创建时间戳
	OwnedBy     string   `json:"owned_by"`     // 所有者
	DisplayName string   `json:"display_name"` // 显示名称
	Description string   `json:"description"`  // 模型描述
	ContextWindow int    `json:"context_window"` // 上下文窗口大小
	MaxTokens   int      `json:"max_tokens"`   // 最大输出 tokens
	Capabilities []string `json:"capabilities"` // 能力列表（如 "chat", "embeddings"）
}

// ModelsResponse 模型列表响应
type ModelsResponse struct {
	Object string  `json:"object"` // "list"
	Data   []Model `json:"data"`   // 模型列表
}
