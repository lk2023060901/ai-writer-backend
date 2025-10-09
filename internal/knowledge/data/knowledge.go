package data

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
)

// KnowledgeBasePO 知识库数据库模型
type KnowledgeBasePO struct {
	ID               string    `gorm:"type:uuid;primarykey"`
	OwnerID          string    `gorm:"type:uuid;not null;index:idx_knowledge_bases_owner_id"`
	Name             string    `gorm:"size:255;not null"`
	EmbeddingModelID string    `gorm:"type:uuid;not null;index:idx_kb_embedding_model"`
	RerankModelID    *string   `gorm:"type:uuid;index:idx_kb_rerank_model"`
	ChunkSize        int       `gorm:"not null;default:512"`
	ChunkOverlap     int       `gorm:"not null;default:50"`
	ChunkStrategy    string    `gorm:"size:50;not null;default:'recursive'"`
	MilvusCollection string    `gorm:"size:255;not null"`
	DocumentCount    int64     `gorm:"not null;default:0"`

	// 检索配置
	Threshold           float32 `gorm:"type:real;not null;default:0.0"`
	TopK                int     `gorm:"not null;default:5"`
	EnableHybridSearch  bool    `gorm:"not null;default:false"`

	CreatedAt        time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt        time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (KnowledgeBasePO) TableName() string {
	return "knowledge_bases"
}

// KnowledgeBaseRepo 知识库仓储实现
type KnowledgeBaseRepo struct {
	db *database.DB
}

// NewKnowledgeBaseRepo 创建知识库仓储
func NewKnowledgeBaseRepo(db *database.DB) biz.KnowledgeBaseRepo {
	return &KnowledgeBaseRepo{db: db}
}

// Create 创建知识库
func (r *KnowledgeBaseRepo) Create(ctx context.Context, kb *biz.KnowledgeBase) error {
	po := &KnowledgeBasePO{
		ID:               kb.ID,
		OwnerID:          kb.OwnerID,
		Name:             kb.Name,
		EmbeddingModelID: kb.EmbeddingModelID,
		RerankModelID:    kb.RerankModelID,
		ChunkSize:        kb.ChunkSize,
		ChunkOverlap:     kb.ChunkOverlap,
		ChunkStrategy:    kb.ChunkStrategy,
		MilvusCollection: kb.MilvusCollection,
		DocumentCount:    kb.DocumentCount,
		Threshold:        kb.Threshold,
		TopK:             kb.TopK,
		EnableHybridSearch: kb.EnableHybridSearch,
		CreatedAt:        kb.CreatedAt,
		UpdatedAt:        kb.UpdatedAt,
	}

	return r.db.WithContext(ctx).GetDB().Create(po).Error
}

// GetByID 根据ID获取知识库（需要验证权限：官方知识库或用户自己的知识库）
func (r *KnowledgeBaseRepo) GetByID(ctx context.Context, id string, userID string) (*biz.KnowledgeBase, error) {
	var po KnowledgeBasePO
	query := r.db.WithContext(ctx).GetDB().Where("id = ?", id)

	// 如果提供了 userID，则验证权限
	if userID != "" {
		query = query.Where("owner_id = ? OR owner_id = ?", biz.SystemOwnerID, userID)
	}

	err := query.First(&po).Error

	if err != nil {
		if database.IsRecordNotFoundError(err) {
			return nil, biz.ErrKnowledgeBaseNotFound
		}
		return nil, err
	}

	return r.toKnowledgeBase(&po), nil
}

// List 获取知识库列表（官方 + 用户自己的，支持分页）
func (r *KnowledgeBaseRepo) List(ctx context.Context, req *biz.ListKnowledgeBasesRequest) ([]*biz.KnowledgeBase, int64, error) {
	var pos []KnowledgeBasePO
	var total int64

	query := r.db.WithContext(ctx).GetDB().Model(&KnowledgeBasePO{}).
		Where("owner_id = ? OR owner_id = ?", biz.SystemOwnerID, req.UserID)

	// 关键词搜索（按名称）
	if req.Keyword != "" {
		query = query.Where("name ILIKE ?", "%"+req.Keyword+"%")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	err := query.
		Order("CASE WHEN owner_id = '" + biz.SystemOwnerID + "' THEN 0 ELSE 1 END, created_at DESC").
		Limit(req.PageSize).
		Offset(offset).
		Find(&pos).Error

	if err != nil {
		return nil, 0, err
	}

	kbs := make([]*biz.KnowledgeBase, len(pos))
	for i, po := range pos {
		kbs[i] = r.toKnowledgeBase(&po)
	}

	return kbs, total, nil
}

// Update 更新知识库
func (r *KnowledgeBaseRepo) Update(ctx context.Context, kb *biz.KnowledgeBase) error {
	updates := map[string]interface{}{
		"name":                 kb.Name,
		"threshold":            kb.Threshold,
		"top_k":                kb.TopK,
		"enable_hybrid_search": kb.EnableHybridSearch,
		"updated_at":           kb.UpdatedAt,
	}

	result := r.db.WithContext(ctx).GetDB().
		Model(&KnowledgeBasePO{}).
		Where("id = ? AND owner_id = ?", kb.ID, kb.OwnerID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return biz.ErrKnowledgeBaseNotFound
	}

	return nil
}

// Delete 删除知识库
func (r *KnowledgeBaseRepo) Delete(ctx context.Context, id string, ownerID string) error {
	result := r.db.WithContext(ctx).GetDB().
		Where("id = ? AND owner_id = ?", id, ownerID).
		Delete(&KnowledgeBasePO{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return biz.ErrKnowledgeBaseNotFound
	}

	return nil
}

// IncrementDocumentCount 增加/减少文档数量
func (r *KnowledgeBaseRepo) IncrementDocumentCount(ctx context.Context, id string, delta int) error {
	result := r.db.WithContext(ctx).GetDB().
		Exec("UPDATE knowledge_bases SET document_count = document_count + ? WHERE id = ?", delta, id)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return biz.ErrKnowledgeBaseNotFound
	}

	return nil
}

// BatchUpdateDocumentCounts 批量更新多个知识库的文档计数
func (r *KnowledgeBaseRepo) BatchUpdateDocumentCounts(ctx context.Context, deltas map[string]int) error {
	if len(deltas) == 0 {
		return nil
	}

	// 使用事务批量更新
	tx := r.db.WithContext(ctx).GetDB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for kbID, delta := range deltas {
		if err := tx.Exec("UPDATE knowledge_bases SET document_count = document_count + ? WHERE id = ?", delta, kbID).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update knowledge base %s: %w", kbID, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// toKnowledgeBase 转换 PO 到业务对象
func (r *KnowledgeBaseRepo) toKnowledgeBase(po *KnowledgeBasePO) *biz.KnowledgeBase {
	return &biz.KnowledgeBase{
		ID:               po.ID,
		OwnerID:          po.OwnerID,
		Name:             po.Name,
		EmbeddingModelID: po.EmbeddingModelID,
		RerankModelID:    po.RerankModelID,
		ChunkSize:        po.ChunkSize,
		ChunkOverlap:     po.ChunkOverlap,
		ChunkStrategy:    po.ChunkStrategy,
		MilvusCollection: po.MilvusCollection,
		DocumentCount:    po.DocumentCount,
		Threshold:        po.Threshold,
		TopK:             po.TopK,
		EnableHybridSearch: po.EnableHybridSearch,
		CreatedAt:        po.CreatedAt,
		UpdatedAt:        po.UpdatedAt,
	}
}
