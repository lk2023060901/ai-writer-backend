package hybrid

import (
	"testing"
)

func TestReciprocalRankFusion(t *testing.T) {
	// 模拟向量搜索结果
	vectorResults := []SearchResult{
		&VectorSearchResult{ID: "doc1", Score: 0.95},
		&VectorSearchResult{ID: "doc2", Score: 0.85},
		&VectorSearchResult{ID: "doc3", Score: 0.75},
	}

	// 模拟关键词搜索结果
	keywordResults := []SearchResult{
		&KeywordSearchResult{ID: "doc2", Score: 0.90},
		&KeywordSearchResult{ID: "doc4", Score: 0.80},
		&KeywordSearchResult{ID: "doc1", Score: 0.70},
	}

	// RRF 融合
	results := ReciprocalRankFusion([][]SearchResult{vectorResults, keywordResults}, 60)

	// 验证结果数量
	if len(results) != 4 {
		t.Errorf("Expected 4 unique results, got %d", len(results))
	}

	// 验证排序（doc1 和 doc2 应该排名最高，因为它们在两个列表中都出现）
	if results[0].ID != "doc1" && results[0].ID != "doc2" {
		t.Errorf("Expected doc1 or doc2 to be first, got %s", results[0].ID)
	}

	// 验证 RRF 分数计算
	// doc1: 1/(60+1) + 1/(60+3) = 0.0164 + 0.0159 = 0.0323
	// doc2: 1/(60+2) + 1/(60+1) = 0.0161 + 0.0164 = 0.0325
	// doc2 应该略高于 doc1
	if results[0].ID != "doc2" {
		t.Errorf("Expected doc2 to be first (highest RRF score), got %s", results[0].ID)
	}

	t.Logf("RRF Results:")
	for _, result := range results {
		t.Logf("  %d. ID=%s, RRFScore=%.6f, OriginalScore=%.2f",
			result.Rank, result.ID, result.RRFScore, result.Score)
	}
}

func TestReciprocalRankFusion_SingleList(t *testing.T) {
	// 测试单个列表的情况
	vectorResults := []SearchResult{
		&VectorSearchResult{ID: "doc1", Score: 0.95},
		&VectorSearchResult{ID: "doc2", Score: 0.85},
	}

	results := ReciprocalRankFusion([][]SearchResult{vectorResults}, 60)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// 验证排序保持不变
	if results[0].ID != "doc1" {
		t.Errorf("Expected doc1 to be first, got %s", results[0].ID)
	}
}

func TestReciprocalRankFusion_EmptyList(t *testing.T) {
	// 测试空列表
	results := ReciprocalRankFusion([][]SearchResult{}, 60)

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty input, got %d", len(results))
	}
}

func TestReciprocalRankFusion_NoOverlap(t *testing.T) {
	// 测试没有重叠的情况
	vectorResults := []SearchResult{
		&VectorSearchResult{ID: "doc1", Score: 0.95},
		&VectorSearchResult{ID: "doc2", Score: 0.85},
	}

	keywordResults := []SearchResult{
		&KeywordSearchResult{ID: "doc3", Score: 0.90},
		&KeywordSearchResult{ID: "doc4", Score: 0.80},
	}

	results := ReciprocalRankFusion([][]SearchResult{vectorResults, keywordResults}, 60)

	if len(results) != 4 {
		t.Errorf("Expected 4 results, got %d", len(results))
	}

	// 所有文档应该有相同的 RRF 贡献（每个只在一个列表中）
	// 但排名会根据它们在各自列表中的位置决定
}

func TestReciprocalRankFusion_CustomK(t *testing.T) {
	vectorResults := []SearchResult{
		&VectorSearchResult{ID: "doc1", Score: 0.95},
	}

	// 测试自定义 k 值
	results := ReciprocalRankFusion([][]SearchResult{vectorResults}, 100)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// 验证 k=100 时的分数: 1/(100+1) ≈ 0.0099
	expectedScore := 1.0 / 101.0
	if results[0].RRFScore < expectedScore-0.0001 || results[0].RRFScore > expectedScore+0.0001 {
		t.Errorf("Expected RRF score ≈ %.6f, got %.6f", expectedScore, results[0].RRFScore)
	}
}

func TestReciprocalRankFusion_DefaultK(t *testing.T) {
	vectorResults := []SearchResult{
		&VectorSearchResult{ID: "doc1", Score: 0.95},
	}

	// 测试 k <= 0 时使用默认值 60
	results := ReciprocalRankFusion([][]SearchResult{vectorResults}, 0)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// 验证使用默认 k=60: 1/(60+1) ≈ 0.0164
	expectedScore := 1.0 / 61.0
	if results[0].RRFScore < expectedScore-0.0001 || results[0].RRFScore > expectedScore+0.0001 {
		t.Errorf("Expected RRF score ≈ %.6f with default k=60, got %.6f", expectedScore, results[0].RRFScore)
	}
}
