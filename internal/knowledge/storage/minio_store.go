package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	pkgminio "github.com/lk2023060901/ai-writer-backend/internal/pkg/minio"
	"go.uber.org/zap"
)

// MinIOStore MinIO 文件存储实现
type MinIOStore struct {
	client *pkgminio.Client
	logger *logger.Logger
}

// NewMinIOStore 创建 MinIO 文件存储
func NewMinIOStore(client *pkgminio.Client, lgr *logger.Logger) *MinIOStore {
	if lgr == nil {
		lgr = logger.L()
	}
	return &MinIOStore{
		client: client,
		logger: lgr,
	}
}

// Upload 上传文件
func (s *MinIOStore) Upload(ctx context.Context, req *UploadFileRequest) (*UploadFileResponse, error) {
	// 确保 Bucket 存在
	if err := s.EnsureBucket(ctx, req.Bucket); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket: %w", err)
	}

	// 上传选项
	opts := pkgminio.PutObjectOptions{
		ContentType:  req.ContentType,
		UserMetadata: req.Metadata,
	}

	// 上传文件
	info, err := s.client.PutObject(ctx, req.Bucket, req.Key, req.Content, req.Size, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	s.logger.Info("file uploaded successfully",
		zap.String("bucket", req.Bucket),
		zap.String("key", req.Key),
		zap.Int64("size", req.Size))

	return &UploadFileResponse{
		Bucket:    info.Bucket,
		Key:       info.Key,
		ETag:      info.ETag,
		Size:      info.Size,
		VersionID: info.VersionID,
	}, nil
}

// Download 下载文件
func (s *MinIOStore) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, bucket, key, pkgminio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	s.logger.Info("file downloaded successfully",
		zap.String("bucket", bucket),
		zap.String("key", key))

	return obj, nil
}

// Delete 删除文件
func (s *MinIOStore) Delete(ctx context.Context, bucket, key string) error {
	if err := s.client.RemoveObject(ctx, bucket, key, pkgminio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	s.logger.Info("file deleted successfully",
		zap.String("bucket", bucket),
		zap.String("key", key))

	return nil
}

// Exists 检查文件是否存在
func (s *MinIOStore) Exists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, bucket, key, pkgminio.StatObjectOptions{})
	if err != nil {
		// 判断是否是 NotFound 错误
		if pkgminio.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}
	return true, nil
}

// GetMetadata 获取文件元数据
func (s *MinIOStore) GetMetadata(ctx context.Context, bucket, key string) (*FileMetadata, error) {
	info, err := s.client.StatObject(ctx, bucket, key, pkgminio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}

	return &FileMetadata{
		Bucket:       bucket,
		Key:          key,
		Size:         info.Size,
		ContentType:  info.ContentType,
		ETag:         info.ETag,
		LastModified: info.LastModified,
		Metadata:     info.Metadata,
	}, nil
}

// EnsureBucket 确保 Bucket 存在
func (s *MinIOStore) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		if err := s.client.MakeBucket(ctx, bucket, pkgminio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		s.logger.Info("bucket created successfully",
			zap.String("bucket", bucket))
	}

	return nil
}
