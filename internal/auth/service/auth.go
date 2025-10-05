package service

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/auth/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/response"
	"go.uber.org/zap"
)

// AuthService 认证服务
type AuthService struct {
	authUC *biz.AuthUseCase
	logger *logger.Logger
}

// NewAuthService 创建认证服务
func NewAuthService(authUC *biz.AuthUseCase, log *logger.Logger) *AuthService {
	return &AuthService{
		authUC: authUC,
		logger: log,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	UserID string  `json:"user_id"`
	Email  string `json:"email"`
	Message string `json:"message"`
}

// Register 用户注册
// @Summary 用户注册
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "注册信息"
// @Success 201 {object} RegisterResponse
// @Router /auth/register [post]
func (s *AuthService) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, err := s.authUC.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		s.logger.Error("failed to register user", zap.Error(err), zap.String("email", req.Email))

		if err == biz.ErrEmailAlreadyExists {
			response.Error(c, http.StatusConflict, "邮箱已存在")
			return
		}

		response.InternalError(c, "注册失败")
		return
	}

	response.Created(c, gin.H{
		"user_id": user.ID,
		"email":   user.Email,
		"message": "注册成功,请验证您的邮箱",
	})
}

// LoginRequest 登录请求
type LoginRequest struct {
	Account    string `json:"account" binding:"required"`     // 用户名或邮箱
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"remember_me"`                    // 90天免登录
}

// LoginResponse 登录响应
type LoginResponse struct {
	Require2FA    bool           `json:"require_2fa"`
	PendingAuthID string         `json:"pending_auth_id,omitempty"` // 当 require_2fa=true 时返回
	Tokens        *biz.TokenPair `json:"tokens,omitempty"`           // 当 require_2fa=false 时返回
}

// Login 用户登录
// @Summary 用户登录
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录信息"
// @Success 200 {object} LoginResponse
// @Router /auth/login [post]
func (s *AuthService) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 获取客户端 IP
	ip := c.ClientIP()

	result, err := s.authUC.Login(c.Request.Context(), req.Account, req.Password, ip, req.RememberMe)
	if err != nil {
		s.logger.Warn("login failed",
			zap.Error(err),
			zap.String("account", req.Account),
			zap.String("ip", ip))

		switch err {
		case biz.ErrInvalidCredentials:
			response.Unauthorized(c, "账号或密码错误")
		case biz.ErrAccountLocked:
			response.Forbidden(c, "账号已被锁定,请15分钟后重试")
		default:
			response.InternalError(c, "登录失败")
		}
		return
	}

	// 构造响应数据
	data := gin.H{
		"require_2fa": result.Require2FA,
	}
	if result.Require2FA {
		data["pending_auth_id"] = result.PendingAuthID
	} else {
		data["tokens"] = result.Tokens
	}

	response.Success(c, data)
}

// Verify2FARequest 验证 2FA 请求
type Verify2FARequest struct {
	PendingAuthID string `json:"pending_auth_id" binding:"required"`
	Code          string `json:"code" binding:"required,len=6"`
}

// Verify2FAResponse 验证 2FA 响应
type Verify2FAResponse struct {
	Tokens *biz.TokenPair `json:"tokens"`
}

// Verify2FA 验证 2FA 代码
// @Summary 验证双因子认证代码
// @Tags auth
// @Accept json
// @Produce json
// @Param request body Verify2FARequest true "2FA 验证码"
// @Success 200 {object} Verify2FAResponse
// @Router /auth/2fa/verify [post]
func (s *AuthService) Verify2FA(c *gin.Context) {
	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := s.authUC.Verify2FA(c.Request.Context(), req.PendingAuthID, req.Code)
	if err != nil {
		s.logger.Warn("2FA verification failed",
			zap.Error(err),
			zap.String("pending_auth_id", req.PendingAuthID))

		switch err {
		case biz.ErrPendingAuthNotFound, biz.ErrPendingAuthExpired:
			response.NotFound(c, "验证会话已过期,请重新登录")
		case biz.ErrTooManyAttempts:
			response.Error(c, http.StatusTooManyRequests, "验证尝试次数过多,请重新登录")
		case biz.ErrInvalid2FACode:
			response.Unauthorized(c, "验证码错误")
		default:
			response.InternalError(c, "2FA验证失败")
		}
		return
	}

	response.Success(c, gin.H{
		"tokens": result.Tokens,
	})
}

// RefreshTokenRequest 刷新 token 请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse 刷新 token 响应
type RefreshTokenResponse struct {
	*biz.TokenPair
}

// RefreshToken 刷新 access token
// @Summary 刷新访问令牌
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh Token"
// @Success 200 {object} RefreshTokenResponse
// @Router /auth/refresh [post]
func (s *AuthService) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tokens, err := s.authUC.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		s.logger.Warn("token refresh failed", zap.Error(err))

		if err == biz.ErrInvalidToken {
			response.Unauthorized(c, "refresh token无效或已过期")
			return
		}

		response.InternalError(c, "刷新token失败")
		return
	}

	response.Success(c, tokens)
}

// Enable2FAResponse 启用 2FA 响应
type Enable2FAResponse struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`      // 二维码下载 URL
	BackupCodes []string `json:"backup_codes"`
}

// Enable2FA 启用双因子认证
// @Summary 启用双因子认证
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} Enable2FAResponse
// @Router /auth/2fa/enable [post]
func (s *AuthService) Enable2FA(c *gin.Context) {
	// 从上下文获取用户 ID（由中间件注入）
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}

	setup, err := s.authUC.Enable2FA(c.Request.Context(), userID.(string))
	if err != nil {
		s.logger.Error("failed to enable 2FA", zap.Error(err), zap.String("user_id", userID.(string)))
		response.InternalError(c, "启用2FA失败")
		return
	}

	response.Success(c, gin.H{
		"secret":       setup.Secret,
		"qr_code_url":  "/api/v1/auth/2fa/qrcode",
		"backup_codes": setup.BackupCodes,
	})

	// 同时设置二维码到上下文供下次请求使用
	// 实际生产环境应该用 Redis 缓存
	c.Set("qr_code", setup.QRCode)
}

// GetQRCode 获取 2FA 二维码
// @Summary 获取双因子认证二维码
// @Tags auth
// @Security BearerAuth
// @Produce image/png
// @Success 200 {file} binary
// @Router /auth/2fa/qrcode [get]
func (s *AuthService) GetQRCode(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}

	// 重新生成二维码（实际应该从缓存读取）
	setup, err := s.authUC.Enable2FA(c.Request.Context(), userID.(string))
	if err != nil {
		s.logger.Error("failed to get QR code", zap.Error(err))
		response.InternalError(c, "获取二维码失败")
		return
	}

	c.Data(http.StatusOK, "image/png", setup.QRCode)
}

// Confirm2FARequest 确认启用 2FA 请求
type Confirm2FARequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

// Confirm2FA 确认启用 2FA（验证第一个验证码）
// @Summary 确认启用双因子认证
// @Tags auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body Confirm2FARequest true "验证码"
// @Success 200 {object} map[string]string
// @Router /auth/2fa/confirm [post]
func (s *AuthService) Confirm2FA(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}

	var req Confirm2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := s.authUC.Confirm2FA(c.Request.Context(), userID.(string), req.Code); err != nil {
		s.logger.Warn("2FA confirmation failed", zap.Error(err), zap.String("user_id", userID.(string)))

		if err == biz.ErrInvalid2FACode {
			response.Unauthorized(c, "验证码错误")
			return
		}

		response.InternalError(c, "确认2FA失败")
		return
	}

	response.SuccessWithMessage(c, "2FA已成功启用", nil)
}

// Disable2FARequest 禁用 2FA 请求
type Disable2FARequest struct {
	Code string `json:"code" binding:"required,len=6"`
}

// Disable2FA 禁用双因子认证
// @Summary 禁用双因子认证
// @Tags auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body Disable2FARequest true "验证码"
// @Success 200 {object} map[string]string
// @Router /auth/2fa/disable [post]
func (s *AuthService) Disable2FA(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}

	var req Disable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := s.authUC.Disable2FA(c.Request.Context(), userID.(string), req.Code); err != nil {
		s.logger.Warn("2FA disable failed", zap.Error(err), zap.String("user_id", userID.(string)))

		if err == biz.ErrInvalid2FACode {
			response.Unauthorized(c, "验证码错误")
			return
		}

		response.InternalError(c, "禁用2FA失败")
		return
	}

	response.SuccessWithMessage(c, "2FA已成功禁用", nil)
}

// RegisterRoutes 注册路由
func (s *AuthService) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		// 公开端点
		auth.POST("/register", s.Register)
		auth.POST("/login", s.Login)
		auth.POST("/2fa/verify", s.Verify2FA)
		auth.POST("/refresh", s.RefreshToken)

		// 需要认证的端点（需要在路由注册时添加中间件）
		// protected := auth.Use(middleware.JWTAuth())
		// {
		//     protected.POST("/2fa/enable", s.Enable2FA)
		//     protected.GET("/2fa/qrcode", s.GetQRCode)
		//     protected.POST("/2fa/confirm", s.Confirm2FA)
		//     protected.POST("/2fa/disable", s.Disable2FA)
		// }
	}
}

// GetClientIP 获取客户端真实 IP
func GetClientIP(c *gin.Context) string {
	// 优先从 X-Forwarded-For 获取
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 从 X-Real-IP 获取
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	// 最后使用 RemoteAddr
	return c.ClientIP()
}
