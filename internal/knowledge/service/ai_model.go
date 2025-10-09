package service

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/response"
	"go.uber.org/zap"
)

// AIModelService AI 模型服务
type AIModelService struct {
	modelUseCase *biz.AIModelUseCase
	syncUseCase  *biz.ModelSyncUseCase
	log          *zap.Logger
}

// NewAIModelService 创建 AI 模型服务
func NewAIModelService(
	modelUseCase *biz.AIModelUseCase,
	syncUseCase *biz.ModelSyncUseCase,
	log *zap.Logger,
) *AIModelService {
	return &AIModelService{
		modelUseCase: modelUseCase,
		syncUseCase:  syncUseCase,
		log:          log,
	}
}

// GetModelByID 获取模型详情
func (s *AIModelService) GetModelByID(ctx context.Context, req *GetModelByIDRequest) (*AIModelResponse, error) {
	model, err := s.modelUseCase.GetAIModelByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return toAIModelResponse(model), nil
}

// ListModelsByProvider 获取服务商的模型列表
func (s *AIModelService) ListModelsByProvider(ctx context.Context, req *ListModelsByProviderRequest) (*ListModelsResponse, error) {
	models, err := s.modelUseCase.ListAIModelsByProviderID(ctx, req.ProviderID)
	if err != nil {
		return nil, err
	}

	items := make([]*AIModelResponse, len(models))
	for i, m := range models {
		items[i] = toAIModelResponse(m)
	}

	return &ListModelsResponse{
		Items: items,
		Total: len(items),
	}, nil
}

// ListModelsByCapability 根据能力类型获取模型列表
func (s *AIModelService) ListModelsByCapability(ctx context.Context, req *ListModelsByCapabilityRequest) (*ListModelsResponse, error) {
	models, err := s.modelUseCase.ListAIModelsByCapabilityType(ctx, req.CapabilityType)
	if err != nil {
		return nil, err
	}

	items := make([]*AIModelResponse, len(models))
	for i, m := range models {
		items[i] = toAIModelResponse(m)
	}

	return &ListModelsResponse{
		Items: items,
		Total: len(items),
	}, nil
}

// SyncProviderModels 同步服务商模型
func (s *AIModelService) SyncProviderModels(ctx context.Context, req *SyncProviderModelsRequest) (*SyncResultResponse, error) {
	syncReq := &biz.ModelSyncRequest{
		ProviderID: req.ProviderID,
		SyncedBy:   req.SyncedBy,
		SyncType:   "manual",
	}

	result, err := s.syncUseCase.SyncProviderModels(ctx, syncReq)
	if err != nil {
		return nil, err
	}

	return &SyncResultResponse{
		NewModelsCount:        len(result.NewModels),
		DeprecatedModelsCount: len(result.DeprecatedModels),
		UpdatedModelsCount:    len(result.UpdatedModels),
		ErrorCount:            len(result.Errors),
		NewModels:             extractModelNames(result.NewModels),
		DeprecatedModels:      extractModelNames(result.DeprecatedModels),
		UpdatedModels:         extractModelNames(result.UpdatedModels),
	}, nil
}

// GetSyncHistory 获取同步历史
func (s *AIModelService) GetSyncHistory(ctx context.Context, req *GetSyncHistoryRequest) (*SyncHistoryResponse, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	logs, err := s.syncUseCase.GetSyncHistory(ctx, req.ProviderID, limit)
	if err != nil {
		return nil, err
	}

	items := make([]*SyncLogResponse, len(logs))
	for i, log := range logs {
		items[i] = toSyncLogResponse(log)
	}

	return &SyncHistoryResponse{
		Items: items,
		Total: len(items),
	}, nil
}

// ListAllModels 获取所有启用的AI模型
func (s *AIModelService) ListAllModels(ctx context.Context) (*ListModelsResponse, error) {
	models, err := s.modelUseCase.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*AIModelResponse, len(models))
	for i, m := range models {
		items[i] = toAIModelResponse(m)
	}

	return &ListModelsResponse{
		Items: items,
		Total: len(items),
	}, nil
}

// Request/Response Types

type GetModelByIDRequest struct {
	ID string `json:"id" binding:"required"`
}

type ListModelsByProviderRequest struct {
	ProviderID string `json:"provider_id" binding:"required"`
}

type ListModelsByCapabilityRequest struct {
	CapabilityType string `json:"capability_type" binding:"required"`
}

type SyncProviderModelsRequest struct {
	ProviderID string `json:"provider_id" binding:"required"`
	SyncedBy   string `json:"synced_by"`
}

type GetSyncHistoryRequest struct {
	ProviderID string `json:"provider_id" binding:"required"`
	Limit      int    `json:"limit"`
}

type AIModelResponse struct {
	ID                      string    `json:"id"`
	ProviderID              string    `json:"provider_id"`
	ModelName               string    `json:"model_name"`
	DisplayName             string    `json:"display_name,omitempty"`
	MaxTokens               *int      `json:"max_tokens,omitempty"`
	IsEnabled               bool      `json:"is_enabled"`
	LastVerifiedAt          *time.Time `json:"last_verified_at,omitempty"`
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

type ListModelsResponse struct {
	Items []*AIModelResponse `json:"items"`
	Total int                `json:"total"`
}

type SyncResultResponse struct {
	// 主字段名（保持向后兼容）
	NewModelsCount        int      `json:"new_models_count"`
	DeprecatedModelsCount int      `json:"deprecated_models_count"`
	UpdatedModelsCount    int      `json:"updated_models_count"`
	ErrorCount            int      `json:"error_count"`
	NewModels             []string `json:"new_models,omitempty"`
	DeprecatedModels      []string `json:"deprecated_models,omitempty"`
	UpdatedModels         []string `json:"updated_models,omitempty"`

	// 前端兼容字段（别名）
	ModelsAdded   int      `json:"models_added"`
	ModelsUpdated int      `json:"models_updated"`
	ModelsRemoved int      `json:"models_removed"`
	TotalModels   int      `json:"total_models"`
}

type SyncLogResponse struct {
	ID                    string    `json:"id"`
	ProviderID            string    `json:"provider_id"`
	SyncType              string    `json:"sync_type"`
	NewModelsCount        int       `json:"new_models_count"`
	DeprecatedModelsCount int       `json:"deprecated_models_count"`
	UpdatedModelsCount    int       `json:"updated_models_count"`
	ErrorCount            int       `json:"error_count"`
	NewModels             []string  `json:"new_models,omitempty"`
	DeprecatedModels      []string  `json:"deprecated_models,omitempty"`
	UpdatedModels         []string  `json:"updated_models,omitempty"`
	ErrorMessage          string    `json:"error_message,omitempty"`
	SyncedBy              string    `json:"synced_by"`
	SyncedAt              time.Time `json:"synced_at"`
}

type SyncHistoryResponse struct {
	Items []*SyncLogResponse `json:"items"`
	Total int                `json:"total"`
}

// Helper functions

func toAIModelResponse(model *biz.AIModel) *AIModelResponse {
	return &AIModelResponse{
		ID:                      model.ID,
		ProviderID:              model.ProviderID,
		ModelName:               model.ModelName,
		DisplayName:             model.DisplayName,
		MaxTokens:               model.MaxTokens,
		IsEnabled:               model.IsEnabled,
		LastVerifiedAt:          model.LastVerifiedAt,
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

func toSyncLogResponse(log *biz.ModelSyncLog) *SyncLogResponse {
	return &SyncLogResponse{
		ID:                    log.ID,
		ProviderID:            log.ProviderID,
		SyncType:              log.SyncType,
		NewModelsCount:        log.NewModelsCount,
		DeprecatedModelsCount: log.DeprecatedModelsCount,
		UpdatedModelsCount:    log.UpdatedModelsCount,
		ErrorCount:            log.ErrorCount,
		NewModels:             log.NewModels,
		DeprecatedModels:      log.DeprecatedModels,
		UpdatedModels:         log.UpdatedModels,
		ErrorMessage:          log.ErrorMessage,
		SyncedBy:              log.SyncedBy,
		SyncedAt:              log.SyncedAt,
	}
}

func extractModelNames(models []*biz.AIModel) []string {
	if len(models) == 0 {
		return []string{}
	}
	names := make([]string, len(models))
	for i, m := range models {
		names[i] = m.ModelName
	}
	return names
}

// HTTP Handlers

// HandleGetModelByID Gin handler
func (s *AIModelService) HandleGetModelByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "model ID is required")
		return
	}

	req := &GetModelByIDRequest{ID: id}
	resp, err := s.GetModelByID(c.Request.Context(), req)
	if err != nil {
		s.log.Error("failed to get model", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, resp)
}

// HandleListModelsByProvider Gin handler
func (s *AIModelService) HandleListModelsByProvider(c *gin.Context) {
	providerID := c.Param("provider_id")
	if providerID == "" {
		response.Error(c, http.StatusBadRequest, "provider ID is required")
		return
	}

	req := &ListModelsByProviderRequest{ProviderID: providerID}
	resp, err := s.ListModelsByProvider(c.Request.Context(), req)
	if err != nil {
		s.log.Error("failed to list models by provider", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, resp)
}

// HandleListModelsByCapability Gin handler
func (s *AIModelService) HandleListModelsByCapability(c *gin.Context) {
	capabilityType := c.Param("type")
	if capabilityType == "" {
		response.Error(c, http.StatusBadRequest, "capability type is required")
		return
	}

	req := &ListModelsByCapabilityRequest{CapabilityType: capabilityType}
	resp, err := s.ListModelsByCapability(c.Request.Context(), req)
	if err != nil {
		s.log.Error("failed to list models by capability", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, resp)
}

// HandleSyncProviderModels Gin handler
func (s *AIModelService) HandleSyncProviderModels(c *gin.Context) {
	providerID := c.Param("provider_id")
	if providerID == "" {
		response.Error(c, http.StatusBadRequest, "provider ID is required")
		return
	}

	// 从 JWT token 中获取用户信息作为 synced_by
	userID, _ := c.Get("user_id")
	syncedBy := "system"
	if userID != nil {
		syncedBy = userID.(string)
	}

	req := &SyncProviderModelsRequest{
		ProviderID: providerID,
		SyncedBy:   syncedBy,
	}
	resp, err := s.SyncProviderModels(c.Request.Context(), req)
	if err != nil {
		s.log.Error("failed to sync provider models", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, resp)
}

// HandleGetSyncHistory Gin handler
func (s *AIModelService) HandleGetSyncHistory(c *gin.Context) {
	providerID := c.Param("provider_id")
	if providerID == "" {
		response.Error(c, http.StatusBadRequest, "provider ID is required")
		return
	}

	var req GetSyncHistoryRequest
	req.ProviderID = providerID
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := s.GetSyncHistory(c.Request.Context(), &req)
	if err != nil {
		s.log.Error("failed to get sync history", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, resp)
}

// HandleListAllModels Gin handler
func (s *AIModelService) HandleListAllModels(c *gin.Context) {
	resp, err := s.ListAllModels(c.Request.Context())
	if err != nil {
		s.log.Error("failed to list all models", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, resp)
}
