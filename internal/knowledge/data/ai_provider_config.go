package data

import (
	"context"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
)

// AIProviderConfigPO AI服务商配置数据库模型
type AIProviderConfigPO struct {
	ID                  string    `gorm:"type:uuid;primarykey"`
	OwnerID             string    `gorm:"type:uuid;not null;index:idx_ai_provider_configs_owner_id"`
	ProviderType        string    `gorm:"size:50;not null"`
	ProviderName        string    `gorm:"size:255;not null"`
	APIKey              string    `gorm:"type:text;not null"`
	APIBaseURL          string    `gorm:"type:text"`
	EmbeddingModel      string    `gorm:"size:255;not null"`
	EmbeddingDimensions int       `gorm:"not null"`
	IsEnabled           bool      `gorm:"not null;default:true"`
	CreatedAt           time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt           time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (AIProviderConfigPO) TableName() string {
	return "ai_provider_configs"
}

// AIProviderConfigRepo AI服务商配置仓储实现
type AIProviderConfigRepo struct {
	db *database.DB
}

// NewAIProviderConfigRepo 创建AI服务商配置仓储
func NewAIProviderConfigRepo(db *database.DB) biz.AIProviderConfigRepo {
	return &AIProviderConfigRepo{db: db}
}

// Create 创建AI服务商配置
func (r *AIProviderConfigRepo) Create(ctx context.Context, config *biz.AIProviderConfig) error {
	po := &AIProviderConfigPO{
		ID:                  config.ID,
		OwnerID:             config.OwnerID,
		ProviderType:        config.ProviderType,
		ProviderName:        config.ProviderName,
		APIKey:              config.APIKey,
		APIBaseURL:          config.APIBaseURL,
		EmbeddingModel:      config.EmbeddingModel,
		EmbeddingDimensions: config.EmbeddingDimensions,
		IsEnabled:           config.IsEnabled,
		CreatedAt:           config.CreatedAt,
		UpdatedAt:           config.UpdatedAt,
	}

	return r.db.WithContext(ctx).GetDB().Create(po).Error
}

// GetByID 根据ID获取AI服务商配置（需要验证权限：官方配置或用户自己的配置）
func (r *AIProviderConfigRepo) GetByID(ctx context.Context, id string, userID string) (*biz.AIProviderConfig, error) {
	var po AIProviderConfigPO
	err := r.db.WithContext(ctx).GetDB().
		Where("id = ? AND (owner_id = ? OR owner_id = ?)", id, biz.SystemOwnerID, userID).
		First(&po).Error

	if err != nil {
		if database.IsRecordNotFoundError(err) {
			return nil, biz.ErrAIProviderConfigNotFound
		}
		return nil, err
	}

	return r.toConfig(&po), nil
}

// GetFirstAvailable 获取第一个可用的AI服务商配置（优先用户自己的，否则官方的）
func (r *AIProviderConfigRepo) GetFirstAvailable(ctx context.Context, userID string) (*biz.AIProviderConfig, error) {
	var po AIProviderConfigPO

	// 优先查找用户自己的配置
	err := r.db.WithContext(ctx).GetDB().
		Where("owner_id = ? AND is_enabled = true", userID).
		Order("created_at DESC").
		First(&po).Error

	if err == nil {
		return r.toConfig(&po), nil
	}

	if !database.IsRecordNotFoundError(err) {
		return nil, err
	}

	// 如果用户没有配置，使用官方配置
	err = r.db.WithContext(ctx).GetDB().
		Where("owner_id = ? AND is_enabled = true", biz.SystemOwnerID).
		Order("created_at DESC").
		First(&po).Error

	if err != nil {
		if database.IsRecordNotFoundError(err) {
			return nil, biz.ErrNoDefaultAIConfig
		}
		return nil, err
	}

	return r.toConfig(&po), nil
}

// List 获取AI服务商配置列表（官方配置 + 用户自己的配置）
func (r *AIProviderConfigRepo) List(ctx context.Context, userID string) ([]*biz.AIProviderConfig, error) {
	var pos []AIProviderConfigPO
	err := r.db.WithContext(ctx).GetDB().
		Where("(owner_id = ? OR owner_id = ?) AND is_enabled = true", biz.SystemOwnerID, userID).
		Order("CASE WHEN owner_id = '" + biz.SystemOwnerID + "' THEN 0 ELSE 1 END, created_at DESC").
		Find(&pos).Error

	if err != nil {
		return nil, err
	}

	configs := make([]*biz.AIProviderConfig, len(pos))
	for i, po := range pos {
		configs[i] = r.toConfig(&po)
	}

	return configs, nil
}

// Update 更新AI服务商配置
func (r *AIProviderConfigRepo) Update(ctx context.Context, config *biz.AIProviderConfig) error {
	updates := map[string]interface{}{
		"provider_name":        config.ProviderName,
		"api_key":              config.APIKey,
		"api_base_url":         config.APIBaseURL,
		"embedding_model":      config.EmbeddingModel,
		"embedding_dimensions": config.EmbeddingDimensions,
		"is_enabled":           config.IsEnabled,
		"updated_at":           config.UpdatedAt,
	}

	result := r.db.WithContext(ctx).GetDB().
		Model(&AIProviderConfigPO{}).
		Where("id = ? AND owner_id = ?", config.ID, config.OwnerID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return biz.ErrAIProviderConfigNotFound
	}

	return nil
}

// Delete 删除AI服务商配置
func (r *AIProviderConfigRepo) Delete(ctx context.Context, id string, ownerID string) error {
	result := r.db.WithContext(ctx).GetDB().
		Where("id = ? AND owner_id = ?", id, ownerID).
		Delete(&AIProviderConfigPO{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return biz.ErrAIProviderConfigNotFound
	}

	return nil
}

// CountByKnowledgeBase 统计使用该配置的知识库数量
func (r *AIProviderConfigRepo) CountByKnowledgeBase(ctx context.Context, configID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).GetDB().
		Table("knowledge_bases").
		Where("ai_provider_config_id = ?", configID).
		Count(&count).Error

	return count, err
}

// toConfig 转换 PO 到业务对象
func (r *AIProviderConfigRepo) toConfig(po *AIProviderConfigPO) *biz.AIProviderConfig {
	return &biz.AIProviderConfig{
		ID:                  po.ID,
		OwnerID:             po.OwnerID,
		ProviderType:        po.ProviderType,
		ProviderName:        po.ProviderName,
		APIKey:              po.APIKey,
		APIBaseURL:          po.APIBaseURL,
		EmbeddingModel:      po.EmbeddingModel,
		EmbeddingDimensions: po.EmbeddingDimensions,
		IsEnabled:           po.IsEnabled,
		CreatedAt:           po.CreatedAt,
		UpdatedAt:           po.UpdatedAt,
	}
}
