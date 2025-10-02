package embedding

import (
	"fmt"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/redis"
)

// Factory Embedder 工厂
type Factory struct {
	logger *logger.Logger
	cache  *redis.Client
}

// NewFactory 创建 Embedder 工厂
func NewFactory(lgr *logger.Logger, cache *redis.Client) *Factory {
	if lgr == nil {
		lgr = logger.L()
	}
	return &Factory{
		logger: lgr,
		cache:  cache,
	}
}

// CreateEmbedderConfig 创建 Embedder 配置
type CreateEmbedderConfig struct {
	Provider   kbtypes.EmbeddingProvider
	Model      string
	Dimension  int
	APIKey     string
	BaseURL    string
	EnableCache bool
}

// CreateEmbedder 创建 Embedder
func (f *Factory) CreateEmbedder(cfg *CreateEmbedderConfig) (Embedder, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	var embedder Embedder
	var err error

	switch cfg.Provider {
	case kbtypes.EmbeddingProviderOpenAI:
		embedder, err = NewOpenAIEmbedder(&OpenAIEmbedderConfig{
			APIKey:    cfg.APIKey,
			BaseURL:   cfg.BaseURL,
			Model:     cfg.Model,
			Dimension: cfg.Dimension,
		}, f.logger)

	case kbtypes.EmbeddingProviderAnthropic:
		// TODO: 实现 Anthropic Embedder（如果支持）
		return nil, fmt.Errorf("anthropic embedder not implemented yet")

	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// 如果启用缓存，包装为缓存 Embedder
	if cfg.EnableCache && f.cache != nil {
		embedder = NewCacheEmbedder(embedder, f.cache, nil, f.logger)
	}

	return embedder, nil
}
