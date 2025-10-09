package providers

import (
	"context"
	"fmt"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/llm"
)

// GrokProvider xAI Grok 服务商适配器
type GrokProvider struct {
	apiKey  string
	baseURL string
}

// NewGrokProvider 创建 Grok 提供者
func NewGrokProvider(apiKey, baseURL string) *GrokProvider {
	if baseURL == "" {
		baseURL = "https://api.x.ai/v1"
	}

	return &GrokProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

// Name 返回服务商名称
func (p *GrokProvider) Name() string {
	return "grok"
}

// ValidateConfig 验证配置
func (p *GrokProvider) ValidateConfig() error {
	if p.apiKey == "" {
		return fmt.Errorf("grok api key is required")
	}
	return nil
}

// SupportedModels 返回支持的模型
func (p *GrokProvider) SupportedModels() []string {
	return []string{
		"grok-2-1212",
		"grok-2-vision-1212",
		"grok-beta",
	}
}

// SupportsMultimodal 是否支持多模态
func (p *GrokProvider) SupportsMultimodal() bool {
	return true
}

// ChatStream 流式聊天
func (p *GrokProvider) ChatStream(ctx context.Context, req *llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	// TODO: 实现 Grok API 调用
	// Grok API 与 OpenAI 兼容，可以复用 OpenAI 的实现逻辑

	eventChan := make(chan llm.StreamEvent, 100)

	go func() {
		defer close(eventChan)

		eventChan <- llm.StreamEvent{
			Type:  llm.EventError,
			Error: fmt.Errorf("grok provider not implemented yet"),
		}
	}()

	return eventChan, nil
}
