package types

import (
	"time"

	"github.com/google/uuid"
)

// KnowledgeBaseConfig 知识库配置
type KnowledgeBaseConfig struct {
	// Embedding 配置
	EmbeddingProvider   EmbeddingProvider `json:"embedding_provider"`
	EmbeddingModel      string            `json:"embedding_model"`
	EmbeddingDimensions int               `json:"embedding_dimensions"`

	// Chunking 配置
	ChunkSize     int           `json:"chunk_size"`
	ChunkOverlap  int           `json:"chunk_overlap"`
	ChunkStrategy ChunkStrategy `json:"chunk_strategy"`
}

// KnowledgeBase 知识库业务对象
type KnowledgeBase struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	UserID      uuid.UUID `json:"user_id"`

	Config KnowledgeBaseConfig `json:"config"`

	// Milvus Collection 名称
	MilvusCollection string `json:"milvus_collection"`

	// 统计信息
	DocumentCount int64 `json:"document_count"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// CreateKnowledgeBaseRequest 创建知识库请求
type CreateKnowledgeBaseRequest struct {
	Name        string              `json:"name" validate:"required,min=1,max=255"`
	Description string              `json:"description" validate:"max=1000"`
	Config      KnowledgeBaseConfig `json:"config" validate:"required"`
}

// UpdateKnowledgeBaseRequest 更新知识库请求
type UpdateKnowledgeBaseRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
}

// ListKnowledgeBasesRequest 列表查询请求
type ListKnowledgeBasesRequest struct {
	UserID uuid.UUID `json:"user_id" validate:"required"`
	Page   int       `json:"page" validate:"min=1"`
	Size   int       `json:"size" validate:"min=1,max=100"`
}

// ListKnowledgeBasesResponse 列表查询响应
type ListKnowledgeBasesResponse struct {
	Items      []*KnowledgeBase `json:"items"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	Size       int              `json:"size"`
	TotalPages int              `json:"total_pages"`
}
