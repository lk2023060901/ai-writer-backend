-- AI 服务商配置模块
-- 系统预设的 AI 服务商列表（只读，用户不可添加/删除）

CREATE TABLE ai_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_type VARCHAR(50) NOT NULL UNIQUE,
    provider_name VARCHAR(100) NOT NULL,
    api_base_url VARCHAR(255) NOT NULL,
    api_key TEXT,  -- 开发阶段明文存储，生产环境建议加密或使用环境变量
    is_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_ai_providers_type ON ai_providers(provider_type);

-- 注释
COMMENT ON TABLE ai_providers IS 'AI 服务商配置';
COMMENT ON COLUMN ai_providers.id IS '主键ID';
COMMENT ON COLUMN ai_providers.provider_type IS '提供商类型（唯一标识）';
COMMENT ON COLUMN ai_providers.provider_name IS '提供商显示名称';
COMMENT ON COLUMN ai_providers.api_base_url IS 'API 基础地址';
COMMENT ON COLUMN ai_providers.api_key IS 'API 密钥（开发环境明文，生产环境使用环境变量）';
COMMENT ON COLUMN ai_providers.is_enabled IS '是否启用';

-- 插入预设数据
INSERT INTO ai_providers (provider_type, provider_name, api_base_url, api_key, is_enabled) VALUES
('siliconflow', '硅基流动', 'https://api.siliconflow.cn/v1', NULL, true),
('openai', 'OpenAI', 'https://api.openai.com/v1', NULL, true),
('anthropic', 'Anthropic', 'https://api.anthropic.com/v1', NULL, true);
