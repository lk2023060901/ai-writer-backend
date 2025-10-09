-- 智能体快捷访问列表
-- 用于存储用户收藏的智能体，支持快捷访问

CREATE TABLE assistant_favorites (
    -- 主键 (UUID v7, 由应用层生成)
    id UUID PRIMARY KEY,

    -- 用户 ID
    user_id UUID NOT NULL,

    -- 智能体 ID
    assistant_id UUID NOT NULL,

    -- 排序顺序（数字越小越靠前，支持用户自定义排序）
    sort_order INTEGER NOT NULL DEFAULT 0,

    -- 添加时间
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- 唯一约束：同一个用户不能重复添加同一个智能体
    CONSTRAINT uk_user_assistant UNIQUE (user_id, assistant_id)
);

-- 索引：根据 user_id 查询该用户的快捷访问列表（按排序顺序）
CREATE INDEX idx_favorites_user_id ON assistant_favorites(user_id, sort_order ASC);

-- 索引：根据 assistant_id 查询有多少用户收藏了该智能体
CREATE INDEX idx_favorites_assistant_id ON assistant_favorites(assistant_id);

-- 索引：按添加时间查询
CREATE INDEX idx_favorites_created_at ON assistant_favorites(created_at DESC);

-- 注释
COMMENT ON TABLE assistant_favorites IS '智能体快捷访问列表：用户收藏的智能体';
COMMENT ON COLUMN assistant_favorites.user_id IS '用户 ID（外键到 users 表）';
COMMENT ON COLUMN assistant_favorites.assistant_id IS '智能体 ID（可以是 agents.id 或 official_agents.id）';
COMMENT ON COLUMN assistant_favorites.sort_order IS '排序顺序，数字越小越靠前，支持用户自定义排序';
COMMENT ON COLUMN assistant_favorites.created_at IS '添加到快捷访问的时间';
