package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ModelSyncLog 模型同步日志
type ModelSyncLog struct {
	ID                    string
	ProviderID            string
	SyncType              string // manual, scheduled
	NewModelsCount        int
	DeprecatedModelsCount int
	UpdatedModelsCount    int
	ErrorCount            int
	NewModels             []string
	DeprecatedModels      []string
	UpdatedModels         []string
	ErrorMessage          string
	SyncedBy              string
	SyncedAt              time.Time
}

// ModelSyncLogRepo 模型同步日志仓储接口
type ModelSyncLogRepo interface {
	Create(ctx context.Context, log *ModelSyncLog) error
	ListByProviderID(ctx context.Context, providerID string, limit int) ([]*ModelSyncLog, error)
	GetLatest(ctx context.Context, providerID string) (*ModelSyncLog, error)
}

// ModelSyncRequest 模型同步请求
type ModelSyncRequest struct {
	ProviderID string
	SyncedBy   string // 同步操作者
	SyncType   string // manual, scheduled
}

// ModelSyncResult 模型同步结果
type ModelSyncResult struct {
	NewModels        []*AIModel
	DeprecatedModels []*AIModel
	UpdatedModels    []*AIModel
	Errors           []error
}

// ModelSyncUseCase 模型同步用例
type ModelSyncUseCase struct {
	aiProviderRepo AIProviderRepo
	aiModelRepo    AIModelRepo
	syncLogRepo    ModelSyncLogRepo
}

// NewModelSyncUseCase 创建模型同步用例
func NewModelSyncUseCase(
	aiProviderRepo AIProviderRepo,
	aiModelRepo AIModelRepo,
	syncLogRepo ModelSyncLogRepo,
) *ModelSyncUseCase {
	return &ModelSyncUseCase{
		aiProviderRepo: aiProviderRepo,
		aiModelRepo:    aiModelRepo,
		syncLogRepo:    syncLogRepo,
	}
}

// SyncProviderModels 同步服务商的模型列表
func (uc *ModelSyncUseCase) SyncProviderModels(ctx context.Context, req *ModelSyncRequest) (*ModelSyncResult, error) {
	// 验证 Provider 存在
	provider, err := uc.aiProviderRepo.GetByID(ctx, req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	// 获取当前数据库中的模型列表
	currentModels, err := uc.aiModelRepo.ListByProviderID(ctx, req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to list current models: %w", err)
	}

	// 获取最新的模型列表（根据不同 Provider 调用不同的实现）
	latestModels, err := uc.fetchLatestModels(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest models: %w", err)
	}

	// 对比模型列表，找出新增、更新、弃用的模型
	result := uc.compareModels(currentModels, latestModels)

	// 应用变更到数据库
	if err := uc.applyChanges(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to apply changes: %w", err)
	}

	// 记录同步日志
	syncLog := &ModelSyncLog{
		ID:                    uuid.New().String(),
		ProviderID:            req.ProviderID,
		SyncType:              req.SyncType,
		NewModelsCount:        len(result.NewModels),
		DeprecatedModelsCount: len(result.DeprecatedModels),
		UpdatedModelsCount:    len(result.UpdatedModels),
		ErrorCount:            len(result.Errors),
		NewModels:             extractModelNames(result.NewModels),
		DeprecatedModels:      extractModelNames(result.DeprecatedModels),
		UpdatedModels:         extractModelNames(result.UpdatedModels),
		SyncedBy:              req.SyncedBy,
		SyncedAt:              time.Now(),
	}

	if len(result.Errors) > 0 {
		syncLog.ErrorMessage = fmt.Sprintf("%v", result.Errors)
	}

	if err := uc.syncLogRepo.Create(ctx, syncLog); err != nil {
		return nil, fmt.Errorf("failed to create sync log: %w", err)
	}

	// 临时返回当前模型列表，表明系统正常工作
	_ = currentModels
	_ = provider

	return result, nil
}

// GetSyncHistory 获取同步历史
func (uc *ModelSyncUseCase) GetSyncHistory(ctx context.Context, providerID string, limit int) ([]*ModelSyncLog, error) {
	return uc.syncLogRepo.ListByProviderID(ctx, providerID, limit)
}

// GetLatestSync 获取最近一次同步记录
func (uc *ModelSyncUseCase) GetLatestSync(ctx context.Context, providerID string) (*ModelSyncLog, error) {
	return uc.syncLogRepo.GetLatest(ctx, providerID)
}

// extractModelNames 提取模型名称列表
func extractModelNames(models []*AIModel) []string {
	names := make([]string, len(models))
	for i, m := range models {
		names[i] = m.ModelName
	}
	return names
}
