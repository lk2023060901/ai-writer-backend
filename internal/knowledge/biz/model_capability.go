package biz

import (
	"context"
	"time"
)

// ModelCapability 模型能力
type ModelCapability struct {
	ID                      string
	ModelID                 string
	CapabilityType          string // embedding, rerank, chat, vision, reasoning, function_calling, websearch
	EmbeddingDimensions     *int   // 仅 embedding 类型有值
	SupportsStream          bool
	SupportsVision          bool
	SupportsFunctionCalling bool
	SupportsReasoning       bool
	SupportsWebSearch       bool
	Metadata                map[string]interface{}
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// ModelCapabilityRepo 模型能力仓储接口
type ModelCapabilityRepo interface {
	// 基础 CRUD
	Create(ctx context.Context, capability *ModelCapability) error
	GetByID(ctx context.Context, id string) (*ModelCapability, error)
	Update(ctx context.Context, capability *ModelCapability) error
	Delete(ctx context.Context, id string) error
	
	// 查询接口
	ListByModelID(ctx context.Context, modelID string) ([]*ModelCapability, error)
	GetByModelIDAndType(ctx context.Context, modelID, capabilityType string) (*ModelCapability, error)
	ListByCapabilityType(ctx context.Context, capabilityType string) ([]*ModelCapability, error)
	
	// 批量操作
	BatchCreate(ctx context.Context, capabilities []*ModelCapability) error
	DeleteByModelID(ctx context.Context, modelID string) error
}

// ModelCapabilityUseCase 模型能力用例
type ModelCapabilityUseCase struct {
	repo ModelCapabilityRepo
}

// NewModelCapabilityUseCase 创建模型能力用例
func NewModelCapabilityUseCase(repo ModelCapabilityRepo) *ModelCapabilityUseCase {
	return &ModelCapabilityUseCase{repo: repo}
}

// AddCapability 添加模型能力
func (uc *ModelCapabilityUseCase) AddCapability(ctx context.Context, capability *ModelCapability) error {
	return uc.repo.Create(ctx, capability)
}

// GetCapabilitiesByModelID 获取模型的所有能力
func (uc *ModelCapabilityUseCase) GetCapabilitiesByModelID(ctx context.Context, modelID string) ([]*ModelCapability, error) {
	return uc.repo.ListByModelID(ctx, modelID)
}

// GetCapabilityByType 获取模型的特定能力
func (uc *ModelCapabilityUseCase) GetCapabilityByType(ctx context.Context, modelID, capabilityType string) (*ModelCapability, error) {
	return uc.repo.GetByModelIDAndType(ctx, modelID, capabilityType)
}

// ListModelsByCapability 获取具有特定能力的所有模型能力记录
func (uc *ModelCapabilityUseCase) ListModelsByCapability(ctx context.Context, capabilityType string) ([]*ModelCapability, error) {
	return uc.repo.ListByCapabilityType(ctx, capabilityType)
}

// BatchAddCapabilities 批量添加模型能力
func (uc *ModelCapabilityUseCase) BatchAddCapabilities(ctx context.Context, capabilities []*ModelCapability) error {
	return uc.repo.BatchCreate(ctx, capabilities)
}

// RemoveModelCapabilities 移除模型的所有能力
func (uc *ModelCapabilityUseCase) RemoveModelCapabilities(ctx context.Context, modelID string) error {
	return uc.repo.DeleteByModelID(ctx, modelID)
}

// 能力类型常量
const (
	CapabilityTypeEmbedding        = "embedding"
	CapabilityTypeRerank           = "rerank"
	CapabilityTypeChat             = "chat"
	CapabilityTypeVision           = "vision"
	CapabilityTypeReasoning        = "reasoning"
	CapabilityTypeFunctionCalling  = "function_calling"
	CapabilityTypeWebSearch        = "websearch"
)
