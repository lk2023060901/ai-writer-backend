package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/websearch/types"
)

// BochaProvider implements the Bocha AI search API
type BochaProvider struct {
	*BaseProvider
}

// NewBochaProvider creates a new Bocha provider
func NewBochaProvider(config *types.ProviderConfig) (Provider, error) {
	base := NewBaseProvider(config)
	return &BochaProvider{BaseProvider: base}, nil
}

// bochaRequest represents a Bocha API request
type bochaRequest struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results,omitempty"`
	SearchType string `json:"search_type,omitempty"` // "web", "news", "academic"
}

// bochaResponse represents a Bocha API response
type bochaResponse struct {
	Results []struct {
		Title       string  `json:"title"`
		URL         string  `json:"url"`
		Snippet     string  `json:"snippet"`
		Content     string  `json:"content,omitempty"`
		Score       float32 `json:"score,omitempty"`
		PublishedAt string  `json:"published_at,omitempty"`
	} `json:"results"`
	Query      string `json:"query"`
	TotalCount int    `json:"total_count,omitempty"`
}

// Search executes a search query using the Bocha API
func (p *BochaProvider) Search(ctx context.Context, req *types.SearchRequest) (*types.SearchResponse, error) {
	startTime := time.Now()

	// Build request body
	bochaReq := bochaRequest{
		Query:      req.Query,
		MaxResults: req.MaxResults,
		SearchType: "web",
	}

	if bochaReq.MaxResults == 0 {
		bochaReq.MaxResults = 10
	}

	reqBody, err := json.Marshal(bochaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build HTTP request
	apiURL := fmt.Sprintf("%s/v1/search", p.config.APIHost)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range p.BuildDefaultHeaders() {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.GetAPIKey()))

	// Execute request
	resp, err := p.DoRequest(ctx, httpReq)
	if err != nil {
		return nil, &types.ProviderError{
			Provider: p.GetID(),
			Code:     "REQUEST_FAILED",
			Message:  "Failed to execute request",
			Err:      err,
		}
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &types.ProviderError{
			Provider: p.GetID(),
			Code:     fmt.Sprintf("HTTP_%d", resp.StatusCode),
			Message:  string(body),
		}
	}

	// Parse response
	var bochaResp bochaResponse
	if err := json.NewDecoder(resp.Body).Decode(&bochaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to standard response
	results := make([]*types.SearchResult, len(bochaResp.Results))
	for i, r := range bochaResp.Results {
		content := r.Content
		if content == "" {
			content = r.Snippet
		}

		results[i] = &types.SearchResult{
			Title:       r.Title,
			URL:         r.URL,
			Content:     content,
			Score:       r.Score,
			PublishedAt: r.PublishedAt,
		}
	}

	return &types.SearchResponse{
		Query:      req.Query,
		Results:    results,
		TotalCount: len(results),
		Took:       time.Since(startTime).Milliseconds(),
		Provider:   p.GetID(),
	}, nil
}
