package minio

import (
	"context"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

// BucketInfo represents bucket information
type BucketInfo struct {
	Name         string
	CreationDate string
}

// MakeBucketOptions represents options for creating a bucket
type MakeBucketOptions struct {
	// Region is the region where the bucket will be created
	Region string
	// ObjectLocking enables object locking for the bucket
	ObjectLocking bool
}

// ListObjectsOptions represents options for listing objects
type ListObjectsOptions struct {
	// Prefix filters objects by prefix
	Prefix string
	// Recursive lists objects recursively
	Recursive bool
	// MaxKeys limits the number of objects returned (0 = unlimited)
	MaxKeys int
	// StartAfter starts listing after this object key
	StartAfter string
	// UseV1 uses ListObjects V1 API (default is V2)
	UseV1 bool
}

// ObjectInfo represents object information
type ObjectInfo struct {
	Key          string
	Size         int64
	ETag         string
	LastModified string
	ContentType  string
	StorageClass string
	IsDir        bool
	Metadata     map[string]string
}

// MultipartInfo represents incomplete multipart upload information
type MultipartInfo struct {
	Key          string
	UploadID     string
	Size         int64
	Initiated    string
	StorageClass string
}

// MakeBucket creates a new bucket
func (c *Client) MakeBucket(ctx context.Context, bucketName string, opts MakeBucketOptions) error {
	if err := c.checkClosed(); err != nil {
		return err
	}

	if bucketName == "" {
		return WrapError("MakeBucket", ErrInvalidBucketName, bucketName, "")
	}

	minioOpts := minio.MakeBucketOptions{
		Region:        opts.Region,
		ObjectLocking: opts.ObjectLocking,
	}

	err := c.client.MakeBucket(ctx, bucketName, minioOpts)
	if err != nil {
		return WrapError("MakeBucket", err, bucketName, "")
	}

	if c.logger != nil {
		c.logger.Info("bucket created successfully",
			zap.String("bucket", bucketName),
			zap.String("region", opts.Region),
			zap.Bool("object_locking", opts.ObjectLocking),
		)
	}

	return nil
}

// ListBuckets lists all buckets
func (c *Client) ListBuckets(ctx context.Context) ([]BucketInfo, error) {
	if err := c.checkClosed(); err != nil {
		return nil, err
	}

	buckets, err := c.client.ListBuckets(ctx)
	if err != nil {
		return nil, WrapErrorWithMessage("ListBuckets", err, "failed to list buckets")
	}

	result := make([]BucketInfo, 0, len(buckets))
	for _, bucket := range buckets {
		result = append(result, BucketInfo{
			Name:         bucket.Name,
			CreationDate: bucket.CreationDate.Format("2006-01-02 15:04:05"),
		})
	}

	return result, nil
}

// BucketExists checks if a bucket exists
func (c *Client) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	if err := c.checkClosed(); err != nil {
		return false, err
	}

	if bucketName == "" {
		return false, WrapError("BucketExists", ErrInvalidBucketName, bucketName, "")
	}

	exists, err := c.client.BucketExists(ctx, bucketName)
	if err != nil {
		return false, WrapError("BucketExists", err, bucketName, "")
	}

	return exists, nil
}

// RemoveBucket removes a bucket
func (c *Client) RemoveBucket(ctx context.Context, bucketName string) error {
	if err := c.checkClosed(); err != nil {
		return err
	}

	if bucketName == "" {
		return WrapError("RemoveBucket", ErrInvalidBucketName, bucketName, "")
	}

	err := c.client.RemoveBucket(ctx, bucketName)
	if err != nil {
		return WrapError("RemoveBucket", err, bucketName, "")
	}

	if c.logger != nil {
		c.logger.Info("bucket removed successfully", zap.String("bucket", bucketName))
	}

	return nil
}

// ListObjects lists objects in a bucket
func (c *Client) ListObjects(ctx context.Context, bucketName string, opts ListObjectsOptions) (<-chan ObjectInfo, <-chan error) {
	objCh := make(chan ObjectInfo)
	errCh := make(chan error, 1)

	go func() {
		defer close(objCh)
		defer close(errCh)

		if err := c.checkClosed(); err != nil {
			errCh <- err
			return
		}

		if bucketName == "" {
			errCh <- WrapError("ListObjects", ErrInvalidBucketName, bucketName, "")
			return
		}

		minioOpts := minio.ListObjectsOptions{
			Prefix:     opts.Prefix,
			Recursive:  opts.Recursive,
			MaxKeys:    opts.MaxKeys,
			StartAfter: opts.StartAfter,
			UseV1:      opts.UseV1,
		}

		for object := range c.client.ListObjects(ctx, bucketName, minioOpts) {
			if object.Err != nil {
				errCh <- WrapError("ListObjects", object.Err, bucketName, "")
				return
			}

			objInfo := ObjectInfo{
				Key:          object.Key,
				Size:         object.Size,
				ETag:         object.ETag,
				LastModified: object.LastModified.Format("2006-01-02 15:04:05"),
				ContentType:  object.ContentType,
				StorageClass: object.StorageClass,
				IsDir:        object.Key != "" && object.Key[len(object.Key)-1] == '/',
				Metadata:     object.UserMetadata,
			}

			select {
			case objCh <- objInfo:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
		}
	}()

	return objCh, errCh
}

// ListIncompleteUploads lists incomplete multipart uploads in a bucket
func (c *Client) ListIncompleteUploads(ctx context.Context, bucketName, prefix string, recursive bool) (<-chan MultipartInfo, <-chan error) {
	mpCh := make(chan MultipartInfo)
	errCh := make(chan error, 1)

	go func() {
		defer close(mpCh)
		defer close(errCh)

		if err := c.checkClosed(); err != nil {
			errCh <- err
			return
		}

		if bucketName == "" {
			errCh <- WrapError("ListIncompleteUploads", ErrInvalidBucketName, bucketName, "")
			return
		}

		for upload := range c.client.ListIncompleteUploads(ctx, bucketName, prefix, recursive) {
			if upload.Err != nil {
				errCh <- WrapError("ListIncompleteUploads", upload.Err, bucketName, "")
				return
			}

			mpInfo := MultipartInfo{
				Key:          upload.Key,
				UploadID:     upload.UploadID,
				Size:         upload.Size,
				Initiated:    upload.Initiated.Format("2006-01-02 15:04:05"),
				StorageClass: upload.StorageClass,
			}

			select {
			case mpCh <- mpInfo:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
		}
	}()

	return mpCh, errCh
}
