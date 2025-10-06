package service

import (
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
)

// Knowledge Base DTO

// CreateKnowledgeBaseRequest 创建知识库请求
type CreateKnowledgeBaseRequest struct {
	Name             string  `json:"name" binding:"required"`
	EmbeddingModelID string  `json:"embedding_model_id" binding:"required"` // 必填，Embedding 模型 ID
	RerankModelID    *string `json:"rerank_model_id"`                       // 可选，Rerank 模型 ID
	ChunkSize        *int    `json:"chunk_size"`                            // 可选，不传则根据嵌入模型 max_context 自动设置
	ChunkOverlap     *int    `json:"chunk_overlap"`                         // 可选，不传则为 0（不重叠）
	ChunkStrategy    *string `json:"chunk_strategy"`                        // 可选，不传则为 "recursive"
}

// UpdateKnowledgeBaseRequest 更新知识库请求
type UpdateKnowledgeBaseRequest struct {
	Name *string `json:"name"`
}

// KnowledgeBaseResponse 知识库响应
type KnowledgeBaseResponse struct {
	// 公开字段（所有用户可见）
	ID            string `json:"id"`
	Name          string `json:"name"`
	IsOfficial    bool   `json:"is_official"`
	DocumentCount int64  `json:"document_count"`

	// 私有字段（仅所有者可见，官方知识库返回 nil）
	OwnerID          *string `json:"owner_id,omitempty"`
	EmbeddingModelID *string `json:"embedding_model_id,omitempty"`
	RerankModelID    *string `json:"rerank_model_id,omitempty"`
	ChunkSize        *int    `json:"chunk_size,omitempty"`
	ChunkOverlap     *int    `json:"chunk_overlap,omitempty"`
	ChunkStrategy    *string `json:"chunk_strategy,omitempty"`
	MilvusCollection *string `json:"milvus_collection,omitempty"`
	CreatedAt        *string `json:"created_at,omitempty"`
	UpdatedAt        *string `json:"updated_at,omitempty"`
}

// ListKnowledgeBasesRequest 知识库列表请求
type ListKnowledgeBasesRequest struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
}

// ListKnowledgeBasesResponse 知识库列表响应
type ListKnowledgeBasesResponse struct {
	Items      []*KnowledgeBaseResponse `json:"items"`
	Pagination *PaginationResponse      `json:"pagination"`
}

// PaginationResponse 分页响应
type PaginationResponse struct {
	Page      int   `json:"page"`
	PageSize  int   `json:"page_size"`
	Total     int64 `json:"total"`
	TotalPage int   `json:"total_page"`
}

// DocumentResponse 文档响应 (使用 biz 包中的公共类型)
type DocumentResponse = biz.DocumentResponse

// SearchResultItem 搜索结果项
type SearchResultItem struct {
	DocumentID string                 `json:"document_id"`
	Content    string                 `json:"content"`
	Score      float32                `json:"score"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// toDocumentResponse 使用公共转换函数
func toDocumentResponse(doc *biz.Document) *DocumentResponse {
	return biz.ToDocumentResponse(doc)
}

func toSearchResults(results []*biz.SearchResult) []SearchResultItem {
	items := make([]SearchResultItem, len(results))
	for i, result := range results {
		items[i] = SearchResultItem{
			DocumentID: result.DocumentID,
			Content:    result.Content,
			Score:      result.Score,
			Metadata:   result.Metadata,
		}
	}
	return items
}
