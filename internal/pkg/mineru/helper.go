package mineru

import (
	"go.uber.org/zap"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

)

// DownloadResult 下载解析结果压缩包
func (c *Client) DownloadResult(ctx context.Context, zipURL, destPath string) error {
	c.logger.Info("downloading result",
		zap.String("url", zipURL),
		zap.String("dest", destPath),
	)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zipURL, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}

	// 执行请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do download request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// 确保目录存在
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// 创建文件
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	// 写入文件
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	c.logger.Info("result downloaded successfully",
		zap.String("dest", destPath),
		zap.Int64("size", written),
	)

	return nil
}

// IsTaskCompleted 检查任务是否完成
func IsTaskCompleted(state TaskState) bool {
	return state == TaskStateDone || state == TaskStateFailed
}

// IsTaskSuccessful 检查任务是否成功
func IsTaskSuccessful(state TaskState) bool {
	return state == TaskStateDone
}

// IsTaskFailed 检查任务是否失败
func IsTaskFailed(state TaskState) bool {
	return state == TaskStateFailed
}

// IsTaskProcessing 检查任务是否正在处理
func IsTaskProcessing(state TaskState) bool {
	return state == TaskStatePending ||
		state == TaskStateRunning ||
		state == TaskStateConverting ||
		state == TaskStateWaitingFile
}

// GetBatchStatistics 获取批量任务统计信息
func GetBatchStatistics(results []BatchExtractResult) (done, failed, processing int) {
	for _, result := range results {
		switch {
		case IsTaskSuccessful(result.State):
			done++
		case IsTaskFailed(result.State):
			failed++
		case IsTaskProcessing(result.State):
			processing++
		}
	}
	return
}

// FilterSuccessfulResults 过滤成功的结果
func FilterSuccessfulResults(results []BatchExtractResult) []BatchExtractResult {
	var successful []BatchExtractResult
	for _, result := range results {
		if IsTaskSuccessful(result.State) {
			successful = append(successful, result)
		}
	}
	return successful
}

// FilterFailedResults 过滤失败的结果
func FilterFailedResults(results []BatchExtractResult) []BatchExtractResult {
	var failed []BatchExtractResult
	for _, result := range results {
		if IsTaskFailed(result.State) {
			failed = append(failed, result)
		}
	}
	return failed
}

// GetTaskProgress 获取任务进度百分比
func GetTaskProgress(progress *ExtractProgress) float64 {
	if progress == nil || progress.TotalPages == 0 {
		return 0
	}
	return float64(progress.ExtractedPages) / float64(progress.TotalPages) * 100
}

// FormatTaskProgress 格式化任务进度
func FormatTaskProgress(progress *ExtractProgress) string {
	if progress == nil {
		return "N/A"
	}
	return fmt.Sprintf("%d/%d (%.1f%%)",
		progress.ExtractedPages,
		progress.TotalPages,
		GetTaskProgress(progress),
	)
}

// CreateTaskWithFile 创建单个文件解析任务（带文件上传）并等待完成
func (c *Client) CreateTaskWithFile(ctx context.Context, filename string, fileData []byte, req *CreateTaskRequest) (*TaskResult, error) {
	// 使用批量接口上传单个文件
	batchReq := &BatchUploadRequest{
		EnableFormula: req.EnableFormula,
		EnableTable:   req.EnableTable,
		Language:      req.Language,
		Files: []BatchFileInfo{
			{
				Name:       filename,
				IsOCR:      req.IsOCR,
				DataID:     req.DataID,
				PageRanges: req.PageRanges,
			},
		},
		Callback:     req.Callback,
		Seed:         req.Seed,
		ExtraFormats: req.ExtraFormats,
		ModelVersion: req.ModelVersion,
	}

	// 1. 申请上传 URL
	c.applyDefaultBatchUploadConfig(batchReq)

	var resp BatchUploadResponse
	if err := c.doRequest(ctx, "POST", "/api/v4/file-urls/batch", batchReq, &resp); err != nil {
		return nil, err
	}

	if resp.Data.BatchID == "" || len(resp.Data.FileURLs) == 0 {
		return nil, ErrEmptyResponse
	}

	c.logger.Info("upload URL created",
		zap.String("batch_id", resp.Data.BatchID),
		zap.String("filename", filename),
	)

	// 2. 上传文件
	uploadURL := resp.Data.FileURLs[0]
	if err := c.uploadFile(ctx, uploadURL, bytes.NewReader(fileData)); err != nil {
		return nil, fmt.Errorf("upload file failed: %w", err)
	}

	// 3. 等待批量任务完成
	pollOpts := &PollOptions{
		Interval: 3 * time.Second,
		Timeout:  10 * time.Minute,
	}

	batchResult, err := c.WaitForBatch(ctx, resp.Data.BatchID, pollOpts)
	if err != nil {
		return nil, err
	}

	// 返回第一个任务结果
	if len(batchResult.Data.ExtractResult) == 0 {
		return nil, ErrEmptyResponse
	}

	firstResult := batchResult.Data.ExtractResult[0]
	return &TaskResult{
		TaskID:          resp.Data.BatchID, // 使用 batch_id 作为 task_id
		DataID:          firstResult.DataID,
		State:           firstResult.State,
		FullZipURL:      firstResult.FullZipURL,
		ErrMsg:          firstResult.ErrMsg,
		ExtractProgress: firstResult.ExtractProgress,
	}, nil
}
