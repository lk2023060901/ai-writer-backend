package service

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/queue"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/response"
	"go.uber.org/zap"
)

type DocumentService struct {
	docUseCase *biz.DocumentUseCase
	worker     *queue.Worker
	logger     *zap.Logger
}

func NewDocumentService(
	docUseCase *biz.DocumentUseCase,
	worker *queue.Worker,
	logger *zap.Logger,
) *DocumentService {
	return &DocumentService{
		docUseCase: docUseCase,
		worker:     worker,
		logger:     logger,
	}
}

// UploadDocument 上传文档
func (s *DocumentService) UploadDocument(c *gin.Context) {
	kbID := c.Param("id")
	userID := c.GetString("user_id")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid file")
		return
	}
	defer file.Close()

	// 读取文件内容
	fileData, err := io.ReadAll(file)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to read file")
		return
	}

	// 获取文件类型
	fileName := header.Filename
	fileType := getFileExtension(fileName)

	// 上传文档
	doc, err := s.docUseCase.UploadDocument(c.Request.Context(), kbID, userID, fileName, fileData, fileType)
	if err != nil {
		s.logger.Error("failed to upload document", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 加入处理队列
	err = s.worker.EnqueueDocument(c.Request.Context(), doc.ID)
	if err != nil {
		s.logger.Error("failed to enqueue document", zap.Error(err))
	}

	response.Success(c, toDocumentResponse(doc))
}

// ListDocuments 列出文档
func (s *DocumentService) ListDocuments(c *gin.Context) {
	kbID := c.Param("id")
	
	var req struct {
		Page     int `form:"page" binding:"required,min=1"`
		PageSize int `form:"page_size" binding:"required,min=1,max=100"`
	}
	
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid parameters")
		return
	}

	docs, total, err := s.docUseCase.DocumentRepo.List(c.Request.Context(), kbID, &biz.ListDocumentsRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]DocumentResponse, len(docs))
	for i, doc := range docs {
		items[i] = *toDocumentResponse(doc)
	}

	response.Success(c, map[string]interface{}{
		"items": items,
		"pagination": map[string]interface{}{
			"page":       req.Page,
			"page_size":  req.PageSize,
			"total":      total,
			"total_page": (total + int64(req.PageSize) - 1) / int64(req.PageSize),
		},
	})
}

// GetDocument 获取文档详情
func (s *DocumentService) GetDocument(c *gin.Context) {
	docID := c.Param("doc_id")

	doc, err := s.docUseCase.DocumentRepo.GetByID(c.Request.Context(), docID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "document not found")
		return
	}

	response.Success(c, toDocumentResponse(doc))
}

// DeleteDocument 删除文档
func (s *DocumentService) DeleteDocument(c *gin.Context) {
	docID := c.Param("doc_id")
	userID := c.GetString("user_id")

	err := s.docUseCase.DeleteDocument(c.Request.Context(), docID, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// ReprocessDocument 重新处理文档
func (s *DocumentService) ReprocessDocument(c *gin.Context) {
	docID := c.Param("doc_id")
	_ = c.GetString("user_id") // userID

	// 加入处理队列
	err := s.worker.EnqueueDocument(c.Request.Context(), docID)
	if err != nil {
		s.logger.Error("failed to enqueue document for reprocessing", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "failed to enqueue document")
		return
	}

	response.Success(c, map[string]string{"message": "document queued for reprocessing"})
}

// SearchDocuments 向量搜索
func (s *DocumentService) SearchDocuments(c *gin.Context) {
	kbID := c.Param("id")
	userID := c.GetString("user_id")

	var req struct {
		Query string `json:"query" binding:"required"`
		TopK  int    `json:"top_k" binding:"required,min=1,max=20"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid parameters")
		return
	}

	results, err := s.docUseCase.SearchDocuments(c.Request.Context(), kbID, userID, req.Query, req.TopK)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, map[string]interface{}{
		"results": toSearchResults(results),
	})
}

func getFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i+1:]
		}
	}
	return ""
}
