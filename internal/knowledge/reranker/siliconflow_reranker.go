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

// SiliconFlowReranker SiliconFlow Reranker implementation
type SiliconFlowReranker struct {
	apiKey  string
	baseURL string
	model   string
	logger  *logger.Logger
	client  *http.Client
}

// SiliconFlowRerankerConfig SiliconFlow Reranker configuration
type SiliconFlowRerankerConfig struct {
	APIKey  string
	BaseURL string // e.g. https://api.siliconflow.cn/v1
	Model   string // e.g. BAAI/bge-reranker-v2-m3
}

// NewSiliconFlowReranker creates a new SiliconFlow Reranker
func NewSiliconFlowReranker(cfg *SiliconFlowRerankerConfig, lgr *logger.Logger) (*SiliconFlowReranker, error) {
	if cfg == nil || cfg.APIKey == "" {
		return nil, fmt.Errorf("api key is required")
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.siliconflow.cn/v1"
	}

	if cfg.Model == "" {
		cfg.Model = "BAAI/bge-reranker-v2-m3"
	}

	if lgr == nil {
		lgr = logger.L()
	}

	return &SiliconFlowReranker{
		apiKey:  cfg.APIKey,
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		logger:  lgr,
		client:  &http.Client{},
	}, nil
}

// siliconFlowRerankRequest SiliconFlow API request
type siliconFlowRerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n"`
}

// siliconFlowRerankResponse SiliconFlow API response
type siliconFlowRerankResponse struct {
	Model   string                      `json:"model"`
	Results []siliconFlowRerankResult `json:"results"`
}

// siliconFlowRerankResult SiliconFlow rerank result
type siliconFlowRerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float32 `json:"relevance_score"`
}

// Rerank reranks search results
func (r *SiliconFlowReranker) Rerank(ctx context.Context, query string, results []*kbtypes.ChunkWithScore) ([]*kbtypes.ChunkWithScore, error) {
	if len(results) == 0 {
		return results, nil
	}

	// Build documents list
	documents := make([]string, len(results))
	for i, result := range results {
		documents[i] = result.Content
	}

	// Build request
	reqBody := siliconFlowRerankRequest{
		Model:     r.model,
		Query:     query,
		Documents: documents,
		TopN:      len(documents),
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request
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

	// Parse response
	var respBody siliconFlowRerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Build score map
	scoreMap := make(map[int]float32)
	for _, result := range respBody.Results {
		scoreMap[result.Index] = result.RelevanceScore
	}

	// Create reranked results with new scores
	reranked := make([]*kbtypes.ChunkWithScore, 0, len(results))
	for i, result := range results {
		if score, ok := scoreMap[i]; ok {
			newResult := *result
			newResult.Score = score
			newResult.Reranked = true
			reranked = append(reranked, &newResult)
		}
	}

	// Sort by score descending
	sort.Slice(reranked, func(i, j int) bool {
		return reranked[i].Score > reranked[j].Score
	})

	r.logger.Info("reranked search results",
		zap.String("provider", "siliconflow"),
		zap.String("model", r.model),
		zap.Int("original_count", len(results)),
		zap.Int("reranked_count", len(reranked)))

	return reranked, nil
}
