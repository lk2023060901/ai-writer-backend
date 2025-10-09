package biz

import (
	"context"
	"time"
)

// DocumentProvider 文档处理服务商（系统预设，只读）
type DocumentProvider struct {
	ID           string
	ProviderType string
	ProviderName string
	APIBaseURL   string
	IsEnabled    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DocumentProviderRepo 文档处理服务商仓储接口（只读）
type DocumentProviderRepo interface {
	ListAll(ctx context.Context) ([]*DocumentProvider, error)
	GetByType(ctx context.Context, providerType string) (*DocumentProvider, error)
}

// DocumentProviderUseCase 文档处理服务商用例（只读）
type DocumentProviderUseCase struct {
	repo DocumentProviderRepo
}

// NewDocumentProviderUseCase 创建文档处理服务商用例
func NewDocumentProviderUseCase(repo DocumentProviderRepo) *DocumentProviderUseCase {
	return &DocumentProviderUseCase{repo: repo}
}

// ListDocumentProviders 获取所有文档处理服务商列表
func (uc *DocumentProviderUseCase) ListDocumentProviders(ctx context.Context) ([]*DocumentProvider, error) {
	return uc.repo.ListAll(ctx)
}

// GetDocumentProviderByType 根据类型获取文档处理服务商
func (uc *DocumentProviderUseCase) GetDocumentProviderByType(ctx context.Context, providerType string) (*DocumentProvider, error) {
	return uc.repo.GetByType(ctx, providerType)
}
