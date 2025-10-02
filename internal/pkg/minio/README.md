# MinIO 客户端封装

基于 [minio-go/v7](https://github.com/minio/minio-go) 的 MinIO 对象存储客户端封装，提供简洁易用的 API 和完善的错误处理。

## 特性

- ✅ **完整的 Bucket 操作**：创建、列表、检查存在、删除
- ✅ **对象 CRUD 操作**：上传、下载、复制、删除、元数据查询
- ✅ **预签名 URL**：GET/PUT/HEAD/POST 预签名 URL 生成
- ✅ **标签管理**：Bucket 和 Object 标签操作
- ✅ **工具函数**：Bucket 名称校验、对象名称校验、进度跟踪
- ✅ **统一错误处理**：错误包装和类型判断辅助函数
- ✅ **Context 支持**：所有操作支持超时控制和请求取消
- ✅ **日志集成**：基于 zap 的结构化日志
- ✅ **生产级测试**：基于真实 MinIO 服务器的完整测试覆盖

## 快速开始

### 安装

```bash
go get github.com/minio/minio-go/v7
go get go.uber.org/zap
```

### 初始化客户端

```go
package main

import (
    "context"
    "log"

    "github.com/lk2023060901/ai-writer-backend/internal/pkg/minio"
    "go.uber.org/zap"
)

func main() {
    // 创建配置
    cfg := &minio.Config{
        Endpoint:        "localhost:9000",
        AccessKeyID:     "minioadmin",
        SecretAccessKey: "minioadmin",
        UseSSL:          false,
        Region:          "us-east-1",
    }

    // 创建 logger
    logger, _ := zap.NewProduction()

    // 初始化客户端
    client, err := minio.NewClient(cfg, logger)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 测试连接
    ctx := context.Background()
    if err := client.Ping(ctx); err != nil {
        log.Fatal("Failed to connect to MinIO:", err)
    }

    log.Println("Successfully connected to MinIO!")
}
```

## 使用示例

### Bucket 操作

#### 创建 Bucket

```go
ctx := context.Background()

err := client.MakeBucket(ctx, "my-bucket", minio.MakeBucketOptions{
    Region: "us-east-1",
})
if err != nil {
    if minio.IsBucketAlreadyExists(err) {
        log.Println("Bucket already exists")
    } else {
        log.Fatal(err)
    }
}
```

#### 列出所有 Buckets

```go
buckets, err := client.ListBuckets(ctx)
if err != nil {
    log.Fatal(err)
}

for _, bucket := range buckets {
    log.Printf("Bucket: %s, Created: %s", bucket.Name, bucket.CreationDate)
}
```

#### 检查 Bucket 是否存在

```go
exists, err := client.BucketExists(ctx, "my-bucket")
if err != nil {
    log.Fatal(err)
}

if exists {
    log.Println("Bucket exists")
}
```

#### 删除 Bucket

```go
err := client.RemoveBucket(ctx, "my-bucket")
if err != nil {
    log.Fatal(err)
}
```

### Object 操作

#### 上传对象

```go
import (
    "bytes"
)

content := []byte("Hello, MinIO!")
reader := bytes.NewReader(content)

info, err := client.PutObject(ctx, "my-bucket", "hello.txt", reader, int64(len(content)), minio.PutObjectOptions{
    ContentType: "text/plain",
    UserMetadata: map[string]string{
        "x-amz-meta-author": "John Doe",
    },
})
if err != nil {
    log.Fatal(err)
}

log.Printf("Uploaded: %s, Size: %d, ETag: %s", info.Key, info.Size, info.ETag)
```

#### 上传文件

```go
info, err := client.FPutObject(ctx, "my-bucket", "document.pdf", "/path/to/local/file.pdf", minio.PutObjectOptions{
    ContentType: "application/pdf",
})
if err != nil {
    log.Fatal(err)
}

log.Printf("File uploaded: %s", info.Key)
```

#### 下载对象

```go
import (
    "io"
)

object, err := client.GetObject(ctx, "my-bucket", "hello.txt", minio.GetObjectOptions{})
if err != nil {
    log.Fatal(err)
}
defer object.Close()

// 读取内容
content, err := io.ReadAll(object)
if err != nil {
    log.Fatal(err)
}

log.Printf("Content: %s", string(content))
```

#### 下载文件

```go
err := client.FGetObject(ctx, "my-bucket", "document.pdf", "/path/to/download/file.pdf", minio.GetObjectOptions{})
if err != nil {
    log.Fatal(err)
}

log.Println("File downloaded successfully")
```

#### 获取对象元数据

```go
info, err := client.StatObject(ctx, "my-bucket", "hello.txt", minio.StatObjectOptions{})
if err != nil {
    if minio.IsNotFound(err) {
        log.Println("Object not found")
    } else {
        log.Fatal(err)
    }
}

log.Printf("Object: %s, Size: %d, ContentType: %s", info.Key, info.Size, info.ContentType)
```

#### 复制对象

```go
dst := minio.CopyDestOptions{
    Bucket: "my-bucket",
    Object: "destination.txt",
}

src := minio.CopySrcOptions{
    Bucket: "my-bucket",
    Object: "source.txt",
}

info, err := client.CopyObject(ctx, dst, src)
if err != nil {
    log.Fatal(err)
}

log.Printf("Object copied: %s", info.Key)
```

#### 删除对象

```go
err := client.RemoveObject(ctx, "my-bucket", "hello.txt", minio.RemoveObjectOptions{})
if err != nil {
    log.Fatal(err)
}

log.Println("Object removed successfully")
```

#### 列出对象

```go
objCh, errCh := client.ListObjects(ctx, "my-bucket", minio.ListObjectsOptions{
    Prefix:    "documents/",
    Recursive: true,
})

for {
    select {
    case obj, ok := <-objCh:
        if !ok {
            goto Done
        }
        log.Printf("Object: %s, Size: %d", obj.Key, obj.Size)
    case err := <-errCh:
        if err != nil {
            log.Fatal(err)
        }
    }
}
Done:
```

### 预签名 URL

#### 生成 GET 预签名 URL

```go
import (
    "net/url"
    "time"
)

reqParams := make(url.Values)
reqParams.Set("response-content-disposition", "attachment; filename=\"download.txt\"")

presignedURL, err := client.PresignedGetObject(ctx, "my-bucket", "hello.txt", time.Hour, reqParams)
if err != nil {
    log.Fatal(err)
}

log.Printf("Presigned GET URL: %s", presignedURL.String())
// 此 URL 可以直接在浏览器中访问，有效期 1 小时
```

#### 生成 PUT 预签名 URL

```go
presignedURL, err := client.PresignedPutObject(ctx, "my-bucket", "upload.txt", time.Hour)
if err != nil {
    log.Fatal(err)
}

log.Printf("Presigned PUT URL: %s", presignedURL.String())
// 使用此 URL 可以直接通过 HTTP PUT 上传文件
```

#### 生成 POST 预签名策略

```go
policy := minio.NewPostPolicy()
policy.SetBucket("my-bucket")
policy.SetKey("upload.txt")
policy.SetExpires(time.Now().UTC().Add(time.Hour))
policy.SetContentType("text/plain")
policy.SetContentLengthRange(1, 1024*1024) // 1B to 1MB

presignedURL, formData, err := client.PresignedPostPolicy(ctx, policy)
if err != nil {
    log.Fatal(err)
}

log.Printf("POST URL: %s", presignedURL.String())
for k, v := range formData {
    log.Printf("Form field: %s = %s", k, v)
}
```

### 标签管理

#### 设置 Bucket 标签

```go
tags := map[string]string{
    "Environment": "Production",
    "Team":        "DevOps",
}

err := client.SetBucketTagging(ctx, "my-bucket", tags)
if err != nil {
    log.Fatal(err)
}
```

#### 获取 Bucket 标签

```go
tags, err := client.GetBucketTagging(ctx, "my-bucket")
if err != nil {
    log.Fatal(err)
}

log.Printf("Bucket tags: %v", tags)
```

#### 设置对象标签

```go
tags := map[string]string{
    "Status": "Processed",
    "Version": "v1.0",
}

err := client.PutObjectTagging(ctx, "my-bucket", "document.pdf", tags)
if err != nil {
    log.Fatal(err)
}
```

#### 获取对象标签

```go
tags, err := client.GetObjectTagging(ctx, "my-bucket", "document.pdf")
if err != nil {
    log.Fatal(err)
}

log.Printf("Object tags: %v", tags)
```

## 配置选项

### Config 结构体

```go
type Config struct {
    // 必填字段
    Endpoint        string   // MinIO 服务端点，如 "localhost:9000"
    AccessKeyID     string   // 访问密钥 ID
    SecretAccessKey string   // 访问密钥 Secret

    // 可选字段
    SessionToken    string             // 临时会话 Token（用于临时凭证）
    Region          string             // 区域，如 "us-east-1"
    UseSSL          bool               // 是否使用 HTTPS，默认 false
    BucketLookup    BucketLookupType   // Bucket 查找类型（auto/dns/path）
    Transport       *http.Transport    // 自定义 HTTP 传输
    TraceEnabled    bool               // 是否启用 HTTP 请求追踪

    // 重试和超时配置
    MaxRetries      int                // 最大重试次数，默认 3
    RetryDelay      time.Duration      // 重试间隔，默认 1s
    ConnectTimeout  time.Duration      // 连接超时，默认 10s
    RequestTimeout  time.Duration      // 请求超时，默认 30s
}
```

### BucketLookupType

- `BucketLookupAuto`: 自动选择（默认）
- `BucketLookupDNS`: DNS 风格（bucket.endpoint）
- `BucketLookupPath`: 路径风格（endpoint/bucket）

## 错误处理

### 错误类型判断

```go
err := client.GetObject(ctx, "my-bucket", "non-existent.txt", minio.GetObjectOptions{})

if minio.IsNotFound(err) {
    log.Println("Object not found")
} else if minio.IsAccessDenied(err) {
    log.Println("Access denied")
} else if minio.IsBucketAlreadyExists(err) {
    log.Println("Bucket already exists")
} else if minio.IsInvalidArgument(err) {
    log.Println("Invalid argument")
} else if err != nil {
    log.Fatal("Unknown error:", err)
}
```

### 预定义错误

- `ErrBucketNotFound`: Bucket 不存在
- `ErrObjectNotFound`: 对象不存在
- `ErrInvalidArgument`: 参数无效
- `ErrAccessDenied`: 访问被拒绝
- `ErrBucketAlreadyExists`: Bucket 已存在
- `ErrInvalidBucketName`: Bucket 名称无效
- `ErrInvalidObjectName`: 对象名称无效
- `ErrConnectionFailed`: 连接失败
- `ErrOperationTimeout`: 操作超时

## 工具函数

### Bucket 名称校验

```go
err := minio.ValidateBucketName("my-bucket")
if err != nil {
    log.Fatal("Invalid bucket name:", err)
}
```

### 对象名称校验

```go
err := minio.ValidateObjectName("path/to/my-object.txt")
if err != nil {
    log.Fatal("Invalid object name:", err)
}
```

### Content-Type 检测

```go
contentType := minio.DetectContentType("/path/to/file.pdf")
log.Printf("Content-Type: %s", contentType) // Output: application/pdf
```

### 进度跟踪

```go
import (
    "os"
)

file, _ := os.Open("/path/to/large-file.zip")
defer file.Close()

fileStat, _ := file.Stat()
fileSize := fileStat.Size()

// 创建带进度回调的 Reader
progressReader := minio.NewProgressReader(file, fileSize, func(current, total int64) {
    percentage := float64(current) / float64(total) * 100
    log.Printf("Upload progress: %.2f%% (%s / %s)",
        percentage,
        minio.FormatBytes(current),
        minio.FormatBytes(total))
})

info, err := client.PutObject(ctx, "my-bucket", "large-file.zip", progressReader, fileSize, minio.PutObjectOptions{
    ContentType: "application/zip",
})
```

## 测试

### 运行测试

```bash
# 确保 MinIO 服务正在运行（localhost:9000）
docker run -d -p 9000:9000 -p 9001:9001 \
  -e "MINIO_ROOT_USER=minioadmin" \
  -e "MINIO_ROOT_PASSWORD=minioadmin" \
  minio/minio server /data --console-address ":9001"

# 运行所有测试
go test -v ./internal/pkg/minio/...

# 运行特定测试
go test -v ./internal/pkg/minio/... -run TestPutObject
```

### 测试覆盖

- ✅ 客户端初始化和配置校验
- ✅ Bucket CRUD 操作
- ✅ 对象上传/下载（内存和文件）
- ✅ 对象元数据查询
- ✅ 对象复制和删除
- ✅ 预签名 URL 生成和验证
- ✅ 标签管理
- ✅ 错误处理和类型判断

总计 **50+ 测试用例**，所有测试基于真实 MinIO 服务器运行。

## API 参考

### Client 方法

#### Bucket 操作
- `MakeBucket(ctx, bucketName, opts)` - 创建 Bucket
- `ListBuckets(ctx)` - 列出所有 Buckets
- `BucketExists(ctx, bucketName)` - 检查 Bucket 是否存在
- `RemoveBucket(ctx, bucketName)` - 删除 Bucket
- `ListObjects(ctx, bucketName, opts)` - 列出对象
- `ListIncompleteUploads(ctx, bucketName, prefix, recursive)` - 列出未完成的分片上传

#### Object 操作
- `PutObject(ctx, bucket, object, reader, size, opts)` - 上传对象
- `GetObject(ctx, bucket, object, opts)` - 下载对象
- `FPutObject(ctx, bucket, object, filePath, opts)` - 上传文件
- `FGetObject(ctx, bucket, object, filePath, opts)` - 下载到文件
- `StatObject(ctx, bucket, object, opts)` - 获取对象元数据
- `RemoveObject(ctx, bucket, object, opts)` - 删除对象
- `CopyObject(ctx, dst, src)` - 复制对象
- `RemoveIncompleteUpload(ctx, bucket, object)` - 删除未完成上传

#### 预签名操作
- `PresignedGetObject(ctx, bucket, object, expiry, reqParams)` - 生成 GET 预签名 URL
- `PresignedPutObject(ctx, bucket, object, expiry)` - 生成 PUT 预签名 URL
- `PresignedHeadObject(ctx, bucket, object, expiry, reqParams)` - 生成 HEAD 预签名 URL
- `PresignedPostPolicy(ctx, policy)` - 生成 POST 预签名策略

#### 标签操作
- `SetBucketTagging(ctx, bucket, tags)` - 设置 Bucket 标签
- `GetBucketTagging(ctx, bucket)` - 获取 Bucket 标签
- `RemoveBucketTagging(ctx, bucket)` - 删除 Bucket 标签
- `PutObjectTagging(ctx, bucket, object, tags)` - 设置对象标签
- `GetObjectTagging(ctx, bucket, object)` - 获取对象标签
- `RemoveObjectTagging(ctx, bucket, object)` - 删除对象标签

#### 工具方法
- `Ping(ctx)` - 检查连接
- `Close()` - 关闭客户端
- `IsClosed()` - 检查客户端是否已关闭
- `GetUnderlyingClient()` - 获取底层 MinIO 客户端

## 最佳实践

1. **使用 Context 超时控制**
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   ```

2. **正确处理错误**
   ```go
   err := client.PutObject(ctx, bucket, object, reader, size, opts)
   if err != nil {
       if minio.IsAccessDenied(err) {
           // 权限不足
       } else if minio.IsInvalidArgument(err) {
           // 参数错误
       } else {
           // 其他错误
       }
   }
   ```

3. **关闭资源**
   ```go
   object, err := client.GetObject(ctx, bucket, key, opts)
   if err != nil {
       return err
   }
   defer object.Close() // 务必关闭
   ```

4. **使用结构化日志**
   ```go
   logger, _ := zap.NewProduction()
   client, _ := minio.NewClient(cfg, logger)
   // 所有操作会自动记录结构化日志
   ```

5. **配置合理的超时和重试**
   ```go
   cfg := &minio.Config{
       // ... 其他配置
       MaxRetries:     3,
       RetryDelay:     time.Second,
       ConnectTimeout: 10 * time.Second,
       RequestTimeout: 30 * time.Second,
   }
   ```

## 参考资料

- [MinIO Go Client SDK](https://github.com/minio/minio-go)
- [MinIO 官方文档](https://min.io/docs/minio/linux/index.html)
- [AWS S3 API 参考](https://docs.aws.amazon.com/AmazonS3/latest/API/Welcome.html)

## 许可证

本项目遵循 MIT 许可证。
