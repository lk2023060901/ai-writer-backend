package types

import (
	"time"

	"github.com/google/uuid"
)

// Document 文档业务对象
type Document struct {
	ID              uuid.UUID `json:"id"`
	KnowledgeBaseID uuid.UUID `json:"knowledge_base_id"`

	Filename string   `json:"filename"`
	FileType FileType `json:"file_type"`
	FileSize int64    `json:"file_size"`
	FileHash string   `json:"file_hash"`

	// MinIO 存储路径
	MinioBucket    string `json:"minio_bucket"`
	MinioObjectKey string `json:"minio_object_key"`

	// 处理状态
	Status       DocumentStatus `json:"status"`
	ErrorMessage string         `json:"error_message,omitempty"`

	// 统计信息
	ChunkCount int `json:"chunk_count"`
	TokenCount int `json:"token_count"`

	// 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UploadDocumentRequest 上传文档请求
type UploadDocumentRequest struct {
	KnowledgeBaseID uuid.UUID              `json:"knowledge_base_id" validate:"required"`
	Filename        string                 `json:"filename" validate:"required,min=1,max=255"`
	FileType        FileType               `json:"file_type" validate:"required"`
	FileSize        int64                  `json:"file_size" validate:"required,min=1"`
	Content         []byte                 `json:"-"` // 文件内容（不序列化）
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ListDocumentsRequest 文档列表查询请求
type ListDocumentsRequest struct {
	KnowledgeBaseID uuid.UUID      `json:"knowledge_base_id" validate:"required"`
	Status          DocumentStatus `json:"status,omitempty"`
	Page            int            `json:"page" validate:"min=1"`
	Size            int            `json:"size" validate:"min=1,max=100"`
}

// ListDocumentsResponse 文档列表响应
type ListDocumentsResponse struct {
	Items      []*Document `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	Size       int         `json:"size"`
	TotalPages int         `json:"total_pages"`
}

// DeleteDocumentRequest 删除文档请求
type DeleteDocumentRequest struct {
	ID              uuid.UUID `json:"id" validate:"required"`
	KnowledgeBaseID uuid.UUID `json:"knowledge_base_id" validate:"required"`
}
