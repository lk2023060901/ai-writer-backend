-- ============================================
-- 用户表性能基准测试
-- 使用方法：psql -U postgres -d ai_writer -f scripts/benchmark_users_table.sql
-- ============================================

\echo '=========================================='
\echo '准备测试：创建测试数据'
\echo '=========================================='

-- 启用计时
\timing on

-- 插入测试数据（如果还没有）
DO $$
DECLARE
    user_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO user_count FROM users;

    IF user_count < 1000 THEN
        RAISE NOTICE '插入测试数据中...';
        INSERT INTO users (name, email, password_hash, email_verified, created_at)
        SELECT
            'Test User ' || i,
            'testuser' || i || '@example.com',
            '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5lW3Le8gVf7Vy',  -- "password"
            (i % 2 = 0),  -- 50% 已验证邮箱
            NOW() - (i || ' hours')::INTERVAL
        FROM generate_series(1, 1000) AS i
        ON CONFLICT (email) DO NOTHING;

        RAISE NOTICE '插入完成！共 % 条记录', (SELECT COUNT(*) FROM users);
    ELSE
        RAISE NOTICE '已存在 % 条用户数据，跳过插入', user_count;
    END IF;
END $$;

\echo ''
\echo '=========================================='
\echo '测试 1: 邮箱精确查询（登录场景）'
\echo '期望：< 5ms，使用 Index Scan'
\echo '=========================================='
EXPLAIN ANALYZE
SELECT *
FROM users
WHERE email = 'testuser500@example.com'
  AND deleted_at IS NULL;

\echo ''
\echo '=========================================='
\echo '测试 2: 邮箱查询 + 密码验证'
\echo '期望：< 5ms'
\echo '=========================================='
EXPLAIN ANALYZE
SELECT id, name, email, password_hash, two_factor_enabled
FROM users
WHERE email = 'testuser500@example.com'
  AND deleted_at IS NULL;

\echo ''
\echo '=========================================='
\echo '测试 3: 分页查询（用户列表）'
\echo '期望：< 20ms'
\echo '=========================================='
EXPLAIN ANALYZE
SELECT id, name, email, email_verified, created_at
FROM users
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 20 OFFSET 0;

\echo ''
\echo '=========================================='
\echo '测试 4: 邮箱验证 Token 查询'
\echo '期望：< 10ms，使用 Index Scan'
\echo '=========================================='
-- 先插入一个测试 token
UPDATE users
SET email_verification_token = 'test_token_12345',
    email_verification_expires_at = NOW() + INTERVAL '24 hours'
WHERE email = 'testuser100@example.com';

EXPLAIN ANALYZE
SELECT id, email, email_verified
FROM users
WHERE email_verification_token = 'test_token_12345'
  AND email_verification_expires_at > NOW()
  AND deleted_at IS NULL;

\echo ''
\echo '=========================================='
\echo '测试 5: 密码重置 Token 查询'
\echo '期望：< 10ms，使用 Index Scan'
\echo '=========================================='
-- 插入测试 token
UPDATE users
SET password_reset_token = 'reset_token_67890',
    password_reset_expires_at = NOW() + INTERVAL '1 hour'
WHERE email = 'testuser200@example.com';

EXPLAIN ANALYZE
SELECT id, email
FROM users
WHERE password_reset_token = 'reset_token_67890'
  AND password_reset_expires_at > NOW()
  AND deleted_at IS NULL;

\echo ''
\echo '=========================================='
\echo '测试 6: 账户锁定查询'
\echo '期望：< 10ms'
\echo '=========================================='
-- 插入测试锁定账户
UPDATE users
SET locked_until = NOW() + INTERVAL '15 minutes',
    failed_login_attempts = 5
WHERE email = 'testuser300@example.com';

EXPLAIN ANALYZE
SELECT id, email, locked_until, failed_login_attempts
FROM users
WHERE locked_until > NOW()
  AND deleted_at IS NULL;

\echo ''
\echo '=========================================='
\echo '测试 7: 统计查询 - 总用户数'
\echo '期望：< 50ms'
\echo '=========================================='
EXPLAIN ANALYZE
SELECT COUNT(*) AS total_users
FROM users
WHERE deleted_at IS NULL;

\echo ''
\echo '=========================================='
\echo '测试 8: 统计查询 - 已验证邮箱用户数'
\echo '期望：< 100ms'
\echo '=========================================='
EXPLAIN ANALYZE
SELECT COUNT(*) AS verified_users
FROM users
WHERE email_verified = true
  AND deleted_at IS NULL;

\echo ''
\echo '=========================================='
\echo '测试 9: 软删除用户查询'
\echo '期望：< 5ms'
\echo '=========================================='
-- 标记一个用户为已删除
UPDATE users
SET deleted_at = NOW()
WHERE email = 'testuser999@example.com';

EXPLAIN ANALYZE
SELECT id, name, email, deleted_at
FROM users
WHERE email = 'testuser999@example.com';

\echo ''
\echo '=========================================='
\echo '测试 10: 批量 ID 查询（IN 查询）'
\echo '期望：< 10ms'
\echo '=========================================='
EXPLAIN ANALYZE
SELECT id, name, email
FROM users
WHERE id IN (1, 2, 3, 4, 5, 10, 20, 30, 40, 50)
  AND deleted_at IS NULL;

\echo ''
\echo '=========================================='
\echo '性能基准测试完成！'
\echo '=========================================='
\echo ''
\echo '评估标准：'
\echo '✓ Execution Time < 5ms：优秀'
\echo '✓ Execution Time < 20ms：良好'
\echo '⚠️ Execution Time < 100ms：可接受'
\echo '❌ Execution Time > 100ms：需要优化'
\echo ''
\echo '优化建议：'
\echo '1. 检查是否使用了索引（Index Scan vs Seq Scan）'
\echo '2. 检查返回的行数是否合理'
\echo '3. 考虑添加复合索引（如：email + deleted_at）'
\echo '4. 运行 ANALYZE users 更新统计信息'
\echo '5. 监控慢查询日志'
\echo ''

-- 清理测试数据
\echo '清理测试 tokens...'
UPDATE users
SET email_verification_token = NULL,
    email_verification_expires_at = NULL,
    password_reset_token = NULL,
    password_reset_expires_at = NULL,
    locked_until = NULL,
    failed_login_attempts = 0
WHERE email IN ('testuser100@example.com', 'testuser200@example.com', 'testuser300@example.com');

UPDATE users
SET deleted_at = NULL
WHERE email = 'testuser999@example.com';

\echo '完成！'
