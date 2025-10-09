-- AI 模型同步日志表
-- 记录每次同步的历史，便于追踪模型变更

CREATE TABLE ai_model_sync_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
    sync_type VARCHAR(20) NOT NULL, -- manual（手动）, scheduled（定时）, triggered（触发）
    
    new_models_count INT DEFAULT 0,
    deprecated_models_count INT DEFAULT 0,
    updated_models_count INT DEFAULT 0,
    error_count INT DEFAULT 0,
    
    new_models JSONB, -- 新增模型列表
    deprecated_models JSONB, -- 下线模型列表
    updated_models JSONB, -- 更新模型列表
    
    error_message TEXT,
    synced_by VARCHAR(100), -- admin_id or 'system'
    synced_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_sync_logs_provider ON ai_model_sync_logs(provider_id);
CREATE INDEX idx_sync_logs_synced_at ON ai_model_sync_logs(synced_at DESC);

-- 注释
COMMENT ON TABLE ai_model_sync_logs IS 'AI 模型同步日志';
COMMENT ON COLUMN ai_model_sync_logs.sync_type IS '同步类型：manual（手动）, scheduled（定时）, triggered（触发）';
COMMENT ON COLUMN ai_model_sync_logs.new_models IS '新增的模型列表（JSON）';
COMMENT ON COLUMN ai_model_sync_logs.deprecated_models IS '已下线的模型列表（JSON）';
COMMENT ON COLUMN ai_model_sync_logs.updated_models IS '更新的模型列表（JSON）';
COMMENT ON COLUMN ai_model_sync_logs.synced_by IS '同步操作者：管理员 ID 或 system';
