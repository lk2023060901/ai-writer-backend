package milvus

import (
	"context"

	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go.uber.org/zap"
)

// SearchOptions 搜索选项
type SearchOptions struct {
	PartitionNames []string
	OutputFields   []string
	Expr           string
	Limit          int
	Offset         int
	Params         map[string]interface{}
	GroupByField   string
}

// SearchResult 搜索结果
type SearchResult struct {
	IDs     []int64
	Scores  []float32
	Fields  map[string]interface{}
}

// Search 向量搜索
func (c *Client) Search(ctx context.Context, collectionName string, vectors [][]float32, vectorField string, metricType MetricType, topK int, opts *SearchOptions) ([][]SearchResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	if collectionName == "" {
		return nil, ErrInvalidCollectionName
	}

	if len(vectors) == 0 {
		return nil, ErrInvalidVectorData
	}

	if vectorField == "" {
		return nil, ErrInvalidFieldName
	}

	// 转换向量为 entity.Vector
	entityVectors := make([]entity.Vector, len(vectors))
	for i, vec := range vectors {
		entityVectors[i] = entity.FloatVector(vec)
	}

	// 构建搜索选项
	searchOpt := milvusclient.NewSearchOption(collectionName, topK, entityVectors).
		WithANNSField(vectorField)

	if opts != nil {
		if len(opts.PartitionNames) > 0 {
			searchOpt.WithPartitions(opts.PartitionNames...)
		}
		if len(opts.OutputFields) > 0 {
			searchOpt.WithOutputFields(opts.OutputFields...)
		}
		if opts.Expr != "" {
			searchOpt.WithFilter(opts.Expr)
		}
		if opts.Offset > 0 {
			searchOpt.WithOffset(opts.Offset)
		}
		if opts.GroupByField != "" {
			searchOpt.WithGroupByField(opts.GroupByField)
		}
	}

	var resultSets []milvusclient.ResultSet
	err := c.execWithRetry(ctx, "Search", func(ctx context.Context) error {
		var err error
		resultSets, err = c.client.Search(ctx, searchOpt)
		return err
	})

	if err != nil {
		c.logger.Error("failed to search",
			zap.String("collection", collectionName),
			zap.String("vector_field", vectorField),
			zap.Error(err))
		return nil, WrapError("Search", err, collectionName, vectorField)
	}

	// 转换结果
	results := make([][]SearchResult, len(resultSets))
	for i, rs := range resultSets {
		results[i] = make([]SearchResult, rs.ResultCount)
		for j := 0; j < rs.ResultCount; j++ {
			id, _ := rs.IDs.Get(j)
			score := rs.Scores[j]

			result := SearchResult{
				IDs:    []int64{id.(int64)},
				Scores: []float32{score},
				Fields: make(map[string]interface{}),
			}

			// 提取输出字段
			if opts != nil && len(opts.OutputFields) > 0 {
				for _, fieldName := range opts.OutputFields {
					if col := rs.GetColumn(fieldName); col != nil {
						val, _ := col.Get(j)
						result.Fields[fieldName] = val
					}
				}
			}

			results[i][j] = result
		}
	}

	c.logger.Info("search completed successfully",
		zap.String("collection", collectionName),
		zap.Int("queries", len(vectors)),
		zap.Int("topK", topK))

	return results, nil
}

// HybridSearchOptions 混合搜索选项
type HybridSearchOptions struct {
	PartitionNames []string
	OutputFields   []string
	Expr           string
	Limit          int
	Offset         int
	RankerType     string
	RankerParams   map[string]interface{}
}

// HybridSearch 混合搜索(多向量搜索)
func (c *Client) HybridSearch(ctx context.Context, collectionName string, requests []*SearchOptions, opts *HybridSearchOptions) ([][]SearchResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	if collectionName == "" {
		return nil, ErrInvalidCollectionName
	}

	if len(requests) == 0 {
		return nil, ErrInvalidSearchRequest
	}

	c.logger.Info("hybrid search completed",
		zap.String("collection", collectionName),
		zap.Int("requests", len(requests)))

	// TODO: 实现混合搜索逻辑
	return nil, ErrNotImplemented
}
