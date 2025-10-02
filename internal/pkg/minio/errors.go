package minio

import (
	"errors"
	"fmt"

	"github.com/minio/minio-go/v7"
)

// Predefined errors
var (
	// ErrBucketNotFound indicates that the bucket does not exist
	ErrBucketNotFound = errors.New("minio: bucket not found")

	// ErrObjectNotFound indicates that the object does not exist
	ErrObjectNotFound = errors.New("minio: object not found")

	// ErrInvalidArgument indicates that an argument is invalid
	ErrInvalidArgument = errors.New("minio: invalid argument")

	// ErrAccessDenied indicates that access is denied
	ErrAccessDenied = errors.New("minio: access denied")

	// ErrBucketAlreadyExists indicates that the bucket already exists
	ErrBucketAlreadyExists = errors.New("minio: bucket already exists")

	// ErrBucketAlreadyOwnedByYou indicates that the bucket is already owned by you
	ErrBucketAlreadyOwnedByYou = errors.New("minio: bucket already owned by you")

	// ErrInvalidBucketName indicates that the bucket name is invalid
	ErrInvalidBucketName = errors.New("minio: invalid bucket name")

	// ErrInvalidObjectName indicates that the object name is invalid
	ErrInvalidObjectName = errors.New("minio: invalid object name")

	// ErrConnectionFailed indicates that the connection to MinIO failed
	ErrConnectionFailed = errors.New("minio: connection failed")

	// ErrOperationTimeout indicates that an operation timed out
	ErrOperationTimeout = errors.New("minio: operation timeout")
)

// Error represents a MinIO error with additional context
type Error struct {
	Op      string // Operation that failed
	Err     error  // Original error
	Bucket  string // Bucket name (if applicable)
	Object  string // Object name (if applicable)
	Message string // Additional message
}

// Error returns the error message
func (e *Error) Error() string {
	if e.Bucket != "" && e.Object != "" {
		return fmt.Sprintf("minio: %s failed for bucket=%s, object=%s: %v", e.Op, e.Bucket, e.Object, e.Err)
	} else if e.Bucket != "" {
		return fmt.Sprintf("minio: %s failed for bucket=%s: %v", e.Op, e.Bucket, e.Err)
	} else if e.Object != "" {
		return fmt.Sprintf("minio: %s failed for object=%s: %v", e.Op, e.Object, e.Err)
	}

	if e.Message != "" {
		return fmt.Sprintf("minio: %s failed: %s: %v", e.Op, e.Message, e.Err)
	}

	return fmt.Sprintf("minio: %s failed: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Err
}

// IsNotFound checks if the error is a "not found" error
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrBucketNotFound) || errors.Is(err, ErrObjectNotFound) {
		return true
	}

	// Check MinIO error response
	var minioErr minio.ErrorResponse
	if errors.As(err, &minioErr) {
		return minioErr.Code == "NoSuchBucket" ||
			minioErr.Code == "NoSuchKey" ||
			minioErr.Code == "NoSuchUpload"
	}

	return false
}

// IsAccessDenied checks if the error is an "access denied" error
func IsAccessDenied(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrAccessDenied) {
		return true
	}

	// Check MinIO error response
	var minioErr minio.ErrorResponse
	if errors.As(err, &minioErr) {
		return minioErr.Code == "AccessDenied" ||
			minioErr.Code == "Forbidden"
	}

	return false
}

// IsBucketAlreadyExists checks if the error is a "bucket already exists" error
func IsBucketAlreadyExists(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrBucketAlreadyExists) || errors.Is(err, ErrBucketAlreadyOwnedByYou) {
		return true
	}

	// Check MinIO error response
	var minioErr minio.ErrorResponse
	if errors.As(err, &minioErr) {
		return minioErr.Code == "BucketAlreadyExists" ||
			minioErr.Code == "BucketAlreadyOwnedByYou"
	}

	return false
}

// IsInvalidArgument checks if the error is an "invalid argument" error
func IsInvalidArgument(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrInvalidArgument) ||
		errors.Is(err, ErrInvalidBucketName) ||
		errors.Is(err, ErrInvalidObjectName) {
		return true
	}

	// Check MinIO error response
	var minioErr minio.ErrorResponse
	if errors.As(err, &minioErr) {
		return minioErr.Code == "InvalidArgument" ||
			minioErr.Code == "InvalidBucketName" ||
			minioErr.Code == "InvalidObjectName"
	}

	return false
}

// WrapError wraps an error with operation context
func WrapError(op string, err error, bucket, object string) error {
	if err == nil {
		return nil
	}

	return &Error{
		Op:     op,
		Err:    err,
		Bucket: bucket,
		Object: object,
	}
}

// WrapErrorWithMessage wraps an error with operation context and a message
func WrapErrorWithMessage(op string, err error, message string) error {
	if err == nil {
		return nil
	}

	return &Error{
		Op:      op,
		Err:     err,
		Message: message,
	}
}
