package factory

import (
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/ai/provider/types"
)

// Option 配置选项函数
type Option func(*types.Config)

// WithModel 返回设置模型的 Option
func WithModel(model string) Option {
	return func(c *types.Config) {
		c.Model = model
	}
}

// WithTimeout 返回设置超时的 Option
func WithTimeout(timeout time.Duration) Option {
	return func(c *types.Config) {
		c.Timeout = timeout
	}
}

// WithHeader 返回添加单个 Header 的 Option
func WithHeader(key, value string) Option {
	return func(c *types.Config) {
		if c.Headers == nil {
			c.Headers = make(map[string]string)
		}
		c.Headers[key] = value
	}
}

// WithHeaders 返回批量设置 Headers 的 Option
func WithHeaders(headers map[string]string) Option {
	return func(c *types.Config) {
		if c.Headers == nil {
			c.Headers = make(map[string]string)
		}
		for key, value := range headers {
			c.Headers[key] = value
		}
	}
}

// OpenAI 快速创建 OpenAI 配置
func OpenAI(apiKey string, opts ...Option) *types.Config {
	config := &types.Config{
		APIKey:  apiKey,
		BaseURL: "https://api.openai.com/v1",
		Timeout: 30 * time.Second,
		Headers: make(map[string]string),
	}

	// 应用选项
	for _, opt := range opts {
		opt(config)
	}

	return config
}

// Anthropic 快速创建 Anthropic 配置
func Anthropic(apiKey, baseURL string, opts ...Option) *types.Config {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	config := &types.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Timeout: 30 * time.Second,
		Headers: make(map[string]string),
	}

	// 应用选项
	for _, opt := range opts {
		opt(config)
	}

	return config
}

// SiliconFlow 快速创建 SiliconFlow 配置（基于 OpenAI 协议）
func SiliconFlow(apiKey string, opts ...Option) *types.Config {
	config := &types.Config{
		APIKey:  apiKey,
		BaseURL: "https://api.siliconflow.cn/v1",
		Timeout: 30 * time.Second,
		Model:   "Qwen/Qwen2.5-7B-Instruct", // 默认模型
		Headers: make(map[string]string),
	}

	// 应用选项
	for _, opt := range opts {
		opt(config)
	}

	return config
}

// OpenAICompatible 快速创建 OpenAI 兼容配置
func OpenAICompatible(apiKey, baseURL string, opts ...Option) *types.Config {
	config := &types.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Timeout: 30 * time.Second,
		Headers: make(map[string]string),
	}

	// 应用选项
	for _, opt := range opts {
		opt(config)
	}

	return config
}
