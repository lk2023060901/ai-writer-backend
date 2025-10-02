-- ============================================
-- 用户表慢查询检测脚本
-- 使用方法：psql -U postgres -d ai_writer -f scripts/check_slow_queries.sql
-- ============================================

\echo '=========================================='
\echo '1. 检查 pg_stat_statements 扩展'
\echo '=========================================='
SELECT EXISTS(
    SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements'
) AS pg_stat_statements_installed;

\echo ''
\echo '如果返回 false，请先安装扩展：'
\echo 'CREATE EXTENSION IF NOT EXISTS pg_stat_statements;'
\echo ''

\echo '=========================================='
\echo '2. users 表最慢的查询（按总执行时间）'
\echo '=========================================='
SELECT
    LEFT(query, 80) AS query_snippet,
    calls,
    ROUND(total_exec_time::numeric / 1000, 2) AS total_time_sec,
    ROUND(mean_exec_time::numeric / 1000, 2) AS mean_time_ms,
    ROUND(max_exec_time::numeric / 1000, 2) AS max_time_ms
FROM pg_stat_statements
WHERE query LIKE '%users%'
  AND query NOT LIKE '%pg_stat_statements%'
ORDER BY total_exec_time DESC
LIMIT 10;

\echo ''
\echo '=========================================='
\echo '3. users 表平均执行时间最慢的查询'
\echo '=========================================='
SELECT
    LEFT(query, 80) AS query_snippet,
    calls,
    ROUND(mean_exec_time::numeric / 1000, 2) AS mean_time_ms,
    ROUND(max_exec_time::numeric / 1000, 2) AS max_time_ms
FROM pg_stat_statements
WHERE query LIKE '%users%'
  AND calls > 5
ORDER BY mean_exec_time DESC
LIMIT 10;

\echo ''
\echo '=========================================='
\echo '4. users 表索引使用情况'
\echo '=========================================='
SELECT
    indexrelname AS index_name,
    idx_scan AS scans,
    idx_tup_read AS tuples_read,
    idx_tup_fetch AS tuples_fetched,
    pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
  AND tablename = 'users'
ORDER BY idx_scan DESC;

\echo ''
\echo '=========================================='
\echo '5. users 表未使用的索引（警告！）'
\echo '=========================================='
SELECT
    indexrelname AS unused_index,
    pg_size_pretty(pg_relation_size(indexrelid)) AS wasted_size
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
  AND tablename = 'users'
  AND idx_scan = 0
ORDER BY pg_relation_size(indexrelid) DESC;

\echo ''
\echo '=========================================='
\echo '6. users 表顺序扫描统计'
\echo '=========================================='
SELECT
    schemaname,
    tablename,
    seq_scan AS sequential_scans,
    seq_tup_read AS seq_tuples_read,
    idx_scan AS index_scans,
    idx_tup_fetch AS idx_tuples_fetched,
    CASE
        WHEN seq_scan > 0
        THEN ROUND((seq_tup_read::numeric / seq_scan), 2)
        ELSE 0
    END AS avg_seq_tuples
FROM pg_stat_user_tables
WHERE schemaname = 'public'
  AND tablename = 'users';

\echo ''
\echo '⚠️  如果 sequential_scans 很高且 avg_seq_tuples 很大，说明可能缺少索引！'
\echo ''

\echo '=========================================='
\echo '7. users 表大小和索引大小'
\echo '=========================================='
SELECT
    pg_size_pretty(pg_total_relation_size('users')) AS total_size,
    pg_size_pretty(pg_relation_size('users')) AS table_size,
    pg_size_pretty(pg_indexes_size('users')) AS indexes_size,
    ROUND(
        (pg_indexes_size('users')::numeric / NULLIF(pg_relation_size('users'), 0) * 100),
        2
    ) AS index_ratio_percent
FROM pg_class
WHERE relname = 'users'
LIMIT 1;

\echo ''
\echo '=========================================='
\echo '8. users 表统计信息更新时间'
\echo '=========================================='
SELECT
    schemaname,
    tablename,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    last_autoanalyze,
    n_tup_ins AS inserts,
    n_tup_upd AS updates,
    n_tup_del AS deletes
FROM pg_stat_user_tables
WHERE tablename = 'users';

\echo ''
\echo '如果 last_analyze 时间很久远，建议运行：ANALYZE users;'
\echo ''

\echo '=========================================='
\echo '9. 缓存命中率（全局）'
\echo '=========================================='
SELECT
    ROUND(
        100.0 * sum(blks_hit) / NULLIF(sum(blks_hit + blks_read), 0),
        2
    ) AS cache_hit_ratio_percent
FROM pg_stat_database
WHERE datname = current_database();

\echo ''
\echo '✓ 理想值：> 95%'
\echo '⚠️ < 90%：考虑增加 shared_buffers'
\echo ''

\echo '=========================================='
\echo '10. 活跃连接数'
\echo '=========================================='
SELECT
    state,
    COUNT(*) AS connections
FROM pg_stat_activity
WHERE datname = current_database()
GROUP BY state
ORDER BY connections DESC;

\echo ''
\echo '=========================================='
\echo '检测完成！'
\echo '=========================================='
