package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
	"gorm.io/gorm/clause"
)

// DocumentPO 文档数据库模型
type DocumentPO struct {
	ID              string    `gorm:"type:uuid;primarykey"`
	KnowledgeBaseID string    `gorm:"column:knowledge_base_id;type:uuid;not null;index:idx_doc_kb_id"`
	FileName        string    `gorm:"column:filename;size:255;not null"`
	FileType        string    `gorm:"column:file_type;size:50;not null;index:idx_doc_file_type"`
	FileSize        int64     `gorm:"column:file_size;not null"`
	FileHash        string    `gorm:"column:file_hash;size:64;not null;index:idx_doc_file_hash"`
	MinioBucket     string    `gorm:"column:minio_bucket;size:100;not null"`
	MinioObjectKey  string    `gorm:"column:minio_object_key;size:500;not null"`
	ProcessStatus   string    `gorm:"column:status;size:50;not null;index:idx_doc_status;default:'pending'"`
	ProcessError    string    `gorm:"column:error_message;type:text"`
	ChunkCount      int64     `gorm:"column:chunk_count;not null;default:0"`
	TokenCount      int       `gorm:"column:token_count;not null;default:0"`
	Metadata        string    `gorm:"column:metadata;type:jsonb"`

	// 多模态支持
	SourceType    string `gorm:"column:source_type;size:20;not null;default:'file'"`
	SourceURL     string `gorm:"column:source_url;type:text"`
	SourceContent string `gorm:"column:source_content;type:text"`

	CreatedAt       time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
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
	// 序列化Metadata
	metadataJSON := "{}"
	if doc.Metadata != nil && len(doc.Metadata) > 0 {
		bytes, err := json.Marshal(doc.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(bytes)
	}

	po := &DocumentPO{
		ID:              doc.ID,
		KnowledgeBaseID: doc.KnowledgeBaseID,
		FileName:        doc.FileName,
		FileType:        doc.FileType,
		FileSize:        doc.FileSize,
		FileHash:        doc.FileHash,
		MinioBucket:     doc.MinioBucket,
		MinioObjectKey:  doc.MinioObjectKey,
		ProcessStatus:   doc.ProcessStatus,
		ProcessError:    doc.ProcessError,
		ChunkCount:      doc.ChunkCount,
		TokenCount:      doc.TokenCount,
		Metadata:        metadataJSON,
		SourceType:      doc.SourceType,
		SourceURL:       doc.SourceURL,
		SourceContent:   doc.SourceContent,
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

// GetByIDs 批量获取文档
func (r *DocumentRepo) GetByIDs(ctx context.Context, ids []string) ([]*biz.Document, error) {
	if len(ids) == 0 {
		return []*biz.Document{}, nil
	}

	var pos []DocumentPO
	err := r.db.WithContext(ctx).GetDB().Where("id IN ?", ids).Find(&pos).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}

	docs := make([]*biz.Document, len(pos))
	for i, po := range pos {
		docs[i] = r.toDomain(&po)
	}
	return docs, nil
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
	// 序列化 Metadata
	metadataJSON := "{}"
	if doc.Metadata != nil && len(doc.Metadata) > 0 {
		bytes, _ := json.Marshal(doc.Metadata)
		metadataJSON = string(bytes)
	}

	po := &DocumentPO{
		ID:              doc.ID,
		KnowledgeBaseID: doc.KnowledgeBaseID,
		FileName:        doc.FileName,
		FileType:        doc.FileType,
		FileSize:        doc.FileSize,
		FileHash:        doc.FileHash,
		MinioBucket:     doc.MinioBucket,
		MinioObjectKey:  doc.MinioObjectKey,
		ProcessStatus:   doc.ProcessStatus,
		ProcessError:    doc.ProcessError,
		ChunkCount:      doc.ChunkCount,
		TokenCount:      doc.TokenCount,
		Metadata:        metadataJSON,
		SourceType:      doc.SourceType,
		SourceURL:       doc.SourceURL,
		SourceContent:   doc.SourceContent,
		CreatedAt:       doc.CreatedAt, // 保持原始创建时间
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

// BatchDelete 批量删除文档
func (r *DocumentRepo) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	err := r.db.WithContext(ctx).GetDB().Where("id IN ?", ids).Delete(&DocumentPO{}).Error
	if err != nil {
		return fmt.Errorf("failed to batch delete documents: %w", err)
	}

	return nil
}

// UpdateStatus 更新文档状态
func (r *DocumentRepo) UpdateStatus(ctx context.Context, id, status, errorMsg string) error {
	updates := map[string]interface{}{
		"status":        status,   // 数据库字段名
		"error_message": errorMsg, // 数据库字段名
		"updated_at":    time.Now(),
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
	// 反序列化Metadata
	var metadata map[string]interface{}
	if po.Metadata != "" && po.Metadata != "{}" {
		_ = json.Unmarshal([]byte(po.Metadata), &metadata)
	}

	return &biz.Document{
		ID:              po.ID,
		KnowledgeBaseID: po.KnowledgeBaseID,
		FileName:        po.FileName,
		FileType:        po.FileType,
		FileSize:        po.FileSize,
		FileHash:        po.FileHash,
		MinioBucket:     po.MinioBucket,
		MinioObjectKey:  po.MinioObjectKey,
		ProcessStatus:   po.ProcessStatus,
		ProcessError:    po.ProcessError,
		ChunkCount:      po.ChunkCount,
		TokenCount:      po.TokenCount,
		Metadata:        metadata,
		SourceType:      po.SourceType,
		SourceURL:       po.SourceURL,
		SourceContent:   po.SourceContent,
		CreatedAt:       po.CreatedAt,
		UpdatedAt:       po.UpdatedAt,
	}
}

// ChunkPO 文档分块数据库模型
type ChunkPO struct {
	ID              string    `gorm:"type:uuid;primarykey"`
	DocumentID      string    `gorm:"column:document_id;type:uuid;not null;index:idx_chunk_doc_id"`
	KnowledgeBaseID string    `gorm:"column:knowledge_base_id;type:uuid;not null;index:idx_chunk_kb_id"`
	ChunkIndex      int       `gorm:"column:chunk_index;not null;index:idx_chunk_doc_index"`
	Content         string    `gorm:"column:content;type:text;not null"`
	TokenCount      int       `gorm:"column:token_count;not null"`
	MilvusID        string    `gorm:"column:milvus_id;size:100;not null;uniqueIndex:idx_chunk_milvus_id"`
	Metadata        string    `gorm:"column:metadata;type:jsonb"`
	ContentTSV      string    `gorm:"column:content_tsv;type:tsvector"` // 全文搜索向量（由触发器自动维护）
	CreatedAt       time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
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
		// 序列化Metadata
		metadataJSON := "{}"
		if chunk.Metadata != nil && len(chunk.Metadata) > 0 {
			bytes, _ := json.Marshal(chunk.Metadata)
			metadataJSON = string(bytes)
		}

		// 生成MilvusID (使用documentID_chunkIndex格式)
		milvusID := fmt.Sprintf("%s_%d", chunk.DocumentID, chunk.Position)

		pos[i] = ChunkPO{
			ID:              chunk.ID,
			DocumentID:      chunk.DocumentID,
			KnowledgeBaseID: chunk.KnowledgeBaseID,
			ChunkIndex:      chunk.Position,
			Content:         chunk.Content,
			TokenCount:      chunk.TokenCount,
			MilvusID:        milvusID,
			Metadata:        metadataJSON,
			CreatedAt:       chunk.CreatedAt,
		}
	}

	// 使用 UPSERT 避免重复键冲突
	err := r.db.WithContext(ctx).GetDB().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "milvus_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"content", "token_count", "metadata"}),
	}).CreateInBatches(pos, 100).Error
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
		// 反序列化Metadata
		var metadata map[string]interface{}
		if po.Metadata != "" && po.Metadata != "{}" {
			_ = json.Unmarshal([]byte(po.Metadata), &metadata)
		}

		chunks[i] = &biz.Chunk{
			ID:              po.ID,
			DocumentID:      po.DocumentID,
			KnowledgeBaseID: po.KnowledgeBaseID,
			Content:         po.Content,
			Position:        po.ChunkIndex,
			TokenCount:      po.TokenCount,
			Metadata:        metadata,
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

// BatchDeleteByDocumentIDs 批量删除文档的分块
func (r *ChunkRepo) BatchDeleteByDocumentIDs(ctx context.Context, docIDs []string) error {
	if len(docIDs) == 0 {
		return nil
	}

	err := r.db.WithContext(ctx).GetDB().
		Where("document_id IN ?", docIDs).
		Delete(&ChunkPO{}).Error

	if err != nil {
		return fmt.Errorf("failed to batch delete chunks: %w", err)
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

// KeywordSearchResult 关键词搜索结果（带相关度分数）
type KeywordSearchResult struct {
	Chunk *biz.Chunk
	Score float32 // BM25 分数
}

// KeywordSearch 关键词搜索（使用 PostgreSQL 全文搜索 + BM25 相关度排序）
func (r *ChunkRepo) KeywordSearch(ctx context.Context, kbID, query string, topK int) ([]*biz.Chunk, error) {
	if topK <= 0 {
		topK = 10
	}

	// 使用 PostgreSQL 全文搜索 + BM25 评分
	// bm25_score: 自定义 BM25 函数，考虑词频饱和和文档长度归一化
	// plainto_tsquery: 将查询文本转换为 tsquery（自动处理空格和特殊字符）
	var results []struct {
		ChunkPO
		BM25Score float32 `gorm:"column:bm25_score"`
	}

	err := r.db.WithContext(ctx).GetDB().
		Model(&ChunkPO{}).
		Select(`
			chunks.*,
			bm25_score(content_tsv, plainto_tsquery('simple', ?), 1.2, 0.75) as bm25_score
		`, query).
		Where("knowledge_base_id = ?", kbID).
		Where("content_tsv @@ plainto_tsquery('simple', ?)", query).
		Order("bm25_score DESC").
		Limit(topK).
		Find(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to full-text search with BM25: %w", err)
	}

	chunks := make([]*biz.Chunk, len(results))
	for i, result := range results {
		// 反序列化Metadata
		var metadata map[string]interface{}
		if result.Metadata != "" && result.Metadata != "{}" {
			_ = json.Unmarshal([]byte(result.Metadata), &metadata)
		}

		chunks[i] = &biz.Chunk{
			ID:              result.ID,
			DocumentID:      result.DocumentID,
			KnowledgeBaseID: result.KnowledgeBaseID,
			Content:         result.Content,
			Position:        result.ChunkIndex,
			TokenCount:      result.TokenCount,
			Metadata:        metadata,
			CreatedAt:       result.CreatedAt,
		}

		// 将 BM25 分数存储到 metadata 中（供混合检索使用）
		if chunks[i].Metadata == nil {
			chunks[i].Metadata = make(map[string]interface{})
		}
		chunks[i].Metadata["bm25_score"] = result.BM25Score
	}

	return chunks, nil
}
