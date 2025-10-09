package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// getDefaultChunkSize 根据嵌入模型返回默认 chunkSize
// 参考 Cherry Studio 的 embedings.ts 配置
func getDefaultChunkSize(embeddingModel string) int {
	// 映射常见嵌入模型的 max_context
	modelMaxContext := map[string]int{
		"BAAI/bge-m3":                  8191,
		"Pro/BAAI/bge-m3":              8191,
		"text-embedding-3-small":       8191,
		"text-embedding-3-large":       8191,
		"text-embedding-ada-002":       8191,
		"jina-embeddings-v3":           8191,
		"nomic-embed-text-v1":          8192,
		"nomic-embed-text-v1.5":        8192,
		"gte-multilingual-base":        8192,
		"embedding-query":              4000,
		"embedding-passage":            4000,
		"BAAI/bge-large-zh-v1.5":       512,
		"BAAI/bge-large-en-v1.5":       512,
		"netease-youdao/bce-embedding": 512,
		"embed-english-v3.0":           512,
		"embed-english-light-v3.0":     512,
	}

	if maxContext, ok := modelMaxContext[embeddingModel]; ok {
		return maxContext
	}

	// 默认返回 512
	return 512
}

// KnowledgeBase 知识库业务对象
type KnowledgeBase struct {
	ID               string
	OwnerID          string  // SystemOwnerID = 官方
	Name             string
	EmbeddingModelID string  // 使用的 Embedding 模型 ID
	RerankModelID    *string // 使用的 Rerank 模型 ID（可选）
	ChunkSize        int
	ChunkOverlap     int
	ChunkStrategy    string
	MilvusCollection string
	DocumentCount    int64

	// 检索配置
	Threshold           float32 // 相似度阈值（0.0-1.0），用于过滤低相关性结果，默认 0.0（不过滤）
	TopK                int     // 返回文档数量，默认 5
	EnableHybridSearch  bool    // 是否启用混合检索，默认 false

	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// IsOfficial 是否为官方知识库
func (kb *KnowledgeBase) IsOfficial() bool {
	return kb.OwnerID == SystemOwnerID
}

// KnowledgeBaseRepo 知识库仓储接口
type KnowledgeBaseRepo interface {
	Create(ctx context.Context, kb *KnowledgeBase) error
	GetByID(ctx context.Context, id string, userID string) (*KnowledgeBase, error)
	List(ctx context.Context, req *ListKnowledgeBasesRequest) ([]*KnowledgeBase, int64, error)
	Update(ctx context.Context, kb *KnowledgeBase) error
	Delete(ctx context.Context, id string, ownerID string) error
	IncrementDocumentCount(ctx context.Context, id string, delta int) error
	BatchUpdateDocumentCounts(ctx context.Context, deltas map[string]int) error  // 批量更新文档计数
}

// CreateKnowledgeBaseRequest 创建知识库请求
type CreateKnowledgeBaseRequest struct {
	Name             string
	EmbeddingModelID string   // 必填，Embedding 模型 ID
	RerankModelID    *string  // 可选，Rerank 模型 ID
	ChunkSize        *int     // 可选，不传则根据嵌入模型 max_context 自动设置
	ChunkOverlap     *int     // 可选，不传则为 0（不重叠）
	ChunkStrategy    *string  // 可选，不传则为 "recursive"
	Threshold        *float32 // 可选，相似度阈值（0.0-1.0），默认 0.0（不过滤）
	TopK             *int     // 可选，返回文档数量（1-20），默认 5
	EnableHybridSearch *bool  // 可选，是否启用混合检索，默认 false
}

// UpdateKnowledgeBaseRequest 更新知识库请求
type UpdateKnowledgeBaseRequest struct {
	Name               *string
	Threshold          *float32 // 可选，相似度阈值
	TopK               *int     // 可选，返回文档数量
	EnableHybridSearch *bool    // 可选，是否启用混合检索
}

// ListKnowledgeBasesRequest 知识库列表请求
type ListKnowledgeBasesRequest struct {
	UserID   string
	Keyword  string
	Page     int
	PageSize int
}

// KnowledgeBaseUseCase 知识库用例
type KnowledgeBaseUseCase struct {
	kbRepo      KnowledgeBaseRepo
	aiModelRepo AIModelRepo
}

// NewKnowledgeBaseUseCase 创建知识库用例
func NewKnowledgeBaseUseCase(
	kbRepo KnowledgeBaseRepo,
	aiModelRepo AIModelRepo,
) *KnowledgeBaseUseCase {
	return &KnowledgeBaseUseCase{
		kbRepo:      kbRepo,
		aiModelRepo: aiModelRepo,
	}
}

// CreateKnowledgeBase 创建知识库
func (uc *KnowledgeBaseUseCase) CreateKnowledgeBase(
	ctx context.Context,
	userID string,
	req *CreateKnowledgeBaseRequest,
) (*KnowledgeBase, error) {
	// 验证必填字段
	if req.Name == "" {
		return nil, ErrKnowledgeBaseNameRequired
	}
	if req.EmbeddingModelID == "" {
		return nil, ErrAIProviderNotFound
	}

	// 1. 获取 AI Model 信息
	aiModel, err := uc.aiModelRepo.GetByID(ctx, req.EmbeddingModelID)
	if err != nil {
		return nil, err
	}

	// 2. 设置默认值（类似 Cherry Studio 的逻辑）
	var chunkSize int
	if req.ChunkSize != nil {
		chunkSize = *req.ChunkSize
	} else {
		// 根据嵌入模型设置默认 chunkSize
		// BAAI/bge-m3: 8191, text-embedding-3-small: 8191, 其他默认 512
		chunkSize = getDefaultChunkSize(aiModel.ModelName)
	}

	var chunkOverlap int
	if req.ChunkOverlap != nil {
		chunkOverlap = *req.ChunkOverlap
	} else {
		// 不传则为 0（不重叠），类似 Cherry Studio
		chunkOverlap = 0
	}

	var chunkStrategy string
	if req.ChunkStrategy != nil {
		chunkStrategy = *req.ChunkStrategy
	} else {
		chunkStrategy = "recursive"
	}

	// 设置检索配置默认值
	threshold := float32(0.0) // 默认不过滤任何结果
	if req.Threshold != nil {
		threshold = *req.Threshold
	}

	topK := 5
	if req.TopK != nil {
		topK = *req.TopK
	}

	enableHybridSearch := false
	if req.EnableHybridSearch != nil {
		enableHybridSearch = *req.EnableHybridSearch
	}

	// 验证参数
	if chunkSize < 100 || chunkSize > 10000 {
		return nil, ErrKnowledgeBaseInvalidChunkSize
	}
	if chunkOverlap < 0 || chunkOverlap >= chunkSize {
		return nil, ErrKnowledgeBaseInvalidOverlap
	}
	if threshold < 0.0 || threshold > 1.0 {
		return nil, fmt.Errorf("threshold must be between 0.0 and 1.0")
	}
	if topK < 1 || topK > 20 {
		return nil, fmt.Errorf("top_k must be between 1 and 20")
	}

	// 3. 生成 Milvus Collection 名称
	collectionName := fmt.Sprintf("kb_%s_%s",
		userID[:8], uuid.New().String()[:8])

	// 4. 【阶段 3】在 Milvus 创建 Collection
	// 当前阶段跳过，仅生成名称

	// 5. 创建知识库
	now := time.Now()
	kb := &KnowledgeBase{
		ID:               uuid.New().String(),
		OwnerID:          userID,
		Name:             req.Name,
		EmbeddingModelID: req.EmbeddingModelID,
		RerankModelID:    req.RerankModelID,
		ChunkSize:        chunkSize,
		ChunkOverlap:     chunkOverlap,
		ChunkStrategy:    chunkStrategy,
		MilvusCollection: collectionName,
		DocumentCount:    0,
		Threshold:        threshold,
		TopK:             topK,
		EnableHybridSearch: enableHybridSearch,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := uc.kbRepo.Create(ctx, kb); err != nil {
		return nil, err
	}

	return kb, nil
}

// GetKnowledgeBase 获取知识库
func (uc *KnowledgeBaseUseCase) GetKnowledgeBase(
	ctx context.Context,
	id string,
	userID string,
) (*KnowledgeBase, error) {
	return uc.kbRepo.GetByID(ctx, id, userID)
}

// ListKnowledgeBases 获取知识库列表
func (uc *KnowledgeBaseUseCase) ListKnowledgeBases(
	ctx context.Context,
	userID string,
	req *ListKnowledgeBasesRequest,
) ([]*KnowledgeBase, int64, error) {
	req.UserID = userID
	return uc.kbRepo.List(ctx, req)
}

// UpdateKnowledgeBase 更新知识库
func (uc *KnowledgeBaseUseCase) UpdateKnowledgeBase(
	ctx context.Context,
	id string,
	userID string,
	req *UpdateKnowledgeBaseRequest,
) (*KnowledgeBase, error) {
	// 获取知识库
	kb, err := uc.kbRepo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// 权限检查：不能编辑官方知识库
	if kb.IsOfficial() {
		return nil, ErrCannotEditOfficialResource
	}

	// 权限检查：只能编辑自己的知识库
	if kb.OwnerID != userID {
		return nil, ErrUnauthorized
	}

	// 更新字段
	if req.Name != nil {
		kb.Name = *req.Name
	}

	// 更新检索配置
	if req.Threshold != nil {
		if *req.Threshold < 0.0 || *req.Threshold > 1.0 {
			return nil, fmt.Errorf("threshold must be between 0.0 and 1.0")
		}
		kb.Threshold = *req.Threshold
	}

	if req.TopK != nil {
		if *req.TopK < 1 || *req.TopK > 20 {
			return nil, fmt.Errorf("top_k must be between 1 and 20")
		}
		kb.TopK = *req.TopK
	}

	if req.EnableHybridSearch != nil {
		kb.EnableHybridSearch = *req.EnableHybridSearch
	}

	kb.UpdatedAt = time.Now()

	if err := uc.kbRepo.Update(ctx, kb); err != nil {
		return nil, err
	}

	return kb, nil
}

// DeleteKnowledgeBase 删除知识库
func (uc *KnowledgeBaseUseCase) DeleteKnowledgeBase(
	ctx context.Context,
	id string,
	userID string,
) error {
	// 获取知识库
	kb, err := uc.kbRepo.GetByID(ctx, id, userID)
	if err != nil {
		return err
	}

	// 权限检查：不能删除官方知识库
	if kb.IsOfficial() {
		return ErrCannotDeleteOfficialResource
	}

	// 权限检查：只能删除自己的知识库
	if kb.OwnerID != userID {
		return ErrUnauthorized
	}

	// 【阶段 3】删除 Milvus Collection
	// 当前阶段跳过

	return uc.kbRepo.Delete(ctx, id, userID)
}
