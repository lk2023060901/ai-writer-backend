package minio

import (
	"context"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

// PostPolicy represents a presigned POST policy
type PostPolicy struct {
	policy *minio.PostPolicy
}

// NewPostPolicy creates a new PostPolicy
func NewPostPolicy() *PostPolicy {
	return &PostPolicy{
		policy: minio.NewPostPolicy(),
	}
}

// SetBucket sets the bucket for the policy
func (p *PostPolicy) SetBucket(bucket string) error {
	return p.policy.SetBucket(bucket)
}

// SetKey sets the object key for the policy
func (p *PostPolicy) SetKey(key string) error {
	return p.policy.SetKey(key)
}

// SetExpires sets the expiration time for the policy
func (p *PostPolicy) SetExpires(expires time.Time) error {
	return p.policy.SetExpires(expires)
}

// SetContentType sets the content type for the policy
func (p *PostPolicy) SetContentType(contentType string) error {
	return p.policy.SetContentType(contentType)
}

// SetContentLengthRange sets the content length range for the policy
func (p *PostPolicy) SetContentLengthRange(min, max int64) error {
	return p.policy.SetContentLengthRange(min, max)
}

// SetUserMetadata sets user metadata for the policy
func (p *PostPolicy) SetUserMetadata(key, value string) error {
	return p.policy.SetUserMetadata(key, value)
}

// PresignedGetObject generates a presigned URL for HTTP GET operations
func (c *Client) PresignedGetObject(ctx context.Context, bucketName, objectName string, expiry time.Duration, reqParams url.Values) (*url.URL, error) {
	if err := c.checkClosed(); err != nil {
		return nil, err
	}

	if bucketName == "" {
		return nil, WrapError("PresignedGetObject", ErrInvalidBucketName, bucketName, objectName)
	}

	if objectName == "" {
		return nil, WrapError("PresignedGetObject", ErrInvalidObjectName, bucketName, objectName)
	}

	if expiry <= 0 {
		return nil, WrapErrorWithMessage("PresignedGetObject", ErrInvalidArgument, "expiry must be greater than 0")
	}

	presignedURL, err := c.client.PresignedGetObject(ctx, bucketName, objectName, expiry, reqParams)
	if err != nil {
		return nil, WrapError("PresignedGetObject", err, bucketName, objectName)
	}

	if c.logger != nil {
		c.logger.Info("presigned GET URL generated successfully",
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
			zap.Duration("expiry", expiry),
		)
	}

	return presignedURL, nil
}

// PresignedPutObject generates a presigned URL for HTTP PUT operations
func (c *Client) PresignedPutObject(ctx context.Context, bucketName, objectName string, expiry time.Duration) (*url.URL, error) {
	if err := c.checkClosed(); err != nil {
		return nil, err
	}

	if bucketName == "" {
		return nil, WrapError("PresignedPutObject", ErrInvalidBucketName, bucketName, objectName)
	}

	if objectName == "" {
		return nil, WrapError("PresignedPutObject", ErrInvalidObjectName, bucketName, objectName)
	}

	if expiry <= 0 {
		return nil, WrapErrorWithMessage("PresignedPutObject", ErrInvalidArgument, "expiry must be greater than 0")
	}

	presignedURL, err := c.client.PresignedPutObject(ctx, bucketName, objectName, expiry)
	if err != nil {
		return nil, WrapError("PresignedPutObject", err, bucketName, objectName)
	}

	if c.logger != nil {
		c.logger.Info("presigned PUT URL generated successfully",
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
			zap.Duration("expiry", expiry),
		)
	}

	return presignedURL, nil
}

// PresignedHeadObject generates a presigned URL for HTTP HEAD operations
func (c *Client) PresignedHeadObject(ctx context.Context, bucketName, objectName string, expiry time.Duration, reqParams url.Values) (*url.URL, error) {
	if err := c.checkClosed(); err != nil {
		return nil, err
	}

	if bucketName == "" {
		return nil, WrapError("PresignedHeadObject", ErrInvalidBucketName, bucketName, objectName)
	}

	if objectName == "" {
		return nil, WrapError("PresignedHeadObject", ErrInvalidObjectName, bucketName, objectName)
	}

	if expiry <= 0 {
		return nil, WrapErrorWithMessage("PresignedHeadObject", ErrInvalidArgument, "expiry must be greater than 0")
	}

	presignedURL, err := c.client.PresignedHeadObject(ctx, bucketName, objectName, expiry, reqParams)
	if err != nil {
		return nil, WrapError("PresignedHeadObject", err, bucketName, objectName)
	}

	if c.logger != nil {
		c.logger.Info("presigned HEAD URL generated successfully",
			zap.String("bucket", bucketName),
			zap.String("object", objectName),
			zap.Duration("expiry", expiry),
		)
	}

	return presignedURL, nil
}

// PresignedPostPolicy generates a presigned POST policy for HTTP POST operations
func (c *Client) PresignedPostPolicy(ctx context.Context, policy *PostPolicy) (*url.URL, map[string]string, error) {
	if err := c.checkClosed(); err != nil {
		return nil, nil, err
	}

	if policy == nil {
		return nil, nil, WrapErrorWithMessage("PresignedPostPolicy", ErrInvalidArgument, "policy is required")
	}

	presignedURL, formData, err := c.client.PresignedPostPolicy(ctx, policy.policy)
	if err != nil {
		return nil, nil, WrapErrorWithMessage("PresignedPostPolicy", err, "failed to generate presigned POST policy")
	}

	if c.logger != nil {
		c.logger.Info("presigned POST policy generated successfully")
	}

	return presignedURL, formData, nil
}
