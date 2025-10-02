package service

import (
	"errors"
	"math"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/agent/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/response"
	"go.uber.org/zap"
)

// AgentService Agent HTTP 服务
type AgentService struct {
	uc     *biz.AgentUseCase
	logger *logger.Logger
}

// NewAgentService 创建 Agent 服务
func NewAgentService(uc *biz.AgentUseCase, logger *logger.Logger) *AgentService {
	return &AgentService{
		uc:     uc,
		logger: logger,
	}
}

// CreateAgent 创建智能体
func (s *AgentService) CreateAgent(c *gin.Context) {
	var req CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	agent, err := s.uc.CreateAgent(
		c.Request.Context(),
		userID,
		req.Name,
		req.Emoji,
		req.Prompt,
		req.KnowledgeBaseIDs,
		req.Tags,
	)
	if err != nil {
		s.handleError(c, err)
		return
	}

	response.Created(c, toAgentResponse(agent))
}

// GetAgent 获取智能体详情
func (s *AgentService) GetAgent(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	agent, err := s.uc.GetAgent(c.Request.Context(), id, userID)
	if err != nil {
		s.handleError(c, err)
		return
	}

	response.Success(c, toAgentResponse(agent))
}

// ListAgents 获取智能体列表
func (s *AgentService) ListAgents(c *gin.Context) {
	var req ListAgentsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	userID := c.GetString("user_id")

	bizReq := &biz.ListAgentsRequest{
		UserID:    userID,
		Page:      req.Page,
		PageSize:  req.PageSize,
		IsEnabled: req.IsEnabled,
		Tags:      req.Tags,
		Keyword:   req.Keyword,
	}

	agents, total, err := s.uc.ListAgents(c.Request.Context(), bizReq)
	if err != nil {
		s.logger.Error("failed to list agents", zap.Error(err))
		response.InternalError(c, "failed to list agents")
		return
	}

	response.Success(c, toListAgentsResponse(agents, total, req.Page, req.PageSize))
}

// UpdateAgent 更新智能体
func (s *AgentService) UpdateAgent(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	var req UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	agent, err := s.uc.UpdateAgent(
		c.Request.Context(),
		id,
		userID,
		req.Name,
		req.Emoji,
		req.Prompt,
		req.KnowledgeBaseIDs,
		req.Tags,
	)
	if err != nil {
		s.handleError(c, err)
		return
	}

	response.Success(c, toAgentResponse(agent))
}

// DeleteAgent 删除智能体
func (s *AgentService) DeleteAgent(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	err := s.uc.DeleteAgent(c.Request.Context(), id, userID)
	if err != nil {
		s.handleError(c, err)
		return
	}

	response.SuccessWithMessage(c, "agent deleted successfully", nil)
}

// EnableAgent 启用智能体
func (s *AgentService) EnableAgent(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	err := s.uc.EnableAgent(c.Request.Context(), id, userID)
	if err != nil {
		s.handleError(c, err)
		return
	}

	response.Success(c, nil)
}

// DisableAgent 禁用智能体
func (s *AgentService) DisableAgent(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	err := s.uc.DisableAgent(c.Request.Context(), id, userID)
	if err != nil {
		s.handleError(c, err)
		return
	}

	response.Success(c, nil)
}

// handleError 统一错误处理
func (s *AgentService) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, biz.ErrAgentNameRequired):
		response.BadRequest(c, "agent name is required")
	case errors.Is(err, biz.ErrAgentPromptRequired):
		response.BadRequest(c, "agent prompt is required")
	case errors.Is(err, biz.ErrAgentPromptTooShort):
		response.BadRequest(c, "agent prompt must be at least 10 characters")
	case errors.Is(err, biz.ErrAgentNotFound):
		response.NotFound(c, "agent not found")
	case errors.Is(err, biz.ErrAgentUnauthorized):
		response.Forbidden(c, "unauthorized to access this agent")
	case errors.Is(err, biz.ErrAgentTagsInvalid):
		response.BadRequest(c, "agent tags invalid")
	case errors.Is(err, biz.ErrAgentKnowledgeBaseInvalid):
		response.BadRequest(c, "knowledge base id invalid")
	default:
		s.logger.Error("internal error", zap.Error(err))
		response.InternalError(c, "internal server error")
	}
}

// toAgentResponse 转换为响应对象
func toAgentResponse(agent *biz.Agent) *AgentResponse {
	// 官方智能体：仅返回公开信息
	if agent.Type == "official" {
		return &AgentResponse{
			ID:               agent.ID,
			Name:             agent.Name,
			Emoji:            agent.Emoji,
			Tags:             agent.Tags,
			KnowledgeBaseIDs: agent.KnowledgeBaseIDs,
			IsOfficial:       true,
			IsEnabled:        agent.IsEnabled,
		}
	}

	// 用户智能体：返回完整信息
	return &AgentResponse{
		ID:               agent.ID,
		Name:             agent.Name,
		Emoji:            agent.Emoji,
		Tags:             agent.Tags,
		KnowledgeBaseIDs: agent.KnowledgeBaseIDs,
		IsOfficial:       false,
		IsEnabled:        agent.IsEnabled,
		OwnerID:          &agent.OwnerID,
		Prompt:           &agent.Prompt,
		Type:             &agent.Type,
		CreatedAt:        &agent.CreatedAt,
		UpdatedAt:        &agent.UpdatedAt,
	}
}

// toListAgentsResponse 转换列表响应
func toListAgentsResponse(agents []*biz.Agent, total int64, page int, pageSize int) *ListAgentsResponse {
	items := make([]*AgentResponse, len(agents))
	for i, agent := range agents {
		items[i] = toAgentResponse(agent)
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &ListAgentsResponse{
		Items: items,
		Pagination: &PaginationResponse{
			Page:       page,
			PageSize:   pageSize,
			Total:      int(total),
			TotalPages: totalPages,
		},
	}
}
