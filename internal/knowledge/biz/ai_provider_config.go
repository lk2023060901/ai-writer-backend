package biz

import (
	"context"
	"time"
)

// AIProvider AI服务商（系统预设，只读）
type AIProvider struct {
	ID           string
	ProviderType string // openai, anthropic, siliconflow 等
	ProviderName string
	APIBaseURL   string
	APIKey       string // API 密钥
	IsEnabled    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AIProviderRepo AI服务商仓储接口
type AIProviderRepo interface {
	ListAll(ctx context.Context) ([]*AIProvider, error)
	GetByID(ctx context.Context, id string) (*AIProvider, error)
	GetByType(ctx context.Context, providerType string) (*AIProvider, error)
	UpdateStatus(ctx context.Context, id string, isEnabled bool) error
	UpdateConfig(ctx context.Context, id string, apiKey, apiBaseURL *string) error
}

// AIProviderUseCase AI服务商用例（只读）
type AIProviderUseCase struct {
	repo AIProviderRepo
}

// NewAIProviderUseCase 创建AI服务商用例
func NewAIProviderUseCase(repo AIProviderRepo) *AIProviderUseCase {
	return &AIProviderUseCase{repo: repo}
}

// ListAIProviders 获取所有AI服务商列表
func (uc *AIProviderUseCase) ListAIProviders(ctx context.Context) ([]*AIProvider, error) {
	return uc.repo.ListAll(ctx)
}

// GetAIProviderByID 根据ID获取AI服务商
func (uc *AIProviderUseCase) GetAIProviderByID(ctx context.Context, id string) (*AIProvider, error) {
	return uc.repo.GetByID(ctx, id)
}

// GetAIProviderByType 根据类型获取AI服务商
func (uc *AIProviderUseCase) GetAIProviderByType(ctx context.Context, providerType string) (*AIProvider, error) {
	return uc.repo.GetByType(ctx, providerType)
}

// UpdateProviderStatus 更新服务商启用状态
func (uc *AIProviderUseCase) UpdateProviderStatus(ctx context.Context, id string, isEnabled bool) error {
	return uc.repo.UpdateStatus(ctx, id, isEnabled)
}

// UpdateProviderConfig 更新服务商配置（API Key 和 API 地址）
func (uc *AIProviderUseCase) UpdateProviderConfig(ctx context.Context, id string, apiKey, apiBaseURL *string) error {
	return uc.repo.UpdateConfig(ctx, id, apiKey, apiBaseURL)
}
