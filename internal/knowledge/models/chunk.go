package models

import (
	"time"

	"github.com/google/uuid"
)

// Chunk 分块模型
type Chunk struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	DocumentID      uuid.UUID `gorm:"type:uuid;not null;index"`
	KnowledgeBaseID uuid.UUID `gorm:"type:uuid;not null;index"`

	// 分块信息
	ChunkIndex int    `gorm:"not null"` // 块序号（从 0 开始）
	Content    string `gorm:"type:text;not null"`
	TokenCount int    `gorm:"not null"`

	// Milvus 向量 ID（与 chunk.ID 相同）
	MilvusID string `gorm:"type:varchar(100);not null;unique_index"`

	// 元数据（JSONB）
	Metadata map[string]interface{} `gorm:"type:jsonb"`

	// 时间戳
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`

	// 关联
	Document      *Document      `gorm:"foreignKey:DocumentID"`
	KnowledgeBase *KnowledgeBase `gorm:"foreignKey:KnowledgeBaseID"`
}

// TableName 指定表名
func (Chunk) TableName() string {
	return "chunks"
}

// Validate 验证分块
func (c *Chunk) Validate() error {
	if c.DocumentID == uuid.Nil {
		return ErrInvalidDocumentID
	}

	if c.KnowledgeBaseID == uuid.Nil {
		return ErrInvalidKnowledgeBaseID
	}

	if c.Content == "" {
		return ErrEmptyContent
	}

	if c.ChunkIndex < 0 {
		return ErrInvalidChunkIndex
	}

	if c.TokenCount <= 0 {
		return ErrInvalidTokenCount
	}

	if c.MilvusID == "" {
		return ErrInvalidMilvusID
	}

	return nil
}
