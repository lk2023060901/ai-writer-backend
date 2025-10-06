package data

import (
	"testing"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
)

func TestDocumentPOMapping(t *testing.T) {
	// 创建测试文档
	now := time.Now()
	createdTime := now.Add(-1 * time.Hour) // 1小时前创建

	doc := &biz.Document{
		ID:              "test-id",
		KnowledgeBaseID: "kb-id",
		FileName:        "test.pdf",
		FileType:        "pdf",
		FileSize:        1024000,
		FileHash:        "hash123",
		MinioBucket:     "bucket",
		MinioObjectKey:  "key",
		ProcessStatus:   "completed",
		ProcessError:    "",
		ChunkCount:      15,
		TokenCount:      1000,
		CreatedAt:       createdTime,
		UpdatedAt:       now,
	}

	repo := &DocumentRepo{}

	// 测试转换为 PO（模拟 Update 方法中的逻辑）
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
		Metadata:        "{}",
		CreatedAt:       doc.CreatedAt, // 关键：保持原始创建时间
		UpdatedAt:       time.Now(),
	}

	// 验证 CreatedAt 被正确保留
	if po.CreatedAt != createdTime {
		t.Errorf("Expected CreatedAt %v, got %v", createdTime, po.CreatedAt)
	}

	// 测试转换回 Domain
	domainDoc := repo.toDomain(po)

	// 验证时间字段
	if domainDoc.CreatedAt != createdTime {
		t.Errorf("Expected CreatedAt %v, got %v", createdTime, domainDoc.CreatedAt)
	}

	if domainDoc.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero time")
	}

	// 验证格式化后的时间
	formattedTime := domainDoc.CreatedAt.Format("2006-01-02 15:04:05")
	if formattedTime == "0001-01-01 00:00:00" || formattedTime == "0001-01-01 08:05:43" {
		t.Errorf("CreatedAt should not be zero time, got %s", formattedTime)
	}
}
