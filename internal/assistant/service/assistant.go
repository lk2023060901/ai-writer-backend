package service

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lk2023060901/ai-writer-backend/internal/assistant/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/assistant/llm"
	"github.com/lk2023060901/ai-writer-backend/internal/assistant/types"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/sse"
)

// AssistantService handles HTTP requests for assistant operations
type AssistantService struct {
	useCase        *biz.AssistantUseCase
	topicUseCase   *biz.TopicUseCase
	messageUseCase *biz.MessageUseCase
	sseHub         *sse.Hub
	orchestrator   llm.MultiProviderOrchestrator
}

// NewAssistantService creates a new assistant service
func NewAssistantService(
	useCase *biz.AssistantUseCase,
	topicUseCase *biz.TopicUseCase,
	messageUseCase *biz.MessageUseCase,
	sseHub *sse.Hub,
	orchestrator llm.MultiProviderOrchestrator,
) *AssistantService {
	return &AssistantService{
		useCase:        useCase,
		topicUseCase:   topicUseCase,
		messageUseCase: messageUseCase,
		sseHub:         sseHub,
		orchestrator:   orchestrator,
	}
}

// RegisterRoutes registers assistant routes
func (s *AssistantService) RegisterRoutes(r *gin.RouterGroup) {
	assistants := r.Group("/assistants")
	{
		assistants.POST("", s.CreateAssistant)
		assistants.GET("", s.ListAssistants)
		assistants.GET("/:id", s.GetAssistant)
		assistants.PUT("/:id", s.UpdateAssistant)
		assistants.DELETE("/:id", s.DeleteAssistant)
	}
}

// CreateAssistant creates a new assistant
// @Summary Create assistant
// @Tags assistants
// @Accept json
// @Produce json
// @Param request body biz.CreateAssistantRequest true "Create Assistant Request"
// @Success 200 {object} types.Assistant
// @Router /api/v1/assistants [post]
func (s *AssistantService) CreateAssistant(c *gin.Context) {
	var req biz.CreateAssistantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from JWT middleware (set by auth middleware)
	userID := c.GetString("user_id")
	if userID == "" {
		// Fallback for development/testing without auth
		userID = "default-user"
	}

	assistant, err := s.useCase.CreateAssistant(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, assistant)
}

// GetAssistant retrieves an assistant by ID
// @Summary Get assistant
// @Tags assistants
// @Produce json
// @Param id path string true "Assistant ID"
// @Success 200 {object} types.Assistant
// @Router /api/v1/assistants/{id} [get]
func (s *AssistantService) GetAssistant(c *gin.Context) {
	id := c.Param("id")

	// TODO: Get user ID from context/JWT
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default-user"
	}

	assistant, err := s.useCase.GetAssistant(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, assistant)
}

// ListAssistants lists all assistants for the current user
// @Summary List assistants
// @Tags assistants
// @Produce json
// @Param tags query string false "Filter by tags (comma-separated)"
// @Param keyword query string false "Search keyword"
// @Success 200 {array} types.Assistant
// @Router /api/v1/assistants [get]
func (s *AssistantService) ListAssistants(c *gin.Context) {
	// TODO: Get user ID from context/JWT
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default-user"
	}

	var filter types.AssistantFilter
	if keyword := c.Query("keyword"); keyword != "" {
		filter.Keyword = keyword
	}
	// Parse tags from comma-separated query parameter
	if tagsStr := c.Query("tags"); tagsStr != "" {
		filter.Tags = strings.Split(tagsStr, ",")
	}

	assistants, err := s.useCase.ListAssistants(c.Request.Context(), userID, &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, assistants)
}

// UpdateAssistant updates an existing assistant
// @Summary Update assistant
// @Tags assistants
// @Accept json
// @Produce json
// @Param id path string true "Assistant ID"
// @Param request body biz.UpdateAssistantRequest true "Update Assistant Request"
// @Success 200 {object} types.Assistant
// @Router /api/v1/assistants/{id} [put]
func (s *AssistantService) UpdateAssistant(c *gin.Context) {
	id := c.Param("id")

	var req biz.UpdateAssistantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Get user ID from context/JWT
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default-user"
	}

	assistant, err := s.useCase.UpdateAssistant(c.Request.Context(), id, userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, assistant)
}

// DeleteAssistant deletes an assistant
// @Summary Delete assistant
// @Tags assistants
// @Param id path string true "Assistant ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/assistants/{id} [delete]
func (s *AssistantService) DeleteAssistant(c *gin.Context) {
	id := c.Param("id")

	// TODO: Get user ID from context/JWT
	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default-user"
	}

	if err := s.useCase.DeleteAssistant(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Assistant deleted successfully"})
}

// ChatStream SSE 流式聊天接口
func (s *AssistantService) ChatStream(c *gin.Context) {
	var req struct {
		Message      string `json:"message" binding:"required"`
		AssistantID  string `json:"assistant_id"`
		KnowledgeBaseID string `json:"knowledge_base_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		userID = "default-user"
	}

	// 创建 SSE 客户端
	sessionID := uuid.New().String()
	client := &sse.Client{
		ID:       sessionID,
		Channel:  make(chan sse.Event, 100),
		Resource: "chat:" + sessionID,
	}

	// 在 goroutine 中调用 AI 生成响应
	go func() {
		defer close(client.Channel)

		// TODO: 实现 AI 流式响应逻辑
		// 这里需要调用 LLM API 获取流式响应,并逐个 token 发送

		// 示例: 模拟流式响应
		response := "这是一个示例回答。AI 流式响应需要集成 LLM API。"
		words := strings.Fields(response)

		for i, word := range words {
			select {
			case <-c.Request.Context().Done():
				return
			default:
				client.Channel <- sse.Event{
					Type: "token",
					Data: map[string]interface{}{
						"content": word + " ",
						"index":   i,
					},
				}
				time.Sleep(100 * time.Millisecond)
			}
		}

		// 发送完成事件
		client.Channel <- sse.Event{
			Type: "done",
			Data: map[string]interface{}{
				"message": "Response completed",
			},
		}
	}()

	// 开始 SSE 流式传输
	sse.StreamResponse(c, client, s.sseHub, 30*time.Second)
}
