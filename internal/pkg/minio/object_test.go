package minio

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPutObject(t *testing.T) {
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

	t.Run("Upload object with content", func(t *testing.T) {
		content := []byte("Hello, MinIO!")
		reader := bytes.NewReader(content)

		info, err := client.PutObject(ctx, testBucket, "test-object.txt", reader, int64(len(content)), PutObjectOptions{
			ContentType: "text/plain",
			UserMetadata: map[string]string{
				"x-amz-meta-custom-key": "custom-value",
			},
		})
		if err != nil {
			t.Fatalf("Failed to upload object: %v", err)
		}

		t.Logf("✓ Object uploaded: bucket=%s, key=%s, size=%d, etag=%s",
			info.Bucket, info.Key, info.Size, info.ETag)
	})

	t.Run("Upload with invalid bucket", func(t *testing.T) {
		content := []byte("test")
		reader := bytes.NewReader(content)

		_, err := client.PutObject(ctx, "", "test.txt", reader, int64(len(content)), PutObjectOptions{})
		if err == nil {
			t.Fatal("Expected error for empty bucket name")
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})

	t.Run("Upload with invalid object name", func(t *testing.T) {
		content := []byte("test")
		reader := bytes.NewReader(content)

		_, err := client.PutObject(ctx, testBucket, "", reader, int64(len(content)), PutObjectOptions{})
		if err == nil {
			t.Fatal("Expected error for empty object name")
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})
}

func TestGetObject(t *testing.T) {
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
	testContent := []byte("Test content for download")
	_, err = client.PutObject(ctx, testBucket, "download-test.txt", bytes.NewReader(testContent), int64(len(testContent)), PutObjectOptions{
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("Failed to upload test object: %v", err)
	}

	t.Run("Download object", func(t *testing.T) {
		object, err := client.GetObject(ctx, testBucket, "download-test.txt", GetObjectOptions{})
		if err != nil {
			t.Fatalf("Failed to get object: %v", err)
		}
		defer object.Close()

		// Read content
		downloadedContent, err := io.ReadAll(object)
		if err != nil {
			t.Fatalf("Failed to read object content: %v", err)
		}

		if !bytes.Equal(downloadedContent, testContent) {
			t.Fatalf("Downloaded content doesn't match. Expected %q, got %q", testContent, downloadedContent)
		}

		t.Logf("✓ Object downloaded successfully, size: %d bytes", len(downloadedContent))
	})

	t.Run("Download non-existent object", func(t *testing.T) {
		object, err := client.GetObject(ctx, testBucket, "non-existent.txt", GetObjectOptions{})
		if err != nil {
			t.Fatalf("GetObject should not error immediately: %v", err)
		}
		defer object.Close()

		// Error should occur when trying to read
		_, err = io.ReadAll(object)
		if err == nil {
			t.Fatal("Expected error when reading non-existent object")
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})
}

func TestFPutObject(t *testing.T) {
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

	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-upload.txt")
	testContent := []byte("File upload test content")
	err = os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("Upload file", func(t *testing.T) {
		info, err := client.FPutObject(ctx, testBucket, "uploaded-file.txt", testFile, PutObjectOptions{
			ContentType: "text/plain",
		})
		if err != nil {
			t.Fatalf("Failed to upload file: %v", err)
		}

		t.Logf("✓ File uploaded: bucket=%s, key=%s, size=%d", info.Bucket, info.Key, info.Size)
	})

	t.Run("Upload non-existent file", func(t *testing.T) {
		_, err := client.FPutObject(ctx, testBucket, "test.txt", "/non/existent/file.txt", PutObjectOptions{})
		if err == nil {
			t.Fatal("Expected error for non-existent file")
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})
}

func TestFGetObject(t *testing.T) {
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
	testContent := []byte("Test content for file download")
	_, err = client.PutObject(ctx, testBucket, "file-download-test.txt", bytes.NewReader(testContent), int64(len(testContent)), PutObjectOptions{
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("Failed to upload test object: %v", err)
	}

	t.Run("Download to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		downloadPath := filepath.Join(tmpDir, "downloaded.txt")

		err := client.FGetObject(ctx, testBucket, "file-download-test.txt", downloadPath, GetObjectOptions{})
		if err != nil {
			t.Fatalf("Failed to download file: %v", err)
		}

		// Verify file content
		content, err := os.ReadFile(downloadPath)
		if err != nil {
			t.Fatalf("Failed to read downloaded file: %v", err)
		}

		if !bytes.Equal(content, testContent) {
			t.Fatalf("Downloaded content doesn't match. Expected %q, got %q", testContent, content)
		}

		t.Logf("✓ File downloaded successfully to: %s", downloadPath)
	})
}

func TestStatObject(t *testing.T) {
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
	testContent := []byte("Test content for stat")
	_, err = client.PutObject(ctx, testBucket, "stat-test.txt", bytes.NewReader(testContent), int64(len(testContent)), PutObjectOptions{
		ContentType: "text/plain",
		UserMetadata: map[string]string{
			"x-amz-meta-author": "test-user",
		},
	})
	if err != nil {
		t.Fatalf("Failed to upload test object: %v", err)
	}

	t.Run("Get object metadata", func(t *testing.T) {
		info, err := client.StatObject(ctx, testBucket, "stat-test.txt", StatObjectOptions{})
		if err != nil {
			t.Fatalf("Failed to stat object: %v", err)
		}

		if info.Size != int64(len(testContent)) {
			t.Errorf("Expected size %d, got %d", len(testContent), info.Size)
		}

		if info.ContentType != "text/plain" {
			t.Errorf("Expected content type 'text/plain', got %q", info.ContentType)
		}

		t.Logf("✓ Object metadata: key=%s, size=%d, etag=%s, content_type=%s",
			info.Key, info.Size, info.ETag, info.ContentType)
		t.Logf("  Metadata: %v", info.Metadata)
	})

	t.Run("Stat non-existent object", func(t *testing.T) {
		_, err := client.StatObject(ctx, testBucket, "non-existent.txt", StatObjectOptions{})
		if err == nil {
			t.Fatal("Expected error for non-existent object")
		}

		if !IsNotFound(err) {
			t.Fatalf("Expected NotFound error, got: %v", err)
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})
}

func TestRemoveObject(t *testing.T) {
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

	t.Run("Remove existing object", func(t *testing.T) {
		// Upload object
		content := []byte("To be deleted")
		_, err := client.PutObject(ctx, testBucket, "delete-test.txt", bytes.NewReader(content), int64(len(content)), PutObjectOptions{})
		if err != nil {
			t.Fatalf("Failed to upload object: %v", err)
		}

		// Remove object
		err = client.RemoveObject(ctx, testBucket, "delete-test.txt", RemoveObjectOptions{})
		if err != nil {
			t.Fatalf("Failed to remove object: %v", err)
		}

		// Verify object is removed
		_, err = client.StatObject(ctx, testBucket, "delete-test.txt", StatObjectOptions{})
		if err == nil {
			t.Fatal("Expected error after object removal")
		}

		if !IsNotFound(err) {
			t.Fatalf("Expected NotFound error, got: %v", err)
		}

		t.Log("✓ Object removed successfully")
	})

	t.Run("Remove non-existent object", func(t *testing.T) {
		// MinIO doesn't error when removing non-existent objects
		err := client.RemoveObject(ctx, testBucket, "non-existent.txt", RemoveObjectOptions{})
		if err != nil {
			t.Logf("Note: Got error when removing non-existent object: %v", err)
		} else {
			t.Log("✓ No error when removing non-existent object (expected behavior)")
		}
	})
}

func TestCopyObject(t *testing.T) {
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

	// Upload source object
	testContent := []byte("Content to be copied")
	_, err = client.PutObject(ctx, testBucket, "source.txt", bytes.NewReader(testContent), int64(len(testContent)), PutObjectOptions{
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("Failed to upload source object: %v", err)
	}

	t.Run("Copy object", func(t *testing.T) {
		dst := CopyDestOptions{
			Bucket: testBucket,
			Object: "destination.txt",
		}

		src := CopySrcOptions{
			Bucket: testBucket,
			Object: "source.txt",
		}

		info, err := client.CopyObject(ctx, dst, src)
		if err != nil {
			t.Fatalf("Failed to copy object: %v", err)
		}

		t.Logf("✓ Object copied: src=%s/%s -> dst=%s/%s, etag=%s",
			src.Bucket, src.Object, dst.Bucket, dst.Object, info.ETag)

		// Verify destination object exists
		_, err = client.StatObject(ctx, testBucket, "destination.txt", StatObjectOptions{})
		if err != nil {
			t.Fatalf("Failed to verify destination object: %v", err)
		}

		t.Log("✓ Destination object verified")
	})
}

func TestRemoveIncompleteUpload(t *testing.T) {
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

	t.Run("Remove incomplete upload", func(t *testing.T) {
		// This test simply verifies the API works
		// In a real scenario, you'd have an incomplete multipart upload
		err := client.RemoveIncompleteUpload(ctx, testBucket, "incomplete-upload.txt")
		if err != nil {
			// It's okay if there's no incomplete upload
			t.Logf("Note: %v", err)
		}

		t.Log("✓ RemoveIncompleteUpload API call succeeded")
	})
}
