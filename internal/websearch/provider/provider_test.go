package provider

import (
	"testing"

	"github.com/lk2023060901/ai-writer-backend/internal/websearch/types"

	"github.com/stretchr/testify/assert"
)

func TestNewBaseProvider(t *testing.T) {
	config := &types.ProviderConfig{
		ID:      types.ProviderTavily,
		Name:    "Tavily",
		APIHost: "https://api.tavily.com",
		APIKey:  "test-key",
		Timeout: 30,
	}

	base := NewBaseProvider(config)
	assert.NotNil(t, base)
	assert.Equal(t, types.ProviderTavily, base.GetID())
	assert.Equal(t, "Tavily", base.GetName())
	assert.Equal(t, "test-key", base.GetAPIKey())
}

func TestBaseProvider_GetAPIKey_Rotation(t *testing.T) {
	config := &types.ProviderConfig{
		ID:      types.ProviderTavily,
		Name:    "Tavily",
		APIHost: "https://api.tavily.com",
		APIKey:  "key1, key2, key3",
		Timeout: 30,
	}

	base := NewBaseProvider(config)

	// Test key rotation
	assert.Equal(t, "key1", base.GetAPIKey())
	assert.Equal(t, "key2", base.GetAPIKey())
	assert.Equal(t, "key3", base.GetAPIKey())
	assert.Equal(t, "key1", base.GetAPIKey()) // Should rotate back to first
}

func TestProviderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *types.ProviderConfig
		wantErr error
	}{
		{
			name: "valid tavily config",
			config: &types.ProviderConfig{
				ID:      types.ProviderTavily,
				Name:    "Tavily",
				APIHost: "https://api.tavily.com",
				APIKey:  "test-key",
			},
			wantErr: nil,
		},
		{
			name: "valid searxng config",
			config: &types.ProviderConfig{
				ID:      types.ProviderSearXNG,
				Name:    "SearXNG",
				APIHost: "https://search.example.com",
			},
			wantErr: nil,
		},
		{
			name: "missing provider ID",
			config: &types.ProviderConfig{
				Name:    "Test",
				APIHost: "https://api.test.com",
				APIKey:  "test-key",
			},
			wantErr: types.ErrInvalidProviderID,
		},
		{
			name: "missing API host",
			config: &types.ProviderConfig{
				ID:     types.ProviderTavily,
				Name:   "Tavily",
				APIKey: "test-key",
			},
			wantErr: types.ErrInvalidAPIHost,
		},
		{
			name: "missing API key for non-SearXNG provider",
			config: &types.ProviderConfig{
				ID:      types.ProviderTavily,
				Name:    "Tavily",
				APIHost: "https://api.tavily.com",
			},
			wantErr: types.ErrMissingAPIKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
