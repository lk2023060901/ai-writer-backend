package mineru

import (
	"go.uber.org/zap"
	"context"
	"fmt"
	"time"

)

// CreateTask 创建单个文档解析任务
func (c *Client) CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error) {
	// 应用默认配置
	c.applyDefaultTaskConfig(req)

	var resp CreateTaskResponse
	if err := c.doRequest(ctx, "POST", "/api/v4/extract/task", req, &resp); err != nil {
		return nil, err
	}

	if resp.Data.TaskID == "" {
		return nil, ErrEmptyResponse
	}

	c.logger.Info("task created successfully",
		zap.String("task_id", resp.Data.TaskID),
		zap.String("url", req.URL),
	)

	return &resp, nil
}

// GetTaskResult 获取任务结果
func (c *Client) GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	path := fmt.Sprintf("/api/v4/extract/task/%s", taskID)
	var resp GetTaskResultResponse
	if err := c.doRequest(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	return &resp.Data, nil
}

// WaitForTask 等待任务完成（轮询）
func (c *Client) WaitForTask(ctx context.Context, taskID string, opts *PollOptions) (*TaskResult, error) {
	if opts == nil {
		opts = DefaultPollOptions()
	}

	c.logger.Info("waiting for task to complete",
		zap.String("task_id", taskID),
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
			result, err := c.GetTaskResult(timeoutCtx, taskID)
			if err != nil {
				return nil, err
			}

			// 记录进度
			if result.ExtractProgress != nil && opts.OnProgress != nil {
				opts.OnProgress(result.ExtractProgress)
			}

			c.logger.Debug("task status",
				zap.String("task_id", taskID),
				zap.String("state", string(result.State)),
			)

			switch result.State {
			case TaskStateDone:
				c.logger.Info("task completed successfully",
					zap.String("task_id", taskID),
					zap.String("zip_url", result.FullZipURL),
				)
				return result, nil

			case TaskStateFailed:
				c.logger.Error("task failed",
					zap.String("task_id", taskID),
					zap.String("error", result.ErrMsg),
				)
				return result, fmt.Errorf("task failed: %s", result.ErrMsg)

			case TaskStatePending, TaskStateRunning, TaskStateConverting:
				// 继续等待
				if result.ExtractProgress != nil {
					c.logger.Info("task in progress",
						zap.String("task_id", taskID),
						zap.String("state", string(result.State)),
						zap.Int("extracted_pages", result.ExtractProgress.ExtractedPages),
						zap.Int("total_pages", result.ExtractProgress.TotalPages),
					)
				}
				continue

			default:
				c.logger.Warn("unknown task state",
					zap.String("task_id", taskID),
					zap.String("state", string(result.State)),
				)
				continue
			}
		}
	}
}

// CreateTaskAndWait 创建任务并等待完成
func (c *Client) CreateTaskAndWait(ctx context.Context, req *CreateTaskRequest, opts *PollOptions) (*TaskResult, error) {
	// 创建任务
	resp, err := c.CreateTask(ctx, req)
	if err != nil {
		return nil, err
	}

	// 等待完成
	return c.WaitForTask(ctx, resp.Data.TaskID, opts)
}

// applyDefaultTaskConfig 应用默认配置到任务请求
func (c *Client) applyDefaultTaskConfig(req *CreateTaskRequest) {
	if req.Language == "" {
		req.Language = c.config.DefaultLanguage
	}
	if req.EnableFormula == nil {
		enableFormula := c.config.EnableFormula
		req.EnableFormula = &enableFormula
	}
	if req.EnableTable == nil {
		enableTable := c.config.EnableTable
		req.EnableTable = &enableTable
	}
	if req.ModelVersion == "" {
		req.ModelVersion = c.config.ModelVersion
	}
}
