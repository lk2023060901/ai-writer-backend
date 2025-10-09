package hybrid

import (
	"sort"
)

// RRFResult RRF 融合后的结果
type RRFResult struct {
	ID       string
	Score    float32
	RRFScore float64
	Rank     int
}

// SearchResult 搜索结果接口
type SearchResult interface {
	GetID() string
	GetScore() float32
}

// ReciprocalRankFusion RRF 算法实现
// RRF 公式: score = Σ(1 / (k + rank))
// k 是常数，通常为 60（根据论文推荐）
func ReciprocalRankFusion(results [][]SearchResult, k int) []*RRFResult {
	if k <= 0 {
		k = 60 // 默认值
	}

	// 计算每个文档的 RRF 分数
	rrfScores := make(map[string]*RRFResult)

	for _, resultSet := range results {
		for rank, result := range resultSet {
			id := result.GetID()

			// 初始化或累加 RRF 分数
			if _, exists := rrfScores[id]; !exists {
				rrfScores[id] = &RRFResult{
					ID:       id,
					Score:    result.GetScore(),
					RRFScore: 0,
				}
			}

			// RRF 公式: 1 / (k + rank)
			// rank 从 0 开始，所以 rank+1 才是真正的排名
			rrfScores[id].RRFScore += 1.0 / float64(k+rank+1)
		}
	}

	// 转换为切片并按 RRF 分数排序
	fusedResults := make([]*RRFResult, 0, len(rrfScores))
	for _, result := range rrfScores {
		fusedResults = append(fusedResults, result)
	}

	sort.Slice(fusedResults, func(i, j int) bool {
		return fusedResults[i].RRFScore > fusedResults[j].RRFScore
	})

	// 设置最终排名
	for i := range fusedResults {
		fusedResults[i].Rank = i + 1
	}

	return fusedResults
}

// VectorSearchResult 向量搜索结果适配器
type VectorSearchResult struct {
	ID    string
	Score float32
}

func (r *VectorSearchResult) GetID() string {
	return r.ID
}

func (r *VectorSearchResult) GetScore() float32 {
	return r.Score
}

// KeywordSearchResult 关键词搜索结果适配器
type KeywordSearchResult struct {
	ID    string
	Score float32
}

func (r *KeywordSearchResult) GetID() string {
	return r.ID
}

func (r *KeywordSearchResult) GetScore() float32 {
	return r.Score
}
