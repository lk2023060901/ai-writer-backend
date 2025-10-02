# GORM PostgreSQL 封装设计文档

## 1. 核心接口定义

### 1.1 数据库客户端接口

```go
// pkg/database/postgres/client.go
package postgres

import (
    "context"
    "gorm.io/gorm"
)

// Client 数据库客户端接口
type Client interface {
    // DB 获取 GORM 实例
    DB() *gorm.DB

    // Transaction 执行事务
    Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error

    // WithContext 带上下文的数据库实例
    WithContext(ctx context.Context) *gorm.DB

    // Close 关闭连接
    Close() error

    // Health 健康检查
    Health(ctx context.Context) error
}

// Config 数据库配置
type Config struct {
    Host            string
    Port            int
    User            string
    Password        string
    DBName          string
    SSLMode         string
    MaxIdleConns    int
    MaxOpenConns    int
    ConnMaxLifetime time.Duration
    LogLevel        string
}
```

### 1.2 仓储基础接口

```go
// internal/data/repository/base.go
package repository

import "context"

// BaseRepository 基础仓储接口
type BaseRepository[T any] interface {
    // Create 创建
    Create(ctx context.Context, entity *T) error

    // GetByID 根据ID获取
    GetByID(ctx context.Context, id int64) (*T, error)

    // Update 更新
    Update(ctx context.Context, entity *T) error

    // Delete 删除（软删除）
    Delete(ctx context.Context, id int64) error

    // List 列表查询
    List(ctx context.Context, query Query) ([]*T, int64, error)
}

// Query 查询条件
type Query struct {
    Filters  map[string]interface{} // 过滤条件
    OrderBy  []string               // 排序
    Page     int                    // 页码
    PageSize int                    // 每页数量
}
```

### 1.3 事务管理器接口

```go
// pkg/database/postgres/transaction.go
package postgres

import (
    "context"
    "gorm.io/gorm"
)

// TransactionManager 事务管理器
type TransactionManager interface {
    // Execute 执行事务
    Execute(ctx context.Context, fn func(tx *gorm.DB) error) error

    // ExecuteNested 执行嵌套事务
    ExecuteNested(ctx context.Context, fn func(tx *gorm.DB) error) error

    // SavePoint 创建保存点
    SavePoint(ctx context.Context, name string) error

    // RollbackTo 回滚到保存点
    RollbackTo(ctx context.Context, name string) error
}
```

## 2. 查询构建器

### 2.1 链式查询

```go
// pkg/database/query/builder.go
package query

import "gorm.io/gorm"

// Builder 查询构建器
type Builder struct {
    db *gorm.DB
}

func NewBuilder(db *gorm.DB) *Builder {
    return &Builder{db: db}
}

// Where 添加条件
func (b *Builder) Where(condition string, args ...interface{}) *Builder {
    b.db = b.db.Where(condition, args...)
    return b
}

// WhereIn IN 查询
func (b *Builder) WhereIn(field string, values []interface{}) *Builder {
    b.db = b.db.Where(field+" IN ?", values)
    return b
}

// OrderBy 排序
func (b *Builder) OrderBy(field string, desc bool) *Builder {
    order := field
    if desc {
        order += " DESC"
    }
    b.db = b.db.Order(order)
    return b
}

// Paginate 分页
func (b *Builder) Paginate(page, pageSize int) *Builder {
    offset := (page - 1) * pageSize
    b.db = b.db.Offset(offset).Limit(pageSize)
    return b
}

// Find 执行查询
func (b *Builder) Find(dest interface{}) error {
    return b.db.Find(dest).Error
}

// Count 统计数量
func (b *Builder) Count() (int64, error) {
    var count int64
    err := b.db.Count(&count).Error
    return count, err
}
```

### 2.2 动态条件构建

```go
// pkg/database/query/condition.go
package query

import "gorm.io/gorm"

// Condition 动态条件
type Condition struct {
    Field    string
    Operator string // =, !=, >, <, >=, <=, LIKE, IN
    Value    interface{}
}

// ApplyConditions 应用条件
func ApplyConditions(db *gorm.DB, conditions []Condition) *gorm.DB {
    for _, cond := range conditions {
        switch cond.Operator {
        case "=":
            db = db.Where(cond.Field+" = ?", cond.Value)
        case "!=":
            db = db.Where(cond.Field+" != ?", cond.Value)
        case ">":
            db = db.Where(cond.Field+" > ?", cond.Value)
        case "LIKE":
            db = db.Where(cond.Field+" LIKE ?", "%"+cond.Value.(string)+"%")
        case "IN":
            db = db.Where(cond.Field+" IN ?", cond.Value)
        }
    }
    return db
}
```

## 3. 模型基类

### 3.1 基础模型

```go
// pkg/database/model/base.go
package model

import (
    "time"
    "gorm.io/gorm"
)

// BaseModel 基础模型
type BaseModel struct {
    ID        int64          `gorm:"primarykey" json:"id"`
    CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// SoftDeleteModel 软删除模型
type SoftDeleteModel struct {
    BaseModel
}

// AuditableModel 可审计模型
type AuditableModel struct {
    BaseModel
    CreatedBy int64 `json:"created_by"`
    UpdatedBy int64 `json:"updated_by"`
}
```

## 4. 缓存装饰器

### 4.1 缓存接口

```go
// pkg/database/cache/cache.go
package cache

import (
    "context"
    "time"
)

// Cache 缓存接口
type Cache interface {
    Get(ctx context.Context, key string, dest interface{}) error
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    DeletePattern(ctx context.Context, pattern string) error
}

// CachedRepository 带缓存的仓储装饰器
type CachedRepository[T any] struct {
    repo  repository.BaseRepository[T]
    cache Cache
    ttl   time.Duration
}

func (r *CachedRepository[T]) GetByID(ctx context.Context, id int64) (*T, error) {
    // 1. 尝试从缓存获取
    key := fmt.Sprintf("entity:%T:%d", new(T), id)
    var entity T
    if err := r.cache.Get(ctx, key, &entity); err == nil {
        return &entity, nil
    }

    // 2. 从数据库获取
    entity, err := r.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // 3. 写入缓存
    _ = r.cache.Set(ctx, key, entity, r.ttl)

    return entity, nil
}
```

## 5. 中间件系统

### 5.1 慢查询日志

```go
// pkg/database/middleware/logger.go
package middleware

import (
    "context"
    "time"
    "go.uber.org/zap"
    "gorm.io/gorm"
)

// SlowQueryLogger 慢查询日志中间件
func SlowQueryLogger(threshold time.Duration, logger *zap.Logger) gorm.Plugin {
    return &slowQueryPlugin{
        threshold: threshold,
        logger:    logger,
    }
}

type slowQueryPlugin struct {
    threshold time.Duration
    logger    *zap.Logger
}

func (p *slowQueryPlugin) Name() string {
    return "slowQueryLogger"
}

func (p *slowQueryPlugin) Initialize(db *gorm.DB) error {
    db.Callback().Query().Before("gorm:query").Register("slow_query:before", func(db *gorm.DB) {
        db.Set("query_start_time", time.Now())
    })

    db.Callback().Query().After("gorm:query").Register("slow_query:after", func(db *gorm.DB) {
        if v, ok := db.Get("query_start_time"); ok {
            startTime := v.(time.Time)
            duration := time.Since(startTime)

            if duration > p.threshold {
                p.logger.Warn("slow query detected",
                    zap.Duration("duration", duration),
                    zap.String("sql", db.Statement.SQL.String()),
                )
            }
        }
    })

    return nil
}
```

## 6. 使用示例

### 6.1 初始化数据库

```go
// internal/data/data.go
package data

import (
    "github.com/lk2023060901/ai-writer-backend/internal/conf"
    "github.com/lk2023060901/ai-writer-backend/pkg/database/postgres"
    "github.com/lk2023060901/ai-writer-backend/pkg/database/middleware"
)

func NewData(config *conf.Config, logger *zap.Logger) (*Data, func(), error) {
    // 创建 PostgreSQL 客户端
    pgConfig := &postgres.Config{
        Host:            config.Database.Host,
        Port:            config.Database.Port,
        User:            config.Database.User,
        Password:        config.Database.Password,
        DBName:          config.Database.DBName,
        MaxOpenConns:    100,
        MaxIdleConns:    10,
        ConnMaxLifetime: time.Hour,
    }

    client, err := postgres.NewClient(pgConfig)
    if err != nil {
        return nil, nil, err
    }

    // 注册中间件
    db := client.DB()
    db.Use(middleware.SlowQueryLogger(500*time.Millisecond, logger))

    return &Data{
        DB: db,
        Client: client,
    }, func() {
        client.Close()
    }, nil
}
```

### 6.2 实现仓储

```go
// internal/user/data/user.go
package data

import (
    "context"
    "github.com/lk2023060901/ai-writer-backend/internal/user/biz"
    "github.com/lk2023060901/ai-writer-backend/pkg/database/query"
    "gorm.io/gorm"
)

type userRepo struct {
    db *gorm.DB
}

func NewUserRepo(db *gorm.DB) biz.UserRepo {
    return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *biz.User) error {
    po := toUserPO(user)
    return r.db.WithContext(ctx).Create(po).Error
}

func (r *userRepo) List(ctx context.Context, filters map[string]interface{}, page, pageSize int) ([]*biz.User, int64, error) {
    builder := query.NewBuilder(r.db.WithContext(ctx).Model(&UserPO{}))

    // 动态条件
    if name, ok := filters["name"]; ok {
        builder.Where("name LIKE ?", "%"+name.(string)+"%")
    }

    // 统计总数
    total, err := builder.Count()
    if err != nil {
        return nil, 0, err
    }

    // 分页查询
    var pos []UserPO
    if err := builder.Paginate(page, pageSize).Find(&pos); err != nil {
        return nil, 0, err
    }

    users := make([]*biz.User, len(pos))
    for i, po := range pos {
        users[i] = toUser(&po)
    }

    return users, total, nil
}
```

## 7. 依赖管理

### 7.1 go.mod 依赖版本

```go
module github.com/lk2023060901/ai-writer-backend

go 1.21

require (
    gorm.io/gorm v1.25.5
    gorm.io/driver/postgres v1.5.4
    github.com/redis/go-redis/v9 v9.3.0
    go.uber.org/zap v1.26.0
)
```

### 7.2 避免循环依赖

```
规则：
1. pkg/database 不依赖 internal 任何包
2. internal/data 依赖 pkg/database
3. internal/*/biz 只依赖接口，不依赖实现
4. internal/*/data 实现接口，依赖 pkg/database
```

## 8. 迁移策略

### 8.1 从现有代码迁移

```go
// Step 1: 保持现有 internal/data/data.go
// Step 2: 创建 pkg/database/postgres/client.go
// Step 3: 逐步将 data.go 的功能迁移到 client.go
// Step 4: 更新 internal/data/data.go 使用新的 client
// Step 5: 删除旧的实现
```

## 9. 性能优化建议

1. **连接池配置**：MaxOpenConns=100, MaxIdleConns=10
2. **预编译语句**：对高频查询使用 `Prepared Statement`
3. **批量操作**：使用 `CreateInBatches` 和 `Updates`
4. **索引优化**：为常用查询字段添加索引
5. **读写分离**：使用 GORM DBResolver
6. **缓存策略**：对读多写少的数据使用 Redis 缓存

## 10. 监控指标

```go
// 关键指标
- 慢查询数量 (>500ms)
- 连接池使用率
- 事务成功率
- 平均查询耗时
- 缓存命中率
```
