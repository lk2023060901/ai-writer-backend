package biz

import (
	"context"
	"time"
)

// AIModel AI模型
type AIModel struct {
	ID                 string
	ProviderID         string
	ModelName          string
	DisplayName        string
	MaxTokens          *int
	IsEnabled          bool
	LastVerifiedAt     *time.Time
	VerificationStatus string // available, deprecated, error, unknown

	// 能力字段（JSONB 存储）
	Capabilities            []string // 能力类型数组: ["chat", "embedding", "rerank"]
	SupportsStream          bool
	SupportsVision          bool
	SupportsFunctionCalling bool
	SupportsReasoning       bool
	SupportsWebSearch       bool
	EmbeddingDimensions     *int // Embedding 模型的向量维度

	CreatedAt time.Time
	UpdatedAt time.Time
}

// AIModelRepo AI模型仓储接口
type AIModelRepo interface {
	GetByID(ctx context.Context, id string) (*AIModel, error)
	ListByProviderID(ctx context.Context, providerID string) ([]*AIModel, error)
	ListByCapabilityType(ctx context.Context, capabilityType string) ([]*AIModel, error) // 根据能力类型查询（在 JSONB 数组中查找）
	ListAll(ctx context.Context) ([]*AIModel, error)
	Create(ctx context.Context, model *AIModel) error
	Update(ctx context.Context, model *AIModel) error
	Delete(ctx context.Context, id string) error
}

// AIModelUseCase AI模型用例
type AIModelUseCase struct {
	repo AIModelRepo
}

// NewAIModelUseCase 创建AI模型用例
func NewAIModelUseCase(repo AIModelRepo) *AIModelUseCase {
	return &AIModelUseCase{repo: repo}
}

// GetAIModelByID 根据ID获取AI模型
func (uc *AIModelUseCase) GetAIModelByID(ctx context.Context, id string) (*AIModel, error) {
	return uc.repo.GetByID(ctx, id)
}

// ListAIModelsByProviderID 根据服务商ID获取模型列表
func (uc *AIModelUseCase) ListAIModelsByProviderID(ctx context.Context, providerID string) ([]*AIModel, error) {
	return uc.repo.ListByProviderID(ctx, providerID)
}

// ListAIModelsByCapabilityType 根据能力类型获取模型列表
func (uc *AIModelUseCase) ListAIModelsByCapabilityType(ctx context.Context, capabilityType string) ([]*AIModel, error) {
	return uc.repo.ListByCapabilityType(ctx, capabilityType)
}

// ListAll 获取所有启用的AI模型
func (uc *AIModelUseCase) ListAll(ctx context.Context) ([]*AIModel, error) {
	return uc.repo.ListAll(ctx)
}

// CreateAIModel 创建模型
func (uc *AIModelUseCase) CreateAIModel(ctx context.Context, model *AIModel) error {
	return uc.repo.Create(ctx, model)
}

// UpdateAIModel 更新模型
func (uc *AIModelUseCase) UpdateAIModel(ctx context.Context, model *AIModel) error {
	return uc.repo.Update(ctx, model)
}

// DeleteAIModel 删除模型
func (uc *AIModelUseCase) DeleteAIModel(ctx context.Context, id string) error {
	return uc.repo.Delete(ctx, id)
}

// ListAllAIModels 获取所有AI模型列表
func (uc *AIModelUseCase) ListAllAIModels(ctx context.Context) ([]*AIModel, error) {
	return uc.repo.ListAll(ctx)
}
