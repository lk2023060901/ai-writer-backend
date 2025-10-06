package biz

// DocumentResponse 文档响应结构体，用于 API 响应和 SSE 事件
type DocumentResponse struct {
	ID              string  `json:"id"`
	KnowledgeBaseID string  `json:"knowledge_base_id"`
	FileName        string  `json:"file_name"`
	FileType        string  `json:"file_type"`
	FileSize        int64   `json:"file_size"`
	ProcessStatus   string  `json:"process_status"`
	ProcessError    *string `json:"process_error,omitempty"`
	ChunkCount      int64   `json:"chunk_count"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// ToDocumentResponse 将 Document 转换为 DocumentResponse
func ToDocumentResponse(doc *Document) *DocumentResponse {
	if doc == nil {
		return nil
	}

	resp := &DocumentResponse{
		ID:              doc.ID,
		KnowledgeBaseID: doc.KnowledgeBaseID,
		FileName:        doc.FileName,
		FileType:        doc.FileType,
		FileSize:        doc.FileSize,
		ProcessStatus:   doc.ProcessStatus,
		ChunkCount:      doc.ChunkCount,
		CreatedAt:       doc.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:       doc.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	if doc.ProcessError != "" {
		resp.ProcessError = &doc.ProcessError
	}

	return resp
}
