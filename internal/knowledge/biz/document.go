package biz

import (
	"fmt"
	"github.com/google/uuid"
	"context"
	"time"
)

// Document 文档模型
type Document struct {
	ID              string
	KnowledgeBaseID string
	FileName        string
	FileType        string // pdf, docx, txt, md
	FileSize        int64
	FilePath        string // MinIO 路径
	ProcessStatus   string // pending, processing, completed, failed
	ProcessError    string
	ChunkCount      int64
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
	DocumentRepo DocumentRepo
	chunkRepo     ChunkRepo
	kbRepo        KnowledgeBaseRepo
	aiConfigRepo  AIProviderConfigRepo
	storage       StorageService
	vectorDB      VectorDBService
	embedder      EmbeddingService
	processor     DocumentProcessor
}

// DocumentRepo 文档仓储接口
type DocumentRepo interface {
	Create(ctx context.Context, doc *Document) error
	GetByID(ctx context.Context, id string) (*Document, error)
	List(ctx context.Context, kbID string, req *ListDocumentsRequest) ([]*Document, int64, error)
	Update(ctx context.Context, doc *Document) error
	Delete(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id, status, errorMsg string) error
}

// ChunkRepo 分块仓储接口
type ChunkRepo interface {
	BatchCreate(ctx context.Context, chunks []*Chunk) error
	GetByDocumentID(ctx context.Context, docID string) ([]*Chunk, error)
	DeleteByDocumentID(ctx context.Context, docID string) error
	DeleteByKnowledgeBaseID(ctx context.Context, kbID string) error
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
	DeleteByDocumentID(ctx context.Context, collectionName, documentID string) error
	DropCollection(ctx context.Context, collectionName string) error
}

// EmbeddingService Embedding 生成服务接口
type EmbeddingService interface {
	GenerateEmbeddings(ctx context.Context, texts []string, config *AIProviderConfig) ([][]float32, error)
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
	aiConfigRepo AIProviderConfigRepo,
	storage StorageService,
	vectorDB VectorDBService,
	embedder EmbeddingService,
	processor DocumentProcessor,
) *DocumentUseCase {
	return &DocumentUseCase{
		DocumentRepo:  documentRepo,
		chunkRepo:     chunkRepo,
		kbRepo:        kbRepo,
		aiConfigRepo:  aiConfigRepo,
		storage:       storage,
		vectorDB:      vectorDB,
		embedder:      embedder,
		processor:     processor,
	}
}
// UploadDocument 上传文档
func (uc *DocumentUseCase) UploadDocument(ctx context.Context, kbID, userID string, fileName string, fileData []byte, fileType string) (*Document, error) {
	// 验证知识库权限
	kb, err := uc.kbRepo.GetByID(ctx, kbID, userID)
	if err != nil {
		return nil, fmt.Errorf("knowledge base not found: %w", err)
	}

	if kb.OwnerID != userID && kb.OwnerID != SystemOwnerID {
		return nil, fmt.Errorf("permission denied")
	}

	// 生成文档ID和文件路径
	docID := uuid.New().String()
	objectName := fmt.Sprintf("documents/%s/%s/%s", kbID, docID, fileName)

	// 上传到MinIO
	_, err = uc.storage.UploadFile(ctx, "", objectName, fileData, getContentType(fileType))
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// 创建文档记录
	doc := &Document{
		ID:              docID,
		KnowledgeBaseID: kbID,
		FileName:        fileName,
		FileType:        fileType,
		FileSize:        int64(len(fileData)),
		FilePath:        objectName,
		ProcessStatus:   "pending",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err = uc.DocumentRepo.Create(ctx, doc)
	if err != nil {
		// 清理MinIO文件
		_ = uc.storage.DeleteFile(ctx, "", objectName)
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

	// 获取AI配置
	aiConfig, err := uc.aiConfigRepo.GetByID(ctx, kb.AIProviderConfigID, kb.OwnerID)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", "AI config not found")
		return fmt.Errorf("AI config not found: %w", err)
	}

	// 从MinIO获取文件
	fileData, err := uc.storage.GetFile(ctx, "", doc.FilePath)
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
	embeddings, err := uc.embedder.GenerateEmbeddings(ctx, chunkTexts, aiConfig)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", fmt.Sprintf("failed to generate embeddings: %v", err))
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// 创建 Chunks
	chunks := make([]*Chunk, len(chunkTexts))
	for i, chunkText := range chunkTexts {
		chunks[i] = &Chunk{
			ID:              uuid.New().String(),
			DocumentID:      documentID,
			KnowledgeBaseID: doc.KnowledgeBaseID,
			Content:         chunkText,
			Position:        i,
			TokenCount:      len(chunkText) / 4, // 粗略估算
			Embedding:       embeddings[i],
			CreatedAt:       time.Now(),
		}
	}

	// 保存到数据库
	err = uc.chunkRepo.BatchCreate(ctx, chunks)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", fmt.Sprintf("failed to save chunks: %v", err))
		return fmt.Errorf("failed to save chunks: %w", err)
	}

	// 确保 Milvus collection 存在
	collectionName := kb.MilvusCollection
	err = uc.vectorDB.CreateCollection(ctx, collectionName, aiConfig.EmbeddingDimensions)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", fmt.Sprintf("failed to create collection: %v", err))
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// 插入向量到 Milvus
	err = uc.vectorDB.InsertVectors(ctx, collectionName, chunks)
	if err != nil {
		_ = uc.DocumentRepo.UpdateStatus(ctx, documentID, "failed", fmt.Sprintf("failed to insert vectors: %v", err))
		return fmt.Errorf("failed to insert vectors: %w", err)
	}

	// 更新文档状态
	doc.ProcessStatus = "completed"
	doc.ChunkCount = int64(len(chunks))
	doc.UpdatedAt = time.Now()
	err = uc.DocumentRepo.Update(ctx, doc)
	if err != nil {
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

	// 删除 MinIO 文件
	_ = uc.storage.DeleteFile(ctx, "", doc.FilePath)

	// 删除文档记录
	err = uc.DocumentRepo.Delete(ctx, documentID)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// SearchDocuments 向量搜索
func (uc *DocumentUseCase) SearchDocuments(ctx context.Context, kbID, userID, query string, topK int) ([]*SearchResult, error) {
	// 验证权限
	kb, err := uc.kbRepo.GetByID(ctx, kbID, userID)
	if err != nil {
		return nil, fmt.Errorf("knowledge base not found: %w", err)
	}

	if kb.OwnerID != userID && kb.OwnerID != SystemOwnerID {
		return nil, fmt.Errorf("permission denied")
	}

	// 获取AI配置
	aiConfig, err := uc.aiConfigRepo.GetByID(ctx, kb.AIProviderConfigID, kb.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("AI config not found: %w", err)
	}

	// 生成查询的 embedding
	embeddings, err := uc.embedder.GenerateEmbeddings(ctx, []string{query}, aiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// 向量搜索
	results, err := uc.vectorDB.Search(ctx, kb.MilvusCollection, embeddings[0], topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	return results, nil
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
