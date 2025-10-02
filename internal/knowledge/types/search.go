package types

import "github.com/google/uuid"

// SearchRequest 搜索请求
type SearchRequest struct {
	KnowledgeBaseID uuid.UUID `json:"knowledge_base_id" validate:"required"`
	Query           string    `json:"query" validate:"required,min=1,max=1000"`
	TopK            int       `json:"top_k" validate:"min=1,max=100"`
	MinScore        float32   `json:"min_score,omitempty" validate:"omitempty,min=0,max=1"`
	EnableRerank    bool      `json:"enable_rerank"`
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Query   string             `json:"query"`
	Results []*ChunkWithScore  `json:"results"`
	Total   int                `json:"total"`
	Took    int64              `json:"took"` // 耗时（毫秒）
}

// SearchMetrics 搜索指标
type SearchMetrics struct {
	VectorSearchTime int64 `json:"vector_search_time"` // 向量搜索耗时
	RerankTime       int64 `json:"rerank_time"`        // 重排序耗时
	TotalTime        int64 `json:"total_time"`         // 总耗时
	ResultCount      int   `json:"result_count"`       // 结果数量
}
