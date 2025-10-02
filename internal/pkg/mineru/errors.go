package mineru

import (
	"errors"
	"fmt"
)

// 错误码常量
const (
	// 认证相关错误
	ErrCodeTokenInvalid = "A0202" // Token 错误
	ErrCodeTokenExpired = "A0211" // Token 过期

	// 通用错误
	ErrCodeParamError    = -500   // 传参错误
	ErrCodeServiceError  = -10001 // 服务异常
	ErrCodeRequestParam  = -10002 // 请求参数错误

	// 文件相关错误
	ErrCodeUploadURLFailed   = -60001 // 生成上传 URL 失败
	ErrCodeFileFormatFailed  = -60002 // 获取匹配的文件格式失败
	ErrCodeFileReadFailed    = -60003 // 文件读取失败
	ErrCodeEmptyFile         = -60004 // 空文件
	ErrCodeFileSizeExceeded  = -60005 // 文件大小超出限制
	ErrCodeFilePageExceeded  = -60006 // 文件页数超过限制
	ErrCodeModelUnavailable  = -60007 // 模型服务暂时不可用
	ErrCodeFileReadTimeout   = -60008 // 文件读取超时
	ErrCodeQueueFull         = -60009 // 任务提交队列已满
	ErrCodeParseFailed       = -60010 // 解析失败
	ErrCodeInvalidFile       = -60011 // 获取有效文件失败
	ErrCodeTaskNotFound      = -60012 // 找不到任务
	ErrCodeNoPermission      = -60013 // 没有权限访问该任务
	ErrCodeDeleteRunning     = -60014 // 删除运行中的任务
	ErrCodeConvertFailed     = -60015 // 文件转换失败
	ErrCodeFormatConvertFail = -60016 // 文件转换为指定格式失败
)

// MinerUError MinerU 错误
type MinerUError struct {
	Code    interface{} // 可能是 int 或 string
	Message string
	TraceID string
}

func (e *MinerUError) Error() string {
	if e.TraceID != "" {
		return fmt.Sprintf("mineru error (code=%v, trace_id=%s): %s", e.Code, e.TraceID, e.Message)
	}
	return fmt.Sprintf("mineru error (code=%v): %s", e.Code, e.Message)
}

// NewMinerUError 创建 MinerU 错误
func NewMinerUError(code interface{}, message, traceID string) *MinerUError {
	return &MinerUError{
		Code:    code,
		Message: message,
		TraceID: traceID,
	}
}

// GetErrorMessage 根据错误码获取友好的错误信息
func GetErrorMessage(code interface{}) string {
	var intCode int
	switch v := code.(type) {
	case int:
		intCode = v
	case float64:
		intCode = int(v)
	case string:
		switch v {
		case "A0202":
			return "Token 错误，请检查 Token 是否正确或是否有 Bearer 前缀"
		case "A0211":
			return "Token 已过期，请更换新 Token"
		default:
			return "未知错误"
		}
	default:
		return "未知错误"
	}

	switch intCode {
	case ErrCodeParamError:
		return "传参错误，请确保参数类型及 Content-Type 正确"
	case ErrCodeServiceError:
		return "服务异常，请稍后再试"
	case ErrCodeRequestParam:
		return "请求参数错误，检查请求参数格式"
	case ErrCodeUploadURLFailed:
		return "生成上传 URL 失败，请稍后再试"
	case ErrCodeFileFormatFailed:
		return "检测文件类型失败，请求的文件名及链接中带有正确的后缀名"
	case ErrCodeFileReadFailed:
		return "文件读取失败，请检查文件是否损坏并重新上传"
	case ErrCodeEmptyFile:
		return "空文件，请上传有效文件"
	case ErrCodeFileSizeExceeded:
		return "文件大小超出限制，最大支持 200MB"
	case ErrCodeFilePageExceeded:
		return "文件页数超过限制，请拆分文件后重试"
	case ErrCodeModelUnavailable:
		return "模型服务暂时不可用，请稍后重试或联系技术支持"
	case ErrCodeFileReadTimeout:
		return "文件读取超时，请检查 URL 可访问性"
	case ErrCodeQueueFull:
		return "任务提交队列已满，请稍后再试"
	case ErrCodeParseFailed:
		return "解析失败，请稍后再试"
	case ErrCodeInvalidFile:
		return "获取有效文件失败，请确保文件已上传"
	case ErrCodeTaskNotFound:
		return "找不到任务，请确保 task_id 有效且未删除"
	case ErrCodeNoPermission:
		return "没有权限访问该任务，只能访问自己提交的任务"
	case ErrCodeDeleteRunning:
		return "运行中的任务暂不支持删除"
	case ErrCodeConvertFailed:
		return "文件转换失败，可以手动转为 PDF 再上传"
	case ErrCodeFormatConvertFail:
		return "文件转换为指定格式失败，可以尝试其他格式导出或重试"
	default:
		return fmt.Sprintf("未知错误码: %v", code)
	}
}

// 预定义错误
var (
	ErrTimeout        = errors.New("mineru: operation timeout")
	ErrInvalidConfig  = errors.New("mineru: invalid configuration")
	ErrEmptyResponse  = errors.New("mineru: empty response")
	ErrTaskNotReady   = errors.New("mineru: task not ready")
	ErrAllTasksFailed = errors.New("mineru: all tasks failed")
)
