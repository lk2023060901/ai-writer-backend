-- +goose Up
-- 创建基于内容寻址的文件存储表
-- Migration: 00005_create_file_storage_table
-- Date: 2025-01-08

-- 文件物理存储表（去重存储）
CREATE TABLE IF NOT EXISTS file_storage (
    file_hash VARCHAR(64) PRIMARY KEY,           -- SHA256 文件哈希
    bucket VARCHAR(100) NOT NULL,                -- MinIO bucket
    object_key VARCHAR(500) NOT NULL,            -- MinIO object key (基于 hash 的路径)
    file_size BIGINT NOT NULL,                   -- 文件大小（字节）
    content_type VARCHAR(100),                   -- MIME 类型
    reference_count INT NOT NULL DEFAULT 1,      -- 引用计数
    first_uploaded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_referenced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_file_storage_ref_count ON file_storage(reference_count);
CREATE INDEX idx_file_storage_size ON file_storage(file_size);

-- 注释
COMMENT ON TABLE file_storage IS '文件物理存储表，基于内容寻址（content-addressable storage），实现文件去重';
COMMENT ON COLUMN file_storage.file_hash IS 'SHA256 文件哈希，作为主键和存储路径依据';
COMMENT ON COLUMN file_storage.reference_count IS '引用计数，当为 0 时可以删除物理文件';
COMMENT ON COLUMN file_storage.object_key IS 'MinIO 对象键，格式: files/{hash[:2]}/{hash}';

-- 为 documents 表添加外键引用
ALTER TABLE documents
ADD COLUMN file_storage_hash VARCHAR(64);

-- 添加外键约束（允许 NULL，兼容旧数据）
ALTER TABLE documents
ADD CONSTRAINT fk_doc_file_storage
FOREIGN KEY (file_storage_hash)
REFERENCES file_storage(file_hash)
ON DELETE SET NULL;

-- 添加索引
CREATE INDEX idx_doc_file_storage_hash ON documents(file_storage_hash);

-- 注释
COMMENT ON COLUMN documents.file_storage_hash IS '引用 file_storage 表的文件哈希，实现去重存储';

-- +goose Down
-- 移除 documents 表的外键和字段
ALTER TABLE documents DROP CONSTRAINT IF EXISTS fk_doc_file_storage;
DROP INDEX IF EXISTS idx_doc_file_storage_hash;
ALTER TABLE documents DROP COLUMN IF EXISTS file_storage_hash;

-- 删除 file_storage 表
DROP INDEX IF EXISTS idx_file_storage_ref_count;
DROP INDEX IF EXISTS idx_file_storage_size;
DROP TABLE IF EXISTS file_storage;
