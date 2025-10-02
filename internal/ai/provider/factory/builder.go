package factory

import (
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/ai/provider/types"
)

// ConfigBuilder 配置构建器（Builder 模式）
type ConfigBuilder struct {
	config *types.Config
}

// NewConfig 创建配置构建器
func NewConfig() *ConfigBuilder {
	return &ConfigBuilder{
		config: &types.Config{
			Timeout: 30 * time.Second, // 默认超时
			Headers: make(map[string]string),
		},
	}
}

// WithAPIKey 设置 API Key
func (b *ConfigBuilder) WithAPIKey(apiKey string) *ConfigBuilder {
	b.config.APIKey = apiKey
	return b
}

// WithBaseURL 设置 Base URL
func (b *ConfigBuilder) WithBaseURL(baseURL string) *ConfigBuilder {
	b.config.BaseURL = baseURL
	return b
}

// WithModel 设置默认模型
func (b *ConfigBuilder) WithModel(model string) *ConfigBuilder {
	b.config.Model = model
	return b
}

// WithTimeout 设置超时时间
func (b *ConfigBuilder) WithTimeout(timeout time.Duration) *ConfigBuilder {
	b.config.Timeout = timeout
	return b
}

// WithHeader 添加单个 Header
func (b *ConfigBuilder) WithHeader(key, value string) *ConfigBuilder {
	if b.config.Headers == nil {
		b.config.Headers = make(map[string]string)
	}
	b.config.Headers[key] = value
	return b
}

// WithHeaders 批量设置 Headers
func (b *ConfigBuilder) WithHeaders(headers map[string]string) *ConfigBuilder {
	if b.config.Headers == nil {
		b.config.Headers = make(map[string]string)
	}
	for key, value := range headers {
		b.config.Headers[key] = value
	}
	return b
}

// Build 构建最终配置
func (b *ConfigBuilder) Build() *types.Config {
	return b.config
}
