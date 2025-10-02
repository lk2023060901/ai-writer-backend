package storage

import (
	"context"
	"io"
)

// FileStore 文件存储接口
type FileStore interface {
	// Upload 上传文件
	Upload(ctx context.Context, req *UploadFileRequest) (*UploadFileResponse, error)

	// Download 下载文件
	Download(ctx context.Context, bucket, key string) (io.ReadCloser, error)

	// Delete 删除文件
	Delete(ctx context.Context, bucket, key string) error

	// Exists 检查文件是否存在
	Exists(ctx context.Context, bucket, key string) (bool, error)

	// GetMetadata 获取文件元数据
	GetMetadata(ctx context.Context, bucket, key string) (*FileMetadata, error)

	// EnsureBucket 确保 Bucket 存在
	EnsureBucket(ctx context.Context, bucket string) error
}

// UploadFileRequest 上传文件请求
type UploadFileRequest struct {
	Bucket      string
	Key         string
	Content     io.Reader
	Size        int64
	ContentType string
	Metadata    map[string]string
}

// UploadFileResponse 上传文件响应
type UploadFileResponse struct {
	Bucket    string
	Key       string
	ETag      string
	Size      int64
	VersionID string
}

// FileMetadata 文件元数据
type FileMetadata struct {
	Bucket       string
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified string
	Metadata     map[string]string
}
