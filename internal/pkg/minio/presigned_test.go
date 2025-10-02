package minio

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestPresignedGetObject(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	defer cleanupTestBucket(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create bucket
	err := client.MakeBucket(ctx, testBucket, MakeBucketOptions{
		Region: testRegion,
	})
	if err != nil && !IsBucketAlreadyExists(err) {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	// Upload test object
	testContent := []byte("Presigned GET test content")
	_, err = client.PutObject(ctx, testBucket, "presigned-get-test.txt", bytes.NewReader(testContent), int64(len(testContent)), PutObjectOptions{
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("Failed to upload test object: %v", err)
	}

	t.Run("Generate presigned GET URL", func(t *testing.T) {
		reqParams := make(url.Values)
		reqParams.Set("response-content-disposition", "attachment; filename=\"downloaded.txt\"")

		presignedURL, err := client.PresignedGetObject(ctx, testBucket, "presigned-get-test.txt", time.Hour, reqParams)
		if err != nil {
			t.Fatalf("Failed to generate presigned URL: %v", err)
		}

		t.Logf("✓ Presigned GET URL: %s", presignedURL.String())

		// Test the URL by making a request
		resp, err := http.Get(presignedURL.String())
		if err != nil {
			t.Fatalf("Failed to fetch presigned URL: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify content
		downloadedContent, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		if !bytes.Equal(downloadedContent, testContent) {
			t.Fatalf("Downloaded content doesn't match. Expected %q, got %q", testContent, downloadedContent)
		}

		t.Log("✓ Presigned URL verified with HTTP GET request")
	})

	t.Run("Generate presigned GET URL with invalid expiry", func(t *testing.T) {
		_, err := client.PresignedGetObject(ctx, testBucket, "test.txt", 0, nil)
		if err == nil {
			t.Fatal("Expected error for zero expiry")
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})

	t.Run("Generate presigned GET URL with negative expiry", func(t *testing.T) {
		_, err := client.PresignedGetObject(ctx, testBucket, "test.txt", -time.Hour, nil)
		if err == nil {
			t.Fatal("Expected error for negative expiry")
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})
}

func TestPresignedPutObject(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	defer cleanupTestBucket(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create bucket
	err := client.MakeBucket(ctx, testBucket, MakeBucketOptions{
		Region: testRegion,
	})
	if err != nil && !IsBucketAlreadyExists(err) {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	t.Run("Generate presigned PUT URL", func(t *testing.T) {
		presignedURL, err := client.PresignedPutObject(ctx, testBucket, "presigned-put-test.txt", time.Hour)
		if err != nil {
			t.Fatalf("Failed to generate presigned PUT URL: %v", err)
		}

		t.Logf("✓ Presigned PUT URL: %s", presignedURL.String())

		// Test the URL by uploading content
		testContent := []byte("Presigned PUT test content")
		req, err := http.NewRequest(http.MethodPut, presignedURL.String(), bytes.NewReader(testContent))
		if err != nil {
			t.Fatalf("Failed to create PUT request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to PUT to presigned URL: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify object was uploaded
		_, err = client.StatObject(ctx, testBucket, "presigned-put-test.txt", StatObjectOptions{})
		if err != nil {
			t.Fatalf("Failed to verify uploaded object: %v", err)
		}

		t.Log("✓ Presigned PUT URL verified with HTTP PUT request")
	})

	t.Run("Generate presigned PUT URL with invalid parameters", func(t *testing.T) {
		_, err := client.PresignedPutObject(ctx, "", "test.txt", time.Hour)
		if err == nil {
			t.Fatal("Expected error for empty bucket name")
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})
}

func TestPresignedHeadObject(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	defer cleanupTestBucket(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create bucket
	err := client.MakeBucket(ctx, testBucket, MakeBucketOptions{
		Region: testRegion,
	})
	if err != nil && !IsBucketAlreadyExists(err) {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	// Upload test object
	testContent := []byte("Presigned HEAD test content")
	_, err = client.PutObject(ctx, testBucket, "presigned-head-test.txt", bytes.NewReader(testContent), int64(len(testContent)), PutObjectOptions{
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("Failed to upload test object: %v", err)
	}

	t.Run("Generate presigned HEAD URL", func(t *testing.T) {
		reqParams := make(url.Values)

		presignedURL, err := client.PresignedHeadObject(ctx, testBucket, "presigned-head-test.txt", time.Hour, reqParams)
		if err != nil {
			t.Fatalf("Failed to generate presigned HEAD URL: %v", err)
		}

		t.Logf("✓ Presigned HEAD URL: %s", presignedURL.String())

		// Test the URL by making a HEAD request
		req, err := http.NewRequest(http.MethodHead, presignedURL.String(), nil)
		if err != nil {
			t.Fatalf("Failed to create HEAD request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to HEAD presigned URL: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify headers
		contentLength := resp.Header.Get("Content-Length")
		if contentLength == "" {
			t.Fatal("Expected Content-Length header")
		}

		t.Logf("✓ Presigned HEAD URL verified, Content-Length: %s", contentLength)
	})
}

func TestPresignedPostPolicy(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	defer cleanupTestBucket(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create bucket
	err := client.MakeBucket(ctx, testBucket, MakeBucketOptions{
		Region: testRegion,
	})
	if err != nil && !IsBucketAlreadyExists(err) {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	t.Run("Generate presigned POST policy", func(t *testing.T) {
		policy := NewPostPolicy()

		err := policy.SetBucket(testBucket)
		if err != nil {
			t.Fatalf("Failed to set bucket: %v", err)
		}

		err = policy.SetKey("presigned-post-test.txt")
		if err != nil {
			t.Fatalf("Failed to set key: %v", err)
		}

		err = policy.SetExpires(time.Now().UTC().Add(time.Hour))
		if err != nil {
			t.Fatalf("Failed to set expires: %v", err)
		}

		err = policy.SetContentType("text/plain")
		if err != nil {
			t.Fatalf("Failed to set content type: %v", err)
		}

		err = policy.SetContentLengthRange(1, 1024*1024)
		if err != nil {
			t.Fatalf("Failed to set content length range: %v", err)
		}

		presignedURL, formData, err := client.PresignedPostPolicy(ctx, policy)
		if err != nil {
			t.Fatalf("Failed to generate presigned POST policy: %v", err)
		}

		t.Logf("✓ Presigned POST URL: %s", presignedURL.String())
		t.Log("  Form data:")
		for k, v := range formData {
			t.Logf("    %s = %s", k, v)
		}
	})

	t.Run("Generate presigned POST policy with nil policy", func(t *testing.T) {
		_, _, err := client.PresignedPostPolicy(ctx, nil)
		if err == nil {
			t.Fatal("Expected error for nil policy")
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})
}

func TestPresignedURLExpiry(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	defer cleanupTestBucket(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create bucket
	err := client.MakeBucket(ctx, testBucket, MakeBucketOptions{
		Region: testRegion,
	})
	if err != nil && !IsBucketAlreadyExists(err) {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	// Upload test object
	testContent := []byte("Expiry test content")
	_, err = client.PutObject(ctx, testBucket, "expiry-test.txt", bytes.NewReader(testContent), int64(len(testContent)), PutObjectOptions{})
	if err != nil {
		t.Fatalf("Failed to upload test object: %v", err)
	}

	t.Run("Generate presigned URL with different expiry durations", func(t *testing.T) {
		testCases := []struct {
			name   string
			expiry time.Duration
		}{
			{"1 second", time.Second},
			{"1 minute", time.Minute},
			{"1 hour", time.Hour},
			{"1 day", 24 * time.Hour},
			{"7 days", 7 * 24 * time.Hour},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				presignedURL, err := client.PresignedGetObject(ctx, testBucket, "expiry-test.txt", tc.expiry, nil)
				if err != nil {
					t.Fatalf("Failed to generate presigned URL: %v", err)
				}

				t.Logf("✓ Generated presigned URL with %s expiry: %s", tc.name, presignedURL.String())
			})
		}
	})
}
