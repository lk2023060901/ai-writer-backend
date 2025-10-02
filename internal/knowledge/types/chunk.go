package types

import (
	"time"

	"github.com/google/uuid"
)

// Chunk 文本分块业务对象
type Chunk struct {
	ID              uuid.UUID `json:"id"`
	DocumentID      uuid.UUID `json:"document_id"`
	KnowledgeBaseID uuid.UUID `json:"knowledge_base_id"`

	ChunkIndex int    `json:"chunk_index"`
	Content    string `json:"content"`
	TokenCount int    `json:"token_count"`

	// Milvus 向量 ID
	MilvusID string `json:"milvus_id"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// ChunkWithScore 带分数的分块（用于搜索结果）
type ChunkWithScore struct {
	Chunk
	Score    float32 `json:"score"`     // 相似度分数
	Distance float32 `json:"distance"`  // 向量距离
	Reranked bool    `json:"reranked"`  // 是否经过重排序
}

// CreateChunkRequest 创建分块请求
type CreateChunkRequest struct {
	DocumentID      uuid.UUID              `json:"document_id" validate:"required"`
	KnowledgeBaseID uuid.UUID              `json:"knowledge_base_id" validate:"required"`
	ChunkIndex      int                    `json:"chunk_index" validate:"min=0"`
	Content         string                 `json:"content" validate:"required,min=1"`
	TokenCount      int                    `json:"token_count" validate:"min=1"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ListChunksRequest 分块列表查询请求
type ListChunksRequest struct {
	DocumentID uuid.UUID `json:"document_id" validate:"required"`
	Page       int       `json:"page" validate:"min=1"`
	Size       int       `json:"size" validate:"min=1,max=100"`
}

// ListChunksResponse 分块列表响应
type ListChunksResponse struct {
	Items      []*Chunk `json:"items"`
	Total      int64    `json:"total"`
	Page       int      `json:"page"`
	Size       int      `json:"size"`
	TotalPages int      `json:"total_pages"`
}
