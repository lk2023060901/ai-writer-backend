package openai

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

// Provider OpenAI Provider 实现
type Provider struct {
	config *types.Config
	client *http.Client
}

// New 创建 OpenAI Provider
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
	return "openai"
}

// setHeaders 设置请求 headers（包括默认 headers 和自定义 headers）
func (p *Provider) setHeaders(req *http.Request, includeContentType bool) {
	if includeContentType {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	// 添加自定义 headers
	if p.config.Headers != nil {
		for key, value := range p.config.Headers {
			req.Header.Set(key, value)
		}
	}
}

// CreateChatCompletion 创建聊天补全（同步）
func (p *Provider) CreateChatCompletion(ctx context.Context, req types.ChatCompletionRequest) (*types.ChatCompletionResponse, error) {
	req.Stream = false
	if req.Model == "" {
		req.Model = p.config.Model
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewProviderError(p.Name(), "marshal request failed", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/chat/completions", bytes.NewBuffer(reqBody))
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

	var chatResp types.ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, types.NewProviderError(p.Name(), "unmarshal response failed", err)
	}

	return &chatResp, nil
}

// CreateChatCompletionStream 创建聊天补全（流式）
func (p *Provider) CreateChatCompletionStream(ctx context.Context, req types.ChatCompletionRequest) (<-chan types.StreamChunk, error) {
	req.Stream = true
	if req.Model == "" {
		req.Model = p.config.Model
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewProviderError(p.Name(), "marshal request failed", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/chat/completions", bytes.NewBuffer(reqBody))
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
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					chunks <- types.StreamChunk{Done: true}
					return
				}

				var chunk types.StreamChunk
				if err := json.Unmarshal([]byte(data), &chunk); err != nil {
					chunks <- types.StreamChunk{
						Done:  true,
						Error: types.NewProviderError(p.Name(), "unmarshal chunk failed", err),
					}
					return
				}

				chunks <- chunk
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
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.config.BaseURL+"/models", nil)
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
