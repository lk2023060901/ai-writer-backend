-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    -- 主键 (UUID v7, 由应用层生成)
    id UUID PRIMARY KEY,

    -- 基础信息
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,

    -- 认证信息
    password_hash VARCHAR(255) NOT NULL,

    -- JWT Refresh Token
    refresh_token VARCHAR(512),
    refresh_token_expires_at TIMESTAMPTZ,

    -- 双因子认证 (2FA)
    two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    two_factor_secret VARCHAR(32),
    two_factor_backup_codes JSONB,

    -- 登录追踪
    last_login_at TIMESTAMPTZ,
    last_login_ip VARCHAR(45), -- 支持 IPv4 和 IPv6
    failed_login_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,

    -- 邮箱验证
    email_verification_token VARCHAR(64),
    email_verification_expires_at TIMESTAMPTZ,

    -- 密码重置
    password_reset_token VARCHAR(64),
    password_reset_expires_at TIMESTAMPTZ,

    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

-- 唯一索引（软删除下的唯一邮箱）
CREATE UNIQUE INDEX idx_users_email ON users (email) WHERE deleted_at IS NULL;

-- 查询优化索引
CREATE INDEX idx_users_deleted_at ON users (deleted_at);
CREATE INDEX idx_users_email_verification_token ON users (email_verification_token) WHERE email_verification_token IS NOT NULL;
CREATE INDEX idx_users_password_reset_token ON users (password_reset_token) WHERE password_reset_token IS NOT NULL;
CREATE INDEX idx_users_locked_until ON users (locked_until) WHERE locked_until IS NOT NULL;

-- 注释
COMMENT ON TABLE users IS '用户表：支持密码认证、JWT Refresh Token、双因子认证';
COMMENT ON COLUMN users.password_hash IS 'bcrypt 哈希值（cost=12），由 Go 代码生成';
COMMENT ON COLUMN users.two_factor_backup_codes IS 'JSONB 格式：[{"hash":"$2a$12$...","used":false,"used_at":null,"used_ip":null}]';
COMMENT ON COLUMN users.failed_login_attempts IS '连续登录失败次数，成功登录后重置为 0';
COMMENT ON COLUMN users.locked_until IS '账户锁定截止时间，5 次失败后锁定 15 分钟';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
