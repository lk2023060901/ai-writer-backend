package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/queue"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/response"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/sse"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/workerpool"
	"go.uber.org/zap"
)

type DocumentService struct {
	docUseCase *biz.DocumentUseCase
	worker     *queue.Worker
	uploadPool *workerpool.Pool // 新增：上传 Worker Pool
	sseHub     *sse.Hub
	logger     *zap.Logger
}

func NewDocumentService(
	docUseCase *biz.DocumentUseCase,
	worker *queue.Worker,
	uploadPool *workerpool.Pool,
	sseHub *sse.Hub,
	logger *zap.Logger,
) *DocumentService {
	return &DocumentService{
		docUseCase: docUseCase,
		worker:     worker,
		uploadPool: uploadPool,
		sseHub:     sseHub,
		logger:     logger,
	}
}

// UploadDocument 单文件上传（返回 JSON）
func (s *DocumentService) UploadDocument(c *gin.Context) {
	kbID := c.Param("id")
	userID := c.GetString("user_id")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid file or field name is not 'file'")
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

	s.logger.Info("single file upload",
		zap.String("kb_id", kbID),
		zap.String("filename", fileName),
		zap.String("file_type", fileType),
		zap.Int("file_size", len(fileData)))

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
		s.logger.Error("failed to enqueue document", zap.String("doc_id", doc.ID), zap.Error(err))
		// 不影响响应，只记录错误
	}

	// 返回 JSON 响应
	response.Success(c, map[string]interface{}{
		"document": toDocumentResponse(doc),
		"message":  fmt.Sprintf("File '%s' uploaded successfully", fileName),
	})
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

// BatchDeleteDocuments 批量删除文档
func (s *DocumentService) BatchDeleteDocuments(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		DocumentIDs []string `json:"document_ids" binding:"required,min=1,max=100"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid parameters: document_ids required (1-100 items)")
		return
	}

	result := s.docUseCase.BatchDeleteDocuments(c.Request.Context(), req.DocumentIDs, userID)

	response.Success(c, result)
}

// BatchUploadDocuments 批量上传文档
func (s *DocumentService) BatchUploadDocuments(c *gin.Context) {
	kbID := c.Param("id")
	userID := c.GetString("user_id")

	// 解析 multipart form（最大 100MB）
	if err := c.Request.ParseMultipartForm(100 << 20); err != nil {
		response.Error(c, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
		return
	}

	form := c.Request.MultipartForm
	if form == nil || form.File == nil {
		response.Error(c, http.StatusBadRequest, "no files uploaded")
		return
	}

	// 获取所有上传的文件
	// 收集所有文件（支持多种字段名：files, file, 或其他自定义字段名）
	var allFileHeaders []*multipart.FileHeader

	// 优先使用 "files" 字段
	if fileHeaders, ok := form.File["files"]; ok && len(fileHeaders) > 0 {
		allFileHeaders = fileHeaders
	} else {
		// 如果没有 "files" 字段，收集所有字段的文件
		for _, fileHeaders := range form.File {
			allFileHeaders = append(allFileHeaders, fileHeaders...)
		}
	}

	if len(allFileHeaders) == 0 {
		response.Error(c, http.StatusBadRequest, "no files uploaded")
		return
	}

	s.logger.Info("batch upload request",
		zap.Int("file_count", len(allFileHeaders)),
		zap.String("kb_id", kbID))

	// 限制最多上传 50 个文件
	if len(allFileHeaders) > 50 {
		response.Error(c, http.StatusBadRequest, "too many files: maximum 50 files per batch")
		return
	}

	// 读取文件数据
	files := make([]*biz.UploadFile, 0, len(allFileHeaders))
	for _, fileHeader := range allFileHeaders {
		file, err := fileHeader.Open()
		if err != nil {
			response.Error(c, http.StatusBadRequest, "failed to open file: "+fileHeader.Filename)
			return
		}
		defer file.Close()

		fileData, err := io.ReadAll(file)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "failed to read file: "+fileHeader.Filename)
			return
		}

		fileName := fileHeader.Filename
		fileType := getFileExtension(fileName)

		files = append(files, &biz.UploadFile{
			FileName: fileName,
			FileType: fileType,
			FileData: fileData,
		})
	}

	// 创建 SSE 客户端（用于批量上传）
	client := &sse.Client{
		ID:       uuid.New().String(),
		Channel:  make(chan sse.Event, 50), // 增加缓冲区以支持多个文件
		Resource: "kb:" + kbID,              // 使用知识库资源，与 Worker 广播一致
	}

	// 在 goroutine 中使用 Worker Pool 并发上传
	go func() {
		defer close(client.Channel)

		// 发送开始事件
		client.Channel <- sse.Event{
			Type: "batch-start",
			Data: map[string]interface{}{
				"total_count": len(files),
				"message":     fmt.Sprintf("Starting batch upload of %d files", len(files)),
			},
		}

		// 使用 Worker Pool 并发上传
		successCount := 0
		failedCount := 0
		completedCount := 0

		// 结果 channel
		type uploadResult struct {
			Index    int
			Doc      *biz.Document
			FileName string
			Error    error
		}
		resultCh := make(chan uploadResult, len(files))

		// 提交所有文件到 Worker Pool
		for i, file := range files {
			idx := i
			f := file

			s.uploadPool.Submit(func() {
				// 上传单个文件
				doc, err := s.docUseCase.UploadDocument(c.Request.Context(), kbID, userID, f.FileName, f.FileData, f.FileType)
				resultCh <- uploadResult{
					Index:    idx,
					Doc:      doc,
					FileName: f.FileName,
					Error:    err,
				}
			})
		}

		// 收集结果并实时推送
		for range files {
			result := <-resultCh
			completedCount++

			if result.Error != nil {
				failedCount++
				client.Channel <- sse.Event{
					Type: "file-failed",
					Data: map[string]interface{}{
						"index":     result.Index + 1,
						"filename":  result.FileName,
						"error":     result.Error.Error(),
						"message":   fmt.Sprintf("File '%s' upload failed: %s", result.FileName, result.Error.Error()),
						"completed": completedCount,
						"total":     len(files),
					},
				}
			} else {
				successCount++
				client.Channel <- sse.Event{
					Type: "file-uploaded",
					Data: map[string]interface{}{
						"index":     result.Index + 1,
						"total":     len(files),
						"document":  toDocumentResponse(result.Doc),
						"message":   fmt.Sprintf("File '%s' uploaded successfully", result.Doc.FileName),
						"completed": completedCount,
					},
				}

				// 加入处理队列
				err := s.worker.EnqueueDocument(c.Request.Context(), result.Doc.ID)
				if err != nil {
					s.logger.Error("failed to enqueue document", zap.String("doc_id", result.Doc.ID), zap.Error(err))
				}
			}
		}

		close(resultCh)

		// 发送完成事件
		client.Channel <- sse.Event{
			Type: "batch-complete",
			Data: map[string]interface{}{
				"total_count":   len(files),
				"success_count": successCount,
				"failed_count":  failedCount,
				"message":       fmt.Sprintf("Batch upload completed: %d succeeded, %d failed", successCount, failedCount),
			},
		}
	}()

	// 开始流式传输（30秒超时）
	sse.StreamResponse(c, client, s.sseHub, 30*time.Second)
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
// 前端只需传 query，所有配置（TopK、Rerank、HybridSearch）都从知识库配置中读取
func (s *DocumentService) SearchDocuments(c *gin.Context) {
	kbID := c.Param("id")
	userID := c.GetString("user_id")

	var req struct {
		Query string `json:"query" binding:"required,min=1,max=1000"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid parameters: query required (1-1000 chars)")
		return
	}

	// 使用知识库配置的默认 TopK（不允许前端覆盖）
	results, err := s.docUseCase.SearchDocuments(c.Request.Context(), kbID, userID, req.Query, 0)
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

	// 支持通过 query 参数自定义 resource（用于批量上传监听整个知识库）
	resource := c.Query("resource")
	if resource == "" {
		// 默认监听单个文档
		resource = "doc:" + docID
	}

	// 如果不是批量监听（kb:xxx），验证文档是否存在
	if resource == "doc:"+docID {
		doc, err := s.docUseCase.DocumentRepo.GetByID(c.Request.Context(), docID)
		if err != nil {
			response.Error(c, http.StatusNotFound, "document not found")
			return
		}

		// 创建 SSE 客户端
		client := &sse.Client{
			ID:       uuid.New().String(),
			Channel:  make(chan sse.Event, 10),
			Resource: resource,
		}

		// 发送当前状态
		go func() {
			time.Sleep(100 * time.Millisecond)
			client.Channel <- sse.Event{
				Type: "status",
				Data: map[string]interface{}{
					"document": toDocumentResponse(doc),
					"message":  "Current document status",
				},
			}
		}()

		// 开始流式传输
		sse.StreamResponse(c, client, s.sseHub, 30*time.Second)
	} else {
		// 批量监听知识库级别（kb:xxx）
		client := &sse.Client{
			ID:       uuid.New().String(),
			Channel:  make(chan sse.Event, 50), // 批量上传需要更大的缓冲区
			Resource: resource,
		}

		// 开始流式传输（批量监听不需要发送初始状态）
		sse.StreamResponse(c, client, s.sseHub, 5*time.Minute) // 批量上传可能需要更长时间
	}
}

func getFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i+1:]
		}
	}
	return ""
}
