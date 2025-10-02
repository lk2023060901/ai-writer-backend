package milvus

import (
	"context"

	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go.uber.org/zap"
)

// IndexOptions 索引创建选项
type IndexOptions struct {
	IndexType  IndexType
	MetricType MetricType
	Params     map[string]interface{}
}

// CreateIndex 创建索引
func (c *Client) CreateIndex(ctx context.Context, collectionName, fieldName string, opts *IndexOptions) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if collectionName == "" {
		return ErrInvalidCollectionName
	}

	if fieldName == "" {
		return ErrInvalidFieldName
	}

	if opts == nil {
		return ErrInvalidIndexParams
	}

	// 构建索引
	var idx index.Index
	switch opts.IndexType {
	case IndexTypeFlat:
		idx = index.NewFlatIndex(toEntityMetricType(opts.MetricType))
	case IndexTypeIVFFlat:
		nlist := 128
		if v, ok := opts.Params["nlist"].(int); ok {
			nlist = v
		}
		idx = index.NewIvfFlatIndex(toEntityMetricType(opts.MetricType), nlist)
	case IndexTypeIVFSQ8:
		nlist := 128
		if v, ok := opts.Params["nlist"].(int); ok {
			nlist = v
		}
		idx = index.NewIvfSQ8Index(toEntityMetricType(opts.MetricType), nlist)
	case IndexTypeHNSW:
		M := 16
		efConstruction := 200
		if v, ok := opts.Params["M"].(int); ok {
			M = v
		}
		if v, ok := opts.Params["efConstruction"].(int); ok {
			efConstruction = v
		}
		idx = index.NewHNSWIndex(toEntityMetricType(opts.MetricType), M, efConstruction)
	default:
		return ErrInvalidIndexType
	}

	createOpt := milvusclient.NewCreateIndexOption(collectionName, fieldName, idx)

	err := c.execWithRetry(ctx, "CreateIndex", func(ctx context.Context) error {
		task, err := c.client.CreateIndex(ctx, createOpt)
		if err != nil {
			return err
		}
		return task.Await(ctx)
	})

	if err != nil {
		c.logger.Error("failed to create index",
			zap.String("collection", collectionName),
			zap.String("field", fieldName),
			zap.Error(err))
		return WrapError("CreateIndex", err, collectionName, fieldName)
	}

	c.logger.Info("index created successfully",
		zap.String("collection", collectionName),
		zap.String("field", fieldName))

	return nil
}

// DropIndex 删除索引
func (c *Client) DropIndex(ctx context.Context, collectionName, fieldName string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if collectionName == "" {
		return ErrInvalidCollectionName
	}

	if fieldName == "" {
		return ErrInvalidFieldName
	}

	err := c.execWithRetry(ctx, "DropIndex", func(ctx context.Context) error {
		return c.client.DropIndex(ctx, milvusclient.NewDropIndexOption(collectionName, fieldName))
	})

	if err != nil {
		c.logger.Error("failed to drop index",
			zap.String("collection", collectionName),
			zap.String("field", fieldName),
			zap.Error(err))
		return WrapError("DropIndex", err, collectionName, fieldName)
	}

	c.logger.Info("index dropped successfully",
		zap.String("collection", collectionName),
		zap.String("field", fieldName))

	return nil
}

// DescribeIndex 描述索引
func (c *Client) DescribeIndex(ctx context.Context, collectionName, fieldName string) (map[string]interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	if collectionName == "" {
		return nil, ErrInvalidCollectionName
	}

	if fieldName == "" {
		return nil, ErrInvalidFieldName
	}

	var result map[string]interface{}
	err := c.execWithRetry(ctx, "DescribeIndex", func(ctx context.Context) error {
		indexDesc, err := c.client.DescribeIndex(ctx, milvusclient.NewDescribeIndexOption(collectionName, fieldName))
		if err != nil {
			return err
		}
		result = map[string]interface{}{
			"index_type":   indexDesc.IndexType(),
			"metric_type":  indexDesc.Params()["metric_type"],
			"params":       indexDesc.Params(),
		}
		return nil
	})

	if err != nil {
		return nil, WrapError("DescribeIndex", err, collectionName, fieldName)
	}

	return result, nil
}

// toEntityMetricType 转换为 entity.MetricType
func toEntityMetricType(mt MetricType) entity.MetricType {
	switch mt {
	case MetricTypeL2:
		return entity.L2
	case MetricTypeIP:
		return entity.IP
	case MetricTypeCosine:
		return entity.COSINE
	default:
		return entity.L2
	}
}
