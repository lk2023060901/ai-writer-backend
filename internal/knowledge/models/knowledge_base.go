package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
)

// KnowledgeBase 知识库模型
type KnowledgeBase struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string    `gorm:"type:varchar(255);not null;index"`
	Description string    `gorm:"type:text"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index"`

	// Embedding 配置
	EmbeddingProvider   string `gorm:"type:varchar(50);not null"`  // openai, anthropic
	EmbeddingModel      string `gorm:"type:varchar(100);not null"` // text-embedding-3-small
	EmbeddingDimensions int    `gorm:"not null"`                   // 1536

	// Chunking 配置
	ChunkSize     int    `gorm:"not null;default:512"`
	ChunkOverlap  int    `gorm:"not null;default:50"`
	ChunkStrategy string `gorm:"type:varchar(50);not null;default:'token'"` // token, recursive

	// Milvus Collection 名称
	MilvusCollection string `gorm:"type:varchar(100);not null;unique_index"`

	// 统计信息
	DocumentCount int64 `gorm:"default:0"`

	// 时间戳
	CreatedAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
	DeletedAt *time.Time `gorm:"index"`

	// 关联
	Documents []Document `gorm:"foreignKey:KnowledgeBaseID;constraint:OnDelete:CASCADE"`
}

// TableName 指定表名
func (KnowledgeBase) TableName() string {
	return "knowledge_bases"
}

// Validate 验证知识库配置
func (kb *KnowledgeBase) Validate() error {
	if kb.Name == "" {
		return ErrInvalidName
	}

	if kb.UserID == uuid.Nil {
		return ErrInvalidUserID
	}

	provider := types.EmbeddingProvider(kb.EmbeddingProvider)
	if !provider.Valid() {
		return ErrInvalidEmbeddingProvider
	}

	if kb.EmbeddingModel == "" {
		return ErrInvalidEmbeddingModel
	}

	if kb.EmbeddingDimensions <= 0 {
		return ErrInvalidEmbeddingDimensions
	}

	strategy := types.ChunkStrategy(kb.ChunkStrategy)
	if !strategy.Valid() {
		return ErrInvalidChunkStrategy
	}

	if kb.ChunkSize <= 0 || kb.ChunkSize > 8192 {
		return ErrInvalidChunkSize
	}

	if kb.ChunkOverlap < 0 || kb.ChunkOverlap >= kb.ChunkSize {
		return ErrInvalidChunkOverlap
	}

	return nil
}

// IsDeleted 检查知识库是否已删除
func (kb *KnowledgeBase) IsDeleted() bool {
	return kb.DeletedAt != nil
}
