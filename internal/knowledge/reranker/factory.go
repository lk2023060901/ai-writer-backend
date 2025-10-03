package reranker

import (
	"fmt"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
)

// Factory Reranker 工厂
type Factory struct {
	logger *logger.Logger
}

// NewFactory 创建 Reranker 工厂
func NewFactory(lgr *logger.Logger) *Factory {
	if lgr == nil {
		lgr = logger.L()
	}
	return &Factory{
		logger: lgr,
	}
}

// CreateRerankerConfig 创建 Reranker 配置
type CreateRerankerConfig struct {
	Provider RerankProvider
	APIKey   string
	BaseURL  string
	Model    string
}

// CreateReranker 创建 Reranker
func (f *Factory) CreateReranker(cfg *CreateRerankerConfig) (Reranker, error) {
	if cfg == nil {
		return NewNoOpReranker(), nil
	}

	switch cfg.Provider {
	case RerankProviderJina:
		return NewJinaReranker(&JinaRerankerConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}, f.logger)

	case RerankProviderVoyage:
		return NewVoyageReranker(&VoyageRerankerConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}, f.logger)

	case RerankProviderCohere:
		// TODO: 实现 Cohere Reranker
		return nil, fmt.Errorf("cohere reranker not implemented yet")

	case RerankProviderSiliconFlow:
		return NewSiliconFlowReranker(&SiliconFlowRerankerConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}, f.logger)

	default:
		return NewNoOpReranker(), nil
	}
}
