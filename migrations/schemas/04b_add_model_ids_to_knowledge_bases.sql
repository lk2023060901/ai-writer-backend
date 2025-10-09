-- 添加 embedding_model_id 和 rerank_model_id 字段到 knowledge_bases 表
-- 替换原有的 ai_provider_type 字段

ALTER TABLE knowledge_bases
    ADD COLUMN IF NOT EXISTS embedding_model_id UUID,
    ADD COLUMN IF NOT EXISTS rerank_model_id UUID;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_kb_embedding_model ON knowledge_bases(embedding_model_id);
CREATE INDEX IF NOT EXISTS idx_kb_rerank_model ON knowledge_bases(rerank_model_id);

-- 如果 ai_provider_type 字段存在，可以选择删除（如果需要保留历史兼容性，可以注释掉此行）
-- ALTER TABLE knowledge_bases DROP COLUMN IF EXISTS ai_provider_type;
-- DROP INDEX IF EXISTS idx_kb_provider_type;

-- 注释
COMMENT ON COLUMN knowledge_bases.embedding_model_id IS 'Embedding 模型 ID（关联 ai_models 表）';
COMMENT ON COLUMN knowledge_bases.rerank_model_id IS 'Rerank 模型 ID（关联 ai_models 表，可选）';
