package service

import (
	"errors"
	"math"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/response"
	"go.uber.org/zap"
)

// KnowledgeBaseService 知识库 HTTP 服务
type KnowledgeBaseService struct {
	kbUseCase         *biz.KnowledgeBaseUseCase
	aiProviderUseCase *biz.AIProviderUseCase
	logger            *logger.Logger
}

// NewKnowledgeBaseService 创建知识库服务
func NewKnowledgeBaseService(
	kbUseCase *biz.KnowledgeBaseUseCase,
	aiProviderUseCase *biz.AIProviderUseCase,
	logger *logger.Logger,
) *KnowledgeBaseService {
	return &KnowledgeBaseService{
		kbUseCase:         kbUseCase,
		aiProviderUseCase: aiProviderUseCase,
		logger:            logger,
	}
}

// CreateKnowledgeBase 创建知识库
func (s *KnowledgeBaseService) CreateKnowledgeBase(c *gin.Context) {
	var req CreateKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	kb, err := s.kbUseCase.CreateKnowledgeBase(c.Request.Context(), userID, &biz.CreateKnowledgeBaseRequest{
		Name:             req.Name,
		EmbeddingModelID: req.EmbeddingModelID,
		RerankModelID:    req.RerankModelID,
		ChunkSize:        req.ChunkSize,
		ChunkOverlap:     req.ChunkOverlap,
		ChunkStrategy:    req.ChunkStrategy,
		Threshold:        req.Threshold,
		TopK:             req.TopK,
		EnableHybridSearch: req.EnableHybridSearch,
	})

	if err != nil {
		s.handleError(c, err)
		return
	}

	response.Created(c, toKnowledgeBaseResponse(kb, userID))
}

// ListKnowledgeBases 获取知识库列表
func (s *KnowledgeBaseService) ListKnowledgeBases(c *gin.Context) {
	var req ListKnowledgeBasesRequest
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
	if userID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	bizReq := &biz.ListKnowledgeBasesRequest{
		UserID:   userID,
		Keyword:  req.Keyword,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	kbs, total, err := s.kbUseCase.ListKnowledgeBases(c.Request.Context(), userID, bizReq)
	if err != nil {
		s.handleError(c, err)
		return
	}

	items := make([]*KnowledgeBaseResponse, len(kbs))
	for i, kb := range kbs {
		items[i] = toKnowledgeBaseResponse(kb, userID)
	}

	totalPage := int(math.Ceil(float64(total) / float64(req.PageSize)))

	resp := &ListKnowledgeBasesResponse{
		Items: items,
		Pagination: &PaginationResponse{
			Page:      req.Page,
			PageSize:  req.PageSize,
			Total:     total,
			TotalPage: totalPage,
		},
	}

	response.Success(c, resp)
}

// GetKnowledgeBase 获取知识库详情
func (s *KnowledgeBaseService) GetKnowledgeBase(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	kb, err := s.kbUseCase.GetKnowledgeBase(c.Request.Context(), id, userID)
	if err != nil {
		s.handleError(c, err)
		return
	}

	response.Success(c, toKnowledgeBaseResponse(kb, userID))
}

// UpdateKnowledgeBase 更新知识库
func (s *KnowledgeBaseService) UpdateKnowledgeBase(c *gin.Context) {
	id := c.Param("id")
	var req UpdateKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	kb, err := s.kbUseCase.UpdateKnowledgeBase(c.Request.Context(), id, userID, &biz.UpdateKnowledgeBaseRequest{
		Name:               req.Name,
		Threshold:          req.Threshold,
		TopK:               req.TopK,
		EnableHybridSearch: req.EnableHybridSearch,
	})

	if err != nil {
		s.handleError(c, err)
		return
	}

	response.Success(c, toKnowledgeBaseResponse(kb, userID))
}

// DeleteKnowledgeBase 删除知识库
func (s *KnowledgeBaseService) DeleteKnowledgeBase(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "unauthorized")
		return
	}

	err := s.kbUseCase.DeleteKnowledgeBase(c.Request.Context(), id, userID)
	if err != nil {
		s.handleError(c, err)
		return
	}

	response.Success(c, struct{}{})
}

// handleError 处理错误
func (s *KnowledgeBaseService) handleError(c *gin.Context, err error) {
	s.logger.Error("Knowledge base operation failed", zap.Error(err))

	switch {
	case errors.Is(err, biz.ErrKnowledgeBaseNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, biz.ErrKnowledgeBaseNameRequired),
		errors.Is(err, biz.ErrKnowledgeBaseInvalidChunkSize),
		errors.Is(err, biz.ErrKnowledgeBaseInvalidOverlap):
		response.BadRequest(c, err.Error())
	case errors.Is(err, biz.ErrUnauthorized):
		response.Forbidden(c, err.Error())
	case errors.Is(err, biz.ErrCannotEditOfficialResource),
		errors.Is(err, biz.ErrCannotDeleteOfficialResource):
		response.Forbidden(c, err.Error())
	case errors.Is(err, biz.ErrAIProviderNotFound):
		response.BadRequest(c, err.Error())
	default:
		response.InternalError(c, "internal server error")
	}
}

// toKnowledgeBaseResponse 转换为响应对象
func toKnowledgeBaseResponse(kb *biz.KnowledgeBase, currentUserID string) *KnowledgeBaseResponse {
	// 官方知识库：仅返回 ID 和名称
	if kb.IsOfficial() {
		return &KnowledgeBaseResponse{
			ID:            kb.ID,
			Name:          kb.Name,
			IsOfficial:    true,
			DocumentCount: kb.DocumentCount,
		}
	}

	// 用户知识库：返回完整信息
	createdAt := kb.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	updatedAt := kb.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")

	return &KnowledgeBaseResponse{
		ID:               kb.ID,
		Name:             kb.Name,
		IsOfficial:       false,
		DocumentCount:    kb.DocumentCount,
		OwnerID:          &kb.OwnerID,
		EmbeddingModelID: &kb.EmbeddingModelID,
		RerankModelID:    kb.RerankModelID,
		ChunkSize:        &kb.ChunkSize,
		ChunkOverlap:     &kb.ChunkOverlap,
		ChunkStrategy:    &kb.ChunkStrategy,
		MilvusCollection: &kb.MilvusCollection,
		Threshold:        &kb.Threshold,
		TopK:             &kb.TopK,
		EnableHybridSearch: &kb.EnableHybridSearch,
		CreatedAt:        &createdAt,
		UpdatedAt:        &updatedAt,
	}
}
