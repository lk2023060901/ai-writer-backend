package biz

import (
	"testing"
	"time"
)

func TestToDocumentResponse(t *testing.T) {
	// 测试 nil 输入
	if result := ToDocumentResponse(nil); result != nil {
		t.Error("Expected nil for nil input")
	}

	// 创建测试文档
	now := time.Now()
	doc := &Document{
		ID:              "test-id",
		KnowledgeBaseID: "kb-id",
		FileName:        "test.pdf",
		FileType:        "pdf",
		FileSize:        1024000,
		ProcessStatus:   "completed",
		ProcessError:    "test error",
		ChunkCount:      15,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// 测试转换
	resp := ToDocumentResponse(doc)

	// 验证结果
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	if resp.ID != doc.ID {
		t.Errorf("Expected ID %s, got %s", doc.ID, resp.ID)
	}

	if resp.KnowledgeBaseID != doc.KnowledgeBaseID {
		t.Errorf("Expected KnowledgeBaseID %s, got %s", doc.KnowledgeBaseID, resp.KnowledgeBaseID)
	}

	if resp.FileName != doc.FileName {
		t.Errorf("Expected FileName %s, got %s", doc.FileName, resp.FileName)
	}

	if resp.FileType != doc.FileType {
		t.Errorf("Expected FileType %s, got %s", doc.FileType, resp.FileType)
	}

	if resp.FileSize != doc.FileSize {
		t.Errorf("Expected FileSize %d, got %d", doc.FileSize, resp.FileSize)
	}

	if resp.ProcessStatus != doc.ProcessStatus {
		t.Errorf("Expected ProcessStatus %s, got %s", doc.ProcessStatus, resp.ProcessStatus)
	}

	if resp.ProcessError == nil || *resp.ProcessError != doc.ProcessError {
		t.Errorf("Expected ProcessError %s, got %v", doc.ProcessError, resp.ProcessError)
	}

	if resp.ChunkCount != doc.ChunkCount {
		t.Errorf("Expected ChunkCount %d, got %d", doc.ChunkCount, resp.ChunkCount)
	}

	expectedTime := now.Format("2006-01-02 15:04:05")
	if resp.CreatedAt != expectedTime {
		t.Errorf("Expected CreatedAt %s, got %s", expectedTime, resp.CreatedAt)
	}

	if resp.UpdatedAt != expectedTime {
		t.Errorf("Expected UpdatedAt %s, got %s", expectedTime, resp.UpdatedAt)
	}
}

func TestToDocumentResponseWithoutError(t *testing.T) {
	// 测试没有错误的情况
	now := time.Now()
	doc := &Document{
		ID:              "test-id",
		KnowledgeBaseID: "kb-id",
		FileName:        "test.pdf",
		FileType:        "pdf",
		FileSize:        1024000,
		ProcessStatus:   "completed",
		ProcessError:    "", // 空错误
		ChunkCount:      15,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	resp := ToDocumentResponse(doc)

	if resp.ProcessError != nil {
		t.Errorf("Expected ProcessError to be nil for empty error, got %v", resp.ProcessError)
	}
}
