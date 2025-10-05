package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

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

// ImportFromFile 从上传的 JSON 文件导入智能体
func (s *AgentService) ImportFromFile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "未授权")
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请上传 JSON 文件")
		return
	}

	// 检查文件大小（最大 1MB）
	if file.Size > 1*1024*1024 {
		response.BadRequest(c, "文件大小不能超过 1MB")
		return
	}

	// 打开文件
	f, err := file.Open()
	if err != nil {
		s.logger.Error("failed to open file", zap.Error(err))
		response.InternalError(c, "读取文件失败")
		return
	}
	defer f.Close()

	// 读取文件内容
	data, err := io.ReadAll(f)
	if err != nil {
		s.logger.Error("failed to read file", zap.Error(err))
		response.InternalError(c, "读取文件失败")
		return
	}

	// 解析 JSON
	var items []AgentImportItem
	if err := json.Unmarshal(data, &items); err != nil {
		response.BadRequest(c, "JSON 格式错误")
		return
	}

	s.processBatchImport(c, userID, items)
}

// ImportFromURL 从 URL 导入智能体
func (s *AgentService) ImportFromURL(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "未授权")
		return
	}

	var req ImportFromURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 创建 HTTP 客户端（10秒超时）
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 获取 URL 内容
	resp, err := client.Get(req.URL)
	if err != nil {
		s.logger.Error("failed to fetch URL", zap.Error(err), zap.String("url", req.URL))
		response.BadRequest(c, "无法访问该 URL")
		return
	}
	defer resp.Body.Close()

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		response.BadRequest(c, fmt.Sprintf("URL 返回错误状态码: %d", resp.StatusCode))
		return
	}

	// 检查 Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && contentType != "application/json" && contentType != "text/plain" {
		s.logger.Warn("unexpected content type", zap.String("content_type", contentType))
	}

	// 限制响应大小（最大 1MB）
	limitedReader := io.LimitReader(resp.Body, 1*1024*1024)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		s.logger.Error("failed to read response", zap.Error(err))
		response.InternalError(c, "读取 URL 内容失败")
		return
	}

	// 解析 JSON
	var items []AgentImportItem
	if err := json.Unmarshal(data, &items); err != nil {
		response.BadRequest(c, "URL 返回的 JSON 格式错误")
		return
	}

	s.processBatchImport(c, userID, items)
}

// processBatchImport 处理批量导入逻辑
func (s *AgentService) processBatchImport(c *gin.Context, userID string, items []AgentImportItem) {
	if len(items) == 0 {
		response.BadRequest(c, "导入列表为空")
		return
	}

	if len(items) > 100 {
		response.BadRequest(c, "单次最多导入 100 个智能体")
		return
	}

	// 转换为 UseCase 需要的格式
	bizItems := make([]struct {
		Name   string
		Emoji  string
		Prompt string
		Tags   []string
	}, len(items))

	for i, item := range items {
		bizItems[i].Name = item.Name
		bizItems[i].Emoji = item.Emoji
		bizItems[i].Prompt = item.Prompt
		bizItems[i].Tags = item.Tags
	}

	// 批量创建
	agents, errs := s.uc.BatchCreateAgents(c.Request.Context(), userID, bizItems)

	// 构建响应
	successCount := len(agents)
	failCount := len(errs)
	errorMsgs := make([]string, 0, failCount)

	for _, err := range errs {
		errorMsgs = append(errorMsgs, err.Error())
	}

	// 返回结果
	result := gin.H{
		"success_count": successCount,
		"fail_count":    failCount,
	}

	if failCount > 0 {
		result["errors"] = errorMsgs
	}

	if successCount > 0 {
		agentResponses := make([]*AgentResponse, len(agents))
		for i, agent := range agents {
			agentResponses[i] = toAgentResponse(agent)
		}
		result["agents"] = agentResponses
	}

	response.Success(c, result)
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
