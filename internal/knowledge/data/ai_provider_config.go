package data

import (
	"context"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
)

// AIProviderPO AI服务商数据库模型
type AIProviderPO struct {
	ID           string    `gorm:"type:uuid;primarykey;default:gen_random_uuid()"`
	ProviderType string    `gorm:"size:50;not null;unique"`
	ProviderName string    `gorm:"size:100;not null"`
	APIBaseURL   string    `gorm:"column:api_base_url;size:255"`
	APIKey       string    `gorm:"column:api_key;type:text"`
	IsEnabled    bool      `gorm:"default:true"`
	CreatedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (AIProviderPO) TableName() string {
	return "ai_providers"
}

// AIProviderRepo AI服务商仓储实现
type AIProviderRepo struct {
	db *database.DB
}

// NewAIProviderRepo 创建AI服务商仓储
func NewAIProviderRepo(db *database.DB) biz.AIProviderRepo {
	return &AIProviderRepo{db: db}
}

// ListAll 获取所有AI服务商列表（包括禁用的，前端根据 is_enabled 字段显示开关状态）
func (r *AIProviderRepo) ListAll(ctx context.Context) ([]*biz.AIProvider, error) {
	var pos []AIProviderPO
	err := r.db.WithContext(ctx).GetDB().
		Order("id ASC").
		Find(&pos).Error

	if err != nil {
		return nil, err
	}

	providers := make([]*biz.AIProvider, len(pos))
	for i, po := range pos {
		providers[i] = r.toProvider(&po)
	}

	return providers, nil
}

// GetByID 根据ID获取AI服务商（不过滤启用状态，用于管理操作）
func (r *AIProviderRepo) GetByID(ctx context.Context, id string) (*biz.AIProvider, error) {
	var po AIProviderPO
	err := r.db.WithContext(ctx).GetDB().
		Where("id = ?", id).
		First(&po).Error

	if err != nil {
		if database.IsRecordNotFoundError(err) {
			return nil, biz.ErrAIProviderNotFound
		}
		return nil, err
	}

	return r.toProvider(&po), nil
}

// GetByType 根据类型获取AI服务商
func (r *AIProviderRepo) GetByType(ctx context.Context, providerType string) (*biz.AIProvider, error) {
	var po AIProviderPO
	err := r.db.WithContext(ctx).GetDB().
		Where("provider_type = ? AND is_enabled = true", providerType).
		First(&po).Error

	if err != nil {
		if database.IsRecordNotFoundError(err) {
			return nil, biz.ErrAIProviderNotFound
		}
		return nil, err
	}

	return r.toProvider(&po), nil
}

// UpdateStatus 更新服务商启用状态
func (r *AIProviderRepo) UpdateStatus(ctx context.Context, id string, isEnabled bool) error {
	return r.db.WithContext(ctx).GetDB().
		Model(&AIProviderPO{}).
		Where("id = ?", id).
		Update("is_enabled", isEnabled).
		Error
}

// UpdateConfig 更新服务商配置（API Key 和 API 地址）
func (r *AIProviderRepo) UpdateConfig(ctx context.Context, id string, apiKey, apiBaseURL *string) error {
	updates := make(map[string]interface{})

	if apiKey != nil {
		updates["api_key"] = *apiKey
	}

	if apiBaseURL != nil {
		updates["api_base_url"] = *apiBaseURL
	}

	// 如果没有要更新的字段，直接返回
	if len(updates) == 0 {
		return nil
	}

	// 添加更新时间
	updates["updated_at"] = time.Now()

	return r.db.WithContext(ctx).GetDB().
		Model(&AIProviderPO{}).
		Where("id = ?", id).
		Updates(updates).
		Error
}

// toProvider 转换 PO 到业务对象
func (r *AIProviderRepo) toProvider(po *AIProviderPO) *biz.AIProvider {
	return &biz.AIProvider{
		ID:           po.ID,
		ProviderType: po.ProviderType,
		ProviderName: po.ProviderName,
		APIBaseURL:   po.APIBaseURL,
		APIKey:       po.APIKey,
		IsEnabled:    po.IsEnabled,
		CreatedAt:    po.CreatedAt,
		UpdatedAt:    po.UpdatedAt,
	}
}
