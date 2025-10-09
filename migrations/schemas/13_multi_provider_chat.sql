-- 多服务商聊天响应表

-- 聊天会话表（扩展）
-- 已有 topics 表，这里添加注释说明支持多服务商
COMMENT ON TABLE topics IS '聊天会话表，支持多服务商并发响应';

-- 消息表增强（支持多服务商响应）
-- 为 messages 表添加服务商信息字段
ALTER TABLE messages ADD COLUMN IF NOT EXISTS provider VARCHAR(50);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS model VARCHAR(100);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS finish_reason VARCHAR(50);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS metadata JSONB;

COMMENT ON COLUMN messages.provider IS 'AI 服务商：openai, anthropic, gemini, grok 等';
COMMENT ON COLUMN messages.model IS '使用的模型：gpt-4o, claude-3-5-sonnet 等';
COMMENT ON COLUMN messages.finish_reason IS '完成原因：stop, length, content_filter 等';
COMMENT ON COLUMN messages.metadata IS '元数据：token 数量、耗时等';

-- 多服务商响应记录表（可选，用于分析和对比）
CREATE TABLE IF NOT EXISTS multi_provider_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic_id UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id VARCHAR(100) NOT NULL,

    -- 请求信息
    user_message TEXT NOT NULL,
    content_blocks JSONB,
    enable_web_search BOOLEAN DEFAULT FALSE,

    -- 服务商配置
    providers JSONB NOT NULL, -- 使用的服务商列表

    -- 响应汇总
    total_providers INT NOT NULL DEFAULT 1,
    completed_providers INT NOT NULL DEFAULT 0,
    failed_providers INT NOT NULL DEFAULT 0,

    -- 时间信息
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INT,

    -- 元数据
    metadata JSONB,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_multi_provider_responses_topic_id ON multi_provider_responses(topic_id);
CREATE INDEX idx_multi_provider_responses_user_id ON multi_provider_responses(user_id);
CREATE INDEX idx_multi_provider_responses_session_id ON multi_provider_responses(session_id);
CREATE INDEX idx_multi_provider_responses_started_at ON multi_provider_responses(started_at DESC);

COMMENT ON TABLE multi_provider_responses IS '多服务商响应记录表，用于追踪和分析并发请求';
COMMENT ON COLUMN multi_provider_responses.providers IS '服务商配置列表 JSON 格式：[{provider, model, temperature}]';
COMMENT ON COLUMN multi_provider_responses.metadata IS '元数据：总 token 数、总成本等';

-- 单个服务商响应详情表
CREATE TABLE IF NOT EXISTS provider_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    multi_response_id UUID NOT NULL REFERENCES multi_provider_responses(id) ON DELETE CASCADE,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,

    -- 服务商信息
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(100) NOT NULL,

    -- 响应内容
    content TEXT,
    content_blocks JSONB,

    -- 完成信息
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending | streaming | completed | failed
    finish_reason VARCHAR(50),
    error_message TEXT,

    -- Token 统计
    input_tokens INT,
    output_tokens INT,
    total_tokens INT,

    -- 性能指标
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    first_token_at TIMESTAMP, -- 首个 token 到达时间（TTFT）
    completed_at TIMESTAMP,
    duration_ms INT,
    tokens_per_second DECIMAL(10, 2),

    -- 成本估算
    estimated_cost DECIMAL(10, 6),

    -- 元数据
    metadata JSONB,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_provider_responses_multi_response_id ON provider_responses(multi_response_id);
CREATE INDEX idx_provider_responses_message_id ON provider_responses(message_id);
CREATE INDEX idx_provider_responses_provider ON provider_responses(provider);
CREATE INDEX idx_provider_responses_status ON provider_responses(status);
CREATE INDEX idx_provider_responses_started_at ON provider_responses(started_at DESC);

COMMENT ON TABLE provider_responses IS '单个服务商的响应详情，用于性能分析和对比';
COMMENT ON COLUMN provider_responses.first_token_at IS 'Time To First Token (TTFT)';
COMMENT ON COLUMN provider_responses.tokens_per_second IS '吞吐率（tokens/秒）';

-- 服务商性能统计视图
CREATE OR REPLACE VIEW provider_performance_stats AS
SELECT
    provider,
    model,
    COUNT(*) as total_requests,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as successful_requests,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_requests,
    ROUND(AVG(duration_ms), 2) as avg_duration_ms,
    ROUND(AVG(EXTRACT(EPOCH FROM (first_token_at - started_at)) * 1000), 2) as avg_ttft_ms,
    ROUND(AVG(tokens_per_second), 2) as avg_tokens_per_second,
    ROUND(AVG(output_tokens), 2) as avg_output_tokens,
    ROUND(SUM(estimated_cost), 4) as total_cost,
    MAX(completed_at) as last_used_at
FROM provider_responses
WHERE started_at > NOW() - INTERVAL '30 days'
GROUP BY provider, model
ORDER BY total_requests DESC;

COMMENT ON VIEW provider_performance_stats IS '服务商性能统计（最近 30 天）';

-- 用户使用统计视图
CREATE OR REPLACE VIEW user_provider_usage AS
SELECT
    mpr.user_id,
    pr.provider,
    pr.model,
    COUNT(*) as request_count,
    SUM(pr.total_tokens) as total_tokens,
    SUM(pr.estimated_cost) as total_cost,
    MAX(pr.completed_at) as last_used_at
FROM multi_provider_responses mpr
JOIN provider_responses pr ON pr.multi_response_id = mpr.id
WHERE pr.status = 'completed'
  AND pr.completed_at > NOW() - INTERVAL '30 days'
GROUP BY mpr.user_id, pr.provider, pr.model
ORDER BY request_count DESC;

COMMENT ON VIEW user_provider_usage IS '用户服务商使用统计（最近 30 天）';

-- 创建自动更新 updated_at 的触发器
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_multi_provider_responses_updated_at BEFORE UPDATE ON multi_provider_responses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_provider_responses_updated_at BEFORE UPDATE ON provider_responses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
