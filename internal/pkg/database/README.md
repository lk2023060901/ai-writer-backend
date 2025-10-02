# Database Package

基于 GORM 和 PostgreSQL 封装的高性能数据库库，集成日志、事务管理和查询助手。

## 功能特性

- ✅ **连接池管理**: 可配置的连接池参数
- ✅ **事务管理**: 支持嵌套事务、隔离级别、自动重试
- ✅ **查询助手**: 分页、排序、条件查询等常用功能
- ✅ **日志集成**: 与自定义 logger 无缝集成
- ✅ **健康检查**: 数据库连接状态监控
- ✅ **错误处理**: 友好的错误判断函数

## 快速开始

### 1. 基础使用

```go
package main

import (
    "context"

    "github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
    "github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
)

func main() {
    // 初始化日志
    log, _ := logger.New(logger.DefaultConfig())

    // 创建数据库配置
    cfg := &database.Config{
        Host:     "localhost",
        Port:     5432,
        User:     "postgres",
        Password: "postgres",
        DBName:   "mydb",
        SSLMode:  "disable",

        MaxIdleConns:    10,
        MaxOpenConns:    100,
        ConnMaxLifetime: time.Hour,

        LogLevel:      "warn",
        SlowThreshold: 200 * time.Millisecond,
    }

    // 连接数据库
    db, err := database.New(cfg, log)
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // 使用数据库
    var user User
    db.First(&user, 1)
}
```

### 2. 配置文件使用

在 `config.yaml` 中配置:

```yaml
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  dbname: "mydb"
  sslmode: "disable"

  maxidleconns: 10
  maxopenconns: 100
  connmaxlifetime: "1h"
  connmaxidletime: "10m"

  loglevel: "warn"
  slowthreshold: "200ms"
  preparestmt: true

  timezone: "Asia/Shanghai"
  automigrate: true
```

加载配置:

```go
config, _ := conf.LoadConfig("config.yaml")
db, _ := database.New(&database.Config{
    Host:     config.Database.Host,
    Port:     config.Database.Port,
    User:     config.Database.User,
    Password: config.Database.Password,
    DBName:   config.Database.DBName,
    SSLMode:  config.Database.SSLMode,
    // ... 其他配置
}, log)
```

## 事务管理

### 基础事务

```go
// 自动提交/回滚
err := db.Transaction(ctx, func(ctx context.Context, tx *gorm.DB) error {
    // 在事务中执行操作
    if err := tx.Create(&user).Error; err != nil {
        return err // 自动回滚
    }

    if err := tx.Create(&order).Error; err != nil {
        return err // 自动回滚
    }

    return nil // 自动提交
})
```

### 事务管理器

```go
tm := database.NewTransactionManager(db)

// 基础执行（带3次重试）
err := tm.Execute(ctx, func(ctx context.Context, tx *gorm.DB) error {
    return tx.Create(&user).Error
})

// 自定义重试次数
err := tm.ExecuteWithRetry(ctx, 5, func(ctx context.Context, tx *gorm.DB) error {
    return tx.Create(&user).Error
})
```

### 隔离级别

```go
// READ COMMITTED
err := tm.ReadCommitted(ctx, func(ctx context.Context, tx *gorm.DB) error {
    return tx.Create(&user).Error
})

// REPEATABLE READ
err := tm.RepeatableRead(ctx, func(ctx context.Context, tx *gorm.DB) error {
    return tx.Find(&users).Error
})

// SERIALIZABLE
err := tm.Serializable(ctx, func(ctx context.Context, tx *gorm.DB) error {
    return tx.Create(&user).Error
})

// 只读事务
err := tm.ReadOnly(ctx, func(ctx context.Context, tx *gorm.DB) error {
    return tx.Find(&users).Error
})
```

### 嵌套事务（Savepoint）

```go
db.Transaction(ctx, func(ctx context.Context, tx *gorm.DB) error {
    // 外层事务
    tx.Create(&user)

    // 嵌套事务（使用 savepoint）
    err := tm.ExecuteNested(ctx, tx, func(ctx context.Context, tx *gorm.DB) error {
        return tx.Create(&order).Error
    })

    return nil
})
```

### 手动事务控制

```go
// 开始事务
tx := db.BeginTx(ctx, &sql.TxOptions{
    Isolation: sql.LevelReadCommitted,
})

// 执行操作
if err := tx.Create(&user).Error; err != nil {
    db.Rollback(tx)
    return err
}

// 提交事务
if err := db.Commit(tx); err != nil {
    return err
}
```

## 查询助手

### 分页查询

```go
// 方式1: 使用 Scope
db.Scopes(database.Paginate(page, pageSize)).Find(&users)

// 方式2: 使用辅助函数
result, err := database.FindWithPagination(ctx, db.DB, &users, page, pageSize)
// result.Data, result.Total, result.TotalPages
```

### 排序

```go
// 升序
db.Scopes(database.OrderBy("created_at", false)).Find(&users)

// 降序
db.Scopes(database.OrderBy("created_at", true)).Find(&users)
```

### 条件查询

```go
// 只在条件为真时添加 WHERE
status := "active"
db.Scopes(
    database.WhereIf(status != "", "status = ?", status),
    database.WhereIf(age > 0, "age > ?", age),
).Find(&users)
```

### 预加载

```go
db.Scopes(
    database.Preloads("Profile", "Orders"),
).Find(&users)
```

### 复杂查询构建器

```go
qb := database.NewQueryBuilder(db.DB).
    Where("status = ?", "active").
    Where("age > ?", 18).
    Order("created_at DESC").
    Limit(10).
    Offset(0).
    Preload("Profile").
    Select("id", "name", "email")

// 执行查询
var users []User
err := qb.Find(ctx, &users)

// 或获取数量
count, err := qb.Count(ctx)
```

### 批量操作

```go
// 批量插入
users := []User{{Name: "A"}, {Name: "B"}}
err := database.BatchInsert(ctx, db.DB, users, 100)

// 批量更新
updates := map[string]interface{}{
    "status": "inactive",
}
err := database.BulkUpdate(ctx, db.DB, &User{}, updates, "age < ?", 18)

// 批量处理
err := database.FindInBatches(ctx, db.DB, &users, 100, func(tx *gorm.DB, batch int) error {
    // 处理每批数据
    return nil
})
```

### 软删除和恢复

```go
// 软删除
err := database.SoftDelete(ctx, db.DB, &User{}, userID)

// 硬删除（永久）
err := database.HardDelete(ctx, db.DB, &User{}, userID)

// 恢复软删除的记录
err := database.Restore(ctx, db.DB, &User{}, userID)
```

### 其他工具函数

```go
// 检查记录是否存在
exists, err := database.Exists(ctx, db.DB, &User{}, "email = ?", email)

// FirstOrCreate
user := &User{Email: "test@example.com"}
err := database.FirstOrCreate(ctx, db.DB, user)

// 更新指定字段
updates := map[string]interface{}{"status": "active"}
err := database.UpdateFields(ctx, db.DB, &user, updates)

// 获取单列数据
var emails []string
err := database.Pluck(ctx, db.DB, "email", &emails)

// 计数
count, err := database.Count(ctx, db.DB, &User{}, "status = ?", "active")
```

## 健康检查

```go
// 健康检查
if err := db.HealthCheck(ctx); err != nil {
    log.Error("database health check failed", zap.Error(err))
}

// 获取连接池统计
stats := db.Stats()
// stats["open_connections"], stats["in_use"], stats["idle"], etc.
```

## 错误处理

```go
err := db.First(&user, id).Error

// 判断记录不存在
if database.IsRecordNotFoundError(err) {
    return errors.New("user not found")
}

// 判断唯一键冲突
if database.IsDuplicateKeyError(err) {
    return errors.New("email already exists")
}
```

## 上下文事务

```go
// 在 context 中传递事务
func CreateUser(ctx context.Context, db *database.DB) error {
    return db.Transaction(ctx, func(ctx context.Context, tx *gorm.DB) error {
        // 将事务添加到 context
        ctx = database.ContextWithTransaction(ctx, tx)

        // 在其他函数中使用
        return createUserProfile(ctx)
    })
}

func createUserProfile(ctx context.Context) error {
    // 从 context 获取事务
    tx, ok := database.TransactionFromContext(ctx)
    if !ok {
        return errors.New("no transaction in context")
    }

    return tx.Create(&profile).Error
}

// 或使用 GetDBFromContext
func createOrder(ctx context.Context, db *database.DB) error {
    // 如果 context 中有事务则使用，否则使用普通 DB
    txOrDB := db.GetDBFromContext(ctx)
    return txOrDB.Create(&order).Error
}
```

## 自动迁移

```go
// 启用自动迁移
cfg := database.DefaultConfig()
cfg.AutoMigrate = true

db, _ := database.New(cfg, log)

// 执行迁移
err := db.AutoMigrate(&User{}, &Order{}, &Product{})
```

## 日志集成

数据库操作会自动记录日志:

```json
{
  "level": "warn",
  "msg": "slow SQL query",
  "elapsed": "250ms",
  "threshold": "200ms",
  "rows": 100,
  "sql": "SELECT * FROM users WHERE status = 'active'"
}
```

日志级别:
- `silent`: 不记录
- `error`: 只记录错误
- `warn`: 记录错误和慢查询
- `info`: 记录所有查询

## 性能优化

### 1. 连接池配置

```go
cfg := &database.Config{
    MaxIdleConns:    10,   // 空闲连接数
    MaxOpenConns:    100,  // 最大连接数
    ConnMaxLifetime: time.Hour,      // 连接最大生命周期
    ConnMaxIdleTime: 10 * time.Minute, // 空闲连接超时
}
```

### 2. 预编译语句

```go
cfg.PrepareStmt = true  // 启用预编译语句缓存
```

### 3. 跳过默认事务

```go
cfg.SkipDefaultTx = true  // 跳过 GORM 默认事务（提升性能）
```

### 4. 批量操作

```go
// 批量插入，每批100条
database.BatchInsert(ctx, db.DB, records, 100)
```

## 最佳实践

1. **始终使用 context**
   ```go
   db.WithContext(ctx).Find(&users)
   ```

2. **使用事务管理器**
   ```go
   tm := database.NewTransactionManager(db)
   tm.Execute(ctx, txFunc)
   ```

3. **合理设置连接池**
   - 开发环境: MaxOpenConns = 10
   - 生产环境: MaxOpenConns = 100+

4. **监控慢查询**
   ```go
   cfg.SlowThreshold = 200 * time.Millisecond
   ```

5. **使用查询构建器**
   ```go
   qb := database.NewQueryBuilder(db.DB)
   ```

6. **优雅关闭**
   ```go
   defer db.Close()
   ```

## 测试

### 单元测试

运行单元测试（不需要数据库）:

```bash
# 使用 go test
go test -v ./internal/pkg/database/

# 使用 make
make test-unit
```

### 集成测试

运行集成测试（需要 PostgreSQL 数据库）:

```bash
# 使用 make（自动启动/停止 Docker）
make test-integration

# 手动运行
docker-compose up -d
go test -v -tags=integration ./internal/pkg/database/
docker-compose down
```

集成测试包括:
- ✅ 数据库连接测试
- ✅ CRUD 操作测试
- ✅ 事务管理测试（提交/回滚/重试）
- ✅ 查询助手测试（分页/排序/条件查询）
- ✅ 批量操作测试
- ✅ 软删除/恢复测试

详细测试说明请查看 [TESTING.md](TESTING.md)

## 依赖

- [gorm.io/gorm](https://gorm.io) - ORM 框架
- [gorm.io/driver/postgres](https://gorm.io/docs/connecting_to_the_database.html#PostgreSQL) - PostgreSQL 驱动
- [internal/pkg/logger](../logger) - 日志库

## License

MIT
