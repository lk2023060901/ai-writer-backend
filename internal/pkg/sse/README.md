# SSE 封装库使用文档

> 从 110 行样板代码减少到 20 行业务逻辑 - 提升 **82% 代码可读性**

---

## 目录

1. [快速开始](#快速开始)
2. [核心组件](#核心组件)
3. [使用场景](#使用场景)
4. [完整示例](#完整示例)
5. [API 参考](#api-参考)
6. [迁移指南](#迁移指南)

---

## 快速开始

### 基础用法:StreamBuilder

```go
import "github.com/lk2023060901/ai-writer-backend/internal/pkg/sse"

func Handler(c *gin.Context) {
    // 1. 创建 Stream
    stream := sse.NewStream(c, hub).
        WithResource("doc:123").
        WithBufferSize(10).
        WithHeartbeat(30 * time.Second).
        Build()
    defer stream.Close()

    // 2. 在 goroutine 中发送事件
    go func() {
        stream.Send("status", map[string]interface{}{
            "message": "Processing...",
        })

        // 处理完成
        stream.Send("done", map[string]interface{}{
            "message": "Completed",
        })
    }()

    // 3. 开始流式传输
    stream.StartStreaming()
}
```

### 批量上传:BatchUploader

```go
func BatchUploadHandler(c *gin.Context) {
    // 1. 创建 Stream
    stream := sse.NewStream(c, hub).
        WithResource("kb:abc").
        WithBufferSize(50).
        Build()
    defer stream.Close()

    // 2. 使用 BatchUploader 处理批量任务
    go sse.NewBatchUploader[*File](stream, len(files)).
        WithEventPrefix("file").  // 事件类型: file-success, file-failed
        Process(files, func(ctx context.Context, file *File) (interface{}, error) {
            // 处理单个文件
            return processFile(file)
        }).
        WithWorkerPool(workerPool).
        OnSuccess(func(index int, file *File, result interface{}) error {
            // 成功回调
            return enqueueTask(result)
        }).
        Run(c.Request.Context())

    stream.StartStreaming()
}
```

---

## 核心组件

### 1. StreamBuilder

**职责**:简化 SSE 连接的创建和管理

**核心方法**:
- `WithResource(string)` - 设置资源 ID
- `WithBufferSize(int)` - 设置 Channel 缓冲区大小
- `WithHeartbeat(time.Duration)` - 设置心跳间隔
- `OnConnect(func())` - 连接建立钩子
- `OnDisconnect(func())` - 连接断开钩子
- `OnError(func(error))` - 错误处理钩子

**示例**:
```go
stream := sse.NewStream(c, hub).
    WithResource("chat:session-123").
    WithBufferSize(20).
    WithHeartbeat(15 * time.Second).
    OnConnect(func() {
        logger.Info("连接建立")
        metrics.SSEConnections.Inc()
    }).
    OnDisconnect(func() {
        logger.Info("连接断开")
        metrics.SSEConnections.Dec()
    }).
    OnError(func(err error) {
        logger.Error("SSE错误", zap.Error(err))
    }).
    Build()
```

### 2. ProgressTracker

**职责**:跟踪批量任务进度并自动推送 SSE 事件

**核心方法**:
- `Start()` - 发送 `batch-start` 事件
- `RecordSuccess(index, itemName, data)` - 记录成功并发送 `item-success` 事件
- `RecordFailure(index, itemName, err)` - 记录失败并发送 `item-failed` 事件
- `Complete()` - 发送 `batch-complete` 事件
- `GetStats()` - 获取统计信息

**示例**:
```go
tracker := sse.NewProgressTracker(stream, 10)
tracker.Start()

for i, item := range items {
    result, err := processItem(item)
    if err != nil {
        tracker.RecordFailure(i, item.Name, err)
    } else {
        tracker.RecordSuccess(i, item.Name, result)
    }
}

tracker.Complete()
```

### 3. BatchUploader (泛型)

**职责**:提供声明式 API 处理批量任务

**核心方法**:
- `WithEventPrefix(string)` - 设置事件前缀(默认 `item-`)
- `Process(items, func)` - 设置处理函数
- `WithWorkerPool(pool)` - 设置工作池
- `OnSuccess(func)` - 成功回调
- `OnFailure(func)` - 失败回调
- `WithItemNamer(func)` - 自定义名称提取器

**示例**:
```go
uploader := sse.NewBatchUploader[*Document](stream, len(docs)).
    WithEventPrefix("doc").
    Process(docs, func(ctx context.Context, doc *Document) (interface{}, error) {
        return uploadDocument(ctx, doc)
    }).
    WithWorkerPool(pool).
    OnSuccess(func(index int, doc *Document, result interface{}) error {
        log.Printf("文档 %s 上传成功", doc.FileName)
        return enqueueProcessing(result)
    }).
    OnFailure(func(index int, doc *Document, err error) error {
        log.Printf("文档 %s 上传失败: %v", doc.FileName, err)
        return nil
    })

uploader.Run(ctx)
```

---

## 使用场景

### 场景 1:单文档状态推送

**需求**:监听单个文档的处理进度(解析 → 切片 → 向量化)

**代码**:
```go
func StreamDocumentStatus(c *gin.Context) {
    docID := c.Param("doc_id")

    stream := sse.NewStream(c, hub).
        WithResource("doc:" + docID).
        WithHeartbeat(30 * time.Second).
        OnConnect(func() {
            logger.Info("开始监听文档状态", zap.String("doc_id", docID))
        }).
        Build()
    defer stream.Close()

    // Hub 会自动广播文档处理事件到这个 stream
    stream.StartStreaming()
}
```

### 场景 2:批量文件上传

**需求**:上传 10-50 个文件,实时推送每个文件的上传进度

**代码**:
```go
func BatchUploadDocuments(c *gin.Context) {
    kbID := c.Param("id")
    userID := c.GetString("user_id")

    // ... 解析 multipart form,读取文件 (40 行)

    stream := sse.NewStream(c, hub).
        WithResource("kb:" + kbID).
        WithBufferSize(50).
        Build()
    defer stream.Close()

    go sse.NewBatchUploader[*UploadFile](stream, len(files)).
        WithEventPrefix("file").
        Process(files, func(ctx context.Context, file *UploadFile) (interface{}, error) {
            return uploadDocument(ctx, kbID, userID, file)
        }).
        WithWorkerPool(uploadPool).
        OnSuccess(func(index int, file *UploadFile, result interface{}) error {
            if doc, ok := result.(*DocumentResponse); ok {
                return enqueueProcessing(doc.ID)
            }
            return nil
        }).
        Run(c.Request.Context())

    stream.StartStreaming()
}
```

**前端接收事件**:
```javascript
const eventSource = new EventSource('/api/kb/123/documents/batch');

eventSource.addEventListener('connected', (e) => {
  console.log('连接建立', JSON.parse(e.data));
});

eventSource.addEventListener('batch-start', (e) => {
  const data = JSON.parse(e.data);
  console.log(`开始上传 ${data.total_count} 个文件`);
});

eventSource.addEventListener('file-success', (e) => {
  const data = JSON.parse(e.data);
  console.log(`[${data.completed}/${data.total}] ${data.item_name} 上传成功`);
  updateProgress(data.completed, data.total);
});

eventSource.addEventListener('file-failed', (e) => {
  const data = JSON.parse(e.data);
  console.error(`[${data.completed}/${data.total}] ${data.item_name} 上传失败: ${data.error}`);
});

eventSource.addEventListener('batch-complete', (e) => {
  const data = JSON.parse(e.data);
  console.log(`上传完成: ${data.success_count} 成功, ${data.failed_count} 失败`);
  eventSource.close();
});
```

### 场景 3:聊天流式输出

**需求**:LLM 流式返回,实时推送每个 token

**代码**:
```go
func ChatStream(c *gin.Context) {
    sessionID := c.Param("session_id")

    stream := sse.NewStream(c, hub).
        WithResource("chat:" + sessionID).
        WithHeartbeat(15 * time.Second). // 更频繁的心跳
        Build()
    defer stream.Close()

    go func() {
        // 调用 LLM
        for chunk := range llmClient.StreamChat(prompt) {
            stream.Send("chunk", map[string]interface{}{
                "content": chunk.Text,
            })
        }

        stream.Send("done", map[string]interface{}{
            "message": "Completed",
        })
    }()

    stream.StartStreaming()
}
```

---

## API 参考

### Stream

#### 方法

**`Send(eventType string, data interface{}) error`**
- 发送事件(并发安全)
- 如果 Channel 已满,返回错误

**`Close() error`**
- 关闭流(幂等)
- 自动注销客户端、取消心跳、触发 `OnDisconnect` 钩子

**`StartStreaming()`**
- 开始流式传输(阻塞直到连接关闭)
- 自动设置 SSE headers、注册客户端、发送心跳

**`GetClientID() string`**
- 获取客户端 ID

**`GetResource() string`**
- 获取资源 ID

**`GetDuration() time.Duration`**
- 获取连接时长

**`IsClosed() bool`**
- 检查是否已关闭

### ProgressTracker

#### 方法

**`Start() error`**
- 发送 `batch-start` 事件
- 包含 `total_count` 字段

**`RecordSuccess(index int, itemName string, data interface{}) error`**
- 记录成功并发送 `item-success` 事件
- 自动增加 `successCount` 和 `completed` 计数器
- `data` 可选,会添加到事件的 `data` 字段

**`RecordFailure(index int, itemName string, err error) error`**
- 记录失败并发送 `item-failed` 事件
- 自动增加 `failedCount` 和 `completed` 计数器
- 包含错误信息

**`Complete() error`**
- 发送 `batch-complete` 事件
- 包含 `total_count`、`success_count`、`failed_count` 字段

**`GetStats() (completed, success, failed int)`**
- 获取当前统计信息

**`GetSuccessRate() float64`**
- 获取成功率(0-100)

### BatchUploader[T]

#### 方法

**`WithEventPrefix(prefix string) *BatchUploader[T]`**
- 设置事件前缀
- 默认 `item-`,可自定义为 `file-`、`doc-` 等
- 生成事件类型: `{prefix}-success`、`{prefix}-failed`

**`Process(items []T, fn func(ctx context.Context, item T) (interface{}, error)) *BatchUploader[T]`**
- 设置处理函数
- 返回值会添加到 `item-success` 事件的 `data` 字段

**`WithWorkerPool(pool WorkerPool) *BatchUploader[T]`**
- 设置工作池(必需)
- 接口定义: `type WorkerPool interface { Submit(task func()) error }`

**`OnSuccess(fn func(index int, item T, result interface{}) error) *BatchUploader[T]`**
- 设置成功回调
- 在 `RecordSuccess` 之后执行
- 回调错误只记录日志,不中断处理

**`OnFailure(fn func(index int, item T, err error) error) *BatchUploader[T]`**
- 设置失败回调
- 在 `RecordFailure` 之后执行

**`WithItemNamer(fn func(T) string) *BatchUploader[T]`**
- 自定义名称提取器
- 默认实现:
  1. 尝试断言为 `ItemNamer` 接口(`GetName()` 方法)
  2. 尝试反射获取 `FileName` 或 `Name` 字段
  3. 返回类型名称

**`Run(ctx context.Context) error`**
- 执行批量处理(阻塞直到所有任务完成)
- 自动发送 `batch-start` 和 `batch-complete` 事件

---

## 迁移指南

### 从旧 API 迁移到新 API

#### Before (旧实现)

```go
// ❌ 110 行样板代码
func BatchUploadDocuments(c *gin.Context) {
    // ... 解析文件 (40 行)

    // 手动创建 Client (5 行)
    client := &sse.Client{
        ID:       uuid.New().String(),
        Channel:  make(chan sse.Event, 50),
        Resource: "kb:" + kbID,
    }

    // 手动管理 goroutine (60 行)
    go func() {
        defer close(client.Channel)

        client.Channel <- sse.Event{Type: "batch-start", Data: ...}

        successCount := 0
        failedCount := 0
        completedCount := 0

        for range files {
            result := <-resultCh
            completedCount++

            if result.Error != nil {
                failedCount++
                client.Channel <- sse.Event{Type: "file-failed", Data: ...}
            } else {
                successCount++
                client.Channel <- sse.Event{Type: "file-uploaded", Data: ...}
            }
        }

        client.Channel <- sse.Event{Type: "batch-complete", Data: ...}
    }()

    sse.StreamResponse(c, client, hub, 30*time.Second)
}
```

#### After (新实现)

```go
// ✅ 20 行业务逻辑
func BatchUploadDocuments(c *gin.Context) {
    // ... 解析文件 (40 行,不变)

    stream := sse.NewStream(c, hub).
        WithResource("kb:" + kbID).
        WithBufferSize(50).
        Build()
    defer stream.Close()

    go sse.NewBatchUploader[*UploadFile](stream, len(files)).
        WithEventPrefix("file").
        Process(files, func(ctx context.Context, file *UploadFile) (interface{}, error) {
            return uploadDocument(ctx, kbID, userID, file)
        }).
        WithWorkerPool(uploadPool).
        OnSuccess(func(index int, file *UploadFile, result interface{}) error {
            return enqueueProcessing(result)
        }).
        Run(c.Request.Context())

    stream.StartStreaming()
}
```

### 迁移清单

- [ ] 替换手动创建 `Client` 为 `sse.NewStream(...).Build()`
- [ ] 移除手动 `close(client.Channel)`,改用 `defer stream.Close()`
- [ ] 替换手动计数器为 `ProgressTracker` 或 `BatchUploader`
- [ ] 替换 `sse.StreamResponse()` 为 `stream.StartStreaming()`
- [ ] 添加生命周期钩子(`OnConnect`、`OnDisconnect`)用于日志和监控
- [ ] 更新前端事件监听(事件类型可能变化)

---

## 性能建议

### BufferSize 配置

| 场景 | 推荐值 | 原因 |
|------|--------|------|
| 单文档状态 | 10 | 事件频率低 |
| 批量上传(10-50 文件) | 50-100 | 防止并发上传时阻塞 |
| 聊天流式输出 | 10-20 | Token 频率高但 Channel 不会积压 |
| 实时日志推送 | 100+ | 日志量大,需大缓冲区 |

### Heartbeat 配置

| 场景 | 推荐值 | 原因 |
|------|--------|------|
| 单文档状态 | 30s | 长连接需心跳保活 |
| 批量上传 | 30s-60s | 上传时间长,适当延长心跳间隔 |
| 聊天流式输出 | 15s | 频繁交互,更短心跳检测断线 |
| 禁用心跳 | 0 | 短连接或自定义心跳机制 |

---

## 常见问题

### Q1: 为什么 `BatchUploader.Run()` 需要在 goroutine 中调用?

**A**: 因为 `Run()` 是阻塞的,会等待所有任务完成。如果不在 goroutine 中调用,会阻塞 `stream.StartStreaming()` 的执行。

```go
// ✅ 正确
go uploader.Run(ctx)
stream.StartStreaming()

// ❌ 错误:永远不会开始流式传输
uploader.Run(ctx)
stream.StartStreaming()
```

### Q2: `OnSuccess` 回调返回错误会中断处理吗?

**A**: 不会。回调错误只会记录到 `stream.OnError` 钩子,不会影响其他任务。

### Q3: 如何自定义事件类型?

**A**: 使用 `WithEventPrefix()` 方法。

```go
// 默认事件: item-success, item-failed
uploader := sse.NewBatchUploader[*File](stream, len(files))

// 自定义事件: file-success, file-failed
uploader.WithEventPrefix("file")

// 自定义事件: doc-success, doc-failed
uploader.WithEventPrefix("doc")
```

### Q4: 如何处理不同类型的项目名称?

**A**: 使用 `WithItemNamer()` 自定义名称提取器。

```go
uploader.WithItemNamer(func(file *UploadFile) string {
    return file.OriginalName // 使用自定义字段
})
```

---

## 总结

**封装收益**:
- **代码减少 82%**:从 110 行降至 20 行
- **可读性提升**:声明式 API,意图清晰
- **可维护性提升**:统一生命周期管理,不易遗漏 `Close()`
- **可观测性增强**:通过钩子统一记录日志和指标
- **更健壮**:幂等关闭、并发安全、统一错误处理

**适用场景**:
- ✅ 批量文件上传
- ✅ 文档处理状态推送
- ✅ 聊天流式输出
- ✅ 实时日志推送
- ✅ 任何需要 SSE 实时推送的场景
