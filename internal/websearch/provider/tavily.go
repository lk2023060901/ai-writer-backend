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

// TavilyProvider implements the Tavily search API
type TavilyProvider struct {
	*BaseProvider
}

// NewTavilyProvider creates a new Tavily provider
func NewTavilyProvider(config *types.ProviderConfig) (Provider, error) {
	base := NewBaseProvider(config)
	return &TavilyProvider{BaseProvider: base}, nil
}

// tavilyRequest represents a Tavily API request
type tavilyRequest struct {
	Query             string   `json:"query"`
	SearchDepth       string   `json:"search_depth,omitempty"`
	MaxResults        int      `json:"max_results,omitempty"`
	IncludeDomains    []string `json:"include_domains,omitempty"`
	ExcludeDomains    []string `json:"exclude_domains,omitempty"`
	IncludeAnswer     bool     `json:"include_answer,omitempty"`
	IncludeRawContent bool     `json:"include_raw_content,omitempty"`
}

// tavilyResponse represents a Tavily API response
type tavilyResponse struct {
	Results []struct {
		Title         string  `json:"title"`
		URL           string  `json:"url"`
		Content       string  `json:"content"`
		Score         float32 `json:"score"`
		PublishedDate string  `json:"published_date,omitempty"`
	} `json:"results"`
	Query string `json:"query"`
}

// Search executes a search query using the Tavily API
func (p *TavilyProvider) Search(ctx context.Context, req *types.SearchRequest) (*types.SearchResponse, error) {
	startTime := time.Now()

	// Build request body
	tavilyReq := tavilyRequest{
		Query:             req.Query,
		SearchDepth:       req.SearchDepth,
		MaxResults:        req.MaxResults,
		IncludeDomains:    req.IncludeDomains,
		ExcludeDomains:    req.ExcludeDomains,
		IncludeAnswer:     true,
		IncludeRawContent: false,
	}

	if tavilyReq.MaxResults == 0 {
		tavilyReq.MaxResults = 10
	}

	if tavilyReq.SearchDepth == "" {
		tavilyReq.SearchDepth = "basic"
	}

	reqBody, err := json.Marshal(tavilyReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build HTTP request
	apiURL := fmt.Sprintf("%s/search", p.config.APIHost)
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
	var tavilyResp tavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&tavilyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to standard response
	results := make([]*types.SearchResult, len(tavilyResp.Results))
	for i, r := range tavilyResp.Results {
		results[i] = &types.SearchResult{
			Title:       r.Title,
			URL:         r.URL,
			Content:     r.Content,
			Score:       r.Score,
			PublishedAt: r.PublishedDate,
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
