package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/email/service"
	"github.com/lk2023060901/ai-writer-backend/internal/email/types"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/response"
)

// EmailHandler 邮件发送处理器
type EmailHandler struct {
	emailService *service.EmailService
}

// NewEmailHandler 创建邮件处理器
func NewEmailHandler(emailService *service.EmailService) *EmailHandler {
	return &EmailHandler{
		emailService: emailService,
	}
}

// SendEmailRequest 发送邮件请求
type SendEmailRequest struct {
	To          []string          `json:"to" binding:"required,min=1,dive,email"`
	Cc          []string          `json:"cc" binding:"omitempty,dive,email"`
	Bcc         []string          `json:"bcc" binding:"omitempty,dive,email"`
	Subject     string            `json:"subject" binding:"required,min=1,max=200"`
	Body        string            `json:"body" binding:"required,min=1"`
	IsHTML      bool              `json:"is_html"`
	Headers     map[string]string `json:"headers"`
}

// SendEmail 发送邮件
// @Summary 发送邮件
// @Description 通过 OAuth2 XOAUTH2 认证发送邮件
// @Tags Email
// @Accept json
// @Produce json
// @Param request body SendEmailRequest true "邮件内容"
// @Success 200 {object} response.Response{data=SendEmailResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/email/send [post]
func (h *EmailHandler) SendEmail(c *gin.Context) {
	var req SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数无效: "+err.Error())
		return
	}

	// 构建邮件
	email := &types.Email{
		To:      req.To,
		Cc:      req.Cc,
		Bcc:     req.Bcc,
		Subject: req.Subject,
		Body:    req.Body,
		IsHTML:  req.IsHTML,
		Headers: req.Headers,
	}

	// 发送邮件
	status, err := h.emailService.SendEmail(c.Request.Context(), email)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "发送邮件失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"message_id": status.MessageID,
		"sent_at":    status.SentAt,
	})
}

// SendEmailResponse 发送邮件响应
type SendEmailResponse struct {
	MessageID string `json:"message_id"`
	SentAt    string `json:"sent_at"`
}
