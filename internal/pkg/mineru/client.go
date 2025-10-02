package mineru

import (
	"go.uber.org/zap"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
)

// Client MinerU HTTP 客户端
type Client struct {
	config     *Config
	httpClient *http.Client
	logger     *logger.Logger
}

// New 创建 MinerU 客户端
func New(cfg *Config, log *logger.Logger) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: log,
	}, nil
}

// doRequest 执行 HTTP 请求
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	url := c.config.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)

		c.logger.Debug("mineru request",
			zap.String("method", method),
			zap.String("url", url),
			zap.String("body", string(data)),
		)
	} else {
		c.logger.Debug("mineru request",
			zap.String("method", method),
			zap.String("url", url),
		)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// 执行请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("mineru request failed",
			zap.String("method", method),
			zap.String("url", url),
			zap.Error(err),
		)
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	c.logger.Debug("mineru response",
		zap.Int("status", resp.StatusCode),
		zap.String("body", string(respData)),
	)

	// 解析响应（无论 HTTP 状态码是什么）
	if result != nil && len(respData) > 0 {
		if err := json.Unmarshal(respData, result); err != nil {
			// 如果 HTTP 状态码不是 200 且解析失败，返回 HTTP 错误
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respData))
			}
			return fmt.Errorf("unmarshal response: %w", err)
		}

		// 检查业务错误码
		if apiResp, ok := result.(interface{ GetCode() int }); ok {
			if apiResp.GetCode() != 0 {
				// 尝试获取 msgCode（新版 API）
				var errorCode interface{} = apiResp.GetCode()
				if apiRespWithMsgCode, ok := result.(interface{ GetMsgCode() string }); ok {
					if msgCode := apiRespWithMsgCode.GetMsgCode(); msgCode != "" {
						errorCode = msgCode
					}
				}

				if apiResp, ok := result.(interface {
					GetCode() int
					GetMsg() string
					GetTraceID() string
				}); ok {
					return NewMinerUError(
						errorCode,
						GetErrorMessage(errorCode)+": "+apiResp.GetMsg(),
						apiResp.GetTraceID(),
					)
				}
			}
		}
	}

	// 检查 HTTP 状态码（如果没有解析出业务错误）
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respData))
	}

	return nil
}

// GetCode 实现 GetCode 接口
func (r *APIResponse) GetCode() int {
	// 如果有 MsgCode（新版 API），转换为错误码
	if r.MsgCode != "" {
		// 新版 API 返回 msgCode（字符串），将其转换为接口返回
		// 如果 success 为 false，返回非 0 值表示错误
		if r.Success != nil && !*r.Success {
			return -1 // 标记为错误
		}
		return 0
	}
	return r.Code
}

// GetMsg 实现 GetMsg 接口
func (r *APIResponse) GetMsg() string {
	return r.Msg
}

// GetTraceID 实现 GetTraceID 接口
func (r *APIResponse) GetTraceID() string {
	if r.TraceId != "" {
		return r.TraceId
	}
	return r.TraceID
}

// GetMsgCode 获取新版 API 的 msgCode
func (r *APIResponse) GetMsgCode() string {
	return r.MsgCode
}

// uploadFile 上传文件到预签名 URL
func (c *Client) uploadFile(ctx context.Context, url string, fileData io.Reader) error {
	c.logger.Debug("uploading file", zap.String("url", url))

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, fileData)
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("upload file failed",
			zap.String("url", url),
			zap.Error(err),
		)
		return fmt.Errorf("do upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("file uploaded successfully", zap.String("url", url))
	return nil
}

// retryWithBackoff 带退避的重试
func (c *Client) retryWithBackoff(ctx context.Context, operation func() error) error {
	var lastErr error
	for i := 0; i < c.config.MaxRetries; i++ {
		if err := operation(); err != nil {
			lastErr = err
			c.logger.Warn("operation failed, retrying",
				zap.Int("attempt", i+1),
				zap.Int("max_retries", c.config.MaxRetries),
				zap.Error(err),
			)

			// 计算退避时间
			backoff := time.Duration(i+1) * 2 * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
		return nil
	}
	return fmt.Errorf("operation failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// Close 关闭客户端
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
