package minio

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

// Client wraps the MinIO client with additional functionality
type Client struct {
	client *minio.Client
	config *Config
	logger *zap.Logger
	mu     sync.RWMutex
	closed bool
}

// NewClient creates a new MinIO client
func NewClient(cfg *Config, logger *zap.Logger) (*Client, error) {
	if cfg == nil {
		return nil, ErrInvalidArgument
	}

	// Set defaults
	cfg.SetDefaults()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, WrapErrorWithMessage("NewClient", err, "invalid configuration")
	}

	// Create MinIO client options
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken),
		Secure: cfg.UseSSL,
	}

	// Set region if provided
	if cfg.Region != "" {
		opts.Region = cfg.Region
	}

	// Set bucket lookup type
	switch cfg.BucketLookup {
	case BucketLookupDNS:
		opts.BucketLookup = minio.BucketLookupDNS
	case BucketLookupPath:
		opts.BucketLookup = minio.BucketLookupPath
	default:
		opts.BucketLookup = minio.BucketLookupAuto
	}

	// Set custom transport if provided
	if cfg.Transport != nil {
		opts.Transport = cfg.Transport
	}

	// Create MinIO client
	minioClient, err := minio.New(cfg.Endpoint, opts)
	if err != nil {
		return nil, WrapErrorWithMessage("NewClient", err, "failed to create minio client")
	}

	// Enable tracing if configured
	if cfg.TraceEnabled {
		minioClient.TraceOn(os.Stderr)
	}

	// Create client wrapper
	client := &Client{
		client: minioClient,
		config: cfg,
		logger: logger,
	}

	// Log successful initialization
	if logger != nil {
		logger.Info("minio client initialized successfully",
			zap.String("endpoint", cfg.Endpoint),
			zap.String("region", cfg.Region),
			zap.Bool("use_ssl", cfg.UseSSL),
			zap.String("bucket_lookup", string(cfg.BucketLookup)),
		)
	}

	return client, nil
}

// Ping checks if the MinIO server is accessible by listing buckets
func (c *Client) Ping(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrConnectionFailed
	}

	// Try to list buckets to verify connectivity
	_, err := c.client.ListBuckets(ctx)
	if err != nil {
		return WrapErrorWithMessage("Ping", err, "failed to connect to minio server")
	}

	return nil
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	if c.logger != nil {
		c.logger.Info("minio client closed")
	}

	return nil
}

// GetUnderlyingClient returns the underlying MinIO client
// This is useful for advanced operations not covered by this wrapper
func (c *Client) GetUnderlyingClient() *minio.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}

// IsClosed returns whether the client is closed
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// checkClosed returns an error if the client is closed
func (c *Client) checkClosed() error {
	if c.IsClosed() {
		return fmt.Errorf("minio: client is closed")
	}
	return nil
}
