package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/models"
	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
)

// ChunkRepository 分块仓储接口
type ChunkRepository interface {
	// Create 创建分块
	Create(ctx context.Context, chunk *models.Chunk) error

	// BatchCreate 批量创建分块
	BatchCreate(ctx context.Context, chunks []*models.Chunk) error

	// GetByID 根据 ID 获取分块
	GetByID(ctx context.Context, id uuid.UUID) (*models.Chunk, error)

	// GetByDocumentID 获取文档的所有分块
	GetByDocumentID(ctx context.Context, docID uuid.UUID, page, size int) ([]*models.Chunk, int64, error)

	// DeleteByDocumentID 删除文档的所有分块
	DeleteByDocumentID(ctx context.Context, docID uuid.UUID) error

	// GetByMilvusID 根据 Milvus ID 获取分块
	GetByMilvusID(ctx context.Context, milvusID string) (*models.Chunk, error)

	// BatchGetByMilvusIDs 批量根据 Milvus ID 获取分块
	BatchGetByMilvusIDs(ctx context.Context, milvusIDs []string) ([]*models.Chunk, error)
}

// chunkRepository 分块仓储实现
type chunkRepository struct {
	db *database.DB
}

// NewChunkRepository 创建分块仓储
func NewChunkRepository(db *database.DB) ChunkRepository {
	return &chunkRepository{
		db: db,
	}
}

// Create 创建分块
func (r *chunkRepository) Create(ctx context.Context, chunk *models.Chunk) error {
	if err := chunk.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(chunk).Error; err != nil {
		return fmt.Errorf("failed to create chunk: %w", err)
	}

	return nil
}

// BatchCreate 批量创建分块
func (r *chunkRepository) BatchCreate(ctx context.Context, chunks []*models.Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	// 验证所有分块
	for _, chunk := range chunks {
		if err := chunk.Validate(); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// 批量插入
	if err := r.db.WithContext(ctx).CreateInBatches(chunks, 100).Error; err != nil {
		return fmt.Errorf("failed to batch create chunks: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取分块
func (r *chunkRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Chunk, error) {
	var chunk models.Chunk
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&chunk).Error; err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}
	return &chunk, nil
}

// GetByDocumentID 获取文档的所有分块
func (r *chunkRepository) GetByDocumentID(ctx context.Context, docID uuid.UUID, page, size int) ([]*models.Chunk, int64, error) {
	var chunks []*models.Chunk
	var total int64

	offset := (page - 1) * size

	// 查询总数
	if err := r.db.WithContext(ctx).Model(&models.Chunk{}).
		Where("document_id = ?", docID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count chunks: %w", err)
	}

	// 查询列表（按 chunk_index 排序）
	if err := r.db.WithContext(ctx).
		Where("document_id = ?", docID).
		Order("chunk_index ASC").
		Limit(size).
		Offset(offset).
		Find(&chunks).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list chunks: %w", err)
	}

	return chunks, total, nil
}

// DeleteByDocumentID 删除文档的所有分块
func (r *chunkRepository) DeleteByDocumentID(ctx context.Context, docID uuid.UUID) error {
	if err := r.db.WithContext(ctx).
		Where("document_id = ?", docID).
		Delete(&models.Chunk{}).Error; err != nil {
		return fmt.Errorf("failed to delete chunks: %w", err)
	}
	return nil
}

// GetByMilvusID 根据 Milvus ID 获取分块
func (r *chunkRepository) GetByMilvusID(ctx context.Context, milvusID string) (*models.Chunk, error) {
	var chunk models.Chunk
	if err := r.db.WithContext(ctx).Where("milvus_id = ?", milvusID).First(&chunk).Error; err != nil {
		return nil, fmt.Errorf("failed to get chunk by milvus id: %w", err)
	}
	return &chunk, nil
}

// BatchGetByMilvusIDs 批量根据 Milvus ID 获取分块
func (r *chunkRepository) BatchGetByMilvusIDs(ctx context.Context, milvusIDs []string) ([]*models.Chunk, error) {
	if len(milvusIDs) == 0 {
		return []*models.Chunk{}, nil
	}

	var chunks []*models.Chunk
	if err := r.db.WithContext(ctx).
		Where("milvus_id IN ?", milvusIDs).
		Find(&chunks).Error; err != nil {
		return nil, fmt.Errorf("failed to batch get chunks: %w", err)
	}

	return chunks, nil
}

// ToChunkBusinessType 将模型转换为业务类型
func ToChunkBusinessType(chunk *models.Chunk) *kbtypes.Chunk {
	if chunk == nil {
		return nil
	}

	return &kbtypes.Chunk{
		ID:              chunk.ID,
		DocumentID:      chunk.DocumentID,
		KnowledgeBaseID: chunk.KnowledgeBaseID,
		ChunkIndex:      chunk.ChunkIndex,
		Content:         chunk.Content,
		TokenCount:      chunk.TokenCount,
		MilvusID:        chunk.MilvusID,
		Metadata:        chunk.Metadata,
		CreatedAt:       chunk.CreatedAt,
	}
}
