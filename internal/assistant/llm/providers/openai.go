package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/llm"
)

// OpenAIProvider OpenAI 服务商适配器
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewOpenAIProvider 创建 OpenAI 提供者
func NewOpenAIProvider(apiKey, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// Name 返回服务商名称
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// ValidateConfig 验证配置
func (p *OpenAIProvider) ValidateConfig() error {
	if p.apiKey == "" {
		return fmt.Errorf("openai api key is required")
	}
	return nil
}

// SupportedModels 返回支持的模型
func (p *OpenAIProvider) SupportedModels() []string {
	return []string{
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-3.5-turbo",
		"o1",
		"o1-mini",
	}
}

// SupportsMultimodal 是否支持多模态
func (p *OpenAIProvider) SupportsMultimodal() bool {
	return true
}

// ChatStream 流式聊天
func (p *OpenAIProvider) ChatStream(ctx context.Context, req *llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	// 1. 转换为 OpenAI 请求格式
	openaiReq := p.convertRequest(req)

	// 2. 序列化请求
	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 3. 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	// 4. 发送请求
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("openai api error: %s - %s", resp.Status, string(body))
	}

	// 5. 创建事件 channel
	eventChan := make(chan llm.StreamEvent, 100)

	// 6. 启动 goroutine 读取流式响应
	go p.readStream(resp.Body, eventChan)

	return eventChan, nil
}

// convertRequest 转换请求格式
func (p *OpenAIProvider) convertRequest(req *llm.ChatRequest) map[string]interface{} {
	openaiReq := map[string]interface{}{
		"model":    req.Model,
		"messages": p.convertMessages(req.Messages),
		"stream":   true,
	}

	if req.Temperature != nil {
		openaiReq["temperature"] = *req.Temperature
	}

	if req.MaxTokens != nil {
		openaiReq["max_tokens"] = *req.MaxTokens
	}

	if req.TopP != nil {
		openaiReq["top_p"] = *req.TopP
	}

	// 添加系统提示（如果有）
	if req.SystemPrompt != "" {
		messages := openaiReq["messages"].([]map[string]interface{})
		systemMsg := map[string]interface{}{
			"role":    "system",
			"content": req.SystemPrompt,
		}
		openaiReq["messages"] = append([]map[string]interface{}{systemMsg}, messages...)
	}

	return openaiReq
}

// convertMessages 转换消息格式
func (p *OpenAIProvider) convertMessages(messages []llm.Message) []map[string]interface{} {
	var result []map[string]interface{}

	for _, msg := range messages {
		openaiMsg := map[string]interface{}{
			"role": msg.Role,
		}

		// 处理内容
		if len(msg.Content) == 1 && msg.Content[0].Type == "text" {
			// 简单文本消息
			openaiMsg["content"] = msg.Content[0].Text
		} else {
			// 多模态消息
			var contentArray []map[string]interface{}
			for _, block := range msg.Content {
				switch block.Type {
				case "text":
					contentArray = append(contentArray, map[string]interface{}{
						"type": "text",
						"text": block.Text,
					})
				case "image_url":
					imageContent := map[string]interface{}{
						"type": "image_url",
						"image_url": map[string]interface{}{
							"url": block.ImageURL.URL,
						},
					}
					if block.ImageURL.Detail != "" {
						imageContent["image_url"].(map[string]interface{})["detail"] = block.ImageURL.Detail
					}
					contentArray = append(contentArray, imageContent)
				}
			}
			openaiMsg["content"] = contentArray
		}

		result = append(result, openaiMsg)
	}

	return result
}

// readStream 读取流式响应
func (p *OpenAIProvider) readStream(body io.ReadCloser, eventChan chan<- llm.StreamEvent) {
	defer close(eventChan)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	tokenIndex := 0

	for scanner.Scan() {
		line := scanner.Text()

		// SSE 格式: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// 结束标志
		if data == "[DONE]" {
			eventChan <- llm.StreamEvent{
				Type: llm.EventDone,
			}
			return
		}

		// 解析 JSON
		var chunk OpenAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			eventChan <- llm.StreamEvent{
				Type:  llm.EventError,
				Error: fmt.Errorf("failed to parse chunk: %w", err),
			}
			continue
		}

		// 提取内容
		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]

			// 检查是否完成
			if choice.FinishReason != "" {
				eventChan <- llm.StreamEvent{
					Type:         llm.EventDone,
					FinishReason: choice.FinishReason,
				}
				return
			}

			// 发送 token
			if choice.Delta.Content != "" {
				eventChan <- llm.StreamEvent{
					Type:    llm.EventToken,
					Content: choice.Delta.Content,
					Index:   tokenIndex,
				}
				tokenIndex++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		eventChan <- llm.StreamEvent{
			Type:  llm.EventError,
			Error: fmt.Errorf("stream read error: %w", err),
		}
	}
}

// OpenAI 响应结构

type OpenAIStreamChunk struct {
	ID      string                `json:"id"`
	Object  string                `json:"object"`
	Created int64                 `json:"created"`
	Model   string                `json:"model"`
	Choices []OpenAIStreamChoice  `json:"choices"`
}

type OpenAIStreamChoice struct {
	Index        int                `json:"index"`
	Delta        OpenAIDelta        `json:"delta"`
	FinishReason string             `json:"finish_reason"`
}

type OpenAIDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}
