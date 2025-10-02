package milvus

import (
	"context"

	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go.uber.org/zap"
)

// CollectionInfo Collection 信息
type CollectionInfo struct {
	Name            string
	ID              int64
	Description     string
	NumPartitions   int64
	NumEntities     int64
	ConsistencyLevel string
	Schema          *CollectionSchema
}

// CreateCollectionOptions Collection 创建选项
type CreateCollectionOptions struct {
	ShardsNum        int32
	ConsistencyLevel string
	Properties       map[string]string
}

// CreateCollection 创建 Collection
func (c *Client) CreateCollection(ctx context.Context, schema *CollectionSchema, opts *CreateCollectionOptions) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if schema == nil {
		return ErrInvalidSchema
	}

	// 验证 Schema
	if err := schema.Validate(); err != nil {
		return WrapError("CreateCollection", err, schema.Name, "")
	}

	// 构建创建选项
	createOpt := milvusclient.NewCreateCollectionOption(schema.Name, schema.ToEntity())

	// 设置选项
	if opts != nil {
		if opts.ShardsNum > 0 {
			createOpt.WithShardNum(opts.ShardsNum)
		}
		if opts.ConsistencyLevel != "" {
			createOpt.WithConsistencyLevel(parseConsistencyLevel(opts.ConsistencyLevel))
		}
		// WithProperty 接受单个键值对
		for k, v := range opts.Properties {
			createOpt.WithProperty(k, v)
		}
	}

	// 执行创建
	err := c.execWithRetry(ctx, "CreateCollection", func(ctx context.Context) error {
		return c.client.CreateCollection(ctx, createOpt)
	})

	if err != nil {
		c.logger.Error("failed to create collection",
			zap.String("collection", schema.Name),
			zap.Error(err))
		return WrapError("CreateCollection", err, schema.Name, "")
	}

	c.logger.Info("collection created successfully",
		zap.String("collection", schema.Name))

	return nil
}

// DropCollection 删除 Collection
func (c *Client) DropCollection(ctx context.Context, collectionName string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if collectionName == "" {
		return ErrInvalidCollectionName
	}

	err := c.execWithRetry(ctx, "DropCollection", func(ctx context.Context) error {
		return c.client.DropCollection(ctx, milvusclient.NewDropCollectionOption(collectionName))
	})

	if err != nil {
		c.logger.Error("failed to drop collection",
			zap.String("collection", collectionName),
			zap.Error(err))
		return WrapError("DropCollection", err, collectionName, "")
	}

	c.logger.Info("collection dropped successfully",
		zap.String("collection", collectionName))

	return nil
}

// HasCollection 检查 Collection 是否存在
func (c *Client) HasCollection(ctx context.Context, collectionName string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return false, ErrClientClosed
	}

	if collectionName == "" {
		return false, ErrInvalidCollectionName
	}

	var exists bool
	err := c.execWithRetry(ctx, "HasCollection", func(ctx context.Context) error {
		var err error
		exists, err = c.client.HasCollection(ctx, milvusclient.NewHasCollectionOption(collectionName))
		return err
	})

	if err != nil {
		return false, WrapError("HasCollection", err, collectionName, "")
	}

	return exists, nil
}

// DescribeCollection 获取 Collection 信息
func (c *Client) DescribeCollection(ctx context.Context, collectionName string) (*CollectionInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	if collectionName == "" {
		return nil, ErrInvalidCollectionName
	}

	var result *entity.Collection
	err := c.execWithRetry(ctx, "DescribeCollection", func(ctx context.Context) error {
		var err error
		result, err = c.client.DescribeCollection(ctx, milvusclient.NewDescribeCollectionOption(collectionName))
		return err
	})

	if err != nil {
		return nil, WrapError("DescribeCollection", err, collectionName, "")
	}

	info := &CollectionInfo{
		Name:        result.Name,
		ID:          result.ID,
		Description: result.Schema.Description,
		Schema:      fromEntitySchema(result.Schema),
	}

	return info, nil
}

// ListCollections 列出所有 Collection
func (c *Client) ListCollections(ctx context.Context) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	var collections []string
	err := c.execWithRetry(ctx, "ListCollections", func(ctx context.Context) error {
		result, err := c.client.ListCollections(ctx, milvusclient.NewListCollectionOption())
		if err != nil {
			return err
		}
		collections = result
		return nil
	})

	if err != nil {
		return nil, WrapError("ListCollections", err, "", "")
	}

	return collections, nil
}

// LoadCollection 加载 Collection 到内存
func (c *Client) LoadCollection(ctx context.Context, collectionName string, async bool) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if collectionName == "" {
		return ErrInvalidCollectionName
	}

	loadOpt := milvusclient.NewLoadCollectionOption(collectionName)

	err := c.execWithRetry(ctx, "LoadCollection", func(ctx context.Context) error {
		task, err := c.client.LoadCollection(ctx, loadOpt)
		if err != nil {
			return err
		}

		if !async {
			return task.Await(ctx)
		}

		return nil
	})

	if err != nil {
		c.logger.Error("failed to load collection",
			zap.String("collection", collectionName),
			zap.Error(err))
		return WrapError("LoadCollection", err, collectionName, "")
	}

	c.logger.Info("collection loaded successfully",
		zap.String("collection", collectionName),
		zap.Bool("async", async))

	return nil
}

// ReleaseCollection 从内存中释放 Collection
func (c *Client) ReleaseCollection(ctx context.Context, collectionName string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if collectionName == "" {
		return ErrInvalidCollectionName
	}

	err := c.execWithRetry(ctx, "ReleaseCollection", func(ctx context.Context) error {
		return c.client.ReleaseCollection(ctx, milvusclient.NewReleaseCollectionOption(collectionName))
	})

	if err != nil {
		c.logger.Error("failed to release collection",
			zap.String("collection", collectionName),
			zap.Error(err))
		return WrapError("ReleaseCollection", err, collectionName, "")
	}

	c.logger.Info("collection released successfully",
		zap.String("collection", collectionName))

	return nil
}

// GetCollectionStatistics 获取 Collection 统计信息
func (c *Client) GetCollectionStatistics(ctx context.Context, collectionName string) (map[string]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	if collectionName == "" {
		return nil, ErrInvalidCollectionName
	}

	var stats map[string]string
	err := c.execWithRetry(ctx, "GetCollectionStatistics", func(ctx context.Context) error {
		var err error
		stats, err = c.client.GetCollectionStats(ctx, milvusclient.NewGetCollectionStatsOption(collectionName))
		return err
	})

	if err != nil {
		return nil, WrapError("GetCollectionStatistics", err, collectionName, "")
	}

	return stats, nil
}

// RenameCollection 重命名 Collection
func (c *Client) RenameCollection(ctx context.Context, oldName, newName string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if oldName == "" || newName == "" {
		return ErrInvalidCollectionName
	}

	err := c.execWithRetry(ctx, "RenameCollection", func(ctx context.Context) error {
		return c.client.RenameCollection(ctx, milvusclient.NewRenameCollectionOption(oldName, newName))
	})

	if err != nil {
		c.logger.Error("failed to rename collection",
			zap.String("old_name", oldName),
			zap.String("new_name", newName),
			zap.Error(err))
		return WrapError("RenameCollection", err, oldName, "")
	}

	c.logger.Info("collection renamed successfully",
		zap.String("old_name", oldName),
		zap.String("new_name", newName))

	return nil
}

// fromEntitySchema 从 entity.Schema 转换
func fromEntitySchema(schema *entity.Schema) *CollectionSchema {
	if schema == nil {
		return nil
	}

	collSchema := &CollectionSchema{
		Name:              schema.CollectionName,
		Description:       schema.Description,
		AutoID:            schema.AutoID,
		EnableDynamicField: schema.EnableDynamicField,
		Fields:            make([]*FieldSchema, 0, len(schema.Fields)),
	}

	for _, field := range schema.Fields {
		fieldSchema := &FieldSchema{
			Name:         field.Name,
			DataType:     DataType(field.DataType),
			IsPrimaryKey: field.PrimaryKey,
			IsAutoID:     field.AutoID,
			Description:  field.Description,
			TypeParams:   make(map[string]interface{}),
		}

		// 获取向量维度
		if dim, err := field.GetDim(); err == nil {
			fieldSchema.Dimension = int(dim)
		}

		// 从 TypeParams 获取 MaxLength
		for k, v := range field.TypeParams {
			fieldSchema.TypeParams[k] = v
		}

		collSchema.Fields = append(collSchema.Fields, fieldSchema)
	}

	return collSchema
}

// parseConsistencyLevel 解析一致性级别
func parseConsistencyLevel(level string) entity.ConsistencyLevel {
	switch level {
	case "Strong":
		return entity.ClStrong
	case "Session":
		return entity.ClSession
	case "Bounded":
		return entity.ClBounded
	case "Eventually":
		return entity.ClEventually
	case "Customized":
		return entity.ClCustomized
	default:
		return entity.ClSession // 默认 Session
	}
}
