package data

import (
	"bytes"
	"context"
	"fmt"
	"io"

	pkgminio "github.com/lk2023060901/ai-writer-backend/internal/pkg/minio"
)

// MinIOStorageService 实现 biz.StorageService 接口
type MinIOStorageService struct {
	client *pkgminio.Client
	bucket string
}

// NewMinIOStorageService 创建 MinIO 存储服务
func NewMinIOStorageService(client *pkgminio.Client, bucket string) *MinIOStorageService {
	return &MinIOStorageService{
		client: client,
		bucket: bucket,
	}
}

// UploadFile 上传文件
func (s *MinIOStorageService) UploadFile(ctx context.Context, bucket, objectName string, data []byte, contentType string) (string, error) {
	if bucket == "" {
		bucket = s.bucket
	}

	reader := bytes.NewReader(data)
	_, err := s.client.PutObject(ctx, bucket, objectName, reader, int64(len(data)), pkgminio.PutObjectOptions{
		ContentType: contentType,
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	return objectName, nil
}

// GetFile 获取文件
func (s *MinIOStorageService) GetFile(ctx context.Context, bucket, objectName string) ([]byte, error) {
	if bucket == "" {
		bucket = s.bucket
	}

	obj, err := s.client.GetObject(ctx, bucket, objectName, pkgminio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return data, nil
}

// DeleteFile 删除文件
func (s *MinIOStorageService) DeleteFile(ctx context.Context, bucket, objectName string) error {
	if bucket == "" {
		bucket = s.bucket
	}

	err := s.client.RemoveObject(ctx, bucket, objectName, pkgminio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}
