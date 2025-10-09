package service

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/response"
	"go.uber.org/zap"
)

// AIProviderService AI服务商 HTTP 服务（只读）
type AIProviderService struct {
	uc      *biz.AIProviderUseCase
	modelUC *biz.AIModelUseCase
	logger  *logger.Logger
}

// NewAIProviderService 创建AI服务商服务
func NewAIProviderService(uc *biz.AIProviderUseCase, modelUC *biz.AIModelUseCase, logger *logger.Logger) *AIProviderService {
	return &AIProviderService{
		uc:      uc,
		modelUC: modelUC,
		logger:  logger,
	}
}

// ListAIProviders 获取AI服务商列表
func (s *AIProviderService) ListAIProviders(c *gin.Context) {
	providers, err := s.uc.ListAIProviders(c.Request.Context())
	if err != nil {
		s.logger.Error("failed to list AI providers", zap.Error(err))
		response.InternalError(c, "获取AI服务商列表失败")
		return
	}

	items := make([]*AIProviderResponse, len(providers))
	for i, provider := range providers {
		items[i] = toAIProviderResponse(provider)
	}

	response.Success(c, items)
}

// UpdateAIProviderStatus 更新AI服务商启用状态
func (s *AIProviderService) UpdateAIProviderStatus(c *gin.Context) {
	providerID := c.Param("id")
	if providerID == "" {
		response.BadRequest(c, "provider ID is required")
		return
	}

	var req UpdateProviderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	// 更新服务商状态
	if err := s.uc.UpdateProviderStatus(c.Request.Context(), providerID, *req.IsEnabled); err != nil {
		s.logger.Error("failed to update provider status",
			zap.String("provider_id", providerID),
			zap.Bool("is_enabled", *req.IsEnabled),
			zap.Error(err))
		response.InternalError(c, "更新服务商状态失败")
		return
	}

	// 返回更新后的服务商信息
	provider, err := s.uc.GetAIProviderByID(c.Request.Context(), providerID)
	if err != nil {
		s.logger.Error("failed to get updated provider", zap.Error(err))
		response.InternalError(c, "获取更新后的服务商信息失败")
		return
	}

	response.Success(c, toAIProviderResponse(provider))
}

// UpdateAIProvider 更新AI服务商配置（API Key 和 API 地址）
func (s *AIProviderService) UpdateAIProvider(c *gin.Context) {
	providerID := c.Param("id")
	if providerID == "" {
		response.BadRequest(c, "provider ID is required")
		return
	}

	var req UpdateAIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	// 更新服务商配置
	if err := s.uc.UpdateProviderConfig(c.Request.Context(), providerID, req.APIKey, req.APIBaseURL); err != nil {
		s.logger.Error("failed to update provider config",
			zap.String("provider_id", providerID),
			zap.Error(err))
		response.InternalError(c, "更新服务商配置失败")
		return
	}

	// 返回更新后的服务商信息
	provider, err := s.uc.GetAIProviderByID(c.Request.Context(), providerID)
	if err != nil {
		s.logger.Error("failed to get updated provider", zap.Error(err))
		response.InternalError(c, "获取更新后的服务商信息失败")
		return
	}

	response.Success(c, toAIProviderResponse(provider))
}

// ListAllProvidersWithModels 获取所有AI服务商及其模型列表
func (s *AIProviderService) ListAllProvidersWithModels(c *gin.Context) {
	// 获取所有服务商
	providers, err := s.uc.ListAIProviders(c.Request.Context())
	if err != nil {
		s.logger.Error("failed to list AI providers", zap.Error(err))
		response.InternalError(c, "获取AI服务商列表失败")
		return
	}

	// 为每个服务商获取模型列表
	result := make([]*ProviderWithModelsResponse, len(providers))
	for i, provider := range providers {
		models, err := s.modelUC.ListAIModelsByProviderID(c.Request.Context(), provider.ID)
		if err != nil {
			s.logger.Error("failed to list models for provider",
				zap.String("provider_id", provider.ID),
				zap.Error(err))
			// 如果获取模型失败，返回空数组而不是整个请求失败
			models = []*biz.AIModel{}
		}

		// 转换模型列表
		modelResponses := make([]*AIModelSimpleResponse, len(models))
		for j, model := range models {
			modelResponses[j] = toAIModelSimpleResponse(model)
		}

		result[i] = &ProviderWithModelsResponse{
			ID:           provider.ID,
			ProviderType: provider.ProviderType,
			ProviderName: provider.ProviderName,
			APIBaseURL:   provider.APIBaseURL,
			APIKey:       provider.APIKey,
			IsEnabled:    provider.IsEnabled,
			Models:       modelResponses,
		}
	}

	response.Success(c, result)
}

// toAIProviderResponse 转换为响应对象
func toAIProviderResponse(provider *biz.AIProvider) *AIProviderResponse {
	return &AIProviderResponse{
		ID:           provider.ID,
		ProviderType: provider.ProviderType,
		ProviderName: provider.ProviderName,
		APIBaseURL:   provider.APIBaseURL,
		APIKey:       provider.APIKey,
		IsEnabled:    provider.IsEnabled,
	}
}

// toAIModelSimpleResponse 转换为简化的模型响应
func toAIModelSimpleResponse(model *biz.AIModel) *AIModelSimpleResponse {
	return &AIModelSimpleResponse{
		ID:                      model.ID,
		ModelName:               model.ModelName,
		DisplayName:             model.DisplayName,
		MaxTokens:               model.MaxTokens,
		IsEnabled:               model.IsEnabled,
		VerificationStatus:      model.VerificationStatus,
		Capabilities:            model.Capabilities,
		SupportsStream:          model.SupportsStream,
		SupportsVision:          model.SupportsVision,
		SupportsFunctionCalling: model.SupportsFunctionCalling,
		SupportsReasoning:       model.SupportsReasoning,
		SupportsWebSearch:       model.SupportsWebSearch,
		EmbeddingDimensions:     model.EmbeddingDimensions,
		CreatedAt:               model.CreatedAt,
		UpdatedAt:               model.UpdatedAt,
	}
}

// UpdateProviderStatusRequest 更新服务商状态请求
type UpdateProviderStatusRequest struct {
	IsEnabled *bool `json:"is_enabled" binding:"required"`
}

// UpdateAIProviderRequest 更新AI服务商配置请求
type UpdateAIProviderRequest struct {
	APIKey     *string `json:"api_key"`      // API密钥（可选）
	APIBaseURL *string `json:"api_base_url"` // API基础URL（可选）
}

// AIProviderResponse AI服务商响应
type AIProviderResponse struct {
	ID           string `json:"id"`
	ProviderType string `json:"provider_type"`
	ProviderName string `json:"provider_name"`
	APIBaseURL   string `json:"api_base_url"`
	APIKey       string `json:"api_key"`
	IsEnabled    bool   `json:"is_enabled"`
}

// ProviderWithModelsResponse AI服务商及其模型响应
type ProviderWithModelsResponse struct {
	ID           string                   `json:"id"`
	ProviderType string                   `json:"provider_type"`
	ProviderName string                   `json:"provider_name"`
	APIBaseURL   string                   `json:"api_base_url"`
	APIKey       string                   `json:"api_key"`
	IsEnabled    bool                     `json:"is_enabled"`
	Models       []*AIModelSimpleResponse `json:"models"`
}

// AIModelSimpleResponse 简化的AI模型响应
type AIModelSimpleResponse struct {
	ID                      string    `json:"id"`
	ModelName               string    `json:"model_name"`
	DisplayName             string    `json:"display_name,omitempty"`
	MaxTokens               *int      `json:"max_tokens,omitempty"`
	IsEnabled               bool      `json:"is_enabled"`
	VerificationStatus      string    `json:"verification_status"`
	Capabilities            []string  `json:"capabilities"` // ["chat", "embedding", "rerank"]
	SupportsStream          bool      `json:"supports_stream"`
	SupportsVision          bool      `json:"supports_vision"`
	SupportsFunctionCalling bool      `json:"supports_function_calling"`
	SupportsReasoning       bool      `json:"supports_reasoning"`
	SupportsWebSearch       bool      `json:"supports_web_search"`
	EmbeddingDimensions     *int      `json:"embedding_dimensions,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}
