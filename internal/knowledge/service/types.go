package service

import (
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
)

// AI Provider Config DTO

// CreateAIProviderConfigRequest 创建AI服务商配置请求
type CreateAIProviderConfigRequest struct {
	ProviderType        string `json:"provider_type" binding:"required"`
	ProviderName        string `json:"provider_name" binding:"required"`
	APIKey              string `json:"api_key" binding:"required"`
	APIBaseURL          string `json:"api_base_url"`
	EmbeddingModel      string `json:"embedding_model" binding:"required"`
	EmbeddingDimensions int    `json:"embedding_dimensions" binding:"required,min=1"`
}

// UpdateAIProviderConfigRequest 更新AI服务商配置请求
type UpdateAIProviderConfigRequest struct {
	ProviderName        *string `json:"provider_name"`
	APIKey              *string `json:"api_key"`
	APIBaseURL          *string `json:"api_base_url"`
	EmbeddingModel      *string `json:"embedding_model"`
	EmbeddingDimensions *int    `json:"embedding_dimensions"`
}

// AIProviderConfigResponse AI服务商配置响应
type AIProviderConfigResponse struct {
	// 公开字段（所有用户可见）
	ID           string `json:"id"`
	ProviderName string `json:"provider_name"`
	IsOfficial   bool   `json:"is_official"`

	// 私有字段（仅所有者可见，官方配置返回 nil）
	OwnerID             *string `json:"owner_id,omitempty"`
	ProviderType        *string `json:"provider_type,omitempty"`
	APIKey              *string `json:"api_key,omitempty"`
	APIBaseURL          *string `json:"api_base_url,omitempty"`
	EmbeddingModel      *string `json:"embedding_model,omitempty"`
	EmbeddingDimensions *int    `json:"embedding_dimensions,omitempty"`
	IsEnabled           *bool   `json:"is_enabled,omitempty"`
	CreatedAt           *string `json:"created_at,omitempty"`
	UpdatedAt           *string `json:"updated_at,omitempty"`
}

// Knowledge Base DTO

// CreateKnowledgeBaseRequest 创建知识库请求
type CreateKnowledgeBaseRequest struct {
	Name               string  `json:"name" binding:"required"`
	AIProviderConfigID string  `json:"ai_provider_config_id"`
	ChunkSize          *int    `json:"chunk_size"`     // 可选，不传则根据嵌入模型 max_context 自动设置
	ChunkOverlap       *int    `json:"chunk_overlap"`  // 可选，不传则为 0（不重叠）
	ChunkStrategy      *string `json:"chunk_strategy"` // 可选，不传则为 "recursive"
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
	OwnerID          *string                    `json:"owner_id,omitempty"`
	AIProviderConfig *AIProviderConfigResponse  `json:"ai_provider_config,omitempty"`
	ChunkSize        *int                       `json:"chunk_size,omitempty"`
	ChunkOverlap     *int                       `json:"chunk_overlap,omitempty"`
	ChunkStrategy    *string                    `json:"chunk_strategy,omitempty"`
	MilvusCollection *string                    `json:"milvus_collection,omitempty"`
	CreatedAt        *string                    `json:"created_at,omitempty"`
	UpdatedAt        *string                    `json:"updated_at,omitempty"`
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

// DocumentResponse 文档响应
type DocumentResponse struct {
	ID              string  `json:"id"`
	KnowledgeBaseID string  `json:"knowledge_base_id"`
	FileName        string  `json:"file_name"`
	FileType        string  `json:"file_type"`
	FileSize        int64   `json:"file_size"`
	ProcessStatus   string  `json:"process_status"`
	ProcessError    *string `json:"process_error,omitempty"`
	ChunkCount      int64   `json:"chunk_count"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// SearchResultItem 搜索结果项
type SearchResultItem struct {
	DocumentID string                 `json:"document_id"`
	Content    string                 `json:"content"`
	Score      float32                `json:"score"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

func toDocumentResponse(doc *biz.Document) *DocumentResponse {
	resp := &DocumentResponse{
		ID:              doc.ID,
		KnowledgeBaseID: doc.KnowledgeBaseID,
		FileName:        doc.FileName,
		FileType:        doc.FileType,
		FileSize:        doc.FileSize,
		ProcessStatus:   doc.ProcessStatus,
		ChunkCount:      doc.ChunkCount,
		CreatedAt:       doc.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:       doc.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	if doc.ProcessError != "" {
		resp.ProcessError = &doc.ProcessError
	}

	return resp
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
