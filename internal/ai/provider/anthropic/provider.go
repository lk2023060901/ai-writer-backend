package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lk2023060901/ai-writer-backend/internal/ai/provider/types"
)

// Provider Anthropic Provider 实现（直接处理协议转换）
type Provider struct {
	config *types.Config
	client *http.Client
}

// New 创建 Anthropic Provider
func New(config *types.Config) (*Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Provider{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

// Name 返回 Provider 名称
func (p *Provider) Name() string {
	return "anthropic"
}

// setHeaders 设置请求 headers（包括默认 headers 和自定义 headers）
func (p *Provider) setHeaders(req *http.Request, includeContentType bool) {
	if includeContentType {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// 添加自定义 headers
	if p.config.Headers != nil {
		for key, value := range p.config.Headers {
			req.Header.Set(key, value)
		}
	}
}

// Anthropic 内部请求结构
type anthropicRequest struct {
	Model       string              `json:"model"`
	Messages    []anthropicMessage  `json:"messages"`
	System      string              `json:"system,omitempty"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature float64             `json:"temperature,omitempty"`
	TopP        float64             `json:"top_p,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
	StopSequences []string          `json:"stop_sequences,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Anthropic 内部响应结构
type anthropicResponse struct {
	ID           string              `json:"id"`
	Type         string              `json:"type"`
	Role         string              `json:"role"`
	Content      []anthropicContent  `json:"content"`
	Model        string              `json:"model"`
	StopReason   string              `json:"stop_reason"`
	Usage        anthropicUsage      `json:"usage"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Anthropic 流式响应事件
type anthropicStreamEvent struct {
	Type         string              `json:"type"`
	Index        int                 `json:"index,omitempty"`
	Delta        *anthropicDelta     `json:"delta,omitempty"`
	Message      *anthropicResponse  `json:"message,omitempty"`
	ContentBlock *anthropicContent   `json:"content_block,omitempty"`
	Usage        *anthropicUsage     `json:"usage,omitempty"`
}

type anthropicDelta struct {
	Type       string `json:"type"`
	Text       string `json:"text,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}

// CreateChatCompletion 创建聊天补全（同步）
// 处理 OpenAI 格式到 Anthropic 格式的转换
func (p *Provider) CreateChatCompletion(ctx context.Context, req types.ChatCompletionRequest) (*types.ChatCompletionResponse, error) {
	// 转换为 Anthropic 格式
	anthropicReq := p.convertRequest(req)

	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, types.NewProviderError(p.Name(), "marshal request failed", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, types.NewProviderError(p.Name(), "create request failed", err)
	}

	p.setHeaders(httpReq, true)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, types.NewProviderError(p.Name(), "request failed", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewProviderError(p.Name(), "read response failed", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &types.ProviderError{
			Provider:   p.Name(),
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("API error: %s", string(body)),
		}
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, types.NewProviderError(p.Name(), "unmarshal response failed", err)
	}

	// 转换为 OpenAI 格式
	return p.convertResponse(&anthropicResp), nil
}

// CreateChatCompletionStream 创建聊天补全（流式）
func (p *Provider) CreateChatCompletionStream(ctx context.Context, req types.ChatCompletionRequest) (<-chan types.StreamChunk, error) {
	// 转换为 Anthropic 格式
	anthropicReq := p.convertRequest(req)
	anthropicReq.Stream = true

	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, types.NewProviderError(p.Name(), "marshal request failed", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, types.NewProviderError(p.Name(), "create request failed", err)
	}

	p.setHeaders(httpReq, true)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, types.NewProviderError(p.Name(), "request failed", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &types.ProviderError{
			Provider:   p.Name(),
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("API error: %s", string(body)),
		}
	}

	chunks := make(chan types.StreamChunk, 10)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		var messageID string
		var model string

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			if strings.HasPrefix(line, "event: ") {
				// Anthropic 使用 event: 标识事件类型
				continue
			}

			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")

				var event anthropicStreamEvent
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					chunks <- types.StreamChunk{
						Done:  true,
						Error: types.NewProviderError(p.Name(), "unmarshal event failed", err),
					}
					return
				}

				// 处理不同类型的事件
				switch event.Type {
				case "message_start":
					if event.Message != nil {
						messageID = event.Message.ID
						model = event.Message.Model
					}
				case "content_block_delta":
					if event.Delta != nil && event.Delta.Text != "" {
						chunks <- types.StreamChunk{
							ID:     messageID,
							Object: "chat.completion.chunk",
							Model:  model,
							Choices: []types.StreamChoice{
								{
									Index: 0,
									Delta: types.MessageDelta{
										Content: event.Delta.Text,
									},
								},
							},
						}
					}
				case "message_delta":
					if event.Delta != nil && event.Delta.StopReason != "" {
						finishReason := event.Delta.StopReason
						chunks <- types.StreamChunk{
							ID:     messageID,
							Object: "chat.completion.chunk",
							Model:  model,
							Choices: []types.StreamChoice{
								{
									Index:        0,
									FinishReason: &finishReason,
								},
							},
							Usage: p.convertUsage(event.Usage),
						}
					}
				case "message_stop":
					chunks <- types.StreamChunk{Done: true}
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			chunks <- types.StreamChunk{
				Done:  true,
				Error: types.NewProviderError(p.Name(), "read stream failed", err),
			}
		}
	}()

	return chunks, nil
}

// ListModels 获取可用模型列表
func (p *Provider) ListModels(ctx context.Context) ([]types.Model, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.config.BaseURL+"/v1/models", nil)
	if err != nil {
		return nil, types.NewProviderError(p.Name(), "create request failed", err)
	}

	p.setHeaders(httpReq, false)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, types.NewProviderError(p.Name(), "request failed", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewProviderError(p.Name(), "read response failed", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &types.ProviderError{
			Provider:   p.Name(),
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("API error: %s", string(body)),
		}
	}

	var modelsResp types.ModelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, types.NewProviderError(p.Name(), "unmarshal response failed", err)
	}

	return modelsResp.Data, nil
}

// Close 关闭 Provider
func (p *Provider) Close() error {
	p.client.CloseIdleConnections()
	return nil
}

// convertRequest 将 OpenAI 请求转换为 Anthropic 请求
func (p *Provider) convertRequest(req types.ChatCompletionRequest) *anthropicRequest {
	anthropicReq := &anthropicRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}

	if anthropicReq.Model == "" {
		anthropicReq.Model = p.config.Model
	}

	if anthropicReq.MaxTokens == 0 {
		anthropicReq.MaxTokens = 1024
	}

	// 转换 stop 为 stop_sequences
	if len(req.Stop) > 0 {
		anthropicReq.StopSequences = req.Stop
	}

	// 分离 system 消息和其他消息
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			anthropicReq.System = msg.Content
		} else {
			anthropicReq.Messages = append(anthropicReq.Messages, anthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	return anthropicReq
}

// convertResponse 将 Anthropic 响应转换为 OpenAI 响应
func (p *Provider) convertResponse(resp *anthropicResponse) *types.ChatCompletionResponse {
	var content string
	if len(resp.Content) > 0 {
		content = resp.Content[0].Text
	}

	return &types.ChatCompletionResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Model:   resp.Model,
		Choices: []types.Choice{
			{
				Index: 0,
				Message: types.Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: resp.StopReason,
			},
		},
		Usage: types.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

// convertUsage 转换 usage 信息
func (p *Provider) convertUsage(usage *anthropicUsage) *types.Usage {
	if usage == nil {
		return nil
	}
	return &types.Usage{
		PromptTokens:     usage.InputTokens,
		CompletionTokens: usage.OutputTokens,
		TotalTokens:      usage.InputTokens + usage.OutputTokens,
	}
}
