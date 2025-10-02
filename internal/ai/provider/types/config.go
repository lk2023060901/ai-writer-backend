package types

import (
	"errors"
	"time"
)

var (
	ErrMissingAPIKey  = errors.New("API key is required")
	ErrMissingBaseURL = errors.New("base URL is required")
)

// Config Provider 通用配置
type Config struct {
	APIKey  string            // API Key
	BaseURL string            // API 基础 URL
	Timeout time.Duration     // 请求超时
	Model   string            // 默认模型
	Headers map[string]string // 自定义 HTTP Headers
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return ErrMissingAPIKey
	}
	if c.BaseURL == "" {
		return ErrMissingBaseURL
	}
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	return nil
}
