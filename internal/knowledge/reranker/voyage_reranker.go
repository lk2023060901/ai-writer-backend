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

// VoyageReranker Voyage AI Reranker 实现
type VoyageReranker struct {
	apiKey  string
	baseURL string
	model   string
	logger  *logger.Logger
	client  *http.Client
}

// VoyageRerankerConfig Voyage Reranker 配置
type VoyageRerankerConfig struct {
	APIKey  string
	BaseURL string // 默认 https://api.voyageai.com/v1
	Model   string // 默认 rerank-1
}

// NewVoyageReranker 创建 Voyage Reranker
func NewVoyageReranker(cfg *VoyageRerankerConfig, lgr *logger.Logger) (*VoyageReranker, error) {
	if cfg == nil || cfg.APIKey == "" {
		return nil, fmt.Errorf("api key is required")
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.voyageai.com/v1"
	}

	if cfg.Model == "" {
		cfg.Model = "rerank-1"
	}

	if lgr == nil {
		lgr = logger.L()
	}

	return &VoyageReranker{
		apiKey:  cfg.APIKey,
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		logger:  lgr,
		client:  &http.Client{},
	}, nil
}

// voyageRerankRequest Voyage API 请求体
type voyageRerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopK      int      `json:"top_k"`
}

// voyageRerankResponse Voyage API 响应体
type voyageRerankResponse struct {
	Data []voyageRerankResult `json:"data"`
}

// voyageRerankResult Voyage 重排序结果
type voyageRerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float32 `json:"relevance_score"`
}

// Rerank 对搜索结果重排序
func (r *VoyageReranker) Rerank(ctx context.Context, query string, results []*kbtypes.ChunkWithScore) ([]*kbtypes.ChunkWithScore, error) {
	if len(results) == 0 {
		return results, nil
	}

	// 构建文档列表
	documents := make([]string, len(results))
	for i, result := range results {
		documents[i] = result.Content
	}

	// 构建请求
	reqBody := voyageRerankRequest{
		Model:     r.model,
		Query:     query,
		Documents: documents,
		TopK:      len(documents),
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
	var respBody voyageRerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 构建索引到分数的映射
	scoreMap := make(map[int]float32)
	for _, result := range respBody.Data {
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
		zap.String("provider", "voyage"),
		zap.String("model", r.model),
		zap.Int("original_count", len(results)),
		zap.Int("reranked_count", len(reranked)))

	return reranked, nil
}
