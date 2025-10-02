package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`              // HTTP 状态码
	Message string      `json:"message,omitempty"` // 提示信息
	Data    interface{} `json:"data"`              // 实际数据（可能为空对象 {}）
}

// Success 成功响应（200）
func Success(c *gin.Context, data interface{}) {
	if data == nil {
		data = struct{}{}
	}
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
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
