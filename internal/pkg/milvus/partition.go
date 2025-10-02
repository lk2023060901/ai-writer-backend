package milvus

import (
	"context"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go.uber.org/zap"
)

// CreatePartition 创建分区
func (c *Client) CreatePartition(ctx context.Context, collectionName, partitionName string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if collectionName == "" {
		return ErrInvalidCollectionName
	}

	if partitionName == "" {
		return ErrInvalidPartitionName
	}

	err := c.execWithRetry(ctx, "CreatePartition", func(ctx context.Context) error {
		return c.client.CreatePartition(ctx, milvusclient.NewCreatePartitionOption(collectionName, partitionName))
	})

	if err != nil {
		c.logger.Error("failed to create partition",
			zap.String("collection", collectionName),
			zap.String("partition", partitionName),
			zap.Error(err))
		return WrapError("CreatePartition", err, collectionName, partitionName)
	}

	c.logger.Info("partition created successfully",
		zap.String("collection", collectionName),
		zap.String("partition", partitionName))

	return nil
}

// DropPartition 删除分区
func (c *Client) DropPartition(ctx context.Context, collectionName, partitionName string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if collectionName == "" {
		return ErrInvalidCollectionName
	}

	if partitionName == "" {
		return ErrInvalidPartitionName
	}

	err := c.execWithRetry(ctx, "DropPartition", func(ctx context.Context) error {
		return c.client.DropPartition(ctx, milvusclient.NewDropPartitionOption(collectionName, partitionName))
	})

	if err != nil {
		c.logger.Error("failed to drop partition",
			zap.String("collection", collectionName),
			zap.String("partition", partitionName),
			zap.Error(err))
		return WrapError("DropPartition", err, collectionName, partitionName)
	}

	c.logger.Info("partition dropped successfully",
		zap.String("collection", collectionName),
		zap.String("partition", partitionName))

	return nil
}

// HasPartition 检查分区是否存在
func (c *Client) HasPartition(ctx context.Context, collectionName, partitionName string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return false, ErrClientClosed
	}

	if collectionName == "" {
		return false, ErrInvalidCollectionName
	}

	if partitionName == "" {
		return false, ErrInvalidPartitionName
	}

	var exists bool
	err := c.execWithRetry(ctx, "HasPartition", func(ctx context.Context) error {
		var err error
		exists, err = c.client.HasPartition(ctx, milvusclient.NewHasPartitionOption(collectionName, partitionName))
		return err
	})

	if err != nil {
		return false, WrapError("HasPartition", err, collectionName, partitionName)
	}

	return exists, nil
}

// ListPartitions 列出所有分区
func (c *Client) ListPartitions(ctx context.Context, collectionName string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	if collectionName == "" {
		return nil, ErrInvalidCollectionName
	}

	var partitions []string
	err := c.execWithRetry(ctx, "ListPartitions", func(ctx context.Context) error {
		result, err := c.client.ListPartitions(ctx, milvusclient.NewListPartitionOption(collectionName))
		if err != nil {
			return err
		}
		partitions = result
		return nil
	})

	if err != nil {
		return nil, WrapError("ListPartitions", err, collectionName, "")
	}

	return partitions, nil
}

// LoadPartitions 加载分区到内存
func (c *Client) LoadPartitions(ctx context.Context, collectionName string, partitionNames []string, async bool) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if collectionName == "" {
		return ErrInvalidCollectionName
	}

	if len(partitionNames) == 0 {
		return ErrInvalidPartitionName
	}

	loadOpt := milvusclient.NewLoadPartitionsOption(collectionName, partitionNames...)

	err := c.execWithRetry(ctx, "LoadPartitions", func(ctx context.Context) error {
		task, err := c.client.LoadPartitions(ctx, loadOpt)
		if err != nil {
			return err
		}

		if !async {
			return task.Await(ctx)
		}

		return nil
	})

	if err != nil {
		c.logger.Error("failed to load partitions",
			zap.String("collection", collectionName),
			zap.Strings("partitions", partitionNames),
			zap.Error(err))
		return WrapError("LoadPartitions", err, collectionName, "")
	}

	c.logger.Info("partitions loaded successfully",
		zap.String("collection", collectionName),
		zap.Strings("partitions", partitionNames),
		zap.Bool("async", async))

	return nil
}

// ReleasePartitions 从内存中释放分区
func (c *Client) ReleasePartitions(ctx context.Context, collectionName string, partitionNames []string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if collectionName == "" {
		return ErrInvalidCollectionName
	}

	if len(partitionNames) == 0 {
		return ErrInvalidPartitionName
	}

	err := c.execWithRetry(ctx, "ReleasePartitions", func(ctx context.Context) error {
		return c.client.ReleasePartitions(ctx, milvusclient.NewReleasePartitionsOptions(collectionName, partitionNames...))
	})

	if err != nil {
		c.logger.Error("failed to release partitions",
			zap.String("collection", collectionName),
			zap.Strings("partitions", partitionNames),
			zap.Error(err))
		return WrapError("ReleasePartitions", err, collectionName, "")
	}

	c.logger.Info("partitions released successfully",
		zap.String("collection", collectionName),
		zap.Strings("partitions", partitionNames))

	return nil
}
