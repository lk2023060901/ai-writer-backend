package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/websearch/types"
)

// SearXNGProvider implements the SearXNG search API
type SearXNGProvider struct {
	*BaseProvider
}

// NewSearXNGProvider creates a new SearXNG provider
func NewSearXNGProvider(config *types.ProviderConfig) (Provider, error) {
	base := NewBaseProvider(config)
	return &SearXNGProvider{BaseProvider: base}, nil
}

// searxngResponse represents a SearXNG API response
type searxngResponse struct {
	Results []struct {
		Title         string `json:"title"`
		URL           string `json:"url"`
		Content       string `json:"content"`
		PublishedDate string `json:"publishedDate,omitempty"`
	} `json:"results"`
	Query string `json:"query"`
}

// Search executes a search query using the SearXNG API
func (p *SearXNGProvider) Search(ctx context.Context, req *types.SearchRequest) (*types.SearchResponse, error) {
	startTime := time.Now()

	// Build query parameters
	params := url.Values{}
	params.Set("q", req.Query)
	params.Set("format", "json")

	if req.MaxResults > 0 {
		params.Set("pageno", "1")
		params.Set("number_of_results", fmt.Sprintf("%d", req.MaxResults))
	}

	// Build request URL
	apiURL := fmt.Sprintf("%s/search?%s", p.config.APIHost, params.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range p.BuildDefaultHeaders() {
		httpReq.Header.Set(k, v)
	}

	// Basic Auth (if configured)
	if p.config.BasicAuthUsername != "" && p.config.BasicAuthPassword != "" {
		httpReq.SetBasicAuth(p.config.BasicAuthUsername, p.config.BasicAuthPassword)
	}

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
	var searxngResp searxngResponse
	if err := json.NewDecoder(resp.Body).Decode(&searxngResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to standard response
	results := make([]*types.SearchResult, len(searxngResp.Results))
	for i, r := range searxngResp.Results {
		results[i] = &types.SearchResult{
			Title:       r.Title,
			URL:         r.URL,
			Content:     r.Content,
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
