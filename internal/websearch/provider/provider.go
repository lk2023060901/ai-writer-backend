package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/websearch/types"
)

// Provider defines the interface for search providers
type Provider interface {
	// Search executes a search query
	Search(ctx context.Context, req *types.SearchRequest) (*types.SearchResponse, error)

	// GetID returns the provider ID
	GetID() types.ProviderID

	// GetName returns the provider name
	GetName() string

	// Validate validates the provider configuration
	Validate() error

	// IsAvailable checks if the provider is available
	IsAvailable(ctx context.Context) bool
}

// BaseProvider provides common functionality for all providers
type BaseProvider struct {
	config     *types.ProviderConfig
	httpClient *http.Client
	apiKeys    []string // Support multiple API keys for rotation
	keyIndex   int      // Current key index
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(config *types.ProviderConfig) *BaseProvider {
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Parse multiple API keys (comma-separated)
	var apiKeys []string
	if config.APIKey != "" {
		apiKeys = strings.Split(config.APIKey, ",")
		for i := range apiKeys {
			apiKeys[i] = strings.TrimSpace(apiKeys[i])
		}
	}

	return &BaseProvider{
		config:     config,
		httpClient: httpClient,
		apiKeys:    apiKeys,
		keyIndex:   0,
	}
}

// GetID returns the provider ID
func (b *BaseProvider) GetID() types.ProviderID {
	return b.config.ID
}

// GetName returns the provider name
func (b *BaseProvider) GetName() string {
	return b.config.Name
}

// GetConfig returns the provider configuration
func (b *BaseProvider) GetConfig() *types.ProviderConfig {
	return b.config
}

// GetHTTPClient returns the HTTP client
func (b *BaseProvider) GetHTTPClient() *http.Client {
	return b.httpClient
}

// GetAPIKey returns the current API key (with rotation support)
func (b *BaseProvider) GetAPIKey() string {
	if len(b.apiKeys) == 0 {
		return ""
	}

	key := b.apiKeys[b.keyIndex]
	b.keyIndex = (b.keyIndex + 1) % len(b.apiKeys)
	return key
}

// BuildDefaultHeaders builds default HTTP headers
func (b *BaseProvider) BuildDefaultHeaders() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   "AI-Writer-Backend/1.0",
		"HTTP-Referer": "https://ai-writer.com",
		"X-Title":      "AI Writer",
	}
}

// DoRequest executes an HTTP request with retry logic
func (b *BaseProvider) DoRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	maxRetries := b.config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		resp, err := b.httpClient.Do(req.WithContext(ctx))
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Exponential backoff
		if i < maxRetries-1 {
			backoff := time.Duration(1<<uint(i)) * time.Second
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

// Validate validates the provider configuration
func (b *BaseProvider) Validate() error {
	return b.config.Validate()
}

// IsAvailable checks if the provider is available (default implementation)
func (b *BaseProvider) IsAvailable(ctx context.Context) bool {
	// Subclasses can override this method to implement health checks
	return true
}
