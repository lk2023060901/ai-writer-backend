package milvus

import (
	"errors"
	"fmt"
)

// Predefined errors
var (
	// ErrCollectionNotFound indicates that the collection does not exist
	ErrCollectionNotFound = errors.New("milvus: collection not found")

	// ErrCollectionExists indicates that the collection already exists
	ErrCollectionExists = errors.New("milvus: collection already exists")

	// ErrPartitionNotFound indicates that the partition does not exist
	ErrPartitionNotFound = errors.New("milvus: partition not found")

	// ErrPartitionExists indicates that the partition already exists
	ErrPartitionExists = errors.New("milvus: partition already exists")

	// ErrIndexNotFound indicates that the index does not exist
	ErrIndexNotFound = errors.New("milvus: index not found")

	// ErrIndexExists indicates that the index already exists
	ErrIndexExists = errors.New("milvus: index already exists")

	// ErrInvalidVectorDim indicates that the vector dimension is invalid
	ErrInvalidVectorDim = errors.New("milvus: invalid vector dimension")

	// ErrInvalidMetricType indicates that the metric type is invalid
	ErrInvalidMetricType = errors.New("milvus: invalid metric type")

	// ErrInvalidIndexType indicates that the index type is invalid
	ErrInvalidIndexType = errors.New("milvus: invalid index type")

	// ErrInvalidCollectionName indicates that the collection name is invalid
	ErrInvalidCollectionName = errors.New("milvus: invalid collection name")

	// ErrInvalidFieldName indicates that the field name is invalid
	ErrInvalidFieldName = errors.New("milvus: invalid field name")

	// ErrInvalidArgument indicates that an argument is invalid
	ErrInvalidArgument = errors.New("milvus: invalid argument")

	// ErrInvalidConfig indicates that the configuration is invalid
	ErrInvalidConfig = errors.New("milvus: invalid config")

	// ErrInvalidSchema indicates that the schema is invalid
	ErrInvalidSchema = errors.New("milvus: invalid schema")

	// ErrConnectionFailed indicates that the connection to Milvus failed
	ErrConnectionFailed = errors.New("milvus: connection failed")

	// ErrOperationTimeout indicates that an operation timed out
	ErrOperationTimeout = errors.New("milvus: operation timeout")

	// ErrClientClosed indicates that the client is closed
	ErrClientClosed = errors.New("milvus: client is closed")

	// ErrEmptyCollection indicates that the collection is empty
	ErrEmptyCollection = errors.New("milvus: collection is empty")

	// ErrNoVectorField indicates that no vector field is found
	ErrNoVectorField = errors.New("milvus: no vector field found")

	// ErrNoPrimaryKey indicates that no primary key field is found
	ErrNoPrimaryKey = errors.New("milvus: no primary key field found")

	// ErrDuplicateField indicates that duplicate field names exist
	ErrDuplicateField = errors.New("milvus: duplicate field name")

	// ErrIndexNotReady indicates that the index is not ready
	ErrIndexNotReady = errors.New("milvus: index not ready")

	// ErrCollectionNotLoaded indicates that the collection is not loaded
	ErrCollectionNotLoaded = errors.New("milvus: collection not loaded")

	// ErrInvalidDataType indicates that the data type is invalid
	ErrInvalidDataType = errors.New("milvus: invalid data type")

	// ErrMismatchedVectorDim indicates that vector dimensions do not match
	ErrMismatchedVectorDim = errors.New("milvus: mismatched vector dimension")

	// ErrInvalidPartitionName indicates that the partition name is invalid
	ErrInvalidPartitionName = errors.New("milvus: invalid partition name")

	// ErrInvalidIndexParams indicates that the index parameters are invalid
	ErrInvalidIndexParams = errors.New("milvus: invalid index parameters")

	// ErrInvalidData indicates that the data is invalid
	ErrInvalidData = errors.New("milvus: invalid data")

	// ErrInvalidVectorData indicates that the vector data is invalid
	ErrInvalidVectorData = errors.New("milvus: invalid vector data")

	// ErrInvalidExpression indicates that the expression is invalid
	ErrInvalidExpression = errors.New("milvus: invalid expression")

	// ErrInvalidIDs indicates that the IDs are invalid
	ErrInvalidIDs = errors.New("milvus: invalid IDs")

	// ErrInvalidSearchRequest indicates that the search request is invalid
	ErrInvalidSearchRequest = errors.New("milvus: invalid search request")

	// ErrTimeout indicates a timeout error
	ErrTimeout = errors.New("milvus: timeout")

	// ErrNotImplemented indicates that the feature is not implemented
	ErrNotImplemented = errors.New("milvus: not implemented")
)

// Error represents a Milvus error with additional context
type Error struct {
	Op         string // Operation that failed
	Collection string // Collection name (if applicable)
	Field      string // Field name (if applicable)
	Err        error  // Original error
	Message    string // Additional message
}

// Error returns the error message
func (e *Error) Error() string {
	var msg string

	if e.Collection != "" && e.Field != "" {
		msg = fmt.Sprintf("milvus: %s failed for collection=%s, field=%s", e.Op, e.Collection, e.Field)
	} else if e.Collection != "" {
		msg = fmt.Sprintf("milvus: %s failed for collection=%s", e.Op, e.Collection)
	} else if e.Field != "" {
		msg = fmt.Sprintf("milvus: %s failed for field=%s", e.Op, e.Field)
	} else {
		msg = fmt.Sprintf("milvus: %s failed", e.Op)
	}

	if e.Message != "" {
		msg = fmt.Sprintf("%s: %s", msg, e.Message)
	}

	if e.Err != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.Err)
	}

	return msg
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

	if errors.Is(err, ErrCollectionNotFound) ||
		errors.Is(err, ErrPartitionNotFound) ||
		errors.Is(err, ErrIndexNotFound) {
		return true
	}

	// Check error message for common "not found" patterns
	errMsg := err.Error()
	return containsAny(errMsg, []string{
		"not found",
		"not exist",
		"doesn't exist",
		"does not exist",
	})
}

// IsAlreadyExists checks if the error is an "already exists" error
func IsAlreadyExists(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrCollectionExists) ||
		errors.Is(err, ErrPartitionExists) ||
		errors.Is(err, ErrIndexExists) {
		return true
	}

	// Check error message for common "already exists" patterns
	errMsg := err.Error()
	return containsAny(errMsg, []string{
		"already exist",
		"already exists",
		"duplicate",
	})
}

// IsInvalidArgument checks if the error is an "invalid argument" error
func IsInvalidArgument(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrInvalidArgument) ||
		errors.Is(err, ErrInvalidCollectionName) ||
		errors.Is(err, ErrInvalidFieldName) ||
		errors.Is(err, ErrInvalidVectorDim) ||
		errors.Is(err, ErrInvalidMetricType) ||
		errors.Is(err, ErrInvalidIndexType) ||
		errors.Is(err, ErrInvalidDataType) {
		return true
	}

	// Check error message for common "invalid" patterns
	errMsg := err.Error()
	return containsAny(errMsg, []string{
		"invalid",
		"illegal",
		"bad",
		"malformed",
	})
}

// IsTimeout checks if the error is a timeout error
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrOperationTimeout) {
		return true
	}

	// Check error message for timeout patterns
	errMsg := err.Error()
	return containsAny(errMsg, []string{
		"timeout",
		"timed out",
		"deadline exceeded",
		"context deadline exceeded",
	})
}

// IsConnectionError checks if the error is a connection error
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrConnectionFailed) {
		return true
	}

	// Check error message for connection error patterns
	errMsg := err.Error()
	return containsAny(errMsg, []string{
		"connection",
		"connect",
		"dial",
		"network",
		"unreachable",
	})
}

// IsConnectionFailed 是 IsConnectionError 的别名
func IsConnectionFailed(err error) bool {
	return IsConnectionError(err)
}

// WrapError wraps an error with operation and collection context
func WrapError(op string, err error, collection, field string) error {
	if err == nil {
		return nil
	}

	return &Error{
		Op:         op,
		Collection: collection,
		Field:      field,
		Err:        err,
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

// NewError creates a new error with operation and message
func NewError(op, message string) error {
	return &Error{
		Op:      op,
		Message: message,
	}
}

// containsAny checks if the string contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
