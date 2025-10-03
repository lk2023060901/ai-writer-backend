package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/lk2023060901/ai-writer-backend/internal/pkg/errors"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`              // 业务错误码（0表示成功）
	Message string      `json:"message,omitempty"` // 提示信息
	Data    interface{} `json:"data"`              // 实际数据（可能为空对象 {}）
}

// Success 成功响应（200）
func Success(c *gin.Context, data interface{}) {
	if data == nil {
		data = struct{}{}
	}
	c.JSON(http.StatusOK, Response{
		Code:    apperrors.Success,
		Message: "",
		Data:    data,
	})
}

// SuccessWithMessage 带消息的成功响应（200）
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	if data == nil {
		data = struct{}{}
	}
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: message,
		Data:    data,
	})
}

// Created 创建资源成功（201）
func Created(c *gin.Context, data interface{}) {
	if data == nil {
		data = struct{}{}
	}
	c.JSON(http.StatusCreated, Response{
		Code:    http.StatusCreated,
		Message: "",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, httpStatus int, message string) {
	c.JSON(httpStatus, Response{
		Code:    httpStatus,
		Message: message,
		Data:    struct{}{},
	})
}

// BadRequest 400 错误
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

// Unauthorized 401 错误
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message)
}

// Forbidden 403 错误
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message)
}

// NotFound 404 错误
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message)
}

// InternalError 500 错误
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}

// HandleError 统一错误处理（使用AppError）
func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// 提取错误码和详情
	code := apperrors.ExtractCode(err)
	httpStatus := apperrors.GetHTTPStatus(code)
	message := apperrors.FormatError(code, apperrors.GetDetails(err))

	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    struct{}{},
	})
}

// ErrorWithCode 使用错误码的错误响应
func ErrorWithCode(c *gin.Context, code int, details ...string) {
	httpStatus := apperrors.GetHTTPStatus(code)
	message := apperrors.FormatError(code, details...)

	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    struct{}{},
	})
}
