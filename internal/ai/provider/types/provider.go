package types

import "context"

// Provider 定义统一的 AI Provider 接口（基于 OpenAI 协议）
type Provider interface {
	// CreateChatCompletion 创建聊天补全（同步）
	CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error)

	// CreateChatCompletionStream 创建聊天补全（流式）
	CreateChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (<-chan StreamChunk, error)

	// ListModels 获取可用模型列表
	ListModels(ctx context.Context) ([]Model, error)

	// Name 返回 Provider 名称
	Name() string

	// Close 关闭 Provider，释放资源
	Close() error
}
