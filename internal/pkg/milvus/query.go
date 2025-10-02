package milvus

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go.uber.org/zap"
)

// QueryOptions 查询选项
type QueryOptions struct {
	PartitionNames []string
	OutputFields   []string
	Limit          int
	Offset         int
}

// QueryResult 查询结果
type QueryResult struct {
	Fields map[string]interface{}
}

// Query 标量查询
func (c *Client) Query(ctx context.Context, collectionName, expr string, opts *QueryOptions) ([]QueryResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	if collectionName == "" {
		return nil, ErrInvalidCollectionName
	}

	if expr == "" {
		return nil, ErrInvalidExpression
	}

	// 构建查询选项
	queryOpt := milvusclient.NewQueryOption(collectionName).
		WithFilter(expr)

	if opts != nil {
		if len(opts.PartitionNames) > 0 {
			queryOpt.WithPartitions(opts.PartitionNames...)
		}
		if len(opts.OutputFields) > 0 {
			queryOpt.WithOutputFields(opts.OutputFields...)
		}
		if opts.Limit > 0 {
			queryOpt.WithLimit(opts.Limit)
		}
		if opts.Offset > 0 {
			queryOpt.WithOffset(opts.Offset)
		}
	}

	var resultSet milvusclient.ResultSet
	err := c.execWithRetry(ctx, "Query", func(ctx context.Context) error {
		var err error
		resultSet, err = c.client.Query(ctx, queryOpt)
		return err
	})

	if err != nil {
		c.logger.Error("failed to query",
			zap.String("collection", collectionName),
			zap.String("expression", expr),
			zap.Error(err))
		return nil, WrapError("Query", err, collectionName, "")
	}

	// 转换结果
	results := make([]QueryResult, resultSet.ResultCount)
	for i := 0; i < resultSet.ResultCount; i++ {
		result := QueryResult{
			Fields: make(map[string]interface{}),
		}

		// 提取所有字段
		for _, col := range resultSet.Fields {
			if col != nil {
				val, _ := col.Get(i)
				result.Fields[col.Name()] = val
			}
		}

		results[i] = result
	}

	c.logger.Info("query completed successfully",
		zap.String("collection", collectionName),
		zap.Int("count", len(results)))

	return results, nil
}

// Get 根据 ID 获取实体
func (c *Client) Get(ctx context.Context, collectionName string, ids []int64, opts *QueryOptions) ([]QueryResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	if collectionName == "" {
		return nil, ErrInvalidCollectionName
	}

	if len(ids) == 0 {
		return nil, ErrInvalidIDs
	}

	// 构建 ID 表达式
	expr := fmt.Sprintf("id in %v", ids)

	// 构建查询选项
	queryOpt := milvusclient.NewQueryOption(collectionName).
		WithFilter(expr)

	if opts != nil {
		if len(opts.PartitionNames) > 0 {
			queryOpt.WithPartitions(opts.PartitionNames...)
		}
		if len(opts.OutputFields) > 0 {
			queryOpt.WithOutputFields(opts.OutputFields...)
		} else {
			queryOpt.WithOutputFields("*")
		}
	} else {
		queryOpt.WithOutputFields("*")
	}

	var resultSet milvusclient.ResultSet
	err := c.execWithRetry(ctx, "Get", func(ctx context.Context) error {
		var err error
		resultSet, err = c.client.Query(ctx, queryOpt)
		return err
	})

	if err != nil {
		c.logger.Error("failed to get entities",
			zap.String("collection", collectionName),
			zap.Int("id_count", len(ids)),
			zap.Error(err))
		return nil, WrapError("Get", err, collectionName, "")
	}

	// 转换结果
	results := make([]QueryResult, resultSet.ResultCount)
	for i := 0; i < resultSet.ResultCount; i++ {
		result := QueryResult{
			Fields: make(map[string]interface{}),
		}

		// 提取所有字段
		for _, col := range resultSet.Fields {
			if col != nil {
				val, _ := col.Get(i)
				result.Fields[col.Name()] = val
			}
		}

		results[i] = result
	}

	c.logger.Info("get entities completed successfully",
		zap.String("collection", collectionName),
		zap.Int("count", len(results)))

	return results, nil
}
