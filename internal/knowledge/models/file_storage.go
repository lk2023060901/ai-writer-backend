package models

import (
	"time"
)

// FileStorage 文件物理存储模型（去重存储）
type FileStorage struct {
	FileHash         string    `gorm:"column:file_hash;primaryKey" json:"file_hash"`
	Bucket           string    `gorm:"column:bucket;not null" json:"bucket"`
	ObjectKey        string    `gorm:"column:object_key;not null" json:"object_key"`
	FileSize         int64     `gorm:"column:file_size;not null" json:"file_size"`
	ContentType      string    `gorm:"column:content_type" json:"content_type"`
	ReferenceCount   int       `gorm:"column:reference_count;not null;default:1" json:"reference_count"`
	FirstUploadedAt  time.Time `gorm:"column:first_uploaded_at;not null" json:"first_uploaded_at"`
	LastReferencedAt time.Time `gorm:"column:last_referenced_at;not null" json:"last_referenced_at"`
	CreatedAt        time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

// TableName 指定表名
func (FileStorage) TableName() string {
	return "file_storage"
}
