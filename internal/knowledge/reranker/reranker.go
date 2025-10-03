package reranker

import (
	"context"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
)

// Reranker 重排序接口
type Reranker interface {
	// Rerank 对搜索结果重排序
	Rerank(ctx context.Context, query string, results []*kbtypes.ChunkWithScore) ([]*kbtypes.ChunkWithScore, error)
}

// RerankProvider 重排序提供商
type RerankProvider string

const (
	// RerankProviderJina Jina AI Reranker
	RerankProviderJina RerankProvider = "jina"
	// RerankProviderVoyage Voyage AI Reranker
	RerankProviderVoyage RerankProvider = "voyage"
	// RerankProviderCohere Cohere Reranker
	RerankProviderCohere RerankProvider = "cohere"
	// RerankProviderSiliconFlow SiliconFlow Reranker
	RerankProviderSiliconFlow RerankProvider = "siliconflow"
)

// RerankResult 重排序结果
type RerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float32 `json:"relevance_score"`
}

// NoOpReranker 无操作重排序器（直接返回原结果）
type NoOpReranker struct{}

// NewNoOpReranker 创建无操作重排序器
func NewNoOpReranker() *NoOpReranker {
	return &NoOpReranker{}
}

// Rerank 直接返回原结果，不做重排序
func (r *NoOpReranker) Rerank(ctx context.Context, query string, results []*kbtypes.ChunkWithScore) ([]*kbtypes.ChunkWithScore, error) {
	return results, nil
}
