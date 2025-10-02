# Milvus 封装库

Milvus 向量数据库的 Go 语言封装库,提供简洁易用的 API。

## 特性

- ✅ **完整的功能覆盖**：Collection、Partition、Index、Data、Search、Query
- ✅ **类型安全**：强类型 Schema 定义和验证
- ✅ **错误处理**：统一的错误封装和类型检查
- ✅ **重试机制**：自动重试可恢复的错误
- ✅ **连接管理**：连接池和健康检查
- ✅ **日志记录**：基于 zap 的结构化日志
- ✅ **Schema 构建器**：链式 API 构建 Schema

## 安装

```bash
go get github.com/milvus-io/milvus/client/v2
```

## 快速开始

### 1. 创建客户端

```go
package main

import (
	"context"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/milvus"
)

func main() {
	// 创建日志
	log, _ := logger.New(logger.DefaultConfig())

	// 创建配置
	cfg := &milvus.Config{
		Address:        "localhost:19530",
		Username:       "",
		Password:       "",
		Database:       "default",
		DialTimeout:    10 * time.Second,
		RequestTimeout: 30 * time.Second,
		MaxRetries:     3,
		RetryDelay:     time.Second,
	}

	// 创建客户端
	client, err := milvus.New(context.Background(), cfg, log)
	if err != nil {
		panic(err)
	}
	defer client.Close(context.Background())

	// 检查连接
	if err := client.Ping(context.Background()); err != nil {
		panic(err)
	}
}
```

### 2. 创建 Collection

```go
// 使用 Schema 构建器
schema, err := milvus.NewSchemaBuilder("my_collection", "示例集合").
	AddInt64Field("id", true, true).                      // 主键，自动ID
	AddVarCharField("title", 256, false).                 // 标题字段
	AddFloatVectorField("embedding", 768).                // 向量字段
	EnableDynamicField().                                 // 启用动态字段
	Build()
if err != nil {
	panic(err)
}

// 创建 Collection
err = client.CreateCollection(ctx, schema, &milvus.CreateCollectionOptions{
	ShardsNum:        2,
	ConsistencyLevel: "Strong",
})
if err != nil {
	panic(err)
}

// 加载到内存
err = client.LoadCollection(ctx, "my_collection", false)
if err != nil {
	panic(err)
}
```

### 3. 创建索引

```go
// 为向量字段创建 HNSW 索引
err = client.CreateIndex(ctx, "my_collection", "embedding", &milvus.IndexOptions{
	IndexType:  milvus.IndexTypeHNSW,
	MetricType: milvus.MetricTypeCosine,
	Params: map[string]interface{}{
		"M":              16,
		"efConstruction": 200,
	},
})
if err != nil {
	panic(err)
}
```

### 4. 插入数据

```go
// 准备数据
ids := []int64{1, 2, 3}
titles := []string{"文档1", "文档2", "文档3"}
embeddings := [][]float32{
	{0.1, 0.2, 0.3, ...}, // 768 维向量
	{0.4, 0.5, 0.6, ...},
	{0.7, 0.8, 0.9, ...},
}

// 构建列
columns := []column.Column{
	milvus.BuildInt64Column("id", ids),
	milvus.BuildVarCharColumn("title", titles),
	milvus.BuildFloatVectorColumn("embedding", 768, embeddings),
}

// 插入数据
insertedIDs, err := client.Insert(ctx, "my_collection", columns, nil)
if err != nil {
	panic(err)
}

fmt.Printf("插入了 %d 条数据\n", len(insertedIDs))
```

### 5. 向量搜索

```go
// 查询向量
queryVector := [][]float32{
	{0.1, 0.2, 0.3, ...}, // 768 维
}

// 执行搜索
results, err := client.Search(ctx, "my_collection", queryVector, "embedding",
	milvus.MetricTypeCosine, 10, &milvus.SearchOptions{
		OutputFields: []string{"id", "title"},
		Expr:         "id > 0",
		Limit:        10,
	})
if err != nil {
	panic(err)
}

// 处理结果
for i, resultSet := range results {
	fmt.Printf("查询 %d 的结果:\n", i)
	for _, result := range resultSet {
		fmt.Printf("  ID: %v, Score: %.4f, Title: %v\n",
			result.IDs[0], result.Scores[0], result.Fields["title"])
	}
}
```

### 6. 标量查询

```go
// 使用表达式查询
results, err := client.Query(ctx, "my_collection", "id in [1, 2, 3]", &milvus.QueryOptions{
	OutputFields: []string{"id", "title", "embedding"},
	Limit:        100,
})
if err != nil {
	panic(err)
}

// 或者根据 ID 直接获取
results, err = client.Get(ctx, "my_collection", []int64{1, 2, 3}, &milvus.QueryOptions{
	OutputFields: []string{"title", "embedding"},
})
```

### 7. 更新和删除

```go
// Upsert (更新或插入)
upsertedIDs, err := client.Upsert(ctx, "my_collection", columns, nil)

// 删除数据
err = client.Delete(ctx, "my_collection", "id in [1, 2, 3]", nil)

// 刷新数据
err = client.Flush(ctx, "my_collection", false)
```

## API 参考

### 客户端管理

```go
// 创建客户端
client, err := milvus.New(ctx, cfg, logger)

// 关闭客户端
err = client.Close(ctx)

// 检查连接
err = client.Ping(ctx)

// 获取统计信息
stats := client.GetStats()
```

### Collection 操作

```go
// 创建 Collection
err = client.CreateCollection(ctx, schema, opts)

// 删除 Collection
err = client.DropCollection(ctx, collectionName)

// 检查是否存在
exists, err := client.HasCollection(ctx, collectionName)

// 获取详情
info, err := client.DescribeCollection(ctx, collectionName)

// 列出所有 Collection
collections, err := client.ListCollections(ctx)

// 加载到内存
err = client.LoadCollection(ctx, collectionName, async)

// 释放内存
err = client.ReleaseCollection(ctx, collectionName)

// 重命名
err = client.RenameCollection(ctx, oldName, newName)

// 获取统计信息
stats, err := client.GetCollectionStatistics(ctx, collectionName)
```

### Partition 操作

```go
// 创建分区
err = client.CreatePartition(ctx, collectionName, partitionName)

// 删除分区
err = client.DropPartition(ctx, collectionName, partitionName)

// 检查是否存在
exists, err := client.HasPartition(ctx, collectionName, partitionName)

// 列出所有分区
partitions, err := client.ListPartitions(ctx, collectionName)

// 加载分区
err = client.LoadPartitions(ctx, collectionName, partitionNames, async)

// 释放分区
err = client.ReleasePartitions(ctx, collectionName, partitionNames)
```

### Index 操作

```go
// 创建索引
err = client.CreateIndex(ctx, collectionName, fieldName, &milvus.IndexOptions{
	IndexType:  milvus.IndexTypeHNSW,
	MetricType: milvus.MetricTypeCosine,
	Params:     map[string]interface{}{"M": 16},
})

// 删除索引
err = client.DropIndex(ctx, collectionName, fieldName)

// 描述索引
info, err := client.DescribeIndex(ctx, collectionName, fieldName)
```

### 数据操作

```go
// 插入数据
ids, err := client.Insert(ctx, collectionName, columns, opts)

// Upsert 数据
ids, err := client.Upsert(ctx, collectionName, columns, opts)

// 删除数据
err = client.Delete(ctx, collectionName, expr, opts)

// 刷新数据
err = client.Flush(ctx, collectionName, async)
```

### 搜索和查询

```go
// 向量搜索
results, err := client.Search(ctx, collectionName, vectors, vectorField,
	metricType, topK, opts)

// 标量查询
results, err := client.Query(ctx, collectionName, expr, opts)

// 根据 ID 获取
results, err := client.Get(ctx, collectionName, ids, opts)
```

## 数据类型

### 标量类型

- `DataTypeInt64` - 64位整数
- `DataTypeFloat` - 32位浮点数
- `DataTypeDouble` - 64位浮点数
- `DataTypeVarChar` - 变长字符串
- `DataTypeBool` - 布尔值
- `DataTypeJSON` - JSON 数据

### 向量类型

- `DataTypeFloatVector` - 32位浮点向量
- `DataTypeBinaryVector` - 二进制向量
- `DataTypeFloat16Vector` - 16位浮点向量
- `DataTypeBFloat16Vector` - BFloat16 向量
- `DataTypeSparseFloatVector` - 稀疏浮点向量

## 索引类型

- `IndexTypeFlat` - 暴力搜索
- `IndexTypeIVFFlat` - IVF 索引
- `IndexTypeIVFSQ8` - IVF + 标量量化
- `IndexTypeHNSW` - HNSW 图索引
- `IndexTypeDiskANN` - DiskANN 索引

## 度量类型

- `MetricTypeL2` - 欧氏距离
- `MetricTypeIP` - 内积
- `MetricTypeCosine` - 余弦相似度
- `MetricTypeJaccard` - Jaccard 距离
- `MetricTypeHamming` - 汉明距离

## 错误处理

```go
// 检查错误类型
if milvus.IsNotFound(err) {
	// Collection 不存在
}

if milvus.IsAlreadyExists(err) {
	// Collection 已存在
}

if milvus.IsInvalidArgument(err) {
	// 参数无效
}

if milvus.IsTimeout(err) {
	// 超时
}

if milvus.IsConnectionFailed(err) {
	// 连接失败
}
```

## 工具函数

```go
// 构建列
column := milvus.BuildInt64Column("id", []int64{1, 2, 3})
column := milvus.BuildFloatVectorColumn("vec", 128, vectors)

// 向量归一化
normalized := milvus.NormalizeVector(vector)
normalized := milvus.NormalizeVectors(vectors)

// 构建表达式
expr := milvus.BuildExprIn("id", []interface{}{1, 2, 3})
expr := milvus.BuildExprRange("age", 18, 65)
expr := milvus.BuildExprAnd("id > 0", "age < 100")
expr := milvus.BuildExprOr("name == 'Alice'", "name == 'Bob'")

// 分块处理
chunks := milvus.ChunkSlice(largeSlice, 1000)
```

## 配置说明

```go
type Config struct {
	Address         string        // Milvus 地址，如 "localhost:19530"
	Username        string        // 用户名(可选)
	Password        string        // 密码(可选)
	APIKey          string        // API Key(可选)
	Database        string        // 数据库名，默认 "default"

	MaxIdleConns    int           // 最大空闲连接数
	MaxOpenConns    int           // 最大打开连接数
	ConnMaxLifetime time.Duration // 连接最大生命周期

	DialTimeout     time.Duration // 连接超时
	RequestTimeout  time.Duration // 请求超时
	KeepAlive       time.Duration // Keep-Alive 间隔

	MaxRetries      int           // 最大重试次数
	RetryDelay      time.Duration // 重试延迟

	EnableTLS       bool          // 是否启用 TLS
	TLSMode         string        // TLS 模式
	EnableTracing   bool          // 是否启用追踪
}
```

## 最佳实践

### 1. 连接管理

```go
// 使用连接池
cfg.MaxIdleConns = 10
cfg.MaxOpenConns = 100
cfg.ConnMaxLifetime = 30 * time.Minute

// 设置合理的超时
cfg.DialTimeout = 10 * time.Second
cfg.RequestTimeout = 30 * time.Second
```

### 2. 批量操作

```go
// 分批插入大量数据
chunks := milvus.ChunkSlice(data, 1000)
for _, chunk := range chunks {
	_, err := client.Insert(ctx, collectionName, chunk, nil)
	if err != nil {
		log.Error("insert failed", zap.Error(err))
	}
}
```

### 3. 向量归一化

```go
// 对于余弦相似度，建议先归一化
normalized := milvus.NormalizeVectors(vectors)
results, err := client.Search(ctx, collectionName, normalized,
	"embedding", milvus.MetricTypeCosine, 10, nil)
```

### 4. 索引选择

```go
// 小数据集(<1M): FLAT
// 中等数据集(1M-10M): IVF_FLAT 或 HNSW
// 大数据集(>10M): HNSW 或 DiskANN

// HNSW 适合高召回率场景
opts := &milvus.IndexOptions{
	IndexType:  milvus.IndexTypeHNSW,
	MetricType: milvus.MetricTypeCosine,
	Params: map[string]interface{}{
		"M":              16,  // 连接数，越大召回越高但构建越慢
		"efConstruction": 200, // 构建参数
	},
}
```

### 5. 搜索优化

```go
// 设置搜索参数
opts := &milvus.SearchOptions{
	Params: map[string]interface{}{
		"ef": 64, // HNSW 搜索参数，越大召回越高但速度越慢
	},
	Limit: 100, // 限制返回数量
	Expr:  "id > 0 && category == 'tech'", // 标量过滤
}
```

## License

MIT
