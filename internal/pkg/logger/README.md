# Logger Package

基于 Zap 和 Lumberjack 封装的高性能结构化日志库。

## 功能特性

- ✅ **高性能**: 基于 Uber Zap，零分配设计
- ✅ **日志轮转**: Lumberjack 支持按大小/时间自动轮转
- ✅ **多输出**: 支持控制台、文件、双输出
- ✅ **结构化**: JSON 和 Console 两种格式
- ✅ **上下文支持**: 支持 TraceID、RequestID、UserID
- ✅ **Gin 中间件**: 开箱即用的 HTTP 请求日志
- ✅ **gRPC 拦截器**: Unary 和 Stream 调用日志支持
- ✅ **灵活配置**: 支持配置文件和代码配置

## 快速开始

### 1. 基础使用

```go
package main

import (
    "github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
    "go.uber.org/zap"
)

func main() {
    // 使用默认配置
    log, err := logger.New(logger.DefaultConfig())
    if err != nil {
        panic(err)
    }
    defer log.Sync()

    // 记录日志
    log.Info("application started",
        zap.String("version", "1.0.0"),
        zap.Int("port", 8080),
    )
}
```

### 2. 配置文件使用

在 `config.yaml` 中配置:

```yaml
log:
  level: "info"              # debug/info/warn/error
  format: "json"             # json/console
  output: "both"             # console/file/both
  enablecaller: true
  enablestacktrace: true
  file:
    filename: "logs/app.log"
    maxsize: 100             # MB
    maxage: 30               # days
    maxbackups: 10
    compress: true
```

加载配置:

```go
config, _ := conf.LoadConfig("config.yaml")
log, _ := logger.New(&logger.Config{
    Level:            config.Log.Level,
    Format:           config.Log.Format,
    Output:           config.Log.Output,
    EnableCaller:     config.Log.EnableCaller,
    EnableStacktrace: config.Log.EnableStacktrace,
    File: logger.FileConfig{
        Filename:   config.Log.File.Filename,
        MaxSize:    config.Log.File.MaxSize,
        MaxAge:     config.Log.File.MaxAge,
        MaxBackups: config.Log.File.MaxBackups,
        Compress:   config.Log.File.Compress,
    },
})
```

### 3. 使用选项模式

```go
// 开发环境
log, _ := logger.Development()

// 生产环境
log, _ := logger.Production("logs/app.log")

// 自定义配置
log, _ := logger.NewWithOptions(
    logger.WithLevel("debug"),
    logger.WithFormat("console"),
    logger.WithOutput("both"),
    logger.WithFilename("logs/custom.log"),
    logger.WithMaxSize(50),
    logger.WithCaller(true),
)
```

## 高级功能

### 1. 上下文日志

```go
import "context"

// 添加上下文信息
ctx := context.Background()
ctx = logger.WithRequestID(ctx, "req-123")
ctx = logger.WithTraceID(ctx, "trace-456")
ctx = logger.WithUserID(ctx, "user-789")

// 使用上下文日志
log := logger.FromContext(ctx)
log.Info("processing request")
// 输出: {"request_id":"req-123","trace_id":"trace-456","user_id":"user-789",...}

// 或使用便捷函数
logger.InfoContext(ctx, "processing request")
```

### 2. Gin 中间件

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
)

func main() {
    log, _ := logger.New(logger.DefaultConfig())

    router := gin.New()

    // 使用日志中间件
    router.Use(logger.GinLogger(log))
    router.Use(logger.GinRecovery(log))

    // 或使用自定义配置
    router.Use(logger.GinLoggerWithConfig(log, logger.MiddlewareOptions{
        SkipPaths: []string{"/health"},
        SkipPathPrefixes: []string{"/metrics"},
    }))
}
```

自动记录每个请求:
```json
{
  "level": "info",
  "time": "2025-10-01T17:00:00.000+0800",
  "msg": "HTTP Request",
  "request_id": "uuid-xxx",
  "method": "GET",
  "path": "/api/v1/users",
  "status": 200,
  "latency": "15.2ms",
  "ip": "127.0.0.1"
}
```

### 3. gRPC 拦截器

```go
import (
    "google.golang.org/grpc"
    "github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
)

func main() {
    log, _ := logger.New(logger.DefaultConfig())

    // 创建 gRPC 服务端（带日志和恢复拦截器）
    server := grpc.NewServer(
        grpc.UnaryInterceptor(logger.ChainUnaryServer(
            logger.RecoveryInterceptor(log),        // Panic 恢复
            logger.UnaryServerInterceptor(log),     // 日志记录
        )),
        grpc.StreamInterceptor(logger.ChainStreamServer(
            logger.RecoveryStreamInterceptor(log),  // Stream Panic 恢复
            logger.StreamServerInterceptor(log),    // Stream 日志记录
        )),
    )

    // 或使用自定义配置
    opts := logger.GRPCInterceptorOptions{
        SkipMethods: []string{"/grpc.health.v1.Health/Check"},
        LogPayload:  true,   // 记录请求/响应数据
        LogMetadata: true,   // 记录 gRPC metadata
    }

    server := grpc.NewServer(
        grpc.UnaryInterceptor(logger.UnaryServerInterceptorWithConfig(log, opts)),
        grpc.StreamInterceptor(logger.StreamServerInterceptorWithConfig(log, opts)),
    )
}
```

gRPC 客户端日志:

```go
// 创建 gRPC 客户端
conn, err := grpc.Dial(
    "localhost:9090",
    grpc.WithInsecure(),
    grpc.WithUnaryInterceptor(logger.UnaryClientInterceptor(log)),
    grpc.WithStreamInterceptor(logger.StreamClientInterceptor(log)),
)
```

自动记录 gRPC 调用:
```json
{
  "level": "info",
  "time": "2025-10-01T17:00:00.000+0800",
  "msg": "gRPC call",
  "request_id": "uuid-xxx",
  "method": "/api.UserService/GetUser",
  "service": "api.UserService",
  "rpc": "GetUser",
  "code": "OK",
  "latency": "12.5ms"
}
```

### 4. 全局日志器

```go
// 初始化全局日志器
logger.InitGlobal(logger.DefaultConfig())

// 在任何地方使用
logger.Info("global log message", zap.String("key", "value"))
logger.Error("error occurred", zap.Error(err))

// 获取全局日志器
log := logger.L()
log.Info("using global logger")
```

### 5. 子日志器

```go
// 创建带命名空间的子日志器
apiLogger := log.Named("api")
apiLogger.Info("API request")
// 输出: {"logger":"api","msg":"API request",...}

// 创建带预设字段的子日志器
userLogger := log.With(
    zap.String("module", "user"),
    zap.String("version", "v1"),
)
userLogger.Info("user created")
// 输出: {"module":"user","version":"v1","msg":"user created",...}
```

## 日志级别

从低到高:

1. `Debug` - 调试信息
2. `Info` - 常规信息
3. `Warn` - 警告信息
4. `Error` - 错误信息（自动记录堆栈）
5. `Fatal` - 致命错误（记录后退出程序）
6. `Panic` - Panic 错误（记录后 panic）

```go
log.Debug("debug message", zap.Any("data", debugData))
log.Info("info message", zap.String("status", "ok"))
log.Warn("warning message", zap.Int("retry", 3))
log.Error("error occurred", zap.Error(err))
log.Fatal("fatal error", zap.Error(err)) // 程序退出
```

## 常用字段

```go
import "go.uber.org/zap"

log.Info("message",
    zap.String("string_key", "value"),
    zap.Int("int_key", 123),
    zap.Int64("int64_key", 123456789),
    zap.Bool("bool_key", true),
    zap.Float64("float_key", 3.14),
    zap.Duration("latency", time.Since(start)),
    zap.Time("timestamp", time.Now()),
    zap.Error(err),
    zap.Any("complex", complexObject),
    zap.Stack("stacktrace"),
)
```

## 文件轮转

Lumberjack 自动管理日志文件轮转:

- **按大小轮转**: 当日志文件达到 `maxsize` MB 时自动轮转
- **按时间清理**: 保留最近 `maxage` 天的日志
- **限制备份数**: 最多保留 `maxbackups` 个备份文件
- **自动压缩**: 启用 `compress` 后自动压缩旧日志为 `.gz`

文件命名示例:
```
logs/app.log           # 当前日志文件
logs/app-2025-10-01.log.gz  # 压缩的旧日志
logs/app-2025-09-30.log.gz
```

## 性能优化

### 1. 采样 (Sampling)

对于高频日志，可以使用采样减少 I/O:

```go
import "go.uber.org/zap/zapcore"

core := zapcore.NewSamplerWithOptions(
    zapcore.NewCore(encoder, writer, level),
    time.Second,  // 每秒
    100,          // 前 100 条全部记录
    10,           // 之后每 10 条记录 1 条
)
```

### 2. 异步写入

Zap 本身已经优化了性能，对于极高频场景可以考虑缓冲写入。

### 3. 条件编译

开发环境使用 Console 格式，生产环境使用 JSON:

```go
var log *logger.Logger
if os.Getenv("ENV") == "production" {
    log, _ = logger.Production("logs/app.log")
} else {
    log, _ = logger.Development()
}
```

## 最佳实践

1. **始终 defer Sync()**: 确保日志刷新到磁盘
   ```go
   log, _ := logger.New(config)
   defer log.Sync()
   ```

2. **使用结构化字段**: 避免字符串拼接
   ```go
   // ✅ 好
   log.Info("user login", zap.String("user_id", userID))

   // ❌ 差
   log.Info(fmt.Sprintf("user %s login", userID))
   ```

3. **错误日志带堆栈**: Error 级别自动记录堆栈
   ```go
   log.Error("database error", zap.Error(err))
   ```

4. **使用上下文传递日志器**: 在 HTTP handler 中
   ```go
   func Handler(c *gin.Context) {
       log := logger.FromContext(c.Request.Context())
       log.Info("processing request")
   }
   ```

5. **按模块命名日志器**: 便于过滤和调试
   ```go
   dbLogger := log.Named("database")
   apiLogger := log.Named("api")
   ```

6. **gRPC 服务使用拦截器链**: 组合多个拦截器
   ```go
   // 正确的顺序：先恢复 panic，再记录日志
   server := grpc.NewServer(
       grpc.UnaryInterceptor(logger.ChainUnaryServer(
           logger.RecoveryInterceptor(log),
           logger.UnaryServerInterceptor(log),
       )),
   )
   ```

## gRPC 拦截器详解

### 功能特性

- ✅ **Unary/Stream 支持**: 完整支持两种调用方式
- ✅ **自动 RequestID**: 自动生成或从 metadata 提取
- ✅ **Panic 恢复**: 捕获并记录 panic，返回 Internal 错误
- ✅ **智能日志级别**: 根据状态码自动选择日志级别
- ✅ **拦截器链**: 支持组合多个拦截器
- ✅ **可选配置**: 支持跳过特定方法、记录 payload 等

### 服务端拦截器

#### UnaryServerInterceptor

记录 Unary RPC 调用:

```go
interceptor := logger.UnaryServerInterceptor(log)

server := grpc.NewServer(
    grpc.UnaryInterceptor(interceptor),
)
```

#### UnaryServerInterceptorWithConfig

带自定义配置的拦截器:

```go
opts := logger.GRPCInterceptorOptions{
    SkipMethods: []string{
        "/grpc.health.v1.Health/Check",
        "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
    },
    LogPayload:  true,   // 记录请求和响应
    LogMetadata: true,   // 记录 gRPC metadata
}

interceptor := logger.UnaryServerInterceptorWithConfig(log, opts)
```

#### StreamServerInterceptor

记录 Stream RPC 调用:

```go
interceptor := logger.StreamServerInterceptor(log)

server := grpc.NewServer(
    grpc.StreamInterceptor(interceptor),
)
```

### 客户端拦截器

#### UnaryClientInterceptor

记录客户端 Unary 调用:

```go
conn, err := grpc.Dial(
    target,
    grpc.WithUnaryInterceptor(logger.UnaryClientInterceptor(log)),
)
```

#### StreamClientInterceptor

记录客户端 Stream 调用:

```go
conn, err := grpc.Dial(
    target,
    grpc.WithStreamInterceptor(logger.StreamClientInterceptor(log)),
)
```

### Panic 恢复拦截器

#### RecoveryInterceptor

捕获 Unary handler 中的 panic:

```go
server := grpc.NewServer(
    grpc.UnaryInterceptor(logger.RecoveryInterceptor(log)),
)
```

#### RecoveryStreamInterceptor

捕获 Stream handler 中的 panic:

```go
server := grpc.NewServer(
    grpc.StreamInterceptor(logger.RecoveryStreamInterceptor(log)),
)
```

### 拦截器链

组合多个拦截器:

```go
// Unary 拦截器链
unaryChain := logger.ChainUnaryServer(
    logger.RecoveryInterceptor(log),      // 第一个：恢复 panic
    logger.UnaryServerInterceptor(log),   // 第二个：记录日志
    // 可以添加更多拦截器...
)

// Stream 拦截器链
streamChain := logger.ChainStreamServer(
    logger.RecoveryStreamInterceptor(log),
    logger.StreamServerInterceptor(log),
)

server := grpc.NewServer(
    grpc.UnaryInterceptor(unaryChain),
    grpc.StreamInterceptor(streamChain),
)
```

### 日志输出示例

#### 成功调用

```json
{
  "level": "info",
  "time": "2025-10-01T17:00:00.000+0800",
  "msg": "gRPC call",
  "request_id": "7e87bd7a-ab18-426c-9dbc-7b5e39bfe694",
  "method": "/api.UserService/GetUser",
  "service": "api.UserService",
  "rpc": "GetUser",
  "latency": "12.5ms",
  "code": "OK"
}
```

#### 错误调用

```json
{
  "level": "error",
  "time": "2025-10-01T17:00:00.000+0800",
  "msg": "gRPC call",
  "request_id": "9b2fc479-c6a7-4433-8003-e04d03c1c78c",
  "method": "/api.UserService/GetUser",
  "service": "api.UserService",
  "rpc": "GetUser",
  "latency": "8.3ms",
  "code": "Internal",
  "error": "rpc error: code = Internal desc = database connection failed",
  "message": "database connection failed",
  "stacktrace": "..."
}
```

#### Panic 恢复

```json
{
  "level": "error",
  "time": "2025-10-01T17:00:00.000+0800",
  "msg": "gRPC panic recovered",
  "request_id": "req-123",
  "method": "/api.UserService/CreateUser",
  "panic": "runtime error: invalid memory address",
  "stacktrace": "..."
}
```

### RequestID 传递

gRPC 拦截器自动处理 RequestID:

1. **服务端**: 从 incoming metadata 提取，或自动生成
2. **客户端**: 从 context 提取，或自动生成
3. **传递**: 通过 `x-request-id` metadata 在服务间传递

```go
// 服务端自动提取
// metadata: x-request-id = "client-req-id"
// -> context: request_id = "client-req-id"

// 客户端发送
ctx := logger.WithRequestID(ctx, "my-req-id")
client.GetUser(ctx, req)
// -> metadata: x-request-id = "my-req-id"
```

## 测试

运行单元测试:

```bash
go test -v ./internal/pkg/logger/
```

## 依赖

- [uber-go/zap](https://github.com/uber-go/zap) - 高性能结构化日志
- [natefinch/lumberjack](https://github.com/natefinch/lumberjack) - 日志轮转
- [google/uuid](https://github.com/google/uuid) - UUID 生成（中间件）
- [google.golang.org/grpc](https://google.golang.org/grpc) - gRPC 框架（gRPC 拦截器）

## License

MIT
