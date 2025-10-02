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

// ZhipuProvider implements the Zhipu GLM search API
type ZhipuProvider struct {
	*BaseProvider
}

// NewZhipuProvider creates a new Zhipu provider
func NewZhipuProvider(config *types.ProviderConfig) (Provider, error) {
	base := NewBaseProvider(config)
	return &ZhipuProvider{BaseProvider: base}, nil
}

// zhipuRequest represents a Zhipu API request
type zhipuRequest struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results,omitempty"`
}

// zhipuResponse represents a Zhipu API response
type zhipuResponse struct {
	Data struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
			Snippet string `json:"snippet"`
		} `json:"results"`
	} `json:"data"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// Search executes a search query using the Zhipu API
func (p *ZhipuProvider) Search(ctx context.Context, req *types.SearchRequest) (*types.SearchResponse, error) {
	startTime := time.Now()

	// Build request body
	zhipuReq := zhipuRequest{
		Query:      req.Query,
		MaxResults: req.MaxResults,
	}

	if zhipuReq.MaxResults == 0 {
		zhipuReq.MaxResults = 10
	}

	reqBody, err := json.Marshal(zhipuReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.APIHost, bytes.NewReader(reqBody))
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
	var zhipuResp zhipuResponse
	if err := json.NewDecoder(resp.Body).Decode(&zhipuResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check success status
	if !zhipuResp.Success {
		return nil, &types.ProviderError{
			Provider: p.GetID(),
			Code:     "API_ERROR",
			Message:  zhipuResp.Message,
		}
	}

	// Convert to standard response
	results := make([]*types.SearchResult, len(zhipuResp.Data.Results))
	for i, r := range zhipuResp.Data.Results {
		content := r.Content
		if content == "" {
			content = r.Snippet
		}

		results[i] = &types.SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Content: content,
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
