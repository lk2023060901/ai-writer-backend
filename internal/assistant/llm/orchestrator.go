package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/types"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"go.uber.org/zap"
)

// KnowledgeSearcher 知识库搜索接口
type KnowledgeSearcher interface {
	SearchDocuments(ctx context.Context, kbID, userID, query string, topK int) ([]*KnowledgeSearchResult, error)
}

// KnowledgeSearchResult 知识库搜索结果
type KnowledgeSearchResult struct {
	DocumentID string
	Content    string
	Score      float32
	Metadata   map[string]interface{}
}

// DefaultOrchestrator 默认的多服务商编排器实现
type DefaultOrchestrator struct {
	providerFactory   ProviderFactory
	contextManager    ContextManager
	webSearch         WebSearchProvider
	fileProcessor     FileProcessor
	errorHandler      ErrorHandler
	metricsCollector  MetricsCollector
	knowledgeSearcher KnowledgeSearcher
	mu                sync.RWMutex
	logger            *zap.Logger
}

// NewOrchestrator 创建编排器实例
func NewOrchestrator(
	providerFactory ProviderFactory,
	contextManager ContextManager,
	webSearch WebSearchProvider,
	fileProcessor FileProcessor,
	errorHandler ErrorHandler,
	metricsCollector MetricsCollector,
	knowledgeSearcher KnowledgeSearcher,
	logger *zap.Logger,
) *DefaultOrchestrator {
	return &DefaultOrchestrator{
		providerFactory:   providerFactory,
		contextManager:    contextManager,
		webSearch:         webSearch,
		fileProcessor:     fileProcessor,
		errorHandler:      errorHandler,
		metricsCollector:  metricsCollector,
		knowledgeSearcher: knowledgeSearcher,
		logger:            logger,
	}
}

// RegisterProvider 注册服务商（保留兼容性，但不再使用）
func (o *DefaultOrchestrator) RegisterProvider(provider Provider) error {
	if err := provider.ValidateConfig(); err != nil {
		return fmt.Errorf("invalid provider config: %w", err)
	}
	o.logger.Info("Registered provider", zap.String("provider", provider.Name()))
	return nil
}

// GetProvider 获取服务商实例（动态创建）
func (o *DefaultOrchestrator) GetProvider(providerType string) (Provider, error) {
	// 使用 ProviderFactory 创建实例
	config := ProviderConfig{
		Provider: providerType,
	}
	return o.providerFactory.CreateProvider(config)
}

// ChatStreamMulti 并发调用多个服务商
func (o *DefaultOrchestrator) ChatStreamMulti(ctx context.Context, req *types.ChatRequest) (<-chan *types.ChatResponse, error) {
	// 1. 构建上下文（获取历史消息）
	messages, err := o.buildMessages(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to build messages: %w", err)
	}

	// 2. 处理知识库搜索（如果提供了 KnowledgeBaseID）
	if req.KnowledgeBaseID != "" && o.knowledgeSearcher != nil {
		// 记录知识库搜索开始
		logger.Info("开始知识库向量搜索",
			zap.String("knowledge_base_id", req.KnowledgeBaseID),
			zap.String("user_id", req.UserID),
			zap.String("query", req.Message))

		// 使用请求中的 UserID
		searchResults, err := o.knowledgeSearcher.SearchDocuments(ctx, req.KnowledgeBaseID, req.UserID, req.Message, 5)
		if err != nil {
			logger.Warn("知识库搜索失败", zap.Error(err))
			o.logger.Warn("Knowledge base search failed", zap.Error(err))
		} else {
			// 记录知识库搜索结果
			searchResultsJSON, _ := json.Marshal(searchResults)
			logger.Info("知识库向量搜索完成",
				zap.String("knowledge_base_id", req.KnowledgeBaseID),
				zap.Int("result_count", len(searchResults)),
				zap.String("search_results", string(searchResultsJSON)))

			// 将搜索结果添加到消息中
			messages = o.appendKnowledgeResults(messages, searchResults)
		}
	}

	// 3. 处理联网搜索（如果启用）
	if req.EnableWebSearch && o.webSearch != nil {
		searchResults, err := o.webSearch.Search(ctx, req.Message, req.SearchDepth)
		if err != nil {
			o.logger.Warn("Web search failed", zap.Error(err))
		} else {
			// 将搜索结果添加到消息中
			messages = o.appendSearchResults(messages, searchResults)
		}
	}

	// 3. 创建输出 channel
	outputChan := make(chan *types.ChatResponse, 100)

	// 4. 为每个服务商启动 goroutine
	var wg sync.WaitGroup
	sessionID := generateSessionID()

	for _, providerConfig := range req.Providers {
		wg.Add(1)

		go func(pc types.ProviderConfig) {
			defer wg.Done()

			o.logger.Info("Getting provider instance",
				zap.String("provider_id", pc.Provider),
				zap.String("model", pc.Model))

			// 获取服务商实例
			provider, err := o.GetProvider(pc.Provider)
			if err != nil {
				o.logger.Error("Failed to get provider",
					zap.String("provider_id", pc.Provider),
					zap.Error(err))
				o.sendErrorResponse(outputChan, sessionID, pc.Provider, pc.Model, err)
				return
			}

			o.logger.Info("Provider instance created successfully",
				zap.String("provider_id", pc.Provider),
				zap.String("provider_name", provider.Name()))

			// 记录请求
			if o.metricsCollector != nil {
				o.metricsCollector.RecordRequest(pc.Provider, pc.Model)
			}
			startTime := time.Now()

			// 构建请求
			llmReq := &ChatRequest{
				Messages:        messages,
				Model:           pc.Model,
				Temperature:     pc.Temperature,
				MaxTokens:       pc.MaxTokens,
				SystemPrompt:    req.SystemPrompt,
				Stream:          true,
				ProviderOptions: pc.Options,
			}

			// 记录发送给 AI 服务商的完整请求数据
			llmReqJSON, _ := json.Marshal(llmReq)
			logger.Info("发送给AI服务商的完整请求",
				zap.String("provider", pc.Provider),
				zap.String("model", pc.Model),
				zap.String("session_id", sessionID),
				zap.String("request_data", string(llmReqJSON)))

			// 调用服务商流式 API
			o.logger.Info("Calling provider ChatStream",
				zap.String("provider_id", pc.Provider),
				zap.String("model", pc.Model))

			streamChan, err := provider.ChatStream(ctx, llmReq)
			if err != nil {
				o.logger.Error("Provider ChatStream failed",
					zap.String("provider_id", pc.Provider),
					zap.String("model", pc.Model),
					zap.Error(err))
				o.sendErrorResponse(outputChan, sessionID, pc.Provider, pc.Model, err)
				if o.metricsCollector != nil {
					o.metricsCollector.RecordError(pc.Provider, pc.Model, "stream_error")
				}
				return
			}

			o.logger.Info("ChatStream started, forwarding events",
				zap.String("provider_id", pc.Provider),
				zap.String("model", pc.Model))

			// 转发流式事件
			o.forwardStreamEvents(ctx, streamChan, outputChan, sessionID, pc.Provider, pc.Model, startTime)

		}(providerConfig)
	}

	// 5. 等待所有服务商完成后关闭输出 channel
	go func() {
		wg.Wait()
		close(outputChan)
		o.logger.Info("All providers completed")
	}()

	return outputChan, nil
}

// buildMessages 构建完整的消息列表
func (o *DefaultOrchestrator) buildMessages(ctx context.Context, req *types.ChatRequest) ([]Message, error) {
	var messages []Message

	// 1. 获取历史消息（如果有 TopicID）
	if req.TopicID != "" && o.contextManager != nil {
		history, err := o.contextManager.GetHistory(req.TopicID, 20) // 最多 20 条历史
		if err != nil {
			o.logger.Warn("Failed to get history", zap.Error(err))
		} else {
			messages = append(messages, history...)
		}
	}

	// 2. 构建当前用户消息
	userMessage := Message{
		Role:    "user",
		Content: o.buildContentBlocks(req),
	}

	messages = append(messages, userMessage)

	return messages, nil
}

// buildContentBlocks 构建内容块
func (o *DefaultOrchestrator) buildContentBlocks(req *types.ChatRequest) []ContentBlock {
	var blocks []ContentBlock

	// 1. 添加文本消息
	if req.Message != "" {
		blocks = append(blocks, ContentBlock{
			Type: "text",
			Text: req.Message,
		})
	}

	// 2. 添加多模态内容块
	for _, cb := range req.ContentBlocks {
		switch cb.Type {
		case "text":
			blocks = append(blocks, ContentBlock{
				Type: "text",
				Text: cb.Text,
			})

		case "image":
			blocks = append(blocks, ContentBlock{
				Type: "image_url",
				ImageURL: &ImageURL{
					URL:    cb.ImageURL,
					Detail: cb.ImageDetail,
				},
			})

		case "file":
			// TODO: 处理文件上传
			blocks = append(blocks, ContentBlock{
				Type:         "file",
				FileURL:      cb.FileURL,
				FileMimeType: cb.FileMimeType,
			})
		}
	}

	return blocks
}

// appendKnowledgeResults 将知识库搜索结果添加到消息中
func (o *DefaultOrchestrator) appendKnowledgeResults(messages []Message, results []*KnowledgeSearchResult) []Message {
	if len(results) == 0 {
		return messages
	}

	// 构建知识库搜索结果文本
	knowledgeText := "以下是知识库中的相关内容：\n\n"
	for i, result := range results {
		fileName := "未知文档"
		if result.Metadata != nil {
			if name, ok := result.Metadata["file_name"].(string); ok {
				fileName = name
			}
		}
		knowledgeText += fmt.Sprintf("%d. [相似度: %.2f] 来自文档: %s\n%s\n\n",
			i+1, result.Score, fileName, result.Content)
	}

	// 添加系统消息
	knowledgeMessage := Message{
		Role: "user",
		Content: []ContentBlock{
			{
				Type: "text",
				Text: knowledgeText,
			},
		},
	}

	return append(messages, knowledgeMessage)
}

// appendSearchResults 将搜索结果添加到消息中
func (o *DefaultOrchestrator) appendSearchResults(messages []Message, results []types.WebSearchResult) []Message {
	if len(results) == 0 {
		return messages
	}

	// 构建搜索结果文本
	searchText := "以下是联网搜索的结果：\n\n"
	for i, result := range results {
		searchText += fmt.Sprintf("%d. %s\n%s\n来源：%s\n\n", i+1, result.Title, result.Snippet, result.URL)
	}

	// 添加系统消息
	searchMessage := Message{
		Role: "user",
		Content: []ContentBlock{
			{
				Type: "text",
				Text: searchText,
			},
		},
	}

	return append(messages, searchMessage)
}

// forwardStreamEvents 转发流式事件
func (o *DefaultOrchestrator) forwardStreamEvents(
	ctx context.Context,
	streamChan <-chan StreamEvent,
	outputChan chan<- *types.ChatResponse,
	sessionID, provider, model string,
	startTime time.Time,
) {
	var tokenCount int
	var totalContent string

	o.logger.Info("Starting to forward stream events",
		zap.String("provider", provider),
		zap.String("model", model))

	for {
		select {
		case <-ctx.Done():
			o.logger.Warn("Context cancelled")
			return

		case event, ok := <-streamChan:
			if !ok {
				// Stream 关闭，发送完成事件
				o.logger.Info("Stream channel closed, sending done event",
					zap.String("provider", provider),
					zap.Int("token_count", tokenCount),
					zap.Int("content_length", len(totalContent)))

				duration := time.Since(startTime).Seconds()
				if o.metricsCollector != nil {
					o.metricsCollector.RecordLatency(provider, model, duration)
				}

				// 记录 AI 服务商的完整流式响应（汇总）
				streamResponseData := map[string]interface{}{
					"provider":      provider,
					"model":         model,
					"session_id":    sessionID,
					"content":       totalContent,
					"token_count":   tokenCount,
					"finish_reason": "stop",
					"duration":      duration,
				}
				streamResponseJSON, _ := json.Marshal(streamResponseData)
				logger.Info("AI服务商流式响应完成汇总",
					zap.String("provider", provider),
					zap.String("model", model),
					zap.String("session_id", sessionID),
					zap.Int("token_count", tokenCount),
					zap.Float64("duration", duration),
					zap.String("response_data", string(streamResponseJSON)))

				outputChan <- &types.ChatResponse{
					SessionID:    sessionID,
					Provider:     provider,
					Model:        model,
					EventType:    "done",
					Content:      totalContent,
					TokenCount:   &tokenCount,
					FinishReason: "stop",
					Timestamp:    time.Now(),
				}
				return
			}

			o.logger.Debug("Received stream event",
				zap.String("provider", provider),
				zap.String("event_type", string(event.Type)))

			// 处理事件
			switch event.Type {
			case EventStart:
				outputChan <- &types.ChatResponse{
					SessionID: sessionID,
					Provider:  provider,
					Model:     model,
					EventType: "start",
					Timestamp: time.Now(),
				}

			case EventToken:
				tokenCount++
				totalContent += event.Content

				// 记录每个 token（调试级别，避免日志过多）
				o.logger.Debug("AI服务商返回token",
					zap.String("provider", provider),
					zap.String("model", model),
					zap.String("session_id", sessionID),
					zap.Int("index", event.Index),
					zap.String("content", event.Content))

				outputChan <- &types.ChatResponse{
					SessionID: sessionID,
					Provider:  provider,
					Model:     model,
					EventType: "token",
					Content:   event.Content,
					Index:     event.Index,
					Timestamp: time.Now(),
				}

			case EventError:
				if o.metricsCollector != nil {
					o.metricsCollector.RecordError(provider, model, "stream_error")
				}
				outputChan <- &types.ChatResponse{
					SessionID: sessionID,
					Provider:  provider,
					Model:     model,
					EventType: "error",
					Error:     event.Error.Error(),
					Timestamp: time.Now(),
				}
				return

			case EventDone:
				// 服务商发送的完成事件
				if o.metricsCollector != nil {
					o.metricsCollector.RecordTokens(provider, model, 0, tokenCount)
				}
				return
			}
		}
	}
}

// sendErrorResponse 发送错误响应
func (o *DefaultOrchestrator) sendErrorResponse(
	outputChan chan<- *types.ChatResponse,
	sessionID, provider, model string,
	err error,
) {
	o.logger.Error("Provider error", zap.String("provider", provider), zap.Error(err))

	outputChan <- &types.ChatResponse{
		SessionID: sessionID,
		Provider:  provider,
		Model:     model,
		EventType: "error",
		Error:     err.Error(),
		Timestamp: time.Now(),
	}
}

// generateSessionID 生成会话 ID
func generateSessionID() string {
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}
