package service

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/response"
	"go.uber.org/zap"
)

// AIProviderService AI服务商配置 HTTP 服务
type AIProviderService struct {
	uc     *biz.AIProviderConfigUseCase
	logger *logger.Logger
}

// NewAIProviderService 创建AI服务商配置服务
func NewAIProviderService(uc *biz.AIProviderConfigUseCase, logger *logger.Logger) *AIProviderService {
	return &AIProviderService{
		uc:     uc,
		logger: logger,
	}
}

// CreateAIProviderConfig 创建AI服务商配置
func (s *AIProviderService) CreateAIProviderConfig(c *gin.Context) {
	var req CreateAIProviderConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	config, err := s.uc.CreateAIProviderConfig(c.Request.Context(), userID, &biz.CreateAIProviderConfigRequest{
		ProviderType:        req.ProviderType,
		ProviderName:        req.ProviderName,
		APIKey:              req.APIKey,
		APIBaseURL:          req.APIBaseURL,
		EmbeddingModel:      req.EmbeddingModel,
		EmbeddingDimensions: req.EmbeddingDimensions,
	})

	if err != nil {
		s.handleError(c, err)
		return
	}

	response.Created(c, toAIProviderConfigResponse(config, userID))
}

// ListAIProviderConfigs 获取AI服务商配置列表
func (s *AIProviderService) ListAIProviderConfigs(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	configs, err := s.uc.ListAIProviderConfigs(c.Request.Context(), userID)
	if err != nil {
		s.handleError(c, err)
		return
	}

	items := make([]*AIProviderConfigResponse, len(configs))
	for i, config := range configs {
		items[i] = toAIProviderConfigResponse(config, userID)
	}

	response.Success(c, items)
}

// GetAIProviderConfig 获取AI服务商配置详情
func (s *AIProviderService) GetAIProviderConfig(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	config, err := s.uc.GetAIProviderConfig(c.Request.Context(), id, userID)
	if err != nil {
		s.handleError(c, err)
		return
	}

	response.Success(c, toAIProviderConfigResponse(config, userID))
}

// UpdateAIProviderConfig 更新AI服务商配置
func (s *AIProviderService) UpdateAIProviderConfig(c *gin.Context) {
	id := c.Param("id")
	var req UpdateAIProviderConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	config, err := s.uc.UpdateAIProviderConfig(c.Request.Context(), id, userID, &biz.UpdateAIProviderConfigRequest{
		ProviderName:        req.ProviderName,
		APIKey:              req.APIKey,
		APIBaseURL:          req.APIBaseURL,
		EmbeddingModel:      req.EmbeddingModel,
		EmbeddingDimensions: req.EmbeddingDimensions,
	})

	if err != nil {
		s.handleError(c, err)
		return
	}

	response.Success(c, toAIProviderConfigResponse(config, userID))
}

// DeleteAIProviderConfig 删除AI服务商配置
func (s *AIProviderService) DeleteAIProviderConfig(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	err := s.uc.DeleteAIProviderConfig(c.Request.Context(), id, userID)
	if err != nil {
		s.handleError(c, err)
		return
	}

	response.Success(c, struct{}{})
}

// handleError 处理错误
func (s *AIProviderService) handleError(c *gin.Context, err error) {
	s.logger.Error("AI Provider config operation failed", zap.Error(err))

	switch {
	case errors.Is(err, biz.ErrAIProviderConfigNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, biz.ErrAIProviderConfigNameRequired),
		errors.Is(err, biz.ErrAIProviderConfigAPIKeyRequired),
		errors.Is(err, biz.ErrAIProviderConfigInvalidProvider):
		response.BadRequest(c, err.Error())
	case errors.Is(err, biz.ErrUnauthorized):
		response.Forbidden(c, err.Error())
	case errors.Is(err, biz.ErrCannotEditOfficialResource),
		errors.Is(err, biz.ErrCannotDeleteOfficialResource):
		response.Forbidden(c, err.Error())
	case errors.Is(err, biz.ErrAIProviderConfigInUse):
		response.BadRequest(c, err.Error())
	default:
		response.InternalError(c, "internal server error")
	}
}

// toAIProviderConfigResponse 转换为响应对象
func toAIProviderConfigResponse(config *biz.AIProviderConfig, currentUserID string) *AIProviderConfigResponse {
	// 官方配置：仅返回 ID 和名称
	if config.IsOfficial() {
		return &AIProviderConfigResponse{
			ID:           config.ID,
			ProviderName: config.ProviderName,
			IsOfficial:   true,
		}
	}

	// 用户配置：返回完整信息（API Key 脱敏）
	createdAt := config.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	updatedAt := config.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	maskedAPIKey := MaskAPIKey(config.APIKey)

	return &AIProviderConfigResponse{
		ID:                  config.ID,
		ProviderName:        config.ProviderName,
		IsOfficial:          false,
		OwnerID:             &config.OwnerID,
		ProviderType:        &config.ProviderType,
		APIKey:              &maskedAPIKey,
		APIBaseURL:          &config.APIBaseURL,
		EmbeddingModel:      &config.EmbeddingModel,
		EmbeddingDimensions: &config.EmbeddingDimensions,
		IsEnabled:           &config.IsEnabled,
		CreatedAt:           &createdAt,
		UpdatedAt:           &updatedAt,
	}
}

// MaskAPIKey API Key 脱敏（保留前4位和后4位）
func MaskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "********"
	}
	return apiKey[:4] + "********" + apiKey[len(apiKey)-4:]
}
