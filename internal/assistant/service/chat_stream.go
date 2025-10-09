package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/assistant/llm"
	"github.com/lk2023060901/ai-writer-backend/internal/assistant/types"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"go.uber.org/zap"
)

// ChatStreamV2 多模态多服务商流式聊天接口
// @Summary Multi-provider multimodal streaming chat
// @Tags chat
// @Accept json
// @Produce text/event-stream
// @Param request body types.ChatRequest true "Chat Request"
// @Success 200 {object} types.ChatResponse
// @Router /api/v1/chat/stream [post]
func (s *AssistantService) ChatStreamV2(c *gin.Context) {
	var req types.ChatRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户 ID
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	ctx := c.Request.Context()

	// 处理 topic_id：如果没有提供，则创建新会话
	topicID := req.TopicID
	if topicID == "" {
		// 获取用户的第一个 agent 作为默认 assistant
		assistants, err := s.useCase.ListAssistants(ctx, userID, &types.AssistantFilter{})
		if err != nil || len(assistants) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no assistant found, please create an assistant first or provide topic_id"})
			return
		}
		assistantID := assistants[0].ID

		topic, err := s.topicUseCase.CreateTopic(ctx, userID, assistantID, "新对话")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create topic: %v", err)})
			return
		}
		topicID = topic.ID
	}

	// 保存用户消息到数据库
	userContentBlocks := []types.ContentBlock{
		{
			Type: "text",
			Text: req.Message,
		},
	}

	// 设置 UserID 到请求中，供 orchestrator 使用
	req.UserID = userID

	// 记录完整的用户请求数据
	requestJSON, _ := json.Marshal(req)
	logger.Info("用户提问完整请求",
		zap.String("user_id", userID),
		zap.String("topic_id", topicID),
		zap.String("request_data", string(requestJSON)))

	userMessage, err := s.messageUseCase.CreateMessage(ctx, topicID, "user", userContentBlocks, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save user message: %v", err)})
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 获取 orchestrator
	orchestrator := s.getOrchestrator()
	if orchestrator == nil {
		s.writeSSEError(c, "orchestrator not initialized")
		return
	}

	// 调用多服务商并发流式响应
	responseChan, err := orchestrator.ChatStreamMulti(ctx, &req)
	if err != nil {
		s.writeSSEError(c, fmt.Sprintf("failed to start chat stream: %v", err))
		return
	}

	// 流式输出响应并保存
	s.streamAndSaveResponses(c, responseChan, topicID, userMessage.ID)
}

// streamAndSaveResponses 流式输出多服务商响应并保存到数据库
func (s *AssistantService) streamAndSaveResponses(c *gin.Context, responseChan <-chan *types.ChatResponse, topicID, userMessageID string) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		s.writeSSEError(c, "streaming not supported")
		return
	}

	// 用于收集每个 provider 的完整响应
	type ProviderData struct {
		Content    string
		Model      string
		TokenCount int
	}
	providerResponses := make(map[string]*ProviderData) // key: provider, value: complete data

	for response := range responseChan {
		// 初始化 provider 数据
		if _, exists := providerResponses[response.Provider]; !exists {
			providerResponses[response.Provider] = &ProviderData{
				Model: response.Model,
			}
		}

		// 序列化响应
		data, err := json.Marshal(response)
		if err != nil {
			s.writeSSEError(c, fmt.Sprintf("failed to marshal response: %v", err))
			continue
		}

		// 写入 SSE 格式数据
		fmt.Fprintf(c.Writer, "event: %s\n", response.EventType)
		fmt.Fprintf(c.Writer, "data: %s\n\n", string(data))
		flusher.Flush()

		// 收集响应内容
		if response.EventType == "token" {
			providerResponses[response.Provider].Content += response.Content
		} else if response.EventType == "done" {
			// 保存完整的响应内容
			if response.TokenCount != nil {
				providerResponses[response.Provider].TokenCount = *response.TokenCount
			}
			// 使用 done 事件中的完整内容
			if response.Content != "" {
				providerResponses[response.Provider].Content = response.Content
			}
			// 更新 model 信息
			if response.Model != "" {
				providerResponses[response.Provider].Model = response.Model
			}
		}

		// 检查客户端是否断开连接
		if c.Request.Context().Err() != nil {
			return
		}
	}

	// 保存所有 provider 的响应到数据库
	ctx := c.Request.Context()
	for provider, data := range providerResponses {
		if data.Content == "" {
			continue
		}

		// 记录 AI 服务商的完整响应数据
		responseData := map[string]interface{}{
			"provider":    provider,
			"model":       data.Model,
			"content":     data.Content,
			"token_count": data.TokenCount,
		}
		responseJSON, _ := json.Marshal(responseData)
		logger.Info("AI服务商回复完整数据",
			zap.String("topic_id", topicID),
			zap.String("provider", provider),
			zap.String("model", data.Model),
			zap.Int("token_count", data.TokenCount),
			zap.String("response_data", string(responseJSON)))

		assistantContentBlocks := []types.ContentBlock{
			{
				Type: "text",
				Text: data.Content,
			},
		}

		_, err := s.messageUseCase.CreateMessageWithModel(ctx, topicID, "assistant", assistantContentBlocks, &data.TokenCount, provider, data.Model)
		if err != nil {
			// 记录错误但不中断流程
			logger.Error("保存助手消息失败",
				zap.String("provider", provider),
				zap.Error(err))
		}
	}

	// 发送完成信号
	fmt.Fprintf(c.Writer, "event: all_done\n")
	fmt.Fprintf(c.Writer, "data: {\"message\":\"All providers completed\"}\n\n")
	flusher.Flush()
}

// streamMultiProviderResponses 流式输出多服务商响应（不保存）
func (s *AssistantService) streamMultiProviderResponses(c *gin.Context, responseChan <-chan *types.ChatResponse) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		s.writeSSEError(c, "streaming not supported")
		return
	}

	for response := range responseChan {
		// 序列化响应
		data, err := json.Marshal(response)
		if err != nil {
			s.writeSSEError(c, fmt.Sprintf("failed to marshal response: %v", err))
			continue
		}

		// 写入 SSE 格式数据
		fmt.Fprintf(c.Writer, "event: %s\n", response.EventType)
		fmt.Fprintf(c.Writer, "data: %s\n\n", string(data))
		flusher.Flush()

		// 检查客户端是否断开连接
		if c.Request.Context().Err() != nil {
			return
		}
	}

	// 发送完成信号
	fmt.Fprintf(c.Writer, "event: all_done\n")
	fmt.Fprintf(c.Writer, "data: {\"message\":\"All providers completed\"}\n\n")
	flusher.Flush()
}

// writeSSEError 写入 SSE 错误
func (s *AssistantService) writeSSEError(c *gin.Context, errMsg string) {
	errorData := map[string]interface{}{
		"error": errMsg,
	}
	data, _ := json.Marshal(errorData)

	fmt.Fprintf(c.Writer, "event: error\n")
	fmt.Fprintf(c.Writer, "data: %s\n\n", string(data))

	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

// getOrchestrator 获取 orchestrator 实例
func (s *AssistantService) getOrchestrator() llm.MultiProviderOrchestrator {
	return s.orchestrator
}

// ChatStreamLegacy 保留旧版本的简单聊天接口（向后兼容）
// @Summary Legacy streaming chat (single provider)
// @Tags chat
// @Accept json
// @Produce text/event-stream
// @Router /api/v1/assistants/:id/chat-stream [post]
func (s *AssistantService) ChatStreamLegacy(c *gin.Context) {
	var req struct {
		Message         string `json:"message" binding:"required"`
		AssistantID     string `json:"assistant_id"`
		KnowledgeBaseID string `json:"knowledge_base_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 转换为新格式并调用 ChatStreamV2
	chatReq := types.ChatRequest{
		Message: req.Message,
		Providers: []types.ProviderConfig{
			{
				Provider: "openai",
				Model:    "gpt-4o-mini",
			},
		},
		KnowledgeBaseID: req.KnowledgeBaseID,
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 获取 orchestrator
	orchestrator := s.getOrchestrator()
	if orchestrator == nil {
		s.writeSSEError(c, "orchestrator not initialized")
		return
	}

	// 调用流式响应
	ctx := c.Request.Context()
	responseChan, err := orchestrator.ChatStreamMulti(ctx, &chatReq)
	if err != nil {
		s.writeSSEError(c, fmt.Sprintf("failed to start chat stream: %v", err))
		return
	}

	// 流式输出（简化格式，兼容旧客户端）
	s.streamLegacyFormat(c, responseChan)
}

// streamLegacyFormat 流式输出（旧格式）
func (s *AssistantService) streamLegacyFormat(c *gin.Context, responseChan <-chan *types.ChatResponse) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		s.writeSSEError(c, "streaming not supported")
		return
	}

	for response := range responseChan {
		// 只输出 token 和 done 事件
		if response.EventType == "token" {
			data := map[string]interface{}{
				"content": response.Content,
				"index":   response.Index,
			}
			dataBytes, _ := json.Marshal(data)

			fmt.Fprintf(c.Writer, "event: token\n")
			fmt.Fprintf(c.Writer, "data: %s\n\n", string(dataBytes))
			flusher.Flush()
		} else if response.EventType == "done" {
			data := map[string]interface{}{
				"message": "Response completed",
			}
			dataBytes, _ := json.Marshal(data)

			fmt.Fprintf(c.Writer, "event: done\n")
			fmt.Fprintf(c.Writer, "data: %s\n\n", string(dataBytes))
			flusher.Flush()
			return
		} else if response.EventType == "error" {
			data := map[string]interface{}{
				"error": response.Error,
			}
			dataBytes, _ := json.Marshal(data)

			fmt.Fprintf(c.Writer, "event: error\n")
			fmt.Fprintf(c.Writer, "data: %s\n\n", string(dataBytes))
			flusher.Flush()
			return
		}
	}
}

// InitializeOrchestrator 初始化 orchestrator（在启动时调用）
func (s *AssistantService) InitializeOrchestrator(orchestrator llm.MultiProviderOrchestrator) {
	// 将 orchestrator 注入到 service 中
	// 这是一个临时方案，更好的做法是通过构造函数注入
	// TODO: 重构为通过构造函数注入
}
