package providers

import (
	"context"
	"fmt"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/llm"
)

// GeminiProvider Google Gemini 服务商适配器
type GeminiProvider struct {
	apiKey  string
	baseURL string
}

// NewGeminiProvider 创建 Gemini 提供者
func NewGeminiProvider(apiKey, baseURL string) *GeminiProvider {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	return &GeminiProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

// Name 返回服务商名称
func (p *GeminiProvider) Name() string {
	return "gemini"
}

// ValidateConfig 验证配置
func (p *GeminiProvider) ValidateConfig() error {
	if p.apiKey == "" {
		return fmt.Errorf("gemini api key is required")
	}
	return nil
}

// SupportedModels 返回支持的模型
func (p *GeminiProvider) SupportedModels() []string {
	return []string{
		"gemini-2.0-flash-exp",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-1.0-pro",
	}
}

// SupportsMultimodal 是否支持多模态
func (p *GeminiProvider) SupportsMultimodal() bool {
	return true
}

// ChatStream 流式聊天
func (p *GeminiProvider) ChatStream(ctx context.Context, req *llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	// TODO: 实现 Gemini API 调用
	// Gemini API 文档: https://ai.google.dev/api/rest/v1beta/models/streamGenerateContent

	eventChan := make(chan llm.StreamEvent, 100)

	go func() {
		defer close(eventChan)

		eventChan <- llm.StreamEvent{
			Type:  llm.EventError,
			Error: fmt.Errorf("gemini provider not implemented yet"),
		}
	}()

	return eventChan, nil
}
