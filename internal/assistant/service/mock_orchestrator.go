package service

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/llm"
	"github.com/lk2023060901/ai-writer-backend/internal/assistant/types"
)

// MockOrchestrator 模拟的编排器（用于测试路由注册）
// TODO: 正式环境需要替换为真实的 Orchestrator 实现
type MockOrchestrator struct{}

// ChatStreamMulti 模拟多服务商并发流式响应
func (m *MockOrchestrator) ChatStreamMulti(ctx context.Context, req *types.ChatRequest) (<-chan *types.ChatResponse, error) {
	responseChan := make(chan *types.ChatResponse, 100)

	go func() {
		defer close(responseChan)

		// 为每个服务商生成模拟响应
		for _, provider := range req.Providers {
			sessionID := fmt.Sprintf("mock_session_%d", time.Now().UnixNano())

			// 发送开始事件
			responseChan <- &types.ChatResponse{
				SessionID: sessionID,
				Provider:  provider.Provider,
				Model:     provider.Model,
				EventType: "start",
				Timestamp: time.Now(),
			}

			// 模拟流式输出
			mockResponse := fmt.Sprintf("这是来自 %s (%s) 的模拟响应。", provider.Provider, provider.Model)
			words := []string{"这是", "来自", provider.Provider, "(" + provider.Model + ")", "的", "模拟", "响应", "。"}

			for i, word := range words {
				// 检查上下文是否取消
				select {
				case <-ctx.Done():
					return
				default:
					responseChan <- &types.ChatResponse{
						SessionID: sessionID,
						Provider:  provider.Provider,
						Model:     provider.Model,
						EventType: "token",
						Content:   word + " ",
						Index:     i,
						Timestamp: time.Now(),
					}
					time.Sleep(50 * time.Millisecond) // 模拟延迟
				}
			}

			// 发送完成事件
			tokenCount := len(words)
			responseChan <- &types.ChatResponse{
				SessionID:    sessionID,
				Provider:     provider.Provider,
				Model:        provider.Model,
				EventType:    "done",
				Content:      mockResponse,
				TokenCount:   &tokenCount,
				FinishReason: "stop",
				Metadata: map[string]interface{}{
					"duration_ms":        400,
					"tokens_per_second":  20.0,
					"is_mock":            true,
				},
				Timestamp: time.Now(),
			}
		}
	}()

	return responseChan, nil
}

// RegisterProvider 注册服务商（模拟实现）
func (m *MockOrchestrator) RegisterProvider(provider llm.Provider) error {
	return nil // 模拟实现，不做任何事
}

// GetProvider 获取服务商（模拟实现）
func (m *MockOrchestrator) GetProvider(name string) (llm.Provider, error) {
	return nil, fmt.Errorf("mock orchestrator does not support GetProvider")
}
