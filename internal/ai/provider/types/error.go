package types

import (
	"errors"
	"fmt"
)

// ErrorType API 错误类型（基于 Anthropic 文档）
type ErrorType string

const (
	// 4xx 客户端错误
	ErrorTypeInvalidRequest  ErrorType = "invalid_request_error"  // 400 - 请求格式或内容错误
	ErrorTypeAuthentication  ErrorType = "authentication_error"   // 401 - API Key 问题
	ErrorTypePermission      ErrorType = "permission_error"       // 403 - API Key 权限不足
	ErrorTypeNotFound        ErrorType = "not_found_error"        // 404 - 资源未找到
	ErrorTypeRequestTooLarge ErrorType = "request_too_large"      // 413 - 请求超过 32MB
	ErrorTypeRateLimit       ErrorType = "rate_limit_error"       // 429 - 达到速率限制

	// 5xx 服务器错误
	ErrorTypeAPI        ErrorType = "api_error"        // 500 - 内部服务器错误
	ErrorTypeOverloaded ErrorType = "overloaded_error" // 529 - API 临时过载
)

// ProviderError Provider 错误
type ProviderError struct {
	Type       ErrorType // 错误类型
	Provider   string    // Provider 名称
	StatusCode int       // HTTP 状态码
	Message    string    // 错误消息
	RequestID  string    // 请求 ID（用于追踪）
	Err        error     // 原始错误

	// Rate Limit 相关（仅在 ErrorTypeRateLimit 时有效）
	RateLimitInfo *RateLimitInfo
}

// RateLimitInfo Rate Limit 详细信息
type RateLimitInfo struct {
	RetryAfter              int    // 重试等待秒数
	RequestsLimit           int    // 请求数限制（RPM）
	RequestsRemaining       int    // 剩余请求数
	RequestsReset           string // 请求限制重置时间
	InputTokensLimit        int    // 输入 token 限制（ITPM）
	InputTokensRemaining    int    // 剩余输入 token
	InputTokensReset        string // 输入 token 限制重置时间
	OutputTokensLimit       int    // 输出 token 限制（OTPM）
	OutputTokensRemaining   int    // 剩余输出 token
	OutputTokensReset       string // 输出 token 限制重置时间
}

func (e *ProviderError) Error() string {
	if e.RequestID != "" {
		if e.Err != nil {
			return fmt.Sprintf("[%s][%s][%s] %s: %v (request_id: %s)",
				e.Provider, e.Type, httpStatusText(e.StatusCode), e.Message, e.Err, e.RequestID)
		}
		return fmt.Sprintf("[%s][%s][%s] %s (request_id: %s)",
			e.Provider, e.Type, httpStatusText(e.StatusCode), e.Message, e.RequestID)
	}

	if e.Err != nil {
		return fmt.Sprintf("[%s][%s] %s: %v", e.Provider, e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s][%s] %s", e.Provider, e.Type, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// IsRateLimitError 判断是否为速率限制错误
func (e *ProviderError) IsRateLimitError() bool {
	return e.Type == ErrorTypeRateLimit
}

// IsRetryable 判断错误是否可重试
func (e *ProviderError) IsRetryable() bool {
	switch e.Type {
	case ErrorTypeRateLimit, ErrorTypeAPI, ErrorTypeOverloaded:
		return true
	default:
		return false
	}
}

// NewProviderError 创建 Provider 错误
func NewProviderError(provider, message string, err error) *ProviderError {
	return &ProviderError{
		Type:     ErrorTypeAPI,
		Provider: provider,
		Message:  message,
		Err:      err,
	}
}

// NewRateLimitError 创建速率限制错误
func NewRateLimitError(provider string, rateLimitInfo *RateLimitInfo) *ProviderError {
	return &ProviderError{
		Type:          ErrorTypeRateLimit,
		Provider:      provider,
		StatusCode:    429,
		Message:       "rate limit exceeded",
		RateLimitInfo: rateLimitInfo,
	}
}

// httpStatusText 返回 HTTP 状态码文本
func httpStatusText(code int) string {
	switch code {
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 413:
		return "Payload Too Large"
	case 429:
		return "Too Many Requests"
	case 500:
		return "Internal Server Error"
	case 529:
		return "Service Overloaded"
	default:
		return fmt.Sprintf("Status %d", code)
	}
}

// 预定义错误（特定于 AI 功能）
var (
	ErrInvalidModel       = errors.New("invalid model")
	ErrMaxTokensExceeded  = errors.New("max tokens exceeded")
	ErrMessagesTruncated  = errors.New("messages truncated, continue with next request")
)
