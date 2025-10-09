-- AI 模型配置表
-- 存储各个 AI 服务商提供的具体模型基础信息
-- 模型能力通过 ai_model_capabilities 关系表管理

CREATE TABLE ai_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
    model_name VARCHAR(255) NOT NULL, -- e.g., BAAI/bge-large-zh-v1.5
    display_name VARCHAR(255), -- 显示名称
    max_tokens INT, -- 最大 token 数
    is_enabled BOOLEAN DEFAULT true,
    last_verified_at TIMESTAMPTZ, -- 最后验证时间
    verification_status VARCHAR(20) DEFAULT 'unknown', -- available, deprecated, error, unknown
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider_id, model_name)
);

-- 索引
CREATE INDEX idx_ai_models_provider ON ai_models(provider_id);
CREATE INDEX idx_ai_models_enabled ON ai_models(is_enabled);
CREATE INDEX idx_ai_models_verification ON ai_models(verification_status);

-- 注释
COMMENT ON TABLE ai_models IS 'AI 模型配置表';
COMMENT ON COLUMN ai_models.id IS '主键ID';
COMMENT ON COLUMN ai_models.provider_id IS '关联的 AI 服务商 ID';
COMMENT ON COLUMN ai_models.model_name IS '模型标识名称';
COMMENT ON COLUMN ai_models.display_name IS '显示名称';
COMMENT ON COLUMN ai_models.max_tokens IS '最大 token 数';
COMMENT ON COLUMN ai_models.is_enabled IS '是否启用';
COMMENT ON COLUMN ai_models.last_verified_at IS '最后验证时间（用于检测模型是否仍可用）';
COMMENT ON COLUMN ai_models.verification_status IS '验证状态：available（可用）, deprecated（已下线）, error（错误）, unknown（未知）';

-- 注意：不再预先插入数据，改为通过同步接口动态获取
