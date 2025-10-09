-- 简化模型能力存储结构：从关联表改为 JSONB

-- 1. 在 ai_models 表添加新字段
ALTER TABLE ai_models
    ADD COLUMN IF NOT EXISTS capabilities JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS supports_stream BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS supports_vision BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS supports_function_calling BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS supports_reasoning BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS supports_web_search BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS embedding_dimensions INTEGER;

-- 2. 迁移现有数据（从 ai_model_capabilities 到 ai_models）
UPDATE ai_models m
SET
    capabilities = (
        SELECT jsonb_agg(DISTINCT capability_type)
        FROM ai_model_capabilities
        WHERE model_id = m.id
    ),
    supports_stream = COALESCE((
        SELECT bool_or(supports_stream)
        FROM ai_model_capabilities
        WHERE model_id = m.id
    ), false),
    supports_vision = COALESCE((
        SELECT bool_or(supports_vision)
        FROM ai_model_capabilities
        WHERE model_id = m.id
    ), false),
    supports_function_calling = COALESCE((
        SELECT bool_or(supports_function_calling)
        FROM ai_model_capabilities
        WHERE model_id = m.id
    ), false),
    supports_reasoning = COALESCE((
        SELECT bool_or(supports_reasoning)
        FROM ai_model_capabilities
        WHERE model_id = m.id
    ), false),
    supports_web_search = COALESCE((
        SELECT bool_or(supports_web_search)
        FROM ai_model_capabilities
        WHERE model_id = m.id
    ), false),
    embedding_dimensions = (
        SELECT embedding_dimensions
        FROM ai_model_capabilities
        WHERE model_id = m.id
          AND capability_type = 'embedding'
        LIMIT 1
    );

-- 3. 修正 NULL 的 capabilities
UPDATE ai_models
SET capabilities = '[]'::jsonb
WHERE capabilities IS NULL;

-- 4. 删除旧的关联表
DROP TABLE IF EXISTS ai_model_capabilities;

-- 5. 添加索引优化查询性能
CREATE INDEX IF NOT EXISTS idx_ai_models_capabilities ON ai_models USING gin(capabilities);
CREATE INDEX IF NOT EXISTS idx_ai_models_supports_vision ON ai_models(supports_vision) WHERE supports_vision = true;
CREATE INDEX IF NOT EXISTS idx_ai_models_supports_reasoning ON ai_models(supports_reasoning) WHERE supports_reasoning = true;

-- 6. 添加注释
COMMENT ON COLUMN ai_models.capabilities IS '模型能力类型数组，如 ["chat", "embedding", "rerank"]';
COMMENT ON COLUMN ai_models.supports_stream IS '是否支持流式输出';
COMMENT ON COLUMN ai_models.supports_vision IS '是否支持视觉理解';
COMMENT ON COLUMN ai_models.supports_function_calling IS '是否支持函数调用';
COMMENT ON COLUMN ai_models.supports_reasoning IS '是否支持推理能力';
COMMENT ON COLUMN ai_models.supports_web_search IS '是否支持联网搜索';
COMMENT ON COLUMN ai_models.embedding_dimensions IS 'Embedding 模型的向量维度';
