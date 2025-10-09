package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/models"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
	"gorm.io/gorm"
)

// FileStorageRepository 文件存储仓储接口
type FileStorageRepository interface {
	// Create 创建文件存储记录
	Create(ctx context.Context, fs *models.FileStorage) error

	// GetByHash 根据文件哈希获取记录
	GetByHash(ctx context.Context, fileHash string) (*models.FileStorage, error)

	// IncrementReference 增加引用计数
	IncrementReference(ctx context.Context, fileHash string) error

	// DecrementReference 减少引用计数
	DecrementReference(ctx context.Context, fileHash string) error

	// BatchDecrementReferences 批量减少引用计数
	BatchDecrementReferences(ctx context.Context, fileHashes []string) error

	// DeleteIfNoReferences 如果引用计数为0则删除记录
	DeleteIfNoReferences(ctx context.Context, fileHash string) (bool, error)

	// GetOrphaned 获取孤立文件（引用计数为0且超过保留期的文件）
	GetOrphaned(ctx context.Context, retentionDays int) ([]*models.FileStorage, error)
}

// fileStorageRepository 文件存储仓储实现
type fileStorageRepository struct {
	db *database.DB
}

// NewFileStorageRepository 创建文件存储仓储
func NewFileStorageRepository(db *database.DB) FileStorageRepository {
	return &fileStorageRepository{
		db: db,
	}
}

// Create 创建文件存储记录
func (r *fileStorageRepository) Create(ctx context.Context, fs *models.FileStorage) error {
	if err := r.db.WithContext(ctx).Create(fs).Error; err != nil {
		return fmt.Errorf("failed to create file storage: %w", err)
	}
	return nil
}

// GetByHash 根据文件哈希获取记录
func (r *fileStorageRepository) GetByHash(ctx context.Context, fileHash string) (*models.FileStorage, error) {
	var fs models.FileStorage
	if err := r.db.WithContext(ctx).Where("file_hash = ?", fileHash).First(&fs).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get file storage: %w", err)
	}
	return &fs, nil
}

// IncrementReference 增加引用计数
func (r *fileStorageRepository) IncrementReference(ctx context.Context, fileHash string) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).Model(&models.FileStorage{}).
		Where("file_hash = ?", fileHash).
		Updates(map[string]interface{}{
			"reference_count":    gorm.Expr("reference_count + 1"),
			"last_referenced_at": now,
			"updated_at":         now,
		}).Error; err != nil {
		return fmt.Errorf("failed to increment reference: %w", err)
	}
	return nil
}

// DecrementReference 减少引用计数
func (r *fileStorageRepository) DecrementReference(ctx context.Context, fileHash string) error {
	now := time.Now()
	// 确保引用计数不会变成负数
	if err := r.db.WithContext(ctx).Model(&models.FileStorage{}).
		Where("file_hash = ? AND reference_count > 0", fileHash).
		Updates(map[string]interface{}{
			"reference_count": gorm.Expr("reference_count - 1"),
			"updated_at":      now,
		}).Error; err != nil {
		return fmt.Errorf("failed to decrement reference: %w", err)
	}
	return nil
}

// BatchDecrementReferences 批量减少引用计数
func (r *fileStorageRepository) BatchDecrementReferences(ctx context.Context, fileHashes []string) error {
	if len(fileHashes) == 0 {
		return nil
	}

	now := time.Now()
	// 批量更新：确保引用计数不会变成负数
	if err := r.db.WithContext(ctx).Model(&models.FileStorage{}).
		Where("file_hash IN ? AND reference_count > 0", fileHashes).
		Updates(map[string]interface{}{
			"reference_count": gorm.Expr("reference_count - 1"),
			"updated_at":      now,
		}).Error; err != nil {
		return fmt.Errorf("failed to batch decrement references: %w", err)
	}
	return nil
}

// DeleteIfNoReferences 如果引用计数为0则删除记录
func (r *fileStorageRepository) DeleteIfNoReferences(ctx context.Context, fileHash string) (bool, error) {
	result := r.db.WithContext(ctx).
		Where("file_hash = ? AND reference_count = 0", fileHash).
		Delete(&models.FileStorage{})

	if result.Error != nil {
		return false, fmt.Errorf("failed to delete file storage: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

// GetOrphaned 获取孤立文件（引用计数为0且超过保留期的文件）
func (r *fileStorageRepository) GetOrphaned(ctx context.Context, retentionDays int) ([]*models.FileStorage, error) {
	var orphaned []*models.FileStorage
	retentionTime := time.Now().AddDate(0, 0, -retentionDays)

	if err := r.db.WithContext(ctx).
		Where("reference_count = 0 AND updated_at < ?", retentionTime).
		Find(&orphaned).Error; err != nil {
		return nil, fmt.Errorf("failed to get orphaned files: %w", err)
	}

	return orphaned, nil
}
