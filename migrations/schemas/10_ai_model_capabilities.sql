-- AI 模型能力表
-- 使用关系表设计，支持一个模型具有多种能力

CREATE TABLE ai_model_capabilities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id UUID NOT NULL REFERENCES ai_models(id) ON DELETE CASCADE,
    capability_type VARCHAR(50) NOT NULL, -- embedding, rerank, chat, vision, reasoning, function_calling, websearch
    
    -- Embedding 专属字段
    embedding_dimensions INT,
    
    -- Chat 专属字段
    supports_stream BOOLEAN DEFAULT true,
    supports_vision BOOLEAN DEFAULT false,
    supports_function_calling BOOLEAN DEFAULT false,
    supports_reasoning BOOLEAN DEFAULT false,
    supports_websearch BOOLEAN DEFAULT false,
    
    -- 扩展元数据
    metadata JSONB,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(model_id, capability_type)
);

-- 索引
CREATE INDEX idx_capabilities_type ON ai_model_capabilities(capability_type);
CREATE INDEX idx_capabilities_model ON ai_model_capabilities(model_id);

-- 注释
COMMENT ON TABLE ai_model_capabilities IS 'AI 模型能力关系表';
COMMENT ON COLUMN ai_model_capabilities.capability_type IS '能力类型：embedding, rerank, chat, vision, reasoning, function_calling, websearch';
COMMENT ON COLUMN ai_model_capabilities.embedding_dimensions IS 'Embedding 模型的向量维度（仅 embedding 类型）';
COMMENT ON COLUMN ai_model_capabilities.supports_stream IS '是否支持流式输出（chat 模型）';
COMMENT ON COLUMN ai_model_capabilities.supports_vision IS '是否支持视觉理解';
COMMENT ON COLUMN ai_model_capabilities.supports_function_calling IS '是否支持函数调用';
COMMENT ON COLUMN ai_model_capabilities.supports_reasoning IS '是否支持推理';
COMMENT ON COLUMN ai_model_capabilities.supports_websearch IS '是否支持联网搜索';
COMMENT ON COLUMN ai_model_capabilities.metadata IS '其他扩展元数据';
