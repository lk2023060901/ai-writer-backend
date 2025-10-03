-- +goose Up
-- 知识库和 AI 服务商配置表

-- Step 1: 创建 AI 服务商配置表
CREATE TABLE ai_provider_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 所有者 ID（'00000000-0000-0000-0000-000000000000' = 官方配置）
    owner_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',

    -- 服务商信息
    provider_type VARCHAR(50) NOT NULL,
    provider_name VARCHAR(100) NOT NULL,

    -- 认证配置（测试阶段明文存储）
    api_key TEXT NOT NULL,
    api_base_url VARCHAR(255),

    -- Embedding 模型配置
    embedding_model VARCHAR(100) NOT NULL,
    embedding_dimensions INTEGER NOT NULL,

    -- 能力标识（预留）
    supports_chat BOOLEAN DEFAULT false,
    supports_embedding BOOLEAN DEFAULT true,
    supports_rerank BOOLEAN DEFAULT false,

    -- 配额管理（用户配置用）
    monthly_quota BIGINT,
    used_tokens BIGINT DEFAULT 0,
    quota_reset_at TIMESTAMPTZ,

    -- 状态
    is_enabled BOOLEAN DEFAULT true,

    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,

    -- 外键约束
    CONSTRAINT fk_provider_owner FOREIGN KEY (owner_id)
        REFERENCES users(id) ON DELETE CASCADE
);

-- AI 服务商配置表索引
CREATE INDEX idx_provider_owner ON ai_provider_configs(owner_id);
CREATE INDEX idx_provider_type ON ai_provider_configs(provider_type);
CREATE INDEX idx_provider_enabled ON ai_provider_configs(is_enabled);
CREATE INDEX idx_provider_deleted ON ai_provider_configs(deleted_at);

-- Step 2: 创建知识库表
CREATE TABLE knowledge_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 所有者 ID（'00000000-0000-0000-0000-000000000000' = 官方知识库）
    owner_id UUID NOT NULL,

    -- 基本信息
    name VARCHAR(255) NOT NULL,

    -- 关联的 AI 服务商配置
    ai_provider_config_id UUID NOT NULL,

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
        REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_kb_ai_provider FOREIGN KEY (ai_provider_config_id)
        REFERENCES ai_provider_configs(id) ON DELETE RESTRICT
);

-- 知识库表索引
CREATE INDEX idx_kb_owner ON knowledge_bases(owner_id);
CREATE INDEX idx_kb_name ON knowledge_bases(name);
CREATE INDEX idx_kb_ai_provider ON knowledge_bases(ai_provider_config_id);
CREATE INDEX idx_kb_deleted ON knowledge_bases(deleted_at);
CREATE INDEX idx_kb_owner_created ON knowledge_bases(owner_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- Step 3: 创建文档表
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

-- 文档表索引
CREATE INDEX idx_doc_kb_id ON documents(knowledge_base_id);
CREATE INDEX idx_doc_file_type ON documents(file_type);
CREATE INDEX idx_doc_file_hash ON documents(file_hash);
CREATE INDEX idx_doc_status ON documents(status);
CREATE INDEX idx_doc_kb_status ON documents(knowledge_base_id, status);
CREATE INDEX idx_doc_kb_created ON documents(knowledge_base_id, created_at DESC);

-- Step 4: 创建分块表
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

-- 分块表索引
CREATE INDEX idx_chunk_doc_id ON chunks(document_id);
CREATE INDEX idx_chunk_kb_id ON chunks(knowledge_base_id);
CREATE UNIQUE INDEX idx_chunk_milvus_id ON chunks(milvus_id);
CREATE INDEX idx_chunk_doc_index ON chunks(document_id, chunk_index);

-- +goose Down
DROP TABLE IF EXISTS chunks CASCADE;
DROP TABLE IF EXISTS documents CASCADE;
DROP TABLE IF EXISTS knowledge_bases CASCADE;
DROP TABLE IF EXISTS ai_provider_configs CASCADE;
