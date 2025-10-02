package milvus

import (
	"errors"
	"fmt"
	"time"
)

// Config represents the configuration for Milvus client
type Config struct {
	// Connection settings
	Address  string // Milvus server address (e.g., "localhost:19530")
	Username string // Username for authentication (optional)
	Password string // Password for authentication (optional)
	APIKey   string // API Key for cloud service (optional)

	// Database settings
	Database string // Database name (optional, default is "default")

	// Connection pool settings
	MaxIdleConns    int           // Maximum number of idle connections
	MaxOpenConns    int           // Maximum number of open connections
	ConnMaxLifetime time.Duration // Maximum connection lifetime

	// Timeout settings
	DialTimeout    time.Duration // Dial timeout
	RequestTimeout time.Duration // Request timeout
	KeepAlive      time.Duration // Keep alive interval

	// Retry settings
	MaxRetries int           // Maximum number of retries
	RetryDelay time.Duration // Delay between retries

	// TLS settings
	EnableTLS bool   // Enable TLS connection
	TLSMode   string // TLS mode (optional)

	// Other settings
	EnableTracing bool // Enable request tracing for debugging
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Address == "" {
		return errors.New("milvus: address is required")
	}

	// Validate authentication
	if c.APIKey != "" && (c.Username != "" || c.Password != "") {
		return errors.New("milvus: cannot use both API key and username/password authentication")
	}

	// Validate connection pool settings
	if c.MaxIdleConns < 0 {
		return errors.New("milvus: max idle connections must be non-negative")
	}

	if c.MaxOpenConns < 0 {
		return errors.New("milvus: max open connections must be non-negative")
	}

	if c.MaxOpenConns > 0 && c.MaxIdleConns > c.MaxOpenConns {
		return errors.New("milvus: max idle connections cannot exceed max open connections")
	}

	// Validate timeout settings
	if c.DialTimeout < 0 {
		return errors.New("milvus: dial timeout must be non-negative")
	}

	if c.RequestTimeout < 0 {
		return errors.New("milvus: request timeout must be non-negative")
	}

	if c.KeepAlive < 0 {
		return errors.New("milvus: keep alive interval must be non-negative")
	}

	// Validate retry settings
	if c.MaxRetries < 0 {
		return errors.New("milvus: max retries must be non-negative")
	}

	if c.RetryDelay < 0 {
		return errors.New("milvus: retry delay must be non-negative")
	}

	return nil
}

// SetDefaults sets default values for unspecified configuration fields
func (c *Config) SetDefaults() {
	if c.Database == "" {
		c.Database = "default"
	}

	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 10
	}

	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = 100
	}

	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = 30 * time.Minute
	}

	if c.DialTimeout == 0 {
		c.DialTimeout = 10 * time.Second
	}

	if c.RequestTimeout == 0 {
		c.RequestTimeout = 30 * time.Second
	}

	if c.KeepAlive == 0 {
		c.KeepAlive = 30 * time.Second
	}

	if c.MaxRetries == 0 {
		c.MaxRetries = DefaultRetries
	}

	if c.RetryDelay == 0 {
		c.RetryDelay = DefaultRetryDelay
	}
}

// String returns a string representation of the configuration (hides sensitive data)
func (c *Config) String() string {
	password := "***"
	if c.Password == "" {
		password = ""
	}

	apiKey := "***"
	if c.APIKey == "" {
		apiKey = ""
	}

	return fmt.Sprintf("Config{Address: %s, Username: %s, Password: %s, APIKey: %s, Database: %s, EnableTLS: %v}",
		c.Address, c.Username, password, apiKey, c.Database, c.EnableTLS)
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	cfg := &Config{
		Address:         "localhost:19530",
		Database:        "default",
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: 30 * time.Minute,
		DialTimeout:     10 * time.Second,
		RequestTimeout:  30 * time.Second,
		KeepAlive:       30 * time.Second,
		MaxRetries:      DefaultRetries,
		RetryDelay:      DefaultRetryDelay,
		EnableTLS:       false,
		EnableTracing:   false,
	}
	return cfg
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}

	clone := *c
	return &clone
}
