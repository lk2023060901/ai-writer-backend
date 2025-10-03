-- AI 服务商配置模块
-- 包含：AI 服务商配置表（支持 Embedding、Chat、Rerank）

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

-- 索引
CREATE INDEX idx_provider_owner ON ai_provider_configs(owner_id);
CREATE INDEX idx_provider_type ON ai_provider_configs(provider_type);
CREATE INDEX idx_provider_enabled ON ai_provider_configs(is_enabled);
CREATE INDEX idx_provider_deleted ON ai_provider_configs(deleted_at);

-- 注释
COMMENT ON TABLE ai_provider_configs IS 'AI 服务商配置表：支持多种 AI 服务商（OpenAI、Claude、本地模型等）';
COMMENT ON COLUMN ai_provider_configs.owner_id IS '所有者 ID，00000000-0000-0000-0000-000000000000 表示官方配置';
COMMENT ON COLUMN ai_provider_configs.provider_type IS '服务商类型：openai、anthropic、local 等';
COMMENT ON COLUMN ai_provider_configs.api_key IS 'API 密钥，测试阶段明文存储（TODO: 生产环境需加密）';
COMMENT ON COLUMN ai_provider_configs.embedding_dimensions IS 'Embedding 向量维度，用于创建 Milvus collection';
