package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/models"
	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
	"gorm.io/gorm"
)

// KnowledgeBaseRepository 知识库仓储接口
type KnowledgeBaseRepository interface {
	// Create 创建知识库
	Create(ctx context.Context, kb *models.KnowledgeBase) error

	// GetByID 根据 ID 获取知识库
	GetByID(ctx context.Context, id uuid.UUID) (*models.KnowledgeBase, error)

	// GetByUserID 获取用户的知识库列表
	GetByUserID(ctx context.Context, userID uuid.UUID, page, size int) ([]*models.KnowledgeBase, int64, error)

	// Update 更新知识库
	Update(ctx context.Context, kb *models.KnowledgeBase) error

	// Delete 软删除知识库
	Delete(ctx context.Context, id uuid.UUID) error

	// IncrementDocumentCount 增减文档数量
	IncrementDocumentCount(ctx context.Context, id uuid.UUID, delta int) error
}

// knowledgeBaseRepository 知识库仓储实现
type knowledgeBaseRepository struct {
	db *database.DB
}

// NewKnowledgeBaseRepository 创建知识库仓储
func NewKnowledgeBaseRepository(db *database.DB) KnowledgeBaseRepository {
	return &knowledgeBaseRepository{
		db: db,
	}
}

// Create 创建知识库
func (r *knowledgeBaseRepository) Create(ctx context.Context, kb *models.KnowledgeBase) error {
	if err := kb.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(kb).Error; err != nil {
		return fmt.Errorf("failed to create knowledge base: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取知识库
func (r *knowledgeBaseRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.KnowledgeBase, error) {
	var kb models.KnowledgeBase
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&kb).Error; err != nil {
		return nil, fmt.Errorf("failed to get knowledge base: %w", err)
	}
	return &kb, nil
}

// GetByUserID 获取用户的知识库列表
func (r *knowledgeBaseRepository) GetByUserID(ctx context.Context, userID uuid.UUID, page, size int) ([]*models.KnowledgeBase, int64, error) {
	var kbs []*models.KnowledgeBase
	var total int64

	// 计算偏移量
	offset := (page - 1) * size

	// 查询总数
	if err := r.db.WithContext(ctx).Model(&models.KnowledgeBase{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count knowledge bases: %w", err)
	}

	// 查询列表
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(size).
		Offset(offset).
		Find(&kbs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list knowledge bases: %w", err)
	}

	return kbs, total, nil
}

// Update 更新知识库
func (r *knowledgeBaseRepository) Update(ctx context.Context, kb *models.KnowledgeBase) error {
	if err := kb.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := r.db.WithContext(ctx).Save(kb).Error; err != nil {
		return fmt.Errorf("failed to update knowledge base: %w", err)
	}

	return nil
}

// Delete 软删除知识库
func (r *knowledgeBaseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.KnowledgeBase{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete knowledge base: %w", err)
	}
	return nil
}

// IncrementDocumentCount 增减文档数量
func (r *knowledgeBaseRepository) IncrementDocumentCount(ctx context.Context, id uuid.UUID, delta int) error {
	if err := r.db.WithContext(ctx).Model(&models.KnowledgeBase{}).
		Where("id = ?", id).
		UpdateColumn("document_count", gorm.Expr("document_count + ?", delta)).Error; err != nil {
		return fmt.Errorf("failed to increment document count: %w", err)
	}
	return nil
}

// ToBusinessType 将模型转换为业务类型
func ToKnowledgeBaseBusinessType(kb *models.KnowledgeBase) *kbtypes.KnowledgeBase {
	if kb == nil {
		return nil
	}

	return &kbtypes.KnowledgeBase{
		ID:          kb.ID,
		Name:        kb.Name,
		Description: kb.Description,
		UserID:      kb.UserID,
		Config: kbtypes.KnowledgeBaseConfig{
			EmbeddingProvider:   kbtypes.EmbeddingProvider(kb.EmbeddingProvider),
			EmbeddingModel:      kb.EmbeddingModel,
			EmbeddingDimensions: kb.EmbeddingDimensions,
			ChunkSize:           kb.ChunkSize,
			ChunkOverlap:        kb.ChunkOverlap,
			ChunkStrategy:       kbtypes.ChunkStrategy(kb.ChunkStrategy),
		},
		MilvusCollection: kb.MilvusCollection,
		DocumentCount:    kb.DocumentCount,
		CreatedAt:        kb.CreatedAt,
		UpdatedAt:        kb.UpdatedAt,
		DeletedAt:        kb.DeletedAt,
	}
}
