package service

import (
	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/response"
	"go.uber.org/zap"
)

// DocumentProviderService 文档处理服务商 HTTP 服务（只读）
type DocumentProviderService struct {
	uc     *biz.DocumentProviderUseCase
	logger *logger.Logger
}

// NewDocumentProviderService 创建文档处理服务商服务
func NewDocumentProviderService(uc *biz.DocumentProviderUseCase, logger *logger.Logger) *DocumentProviderService {
	return &DocumentProviderService{
		uc:     uc,
		logger: logger,
	}
}

// ListDocumentProviders 获取文档处理服务商列表（系统预设，只读）
func (s *DocumentProviderService) ListDocumentProviders(c *gin.Context) {
	providers, err := s.uc.ListDocumentProviders(c.Request.Context())
	if err != nil {
		s.logger.Error("failed to list document providers", zap.Error(err))
		response.InternalError(c, "获取文档处理服务商列表失败")
		return
	}

	items := make([]*DocumentProviderResponse, len(providers))
	for i, provider := range providers {
		items[i] = toDocumentProviderResponse(provider)
	}

	response.Success(c, items)
}

// toDocumentProviderResponse 转换为响应对象（只返回 ID 和名称）
func toDocumentProviderResponse(provider *biz.DocumentProvider) *DocumentProviderResponse {
	return &DocumentProviderResponse{
		ID:           provider.ID,
		ProviderName: provider.ProviderName,
	}
}

// DocumentProviderResponse 文档处理服务商响应
type DocumentProviderResponse struct {
	ID           string `json:"id"`
	ProviderName string `json:"provider_name"`
}
