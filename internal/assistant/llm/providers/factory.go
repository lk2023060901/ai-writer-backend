package providers

import (
	"context"
	"fmt"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/llm"
	knowledgebiz "github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"go.uber.org/zap"
)

// DatabaseProviderFactory 基于数据库的服务商工厂
type DatabaseProviderFactory struct {
	aiProviderUseCase *knowledgebiz.AIProviderUseCase
	logger            *zap.Logger
}

// NewDatabaseProviderFactory 创建服务商工厂
func NewDatabaseProviderFactory(aiProviderUseCase *knowledgebiz.AIProviderUseCase, logger *zap.Logger) *DatabaseProviderFactory {
	return &DatabaseProviderFactory{
		aiProviderUseCase: aiProviderUseCase,
		logger:            logger,
	}
}

// CreateProvider 创建服务商实例（实现 ProviderFactory 接口）
func (f *DatabaseProviderFactory) CreateProvider(config llm.ProviderConfig) (llm.Provider, error) {
	ctx := context.Background()

	f.logger.Info("CreateProvider called",
		zap.String("provider_id", config.Provider))

	// 从数据库获取服务商配置（通过 UUID）
	providerConfig, err := f.aiProviderUseCase.GetAIProviderByID(ctx, config.Provider)
	if err != nil {
		f.logger.Error("Failed to get provider from database",
			zap.String("provider_id", config.Provider),
			zap.Error(err))
		return nil, fmt.Errorf("get provider config: %w", err)
	}

	f.logger.Info("Provider config retrieved from database",
		zap.String("provider_id", config.Provider),
		zap.String("provider_type", providerConfig.ProviderType),
		zap.String("provider_name", providerConfig.ProviderName),
		zap.Bool("is_enabled", providerConfig.IsEnabled))

	if !providerConfig.IsEnabled {
		f.logger.Warn("Provider is disabled",
			zap.String("provider_id", config.Provider),
			zap.String("provider_type", providerConfig.ProviderType))
		return nil, fmt.Errorf("provider %s is disabled", providerConfig.ProviderType)
	}

	// 使用配置中的 API Key，如果没有则使用数据库中的
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = providerConfig.APIKey
	}

	if apiKey == "" {
		return nil, fmt.Errorf("provider %s has no API key configured", providerConfig.ProviderType)
	}

	// 使用配置中的 BaseURL，如果没有则使用数据库中的
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = providerConfig.APIBaseURL
	}

	// 根据类型创建对应的 Provider（使用数据库中的 ProviderType）
	switch providerConfig.ProviderType {
	case "openai":
		return NewOpenAIProvider(apiKey, baseURL), nil

	case "anthropic":
		return NewAnthropicProvider(apiKey, baseURL), nil

	case "gemini":
		return NewGeminiProvider(apiKey, baseURL), nil

	case "siliconflow":
		// SiliconFlow 兼容 OpenAI API
		return NewOpenAIProvider(apiKey, baseURL), nil

	case "grok":
		return NewGrokProvider(apiKey, baseURL), nil

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerConfig.ProviderType)
	}
}

// GetAvailableProviders 获取所有可用的服务商（有 API Key 且已启用）
func (f *DatabaseProviderFactory) GetAvailableProviders(ctx context.Context) ([]string, error) {
	allProviders, err := f.aiProviderUseCase.ListAIProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	availableTypes := make([]string, 0)
	for _, p := range allProviders {
		if p.IsEnabled && p.APIKey != "" {
			availableTypes = append(availableTypes, p.ProviderType)
		}
	}

	return availableTypes, nil
}
