package minio

import (
	"context"
	"testing"
	"time"
)

func TestMakeBucket(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	defer cleanupTestBucket(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("Create new bucket", func(t *testing.T) {
		err := client.MakeBucket(ctx, testBucket, MakeBucketOptions{
			Region: testRegion,
		})
		if err != nil {
			t.Fatalf("Failed to create bucket: %v", err)
		}

		t.Logf("✓ Bucket '%s' created successfully", testBucket)
	})

	t.Run("Create existing bucket", func(t *testing.T) {
		err := client.MakeBucket(ctx, testBucket, MakeBucketOptions{
			Region: testRegion,
		})
		if err == nil {
			t.Fatal("Expected error when creating existing bucket")
		}

		if !IsBucketAlreadyExists(err) {
			t.Fatalf("Expected BucketAlreadyExists error, got: %v", err)
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})

	t.Run("Create bucket with invalid name", func(t *testing.T) {
		err := client.MakeBucket(ctx, "", MakeBucketOptions{})
		if err == nil {
			t.Fatal("Expected error for empty bucket name")
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})
}

func TestListBuckets(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	defer cleanupTestBucket(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a test bucket
	err := client.MakeBucket(ctx, testBucket, MakeBucketOptions{
		Region: testRegion,
	})
	if err != nil && !IsBucketAlreadyExists(err) {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		t.Fatalf("Failed to list buckets: %v", err)
	}

	found := false
	for _, bucket := range buckets {
		t.Logf("  - %s (created: %s)", bucket.Name, bucket.CreationDate)
		if bucket.Name == testBucket {
			found = true
		}
	}

	if !found {
		t.Fatalf("Expected to find bucket '%s' in list", testBucket)
	}

	t.Logf("✓ Listed %d buckets, found test bucket", len(buckets))
}

func TestBucketExists(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	defer cleanupTestBucket(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("Check non-existent bucket", func(t *testing.T) {
		exists, err := client.BucketExists(ctx, "non-existent-bucket-xyz")
		if err != nil {
			t.Fatalf("Failed to check bucket existence: %v", err)
		}

		if exists {
			t.Fatal("Expected bucket to not exist")
		}

		t.Log("✓ Non-existent bucket correctly identified")
	})

	t.Run("Check existing bucket", func(t *testing.T) {
		// Create bucket first
		err := client.MakeBucket(ctx, testBucket, MakeBucketOptions{
			Region: testRegion,
		})
		if err != nil && !IsBucketAlreadyExists(err) {
			t.Fatalf("Failed to create bucket: %v", err)
		}

		exists, err := client.BucketExists(ctx, testBucket)
		if err != nil {
			t.Fatalf("Failed to check bucket existence: %v", err)
		}

		if !exists {
			t.Fatal("Expected bucket to exist")
		}

		t.Log("✓ Existing bucket correctly identified")
	})

	t.Run("Check with invalid name", func(t *testing.T) {
		_, err := client.BucketExists(ctx, "")
		if err == nil {
			t.Fatal("Expected error for empty bucket name")
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})
}

func TestRemoveBucket(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("Remove existing bucket", func(t *testing.T) {
		// Create bucket first
		err := client.MakeBucket(ctx, testBucket, MakeBucketOptions{
			Region: testRegion,
		})
		if err != nil && !IsBucketAlreadyExists(err) {
			t.Fatalf("Failed to create bucket: %v", err)
		}

		// Remove bucket
		err = client.RemoveBucket(ctx, testBucket)
		if err != nil {
			t.Fatalf("Failed to remove bucket: %v", err)
		}

		// Verify bucket is removed
		exists, err := client.BucketExists(ctx, testBucket)
		if err != nil {
			t.Fatalf("Failed to check bucket existence: %v", err)
		}

		if exists {
			t.Fatal("Expected bucket to be removed")
		}

		t.Log("✓ Bucket removed successfully")
	})

	t.Run("Remove non-existent bucket", func(t *testing.T) {
		err := client.RemoveBucket(ctx, "non-existent-bucket-xyz")
		if err == nil {
			t.Fatal("Expected error when removing non-existent bucket")
		}

		if !IsNotFound(err) {
			t.Fatalf("Expected NotFound error, got: %v", err)
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})

	t.Run("Remove with invalid name", func(t *testing.T) {
		err := client.RemoveBucket(ctx, "")
		if err == nil {
			t.Fatal("Expected error for empty bucket name")
		}

		t.Logf("✓ Error correctly returned: %v", err)
	})
}

func TestListObjects(t *testing.T) {
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

	// Upload test objects
	testObjects := []string{
		"test1.txt",
		"test2.txt",
		"folder/test3.txt",
		"folder/subfolder/test4.txt",
	}

	for _, objName := range testObjects {
		_, err := client.PutObject(ctx, testBucket, objName, nil, 0, PutObjectOptions{
			ContentType: "text/plain",
		})
		if err != nil {
			t.Fatalf("Failed to upload object %s: %v", objName, err)
		}
	}

	t.Run("List all objects recursively", func(t *testing.T) {
		objCh, errCh := client.ListObjects(ctx, testBucket, ListObjectsOptions{
			Recursive: true,
		})

		count := 0
		for {
			select {
			case obj, ok := <-objCh:
				if !ok {
					goto Done1
				}
				count++
				t.Logf("  - %s (size: %d, etag: %s)", obj.Key, obj.Size, obj.ETag)
			case err := <-errCh:
				if err != nil {
					t.Fatalf("Error listing objects: %v", err)
				}
			}
		}
	Done1:

		if count != len(testObjects) {
			t.Fatalf("Expected %d objects, got %d", len(testObjects), count)
		}

		t.Logf("✓ Listed %d objects recursively", count)
	})

	t.Run("List objects with prefix", func(t *testing.T) {
		objCh, errCh := client.ListObjects(ctx, testBucket, ListObjectsOptions{
			Prefix:    "folder/",
			Recursive: true,
		})

		count := 0
		for {
			select {
			case obj, ok := <-objCh:
				if !ok {
					goto Done2
				}
				count++
				t.Logf("  - %s", obj.Key)
			case err := <-errCh:
				if err != nil {
					t.Fatalf("Error listing objects: %v", err)
				}
			}
		}
	Done2:

		expectedCount := 2 // folder/test3.txt and folder/subfolder/test4.txt
		if count != expectedCount {
			t.Fatalf("Expected %d objects with prefix, got %d", expectedCount, count)
		}

		t.Logf("✓ Listed %d objects with prefix 'folder/'", count)
	})

	t.Run("List objects non-recursively", func(t *testing.T) {
		objCh, errCh := client.ListObjects(ctx, testBucket, ListObjectsOptions{
			Recursive: false,
		})

		count := 0
		for {
			select {
			case obj, ok := <-objCh:
				if !ok {
					goto Done3
				}
				count++
				t.Logf("  - %s (isDir: %v)", obj.Key, obj.IsDir)
			case err := <-errCh:
				if err != nil {
					t.Fatalf("Error listing objects: %v", err)
				}
			}
		}
	Done3:

		t.Logf("✓ Listed %d items non-recursively", count)
	})
}

func TestListIncompleteUploads(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	defer cleanupTestBucket(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create bucket
	err := client.MakeBucket(ctx, testBucket, MakeBucketOptions{
		Region: testRegion,
	})
	if err != nil && !IsBucketAlreadyExists(err) {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	mpCh, errCh := client.ListIncompleteUploads(ctx, testBucket, "", true)

	count := 0
	for {
		select {
		case mp, ok := <-mpCh:
			if !ok {
				goto Done
			}
			count++
			t.Logf("  - %s (upload_id: %s, size: %d)", mp.Key, mp.UploadID, mp.Size)
		case err := <-errCh:
			if err != nil {
				t.Fatalf("Error listing incomplete uploads: %v", err)
			}
		}
	}
Done:

	t.Logf("✓ Listed %d incomplete uploads", count)
}
