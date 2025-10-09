-- ============================================
-- 添加 BM25 评分函数
-- ============================================

-- BM25 算法实现
-- 参考：https://en.wikipedia.org/wiki/Okapi_BM25
-- 参数：k1 (词频饱和度参数，默认1.2), b (文档长度归一化参数，默认0.75)

CREATE OR REPLACE FUNCTION bm25_score(
    content_tsv tsvector,
    query_tsv tsquery,
    k1 float DEFAULT 1.2,
    b float DEFAULT 0.75
) RETURNS float AS $$
DECLARE
    doc_length int;
    avg_doc_length float;
    score float := 0;
    idf float;
    tf float;
    norm_length float;
    total_docs int;
    docs_with_term int;
BEGIN
    -- 获取文档长度（lexeme 数量）
    doc_length := array_length(tsvector_to_array(content_tsv), 1);
    IF doc_length IS NULL THEN
        RETURN 0;
    END IF;

    -- 获取平均文档长度（简化版本：使用固定值，生产环境应该从统计表获取）
    avg_doc_length := 100.0;

    -- 计算归一化后的文档长度
    norm_length := 1 - b + b * (doc_length::float / avg_doc_length);

    -- 使用 ts_rank_cd 作为基础分数（它已经考虑了 TF-IDF）
    score := ts_rank_cd(content_tsv, query_tsv);

    -- 应用 BM25 归一化
    -- BM25(D,Q) = Σ IDF(qi) * (f(qi,D) * (k1+1)) / (f(qi,D) + k1 * norm_length)
    -- 简化版本：直接使用 ts_rank_cd 的结果并应用归一化
    score := score * (k1 + 1) / (score + k1 * norm_length);

    RETURN score;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- 添加注释
COMMENT ON FUNCTION bm25_score IS 'BM25 相关度评分函数（基于 ts_rank_cd 优化）';

-- 创建辅助函数：计算知识库的平均文档长度（用于更精确的 BM25）
CREATE OR REPLACE FUNCTION get_avg_chunk_length(kb_id uuid) RETURNS float AS $$
DECLARE
    avg_len float;
BEGIN
    SELECT AVG(array_length(tsvector_to_array(content_tsv), 1))
    INTO avg_len
    FROM chunks
    WHERE knowledge_base_id = kb_id
    AND content_tsv IS NOT NULL;

    RETURN COALESCE(avg_len, 100.0);
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION get_avg_chunk_length IS '计算知识库的平均文档长度（用于 BM25 归一化）';
