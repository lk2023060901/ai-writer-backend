-- 00009_fix_threshold_precision.sql
-- 修复 threshold 字段精度问题：将 DOUBLE PRECISION 改为 REAL (float32)

-- +goose Up
-- +goose StatementBegin

-- 修改 threshold 字段类型为 REAL (float32)
ALTER TABLE knowledge_bases
ALTER COLUMN threshold TYPE REAL;

COMMENT ON COLUMN knowledge_bases.threshold IS '相似度阈值（0.0-1.0），用于过滤低相关性结果。值越高过滤越严格，建议：0.0=不过滤，0.3=宽松，0.5=适中，0.7=严格（REAL = float32）';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- 回滚：改回 DOUBLE PRECISION (float64)
ALTER TABLE knowledge_bases
ALTER COLUMN threshold TYPE DOUBLE PRECISION;

COMMENT ON COLUMN knowledge_bases.threshold IS '相似度阈值（0.0-1.0），低于此值的结果将被过滤';

-- +goose StatementEnd
