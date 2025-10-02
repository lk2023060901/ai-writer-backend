package reranker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	kbtypes "github.com/lk2023060901/ai-writer-backend/internal/knowledge/types"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"go.uber.org/zap"
)

// JinaReranker Jina AI Reranker 实现
type JinaReranker struct {
	apiKey  string
	baseURL string
	model   string
	logger  *logger.Logger
	client  *http.Client
}

// JinaRerankerConfig Jina Reranker 配置
type JinaRerankerConfig struct {
	APIKey  string
	BaseURL string // 默认 https://api.jina.ai/v1
	Model   string // 默认 jina-reranker-v2-base-multilingual
}

// NewJinaReranker 创建 Jina Reranker
func NewJinaReranker(cfg *JinaRerankerConfig, lgr *logger.Logger) (*JinaReranker, error) {
	if cfg == nil || cfg.APIKey == "" {
		return nil, fmt.Errorf("api key is required")
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.jina.ai/v1"
	}

	if cfg.Model == "" {
		cfg.Model = "jina-reranker-v2-base-multilingual"
	}

	if lgr == nil {
		lgr = logger.L()
	}

	return &JinaReranker{
		apiKey:  cfg.APIKey,
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		logger:  lgr,
		client:  &http.Client{},
	}, nil
}

// jinaRerankRequest Jina API 请求体
type jinaRerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n"`
}

// jinaRerankResponse Jina API 响应体
type jinaRerankResponse struct {
	Model   string             `json:"model"`
	Results []jinaRerankResult `json:"results"`
}

// jinaRerankResult Jina 重排序结果
type jinaRerankResult struct {
	Index          int     `json:"index"`
	Document       string  `json:"document,omitempty"`
	RelevanceScore float32 `json:"relevance_score"`
}

// Rerank 对搜索结果重排序
func (r *JinaReranker) Rerank(ctx context.Context, query string, results []*kbtypes.ChunkWithScore) ([]*kbtypes.ChunkWithScore, error) {
	if len(results) == 0 {
		return results, nil
	}

	// 构建文档列表
	documents := make([]string, len(results))
	for i, result := range results {
		documents[i] = result.Content
	}

	// 构建请求
	reqBody := jinaRerankRequest{
		Model:     r.model,
		Query:     query,
		Documents: documents,
		TopN:      len(documents),
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送请求
	url := fmt.Sprintf("%s/rerank", r.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.apiKey))

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("rerank API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var respBody jinaRerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 构建索引到分数的映射
	scoreMap := make(map[int]float32)
	for _, result := range respBody.Results {
		scoreMap[result.Index] = result.RelevanceScore
	}

	// 更新结果的分数并标记为已重排序
	reranked := make([]*kbtypes.ChunkWithScore, 0, len(results))
	for i, result := range results {
		if score, ok := scoreMap[i]; ok {
			newResult := *result
			newResult.Score = score
			newResult.Reranked = true
			reranked = append(reranked, &newResult)
		}
	}

	// 按分数降序排序
	sort.Slice(reranked, func(i, j int) bool {
		return reranked[i].Score > reranked[j].Score
	})

	r.logger.Info("reranked search results",
		zap.String("provider", "jina"),
		zap.String("model", r.model),
		zap.Int("original_count", len(results)),
		zap.Int("reranked_count", len(reranked)))

	return reranked, nil
}
