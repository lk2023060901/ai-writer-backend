-- 对话模块
-- 包含：对话主题表、消息表（用户与智能体的对话记录）

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
