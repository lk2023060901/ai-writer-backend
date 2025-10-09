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
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/llm"
)

// AnthropicProvider Anthropic (Claude) 服务商适配器
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewAnthropicProvider 创建 Anthropic 提供者
func NewAnthropicProvider(apiKey, baseURL string) *AnthropicProvider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 120 * time.Second, // 设置2分钟超时
		},
	}
}

// Name 返回服务商名称
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// ValidateConfig 验证配置
func (p *AnthropicProvider) ValidateConfig() error {
	if p.apiKey == "" {
		return fmt.Errorf("anthropic api key is required")
	}
	return nil
}

// SupportedModels 返回支持的模型
func (p *AnthropicProvider) SupportedModels() []string {
	return []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}
}

// SupportsMultimodal 是否支持多模态
func (p *AnthropicProvider) SupportsMultimodal() bool {
	return true
}

// ChatStream 流式聊天
func (p *AnthropicProvider) ChatStream(ctx context.Context, req *llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	// 1. 转换为 Anthropic 请求格式
	anthropicReq := p.convertRequest(req)

	// 2. 序列化请求
	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 3. 创建 HTTP 请求
	// 确保 baseURL 以 /v1 结尾，如果没有则添加
	apiURL := p.baseURL
	if !strings.HasSuffix(apiURL, "/v1") {
		apiURL = apiURL + "/v1"
	}
	apiURL = apiURL + "/messages"

	fmt.Printf("[Anthropic] Sending request to %s\n", apiURL)
	fmt.Printf("[Anthropic] Request body: %s\n", string(body))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// 4. 发送请求
	resp, err := p.client.Do(httpReq)
	if err != nil {
		fmt.Printf("[Anthropic] Request failed: %v\n", err)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	fmt.Printf("[Anthropic] Response status: %s\n", resp.Status)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("[Anthropic] API error response: %s\n", string(body))
		return nil, fmt.Errorf("anthropic api error: %s - %s", resp.Status, string(body))
	}

	// 5. 创建事件 channel
	eventChan := make(chan llm.StreamEvent, 100)

	// 6. 启动 goroutine 读取流式响应
	go p.readStream(resp.Body, eventChan)

	return eventChan, nil
}

// convertRequest 转换请求格式
func (p *AnthropicProvider) convertRequest(req *llm.ChatRequest) map[string]interface{} {
	anthropicReq := map[string]interface{}{
		"model":    req.Model,
		"messages": p.convertMessages(req.Messages),
		"stream":   true,
	}

	if req.MaxTokens != nil {
		anthropicReq["max_tokens"] = *req.MaxTokens
	} else {
		// Anthropic 要求必须提供 max_tokens
		anthropicReq["max_tokens"] = 4096
	}

	if req.Temperature != nil {
		anthropicReq["temperature"] = *req.Temperature
	}

	if req.TopP != nil {
		anthropicReq["top_p"] = *req.TopP
	}

	// 添加系统提示（Anthropic 使用独立的 system 字段）
	if req.SystemPrompt != "" {
		anthropicReq["system"] = req.SystemPrompt
	}

	return anthropicReq
}

// convertMessages 转换消息格式
func (p *AnthropicProvider) convertMessages(messages []llm.Message) []map[string]interface{} {
	var result []map[string]interface{}

	for _, msg := range messages {
		// 跳过 system 角色（Anthropic 使用独立的 system 字段）
		if msg.Role == "system" {
			continue
		}

		anthropicMsg := map[string]interface{}{
			"role": msg.Role,
		}

		// 处理内容
		if len(msg.Content) == 1 && msg.Content[0].Type == "text" {
			// 简单文本消息
			anthropicMsg["content"] = msg.Content[0].Text
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
					// Anthropic 使用 image 类型，需要 base64 编码的数据或 URL
					// 这里假设 URL 是可访问的图片
					contentArray = append(contentArray, map[string]interface{}{
						"type": "image",
						"source": map[string]interface{}{
							"type": "url",
							"url":  block.ImageURL.URL,
						},
					})
				}
			}
			anthropicMsg["content"] = contentArray
		}

		result = append(result, anthropicMsg)
	}

	return result
}

// readStream 读取流式响应
func (p *AnthropicProvider) readStream(body io.ReadCloser, eventChan chan<- llm.StreamEvent) {
	defer close(eventChan)
	defer body.Close()

	fmt.Println("[Anthropic] Starting to read stream")

	scanner := bufio.NewScanner(body)
	tokenIndex := 0
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		fmt.Printf("[Anthropic] Line %d: %s\n", lineCount, line)

		// SSE 格式: "event: xxx" 和 "data: {...}"
		if strings.HasPrefix(line, "event: ") {
			continue // 跳过事件类型行，直接读数据
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		fmt.Printf("[Anthropic] Processing data: %s\n", data)

		// 解析 JSON
		var event AnthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			eventChan <- llm.StreamEvent{
				Type:  llm.EventError,
				Error: fmt.Errorf("failed to parse event: %w", err),
			}
			continue
		}

		// 处理不同事件类型
		switch event.Type {
		case "message_start":
			// 消息开始
			eventChan <- llm.StreamEvent{
				Type: llm.EventStart,
			}

		case "content_block_start":
			// 内容块开始（可以忽略或记录）
			continue

		case "content_block_delta":
			// 内容增量
			if event.Delta.Type == "text_delta" {
				eventChan <- llm.StreamEvent{
					Type:    llm.EventToken,
					Content: event.Delta.Text,
					Index:   tokenIndex,
				}
				tokenIndex++
			} else if event.Delta.Type == "thinking_delta" {
				// 处理思考过程（Extended Thinking）
				// 可以选择发送 EventThink 或者 EventToken
				eventChan <- llm.StreamEvent{
					Type:    llm.EventThink,
					Content: event.Delta.Thinking,
					Index:   tokenIndex,
				}
				tokenIndex++
			}

		case "content_block_stop":
			// 内容块结束
			continue

		case "message_delta":
			// 消息元数据更新（如 stop_reason）
			if event.Delta.StopReason != "" {
				eventChan <- llm.StreamEvent{
					Type:         llm.EventDone,
					FinishReason: event.Delta.StopReason,
				}
			}

		case "message_stop":
			// 消息结束
			eventChan <- llm.StreamEvent{
				Type: llm.EventDone,
			}
			return

		case "error":
			// 错误事件
			eventChan <- llm.StreamEvent{
				Type:  llm.EventError,
				Error: fmt.Errorf("anthropic error: %s", event.Error.Message),
			}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("[Anthropic] Scanner error: %v\n", err)
		eventChan <- llm.StreamEvent{
			Type:  llm.EventError,
			Error: fmt.Errorf("stream read error: %w", err),
		}
	}

	fmt.Printf("[Anthropic] Stream finished, read %d lines\n", lineCount)
}

// Anthropic 响应结构

type AnthropicStreamEvent struct {
	Type  string                `json:"type"`
	Delta AnthropicDelta        `json:"delta,omitempty"`
	Error *AnthropicError       `json:"error,omitempty"`
}

type AnthropicDelta struct {
	Type       string `json:"type,omitempty"`
	Text       string `json:"text,omitempty"`
	Thinking   string `json:"thinking,omitempty"`   // Extended thinking content
	StopReason string `json:"stop_reason,omitempty"`
}

type AnthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
