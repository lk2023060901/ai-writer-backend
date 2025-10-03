package service

import (
	"net/http"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/assistant/types"

	"github.com/gin-gonic/gin"
)

// MessageService handles HTTP requests for message operations
type MessageService struct {
	useCase *biz.MessageUseCase
}

// NewMessageService creates a new message service
func NewMessageService(useCase *biz.MessageUseCase) *MessageService {
	return &MessageService{
		useCase: useCase,
	}
}

// RegisterRoutes registers message routes
func (s *MessageService) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/topics/:topic_id/messages", s.CreateMessage)
	r.GET("/topics/:topic_id/messages", s.ListMessages)
	r.GET("/topics/:topic_id/messages/:message_id", s.GetMessage)
	r.DELETE("/topics/:topic_id/messages", s.DeleteMessages)
}

// CreateMessageRequest represents the request to create a message
type CreateMessageRequest struct {
	Role          string               `json:"role" binding:"required,oneof=user assistant"`
	ContentBlocks []types.ContentBlock `json:"content_blocks" binding:"required,min=1"`
	TokenCount    *int                 `json:"token_count,omitempty"`
}

// ListMessagesRequest represents the request to list messages
type ListMessagesRequest struct {
	Limit  int `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset int `form:"offset" binding:"omitempty,min=0"`
}

// ListMessagesResponse represents the response for listing messages
type ListMessagesResponse struct {
	Messages []*types.Message `json:"messages"`
	Total    int64            `json:"total"`
	Limit    int              `json:"limit"`
	Offset   int              `json:"offset"`
}

// CreateMessage creates a new message
// @Summary Create message
// @Tags messages
// @Accept json
// @Produce json
// @Param topic_id path string true "Topic ID"
// @Param request body CreateMessageRequest true "Message content"
// @Success 200 {object} types.Message
// @Router /api/v1/topics/{topic_id}/messages [post]
func (s *MessageService) CreateMessage(c *gin.Context) {
	topicID := c.Param("topic_id")

	var req CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, err := s.useCase.CreateMessage(
		c.Request.Context(),
		topicID,
		req.Role,
		req.ContentBlocks,
		req.TokenCount,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, message)
}

// GetMessage retrieves a message by ID
// @Summary Get message
// @Tags messages
// @Produce json
// @Param topic_id path string true "Topic ID"
// @Param message_id path string true "Message ID"
// @Success 200 {object} types.Message
// @Router /api/v1/topics/{topic_id}/messages/{message_id} [get]
func (s *MessageService) GetMessage(c *gin.Context) {
	messageID := c.Param("message_id")

	message, err := s.useCase.GetMessage(c.Request.Context(), messageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, message)
}

// ListMessages lists all messages in a topic with pagination
// @Summary List messages
// @Tags messages
// @Produce json
// @Param topic_id path string true "Topic ID"
// @Param limit query int false "Limit (default 50, max 100)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {object} ListMessagesResponse
// @Router /api/v1/topics/{topic_id}/messages [get]
func (s *MessageService) ListMessages(c *gin.Context) {
	topicID := c.Param("topic_id")

	var req ListMessagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default limit
	if req.Limit == 0 {
		req.Limit = 50
	}

	messages, total, err := s.useCase.ListMessages(
		c.Request.Context(),
		topicID,
		req.Limit,
		req.Offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListMessagesResponse{
		Messages: messages,
		Total:    total,
		Limit:    req.Limit,
		Offset:   req.Offset,
	})
}

// DeleteMessages deletes all messages in a topic
// @Summary Delete all messages in topic
// @Tags messages
// @Param topic_id path string true "Topic ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/topics/{topic_id}/messages [delete]
func (s *MessageService) DeleteMessages(c *gin.Context) {
	topicID := c.Param("topic_id")

	if err := s.useCase.DeleteMessagesInTopic(c.Request.Context(), topicID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All messages deleted successfully"})
}
