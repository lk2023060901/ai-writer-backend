package service

import (
	"net/http"

	"github.com/lk2023060901/ai-writer-backend/internal/agent/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/agent/types"

	"github.com/gin-gonic/gin"
)

// AgentService handles HTTP requests for agent operations
type AgentService struct {
	useCase *biz.AgentUseCase
}

// NewAgentService creates a new agent service
func NewAgentService(useCase *biz.AgentUseCase) *AgentService {
	return &AgentService{
		useCase: useCase,
	}
}

// RegisterRoutes registers agent routes
func (s *AgentService) RegisterRoutes(r *gin.RouterGroup) {
	agents := r.Group("/agents")
	{
		agents.POST("", s.CreateAgent)
		agents.GET("", s.ListAgents)
		agents.GET("/:id", s.GetAgent)
		agents.PUT("/:id", s.UpdateAgent)
		agents.DELETE("/:id", s.DeleteAgent)
		agents.GET("/groups", s.ListGroups)
		agents.GET("/group/:name", s.ListByGroup)
	}
}

// CreateAgent creates a new agent
// @Summary Create agent
// @Tags agents
// @Accept json
// @Produce json
// @Param request body biz.CreateAgentRequest true "Create Agent Request"
// @Success 200 {object} types.Agent
// @Router /api/v1/agents [post]
func (s *AgentService) CreateAgent(c *gin.Context) {
	var req biz.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agent, err := s.useCase.CreateAgent(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// GetAgent retrieves an agent by ID
// @Summary Get agent
// @Tags agents
// @Produce json
// @Param id path string true "Agent ID"
// @Success 200 {object} types.Agent
// @Router /api/v1/agents/{id} [get]
func (s *AgentService) GetAgent(c *gin.Context) {
	id := c.Param("id")

	agent, err := s.useCase.GetAgent(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// ListAgents lists all agents with optional filtering
// @Summary List agents
// @Tags agents
// @Produce json
// @Param group query string false "Filter by group"
// @Param is_builtin query boolean false "Filter by builtin status"
// @Param keyword query string false "Search keyword"
// @Success 200 {array} types.Agent
// @Router /api/v1/agents [get]
func (s *AgentService) ListAgents(c *gin.Context) {
	var filter types.AgentFilter

	if group := c.Query("group"); group != "" {
		filter.Group = group
	}
	if keyword := c.Query("keyword"); keyword != "" {
		filter.Keyword = keyword
	}
	if isBuiltin := c.Query("is_builtin"); isBuiltin != "" {
		builtin := isBuiltin == "true"
		filter.IsBuiltin = &builtin
	}

	agents, err := s.useCase.ListAgents(c.Request.Context(), &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agents)
}

// UpdateAgent updates an existing agent
// @Summary Update agent
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "Agent ID"
// @Param request body biz.UpdateAgentRequest true "Update Agent Request"
// @Success 200 {object} types.Agent
// @Router /api/v1/agents/{id} [put]
func (s *AgentService) UpdateAgent(c *gin.Context) {
	id := c.Param("id")

	var req biz.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agent, err := s.useCase.UpdateAgent(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// DeleteAgent deletes an agent
// @Summary Delete agent
// @Tags agents
// @Param id path string true "Agent ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/agents/{id} [delete]
func (s *AgentService) DeleteAgent(c *gin.Context) {
	id := c.Param("id")

	if err := s.useCase.DeleteAgent(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent deleted successfully"})
}

// ListGroups lists all agent groups
// @Summary List agent groups
// @Tags agents
// @Produce json
// @Success 200 {array} types.AgentGroup
// @Router /api/v1/agents/groups [get]
func (s *AgentService) ListGroups(c *gin.Context) {
	groups, err := s.useCase.ListGroups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, groups)
}

// ListByGroup lists agents in a specific group
// @Summary List agents by group
// @Tags agents
// @Produce json
// @Param name path string true "Group Name"
// @Success 200 {array} types.Agent
// @Router /api/v1/agents/group/{name} [get]
func (s *AgentService) ListByGroup(c *gin.Context) {
	groupName := c.Param("name")

	agents, err := s.useCase.ListByGroup(c.Request.Context(), groupName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agents)
}
