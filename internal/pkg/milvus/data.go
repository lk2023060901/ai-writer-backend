package milvus

import (
	"context"

	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go.uber.org/zap"
)

// InsertOptions 插入选项
type InsertOptions struct {
	PartitionName string
}

// Insert 插入数据
func (c *Client) Insert(ctx context.Context, collectionName string, data []column.Column, opts *InsertOptions) ([]int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	if collectionName == "" {
		return nil, ErrInvalidCollectionName
	}

	if len(data) == 0 {
		return nil, ErrInvalidData
	}

	insertOpt := milvusclient.NewColumnBasedInsertOption(collectionName, data...)

	if opts != nil && opts.PartitionName != "" {
		insertOpt.WithPartition(opts.PartitionName)
	}

	var result milvusclient.InsertResult
	err := c.execWithRetry(ctx, "Insert", func(ctx context.Context) error {
		var err error
		result, err = c.client.Insert(ctx, insertOpt)
		return err
	})

	if err != nil {
		c.logger.Error("failed to insert data",
			zap.String("collection", collectionName),
			zap.Error(err))
		return nil, WrapError("Insert", err, collectionName, "")
	}

	c.logger.Info("data inserted successfully",
		zap.String("collection", collectionName),
		zap.Int64("count", result.InsertCount))

	// 转换 IDs 为 int64 切片
	if idCol, ok := result.IDs.(*column.ColumnInt64); ok {
		return idCol.Data(), nil
	}
	return nil, ErrInvalidData
}

// Upsert 更新或插入数据
func (c *Client) Upsert(ctx context.Context, collectionName string, data []column.Column, opts *InsertOptions) ([]int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	if collectionName == "" {
		return nil, ErrInvalidCollectionName
	}

	if len(data) == 0 {
		return nil, ErrInvalidData
	}

	// Upsert 使用与 Insert 相同的 option 类型
	upsertOpt := milvusclient.NewColumnBasedInsertOption(collectionName, data...)

	if opts != nil && opts.PartitionName != "" {
		upsertOpt.WithPartition(opts.PartitionName)
	}

	var result milvusclient.UpsertResult
	err := c.execWithRetry(ctx, "Upsert", func(ctx context.Context) error {
		var err error
		result, err = c.client.Upsert(ctx, upsertOpt)
		return err
	})

	if err != nil {
		c.logger.Error("failed to upsert data",
			zap.String("collection", collectionName),
			zap.Error(err))
		return nil, WrapError("Upsert", err, collectionName, "")
	}

	c.logger.Info("data upserted successfully",
		zap.String("collection", collectionName),
		zap.Int64("count", result.UpsertCount))

	// 转换 IDs 为 int64 切片
	if idCol, ok := result.IDs.(*column.ColumnInt64); ok {
		return idCol.Data(), nil
	}
	return nil, ErrInvalidData
}

// DeleteOptions 删除选项
type DeleteOptions struct {
	PartitionName string
}

// Delete 删除数据
func (c *Client) Delete(ctx context.Context, collectionName, expr string, opts *DeleteOptions) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if collectionName == "" {
		return ErrInvalidCollectionName
	}

	if expr == "" {
		return ErrInvalidExpression
	}

	deleteOpt := milvusclient.NewDeleteOption(collectionName).WithExpr(expr)

	if opts != nil && opts.PartitionName != "" {
		deleteOpt.WithPartition(opts.PartitionName)
	}

	err := c.execWithRetry(ctx, "Delete", func(ctx context.Context) error {
		_, err := c.client.Delete(ctx, deleteOpt)
		return err
	})

	if err != nil {
		c.logger.Error("failed to delete data",
			zap.String("collection", collectionName),
			zap.String("expression", expr),
			zap.Error(err))
		return WrapError("Delete", err, collectionName, "")
	}

	c.logger.Info("data deleted successfully",
		zap.String("collection", collectionName),
		zap.String("expression", expr))

	return nil
}

// Flush 刷新数据到持久化存储
func (c *Client) Flush(ctx context.Context, collectionName string, async bool) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if collectionName == "" {
		return ErrInvalidCollectionName
	}

	err := c.execWithRetry(ctx, "Flush", func(ctx context.Context) error {
		task, err := c.client.Flush(ctx, milvusclient.NewFlushOption(collectionName))
		if err != nil {
			return err
		}

		if !async {
			return task.Await(ctx)
		}

		return nil
	})

	if err != nil {
		c.logger.Error("failed to flush collection",
			zap.String("collection", collectionName),
			zap.Error(err))
		return WrapError("Flush", err, collectionName, "")
	}

	c.logger.Info("collection flushed successfully",
		zap.String("collection", collectionName),
		zap.Bool("async", async))

	return nil
}
