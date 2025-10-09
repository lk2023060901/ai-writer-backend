-- +goose Up
-- 添加文档计数约束，防止负数
-- Migration: 00004_add_document_count_constraint
-- Date: 2025-01-08

-- 首先修复所有负数
UPDATE knowledge_bases SET document_count = 0 WHERE document_count < 0;

-- 添加检查约束
ALTER TABLE knowledge_bases
ADD CONSTRAINT check_document_count_non_negative
CHECK (document_count >= 0);

-- 注释说明
COMMENT ON CONSTRAINT check_document_count_non_negative ON knowledge_bases IS
'确保 document_count 不会为负数，防止批量删除时计数错误';

-- +goose Down
-- 移除约束
ALTER TABLE knowledge_bases DROP CONSTRAINT IF EXISTS check_document_count_non_negative;
