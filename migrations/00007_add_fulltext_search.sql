-- ============================================
-- 添加全文搜索支持
-- ============================================

-- 1. 添加 tsvector 列存储全文搜索向量
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS content_tsv tsvector;

-- 2. 更新现有数据的 tsvector
-- 使用 'simple' 配置，支持中英文混合搜索
UPDATE chunks SET content_tsv = to_tsvector('simple', content);

-- 3. 创建 GIN 索引（用于全文搜索性能优化）
CREATE INDEX IF NOT EXISTS idx_chunks_content_tsv
ON chunks USING GIN(content_tsv);

-- 4. 创建触发器，自动更新 tsvector（当 content 改变时）
CREATE OR REPLACE FUNCTION chunks_content_tsv_trigger() RETURNS trigger AS $$
BEGIN
  NEW.content_tsv := to_tsvector('simple', COALESCE(NEW.content, ''));
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS tsvector_update ON chunks;
CREATE TRIGGER tsvector_update
BEFORE INSERT OR UPDATE ON chunks
FOR EACH ROW EXECUTE FUNCTION chunks_content_tsv_trigger();

-- 5. 添加注释
COMMENT ON COLUMN chunks.content_tsv IS '全文搜索向量（自动维护）';
COMMENT ON INDEX idx_chunks_content_tsv IS 'GIN 索引，用于全文搜索性能优化';
