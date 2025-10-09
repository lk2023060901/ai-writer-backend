-- +goose Up
-- +goose StatementBegin

-- ============================================================================
-- 数据库初始化脚本
-- 创建所有表：用户、智能体、AI 配置、知识库、文档、对话
-- ============================================================================

-- ============================================================================
-- 1. 用户认证模块
-- ============================================================================

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

-- ============================================================================
-- 2. 智能体模块
-- ============================================================================

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

-- ============================================================================
-- 3. AI 服务商配置模块
-- ============================================================================

-- AI 服务商表（系统预设）
CREATE TABLE ai_providers (
    id SERIAL PRIMARY KEY,
    provider_type VARCHAR(50) NOT NULL UNIQUE,
    provider_name VARCHAR(100) NOT NULL,
    api_base_url VARCHAR(255),
    api_key TEXT,
    is_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE ai_providers IS 'AI 服务商配置表（系统预设，不可由用户自定义）';

INSERT INTO ai_providers (provider_type, provider_name, api_base_url, api_key, is_enabled) VALUES
('siliconflow', '硅基流动', 'https://api.siliconflow.cn', 'sk-gkqnwrnkmxqdeuqcpnntuzjtsfmbloyemaolyaxpuicfczxo', true),
('openai', 'OpenAI', 'https://api.openai.com', NULL, true),
('anthropic', 'Anthropic', 'https://api.anthropic.com', 'sk-2QMrtTUhFf3HmxrFkHfIXnqBuGTxXVlDT4eVxxTbX02B0fl5', true),
('gemini', 'Google Gemini', 'https://generativelanguage.googleapis.com', 'AIzaSyASzFYygdDl3nXWUJ_SQxHY-XI8Pz1Ib7E', true);

-- AI 模型表
CREATE TABLE ai_models (
    id SERIAL PRIMARY KEY,
    provider_id INT NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
    model_types JSONB NOT NULL,
    model_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    embedding_dimensions INT,
    max_tokens INT,
    is_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider_id, model_name)
);

CREATE INDEX idx_ai_models_provider ON ai_models(provider_id);
CREATE INDEX idx_ai_models_types ON ai_models USING GIN(model_types);

COMMENT ON TABLE ai_models IS 'AI 模型配置表';
COMMENT ON COLUMN ai_models.model_types IS '模型支持的类型：["chat"], ["embedding"], ["rerank"] 或组合';
COMMENT ON COLUMN ai_models.embedding_dimensions IS 'Embedding 默认维度（可通过 API 动态获取）';

INSERT INTO ai_models (provider_id, model_types, model_name, display_name, embedding_dimensions, max_tokens) VALUES
(1, '["embedding"]', 'BAAI/bge-large-zh-v1.5', 'BGE Large 中文 v1.5', 1024, NULL),
(1, '["embedding"]', 'BAAI/bge-m3', 'BGE M3 多语言', 1024, NULL),
(1, '["chat"]', 'Qwen/Qwen2.5-7B-Instruct', 'Qwen 2.5 7B', NULL, 32768),
(1, '["rerank"]', 'BAAI/bge-reranker-v2-m3', 'BGE Reranker v2 M3', NULL, NULL),
(2, '["embedding"]', 'text-embedding-3-small', 'Text Embedding 3 Small', 1536, NULL),
(2, '["embedding"]', 'text-embedding-3-large', 'Text Embedding 3 Large', 3072, NULL),
(2, '["chat"]', 'gpt-4o', 'GPT-4o', NULL, 128000),
(2, '["chat"]', 'gpt-4o-mini', 'GPT-4o Mini', NULL, 128000),
(3, '["chat"]', 'claude-3-7-sonnet-20250219', 'Claude 3.7 Sonnet', NULL, 200000),
(3, '["chat"]', 'claude-3-5-sonnet-20241022', 'Claude 3.5 Sonnet', NULL, 200000),
(4, '["embedding"]', 'text-embedding-004', 'Text Embedding 004', 768, NULL),
(4, '["chat"]', 'gemini-2.0-flash-exp', 'Gemini 2.0 Flash', NULL, 1048576),
(4, '["chat"]', 'gemini-1.5-pro', 'Gemini 1.5 Pro', NULL, 2097152);

-- ============================================================================
-- 4. 知识库模块
-- ============================================================================

CREATE TABLE knowledge_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 所有者 ID（'00000000-0000-0000-0000-000000000000' = 官方知识库）
    owner_id UUID NOT NULL,

    -- 基本信息
    name VARCHAR(255) NOT NULL,

    -- 关联的 AI 模型
    embedding_model_id INT NOT NULL REFERENCES ai_models(id) ON DELETE RESTRICT,
    rerank_model_id INT REFERENCES ai_models(id) ON DELETE SET NULL,

    -- Chunking 配置
    chunk_size INTEGER NOT NULL DEFAULT 512,
    chunk_overlap INTEGER NOT NULL DEFAULT 50,
    chunk_strategy VARCHAR(50) NOT NULL DEFAULT 'token',

    -- Milvus Collection 名称（全局唯一）
    milvus_collection VARCHAR(100) NOT NULL UNIQUE,

    -- 统计信息
    document_count BIGINT NOT NULL DEFAULT 0,

    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,

    -- 约束
    CONSTRAINT fk_kb_owner FOREIGN KEY (owner_id)
        REFERENCES users(id) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_kb_owner ON knowledge_bases(owner_id);
CREATE INDEX idx_kb_name ON knowledge_bases(name);
CREATE INDEX idx_kb_embedding_model ON knowledge_bases(embedding_model_id);
CREATE INDEX idx_kb_rerank_model ON knowledge_bases(rerank_model_id);
CREATE INDEX idx_kb_deleted ON knowledge_bases(deleted_at);
CREATE INDEX idx_kb_owner_created ON knowledge_bases(owner_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- 注释
COMMENT ON TABLE knowledge_bases IS '知识库表：管理文档集合和 Milvus collection';
COMMENT ON COLUMN knowledge_bases.owner_id IS '所有者 ID，00000000-0000-0000-0000-000000000000 表示官方知识库';
COMMENT ON COLUMN knowledge_bases.chunk_strategy IS 'Chunking 策略：token（按 token 分块）、sentence（按句子分块）';
COMMENT ON COLUMN knowledge_bases.milvus_collection IS 'Milvus collection 名称，全局唯一，格式：kb_{uuid}';

-- ============================================================================
-- 5. 文档和分块模块
-- ============================================================================

-- 文档表
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_base_id UUID NOT NULL,

    -- 文件信息
    filename VARCHAR(255) NOT NULL,
    file_type VARCHAR(50) NOT NULL,
    file_size BIGINT NOT NULL,
    file_hash VARCHAR(64) NOT NULL,

    -- MinIO 存储路径
    minio_bucket VARCHAR(100) NOT NULL,
    minio_object_key VARCHAR(500) NOT NULL,

    -- 处理状态
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error_message TEXT,

    -- 统计信息
    chunk_count INTEGER NOT NULL DEFAULT 0,
    token_count INTEGER NOT NULL DEFAULT 0,

    -- 元数据
    metadata JSONB,

    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- 约束
    CONSTRAINT fk_doc_kb FOREIGN KEY (knowledge_base_id)
        REFERENCES knowledge_bases(id) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_doc_kb_id ON documents(knowledge_base_id);
CREATE INDEX idx_doc_file_type ON documents(file_type);
CREATE INDEX idx_doc_file_hash ON documents(file_hash);
CREATE INDEX idx_doc_status ON documents(status);
CREATE INDEX idx_doc_kb_status ON documents(knowledge_base_id, status);
CREATE INDEX idx_doc_kb_created ON documents(knowledge_base_id, created_at DESC);

-- 注释
COMMENT ON TABLE documents IS '文档表：存储上传的文档信息和处理状态';
COMMENT ON COLUMN documents.status IS '处理状态：pending、processing、completed、failed';
COMMENT ON COLUMN documents.file_hash IS 'SHA256 哈希值，用于去重';
COMMENT ON COLUMN documents.minio_object_key IS 'MinIO 对象键，格式：knowledge_bases/{kb_id}/{uuid}.{ext}';

-- 分块表
CREATE TABLE chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL,
    knowledge_base_id UUID NOT NULL,

    -- 分块信息
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL,
    token_count INTEGER NOT NULL,

    -- Milvus 向量 ID
    milvus_id VARCHAR(100) NOT NULL UNIQUE,

    -- 元数据
    metadata JSONB,

    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- 约束
    CONSTRAINT fk_chunk_doc FOREIGN KEY (document_id)
        REFERENCES documents(id) ON DELETE CASCADE,
    CONSTRAINT fk_chunk_kb FOREIGN KEY (knowledge_base_id)
        REFERENCES knowledge_bases(id) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_chunk_doc_id ON chunks(document_id);
CREATE INDEX idx_chunk_kb_id ON chunks(knowledge_base_id);
CREATE UNIQUE INDEX idx_chunk_milvus_id ON chunks(milvus_id);
CREATE INDEX idx_chunk_doc_index ON chunks(document_id, chunk_index);

-- 注释
COMMENT ON TABLE chunks IS '分块表：文档分块后的文本片段，与 Milvus 向量一一对应';
COMMENT ON COLUMN chunks.chunk_index IS '分块在文档中的序号，从 0 开始';
COMMENT ON COLUMN chunks.milvus_id IS 'Milvus 向量 ID，格式：{doc_id}_{chunk_index}';

-- ============================================================================
-- 6. 对话模块
-- ============================================================================

-- 对话主题表
CREATE TABLE topics (
    -- 主键 (UUID v7, 由应用层生成)
    id UUID PRIMARY KEY,

    -- 关联的智能体 ID (可以是 agents 或 official_agents)
    assistant_id UUID NOT NULL,

    -- 主题名称
    name VARCHAR(255) NOT NULL,

    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

-- 索引：根据 assistant_id 查询对话主题
CREATE INDEX idx_topics_assistant_id ON topics(assistant_id) WHERE deleted_at IS NULL;

-- 索引：软删除
CREATE INDEX idx_topics_deleted_at ON topics(deleted_at);

-- 索引：最近创建的对话（组合索引）
CREATE INDEX idx_topics_created_at ON topics(created_at DESC) WHERE deleted_at IS NULL;

-- 注释
COMMENT ON TABLE topics IS '对话主题表：用户与智能体的对话主题';
COMMENT ON COLUMN topics.assistant_id IS '关联的智能体 ID（可以是 agents.id 或 official_agents.id）';
COMMENT ON COLUMN topics.name IS '对话主题名称，可自动生成或用户指定';

-- 消息表
CREATE TABLE messages (
    -- 主键 (UUID v7, 由应用层生成)
    id UUID PRIMARY KEY,

    -- 所属对话主题
    topic_id UUID NOT NULL,

    -- 消息角色
    role VARCHAR(20) NOT NULL CHECK (role IN ('user', 'assistant')),

    -- 消息内容（JSONB 数组，ContentBlock[]）
    content_blocks JSONB NOT NULL,

    -- Token 消耗（仅 AI 回复有值）
    token_count INTEGER,

    -- 发送时间
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- 外键约束
    CONSTRAINT fk_messages_topic FOREIGN KEY (topic_id)
        REFERENCES topics(id) ON DELETE CASCADE
);

-- 索引：根据 topic_id 查询消息列表（按时间排序）
CREATE INDEX idx_messages_topic_id ON messages(topic_id, created_at ASC);

-- 索引：按时间查询（用于分页）
CREATE INDEX idx_messages_created_at ON messages(created_at);

-- 注释
COMMENT ON TABLE messages IS '消息表：对话中的每条消息（用户输入或 AI 回复）';
COMMENT ON COLUMN messages.topic_id IS '所属对话主题 UUID（外键到 topics 表）';
COMMENT ON COLUMN messages.role IS '消息角色：user（用户）或 assistant（AI）';
COMMENT ON COLUMN messages.content_blocks IS 'JSONB 数组，存储 ContentBlock[] 格式的消息内容，支持 text/thinking/tool_use/tool_result 等类型';
COMMENT ON COLUMN messages.token_count IS 'AI 回复消耗的 token 数量，用户消息为 NULL';
COMMENT ON COLUMN messages.created_at IS '消息发送时间';

-- ============================================================================
-- 8. 智能体快捷访问列表
-- ============================================================================

-- 智能体快捷访问列表
CREATE TABLE assistant_favorites (
    -- 主键 (UUID v7, 由应用层生成)
    id UUID PRIMARY KEY,

    -- 用户 ID
    user_id UUID NOT NULL,

    -- 智能体 ID
    assistant_id UUID NOT NULL,

    -- 排序顺序（数字越小越靠前，支持用户自定义排序）
    sort_order INTEGER NOT NULL DEFAULT 0,

    -- 添加时间
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- 唯一约束：同一个用户不能重复添加同一个智能体
    CONSTRAINT uk_user_assistant UNIQUE (user_id, assistant_id)
);

-- 索引：根据 user_id 查询该用户的快捷访问列表（按排序顺序）
CREATE INDEX idx_favorites_user_id ON assistant_favorites(user_id, sort_order ASC);

-- 索引：根据 assistant_id 查询有多少用户收藏了该智能体
CREATE INDEX idx_favorites_assistant_id ON assistant_favorites(assistant_id);

-- 索引：按添加时间查询
CREATE INDEX idx_favorites_created_at ON assistant_favorites(created_at DESC);

-- 注释
COMMENT ON TABLE assistant_favorites IS '智能体快捷访问列表：用户收藏的智能体';
COMMENT ON COLUMN assistant_favorites.user_id IS '用户 ID（外键到 users 表）';
COMMENT ON COLUMN assistant_favorites.assistant_id IS '智能体 ID（可以是 agents.id 或 official_agents.id）';
COMMENT ON COLUMN assistant_favorites.sort_order IS '排序顺序，数字越小越靠前，支持用户自定义排序';
COMMENT ON COLUMN assistant_favorites.created_at IS '添加到快捷访问的时间';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- 按相反顺序删除表（先删除有外键依赖的表）
DROP TABLE IF EXISTS assistant_favorites CASCADE;
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS topics CASCADE;
DROP TABLE IF EXISTS chunks CASCADE;
DROP TABLE IF EXISTS documents CASCADE;
DROP TABLE IF EXISTS knowledge_bases CASCADE;
DROP TABLE IF EXISTS ai_provider_configs CASCADE;
DROP TABLE IF EXISTS official_agents CASCADE;
DROP TABLE IF EXISTS agents CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- +goose StatementEnd
