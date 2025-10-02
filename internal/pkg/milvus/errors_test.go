package milvus

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrCollectionNotFound", ErrCollectionNotFound},
		{"ErrCollectionExists", ErrCollectionExists},
		{"ErrPartitionNotFound", ErrPartitionNotFound},
		{"ErrPartitionExists", ErrPartitionExists},
		{"ErrInvalidVectorDim", ErrInvalidVectorDim},
		{"ErrInvalidMetricType", ErrInvalidMetricType},
		{"ErrInvalidIndexType", ErrInvalidIndexType},
		{"ErrConnectionFailed", ErrConnectionFailed},
		{"ErrClientClosed", ErrClientClosed},
		{"ErrInvalidConfig", ErrInvalidConfig},
		{"ErrInvalidSchema", ErrInvalidSchema},
		{"ErrInvalidCollectionName", ErrInvalidCollectionName},
		{"ErrInvalidPartitionName", ErrInvalidPartitionName},
		{"ErrInvalidFieldName", ErrInvalidFieldName},
		{"ErrInvalidIndexParams", ErrInvalidIndexParams},
		{"ErrInvalidData", ErrInvalidData},
		{"ErrInvalidVectorData", ErrInvalidVectorData},
		{"ErrInvalidExpression", ErrInvalidExpression},
		{"ErrInvalidIDs", ErrInvalidIDs},
		{"ErrInvalidSearchRequest", ErrInvalidSearchRequest},
		{"ErrTimeout", ErrTimeout},
		{"ErrNotImplemented", ErrNotImplemented},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.NotEmpty(t, tt.err.Error())
		})
	}
}

func TestError_Error(t *testing.T) {
	err := &Error{
		Op:         "CreateCollection",
		Collection: "test_collection",
		Field:      "embedding",
		Err:        errors.New("connection failed"),
		Message:    "failed to create collection",
	}

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "CreateCollection")
	assert.Contains(t, errorMsg, "test_collection")
	assert.Contains(t, errorMsg, "embedding")
	assert.Contains(t, errorMsg, "connection failed")
}

func TestError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	err := &Error{
		Op:  "TestOp",
		Err: innerErr,
	}

	unwrapped := err.Unwrap()
	assert.Equal(t, innerErr, unwrapped)
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name       string
		op         string
		err        error
		collection string
		field      string
		wantNil    bool
	}{
		{
			name:       "wrap normal error",
			op:         "Insert",
			err:        errors.New("insert failed"),
			collection: "test_coll",
			field:      "vec",
			wantNil:    false,
		},
		{
			name:       "wrap nil error",
			op:         "Insert",
			err:        nil,
			collection: "test_coll",
			field:      "vec",
			wantNil:    true,
		},
		{
			name:       "wrap with empty collection",
			op:         "Search",
			err:        errors.New("search failed"),
			collection: "",
			field:      "",
			wantNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapError(tt.op, tt.err, tt.collection, tt.field)
			if tt.wantNil {
				assert.Nil(t, wrapped)
			} else {
				assert.NotNil(t, wrapped)
				assert.Contains(t, wrapped.Error(), tt.op)
				if tt.collection != "" {
					assert.Contains(t, wrapped.Error(), tt.collection)
				}
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "collection not found",
			err:  ErrCollectionNotFound,
			want: true,
		},
		{
			name: "partition not found",
			err:  ErrPartitionNotFound,
			want: true,
		},
		{
			name: "wrapped collection not found",
			err:  WrapError("Test", ErrCollectionNotFound, "coll", ""),
			want: true,
		},
		{
			name: "other error",
			err:  ErrConnectionFailed,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsAlreadyExists(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "collection exists",
			err:  ErrCollectionExists,
			want: true,
		},
		{
			name: "partition exists",
			err:  ErrPartitionExists,
			want: true,
		},
		{
			name: "wrapped collection exists",
			err:  WrapError("Test", ErrCollectionExists, "coll", ""),
			want: true,
		},
		{
			name: "other error",
			err:  ErrConnectionFailed,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAlreadyExists(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsInvalidArgument(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "invalid config",
			err:  ErrInvalidConfig,
			want: true,
		},
		{
			name: "invalid schema",
			err:  ErrInvalidSchema,
			want: true,
		},
		{
			name: "invalid collection name",
			err:  ErrInvalidCollectionName,
			want: true,
		},
		{
			name: "invalid vector dim",
			err:  ErrInvalidVectorDim,
			want: true,
		},
		{
			name: "wrapped invalid argument",
			err:  WrapError("Test", ErrInvalidData, "coll", ""),
			want: true,
		},
		{
			name: "other error",
			err:  ErrConnectionFailed,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsInvalidArgument(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsTimeout(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "timeout error",
			err:  ErrTimeout,
			want: true,
		},
		{
			name: "wrapped timeout",
			err:  WrapError("Test", ErrTimeout, "coll", ""),
			want: true,
		},
		{
			name: "other error",
			err:  ErrConnectionFailed,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTimeout(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsConnectionFailed(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "connection failed",
			err:  ErrConnectionFailed,
			want: true,
		},
		{
			name: "wrapped connection failed",
			err:  WrapError("Test", ErrConnectionFailed, "coll", ""),
			want: true,
		},
		{
			name: "other error",
			err:  ErrInvalidConfig,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsConnectionFailed(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestError_Chaining(t *testing.T) {
	// 测试错误链
	err1 := errors.New("root cause")
	err2 := WrapError("Op1", err1, "coll", "field")
	err3 := WrapError("Op2", err2, "coll2", "")

	// 验证可以通过 errors.Is 检测到原始错误
	assert.True(t, errors.Is(err3, err1))

	// 验证错误信息包含所有操作
	errMsg := err3.Error()
	assert.Contains(t, errMsg, "Op2")
	assert.Contains(t, errMsg, "Op1")
}

func TestError_CustomMessages(t *testing.T) {
	err := &Error{
		Op:      "CustomOp",
		Message: "This is a custom message",
	}

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "CustomOp")
	assert.Contains(t, errorMsg, "This is a custom message")
}

func TestError_PartialFields(t *testing.T) {
	tests := []struct {
		name  string
		err   *Error
		check func(t *testing.T, msg string)
	}{
		{
			name: "only op",
			err: &Error{
				Op: "TestOp",
			},
			check: func(t *testing.T, msg string) {
				assert.Contains(t, msg, "TestOp")
			},
		},
		{
			name: "op and collection",
			err: &Error{
				Op:         "TestOp",
				Collection: "test_coll",
			},
			check: func(t *testing.T, msg string) {
				assert.Contains(t, msg, "TestOp")
				assert.Contains(t, msg, "test_coll")
			},
		},
		{
			name: "op and field",
			err: &Error{
				Op:    "TestOp",
				Field: "test_field",
			},
			check: func(t *testing.T, msg string) {
				assert.Contains(t, msg, "TestOp")
				assert.Contains(t, msg, "test_field")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			tt.check(t, msg)
		})
	}
}
