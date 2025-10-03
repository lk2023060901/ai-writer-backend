-- 文档和分块模块
-- 包含：文档表、分块表（文档处理和向量存储）

-- 文档表
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_base_id UUID NOT NULL,

    -- 文件信息
    filename VARCHAR(255) NOT NULL,
    file_type VARCHAR(50) NOT NULL,
    file_size BIGINT NOT NULL,
    file_hash VARCHAR(64) NOT NULL,

    -- MinIO 存储路径
    minio_bucket VARCHAR(100) NOT NULL,
    minio_object_key VARCHAR(500) NOT NULL,

    -- 处理状态
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error_message TEXT,

    -- 统计信息
    chunk_count INTEGER NOT NULL DEFAULT 0,
    token_count INTEGER NOT NULL DEFAULT 0,

    -- 元数据
    metadata JSONB,

    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- 约束
    CONSTRAINT fk_doc_kb FOREIGN KEY (knowledge_base_id)
        REFERENCES knowledge_bases(id) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_doc_kb_id ON documents(knowledge_base_id);
CREATE INDEX idx_doc_file_type ON documents(file_type);
CREATE INDEX idx_doc_file_hash ON documents(file_hash);
CREATE INDEX idx_doc_status ON documents(status);
CREATE INDEX idx_doc_kb_status ON documents(knowledge_base_id, status);
CREATE INDEX idx_doc_kb_created ON documents(knowledge_base_id, created_at DESC);

-- 注释
COMMENT ON TABLE documents IS '文档表：存储上传的文档信息和处理状态';
COMMENT ON COLUMN documents.status IS '处理状态：pending、processing、completed、failed';
COMMENT ON COLUMN documents.file_hash IS 'SHA256 哈希值，用于去重';
COMMENT ON COLUMN documents.minio_object_key IS 'MinIO 对象键，格式：knowledge_bases/{kb_id}/{uuid}.{ext}';

-- 分块表
CREATE TABLE chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL,
    knowledge_base_id UUID NOT NULL,

    -- 分块信息
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL,
    token_count INTEGER NOT NULL,

    -- Milvus 向量 ID
    milvus_id VARCHAR(100) NOT NULL UNIQUE,

    -- 元数据
    metadata JSONB,

    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- 约束
    CONSTRAINT fk_chunk_doc FOREIGN KEY (document_id)
        REFERENCES documents(id) ON DELETE CASCADE,
    CONSTRAINT fk_chunk_kb FOREIGN KEY (knowledge_base_id)
        REFERENCES knowledge_bases(id) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_chunk_doc_id ON chunks(document_id);
CREATE INDEX idx_chunk_kb_id ON chunks(knowledge_base_id);
CREATE UNIQUE INDEX idx_chunk_milvus_id ON chunks(milvus_id);
CREATE INDEX idx_chunk_doc_index ON chunks(document_id, chunk_index);

-- 注释
COMMENT ON TABLE chunks IS '分块表：文档分块后的文本片段，与 Milvus 向量一一对应';
COMMENT ON COLUMN chunks.chunk_index IS '分块在文档中的序号，从 0 开始';
COMMENT ON COLUMN chunks.milvus_id IS 'Milvus 向量 ID，格式：{doc_id}_{chunk_index}';
