# 用户表慢查询检测指南

## 1. PostgreSQL 慢查询日志配置

### 启用慢查询日志

编辑 `postgresql.conf` 或在 Docker 中通过环境变量配置：

```sql
-- 记录执行时间超过 100ms 的查询
ALTER SYSTEM SET log_min_duration_statement = 100;

-- 记录所有 DDL 语句
ALTER SYSTEM SET log_statement = 'ddl';

-- 重载配置
SELECT pg_reload_conf();
```

### Docker Compose 配置

```yaml
# docker-compose.yaml
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: postgres
    command:
      - "postgres"
      - "-c"
      - "log_min_duration_statement=100"
      - "-c"
      - "log_statement=ddl"
      - "-c"
      - "log_line_prefix=%m [%p] %u@%d "
    volumes:
      - ./logs/postgres:/var/log/postgresql
```

## 2. 慢查询检测 SQL 脚本

### 2.1 启用统计扩展

```sql
-- 启用 pg_stat_statements 扩展（需要超级用户权限）
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- 重置统计数据
SELECT pg_stat_statements_reset();
```

### 2.2 查询最慢的 SQL

```sql
-- 查看 users 表相关的最慢查询（按总执行时间排序）
SELECT
    query,
    calls,
    total_exec_time / 1000 AS total_time_sec,
    mean_exec_time / 1000 AS mean_time_sec,
    max_exec_time / 1000 AS max_time_sec,
    rows / calls AS avg_rows
FROM pg_stat_statements
WHERE query LIKE '%users%'
  AND query NOT LIKE '%pg_stat_statements%'
ORDER BY total_exec_time DESC
LIMIT 10;
```

### 2.3 查询平均执行时间最长的 SQL

```sql
-- 按平均执行时间排序
SELECT
    query,
    calls,
    mean_exec_time / 1000 AS mean_time_sec,
    stddev_exec_time / 1000 AS stddev_sec,
    (total_exec_time / 1000) / 60 AS total_time_min
FROM pg_stat_statements
WHERE query LIKE '%users%'
  AND calls > 10  -- 至少被调用 10 次
ORDER BY mean_exec_time DESC
LIMIT 10;
```

### 2.4 检查缺失的索引

```sql
-- 查找顺序扫描次数多的表
SELECT
    schemaname,
    tablename,
    seq_scan,
    seq_tup_read,
    idx_scan,
    seq_tup_read / seq_scan AS avg_seq_tup_read
FROM pg_stat_user_tables
WHERE schemaname = 'public'
  AND tablename = 'users'
  AND seq_scan > 0
ORDER BY seq_tup_read DESC;
```

### 2.5 查看索引使用情况

```sql
-- 查看 users 表的索引命中率
SELECT
    schemaname,
    tablename,
    indexrelname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
  AND tablename = 'users'
ORDER BY idx_scan DESC;
```

### 2.6 查找未使用的索引

```sql
-- 查找从未被使用的索引
SELECT
    schemaname,
    tablename,
    indexrelname,
    idx_scan,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
  AND tablename = 'users'
  AND idx_scan = 0
ORDER BY pg_relation_size(indexrelid) DESC;
```

## 3. EXPLAIN ANALYZE 性能测试

### 3.1 测试邮箱查询（登录场景）

```sql
-- 测试唯一索引性能
EXPLAIN ANALYZE
SELECT * FROM users
WHERE email = 'test@example.com'
  AND deleted_at IS NULL;

-- 期望结果：
-- Index Scan using idx_users_email on users (cost=0.29..8.31 rows=1 width=...)
-- Planning Time: 0.1ms
-- Execution Time: 0.05ms
```

### 3.2 测试 Token 查询

```sql
-- 测试邮箱验证 Token
EXPLAIN ANALYZE
SELECT * FROM users
WHERE email_verification_token = 'abc123'
  AND email_verification_expires_at > NOW();

-- 测试密码重置 Token
EXPLAIN ANALYZE
SELECT * FROM users
WHERE password_reset_token = 'xyz789'
  AND password_reset_expires_at > NOW();
```

### 3.3 测试软删除查询

```sql
-- 测试活跃用户列表
EXPLAIN ANALYZE
SELECT id, name, email, created_at
FROM users
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 20;
```

### 3.4 测试账户锁定查询

```sql
-- 查找当前被锁定的账户
EXPLAIN ANALYZE
SELECT id, email, locked_until
FROM users
WHERE locked_until > NOW()
  AND deleted_at IS NULL;
```

## 4. 性能基准测试

### 4.1 创建测试数据

```sql
-- 插入 10,000 条测试数据
INSERT INTO users (name, email, password_hash, created_at)
SELECT
    'User ' || i,
    'user' || i || '@example.com',
    '$2a$12$abcdefghijklmnopqrstuvwxyz1234567890',  -- 假哈希
    NOW() - (i || ' seconds')::INTERVAL
FROM generate_series(1, 10000) AS i;
```

### 4.2 基准测试查询

```sql
-- 测试 1: 邮箱精确查询（应该 < 1ms）
\timing
SELECT * FROM users WHERE email = 'user5000@example.com' AND deleted_at IS NULL;

-- 测试 2: 分页查询（应该 < 10ms）
SELECT id, name, email FROM users
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 20 OFFSET 0;

-- 测试 3: Token 查询（应该 < 5ms）
SELECT * FROM users
WHERE email_verification_token = 'test_token'
  AND deleted_at IS NULL;

-- 测试 4: 统计查询（应该 < 50ms）
SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;
SELECT COUNT(*) FROM users WHERE email_verified = true;
```

## 5. 性能优化建议

### 5.1 查询优化

**慢查询诊断流程**：
1. 使用 `EXPLAIN ANALYZE` 查看执行计划
2. 检查是否使用了正确的索引
3. 检查是否有顺序扫描（Seq Scan）
4. 检查返回的行数是否合理

**常见问题**：
- ❌ `Seq Scan on users` → 缺少索引或索引未被使用
- ❌ `Planning Time > 10ms` → 统计信息过时，运行 `ANALYZE users`
- ❌ `Execution Time > 100ms` → 数据量大或查询条件复杂

### 5.2 索引优化

```sql
-- 检查索引膨胀
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
    pg_size_pretty(pg_indexes_size(schemaname||'.'||tablename)) AS index_size
FROM pg_tables
WHERE schemaname = 'public' AND tablename = 'users';

-- 重建索引（如果膨胀严重）
REINDEX TABLE users;
```

### 5.3 更新统计信息

```sql
-- 手动更新表统计信息
ANALYZE users;

-- 查看上次 ANALYZE 时间
SELECT
    schemaname,
    tablename,
    last_analyze,
    last_autoanalyze
FROM pg_stat_user_tables
WHERE tablename = 'users';
```

## 6. 自动化监控脚本

### 6.1 创建监控视图

```sql
-- 创建慢查询监控视图
CREATE OR REPLACE VIEW v_slow_queries AS
SELECT
    query,
    calls,
    total_exec_time / 1000 AS total_time_sec,
    mean_exec_time / 1000 AS mean_time_sec,
    max_exec_time / 1000 AS max_time_sec
FROM pg_stat_statements
WHERE mean_exec_time > 100  -- 平均执行时间 > 100ms
ORDER BY mean_exec_time DESC;

-- 查询慢查询
SELECT * FROM v_slow_queries LIMIT 10;
```

### 6.2 设置告警阈值

```sql
-- 查找执行时间超过 1 秒的查询
SELECT
    query,
    calls,
    max_exec_time / 1000 AS max_time_sec
FROM pg_stat_statements
WHERE max_exec_time > 1000  -- 1 秒
  AND query LIKE '%users%'
ORDER BY max_exec_time DESC;
```

## 7. Go 代码层面的监控

### 7.1 添加查询耗时日志

```go
// internal/user/data/user.go
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        if duration > 100*time.Millisecond {
            log.Warnf("Slow query: GetByEmail took %v for email=%s", duration, email)
        }
    }()

    var user UserPO
    err := r.db.WithContext(ctx).
        Where("email = ? AND deleted_at IS NULL", email).
        First(&user).Error

    return &user, err
}
```

### 7.2 使用 OpenTelemetry 追踪

```go
import "go.opentelemetry.io/otel"

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
    ctx, span := otel.Tracer("user-repo").Start(ctx, "GetByEmail")
    defer span.End()

    span.SetAttributes(
        attribute.String("email", email),
    )

    // ... 执行查询
}
```

## 8. 性能测试检查清单

- [ ] 所有查询使用了正确的索引（无 Seq Scan）
- [ ] 邮箱查询 < 5ms
- [ ] Token 查询 < 10ms
- [ ] 分页查询 < 20ms
- [ ] 统计查询 < 100ms
- [ ] 无未使用的索引
- [ ] 索引大小 < 表大小的 50%
- [ ] 缓存命中率 > 95%
- [ ] 慢查询日志中无 users 表相关查询

## 9. 应急处理

### 发现慢查询后的处理步骤

1. **立即排查**：
   ```sql
   SELECT * FROM v_slow_queries WHERE query LIKE '%users%';
   ```

2. **查看执行计划**：
   ```sql
   EXPLAIN ANALYZE <慢查询>;
   ```

3. **检查统计信息**：
   ```sql
   ANALYZE users;
   ```

4. **考虑添加索引**（谨慎！）：
   ```sql
   -- 示例：如果发现大量按 last_login_at 排序的查询
   CREATE INDEX CONCURRENTLY idx_users_last_login_at
   ON users (last_login_at DESC)
   WHERE deleted_at IS NULL;
   ```

5. **监控新索引使用情况**：
   ```sql
   SELECT * FROM pg_stat_user_indexes
   WHERE indexrelname = 'idx_users_last_login_at';
   ```
