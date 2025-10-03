-- 知识库模块
-- 包含：知识库表（关联 AI 服务商配置，管理 Milvus collection）

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

-- 索引
CREATE INDEX idx_kb_owner ON knowledge_bases(owner_id);
CREATE INDEX idx_kb_name ON knowledge_bases(name);
CREATE INDEX idx_kb_ai_provider ON knowledge_bases(ai_provider_config_id);
CREATE INDEX idx_kb_deleted ON knowledge_bases(deleted_at);
CREATE INDEX idx_kb_owner_created ON knowledge_bases(owner_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- 注释
COMMENT ON TABLE knowledge_bases IS '知识库表：管理文档集合和 Milvus collection';
COMMENT ON COLUMN knowledge_bases.owner_id IS '所有者 ID，00000000-0000-0000-0000-000000000000 表示官方知识库';
COMMENT ON COLUMN knowledge_bases.chunk_strategy IS 'Chunking 策略：token（按 token 分块）、sentence（按句子分块）';
COMMENT ON COLUMN knowledge_bases.milvus_collection IS 'Milvus collection 名称，全局唯一，格式：kb_{uuid}';
