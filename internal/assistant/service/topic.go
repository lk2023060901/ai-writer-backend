package service

import (
	"net/http"

	"github.com/lk2023060901/ai-writer-backend/internal/assistant/biz"

	"github.com/gin-gonic/gin"
)

// TopicService handles HTTP requests for topic operations
type TopicService struct {
	useCase *biz.TopicUseCase
}

// NewTopicService creates a new topic service
func NewTopicService(useCase *biz.TopicUseCase) *TopicService {
	return &TopicService{
		useCase: useCase,
	}
}

// RegisterRoutes registers topic routes
func (s *TopicService) RegisterRoutes(r *gin.RouterGroup) {
	// Topics are nested under assistants
	r.POST("/assistants/:assistant_id/topics", s.CreateTopic)
	r.GET("/assistants/:assistant_id/topics", s.ListTopics)
	r.GET("/assistants/:assistant_id/topics/:topic_id", s.GetTopic)
	r.PUT("/assistants/:assistant_id/topics/:topic_id", s.UpdateTopic)
	r.DELETE("/assistants/:assistant_id/topics/:topic_id", s.DeleteTopic)
	r.DELETE("/assistants/:assistant_id/topics", s.DeleteAllTopics)
}

// CreateTopic creates a new topic
// @Summary Create topic
// @Tags topics
// @Accept json
// @Produce json
// @Param assistant_id path string true "Assistant ID"
// @Param request body map[string]string false "Topic name"
// @Success 200 {object} types.Topic
// @Router /api/v1/assistants/{assistant_id}/topics [post]
func (s *TopicService) CreateTopic(c *gin.Context) {
	assistantID := c.Param("assistant_id")

	var req struct {
		Name string `json:"name"`
	}
	c.ShouldBindJSON(&req)

	topic, err := s.useCase.CreateTopic(c.Request.Context(), assistantID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, topic)
}

// GetTopic retrieves a topic by ID
// @Summary Get topic
// @Tags topics
// @Produce json
// @Param assistant_id path string true "Assistant ID"
// @Param topic_id path string true "Topic ID"
// @Success 200 {object} types.Topic
// @Router /api/v1/assistants/{assistant_id}/topics/{topic_id} [get]
func (s *TopicService) GetTopic(c *gin.Context) {
	topicID := c.Param("topic_id")

	topic, err := s.useCase.GetTopic(c.Request.Context(), topicID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, topic)
}

// ListTopics lists all topics for an assistant
// @Summary List topics
// @Tags topics
// @Produce json
// @Param assistant_id path string true "Assistant ID"
// @Success 200 {array} types.Topic
// @Router /api/v1/assistants/{assistant_id}/topics [get]
func (s *TopicService) ListTopics(c *gin.Context) {
	assistantID := c.Param("assistant_id")

	topics, err := s.useCase.ListTopics(c.Request.Context(), assistantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, topics)
}

// UpdateTopic updates a topic
// @Summary Update topic
// @Tags topics
// @Accept json
// @Produce json
// @Param assistant_id path string true "Assistant ID"
// @Param topic_id path string true "Topic ID"
// @Param request body map[string]string true "Topic name"
// @Success 200 {object} types.Topic
// @Router /api/v1/assistants/{assistant_id}/topics/{topic_id} [put]
func (s *TopicService) UpdateTopic(c *gin.Context) {
	topicID := c.Param("topic_id")

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	topic, err := s.useCase.UpdateTopic(c.Request.Context(), topicID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, topic)
}

// DeleteTopic deletes a topic
// @Summary Delete topic
// @Tags topics
// @Param assistant_id path string true "Assistant ID"
// @Param topic_id path string true "Topic ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/assistants/{assistant_id}/topics/{topic_id} [delete]
func (s *TopicService) DeleteTopic(c *gin.Context) {
	topicID := c.Param("topic_id")

	if err := s.useCase.DeleteTopic(c.Request.Context(), topicID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Topic deleted successfully"})
}

// DeleteAllTopics deletes all topics for an assistant
// @Summary Delete all topics
// @Tags topics
// @Param assistant_id path string true "Assistant ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/assistants/{assistant_id}/topics [delete]
func (s *TopicService) DeleteAllTopics(c *gin.Context) {
	assistantID := c.Param("assistant_id")

	if err := s.useCase.DeleteAllTopics(c.Request.Context(), assistantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All topics deleted successfully"})
}
