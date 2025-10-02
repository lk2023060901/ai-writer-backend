package data

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
)

// DocumentPO 文档数据库模型
type DocumentPO struct {
	ID              string    `gorm:"type:uuid;primarykey"`
	KnowledgeBaseID string    `gorm:"column:knowledge_base_id;type:uuid;not null;index:idx_documents_kb_id"`
	FileName        string    `gorm:"size:255;not null"`
	FileType        string    `gorm:"size:50;not null"`
	FileSize        int64     `gorm:"not null"`
	FilePath        string    `gorm:"size:1024;not null"`
	ProcessStatus   string    `gorm:"size:50;not null;index:idx_documents_status"`
	ProcessError    string    `gorm:"type:text"`
	ChunkCount      int64     `gorm:"not null;default:0"`
	CreatedAt       time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt       time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (DocumentPO) TableName() string {
	return "documents"
}

// DocumentRepo 文档仓储实现
type DocumentRepo struct {
	db *database.DB
}

// NewDocumentRepo 创建文档仓储
func NewDocumentRepo(db *database.DB) *DocumentRepo {
	return &DocumentRepo{db: db}
}

// Create 创建文档
func (r *DocumentRepo) Create(ctx context.Context, doc *biz.Document) error {
	po := &DocumentPO{
		ID:              doc.ID,
		KnowledgeBaseID: doc.KnowledgeBaseID,
		FileName:        doc.FileName,
		FileType:        doc.FileType,
		FileSize:        doc.FileSize,
		FilePath:        doc.FilePath,
		ProcessStatus:   doc.ProcessStatus,
		ProcessError:    doc.ProcessError,
		ChunkCount:      doc.ChunkCount,
		CreatedAt:       doc.CreatedAt,
		UpdatedAt:       doc.UpdatedAt,
	}

	err := r.db.WithContext(ctx).GetDB().Create(po).Error
	if err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取文档
func (r *DocumentRepo) GetByID(ctx context.Context, id string) (*biz.Document, error) {
	var po DocumentPO
	err := r.db.WithContext(ctx).GetDB().Where("id = ?", id).First(&po).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return r.toDomain(&po), nil
}

// List 列出文档
func (r *DocumentRepo) List(ctx context.Context, kbID string, req *biz.ListDocumentsRequest) ([]*biz.Document, int64, error) {
	var pos []DocumentPO
	var total int64

	query := r.db.WithContext(ctx).GetDB().Where("knowledge_base_id = ?", kbID)

	// 计数
	err := query.Model(&DocumentPO{}).Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count documents: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	err = query.Order("created_at DESC").
		Limit(req.PageSize).
		Offset(offset).
		Find(&pos).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list documents: %w", err)
	}

	docs := make([]*biz.Document, len(pos))
	for i, po := range pos {
		docs[i] = r.toDomain(&po)
	}

	return docs, total, nil
}

// Update 更新文档
func (r *DocumentRepo) Update(ctx context.Context, doc *biz.Document) error {
	po := &DocumentPO{
		ID:              doc.ID,
		KnowledgeBaseID: doc.KnowledgeBaseID,
		FileName:        doc.FileName,
		FileType:        doc.FileType,
		FileSize:        doc.FileSize,
		FilePath:        doc.FilePath,
		ProcessStatus:   doc.ProcessStatus,
		ProcessError:    doc.ProcessError,
		ChunkCount:      doc.ChunkCount,
		UpdatedAt:       time.Now(),
	}

	err := r.db.WithContext(ctx).GetDB().Save(po).Error
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	return nil
}

// Delete 删除文档
func (r *DocumentRepo) Delete(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).GetDB().Where("id = ?", id).Delete(&DocumentPO{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// UpdateStatus 更新文档状态
func (r *DocumentRepo) UpdateStatus(ctx context.Context, id, status, errorMsg string) error {
	updates := map[string]interface{}{
		"process_status": status,
		"process_error":  errorMsg,
		"updated_at":     time.Now(),
	}

	err := r.db.WithContext(ctx).GetDB().Model(&DocumentPO{}).
		Where("id = ?", id).
		Updates(updates).Error

	if err != nil {
		return fmt.Errorf("failed to update document status: %w", err)
	}

	return nil
}

// toDomain 转换为领域模型
func (r *DocumentRepo) toDomain(po *DocumentPO) *biz.Document {
	return &biz.Document{
		ID:              po.ID,
		KnowledgeBaseID: po.KnowledgeBaseID,
		FileName:        po.FileName,
		FileType:        po.FileType,
		FileSize:        po.FileSize,
		FilePath:        po.FilePath,
		ProcessStatus:   po.ProcessStatus,
		ProcessError:    po.ProcessError,
		ChunkCount:      po.ChunkCount,
		CreatedAt:       po.CreatedAt,
		UpdatedAt:       po.UpdatedAt,
	}
}

// ChunkPO 文档分块数据库模型
type ChunkPO struct {
	ID              string    `gorm:"type:uuid;primarykey"`
	DocumentID      string    `gorm:"column:document_id;type:uuid;not null;index:idx_chunks_document_id"`
	KnowledgeBaseID string    `gorm:"column:knowledge_base_id;type:uuid;not null;index:idx_chunks_kb_id"`
	Content         string    `gorm:"type:text;not null"`
	Position        int       `gorm:"not null"`
	TokenCount      int       `gorm:"not null"`
	CreatedAt       time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (ChunkPO) TableName() string {
	return "chunks"
}

// ChunkRepo 分块仓储实现
type ChunkRepo struct {
	db *database.DB
}

// NewChunkRepo 创建分块仓储
func NewChunkRepo(db *database.DB) *ChunkRepo {
	return &ChunkRepo{db: db}
}

// BatchCreate 批量创建分块
func (r *ChunkRepo) BatchCreate(ctx context.Context, chunks []*biz.Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	pos := make([]ChunkPO, len(chunks))
	for i, chunk := range chunks {
		pos[i] = ChunkPO{
			ID:              chunk.ID,
			DocumentID:      chunk.DocumentID,
			KnowledgeBaseID: chunk.KnowledgeBaseID,
			Content:         chunk.Content,
			Position:        chunk.Position,
			TokenCount:      chunk.TokenCount,
			CreatedAt:       chunk.CreatedAt,
		}
	}

	err := r.db.WithContext(ctx).GetDB().CreateInBatches(pos, 100).Error
	if err != nil {
		return fmt.Errorf("failed to batch create chunks: %w", err)
	}

	return nil
}

// GetByDocumentID 根据文档 ID 获取分块
func (r *ChunkRepo) GetByDocumentID(ctx context.Context, docID string) ([]*biz.Chunk, error) {
	var pos []ChunkPO
	err := r.db.WithContext(ctx).GetDB().
		Where("document_id = ?", docID).
		Order("position ASC").
		Find(&pos).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get chunks: %w", err)
	}

	chunks := make([]*biz.Chunk, len(pos))
	for i, po := range pos {
		chunks[i] = &biz.Chunk{
			ID:              po.ID,
			DocumentID:      po.DocumentID,
			KnowledgeBaseID: po.KnowledgeBaseID,
			Content:         po.Content,
			Position:        po.Position,
			TokenCount:      po.TokenCount,
			CreatedAt:       po.CreatedAt,
		}
	}

	return chunks, nil
}

// DeleteByDocumentID 根据文档 ID 删除分块
func (r *ChunkRepo) DeleteByDocumentID(ctx context.Context, docID string) error {
	err := r.db.WithContext(ctx).GetDB().
		Where("document_id = ?", docID).
		Delete(&ChunkPO{}).Error

	if err != nil {
		return fmt.Errorf("failed to delete chunks: %w", err)
	}

	return nil
}

// DeleteByKnowledgeBaseID 根据知识库 ID 删除分块
func (r *ChunkRepo) DeleteByKnowledgeBaseID(ctx context.Context, kbID string) error {
	err := r.db.WithContext(ctx).GetDB().
		Where("knowledge_base_id = ?", kbID).
		Delete(&ChunkPO{}).Error

	if err != nil {
		return fmt.Errorf("failed to delete chunks by kb id: %w", err)
	}

	return nil
}
