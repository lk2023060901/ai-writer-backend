package minio

import (
	"errors"
	"net/http"
	"time"
)

// BucketLookupType represents the type of bucket lookup
type BucketLookupType string

const (
	// BucketLookupAuto automatically determines the bucket lookup type
	BucketLookupAuto BucketLookupType = "auto"
	// BucketLookupDNS uses DNS-style bucket lookup (bucket.endpoint)
	BucketLookupDNS BucketLookupType = "dns"
	// BucketLookupPath uses path-style bucket lookup (endpoint/bucket)
	BucketLookupPath BucketLookupType = "path"
)

// Config represents the configuration for MinIO client
type Config struct {
	// Endpoint is the S3-compatible object storage endpoint
	// Examples: "play.min.io", "s3.amazonaws.com", "localhost:9000"
	Endpoint string

	// AccessKeyID is the access key for authentication
	AccessKeyID string

	// SecretAccessKey is the secret key for authentication
	SecretAccessKey string

	// SessionToken is the session token for temporary credentials (optional)
	SessionToken string

	// Region is the region of the object storage (optional)
	// Examples: "us-east-1", "eu-west-1"
	Region string

	// UseSSL determines whether to use HTTPS (true) or HTTP (false)
	UseSSL bool

	// BucketLookup specifies the bucket lookup type
	// Default: BucketLookupAuto
	BucketLookup BucketLookupType

	// Transport is a custom HTTP transport for executing HTTP transactions (optional)
	Transport *http.Transport

	// TraceEnabled enables HTTP request/response tracing for debugging
	TraceEnabled bool

	// MaxRetries is the maximum number of retries for failed requests
	// Default: 3
	MaxRetries int

	// RetryDelay is the initial delay between retries
	// Default: 1 second
	RetryDelay time.Duration

	// ConnectTimeout is the timeout for establishing connections
	// Default: 10 seconds
	ConnectTimeout time.Duration

	// RequestTimeout is the timeout for individual requests
	// Default: 30 seconds
	RequestTimeout time.Duration
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return errors.New("minio: endpoint is required")
	}

	if c.AccessKeyID == "" {
		return errors.New("minio: access key ID is required")
	}

	if c.SecretAccessKey == "" {
		return errors.New("minio: secret access key is required")
	}

	// Validate bucket lookup type
	if c.BucketLookup != "" &&
		c.BucketLookup != BucketLookupAuto &&
		c.BucketLookup != BucketLookupDNS &&
		c.BucketLookup != BucketLookupPath {
		return errors.New("minio: invalid bucket lookup type")
	}

	return nil
}

// SetDefaults sets default values for unspecified configuration fields
func (c *Config) SetDefaults() {
	if c.BucketLookup == "" {
		c.BucketLookup = BucketLookupAuto
	}

	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}

	if c.RetryDelay == 0 {
		c.RetryDelay = time.Second
	}

	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = 10 * time.Second
	}

	if c.RequestTimeout == 0 {
		c.RequestTimeout = 30 * time.Second
	}
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	cfg := &Config{
		UseSSL:         true,
		BucketLookup:   BucketLookupAuto,
		MaxRetries:     3,
		RetryDelay:     time.Second,
		ConnectTimeout: 10 * time.Second,
		RequestTimeout: 30 * time.Second,
	}
	return cfg
}
