package minio

import (
	"context"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/tags"
	"go.uber.org/zap"
)

// SetBucketTagging sets tags for a bucket
func (c *Client) SetBucketTagging(ctx context.Context, bucketName string, tagMap map[string]string) error {
	if err := c.checkClosed(); err != nil {
		return err
	}

	if bucketName == "" {
		return WrapError("SetBucketTagging", ErrInvalidBucketName, bucketName, "")
	}

	if len(tagMap) == 0 {
		return WrapErrorWithMessage("SetBucketTagging", ErrInvalidArgument, "tags cannot be empty")
	}

	// Create tags from map
	bucketTags, err := tags.NewTags(tagMap, false)
	if err != nil {
		return WrapError("SetBucketTagging", err, bucketName, "")
	}

	err = c.client.SetBucketTagging(ctx, bucketName, bucketTags)
	if err != nil {
		return WrapError("SetBucketTagging", err, bucketName, "")
	}

	if c.logger != nil {
		c.logger.Info("bucket tags set successfully",
			zap.String("bucket", bucketName),
			zap.Any("tags", tagMap),
		)
	}

	return nil
}

// GetBucketTagging gets tags for a bucket
func (c *Client) GetBucketTagging(ctx context.Context, bucketName string) (map[string]string, error) {
	if err := c.checkClosed(); err != nil {
		return nil, err
	}

	if bucketName == "" {
		return nil, WrapError("GetBucketTagging", ErrInvalidBucketName, bucketName, "")
	}

	bucketTags, err := c.client.GetBucketTagging(ctx, bucketName)
	if err != nil {
		return nil, WrapError("GetBucketTagging", err, bucketName, "")
	}

	return bucketTags.ToMap(), nil
}

// RemoveBucketTagging removes all tags from a bucket
func (c *Client) RemoveBucketTagging(ctx context.Context, bucketName string) error {
	if err := c.checkClosed(); err != nil {
		return err
	}

	if bucketName == "" {
		return WrapError("RemoveBucketTagging", ErrInvalidBucketName, bucketName, "")
	}

	err := c.client.RemoveBucketTagging(ctx, bucketName)
	if err != nil {
		return WrapError("RemoveBucketTagging", err, bucketName, "")
	}

	if c.logger != nil {
		c.logger.Info("bucket tags removed successfully", zap.String("bucket", bucketName))
	}

	return nil
}

// PutObjectTagging sets tags for an object
func (c *Client) PutObjectTagging(ctx context.Context, bucketName, objectName string, tagMap map[string]string) error {
	if err := c.checkClosed(); err != nil {
		return err
	}

	if bucketName == "" {
		return WrapError("PutObjectTagging", ErrInvalidBucketName, bucketName, objectName)
	}

	if objectName == "" {
		return WrapError("PutObjectTagging", ErrInvalidObjectName, bucketName, objectName)
	}

	if len(tagMap) == 0 {
		return WrapError("PutObjectTagging", ErrInvalidArgument, bucketName, objectName)
	}

	// Create tags from map
	objectTags, err := tags.NewTags(tagMap, false)
	if err != nil {
		return WrapError("PutObjectTagging", err, bucketName, objectName)
	}

	err = c.client.PutObjectTagging(ctx, bucketName, objectName, objectTags, minio.PutObjectTaggingOptions{})
	if err != nil {
		return WrapError("PutObjectTagging", err, bucketName, objectName)
	}

	if c.logger != nil {
		c.logger.Info("object tags set successfully",
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
			zap.Any("tags", tagMap),
		)
	}

	return nil
}

// GetObjectTagging gets tags for an object
func (c *Client) GetObjectTagging(ctx context.Context, bucketName, objectName string) (map[string]string, error) {
	if err := c.checkClosed(); err != nil {
		return nil, err
	}

	if bucketName == "" {
		return nil, WrapError("GetObjectTagging", ErrInvalidBucketName, bucketName, objectName)
	}

	if objectName == "" {
		return nil, WrapError("GetObjectTagging", ErrInvalidObjectName, bucketName, objectName)
	}

	objectTags, err := c.client.GetObjectTagging(ctx, bucketName, objectName, minio.GetObjectTaggingOptions{})
	if err != nil {
		return nil, WrapError("GetObjectTagging", err, bucketName, objectName)
	}

	return objectTags.ToMap(), nil
}

// RemoveObjectTagging removes all tags from an object
func (c *Client) RemoveObjectTagging(ctx context.Context, bucketName, objectName string) error {
	if err := c.checkClosed(); err != nil {
		return err
	}

	if bucketName == "" {
		return WrapError("RemoveObjectTagging", ErrInvalidBucketName, bucketName, objectName)
	}

	if objectName == "" {
		return WrapError("RemoveObjectTagging", ErrInvalidObjectName, bucketName, objectName)
	}

	err := c.client.RemoveObjectTagging(ctx, bucketName, objectName, minio.RemoveObjectTaggingOptions{})
	if err != nil {
		return WrapError("RemoveObjectTagging", err, bucketName, objectName)
	}

	if c.logger != nil {
		c.logger.Info("object tags removed successfully",
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
		)
	}

	return nil
}
