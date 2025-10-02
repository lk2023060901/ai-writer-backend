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
	ID                 string
	OwnerID            string // SystemOwnerID = 官方
	Name               string
	AIProviderConfigID string
	ChunkSize          int
	ChunkOverlap       int
	ChunkStrategy      string
	MilvusCollection   string
	DocumentCount      int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
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
}

// CreateKnowledgeBaseRequest 创建知识库请求
type CreateKnowledgeBaseRequest struct {
	Name               string
	AIProviderConfigID string  // 可选，不指定则自动选择
	ChunkSize          *int    // 可选，不传则根据嵌入模型 max_context 自动设置
	ChunkOverlap       *int    // 可选，不传则为 0（不重叠）
	ChunkStrategy      *string // 可选，不传则为 "recursive"
}

// UpdateKnowledgeBaseRequest 更新知识库请求
type UpdateKnowledgeBaseRequest struct {
	Name *string
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
	kbRepo       KnowledgeBaseRepo
	aiConfigRepo AIProviderConfigRepo
}

// NewKnowledgeBaseUseCase 创建知识库用例
func NewKnowledgeBaseUseCase(
	kbRepo KnowledgeBaseRepo,
	aiConfigRepo AIProviderConfigRepo,
) *KnowledgeBaseUseCase {
	return &KnowledgeBaseUseCase{
		kbRepo:       kbRepo,
		aiConfigRepo: aiConfigRepo,
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

	// 1. 解析 AI 配置
	var aiConfig *AIProviderConfig
	var err error

	if req.AIProviderConfigID != "" {
		// 使用指定的配置
		aiConfig, err = uc.aiConfigRepo.GetByID(ctx, req.AIProviderConfigID, userID)
		if err != nil {
			return nil, err
		}
		// 验证：必须是官方配置 或 用户自己的配置
		if !aiConfig.IsOfficial() && aiConfig.OwnerID != userID {
			return nil, ErrUnauthorized
		}
	} else {
		// 自动选择可用配置
		aiConfig, err = uc.aiConfigRepo.GetFirstAvailable(ctx, userID)
		if err != nil {
			return nil, err
		}
	}

	// 2. 设置默认值（类似 Cherry Studio 的逻辑）
	var chunkSize int
	if req.ChunkSize != nil {
		chunkSize = *req.ChunkSize
	} else {
		// 根据嵌入模型设置默认 chunkSize
		// BAAI/bge-m3: 8191, text-embedding-3-small: 8191, 其他默认 512
		chunkSize = getDefaultChunkSize(aiConfig.EmbeddingModel)
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

	// 验证参数
	if chunkSize < 100 || chunkSize > 10000 {
		return nil, ErrKnowledgeBaseInvalidChunkSize
	}
	if chunkOverlap < 0 || chunkOverlap >= chunkSize {
		return nil, ErrKnowledgeBaseInvalidOverlap
	}

	// 3. 生成 Milvus Collection 名称
	collectionName := fmt.Sprintf("kb_%s_%s",
		userID[:8], uuid.New().String()[:8])

	// 4. 【阶段 3】在 Milvus 创建 Collection
	// 当前阶段跳过，仅生成名称

	// 5. 创建知识库
	now := time.Now()
	kb := &KnowledgeBase{
		ID:                 uuid.New().String(),
		OwnerID:            userID,
		Name:               req.Name,
		AIProviderConfigID: aiConfig.ID,
		ChunkSize:          chunkSize,
		ChunkOverlap:       chunkOverlap,
		ChunkStrategy:      chunkStrategy,
		MilvusCollection:   collectionName,
		DocumentCount:      0,
		CreatedAt:          now,
		UpdatedAt:          now,
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
