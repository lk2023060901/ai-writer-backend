package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
)

// Document 文档模型
type Document struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	KnowledgeBaseID uuid.UUID `gorm:"type:uuid;not null;index"`

	// 文件信息
	Filename string `gorm:"type:varchar(255);not null"`
	FileType string `gorm:"type:varchar(50);not null;index"` // pdf, docx, txt, md, html
	FileSize int64  `gorm:"not null"`
	FileHash string `gorm:"type:varchar(64);not null;index"` // SHA256

	// MinIO 存储路径
	MinioBucket    string `gorm:"type:varchar(100);not null"`
	MinioObjectKey string `gorm:"type:varchar(500);not null"`

	// 处理状态
	Status       string `gorm:"type:varchar(50);not null;default:'pending';index"` // pending, processing, completed, failed
	ErrorMessage string `gorm:"type:text"`

	// 统计信息
	ChunkCount int `gorm:"default:0"`
	TokenCount int `gorm:"default:0"`

	// 元数据（JSONB）
	Metadata map[string]interface{} `gorm:"type:jsonb"`

	// 时间戳
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`

	// 关联
	KnowledgeBase *KnowledgeBase `gorm:"foreignKey:KnowledgeBaseID"`
	Chunks        []Chunk        `gorm:"foreignKey:DocumentID;constraint:OnDelete:CASCADE"`
}

// TableName 指定表名
func (Document) TableName() string {
	return "documents"
}

// Validate 验证文档
func (d *Document) Validate() error {
	if d.KnowledgeBaseID == uuid.Nil {
		return ErrInvalidKnowledgeBaseID
	}

	if d.Filename == "" {
		return ErrInvalidFilename
	}

	fileType := types.FileType(d.FileType)
	if !fileType.Valid() {
		return ErrInvalidFileType
	}

	if d.FileSize <= 0 {
		return ErrInvalidFileSize
	}

	if d.FileHash == "" {
		return ErrInvalidFileHash
	}

	if d.MinioBucket == "" || d.MinioObjectKey == "" {
		return ErrInvalidMinioPath
	}

	status := types.DocumentStatus(d.Status)
	if !status.Valid() {
		return ErrInvalidDocumentStatus
	}

	return nil
}

// IsPending 检查是否待处理
func (d *Document) IsPending() bool {
	return d.Status == string(types.DocumentStatusPending)
}

// IsProcessing 检查是否处理中
func (d *Document) IsProcessing() bool {
	return d.Status == string(types.DocumentStatusProcessing)
}

// IsCompleted 检查是否已完成
func (d *Document) IsCompleted() bool {
	return d.Status == string(types.DocumentStatusCompleted)
}

// IsFailed 检查是否失败
func (d *Document) IsFailed() bool {
	return d.Status == string(types.DocumentStatusFailed)
}

// SetStatus 设置状态
func (d *Document) SetStatus(status types.DocumentStatus) {
	d.Status = status.String()
}

// SetError 设置错误信息
func (d *Document) SetError(err error) {
	d.Status = string(types.DocumentStatusFailed)
	if err != nil {
		d.ErrorMessage = err.Error()
	}
}

// ClearError 清除错误信息
func (d *Document) ClearError() {
	d.ErrorMessage = ""
}
