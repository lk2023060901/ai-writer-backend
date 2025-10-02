-- 删除外键约束
ALTER TABLE chunks DROP CONSTRAINT IF EXISTS fk_chunks_kb;
ALTER TABLE chunks DROP CONSTRAINT IF EXISTS fk_chunks_document;
ALTER TABLE documents DROP CONSTRAINT IF EXISTS fk_documents_kb;

-- 删除索引
DROP INDEX IF EXISTS idx_chunks_kb_id;
DROP INDEX IF EXISTS idx_chunks_document_id;
DROP INDEX IF EXISTS idx_documents_status;
DROP INDEX IF EXISTS idx_documents_kb_id;

-- 删除表
DROP TABLE IF EXISTS chunks;
DROP TABLE IF EXISTS documents;
