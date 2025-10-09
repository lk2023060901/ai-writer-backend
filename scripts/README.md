# 开发脚本说明

## 问题背景

当使用 `make run` 启动服务器时，按 **Ctrl+C** 进程不会退出。这是因为：

1. `make run` → `go run` → Go 程序（多层进程）
2. Ctrl+C 发送 SIGINT 给 `make` 进程
3. `make` 退出，但 `go run` 启动的子进程不会收到信号
4. Go 程序继续运行在后台

## 解决方案

### 方案1：使用 `make dev`（推荐）

```bash
make dev
```

**特点**：
- ✅ 优雅退出：Ctrl+C 会发送 SIGTERM 给 Go 程序
- ✅ 清理完整：等待最多10秒让程序清理资源
- ✅ 进程追踪：保存 PID 到 `/tmp/ai-writer-dev.pid`
- ✅ 日志记录：输出保存到 `/tmp/ai-writer-dev.log`

**工作原理**：
```bash
./scripts/dev.sh config.yaml
```

脚本会：
1. 启动 `go run cmd/server/main.go`
2. 保存进程 PID
3. 注册 `trap` 捕获 INT/TERM 信号
4. 收到信号时发送 SIGTERM 给子进程
5. 等待优雅退出，超时则强制杀死

### 方案2：使用 `make run`（简单版本）

```bash
make run
```

**特点**：
- ✅ 快速启动
- ✅ 使用 `trap` 和 `pkill -P` 杀死子进程
- ⚠️ 强制杀死，不等待优雅退出

**工作原理**：
```makefile
@trap 'echo "Shutting down..."; pkill -P $$; exit' INT TERM; \
go run cmd/server/main.go -config=config.yaml
```

收到 Ctrl+C 时会杀死所有子进程。

## 直接使用脚本

```bash
# 使用默认配置
./scripts/dev.sh

# 指定配置文件
./scripts/dev.sh path/to/config.yaml

# 查看日志
tail -f /tmp/ai-writer-dev.log

# 查看进程 PID
cat /tmp/ai-writer-dev.pid
```

## 生产环境

生产环境应使用编译后的二进制文件：

```bash
# 编译
make build

# 运行（会正确处理信号）
./bin/server -config=config.yaml
```

或使用 systemd/supervisor 等进程管理工具。

## 常见问题

### Q: 为什么不修改 Go 代码？

A: Go 代码已经有信号处理（见 `cmd/server/main.go:105`），问题在于 `go run` 的信号传递机制。

### Q: 按 Ctrl+C 还是没退出怎么办？

A: 手动清理：
```bash
# 查找进程
ps aux | grep ai-writer

# 杀死进程
pkill -9 ai-writer

# 或指定 PID
kill -9 <PID>
```

### Q: 如何检查端口占用？

```bash
# 检查 8080 和 9090 端口
lsof -i :8080 -i :9090

# 杀死占用端口的进程
lsof -ti:8080,9090 | xargs kill -9
```
