package biz

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AIProviderConfig AI服务商配置业务对象
type AIProviderConfig struct {
	ID                  string
	OwnerID             string // SystemOwnerID = 官方
	ProviderType        string // openai, anthropic, siliconflow 等
	ProviderName        string
	APIKey              string
	APIBaseURL          string
	EmbeddingModel      string
	EmbeddingDimensions int
	IsEnabled           bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// IsOfficial 是否为官方配置
func (c *AIProviderConfig) IsOfficial() bool {
	return c.OwnerID == SystemOwnerID
}

// AIProviderConfigRepo AI服务商配置仓储接口
type AIProviderConfigRepo interface {
	Create(ctx context.Context, config *AIProviderConfig) error
	GetByID(ctx context.Context, id string, userID string) (*AIProviderConfig, error)
	GetFirstAvailable(ctx context.Context, userID string) (*AIProviderConfig, error)
	List(ctx context.Context, userID string) ([]*AIProviderConfig, error)
	Update(ctx context.Context, config *AIProviderConfig) error
	Delete(ctx context.Context, id string, ownerID string) error
	CountByKnowledgeBase(ctx context.Context, configID string) (int64, error)
}

// CreateAIProviderConfigRequest 创建AI服务商配置请求
type CreateAIProviderConfigRequest struct {
	ProviderType        string
	ProviderName        string
	APIKey              string
	APIBaseURL          string
	EmbeddingModel      string
	EmbeddingDimensions int
}

// UpdateAIProviderConfigRequest 更新AI服务商配置请求
type UpdateAIProviderConfigRequest struct {
	ProviderName        *string
	APIKey              *string
	APIBaseURL          *string
	EmbeddingModel      *string
	EmbeddingDimensions *int
}

// AIProviderConfigUseCase AI服务商配置用例
type AIProviderConfigUseCase struct {
	repo AIProviderConfigRepo
}

// NewAIProviderConfigUseCase 创建AI服务商配置用例
func NewAIProviderConfigUseCase(repo AIProviderConfigRepo) *AIProviderConfigUseCase {
	return &AIProviderConfigUseCase{repo: repo}
}

// CreateAIProviderConfig 创建AI服务商配置
func (uc *AIProviderConfigUseCase) CreateAIProviderConfig(
	ctx context.Context,
	userID string,
	req *CreateAIProviderConfigRequest,
) (*AIProviderConfig, error) {
	// 验证必填字段
	if req.ProviderType == "" {
		return nil, ErrAIProviderConfigInvalidProvider
	}
	if req.ProviderName == "" {
		return nil, ErrAIProviderConfigNameRequired
	}
	if req.APIKey == "" {
		return nil, ErrAIProviderConfigAPIKeyRequired
	}
	if req.EmbeddingModel == "" || req.EmbeddingDimensions <= 0 {
		return nil, ErrAIProviderConfigInvalidProvider
	}

	now := time.Now()
	config := &AIProviderConfig{
		ID:                  uuid.New().String(),
		OwnerID:             userID,
		ProviderType:        req.ProviderType,
		ProviderName:        req.ProviderName,
		APIKey:              req.APIKey,
		APIBaseURL:          req.APIBaseURL,
		EmbeddingModel:      req.EmbeddingModel,
		EmbeddingDimensions: req.EmbeddingDimensions,
		IsEnabled:           true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := uc.repo.Create(ctx, config); err != nil {
		return nil, err
	}

	return config, nil
}

// GetAIProviderConfig 获取AI服务商配置
func (uc *AIProviderConfigUseCase) GetAIProviderConfig(
	ctx context.Context,
	id string,
	userID string,
) (*AIProviderConfig, error) {
	return uc.repo.GetByID(ctx, id, userID)
}

// ListAIProviderConfigs 获取AI服务商配置列表
func (uc *AIProviderConfigUseCase) ListAIProviderConfigs(
	ctx context.Context,
	userID string,
) ([]*AIProviderConfig, error) {
	return uc.repo.List(ctx, userID)
}

// UpdateAIProviderConfig 更新AI服务商配置
func (uc *AIProviderConfigUseCase) UpdateAIProviderConfig(
	ctx context.Context,
	id string,
	userID string,
	req *UpdateAIProviderConfigRequest,
) (*AIProviderConfig, error) {
	// 获取配置
	config, err := uc.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// 权限检查：不能编辑官方配置
	if config.IsOfficial() {
		return nil, ErrCannotEditOfficialResource
	}

	// 权限检查：只能编辑自己的配置
	if config.OwnerID != userID {
		return nil, ErrUnauthorized
	}

	// 更新字段
	if req.ProviderName != nil {
		config.ProviderName = *req.ProviderName
	}
	if req.APIKey != nil {
		config.APIKey = *req.APIKey
	}
	if req.APIBaseURL != nil {
		config.APIBaseURL = *req.APIBaseURL
	}
	if req.EmbeddingModel != nil {
		config.EmbeddingModel = *req.EmbeddingModel
	}
	if req.EmbeddingDimensions != nil {
		config.EmbeddingDimensions = *req.EmbeddingDimensions
	}
	config.UpdatedAt = time.Now()

	if err := uc.repo.Update(ctx, config); err != nil {
		return nil, err
	}

	return config, nil
}

// DeleteAIProviderConfig 删除AI服务商配置
func (uc *AIProviderConfigUseCase) DeleteAIProviderConfig(
	ctx context.Context,
	id string,
	userID string,
) error {
	// 获取配置
	config, err := uc.repo.GetByID(ctx, id, userID)
	if err != nil {
		return err
	}

	// 权限检查：不能删除官方配置
	if config.IsOfficial() {
		return ErrCannotDeleteOfficialResource
	}

	// 权限检查：只能删除自己的配置
	if config.OwnerID != userID {
		return ErrUnauthorized
	}

	// 检查是否有知识库在使用
	count, err := uc.repo.CountByKnowledgeBase(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrAIProviderConfigInUse
	}

	return uc.repo.Delete(ctx, id, userID)
}
