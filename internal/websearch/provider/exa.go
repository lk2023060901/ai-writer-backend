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

// ExaProvider implements the Exa AI search API
type ExaProvider struct {
	*BaseProvider
}

// NewExaProvider creates a new Exa provider
func NewExaProvider(config *types.ProviderConfig) (Provider, error) {
	base := NewBaseProvider(config)
	return &ExaProvider{BaseProvider: base}, nil
}

// exaRequest represents an Exa API request
type exaRequest struct {
	Query          string   `json:"query"`
	NumResults     int      `json:"numResults,omitempty"`
	IncludeDomains []string `json:"includeDomains,omitempty"`
	ExcludeDomains []string `json:"excludeDomains,omitempty"`
	StartPublishedDate string `json:"startPublishedDate,omitempty"`
	EndPublishedDate   string `json:"endPublishedDate,omitempty"`
	UseAutoprompt  bool     `json:"useAutoprompt,omitempty"`
	Type           string   `json:"type,omitempty"` // "neural", "keyword", or "auto"
	Contents       map[string]interface{} `json:"contents,omitempty"`
}

// exaResponse represents an Exa API response
type exaResponse struct {
	Results []struct {
		Title         string  `json:"title"`
		URL           string  `json:"url"`
		Text          string  `json:"text,omitempty"`
		Highlights    []string `json:"highlights,omitempty"`
		Score         float32 `json:"score"`
		PublishedDate string  `json:"publishedDate,omitempty"`
		Author        string  `json:"author,omitempty"`
	} `json:"results"`
	Autoprompt string `json:"autopromptString,omitempty"`
}

// Search executes a search query using the Exa API
func (p *ExaProvider) Search(ctx context.Context, req *types.SearchRequest) (*types.SearchResponse, error) {
	startTime := time.Now()

	// Build request body
	exaReq := exaRequest{
		Query:          req.Query,
		NumResults:     req.MaxResults,
		IncludeDomains: req.IncludeDomains,
		ExcludeDomains: req.ExcludeDomains,
		UseAutoprompt:  true,
		Type:           "auto",
		Contents: map[string]interface{}{
			"text": true,
		},
	}

	if exaReq.NumResults == 0 {
		exaReq.NumResults = 10
	}

	// Handle time range
	if req.TimeRange != nil {
		if req.TimeRange.Start != "" {
			exaReq.StartPublishedDate = req.TimeRange.Start
		}
		if req.TimeRange.End != "" {
			exaReq.EndPublishedDate = req.TimeRange.End
		}
	}

	reqBody, err := json.Marshal(exaReq)
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
	httpReq.Header.Set("x-api-key", p.GetAPIKey())

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
	var exaResp exaResponse
	if err := json.NewDecoder(resp.Body).Decode(&exaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to standard response
	results := make([]*types.SearchResult, len(exaResp.Results))
	for i, r := range exaResp.Results {
		content := r.Text
		if len(r.Highlights) > 0 {
			// If highlights are available, use them as content
			content = ""
			for _, h := range r.Highlights {
				content += h + "\n"
			}
		}

		results[i] = &types.SearchResult{
			Title:       r.Title,
			URL:         r.URL,
			Content:     content,
			Score:       r.Score,
			PublishedAt: r.PublishedDate,
			Author:      r.Author,
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
