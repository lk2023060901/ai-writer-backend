package types

type ProviderID string

const (
	ProviderTavily  ProviderID = "tavily"
	ProviderSearXNG ProviderID = "searxng"
	ProviderExa     ProviderID = "exa"
	ProviderZhipu   ProviderID = "zhipu"
	ProviderBocha   ProviderID = "bocha"
)

// ProviderConfig represents search provider configuration
type ProviderConfig struct {
	ID   ProviderID `json:"id" yaml:"id"`
	Name string     `json:"name" yaml:"name"`

	// API settings
	APIHost string `json:"api_host" yaml:"api_host"`
	APIKey  string `json:"api_key,omitempty" yaml:"api_key,omitempty"`

	// SearXNG Basic Auth
	BasicAuthUsername string `json:"basic_auth_username,omitempty" yaml:"basic_auth_username,omitempty"`
	BasicAuthPassword string `json:"basic_auth_password,omitempty" yaml:"basic_auth_password,omitempty"`

	// Optional settings
	Timeout    int `json:"timeout,omitempty" yaml:"timeout,omitempty"`         // seconds
	MaxRetries int `json:"max_retries,omitempty" yaml:"max_retries,omitempty"` // default: 3
	RateLimit  int `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`   // requests per second
}

// Validate validates the provider configuration
func (c *ProviderConfig) Validate() error {
	if c.ID == "" {
		return ErrInvalidProviderID
	}
	if c.Name == "" {
		return ErrInvalidProviderName
	}
	if c.APIHost == "" {
		return ErrInvalidAPIHost
	}

	// Provider-specific validation
	switch c.ID {
	case ProviderSearXNG:
		// SearXNG doesn't require API key but may need basic auth
		if c.BasicAuthUsername != "" && c.BasicAuthPassword == "" {
			return ErrMissingBasicAuthPassword
		}
	default:
		// Most providers require API key
		if c.APIKey == "" {
			return ErrMissingAPIKey
		}
	}

	return nil
}
