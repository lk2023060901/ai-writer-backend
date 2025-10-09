-- 00006_enhance_knowledge_base.sql
-- 增强知识库功能：添加灵活配置、多模态支持、混合检索

-- ============================================================
-- Part 1: 知识库表增强（灵活配置、混合检索）
-- ============================================================

-- 添加知识库配置字段
ALTER TABLE knowledge_bases
ADD COLUMN IF NOT EXISTS threshold DOUBLE PRECISION DEFAULT 0.0,
ADD COLUMN IF NOT EXISTS top_k INTEGER DEFAULT 5,
ADD COLUMN IF NOT EXISTS enable_hybrid_search BOOLEAN DEFAULT false;

COMMENT ON COLUMN knowledge_bases.threshold IS '相似度阈值（0.0-1.0），用于过滤低相关性结果。值越高过滤越严格，建议：0.0=不过滤，0.3=宽松，0.5=适中，0.7=严格';
COMMENT ON COLUMN knowledge_bases.top_k IS '返回的文档数量（TopK）';
COMMENT ON COLUMN knowledge_bases.enable_hybrid_search IS '是否启用混合检索（向量+关键词）';

-- 添加约束：阈值范围 0.0-1.0
ALTER TABLE knowledge_bases
ADD CONSTRAINT check_threshold_range CHECK (threshold >= 0.0 AND threshold <= 1.0);

-- 添加约束：top_k 范围 1-20
ALTER TABLE knowledge_bases
ADD CONSTRAINT check_top_k_range CHECK (top_k >= 1 AND top_k <= 20);

-- ============================================================
-- Part 2: 文档表增强（多模态支持）
-- ============================================================

-- 添加多模态字段
ALTER TABLE documents
ADD COLUMN IF NOT EXISTS source_type VARCHAR(20) DEFAULT 'file',
ADD COLUMN IF NOT EXISTS source_url TEXT,
ADD COLUMN IF NOT EXISTS source_content TEXT;

COMMENT ON COLUMN documents.source_type IS '来源类型：file=文件上传, url=URL爬取, text=纯文本';
COMMENT ON COLUMN documents.source_url IS 'URL 来源地址（仅 source_type=url 时有值）';
COMMENT ON COLUMN documents.source_content IS '纯文本内容（仅 source_type=text 时有值）';

-- 添加约束：source_type 只能是 file, url, text
ALTER TABLE documents
ADD CONSTRAINT check_source_type CHECK (source_type IN ('file', 'url', 'text'));

-- 添加索引：按来源类型查询
CREATE INDEX IF NOT EXISTS idx_documents_source_type ON documents(source_type);

-- ============================================================
-- Part 3: 更新现有数据的默认值
-- ============================================================

-- 将现有知识库的 threshold 设置为 0.0（不过滤）
UPDATE knowledge_bases
SET threshold = 0.0,
    top_k = 5,
    enable_hybrid_search = false
WHERE threshold IS NULL;

-- 将现有文档的 source_type 设置为 'file'
UPDATE documents
SET source_type = 'file'
WHERE source_type IS NULL;
