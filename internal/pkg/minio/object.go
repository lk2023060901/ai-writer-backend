package minio

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

// PutObjectOptions represents options for uploading an object
type PutObjectOptions struct {
	// ContentType is the content type of the object
	ContentType string
	// UserMetadata is custom metadata for the object
	UserMetadata map[string]string
	// CacheControl sets the cache control header
	CacheControl string
	// ContentDisposition sets the content disposition header
	ContentDisposition string
	// ContentEncoding sets the content encoding header
	ContentEncoding string
	// ContentLanguage sets the content language header
	ContentLanguage string
	// StorageClass sets the storage class (e.g., "STANDARD", "REDUCED_REDUNDANCY")
	StorageClass string
	// Progress is a reader to track upload progress
	Progress io.Reader
}

// GetObjectOptions represents options for downloading an object
type GetObjectOptions struct {
	// VersionID specifies the version of the object to retrieve
	VersionID string
	// PartNumber specifies the part number to retrieve
	PartNumber int
}

// StatObjectOptions represents options for getting object metadata
type StatObjectOptions struct {
	// VersionID specifies the version of the object
	VersionID string
}

// RemoveObjectOptions represents options for removing an object
type RemoveObjectOptions struct {
	// VersionID specifies the version of the object to remove
	VersionID string
	// ForceDelete forces deletion even if object is locked
	ForceDelete bool
}

// CopyDestOptions represents destination options for copying an object
type CopyDestOptions struct {
	// Bucket is the destination bucket name
	Bucket string
	// Object is the destination object name
	Object string
	// UserMetadata is custom metadata for the destination object
	UserMetadata map[string]string
	// ReplaceMetadata replaces source metadata with new metadata
	ReplaceMetadata bool
}

// CopySrcOptions represents source options for copying an object
type CopySrcOptions struct {
	// Bucket is the source bucket name
	Bucket string
	// Object is the source object name
	Object string
	// VersionID is the source object version ID
	VersionID string
}

// UploadInfo represents information about an uploaded object
type UploadInfo struct {
	Bucket       string
	Key          string
	ETag         string
	Size         int64
	LastModified string
	Location     string
	VersionID    string
}

// PutObject uploads an object to a bucket
func (c *Client) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts PutObjectOptions) (UploadInfo, error) {
	if err := c.checkClosed(); err != nil {
		return UploadInfo{}, err
	}

	if bucketName == "" {
		return UploadInfo{}, WrapError("PutObject", ErrInvalidBucketName, bucketName, objectName)
	}

	if objectName == "" {
		return UploadInfo{}, WrapError("PutObject", ErrInvalidObjectName, bucketName, objectName)
	}

	minioOpts := minio.PutObjectOptions{
		ContentType:        opts.ContentType,
		UserMetadata:       opts.UserMetadata,
		CacheControl:       opts.CacheControl,
		ContentDisposition: opts.ContentDisposition,
		ContentEncoding:    opts.ContentEncoding,
		ContentLanguage:    opts.ContentLanguage,
		StorageClass:       opts.StorageClass,
		Progress:           opts.Progress,
	}

	info, err := c.client.PutObject(ctx, bucketName, objectName, reader, objectSize, minioOpts)
	if err != nil {
		return UploadInfo{}, WrapError("PutObject", err, bucketName, objectName)
	}

	if c.logger != nil {
		c.logger.Info("object uploaded successfully",
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
			zap.Int64("size", info.Size),
			zap.String("etag", info.ETag),
		)
	}

	return UploadInfo{
		Bucket:       info.Bucket,
		Key:          info.Key,
		ETag:         info.ETag,
		Size:         info.Size,
		LastModified: info.LastModified.Format("2006-01-02 15:04:05"),
		Location:     info.Location,
		VersionID:    info.VersionID,
	}, nil
}

// GetObject downloads an object from a bucket
func (c *Client) GetObject(ctx context.Context, bucketName, objectName string, opts GetObjectOptions) (*minio.Object, error) {
	if err := c.checkClosed(); err != nil {
		return nil, err
	}

	if bucketName == "" {
		return nil, WrapError("GetObject", ErrInvalidBucketName, bucketName, objectName)
	}

	if objectName == "" {
		return nil, WrapError("GetObject", ErrInvalidObjectName, bucketName, objectName)
	}

	minioOpts := minio.GetObjectOptions{}
	if opts.VersionID != "" {
		minioOpts.VersionID = opts.VersionID
	}
	if opts.PartNumber > 0 {
		minioOpts.PartNumber = opts.PartNumber
	}

	object, err := c.client.GetObject(ctx, bucketName, objectName, minioOpts)
	if err != nil {
		return nil, WrapError("GetObject", err, bucketName, objectName)
	}

	return object, nil
}

// FPutObject uploads a file to a bucket
func (c *Client) FPutObject(ctx context.Context, bucketName, objectName, filePath string, opts PutObjectOptions) (UploadInfo, error) {
	if err := c.checkClosed(); err != nil {
		return UploadInfo{}, err
	}

	if bucketName == "" {
		return UploadInfo{}, WrapError("FPutObject", ErrInvalidBucketName, bucketName, objectName)
	}

	if objectName == "" {
		return UploadInfo{}, WrapError("FPutObject", ErrInvalidObjectName, bucketName, objectName)
	}

	if filePath == "" {
		return UploadInfo{}, WrapErrorWithMessage("FPutObject", ErrInvalidArgument, "file path is required")
	}

	minioOpts := minio.PutObjectOptions{
		ContentType:        opts.ContentType,
		UserMetadata:       opts.UserMetadata,
		CacheControl:       opts.CacheControl,
		ContentDisposition: opts.ContentDisposition,
		ContentEncoding:    opts.ContentEncoding,
		ContentLanguage:    opts.ContentLanguage,
		StorageClass:       opts.StorageClass,
		Progress:           opts.Progress,
	}

	info, err := c.client.FPutObject(ctx, bucketName, objectName, filePath, minioOpts)
	if err != nil {
		return UploadInfo{}, WrapError("FPutObject", err, bucketName, objectName)
	}

	if c.logger != nil {
		c.logger.Info("file uploaded successfully",
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
			zap.String("file_path", filePath),
			zap.Int64("size", info.Size),
		)
	}

	return UploadInfo{
		Bucket:       info.Bucket,
		Key:          info.Key,
		ETag:         info.ETag,
		Size:         info.Size,
		LastModified: info.LastModified.Format("2006-01-02 15:04:05"),
		Location:     info.Location,
		VersionID:    info.VersionID,
	}, nil
}

// FGetObject downloads an object to a file
func (c *Client) FGetObject(ctx context.Context, bucketName, objectName, filePath string, opts GetObjectOptions) error {
	if err := c.checkClosed(); err != nil {
		return err
	}

	if bucketName == "" {
		return WrapError("FGetObject", ErrInvalidBucketName, bucketName, objectName)
	}

	if objectName == "" {
		return WrapError("FGetObject", ErrInvalidObjectName, bucketName, objectName)
	}

	if filePath == "" {
		return WrapErrorWithMessage("FGetObject", ErrInvalidArgument, "file path is required")
	}

	minioOpts := minio.GetObjectOptions{}
	if opts.VersionID != "" {
		minioOpts.VersionID = opts.VersionID
	}

	err := c.client.FGetObject(ctx, bucketName, objectName, filePath, minioOpts)
	if err != nil {
		return WrapError("FGetObject", err, bucketName, objectName)
	}

	if c.logger != nil {
		c.logger.Info("file downloaded successfully",
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
			zap.String("file_path", filePath),
		)
	}

	return nil
}

// StatObject gets object metadata
func (c *Client) StatObject(ctx context.Context, bucketName, objectName string, opts StatObjectOptions) (ObjectInfo, error) {
	if err := c.checkClosed(); err != nil {
		return ObjectInfo{}, err
	}

	if bucketName == "" {
		return ObjectInfo{}, WrapError("StatObject", ErrInvalidBucketName, bucketName, objectName)
	}

	if objectName == "" {
		return ObjectInfo{}, WrapError("StatObject", ErrInvalidObjectName, bucketName, objectName)
	}

	minioOpts := minio.StatObjectOptions{}
	if opts.VersionID != "" {
		minioOpts.VersionID = opts.VersionID
	}

	info, err := c.client.StatObject(ctx, bucketName, objectName, minioOpts)
	if err != nil {
		return ObjectInfo{}, WrapError("StatObject", err, bucketName, objectName)
	}

	return ObjectInfo{
		Key:          info.Key,
		Size:         info.Size,
		ETag:         info.ETag,
		LastModified: info.LastModified.Format("2006-01-02 15:04:05"),
		ContentType:  info.ContentType,
		Metadata:     info.UserMetadata,
	}, nil
}

// RemoveObject removes an object from a bucket
func (c *Client) RemoveObject(ctx context.Context, bucketName, objectName string, opts RemoveObjectOptions) error {
	if err := c.checkClosed(); err != nil {
		return err
	}

	if bucketName == "" {
		return WrapError("RemoveObject", ErrInvalidBucketName, bucketName, objectName)
	}

	if objectName == "" {
		return WrapError("RemoveObject", ErrInvalidObjectName, bucketName, objectName)
	}

	minioOpts := minio.RemoveObjectOptions{
		VersionID:   opts.VersionID,
		ForceDelete: opts.ForceDelete,
	}

	err := c.client.RemoveObject(ctx, bucketName, objectName, minioOpts)
	if err != nil {
		return WrapError("RemoveObject", err, bucketName, objectName)
	}

	if c.logger != nil {
		c.logger.Info("object removed successfully",
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
		)
	}

	return nil
}

// CopyObject copies an object from source to destination
func (c *Client) CopyObject(ctx context.Context, dst CopyDestOptions, src CopySrcOptions) (UploadInfo, error) {
	if err := c.checkClosed(); err != nil {
		return UploadInfo{}, err
	}

	if dst.Bucket == "" || dst.Object == "" {
		return UploadInfo{}, WrapErrorWithMessage("CopyObject", ErrInvalidArgument, "destination bucket and object are required")
	}

	if src.Bucket == "" || src.Object == "" {
		return UploadInfo{}, WrapErrorWithMessage("CopyObject", ErrInvalidArgument, "source bucket and object are required")
	}

	// Build source options
	srcOpts := minio.CopySrcOptions{
		Bucket: src.Bucket,
		Object: src.Object,
	}
	if src.VersionID != "" {
		srcOpts.VersionID = src.VersionID
	}

	// Build destination options
	dstOpts := minio.CopyDestOptions{
		Bucket:          dst.Bucket,
		Object:          dst.Object,
		UserMetadata:    dst.UserMetadata,
		ReplaceMetadata: dst.ReplaceMetadata,
	}

	info, err := c.client.CopyObject(ctx, dstOpts, srcOpts)
	if err != nil {
		return UploadInfo{}, WrapErrorWithMessage("CopyObject", err, "failed to copy object")
	}

	if c.logger != nil {
		c.logger.Info("object copied successfully",
			zap.String("src_bucket", src.Bucket),
			zap.String("src_object", src.Object),
			zap.String("dst_bucket", dst.Bucket),
			zap.String("dst_object", dst.Object),
		)
	}

	return UploadInfo{
		Bucket:       info.Bucket,
		Key:          info.Key,
		ETag:         info.ETag,
		Size:         info.Size,
		LastModified: info.LastModified.Format("2006-01-02 15:04:05"),
		Location:     info.Location,
		VersionID:    info.VersionID,
	}, nil
}

// RemoveIncompleteUpload removes an incomplete multipart upload
func (c *Client) RemoveIncompleteUpload(ctx context.Context, bucketName, objectName string) error {
	if err := c.checkClosed(); err != nil {
		return err
	}

	if bucketName == "" {
		return WrapError("RemoveIncompleteUpload", ErrInvalidBucketName, bucketName, objectName)
	}

	if objectName == "" {
		return WrapError("RemoveIncompleteUpload", ErrInvalidObjectName, bucketName, objectName)
	}

	err := c.client.RemoveIncompleteUpload(ctx, bucketName, objectName)
	if err != nil {
		return WrapError("RemoveIncompleteUpload", err, bucketName, objectName)
	}

	if c.logger != nil {
		c.logger.Info("incomplete upload removed successfully",
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
		)
	}

	return nil
}
