package embedding

import (
	"context"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
)

// Embedder 文本向量化接口
type Embedder interface {
	// Embed 对单个文本生成向量
	Embed(ctx context.Context, text string) ([]float32, error)

	// BatchEmbed 批量生成向量
	BatchEmbed(ctx context.Context, texts []string) ([][]float32, error)

	// Dimension 返回向量维度
	Dimension() int

	// Provider 返回 Provider 名称
	Provider() kbtypes.EmbeddingProvider

	// Model 返回模型名称
	Model() string
}

// EmbedRequest 向量化请求
type EmbedRequest struct {
	Texts []string
	Model string
}

// EmbedResponse 向量化响应
type EmbedResponse struct {
	Embeddings [][]float32
	Model      string
	Usage      *EmbedUsage
}

// EmbedUsage 向量化使用统计
type EmbedUsage struct {
	PromptTokens int
	TotalTokens  int
}
