package llm

import (
	"context"

	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
)

// KnowledgeAdapter 适配器：将 knowledge 模块的搜索功能适配到 llm 模块
type KnowledgeAdapter struct {
	docUseCase *biz.DocumentUseCase
}

// NewKnowledgeAdapter 创建知识库适配器
func NewKnowledgeAdapter(docUseCase *biz.DocumentUseCase) *KnowledgeAdapter {
	return &KnowledgeAdapter{
		docUseCase: docUseCase,
	}
}

// SearchDocuments 实现 KnowledgeSearcher 接口
func (ka *KnowledgeAdapter) SearchDocuments(
	ctx context.Context,
	kbID, userID, query string,
	topK int,
) ([]*KnowledgeSearchResult, error) {
	// 调用 knowledge 模块的搜索功能
	results, err := ka.docUseCase.SearchDocuments(ctx, kbID, userID, query, topK)
	if err != nil {
		return nil, err
	}

	// 转换 biz.SearchResult 到 llm.KnowledgeSearchResult
	converted := make([]*KnowledgeSearchResult, len(results))
	for i, result := range results {
		converted[i] = &KnowledgeSearchResult{
			DocumentID: result.DocumentID,
			Content:    result.Content,
			Score:      result.Score,
			Metadata:   result.Metadata,
		}
	}

	return converted, nil
}
