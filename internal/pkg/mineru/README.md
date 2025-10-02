# MinerU 客户端封装

MinerU 是一个文档解析服务，支持 PDF、Word、PPT 等多种格式的文档解析，可以提取文档中的文本、表格、公式等内容。

## 功能特性

- ✅ 单文件解析
- ✅ 批量文件解析（文件上传模式）
- ✅ 批量 URL 解析
- ✅ 任务轮询与进度跟踪
- ✅ 自动重试机制
- ✅ 结果下载
- ✅ 完整的错误处理
- ✅ 日志集成

## 快速开始

### 1. 配置

在 `config.yaml` 中添加 MinerU 配置：

```yaml
mineru:
  base_url: "https://mineru.net"
  api_key: "your-api-key-here"
  timeout: 30s
  max_retries: 3
  default_language: "ch"
  enable_formula: true
  enable_table: true
  model_version: "pipeline"
```

### 2. 初始化客户端

```go
import (
    "ai-writer-backend/internal/conf"
    "ai-writer-backend/internal/pkg/logger"
    "ai-writer-backend/internal/pkg/mineru"
)

// 加载配置
cfg, err := conf.LoadConfig("config.yaml")
if err != nil {
    log.Fatal(err)
}

// 创建 logger
log, err := logger.New(&cfg.Log)
if err != nil {
    log.Fatal(err)
}

// 创建 MinerU 客户端
mineruCfg := &mineru.Config{
    BaseURL:         cfg.MinerU.BaseURL,
    APIKey:          cfg.MinerU.APIKey,
    Timeout:         cfg.MinerU.Timeout,
    MaxRetries:      cfg.MinerU.MaxRetries,
    DefaultLanguage: cfg.MinerU.DefaultLanguage,
    EnableFormula:   cfg.MinerU.EnableFormula,
    EnableTable:     cfg.MinerU.EnableTable,
    ModelVersion:    cfg.MinerU.ModelVersion,
}

client, err := mineru.New(mineruCfg, log)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

## 使用示例

### 单文件解析

#### 1. 创建任务

```go
ctx := context.Background()

req := &mineru.CreateTaskRequest{
    URL:           "https://example.com/document.pdf",
    IsOCR:         true,
    EnableFormula: boolPtr(true),
    EnableTable:   boolPtr(true),
    Language:      "ch",
}

resp, err := client.CreateTask(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Task ID: %s\n", resp.Data.TaskID)
```

#### 2. 查询任务结果

```go
taskID := resp.Data.TaskID

result, err := client.GetTaskResult(ctx, taskID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("State: %s\n", result.State)
if result.State == mineru.TaskStateDone {
    fmt.Printf("Result URL: %s\n", result.FullZipURL)
}
```

#### 3. 创建任务并等待完成

```go
req := &mineru.CreateTaskRequest{
    URL:   "https://example.com/document.pdf",
    IsOCR: true,
}

// 自定义轮询选项
opts := &mineru.PollOptions{
    Interval: 5 * time.Second,
    Timeout:  10 * time.Minute,
    OnProgress: func(progress *mineru.ExtractProgress) {
        fmt.Printf("Progress: %s\n", mineru.FormatTaskProgress(progress))
    },
}

result, err := client.CreateTaskAndWait(ctx, req, opts)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Task completed! Result: %s\n", result.FullZipURL)
```

### 批量URL解析

```go
req := &mineru.BatchTaskRequest{
    Language:      "ch",
    EnableFormula: boolPtr(true),
    EnableTable:   boolPtr(true),
    Files: []mineru.BatchFileInfo{
        {
            URL:    "https://example.com/file1.pdf",
            IsOCR:  true,
            DataID: "file1",
        },
        {
            URL:    "https://example.com/file2.pdf",
            IsOCR:  false,
            DataID: "file2",
        },
    },
}

// 创建批量任务并等待完成
results, err := client.CreateBatchWithURLsAndWait(ctx, req, nil)
if err != nil {
    log.Fatal(err)
}

// 处理结果
for _, result := range results.Data.ExtractResult {
    fmt.Printf("File: %s, State: %s\n", result.FileName, result.State)
    if result.State == mineru.TaskStateDone {
        fmt.Printf("  Result: %s\n", result.FullZipURL)
    } else if result.State == mineru.TaskStateFailed {
        fmt.Printf("  Error: %s\n", result.ErrMsg)
    }
}
```

### 批量文件上传解析

```go
req := &mineru.BatchUploadRequest{
    Language: "ch",
    Files: []mineru.BatchFileInfo{
        {
            Name:   "document1.pdf",
            IsOCR:  true,
            DataID: "doc1",
        },
        {
            Name:   "document2.pdf",
            IsOCR:  false,
            DataID: "doc2",
        },
    },
}

filePaths := []string{
    "/path/to/document1.pdf",
    "/path/to/document2.pdf",
}

// 创建批量任务并上传文件
results, err := client.CreateBatchWithFilesAndWait(ctx, req, filePaths, nil)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Batch completed! %d files processed\n", len(results.Data.ExtractResult))
```

### 下载结果

```go
// 下载单个结果
zipURL := result.FullZipURL
destPath := "/path/to/save/result.zip"

err := client.DownloadResult(ctx, zipURL, destPath)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Result downloaded to: %s\n", destPath)
```

## 辅助函数

### 任务状态检查

```go
// 检查任务是否完成
if mineru.IsTaskCompleted(result.State) {
    fmt.Println("Task is completed")
}

// 检查任务是否成功
if mineru.IsTaskSuccessful(result.State) {
    fmt.Println("Task succeeded")
}

// 检查任务是否失败
if mineru.IsTaskFailed(result.State) {
    fmt.Println("Task failed:", result.ErrMsg)
}

// 检查任务是否正在处理
if mineru.IsTaskProcessing(result.State) {
    fmt.Println("Task is still processing")
}
```

### 批量任务统计

```go
results, err := client.GetBatchResults(ctx, batchID)
if err != nil {
    log.Fatal(err)
}

// 获取统计信息
done, failed, processing := mineru.GetBatchStatistics(results.Data.ExtractResult)
fmt.Printf("Done: %d, Failed: %d, Processing: %d\n", done, failed, processing)

// 过滤成功的结果
successful := mineru.FilterSuccessfulResults(results.Data.ExtractResult)
for _, result := range successful {
    fmt.Printf("Success: %s -> %s\n", result.FileName, result.FullZipURL)
}

// 过滤失败的结果
failedResults := mineru.FilterFailedResults(results.Data.ExtractResult)
for _, result := range failedResults {
    fmt.Printf("Failed: %s -> %s\n", result.FileName, result.ErrMsg)
}
```

### 进度显示

```go
if result.ExtractProgress != nil {
    // 获取进度百分比
    progress := mineru.GetTaskProgress(result.ExtractProgress)
    fmt.Printf("Progress: %.1f%%\n", progress)

    // 格式化进度显示
    fmt.Println(mineru.FormatTaskProgress(result.ExtractProgress))
    // 输出: 5/10 (50.0%)
}
```

## 错误处理

```go
result, err := client.CreateTask(ctx, req)
if err != nil {
    // 检查是否是 MinerU 错误
    if mineruErr, ok := err.(*mineru.MinerUError); ok {
        fmt.Printf("Error Code: %v\n", mineruErr.Code)
        fmt.Printf("Error Message: %s\n", mineruErr.Message)
        fmt.Printf("Trace ID: %s\n", mineruErr.TraceID)

        // 根据错误码处理
        switch mineruErr.Code {
        case "A0202", "A0211":
            // Token 相关错误
            fmt.Println("Please check your API key")
        case mineru.ErrCodeFileSizeExceeded:
            // 文件大小超限
            fmt.Println("File too large, please split it")
        default:
            fmt.Println("Unknown error")
        }
    } else {
        fmt.Printf("Other error: %v\n", err)
    }
}
```

## 常见错误码

| 错误码 | 说明 | 处理建议 |
|--------|------|----------|
| A0202 | Token 错误 | 检查 API Key 是否正确 |
| A0211 | Token 过期 | 更换新的 API Key |
| -500 | 传参错误 | 检查请求参数格式 |
| -60005 | 文件大小超限 | 文件最大 200MB，请拆分 |
| -60012 | 找不到任务 | 检查 task_id 是否正确 |

完整错误码列表请参考 [docs/mineru.md](../../../docs/mineru.md)

## 最佳实践

### 1. 使用轮询选项

```go
opts := &mineru.PollOptions{
    Interval: 5 * time.Second,  // 轮询间隔
    Timeout:  10 * time.Minute, // 超时时间
    OnProgress: func(progress *mineru.ExtractProgress) {
        // 进度回调
        log.Info("progress", "extracted", progress.ExtractedPages, "total", progress.TotalPages)
    },
}
```

### 2. 批量处理大量文件

对于大量文件，建议分批处理，每批 10-20 个文件：

```go
const batchSize = 10

for i := 0; i < len(allFiles); i += batchSize {
    end := i + batchSize
    if end > len(allFiles) {
        end = len(allFiles)
    }

    batch := allFiles[i:end]
    // 处理这一批文件
    results, err := client.CreateBatchWithURLsAndWait(ctx, &mineru.BatchTaskRequest{
        Files: batch,
    }, nil)
    // ... 处理结果
}
```

### 3. 错误重试

客户端已内置重试机制，但对于特殊情况可以手动重试：

```go
var result *mineru.TaskResult
var err error

for i := 0; i < 3; i++ {
    result, err = client.CreateTaskAndWait(ctx, req, nil)
    if err == nil {
        break
    }

    // 检查是否是可重试的错误
    if mineruErr, ok := err.(*mineru.MinerUError); ok {
        if mineruErr.Code == mineru.ErrCodeQueueFull ||
           mineruErr.Code == mineru.ErrCodeServiceError {
            time.Sleep(time.Duration(i+1) * 30 * time.Second)
            continue
        }
    }
    break
}
```

## 配置说明

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| base_url | string | https://mineru.net | API 基础地址 |
| api_key | string | 无 | API 密钥（必填） |
| timeout | duration | 30s | 请求超时时间 |
| max_retries | int | 3 | 最大重试次数 |
| default_language | string | ch | 默认文档语言 |
| enable_formula | bool | true | 默认启用公式识别 |
| enable_table | bool | true | 默认启用表格识别 |
| model_version | string | pipeline | 模型版本（pipeline/vlm） |

## 测试

运行单元测试：

```bash
go test -v ./internal/pkg/mineru/
```

## License

MIT
