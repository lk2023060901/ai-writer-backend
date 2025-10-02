package minio

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

const (
	testEndpoint        = "localhost:9000"
	testAccessKeyID     = "minioadmin"
	testSecretAccessKey = "minioadmin"
	testBucket          = "test-bucket"
	testRegion          = "us-east-1"
)

func setupTestClient(t *testing.T) *Client {
	cfg := &Config{
		Endpoint:        testEndpoint,
		AccessKeyID:     testAccessKeyID,
		SecretAccessKey: testSecretAccessKey,
		UseSSL:          false,
		Region:          testRegion,
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	client, err := NewClient(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return client
}

func cleanupTestBucket(t *testing.T, client *Client) {
	ctx := context.Background()

	// Check if bucket exists
	exists, err := client.BucketExists(ctx, testBucket)
	if err != nil {
		t.Logf("Failed to check bucket existence: %v", err)
		return
	}

	if !exists {
		return
	}

	// List and remove all objects in the bucket
	objCh, errCh := client.ListObjects(ctx, testBucket, ListObjectsOptions{
		Recursive: true,
	})

	for {
		select {
		case obj, ok := <-objCh:
			if !ok {
				goto Done
			}
			err := client.RemoveObject(ctx, testBucket, obj.Key, RemoveObjectOptions{})
			if err != nil {
				t.Logf("Failed to remove object %s: %v", obj.Key, err)
			}
		case err := <-errCh:
			if err != nil {
				t.Logf("Error listing objects: %v", err)
			}
		}
	}

Done:
	// Remove the bucket
	err = client.RemoveBucket(ctx, testBucket)
	if err != nil {
		t.Logf("Failed to remove bucket: %v", err)
	}
}

func TestNewClient(t *testing.T) {
	t.Run("Valid configuration", func(t *testing.T) {
		client := setupTestClient(t)
		defer client.Close()

		if client == nil {
			t.Fatal("Expected client to be created")
		}

		if client.IsClosed() {
			t.Fatal("Expected client to not be closed")
		}

		t.Log("✓ Client created successfully")
	})

	t.Run("Invalid configuration - missing endpoint", func(t *testing.T) {
		cfg := &Config{
			AccessKeyID:     testAccessKeyID,
			SecretAccessKey: testSecretAccessKey,
		}

		logger, _ := zap.NewDevelopment()
		_, err := NewClient(cfg, logger)
		if err == nil {
			t.Fatal("Expected error for missing endpoint")
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})

	t.Run("Invalid configuration - missing credentials", func(t *testing.T) {
		cfg := &Config{
			Endpoint: testEndpoint,
		}

		logger, _ := zap.NewDevelopment()
		_, err := NewClient(cfg, logger)
		if err == nil {
			t.Fatal("Expected error for missing credentials")
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})
}

func TestClientPing(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	t.Log("✓ Ping successful")
}

func TestClientClose(t *testing.T) {
	client := setupTestClient(t)

	if client.IsClosed() {
		t.Fatal("Expected client to not be closed initially")
	}

	err := client.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if !client.IsClosed() {
		t.Fatal("Expected client to be closed")
	}

	// Closing again should not error
	err = client.Close()
	if err != nil {
		t.Fatalf("Second close should not error: %v", err)
	}

	t.Log("✓ Client closed successfully")
}

func TestClientOperationsAfterClose(t *testing.T) {
	client := setupTestClient(t)
	client.Close()

	ctx := context.Background()

	t.Run("Ping after close", func(t *testing.T) {
		err := client.Ping(ctx)
		if err == nil {
			t.Fatal("Expected error when pinging closed client")
		}
		t.Logf("✓ Error correctly returned: %v", err)
	})

	t.Run("BucketExists after close", func(t *testing.T) {
		_, err := client.BucketExists(ctx, testBucket)
		if err == nil {
			t.Fatal("Expected error when checking bucket on closed client")
		}
		t.Logf("✓ Error correctly returned: %v", err)
	})
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		shouldErr bool
	}{
		{
			name: "Valid config with all fields",
			config: &Config{
				Endpoint:        testEndpoint,
				AccessKeyID:     testAccessKeyID,
				SecretAccessKey: testSecretAccessKey,
				Region:          testRegion,
				UseSSL:          false,
				BucketLookup:    BucketLookupAuto,
			},
			shouldErr: false,
		},
		{
			name: "Valid config with minimal fields",
			config: &Config{
				Endpoint:        testEndpoint,
				AccessKeyID:     testAccessKeyID,
				SecretAccessKey: testSecretAccessKey,
			},
			shouldErr: false,
		},
		{
			name: "Invalid - missing endpoint",
			config: &Config{
				AccessKeyID:     testAccessKeyID,
				SecretAccessKey: testSecretAccessKey,
			},
			shouldErr: true,
		},
		{
			name: "Invalid - missing access key",
			config: &Config{
				Endpoint:        testEndpoint,
				SecretAccessKey: testSecretAccessKey,
			},
			shouldErr: true,
		},
		{
			name: "Invalid - missing secret key",
			config: &Config{
				Endpoint:    testEndpoint,
				AccessKeyID: testAccessKeyID,
			},
			shouldErr: true,
		},
		{
			name: "Invalid bucket lookup type",
			config: &Config{
				Endpoint:        testEndpoint,
				AccessKeyID:     testAccessKeyID,
				SecretAccessKey: testSecretAccessKey,
				BucketLookup:    "invalid",
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.shouldErr && err == nil {
				t.Fatal("Expected validation error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Fatalf("Expected no validation error but got: %v", err)
			}

			if err != nil {
				t.Logf("✓ Validation error: %v", err)
			} else {
				t.Log("✓ Validation passed")
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := &Config{
		Endpoint:        testEndpoint,
		AccessKeyID:     testAccessKeyID,
		SecretAccessKey: testSecretAccessKey,
	}

	cfg.SetDefaults()

	if cfg.BucketLookup != BucketLookupAuto {
		t.Errorf("Expected BucketLookup to be %s, got %s", BucketLookupAuto, cfg.BucketLookup)
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", cfg.MaxRetries)
	}

	if cfg.RetryDelay != time.Second {
		t.Errorf("Expected RetryDelay to be 1s, got %v", cfg.RetryDelay)
	}

	if cfg.ConnectTimeout != 10*time.Second {
		t.Errorf("Expected ConnectTimeout to be 10s, got %v", cfg.ConnectTimeout)
	}

	if cfg.RequestTimeout != 30*time.Second {
		t.Errorf("Expected RequestTimeout to be 30s, got %v", cfg.RequestTimeout)
	}

	t.Log("✓ All defaults set correctly")
}
