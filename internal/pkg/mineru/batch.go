package mineru

import (
	"go.uber.org/zap"
	"context"
	"fmt"
	"io"
	"os"
	"time"

)

// applyDefaultBatchUploadConfig 应用默认配置到批量上传请求
func (c *Client) applyDefaultBatchUploadConfig(req *BatchUploadRequest) {
	if req.Language == "" {
		req.Language = c.config.DefaultLanguage
	}
	if req.EnableFormula == nil {
		ef := c.config.EnableFormula
		req.EnableFormula = &ef
	}
	if req.EnableTable == nil {
		et := c.config.EnableTable
		req.EnableTable = &et
	}
	if req.ModelVersion == "" {
		req.ModelVersion = c.config.ModelVersion
	}
}

// applyDefaultBatchTaskConfig 应用默认配置到批量任务请求
func (c *Client) applyDefaultBatchTaskConfig(req *BatchTaskRequest) {
	if req.Language == "" {
		req.Language = c.config.DefaultLanguage
	}
	if req.EnableFormula == nil {
		ef := c.config.EnableFormula
		req.EnableFormula = &ef
	}
	if req.EnableTable == nil {
		et := c.config.EnableTable
		req.EnableTable = &et
	}
	if req.ModelVersion == "" {
		req.ModelVersion = c.config.ModelVersion
	}
}

// CreateBatchWithFiles 批量文件上传解析
// 1. 申请上传 URL
// 2. 上传文件
// 3. 返回 batch_id
func (c *Client) CreateBatchWithFiles(ctx context.Context, req *BatchUploadRequest, filePaths []string) (*BatchUploadResponse, error) {
	if len(req.Files) != len(filePaths) {
		return nil, fmt.Errorf("files count mismatch: request has %d files, but got %d file paths", len(req.Files), len(filePaths))
	}

	// 应用默认配置
	c.applyDefaultBatchUploadConfig(req)

	// 1. 申请上传 URL
	var resp BatchUploadResponse
	if err := c.doRequest(ctx, "POST", "/api/v4/file-urls/batch", req, &resp); err != nil {
		return nil, err
	}

	if resp.Data.BatchID == "" || len(resp.Data.FileURLs) == 0 {
		return nil, ErrEmptyResponse
	}

	c.logger.Info("batch upload URLs created",
		zap.String("batch_id", resp.Data.BatchID),
		zap.Int("file_count", len(resp.Data.FileURLs)),
	)

	// 2. 上传文件
	for i, url := range resp.Data.FileURLs {
		if i >= len(filePaths) {
			break
		}

		if err := c.uploadFileFromPath(ctx, url, filePaths[i]); err != nil {
			return nil, fmt.Errorf("upload file %s failed: %w", filePaths[i], err)
		}
	}

	c.logger.Info("all files uploaded successfully",
		zap.String("batch_id", resp.Data.BatchID),
	)

	return &resp, nil
}

// CreateBatchWithURLs 批量 URL 解析
func (c *Client) CreateBatchWithURLs(ctx context.Context, req *BatchTaskRequest) (*BatchTaskResponse, error) {
	// 应用默认配置
	c.applyDefaultBatchTaskConfig(req)

	var resp BatchTaskResponse
	if err := c.doRequest(ctx, "POST", "/api/v4/extract/task/batch", req, &resp); err != nil {
		return nil, err
	}

	if resp.Data.BatchID == "" {
		return nil, ErrEmptyResponse
	}

	c.logger.Info("batch task created",
		zap.String("batch_id", resp.Data.BatchID),
		zap.Int("file_count", len(req.Files)),
	)

	return &resp, nil
}

// GetBatchResults 获取批量任务结果
func (c *Client) GetBatchResults(ctx context.Context, batchID string) (*GetBatchResultsResponse, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch_id is required")
	}

	path := fmt.Sprintf("/api/v4/extract-results/batch/%s", batchID)
	var resp GetBatchResultsResponse
	if err := c.doRequest(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// WaitForBatch 等待批量任务完成
func (c *Client) WaitForBatch(ctx context.Context, batchID string, opts *PollOptions) (*GetBatchResultsResponse, error) {
	if opts == nil {
		opts = DefaultPollOptions()
	}

	c.logger.Info("waiting for batch to complete",
		zap.String("batch_id", batchID),
		zap.Duration("timeout", opts.Timeout),
		zap.Duration("interval", opts.Interval),
	)

	// 创建超时 context
	timeoutCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return nil, ErrTimeout
		case <-ticker.C:
			results, err := c.GetBatchResults(timeoutCtx, batchID)
			if err != nil {
				return nil, err
			}

			// 统计任务状态
			var done, failed, running int
			for _, result := range results.Data.ExtractResult {
				switch result.State {
				case TaskStateDone:
					done++
				case TaskStateFailed:
					failed++
				case TaskStatePending, TaskStateRunning, TaskStateConverting, TaskStateWaitingFile:
					running++
				}
			}

			total := len(results.Data.ExtractResult)
			c.logger.Info("batch progress",
				zap.String("batch_id", batchID),
				zap.Int("total", total),
				zap.Int("done", done),
				zap.Int("failed", failed),
				zap.Int("running", running),
			)

			// 所有任务完成（成功或失败）
			if done+failed == total {
				if failed == total {
					return results, ErrAllTasksFailed
				}
				c.logger.Info("batch completed",
					zap.String("batch_id", batchID),
					zap.Int("success", done),
					zap.Int("failed", failed),
				)
				return results, nil
			}

			// 继续等待
			continue
		}
	}
}

// CreateBatchWithFilesAndWait 批量上传文件并等待完成
func (c *Client) CreateBatchWithFilesAndWait(ctx context.Context, req *BatchUploadRequest, filePaths []string, opts *PollOptions) (*GetBatchResultsResponse, error) {
	// 创建批量任务并上传文件
	resp, err := c.CreateBatchWithFiles(ctx, req, filePaths)
	if err != nil {
		return nil, err
	}

	// 等待完成
	return c.WaitForBatch(ctx, resp.Data.BatchID, opts)
}

// CreateBatchWithURLsAndWait 批量 URL 解析并等待完成
func (c *Client) CreateBatchWithURLsAndWait(ctx context.Context, req *BatchTaskRequest, opts *PollOptions) (*GetBatchResultsResponse, error) {
	// 创建批量任务
	resp, err := c.CreateBatchWithURLs(ctx, req)
	if err != nil {
		return nil, err
	}

	// 等待完成
	return c.WaitForBatch(ctx, resp.Data.BatchID, opts)
}

// uploadFileFromPath 从文件路径上传文件
func (c *Client) uploadFileFromPath(ctx context.Context, url, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	c.logger.Info("uploading file",
		zap.String("file", filePath),
		zap.String("url", url),
	)

	return c.uploadFile(ctx, url, file)
}

// UploadFileWithReader 使用 io.Reader 上传文件
func (c *Client) UploadFileWithReader(ctx context.Context, url string, reader io.Reader) error {
	return c.uploadFile(ctx, url, reader)
}
