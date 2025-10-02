package provider

import (
	"context"
	"testing"

	"github.com/lk2023060901/ai-writer-backend/internal/websearch/types"

	"github.com/stretchr/testify/assert"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	assert.NotNil(t, factory)

	// Check that all built-in providers are registered
	providers := factory.ListProviders()
	assert.Contains(t, providers, types.ProviderTavily)
	assert.Contains(t, providers, types.ProviderSearXNG)
	assert.Contains(t, providers, types.ProviderExa)
	assert.Contains(t, providers, types.ProviderZhipu)
	assert.Contains(t, providers, types.ProviderBocha)
}

func TestFactory_Create(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		name     string
		config   *types.ProviderConfig
		wantType string
		wantErr  bool
	}{
		{
			name: "create tavily provider",
			config: &types.ProviderConfig{
				ID:      types.ProviderTavily,
				Name:    "Tavily",
				APIHost: "https://api.tavily.com",
				APIKey:  "test-key",
			},
			wantType: "*provider.TavilyProvider",
			wantErr:  false,
		},
		{
			name: "create searxng provider",
			config: &types.ProviderConfig{
				ID:      types.ProviderSearXNG,
				Name:    "SearXNG",
				APIHost: "https://search.example.com",
			},
			wantType: "*provider.SearXNGProvider",
			wantErr:  false,
		},
		{
			name: "invalid config",
			config: &types.ProviderConfig{
				ID:   types.ProviderTavily,
				Name: "Tavily",
				// Missing APIHost
			},
			wantErr: true,
		},
		{
			name: "unknown provider",
			config: &types.ProviderConfig{
				ID:      "unknown",
				Name:    "Unknown",
				APIHost: "https://api.unknown.com",
				APIKey:  "test-key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.Create(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, tt.config.ID, provider.GetID())
			}
		})
	}
}

// mockProvider is a mock implementation for testing
type mockProvider struct {
	*BaseProvider
}

func (m *mockProvider) Search(ctx context.Context, req *types.SearchRequest) (*types.SearchResponse, error) {
	return &types.SearchResponse{
		Results:    []*types.SearchResult{},
		TotalCount: 0,
	}, nil
}

func TestFactory_Register(t *testing.T) {
	factory := NewFactory()

	// Register a custom provider
	customID := types.ProviderID("custom")
	constructor := func(config *types.ProviderConfig) (Provider, error) {
		return &mockProvider{
			BaseProvider: NewBaseProvider(config),
		}, nil
	}

	factory.Register(customID, constructor)

	providers := factory.ListProviders()
	assert.Contains(t, providers, customID)
}
