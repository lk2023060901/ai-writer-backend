# 数据库性能测试快速指南

## 快速开始

### 1. 启用性能监控

```bash
# 启用 pg_stat_statements 扩展（首次使用）
make db-enable-stats

# 或手动执行
psql -U postgres -d ai_writer -c "CREATE EXTENSION IF NOT EXISTS pg_stat_statements;"
```

### 2. 运行性能基准测试

```bash
# 完整的性能基准测试
make db-benchmark

# 自定义数据库连接
DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASS=yourpass DB_NAME=ai_writer make db-benchmark
```

**测试内容**：
- ✓ 邮箱精确查询（登录）
- ✓ 分页查询（用户列表）
- ✓ Token 查询（邮箱验证/密码重置）
- ✓ 账户锁定查询
- ✓ 统计查询
- ✓ 软删除查询

**期望结果**：
```
Execution Time: 0.123 ms  ✓ 优秀
Execution Time: 15.456 ms ✓ 良好
Execution Time: 78.901 ms ⚠️ 可接受
Execution Time: 234.567 ms ❌ 需要优化
```

### 3. 检测慢查询

```bash
# 检查 users 表的慢查询
make db-check-slow-queries
```

**检测项目**：
1. 最慢的 SQL 查询（按总执行时间）
2. 平均执行时间最慢的查询
3. 索引使用情况
4. 未使用的索引
5. 顺序扫描统计
6. 表和索引大小
7. 统计信息更新时间
8. 缓存命中率
9. 活跃连接数

**关键指标**：
- 缓存命中率应 > 95%
- 索引命中率应 > 90%
- 无未使用的索引
- 顺序扫描次数应该很低

### 4. 更新统计信息

```bash
# 更新 users 表统计信息（查询变慢时）
make db-analyze-users
```

### 5. 重置统计数据

```bash
# 清空 pg_stat_statements 统计（重新开始监控）
make db-reset-stats
```

## 完整工作流程

### 场景 1: 日常性能检查

```bash
# 1. 运行基准测试
make db-benchmark

# 2. 查看慢查询
make db-check-slow-queries

# 3. 如果发现慢查询，更新统计信息
make db-analyze-users
```

### 场景 2: 发现性能问题

```bash
# 1. 检查慢查询详情
make db-check-slow-queries

# 2. 查看具体查询的执行计划
psql -U postgres -d ai_writer
# 在 psql 中执行：
EXPLAIN ANALYZE SELECT * FROM users WHERE email = 'test@example.com' AND deleted_at IS NULL;

# 3. 检查索引使用情况
# 输出会显示是否使用了 Index Scan（好）还是 Seq Scan（不好）

# 4. 考虑优化方案
# - 添加索引
# - 调整查询
# - 更新统计信息
```

### 场景 3: 生产环境监控

```bash
# 定时任务（每天凌晨 2 点）
0 2 * * * cd /path/to/project && make db-check-slow-queries >> /var/log/db_perf.log 2>&1

# 告警阈值
# - 缓存命中率 < 90%
# - 平均查询时间 > 100ms
# - 顺序扫描次数 > 1000/天
```

## Go 代码测试命令

### 测试指定包

```bash
# 测试 auth 包
go test -v ./internal/auth/

# 测试 user 包
go test -v ./internal/user/

# 测试所有包
go test -v ./...
```

### 测试特定函数

```bash
# 测试单个函数
go test -v -run TestGenerateBackupCode ./internal/auth/

# 测试多个函数（正则匹配）
go test -v -run "TestGenerate|TestVerify" ./internal/auth/

# 测试子测试
go test -v -run "TestVerifyBackupCode/正确的恢复码" ./internal/auth/
```

### 基准测试

```bash
# 运行所有基准测试
go test -bench=. -benchmem ./internal/auth/

# 运行特定基准测试
go test -bench=BenchmarkGenerateBackupCode -benchmem ./internal/auth/

# 只运行基准测试（跳过单元测试）
go test -run=^$ -bench=. -benchmem ./internal/auth/

# 指定运行时间
go test -bench=. -benchtime=5s -benchmem ./internal/auth/
```

### 测试覆盖率

```bash
# 生成覆盖率报告
go test -v -coverprofile=coverage.out ./internal/auth/
go tool cover -html=coverage.out -o coverage.html

# 查看覆盖率百分比
go test -cover ./internal/auth/
```

## 性能优化检查清单

### 索引优化
- [ ] 所有 WHERE 条件字段都有索引
- [ ] 高频查询字段有索引（email, tokens）
- [ ] 部分索引用于软删除查询（WHERE deleted_at IS NULL）
- [ ] 无未使用的索引
- [ ] 索引大小 < 表大小的 50%

### 查询优化
- [ ] 所有查询使用 Index Scan（非 Seq Scan）
- [ ] LIMIT 用于分页查询
- [ ] SELECT 仅选择需要的字段（避免 SELECT *）
- [ ] JOIN 条件使用索引字段
- [ ] 避免在 WHERE 中使用函数（如 LOWER(email)）

### 统计信息
- [ ] last_analyze 在最近 24 小时内
- [ ] last_autovacuum 在最近一周内
- [ ] 缓存命中率 > 95%
- [ ] 连接池配置合理

### 应用层优化
- [ ] 使用连接池（pgx/pgxpool）
- [ ] 查询超时设置（context.WithTimeout）
- [ ] 慢查询日志记录
- [ ] 分页查询使用游标（cursor-based pagination）

## 常见问题

### Q: 为什么我的查询很慢？

**A**: 检查步骤：
1. 运行 `EXPLAIN ANALYZE` 查看执行计划
2. 检查是否使用了索引（Index Scan vs Seq Scan）
3. 检查统计信息是否过期（运行 `ANALYZE users`）
4. 检查数据量是否过大（考虑分区表）

### Q: 缓存命中率低怎么办？

**A**:
1. 检查 `shared_buffers` 配置（推荐设置为物理内存的 25%）
2. 检查是否有大量冷数据访问
3. 考虑使用应用层缓存（Redis）

### Q: 索引未被使用？

**A**:
1. 检查查询条件是否使用了函数（如 `WHERE LOWER(email)`）
2. 运行 `ANALYZE` 更新统计信息
3. 检查索引选择性（`SELECT COUNT(DISTINCT email) / COUNT(*) FROM users`）
4. 考虑添加复合索引

### Q: 需要添加新索引吗？

**A**: 谨慎添加！
- ✓ 添加前先运行 `EXPLAIN` 确认会被使用
- ✓ 使用 `CREATE INDEX CONCURRENTLY` 避免锁表
- ✓ 监控新索引的使用情况
- ✓ 定期检查并删除未使用的索引

## 参考资料

- [PostgreSQL EXPLAIN 文档](https://www.postgresql.org/docs/current/sql-explain.html)
- [pg_stat_statements 文档](https://www.postgresql.org/docs/current/pgstatstatements.html)
- [Go 测试文档](https://golang.org/pkg/testing/)
- [慢查询检测文档](./slow_query_detection.md)
