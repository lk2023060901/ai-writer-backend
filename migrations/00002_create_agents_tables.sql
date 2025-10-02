-- +goose Up
-- +goose StatementBegin

-- 用户智能体表
CREATE TABLE agents (
    -- 主键 (UUID v7, 由应用层生成)
    id UUID PRIMARY KEY,

    -- 所有者
    owner_id UUID NOT NULL,

    -- 基础信息
    name VARCHAR(255) NOT NULL,
    emoji VARCHAR(10) DEFAULT '🤖',  -- emoji 字符，默认机器人
    prompt TEXT NOT NULL,             -- 提示词内容

    -- 关联信息
    knowledge_base_ids JSONB NOT NULL DEFAULT '[]'::JSONB,  -- UUID 数组（暂不验证）
    tags JSONB NOT NULL DEFAULT '[]'::JSONB,                 -- 标签数组

    -- 类型（固定为 'agent'）
    type VARCHAR(50) NOT NULL DEFAULT 'agent',

    -- 状态
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,

    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,

    -- 外键约束
    CONSTRAINT fk_agents_owner FOREIGN KEY (owner_id)
        REFERENCES users(id) ON DELETE CASCADE
);

-- 索引：所有者查询（软删除过滤）
CREATE INDEX idx_agents_owner_id ON agents(owner_id) WHERE deleted_at IS NULL;

-- 索引：标签查询（GIN 索引支持 JSONB）
CREATE INDEX idx_agents_tags ON agents USING GIN(tags);

-- 索引：启用状态查询（组合索引，支持排序）
CREATE INDEX idx_agents_is_enabled ON agents(is_enabled, owner_id) WHERE deleted_at IS NULL;

-- 索引：软删除
CREATE INDEX idx_agents_deleted_at ON agents(deleted_at);

-- 注释
COMMENT ON TABLE agents IS '用户智能体表：用户创建的私有智能体';
COMMENT ON COLUMN agents.owner_id IS '所有者 UUID（外键到 users 表）';
COMMENT ON COLUMN agents.emoji IS 'emoji 字符，如 🤖📝✍️，非必填，默认 🤖';
COMMENT ON COLUMN agents.prompt IS '系统提示词，纯文本格式';
COMMENT ON COLUMN agents.knowledge_base_ids IS 'JSONB 数组，存储关联的知识库 UUID，示例：["uuid1", "uuid2"]，暂不做外键验证';
COMMENT ON COLUMN agents.tags IS 'JSONB 数组，标签列表，示例：["编程助手", "视频文案"]';
COMMENT ON COLUMN agents.type IS '智能体类型，当前固定为 agent';
COMMENT ON COLUMN agents.is_enabled IS '是否启用，禁用的智能体显示在最后且不能添加到快捷列表';

-- 官方智能体表（无 owner_id 字段）
CREATE TABLE official_agents (
    -- 主键 (UUID v7, 由应用层生成)
    id UUID PRIMARY KEY,

    -- 基础信息
    name VARCHAR(255) NOT NULL,
    emoji VARCHAR(10) DEFAULT '🤖',
    prompt TEXT NOT NULL,

    -- 关联信息
    knowledge_base_ids JSONB NOT NULL DEFAULT '[]'::JSONB,
    tags JSONB NOT NULL DEFAULT '[]'::JSONB,

    -- 类型
    type VARCHAR(50) NOT NULL DEFAULT 'agent',

    -- 状态
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,

    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

-- 索引：标签查询
CREATE INDEX idx_official_agents_tags ON official_agents USING GIN(tags);

-- 索引：启用状态
CREATE INDEX idx_official_agents_is_enabled ON official_agents(is_enabled) WHERE deleted_at IS NULL;

-- 索引：软删除
CREATE INDEX idx_official_agents_deleted_at ON official_agents(deleted_at);

-- 注释
COMMENT ON TABLE official_agents IS '官方智能体表：系统预设的官方智能体，所有用户可见（无 owner_id 字段）';
COMMENT ON COLUMN official_agents.emoji IS 'emoji 字符，如 🤖📝✍️，非必填，默认 🤖';
COMMENT ON COLUMN official_agents.prompt IS '系统提示词，纯文本格式';
COMMENT ON COLUMN official_agents.knowledge_base_ids IS 'JSONB 数组，存储关联的知识库 UUID，暂不做外键验证';
COMMENT ON COLUMN official_agents.tags IS 'JSONB 数组，标签列表，示例：["编程助手", "视频文案"]';
COMMENT ON COLUMN official_agents.is_enabled IS '是否启用，禁用的官方智能体不显示给用户';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS official_agents;
DROP TABLE IF EXISTS agents;
-- +goose StatementEnd
