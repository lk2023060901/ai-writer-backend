package service

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/queue"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/response"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/sse"
	"go.uber.org/zap"
)

type DocumentService struct {
	docUseCase *biz.DocumentUseCase
	worker     *queue.Worker
	sseHub     *sse.Hub
	logger     *zap.Logger
}

func NewDocumentService(
	docUseCase *biz.DocumentUseCase,
	worker *queue.Worker,
	sseHub *sse.Hub,
	logger *zap.Logger,
) *DocumentService {
	return &DocumentService{
		docUseCase: docUseCase,
		worker:     worker,
		sseHub:     sseHub,
		logger:     logger,
	}
}

// UploadDocument 上传文档并返回 SSE 流式响应
func (s *DocumentService) UploadDocument(c *gin.Context) {
	kbID := c.Param("id")
	userID := c.GetString("user_id")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file"})
		return
	}
	defer file.Close()

	// 读取文件内容
	fileData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	// 获取文件类型
	fileName := header.Filename
	fileType := getFileExtension(fileName)

	// 上传文档
	doc, err := s.docUseCase.UploadDocument(c.Request.Context(), kbID, userID, fileName, fileData, fileType)
	if err != nil {
		s.logger.Error("failed to upload document", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 创建 SSE 客户端
	client := &sse.Client{
		ID:       uuid.New().String(),
		Channel:  make(chan sse.Event, 10),
		Resource: "doc:" + doc.ID,
	}

	// 发送初始上传成功事件
	go func() {
		time.Sleep(100 * time.Millisecond)
		client.Channel <- sse.Event{
			Type: "uploaded",
			Data: map[string]interface{}{
				"document_id": doc.ID,
				"filename":    doc.FileName,
				"status":      doc.ProcessStatus,
			},
		}
	}()

	// 加入处理队列
	err = s.worker.EnqueueDocument(c.Request.Context(), doc.ID)
	if err != nil {
		s.logger.Error("failed to enqueue document", zap.Error(err))
	}

	// 开始流式传输
	sse.StreamResponse(c, client, s.sseHub, 30*time.Second)
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

// StreamDocumentStatus SSE 流式推送文档处理状态
func (s *DocumentService) StreamDocumentStatus(c *gin.Context) {
	docID := c.Param("doc_id")

	// 验证文档是否存在
	doc, err := s.docUseCase.DocumentRepo.GetByID(c.Request.Context(), docID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "document not found")
		return
	}

	// 创建 SSE 客户端
	client := &sse.Client{
		ID:       uuid.New().String(),
		Channel:  make(chan sse.Event, 10),
		Resource: "doc:" + docID,
	}

	// 发送当前状态
	go func() {
		time.Sleep(100 * time.Millisecond)
		client.Channel <- sse.Event{
			Type: "status",
			Data: map[string]interface{}{
				"document_id": doc.ID,
				"status":      doc.ProcessStatus,
				"chunk_count": doc.ChunkCount,
				"error":       doc.ProcessError,
			},
		}
	}()

	// 开始流式传输
	sse.StreamResponse(c, client, s.sseHub, 30*time.Second)
}

func getFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i+1:]
		}
	}
	return ""
}
