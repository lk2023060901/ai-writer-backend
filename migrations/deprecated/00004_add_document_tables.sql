-- +goose Up
-- +goose StatementBegin

-- 添加文档表
CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY,
    knowledge_base_id UUID NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(50) NOT NULL,
    file_size BIGINT NOT NULL,
    file_path VARCHAR(1024) NOT NULL,
    process_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    process_error TEXT,
    chunk_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_documents_kb_id ON documents(knowledge_base_id);
CREATE INDEX idx_documents_status ON documents(process_status);

-- 添加外键约束
ALTER TABLE documents
ADD CONSTRAINT fk_documents_kb
FOREIGN KEY (knowledge_base_id)
REFERENCES knowledge_bases(id)
ON DELETE CASCADE;

-- 添加文档分块表
CREATE TABLE IF NOT EXISTS chunks (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL,
    knowledge_base_id UUID NOT NULL,
    content TEXT NOT NULL,
    position INTEGER NOT NULL,
    token_count INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_chunks_document_id ON chunks(document_id);
CREATE INDEX idx_chunks_kb_id ON chunks(knowledge_base_id);

-- 添加外键约束
ALTER TABLE chunks
ADD CONSTRAINT fk_chunks_document
FOREIGN KEY (document_id)
REFERENCES documents(id)
ON DELETE CASCADE;

ALTER TABLE chunks
ADD CONSTRAINT fk_chunks_kb
FOREIGN KEY (knowledge_base_id)
REFERENCES knowledge_bases(id)
ON DELETE CASCADE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

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

-- +goose StatementEnd
