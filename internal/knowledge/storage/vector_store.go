package storage

import (
	"context"

	"github.com/google/uuid"
)

// VectorStore 向量存储接口
type VectorStore interface {
	// CreateCollection 创建集合
	CreateCollection(ctx context.Context, collectionName string, dimension int) error

	// DropCollection 删除集合
	DropCollection(ctx context.Context, collectionName string) error

	// CollectionExists 检查集合是否存在
	CollectionExists(ctx context.Context, collectionName string) (bool, error)

	// Insert 插入向量
	Insert(ctx context.Context, req *InsertVectorRequest) error

	// BatchInsert 批量插入向量
	BatchInsert(ctx context.Context, req *BatchInsertVectorRequest) error

	// Delete 删除向量
	Delete(ctx context.Context, collectionName string, ids []string) error

	// Search 向量搜索
	Search(ctx context.Context, req *SearchVectorRequest) ([]*SearchResult, error)

	// GetByID 根据 ID 获取向量
	GetByID(ctx context.Context, collectionName string, id string) (*VectorData, error)
}

// InsertVectorRequest 插入向量请求
type InsertVectorRequest struct {
	CollectionName string
	ID             string
	Vector         []float32
	Metadata       map[string]interface{}
}

// BatchInsertVectorRequest 批量插入向量请求
type BatchInsertVectorRequest struct {
	CollectionName string
	Vectors        []*VectorData
}

// VectorData 向量数据
type VectorData struct {
	ID       string
	Vector   []float32
	Metadata map[string]interface{}
}

// SearchVectorRequest 向量搜索请求
type SearchVectorRequest struct {
	CollectionName string
	Vector         []float32
	TopK           int
	MinScore       float32
}

// SearchResult 搜索结果
type SearchResult struct {
	ID       string
	Score    float32
	Distance float32
	Metadata map[string]interface{}
}

// VectorStoreConfig 向量存储配置
type VectorStoreConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
}

// ConvertUUIDToString 将 UUID 转换为字符串（用于 Milvus ID）
func ConvertUUIDToString(id uuid.UUID) string {
	return id.String()
}
