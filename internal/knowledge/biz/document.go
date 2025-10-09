package biz

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/hybrid"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"go.uber.org/zap"
)

// Document 文档模型
type Document struct {
	ID              string
	KnowledgeBaseID string
	FileName        string
	FileType        string // pdf, docx, txt, md
	FileSize        int64
	FileHash        string          // 文件SHA256哈希（去重用）
	MinioBucket     string          // MinIO bucket名称
	MinioObjectKey  string          // MinIO对象键（基于hash的物理路径: files/{hash[:2]}/{hash}）
	ProcessStatus   string          // pending, processing, completed, failed
	ProcessError    string
	ChunkCount      int64
	TokenCount      int
	Metadata        map[string]interface{} // 额外元数据

	// 多模态支持
	SourceType    string // file, url, text
	SourceURL     string // URL来源（当source_type=url时）
	SourceContent string // 文本内容（当source_type=text时）

	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Chunk 文档分块模型
type Chunk struct {
	ID              string
	DocumentID      string
	KnowledgeBaseID string
	Content         string
	Position        int // 块的位置序号
	TokenCount      int
	Embedding       []float32 // 向量
	Metadata        map[string]interface{}
	CreatedAt       time.Time
}

// DocumentUseCase 文档用例接口
type DocumentUseCase struct {
	DocumentRepo    DocumentRepo
	chunkRepo       ChunkRepo
	kbRepo          KnowledgeBaseRepo
	aiModelRepo     AIModelRepo
	aiProviderRepo  AIProviderRepo
	fileStorageRepo FileStorageRepo
	storage         StorageService
	vectorDB        VectorDBService
	embedder        EmbeddingService
	processor       DocumentProcessor
	logger          *logger.Logger
}

// DocumentRepo 文档仓储接口
type DocumentRepo interface {
	Create(ctx context.Context, doc *Document) error
	GetByID(ctx context.Context, id string) (*Document, error)
	GetByIDs(ctx context.Context, ids []string) ([]*Document, error)  // 批量查询
	List(ctx context.Context, kbID string, req *ListDocumentsRequest) ([]*Document, int64, error)
	Update(ctx context.Context, doc *Document) error
	Delete(ctx context.Context, id string) error
	BatchDelete(ctx context.Context, ids []string) error  // 批量删除
	UpdateStatus(ctx context.Context, id, status, errorMsg string) error
}

// ChunkRepo 分块仓储接口
type ChunkRepo interface {
	BatchCreate(ctx context.Context, chunks []*Chunk) error
	GetByDocumentID(ctx context.Context, docID string) ([]*Chunk, error)
	DeleteByDocumentID(ctx context.Context, docID string) error
	BatchDeleteByDocumentIDs(ctx context.Context, docIDs []string) error  // 批量删除
	DeleteByKnowledgeBaseID(ctx context.Context, kbID string) error
	KeywordSearch(ctx context.Context, kbID, query string, topK int) ([]*Chunk, error) // 关键词搜索
}

// FileStorageRepo 文件存储仓储接口
type FileStorageRepo interface {
	Create(ctx context.Context, fs *FileStorage) error
	GetByHash(ctx context.Context, fileHash string) (*FileStorage, error)
	IncrementReference(ctx context.Context, fileHash string) error
	DecrementReference(ctx context.Context, fileHash string) error
	BatchDecrementReferences(ctx context.Context, fileHashes []string) error  // 批量递减引用
	DeleteIfNoReferences(ctx context.Context, fileHash string) (bool, error)
}

// FileStorage 文件存储模型
type FileStorage struct {
	FileHash         string
	Bucket           string
	ObjectKey        string
	FileSize         int64
	ContentType      string
	ReferenceCount   int
	FirstUploadedAt  time.Time
	LastReferencedAt time.Time
}

// StorageService 对象存储服务接口（MinIO）
type StorageService interface {
	UploadFile(ctx context.Context, bucket, objectName string, data []byte, contentType string) (string, error)
	GetFile(ctx context.Context, bucket, objectName string) ([]byte, error)
	DeleteFile(ctx context.Context, bucket, objectName string) error
}

// VectorDBService 向量数据库服务接口（Milvus）
type VectorDBService interface {
	CreateCollection(ctx context.Context, collectionName string, dimension int) error
	InsertVectors(ctx context.Context, collectionName string, chunks []*Chunk) error
	Search(ctx context.Context, collectionName string, vector []float32, topK int) ([]*SearchResult, error)
	SearchWithThreshold(ctx context.Context, collectionName string, vector []float32, topK int, minScore float32) ([]*SearchResult, error)
	DeleteByDocumentID(ctx context.Context, collectionName, documentID string) error
	DropCollection(ctx context.Context, collectionName string) error
}

// EmbeddingService Embedding 生成服务接口
type EmbeddingService interface {
	GenerateEmbeddings(ctx context.Context, texts []string, provider *AIProvider, model *AIModel) ([][]float32, error)
}

// DocumentProcessor 文档处理器接口
type DocumentProcessor interface {
	ExtractText(ctx context.Context, fileData []byte, fileType string) (string, error)
	ChunkText(text string, chunkSize, chunkOverlap int, strategy string) ([]string, error)
}

// SearchResult 搜索结果
type SearchResult struct {
	ChunkID    string
	DocumentID string
	Content    string
	Score      float32
	Metadata   map[string]interface{}
}

// ListDocumentsRequest 列表请求
type ListDocumentsRequest struct {
	Page     int
	PageSize int
}

// NewDocumentUseCase 创建文档用例
func NewDocumentUseCase(
	documentRepo DocumentRepo,
	chunkRepo ChunkRepo,
	kbRepo KnowledgeBaseRepo,
	aiModelRepo AIModelRepo,
	aiProviderRepo AIProviderRepo,
	fileStorageRepo FileStorageRepo,
	storage StorageService,
	vectorDB VectorDBService,
	embedder EmbeddingService,
	processor DocumentProcessor,
	log *logger.Logger,
) *DocumentUseCase {
	return &DocumentUseCase{
		DocumentRepo:    documentRepo,
		chunkRepo:       chunkRepo,
		kbRepo:          kbRepo,
		aiModelRepo:     aiModelRepo,
		aiProviderRepo:  aiProviderRepo,
		fileStorageRepo: fileStorageRepo,
		storage:         storage,
		vectorDB:        vectorDB,
		embedder:        embedder,
		processor:       processor,
		logger:          log,
	}
}
// UploadDocument 上传文档（支持内容去重）
func (uc *DocumentUseCase) UploadDocument(ctx context.Context, kbID, userID string, fileName string, fileData []byte, fileType string) (*Document, error) {
	// 验证知识库权限
	kb, err := uc.kbRepo.GetByID(ctx, kbID, userID)
	if err != nil {
		return nil, fmt.Errorf("knowledge base not found: %w", err)
	}

	if kb.OwnerID != userID && kb.OwnerID != SystemOwnerID {
		return nil, fmt.Errorf("permission denied")
	}

	// 计算文件hash
	fileHash := calculateSHA256(fileData)
	bucket := "knowledge-bases"
	contentType := getContentType(fileType)

	// 检查文件是否已存在（去重）
	existingFile, err := uc.fileStorageRepo.GetByHash(ctx, fileHash)
	if err != nil {
		return nil, fmt.Errorf("failed to check file existence: %w", err)
	}

	var physicalPath string
	if existingFile != nil {
		// 文件已存在，增加引用计数
		err = uc.fileStorageRepo.IncrementReference(ctx, fileHash)
		if err != nil {
			return nil, fmt.Errorf("failed to increment reference: %w", err)
		}
		physicalPath = existingFile.ObjectKey
	} else {
		// 新文件，基于 hash 生成存储路径
		physicalPath = fmt.Sprintf("files/%s/%s", fileHash[:2], fileHash)

		// 上传到MinIO
		_, err = uc.storage.UploadFile(ctx, bucket, physicalPath, fileData, contentType)
		if err != nil {
			return nil, fmt.Errorf("failed to upload file: %w", err)
		}

		// 创建文件存储记录
		now := time.Now()
		fileStorage := &FileStorage{
			FileHash:         fileHash,
			Bucket:           bucket,
			ObjectKey:        physicalPath,
			FileSize:         int64(len(fileData)),
			ContentType:      contentType,
			ReferenceCount:   1,
			FirstUploadedAt:  now,
			LastReferencedAt: now,
		}

		err = uc.fileStorageRepo.Create(ctx, fileStorage)
		if err != nil {
			// 清理MinIO文件
			_ = uc.storage.DeleteFile(ctx, bucket, physicalPath)
			return nil, fmt.Errorf("failed to create file storage: %w", err)
		}
	}

	// 创建文档记录（引用物理文件）
	docID := uuid.New().String()
	doc := &Document{
		ID:              docID,
		KnowledgeBaseID: kbID,
		FileName:        fileName,
		FileType:        fileType,
		FileSize:        int64(len(fileData)),
		FileHash:        fileHash,
		MinioBucket:     bucket,
		MinioObjectKey:  physicalPath, // 基于hash的物理路径: files/{hash[:2]}/{hash}
		ProcessStatus:   "pending",
		TokenCount:      0,
		ChunkCount:      0,
		SourceType:      "file", // 文件上传类型
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err = uc.DocumentRepo.Create(ctx, doc)
	if err != nil {
		// 回滚引用计数
		if existingFile != nil {
			_ = uc.fileStorageRepo.DecrementReference(ctx, fileHash)
		} else {
			_, _ = uc.fileStorageRepo.DeleteIfNoReferences(ctx, fileHash)
			_ = uc.storage.DeleteFile(ctx, bucket, physicalPath)
		}
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	return doc, nil
}

// ProcessDocument 处理文档（异步任务调用）
func (uc *DocumentUseCase) ProcessDocument(ctx context.Context, documentID string) error {
	// 更新状态为处理中
	err := uc.DocumentRepo.UpdateStatus(ctx, documentID, "processing", "")
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// 获取文档信息
	doc, err := uc.DocumentRepo.GetByID(ctx, documentID)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", "document not found")
		return fmt.Errorf("document not found: %w", err)
	}

	// 获取知识库信息
	kb, err := uc.kbRepo.GetByID(ctx, doc.KnowledgeBaseID, "")
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", "knowledge base not found")
		return fmt.Errorf("knowledge base not found: %w", err)
	}

	// 获取AI Model
	aiModel, err := uc.aiModelRepo.GetByID(ctx, kb.EmbeddingModelID)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", "AI model not found")
		return fmt.Errorf("AI model not found: %w", err)
	}

	// 获取AI Provider
	aiProvider, err := uc.aiProviderRepo.GetByID(ctx, aiModel.ProviderID)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", "AI provider not found")
		return fmt.Errorf("AI provider not found: %w", err)
	}

	// 从MinIO获取文件
	fileData, err := uc.storage.GetFile(ctx, doc.MinioBucket, doc.MinioObjectKey)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", fmt.Sprintf("failed to get file: %v", err))
		return fmt.Errorf("failed to get file: %w", err)
	}

	// 提取文本
	text, err := uc.processor.ExtractText(ctx, fileData, doc.FileType)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", fmt.Sprintf("failed to extract text: %v", err))
		return fmt.Errorf("failed to extract text: %w", err)
	}

	// 分块
	chunkTexts, err := uc.processor.ChunkText(text, kb.ChunkSize, kb.ChunkOverlap, kb.ChunkStrategy)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", fmt.Sprintf("failed to chunk text: %v", err))
		return fmt.Errorf("failed to chunk text: %w", err)
	}

	if len(chunkTexts) == 0 {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", "no content extracted")
		return fmt.Errorf("no content extracted")
	}

	// 生成 Embeddings
	embeddings, err := uc.embedder.GenerateEmbeddings(ctx, chunkTexts, aiProvider, aiModel)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", fmt.Sprintf("failed to generate embeddings: %v", err))
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// 确保 Milvus collection 存在
	collectionName := kb.MilvusCollection

	// 检查模型是否支持 embedding
	hasEmbedding := false
	for _, cap := range aiModel.Capabilities {
		if cap == CapabilityTypeEmbedding {
			hasEmbedding = true
			break
		}
	}

	if !hasEmbedding {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", "model does not support embedding")
		return fmt.Errorf("model does not support embedding: %s", aiModel.ModelName)
	}

	// 从模型直接获取 embedding dimensions
	if aiModel.EmbeddingDimensions == nil || *aiModel.EmbeddingDimensions == 0 {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", "embedding dimensions not configured")
		return fmt.Errorf("embedding dimensions not configured for model: %s", aiModel.ModelName)
	}

	embeddingDimensions := *aiModel.EmbeddingDimensions

	err = uc.vectorDB.CreateCollection(ctx, collectionName, embeddingDimensions)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", fmt.Sprintf("failed to create collection: %v", err))
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// 创建 Chunks
	chunks := make([]*Chunk, len(chunkTexts))
	for i, chunkText := range chunkTexts {
		// 清理无效的 UTF-8 字符
		cleanedText := sanitizeUTF8(chunkText)

		chunks[i] = &Chunk{
			ID:              uuid.New().String(),
			DocumentID:      documentID,
			KnowledgeBaseID: doc.KnowledgeBaseID,
			Content:         cleanedText,
			Position:        i,
			TokenCount:      len(cleanedText) / 4, // 粗略估算
			Embedding:       embeddings[i],
			CreatedAt:       time.Now(),
		}
	}

	// 先插入向量到 Milvus（避免数据库失败导致 Milvus 插入被跳过）
	err = uc.vectorDB.InsertVectors(ctx, collectionName, chunks)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", fmt.Sprintf("failed to insert vectors: %v", err))
		return fmt.Errorf("failed to insert vectors: %w", err)
	}

	// 再保存到数据库
	err = uc.chunkRepo.BatchCreate(ctx, chunks)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", fmt.Sprintf("failed to save chunks: %v", err))
		return fmt.Errorf("failed to save chunks: %w", err)
	}

	// 先增加知识库文档计数
	err = uc.kbRepo.IncrementDocumentCount(ctx, doc.KnowledgeBaseID, 1)
	if err != nil {
		return fmt.Errorf("failed to increment document count: %w", err)
	}

	// 更新文档状态
	doc.ProcessStatus = "completed"
	doc.ChunkCount = int64(len(chunks))
	doc.UpdatedAt = time.Now()
	err = uc.DocumentRepo.Update(ctx, doc)
	if err != nil {
		// 回滚文档计数
		_ = uc.kbRepo.IncrementDocumentCount(ctx, doc.KnowledgeBaseID, -1)
		return fmt.Errorf("failed to update document: %w", err)
	}

	return nil
}

// DeleteDocument 删除文档
func (uc *DocumentUseCase) DeleteDocument(ctx context.Context, documentID, userID string) error {
	// 获取文档
	doc, err := uc.DocumentRepo.GetByID(ctx, documentID)
	if err != nil {
		return fmt.Errorf("document not found: %w", err)
	}

	// 验证权限
	kb, err := uc.kbRepo.GetByID(ctx, doc.KnowledgeBaseID, "")
	if err != nil {
		return fmt.Errorf("knowledge base not found: %w", err)
	}

	if kb.OwnerID != userID && kb.OwnerID != SystemOwnerID {
		return fmt.Errorf("permission denied")
	}

	// 删除 Milvus 向量
	_ = uc.vectorDB.DeleteByDocumentID(ctx, kb.MilvusCollection, documentID)

	// 删除数据库中的 chunks
	_ = uc.chunkRepo.DeleteByDocumentID(ctx, documentID)

	// 删除文档记录
	err = uc.DocumentRepo.Delete(ctx, documentID)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	// 减少文件引用计数
	err = uc.fileStorageRepo.DecrementReference(ctx, doc.FileHash)
	if err != nil {
		// 记录错误但不中断流程
		_ = fmt.Errorf("failed to decrement file reference: %w", err)
	}

	// 如果引用计数为0，删除物理文件
	deleted, err := uc.fileStorageRepo.DeleteIfNoReferences(ctx, doc.FileHash)
	if err != nil {
		_ = fmt.Errorf("failed to check file references: %w", err)
	}
	if deleted {
		// 删除 MinIO 中的物理文件
		_ = uc.storage.DeleteFile(ctx, doc.MinioBucket, doc.MinioObjectKey)
	}

	// 减少知识库文档计数
	_ = uc.kbRepo.IncrementDocumentCount(ctx, doc.KnowledgeBaseID, -1)

	return nil
}

// BatchDeleteDocuments 批量删除文档
func (uc *DocumentUseCase) BatchDeleteDocuments(ctx context.Context, documentIDs []string, userID string) *BatchDeleteResult {
	result := &BatchDeleteResult{
		TotalCount:   len(documentIDs),
		SuccessCount: 0,
		FailedCount:  0,
		FailedItems:  make([]FailedItem, 0),
	}

	// 第1步：批量获取所有文档（1次查询）
	docs, err := uc.DocumentRepo.GetByIDs(ctx, documentIDs)
	if err != nil {
		// 如果批量查询失败，回退到逐个查询
		docs = make([]*Document, 0, len(documentIDs))
		for _, docID := range documentIDs {
			doc, _ := uc.DocumentRepo.GetByID(ctx, docID)
			if doc != nil {
				docs = append(docs, doc)
			}
		}
	}

	// 构建文档 ID 到文档的映射
	docMap := make(map[string]*Document)
	kbIDs := make(map[string]bool)
	for _, doc := range docs {
		docMap[doc.ID] = doc
		kbIDs[doc.KnowledgeBaseID] = true
	}

	// 第2步：批量获取知识库信息（一次性查询所有需要的知识库）
	kbMap := make(map[string]*KnowledgeBase)
	for kbID := range kbIDs {
		kb, _ := uc.kbRepo.GetByID(ctx, kbID, "")
		if kb != nil {
			kbMap[kbID] = kb
		}
	}

	// 第3步：按知识库分组文档ID（为批量删除 Milvus 做准备）
	kbDocGroups := make(map[string][]string)
	fileHashes := make([]string, 0, len(docs))
	kbDeleteCounts := make(map[string]int)

	// 遍历所有要删除的文档ID
	for _, docID := range documentIDs {
		doc, docExists := docMap[docID]
		if !docExists {
			result.FailedCount++
			result.FailedItems = append(result.FailedItems, FailedItem{
				DocumentID: docID,
				Error:      "document not found",
			})
			continue
		}

		kb, kbExists := kbMap[doc.KnowledgeBaseID]
		if !kbExists {
			result.FailedCount++
			result.FailedItems = append(result.FailedItems, FailedItem{
				DocumentID: docID,
				Error:      "knowledge base not found",
			})
			continue
		}

		// 验证权限
		if kb.OwnerID != userID && kb.OwnerID != SystemOwnerID {
			result.FailedCount++
			result.FailedItems = append(result.FailedItems, FailedItem{
				DocumentID: docID,
				Error:      "permission denied",
			})
			continue
		}

		// 收集待删除的文档ID（按知识库分组）
		kbDocGroups[kb.MilvusCollection] = append(kbDocGroups[kb.MilvusCollection], docID)
		fileHashes = append(fileHashes, doc.FileHash)
		kbDeleteCounts[doc.KnowledgeBaseID]++
		result.SuccessCount++
	}

	// 构建成功删除的文档ID列表
	successDocIDs := make([]string, 0, result.SuccessCount)
	for _, docID := range documentIDs {
		if _, exists := docMap[docID]; exists {
			successDocIDs = append(successDocIDs, docID)
		}
	}

	// 如果没有成功的文档，直接返回
	if len(successDocIDs) == 0 {
		return result
	}

	// 第4步：批量删除 Milvus 向量（按知识库批量删除）
	for collection, docIDs := range kbDocGroups {
		if len(docIDs) > 0 {
			// TODO: 如果 VectorDB 接口支持批量删除，这里可以改为一次性删除
			for _, docID := range docIDs {
				_ = uc.vectorDB.DeleteByDocumentID(ctx, collection, docID)
			}
		}
	}

	// 第5步：批量删除 chunks（一次性删除所有）
	_ = uc.chunkRepo.BatchDeleteByDocumentIDs(ctx, successDocIDs)

	// 第6步：批量删除文档记录（新增批量删除方法）
	_ = uc.DocumentRepo.BatchDelete(ctx, successDocIDs)

	// 第7步：批量处理文件引用和物理删除
	// 构建 fileHash 到文档的映射（用于获取正确的 bucket 和 key）
	hashToDoc := make(map[string]*Document)
	for _, doc := range docs {
		if _, exists := hashToDoc[doc.FileHash]; !exists {
			hashToDoc[doc.FileHash] = doc
		}
	}

	// 批量减少文件引用计数（新增批量递减方法）
	_ = uc.fileStorageRepo.BatchDecrementReferences(ctx, fileHashes)

	// 批量检查并删除物理文件
	for _, fileHash := range fileHashes {
		deleted, _ := uc.fileStorageRepo.DeleteIfNoReferences(ctx, fileHash)
		if deleted {
			// 使用正确的文档信息获取 bucket 和 key
			if doc, exists := hashToDoc[fileHash]; exists && doc != nil {
				_ = uc.storage.DeleteFile(ctx, doc.MinioBucket, doc.MinioObjectKey)
			}
		}
	}

	// 第8步：批量更新知识库文档计数（一次性更新所有知识库）
	_ = uc.kbRepo.BatchUpdateDocumentCounts(ctx, kbDeleteCounts)

	return result
}

// BatchDeleteResult 批量删除结果
type BatchDeleteResult struct {
	TotalCount   int          `json:"total_count"`
	SuccessCount int          `json:"success_count"`
	FailedCount  int          `json:"failed_count"`
	FailedItems  []FailedItem `json:"failed_items,omitempty"`
}

// FailedItem 失败项
type FailedItem struct {
	DocumentID string `json:"document_id"`
	Error      string `json:"error"`
}

// BatchUploadDocuments 批量上传文档
func (uc *DocumentUseCase) BatchUploadDocuments(ctx context.Context, kbID, userID string, files []*UploadFile) *BatchUploadResult {
	result := &BatchUploadResult{
		TotalCount:      len(files),
		SuccessCount:    0,
		FailedCount:     0,
		SuccessItems:    make([]*Document, 0),
		FailedUploadItems: make([]FailedUploadItem, 0),
	}

	// 验证知识库权限
	kb, err := uc.kbRepo.GetByID(ctx, kbID, userID)
	if err != nil {
		// 全部失败
		result.FailedCount = len(files)
		for _, file := range files {
			result.FailedUploadItems = append(result.FailedUploadItems, FailedUploadItem{
				FileName: file.FileName,
				Error:    "knowledge base not found",
			})
		}
		return result
	}

	if kb.OwnerID != userID && kb.OwnerID != SystemOwnerID {
		// 全部失败
		result.FailedCount = len(files)
		for _, file := range files {
			result.FailedUploadItems = append(result.FailedUploadItems, FailedUploadItem{
				FileName: file.FileName,
				Error:    "permission denied",
			})
		}
		return result
	}

	// 逐个上传文件（支持去重）
	for _, file := range files {
		doc, err := func() (*Document, error) {
			// 计算文件hash
			fileHash := calculateSHA256(file.FileData)
			bucket := "knowledge-bases"
			contentType := getContentType(file.FileType)

			// 检查文件是否已存在（去重）
			existingFile, err := uc.fileStorageRepo.GetByHash(ctx, fileHash)
			if err != nil {
				return nil, fmt.Errorf("failed to check file existence: %w", err)
			}

			var physicalPath string
			if existingFile != nil {
				// 文件已存在，增加引用计数
				err = uc.fileStorageRepo.IncrementReference(ctx, fileHash)
				if err != nil {
					return nil, fmt.Errorf("failed to increment reference: %w", err)
				}
				physicalPath = existingFile.ObjectKey
			} else {
				// 新文件，基于 hash 生成存储路径
				physicalPath = fmt.Sprintf("files/%s/%s", fileHash[:2], fileHash)

				// 上传到MinIO
				_, err = uc.storage.UploadFile(ctx, bucket, physicalPath, file.FileData, contentType)
				if err != nil {
					return nil, fmt.Errorf("failed to upload file: %w", err)
				}

				// 创建文件存储记录
				now := time.Now()
				fileStorage := &FileStorage{
					FileHash:         fileHash,
					Bucket:           bucket,
					ObjectKey:        physicalPath,
					FileSize:         int64(len(file.FileData)),
					ContentType:      contentType,
					ReferenceCount:   1,
					FirstUploadedAt:  now,
					LastReferencedAt: now,
				}

				err = uc.fileStorageRepo.Create(ctx, fileStorage)
				if err != nil {
					// 清理MinIO文件
					_ = uc.storage.DeleteFile(ctx, bucket, physicalPath)
					return nil, fmt.Errorf("failed to create file storage: %w", err)
				}
			}

			// 创建文档记录
			docID := uuid.New().String()
			doc := &Document{
				ID:              docID,
				KnowledgeBaseID: kbID,
				FileName:        file.FileName,
				FileType:        file.FileType,
				FileSize:        int64(len(file.FileData)),
				FileHash:        fileHash,
				MinioBucket:     bucket,
				MinioObjectKey:  physicalPath,
				ProcessStatus:   "pending",
				TokenCount:      0,
				ChunkCount:      0,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			err = uc.DocumentRepo.Create(ctx, doc)
			if err != nil {
				// 回滚引用计数
				if existingFile != nil {
					_ = uc.fileStorageRepo.DecrementReference(ctx, fileHash)
				} else {
					_, _ = uc.fileStorageRepo.DeleteIfNoReferences(ctx, fileHash)
					_ = uc.storage.DeleteFile(ctx, bucket, physicalPath)
				}
				return nil, fmt.Errorf("failed to create document: %w", err)
			}

			return doc, nil
		}()

		if err != nil {
			result.FailedCount++
			result.FailedUploadItems = append(result.FailedUploadItems, FailedUploadItem{
				FileName: file.FileName,
				Error:    err.Error(),
			})
		} else {
			result.SuccessCount++
			result.SuccessItems = append(result.SuccessItems, doc)
		}
	}

	return result
}

// UploadFile 上传文件数据
type UploadFile struct {
	FileName string
	FileType string
	FileData []byte
}

// BatchUploadResult 批量上传结果
type BatchUploadResult struct {
	TotalCount        int                 `json:"total_count"`
	SuccessCount      int                 `json:"success_count"`
	FailedCount       int                 `json:"failed_count"`
	SuccessItems      []*Document         `json:"success_items,omitempty"`
	FailedUploadItems []FailedUploadItem  `json:"failed_items,omitempty"`
}

// FailedUploadItem 上传失败项
type FailedUploadItem struct {
	FileName string `json:"file_name"`
	Error    string `json:"error"`
}

// SearchDocuments 向量搜索（支持混合检索）
func (uc *DocumentUseCase) SearchDocuments(ctx context.Context, kbID, userID, query string, topK int) ([]*SearchResult, error) {
	// 记录搜索请求
	uc.logger.Info("知识库搜索请求",
		zap.String("kb_id", kbID),
		zap.String("user_id", userID),
		zap.String("query", query),
		zap.Int("requested_top_k", topK))

	// 验证权限
	kb, err := uc.kbRepo.GetByID(ctx, kbID, userID)
	if err != nil {
		uc.logger.Error("知识库未找到", zap.Error(err))
		return nil, fmt.Errorf("knowledge base not found: %w", err)
	}

	if kb.OwnerID != userID && kb.OwnerID != SystemOwnerID {
		uc.logger.Warn("知识库访问权限被拒绝",
			zap.String("kb_id", kbID),
			zap.String("user_id", userID),
			zap.String("owner_id", kb.OwnerID))
		return nil, fmt.Errorf("permission denied")
	}

	// 使用知识库配置的 TopK，如果调用者传入的 topK > 0 则优先使用
	searchTopK := kb.TopK
	if topK > 0 {
		searchTopK = topK
	}

	uc.logger.Info("知识库搜索配置",
		zap.String("kb_name", kb.Name),
		zap.Int("search_top_k", searchTopK),
		zap.Float32("threshold", kb.Threshold),
		zap.Bool("enable_hybrid_search", kb.EnableHybridSearch))

	// 获取AI Model
	aiModel, err := uc.aiModelRepo.GetByID(ctx, kb.EmbeddingModelID)
	if err != nil {
		return nil, fmt.Errorf("AI model not found: %w", err)
	}

	// 获取AI Provider
	aiProvider, err := uc.aiProviderRepo.GetByID(ctx, aiModel.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("AI provider not found: %w", err)
	}

	// 生成查询的 embedding
	embeddings, err := uc.embedder.GenerateEmbeddings(ctx, []string{query}, aiProvider, aiModel)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	var results []*SearchResult

	// 判断是否启用混合检索
	if kb.EnableHybridSearch {
		// 混合检索：向量搜索 + 关键词搜索 + RRF 融合
		results, err = uc.hybridSearch(ctx, kb.MilvusCollection, kbID, embeddings[0], query, searchTopK, kb.Threshold)
		if err != nil {
			return nil, fmt.Errorf("hybrid search failed: %w", err)
		}
	} else {
		// 纯向量搜索（在数据库层面应用阈值过滤）
		results, err = uc.vectorDB.SearchWithThreshold(ctx, kb.MilvusCollection, embeddings[0], searchTopK, kb.Threshold)
		if err != nil {
			return nil, fmt.Errorf("failed to search: %w", err)
		}
	}

	// 补充文档元数据（文件名）
	for _, result := range results {
		if result.DocumentID != "" {
			doc, err := uc.DocumentRepo.GetByID(ctx, result.DocumentID)
			if err == nil {
				// 将文档信息添加到 metadata
				if result.Metadata == nil {
					result.Metadata = make(map[string]interface{})
				}
				result.Metadata["file_name"] = doc.FileName
			}
		}
	}

	// 计算分数统计
	var minScore, maxScore float32
	if len(results) > 0 {
		minScore = results[0].Score
		maxScore = results[0].Score
		for _, result := range results {
			if result.Score < minScore {
				minScore = result.Score
			}
			if result.Score > maxScore {
				maxScore = result.Score
			}
		}
	}

	// 记录搜索结果摘要
	searchType := "vector"
	if kb.EnableHybridSearch {
		searchType = "hybrid"
	}
	uc.logger.Info("知识库搜索结果",
		zap.String("kb_id", kbID),
		zap.Int("result_count", len(results)),
		zap.Float32("min_score", minScore),
		zap.Float32("max_score", maxScore),
		zap.String("search_type", searchType))

	return results, nil
}

// hybridSearch 混合检索（向量 + 关键词 + RRF）
func (uc *DocumentUseCase) hybridSearch(ctx context.Context, collection, kbID string, embedding []float32, query string, topK int, threshold float32) ([]*SearchResult, error) {
	// 1. 向量搜索（应用阈值过滤）
	vectorResults, err := uc.vectorDB.SearchWithThreshold(ctx, collection, embedding, topK*2, threshold) // 取2倍，融合后再截取
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// 2. 关键词搜索
	keywordChunks, err := uc.chunkRepo.KeywordSearch(ctx, kbID, query, topK*2)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}

	// 3. 转换为 RRF 输入格式
	vectorSearchResults := make([]hybrid.SearchResult, len(vectorResults))
	for i, result := range vectorResults {
		vectorSearchResults[i] = &hybrid.VectorSearchResult{
			ID:    result.DocumentID,
			Score: result.Score,
		}
	}

	keywordSearchResults := make([]hybrid.SearchResult, len(keywordChunks))
	for i, chunk := range keywordChunks {
		// 使用 BM25 分数（从 metadata 中获取）
		bm25Score := float32(1.0 / float64(i+1)) // 默认分数
		if chunk.Metadata != nil {
			if score, ok := chunk.Metadata["bm25_score"].(float32); ok {
				bm25Score = score
			} else if score, ok := chunk.Metadata["bm25_score"].(float64); ok {
				bm25Score = float32(score)
			}
		}

		keywordSearchResults[i] = &hybrid.KeywordSearchResult{
			ID:    chunk.DocumentID,
			Score: bm25Score,
		}
	}

	// 4. RRF 融合
	rrfResults := hybrid.ReciprocalRankFusion(
		[][]hybrid.SearchResult{vectorSearchResults, keywordSearchResults},
		60, // k=60，RRF 论文推荐值
	)

	// 5. 转换回 SearchResult 并限制数量
	if len(rrfResults) > topK {
		rrfResults = rrfResults[:topK]
	}

	finalResults := make([]*SearchResult, len(rrfResults))
	for i, rrfResult := range rrfResults {
		finalResults[i] = &SearchResult{
			DocumentID: rrfResult.ID,
			Score:      float32(rrfResult.RRFScore), // 使用 RRF 分数
			Content:    "",                          // 稍后从数据库填充
			Metadata:   make(map[string]interface{}),
		}
	}

	// 6. 填充内容（从向量搜索或关键词搜索结果中获取）
	contentMap := make(map[string]string)
	for _, result := range vectorResults {
		contentMap[result.DocumentID] = result.Content
	}
	for _, chunk := range keywordChunks {
		if _, exists := contentMap[chunk.DocumentID]; !exists {
			contentMap[chunk.DocumentID] = chunk.Content
		}
	}

	for _, result := range finalResults {
		if content, exists := contentMap[result.DocumentID]; exists {
			result.Content = content
		}
	}

	return finalResults, nil
}

// ReprocessDocument 重新处理文档
func (uc *DocumentUseCase) ReprocessDocument(ctx context.Context, documentID, userID string) error {
	// 获取文档
	doc, err := uc.DocumentRepo.GetByID(ctx, documentID)
	if err != nil {
		return fmt.Errorf("document not found: %w", err)
	}

	// 验证权限
	kb, err := uc.kbRepo.GetByID(ctx, doc.KnowledgeBaseID, "")
	if err != nil {
		return fmt.Errorf("knowledge base not found: %w", err)
	}

	if kb.OwnerID != userID && kb.OwnerID != SystemOwnerID {
		return fmt.Errorf("permission denied")
	}

	// 删除旧的向量和chunks
	_ = uc.vectorDB.DeleteByDocumentID(ctx, kb.MilvusCollection, documentID)
	_ = uc.chunkRepo.DeleteByDocumentID(ctx, documentID)

	// 重置状态
	err = uc.DocumentRepo.UpdateStatus(ctx, documentID, "pending", "")
	if err != nil {
		return fmt.Errorf("failed to reset status: %w", err)
	}

	// 重新处理
	return uc.ProcessDocument(ctx, documentID)
}

// Helper functions
func calculateSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

func getContentType(fileType string) string {
	switch fileType {
	case "pdf":
		return "application/pdf"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "txt":
		return "text/plain"
	case "md":
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}

// sanitizeUTF8 清理文本中的无效 UTF-8 字符
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	// 使用 strings.Builder 重建字符串，替换无效字符
	var builder strings.Builder
	builder.Grow(len(s))

	for _, r := range s {
		if r == utf8.RuneError {
			// 用空格替换无效字符
			builder.WriteRune(' ')
		} else {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}
