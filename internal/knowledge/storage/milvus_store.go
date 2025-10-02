package storage

import (
	"context"
	"fmt"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/milvus"
	"github.com/milvus-io/milvus/client/v2/column"
	"go.uber.org/zap"
)

// MilvusStore Milvus 向量存储实现
type MilvusStore struct {
	client *milvus.Client
	logger *logger.Logger
}

// NewMilvusStore 创建 Milvus 向量存储
func NewMilvusStore(client *milvus.Client, lgr *logger.Logger) *MilvusStore {
	if lgr == nil {
		lgr = logger.L()
	}
	return &MilvusStore{
		client: client,
		logger: lgr,
	}
}

// CreateCollection 创建集合
func (s *MilvusStore) CreateCollection(ctx context.Context, collectionName string, dimension int) error {
	// 检查集合是否已存在
	exists, err := s.client.HasCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}
	if exists {
		return fmt.Errorf("collection %s already exists", collectionName)
	}

	// 创建 Schema
	schema := &milvus.CollectionSchema{
		Name:        collectionName,
		Description: "Knowledge base vector collection",
		AutoID:      false,
		Fields: []*milvus.FieldSchema{
			{
				Name:         "id",
				DataType:     milvus.DataTypeVarChar,
				IsPrimaryKey: true,
				TypeParams: map[string]interface{}{
					"max_length": "128",
				},
			},
			{
				Name:       "embedding",
				DataType:   milvus.DataTypeFloatVector,
				Dimension:  dimension,
			},
		},
	}

	// 创建集合
	if err := s.client.CreateCollection(ctx, schema, nil); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// 创建索引
	indexOpts := &milvus.IndexOptions{
		IndexType:  milvus.IndexTypeIVFFlat,
		MetricType: milvus.MetricTypeIP, // Inner Product (余弦相似度，向量需归一化)
		Params: map[string]interface{}{
			"nlist": 1024,
		},
	}

	if err := s.client.CreateIndex(ctx, collectionName, "embedding", indexOpts); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// 加载集合到内存
	if err := s.client.LoadCollection(ctx, collectionName, false); err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	s.logger.Info("milvus collection created successfully",
		zap.String("collection", collectionName),
		zap.Int("dimension", dimension))

	return nil
}

// DropCollection 删除集合
func (s *MilvusStore) DropCollection(ctx context.Context, collectionName string) error {
	if err := s.client.DropCollection(ctx, collectionName); err != nil {
		return fmt.Errorf("failed to drop collection: %w", err)
	}

	s.logger.Info("milvus collection dropped successfully",
		zap.String("collection", collectionName))

	return nil
}

// CollectionExists 检查集合是否存在
func (s *MilvusStore) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	exists, err := s.client.HasCollection(ctx, collectionName)
	if err != nil {
		return false, fmt.Errorf("failed to check collection: %w", err)
	}
	return exists, nil
}

// Insert 插入单个向量
func (s *MilvusStore) Insert(ctx context.Context, req *InsertVectorRequest) error {
	return s.BatchInsert(ctx, &BatchInsertVectorRequest{
		CollectionName: req.CollectionName,
		Vectors: []*VectorData{
			{
				ID:       req.ID,
				Vector:   req.Vector,
				Metadata: req.Metadata,
			},
		},
	})
}

// BatchInsert 批量插入向量
func (s *MilvusStore) BatchInsert(ctx context.Context, req *BatchInsertVectorRequest) error {
	if len(req.Vectors) == 0 {
		return fmt.Errorf("no vectors to insert")
	}

	// 构建列数据
	ids := make([]string, len(req.Vectors))
	vectors := make([][]float32, len(req.Vectors))

	for i, v := range req.Vectors {
		ids[i] = v.ID
		vectors[i] = v.Vector
	}

	// 创建列
	columns := []column.Column{
		column.NewColumnVarChar("id", ids),
		column.NewColumnFloatVector("embedding", len(vectors[0]), vectors),
	}

	// 插入数据
	if _, err := s.client.Insert(ctx, req.CollectionName, columns, nil); err != nil {
		return fmt.Errorf("failed to insert vectors: %w", err)
	}

	// 刷新数据
	if err := s.client.Flush(ctx, req.CollectionName, false); err != nil {
		s.logger.Warn("failed to flush collection after insert",
			zap.String("collection", req.CollectionName),
			zap.Error(err))
	}

	s.logger.Info("vectors inserted successfully",
		zap.String("collection", req.CollectionName),
		zap.Int("count", len(req.Vectors)))

	return nil
}

// Delete 删除向量
func (s *MilvusStore) Delete(ctx context.Context, collectionName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// 构建删除表达式
	expr := fmt.Sprintf("id in %v", ids)

	if err := s.client.Delete(ctx, collectionName, expr, nil); err != nil {
		return fmt.Errorf("failed to delete vectors: %w", err)
	}

	s.logger.Info("vectors deleted successfully",
		zap.String("collection", collectionName),
		zap.Int("count", len(ids)))

	return nil
}

// Search 向量搜索
func (s *MilvusStore) Search(ctx context.Context, req *SearchVectorRequest) ([]*SearchResult, error) {
	// 执行搜索
	searchOpts := &milvus.SearchOptions{
		OutputFields: []string{"id"},
		Limit:        req.TopK,
	}

	results, err := s.client.Search(ctx, req.CollectionName, [][]float32{req.Vector}, "embedding", milvus.MetricTypeIP, req.TopK, searchOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	if len(results) == 0 {
		return []*SearchResult{}, nil
	}

	// 转换结果
	searchResults := make([]*SearchResult, 0)
	for _, result := range results[0] {
		if len(result.Scores) == 0 {
			continue
		}

		score := result.Scores[0]

		// 应用最小分数过滤
		if req.MinScore > 0 && score < req.MinScore {
			continue
		}

		// 提取 ID
		var id string
		if idField, ok := result.Fields["id"]; ok {
			id = idField.(string)
		}

		searchResults = append(searchResults, &SearchResult{
			ID:       id,
			Score:    score,
			Distance: 1 - score, // IP 距离转换
			Metadata: result.Fields,
		})
	}

	s.logger.Info("vector search completed",
		zap.String("collection", req.CollectionName),
		zap.Int("results", len(searchResults)))

	return searchResults, nil
}

// GetByID 根据 ID 获取向量
func (s *MilvusStore) GetByID(ctx context.Context, collectionName string, id string) (*VectorData, error) {
	// Milvus 不直接支持通过 ID 获取，需要通过 Query
	expr := fmt.Sprintf("id == \"%s\"", id)

	queryOpts := &milvus.QueryOptions{
		OutputFields: []string{"id", "embedding"},
	}

	results, err := s.client.Query(ctx, collectionName, expr, queryOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to query vector: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("vector not found: %s", id)
	}

	// 提取结果
	result := results[0]
	vectorData := &VectorData{
		ID:       id,
		Metadata: make(map[string]interface{}),
	}

	// 提取向量
	if embeddingField, ok := result.Fields["embedding"]; ok {
		if vec, ok := embeddingField.([]float32); ok {
			vectorData.Vector = vec
		}
	}

	return vectorData, nil
}
