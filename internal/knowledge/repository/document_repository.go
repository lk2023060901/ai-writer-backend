package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/models"
	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
)

// DocumentRepository 文档仓储接口
type DocumentRepository interface {
	// Create 创建文档
	Create(ctx context.Context, doc *models.Document) error

	// GetByID 根据 ID 获取文档
	GetByID(ctx context.Context, id uuid.UUID) (*models.Document, error)

	// GetByKnowledgeBaseID 获取知识库的文档列表
	GetByKnowledgeBaseID(ctx context.Context, kbID uuid.UUID, status kbtypes.DocumentStatus, page, size int) ([]*models.Document, int64, error)

	// Update 更新文档
	Update(ctx context.Context, doc *models.Document) error

	// Delete 删除文档
	Delete(ctx context.Context, id uuid.UUID) error

	// UpdateStatus 更新文档状态
	UpdateStatus(ctx context.Context, id uuid.UUID, status kbtypes.DocumentStatus, errorMsg string) error

	// GetByFileHash 根据文件哈希查找文档（去重）
	GetByFileHash(ctx context.Context, kbID uuid.UUID, fileHash string) (*models.Document, error)
}

// documentRepository 文档仓储实现
type documentRepository struct {
	db *database.DB
}

// NewDocumentRepository 创建文档仓储
func NewDocumentRepository(db *database.DB) DocumentRepository {
	return &documentRepository{
		db: db,
	}
}

// Create 创建文档
func (r *documentRepository) Create(ctx context.Context, doc *models.Document) error {
	if err := doc.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(doc).Error; err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取文档
func (r *documentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Document, error) {
	var doc models.Document
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&doc).Error; err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	return &doc, nil
}

// GetByKnowledgeBaseID 获取知识库的文档列表
func (r *documentRepository) GetByKnowledgeBaseID(ctx context.Context, kbID uuid.UUID, status kbtypes.DocumentStatus, page, size int) ([]*models.Document, int64, error) {
	var docs []*models.Document
	var total int64

	offset := (page - 1) * size

	query := r.db.WithContext(ctx).Model(&models.Document{}).Where("knowledge_base_id = ?", kbID)

	// 如果指定了状态，添加状态过滤
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 查询总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count documents: %w", err)
	}

	// 查询列表
	if err := query.
		Order("created_at DESC").
		Limit(size).
		Offset(offset).
		Find(&docs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list documents: %w", err)
	}

	return docs, total, nil
}

// Update 更新文档
func (r *documentRepository) Update(ctx context.Context, doc *models.Document) error {
	if err := doc.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := r.db.WithContext(ctx).Save(doc).Error; err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	return nil
}

// Delete 删除文档
func (r *documentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.Document{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	return nil
}

// UpdateStatus 更新文档状态
func (r *documentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status kbtypes.DocumentStatus, errorMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if errorMsg != "" {
		updates["error_message"] = errorMsg
	} else {
		updates["error_message"] = ""
	}

	if err := r.db.WithContext(ctx).Model(&models.Document{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update document status: %w", err)
	}

	return nil
}

// GetByFileHash 根据文件哈希查找文档
func (r *documentRepository) GetByFileHash(ctx context.Context, kbID uuid.UUID, fileHash string) (*models.Document, error) {
	var doc models.Document
	err := r.db.WithContext(ctx).
		Where("knowledge_base_id = ? AND file_hash = ?", kbID, fileHash).
		First(&doc).Error

	if err != nil {
		return nil, err
	}

	return &doc, nil
}

// ToDocumentBusinessType 将模型转换为业务类型
func ToDocumentBusinessType(doc *models.Document) *kbtypes.Document {
	if doc == nil {
		return nil
	}

	return &kbtypes.Document{
		ID:              doc.ID,
		KnowledgeBaseID: doc.KnowledgeBaseID,
		Filename:        doc.Filename,
		FileType:        kbtypes.FileType(doc.FileType),
		FileSize:        doc.FileSize,
		FileHash:        doc.FileHash,
		MinioBucket:     doc.MinioBucket,
		MinioObjectKey:  doc.MinioObjectKey,
		Status:          kbtypes.DocumentStatus(doc.Status),
		ErrorMessage:    doc.ErrorMessage,
		ChunkCount:      doc.ChunkCount,
		TokenCount:      doc.TokenCount,
		Metadata:        doc.Metadata,
		CreatedAt:       doc.CreatedAt,
		UpdatedAt:       doc.UpdatedAt,
	}
}
