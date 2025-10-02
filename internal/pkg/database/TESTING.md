# Database Package Testing Guide

本文档说明如何运行数据库包的测试。

## 测试类型

### 1. 单元测试

单元测试不需要数据库连接，测试配置、辅助函数等逻辑。

```bash
# 运行单元测试
go test -v ./internal/pkg/database/

# 或使用 make
make test-unit
```

**包含的测试**:
- 配置验证
- 分页函数
- 排序函数
- 条件查询
- QueryBuilder 结构
- 错误处理函数

### 2. 集成测试

集成测试需要真实的 PostgreSQL 数据库，使用 `docker-compose.yaml` 中定义的数据库。

```bash
# 使用 make 自动启动/停止 Docker
make test-integration

# 或手动运行
docker-compose up -d
go test -v -tags=integration ./internal/pkg/database/
docker-compose down
```

**包含的测试**:
- 数据库连接
- CRUD 操作
- 事务管理
- 查询助手
- 批量操作
- 软删除/恢复

## 环境配置

### Docker Compose 配置

测试使用 `docker-compose.yaml` 中的 PostgreSQL 配置：

```yaml
postgres:
  image: postgres:15-alpine
  environment:
    POSTGRES_USER: postgres
    POSTGRES_PASSWORD: postgres
    POSTGRES_DB: aiwriter
  ports:
    - "5432:5432"
```

### 环境变量（可选）

可以通过环境变量覆盖默认配置：

```bash
export TEST_DB_HOST=localhost
export TEST_DB_USER=postgres
export TEST_DB_PASSWORD=postgres
export TEST_DB_NAME=aiwriter

go test -v -tags=integration ./internal/pkg/database/
```

## 运行测试

### 快速开始

```bash
# 1. 确保 Docker 运行
docker ps

# 2. 运行集成测试（自动启动/停止数据库）
make test-integration
```

### 详细步骤

#### 步骤 1: 启动数据库

```bash
# 启动所有服务
docker-compose up -d

# 或只启动 PostgreSQL
docker-compose up -d postgres

# 等待数据库就绪
make docker-wait-postgres
```

#### 步骤 2: 运行测试

```bash
# 运行所有集成测试
go test -v -tags=integration ./internal/pkg/database/

# 运行特定测试
go test -v -tags=integration -run TestCRUDOperations ./internal/pkg/database/

# 带详细输出
go test -v -tags=integration ./internal/pkg/database/ 2>&1 | tee test-output.log
```

#### 步骤 3: 停止数据库

```bash
docker-compose down

# 或保留数据卷
docker-compose stop
```

## 测试覆盖率

```bash
# 生成覆盖率报告
go test -v -tags=integration -coverprofile=coverage.out ./internal/pkg/database/

# 查看覆盖率
go tool cover -func=coverage.out

# 生成 HTML 报告
go tool cover -html=coverage.out -o coverage.html
```

## 测试详情

### 数据库连接测试

测试数据库连接、健康检查和统计信息：

```bash
go test -v -tags=integration -run TestDatabaseConnection ./internal/pkg/database/
```

### CRUD 操作测试

测试基本的创建、读取、更新、删除操作：

```bash
go test -v -tags=integration -run TestCRUDOperations ./internal/pkg/database/
```

### 事务测试

测试事务提交、回滚、重试机制：

```bash
go test -v -tags=integration -run TestTransactions ./internal/pkg/database/
```

### 查询助手测试

测试分页、排序、条件查询等辅助函数：

```bash
go test -v -tags=integration -run TestQueryHelpers ./internal/pkg/database/
```

### 批量操作测试

测试批量插入、批量更新：

```bash
go test -v -tags=integration -run TestBatchOperations ./internal/pkg/database/
```

### 软删除测试

测试软删除、恢复、硬删除：

```bash
go test -v -tags=integration -run TestSoftDelete ./internal/pkg/database/
```

## CI/CD 集成

### GitHub Actions 示例

```yaml
name: Tests

on: [push, pull_request]

jobs:
  integration-tests:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: aiwriter
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Run integration tests
        run: go test -v -tags=integration ./internal/pkg/database/
```

## 故障排查

### 问题 1: 连接失败

```
Error: Failed to connect to database
```

**解决方案**:
1. 确认 Docker 容器运行中: `docker ps | grep postgres`
2. 检查端口占用: `lsof -i :5432`
3. 查看容器日志: `docker-compose logs postgres`
4. 等待数据库就绪: `make docker-wait-postgres`

### 问题 2: 表已存在

```
Error: relation "test_users" already exists
```

**解决方案**:
```bash
# 清理数据库
docker-compose down -v
docker-compose up -d
```

### 问题 3: 权限错误

```
Error: permission denied for database
```

**解决方案**:
检查数据库配置和用户权限：
```bash
docker exec -it aiwriter-postgres psql -U postgres -c "\du"
```

## 性能基准测试

```bash
# 运行基准测试
go test -v -tags=integration -bench=. -benchmem ./internal/pkg/database/

# 生成性能分析
go test -v -tags=integration -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./internal/pkg/database/

# 查看性能分析
go tool pprof cpu.prof
go tool pprof mem.prof
```

## 最佳实践

1. **始终在隔离环境中测试**
   - 使用专用测试数据库
   - 每次测试后清理数据

2. **测试数据独立**
   - 每个测试创建自己的测试数据
   - 使用 `t.Run()` 创建子测试

3. **使用 cleanup 函数**
   ```go
   db, cleanup := setupTestDB(t)
   defer cleanup()
   ```

4. **并发测试安全**
   - 避免测试间数据冲突
   - 使用唯一的测试数据

5. **模拟真实场景**
   - 测试边界条件
   - 测试错误处理
   - 测试并发情况

## 参考命令

```bash
# 完整测试流程
make test-integration

# 只启动数据库
make docker-up

# 只运行数据库集成测试
make test-db-integration

# 查看数据库日志
make docker-logs

# 停止所有服务
make docker-down

# 清理并重启
docker-compose down -v && docker-compose up -d
```
