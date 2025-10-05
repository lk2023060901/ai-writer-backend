package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/email/service"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/redis"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/response"
)

const (
	// OAuth2 State 有效期
	stateExpiration = 10 * time.Minute
	// Redis Key 前缀
	stateKeyPrefix = "oauth2:email:state:"
)

// OAuth2Handler OAuth2 授权处理器（生产级实现）
type OAuth2Handler struct {
	emailService *service.EmailService
	redisClient  *redis.Client
}

// NewOAuth2Handler 创建 OAuth2 处理器
func NewOAuth2Handler(emailService *service.EmailService, redisClient *redis.Client) *OAuth2Handler {
	if emailService == nil {
		panic("emailService is required")
	}
	if redisClient == nil {
		panic("redisClient is required for OAuth2 state management")
	}

	return &OAuth2Handler{
		emailService: emailService,
		redisClient:  redisClient,
	}
}

// GetAuthURL 获取 OAuth2 授权 URL
// @Summary 获取邮件服务 OAuth2 授权 URL
// @Description 生成 Google OAuth2 授权 URL,用于获取邮件发送权限
// @Tags Email
// @Produce json
// @Success 200 {object} response.Response{data=AuthURLResponse}
// @Failure 500 {object} response.Response
// @Router /api/v1/email/oauth2/auth-url [get]
func (h *OAuth2Handler) GetAuthURL(c *gin.Context) {
	ctx := c.Request.Context()

	// 生成加密安全的随机 state
	state, err := generateSecureRandomState()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成安全令牌失败")
		return
	}

	// 将 state 存储到 Redis（防 CSRF 攻击）
	stateKey := stateKeyPrefix + state
	if err := h.redisClient.Set(ctx, stateKey, "valid", stateExpiration); err != nil {
		response.Error(c, http.StatusInternalServerError, "存储授权状态失败")
		return
	}

	// 获取授权 URL
	authURL, err := h.emailService.GetAuthURL(state)
	if err != nil {
		// 清理 Redis 中的 state
		_, _ = h.redisClient.Del(ctx, stateKey)
		response.Error(c, http.StatusInternalServerError, "获取授权 URL 失败")
		return
	}

	response.Success(c, gin.H{
		"auth_url": authURL,
		"state":    state,
		"expires_in": int(stateExpiration.Seconds()),
	})
}

// Callback OAuth2 回调处理（生产级实现）
// @Summary OAuth2 授权回调
// @Description 处理 Google OAuth2 授权回调,交换授权码获取 Token（包含完整的安全验证）
// @Tags Email
// @Param code query string true "授权码"
// @Param state query string true "State 参数(防 CSRF)"
// @Param error query string false "错误码"
// @Param error_description query string false "错误描述"
// @Produce json
// @Success 200 {object} response.Response{data=CallbackResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/email/oauth2/callback [get]
func (h *OAuth2Handler) Callback(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. 检查 OAuth2 错误响应
	if errCode := c.Query("error"); errCode != "" {
		errDesc := c.Query("error_description")
		response.Error(c, http.StatusBadRequest,
			fmt.Sprintf("OAuth2 授权失败: %s - %s", errCode, errDesc))
		return
	}

	// 2. 验证必需参数
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		response.Error(c, http.StatusBadRequest, "缺少授权码")
		return
	}
	if state == "" {
		response.Error(c, http.StatusBadRequest, "缺少 state 参数")
		return
	}

	// 3. 验证 state（防 CSRF 攻击）
	stateKey := stateKeyPrefix + state
	exists, err := h.redisClient.Exists(ctx, stateKey)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "验证授权状态失败")
		return
	}
	if exists == 0 {
		response.Error(c, http.StatusUnauthorized, "无效的 state 参数或已过期")
		return
	}

	// 4. 删除已使用的 state（一次性使用）
	if _, err := h.redisClient.Del(ctx, stateKey); err != nil {
		// 删除失败不影响流程，仅记录日志
		// 生产环境应使用结构化日志
	}

	// 5. 交换授权码获取 Token
	authCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := h.emailService.Authorize(authCtx, code); err != nil {
		response.Error(c, http.StatusInternalServerError, "交换授权码失败")
		return
	}

	// 6. 验证授权是否成功
	if !h.emailService.IsAuthorized() {
		response.Error(c, http.StatusInternalServerError, "授权验证失败")
		return
	}

	response.Success(c, gin.H{
		"message": "邮件服务授权成功",
		"authorized": true,
	})
}

// GetStatus 获取 OAuth2 授权状态
// @Summary 获取邮件服务 OAuth2 授权状态
// @Description 检查邮件服务是否已完成 OAuth2 授权
// @Tags Email
// @Produce json
// @Success 200 {object} response.Response{data=StatusResponse}
// @Router /api/v1/email/oauth2/status [get]
func (h *OAuth2Handler) GetStatus(c *gin.Context) {
	authorized := h.emailService.IsAuthorized()

	response.Success(c, gin.H{
		"authorized": authorized,
	})
}

// RevokeAuthorization 撤销授权
// @Summary 撤销邮件服务 OAuth2 授权
// @Description 撤销当前的 OAuth2 访问令牌并清除本地存储
// @Tags Email
// @Produce json
// @Success 200 {object} response.Response{data=RevokeResponse}
// @Failure 500 {object} response.Response
// @Router /api/v1/email/oauth2/revoke [post]
func (h *OAuth2Handler) RevokeAuthorization(c *gin.Context) {
	ctx := c.Request.Context()

	// 撤销授权
	if err := h.emailService.RevokeAuthorization(ctx); err != nil {
		response.Error(c, http.StatusInternalServerError, "撤销授权失败")
		return
	}

	response.Success(c, gin.H{
		"message": "授权已撤销",
		"authorized": false,
	})
}

// generateSecureRandomState 生成加密安全的随机 state (32字节)
func generateSecureRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand.Read failed: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Response 结构体

type AuthURLResponse struct {
	AuthURL   string `json:"auth_url"`
	State     string `json:"state"`
	ExpiresIn int    `json:"expires_in"`
}

type CallbackResponse struct {
	Message    string `json:"message"`
	Authorized bool   `json:"authorized"`
}

type StatusResponse struct {
	Authorized bool `json:"authorized"`
}

type RevokeResponse struct {
	Message    string `json:"message"`
	Authorized bool   `json:"authorized"`
}
