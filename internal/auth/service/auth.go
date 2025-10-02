package service

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/auth/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := s.authUC.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		s.logger.Error("failed to register user", zap.Error(err), zap.String("email", req.Email))

		if err == biz.ErrEmailAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		UserID:  user.ID,
		Email:   user.Email,
		Message: "Registration successful. Please verify your email.",
	})
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取客户端 IP
	ip := c.ClientIP()

	result, err := s.authUC.Login(c.Request.Context(), req.Email, req.Password, ip)
	if err != nil {
		s.logger.Warn("login failed",
			zap.Error(err),
			zap.String("email", req.Email),
			zap.String("ip", ip))

		switch err {
		case biz.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		case biz.ErrAccountLocked:
			c.JSON(http.StatusForbidden, gin.H{"error": "account locked due to too many failed attempts"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		}
		return
	}

	resp := LoginResponse{
		Require2FA: result.Require2FA,
	}
	if result.Require2FA {
		resp.PendingAuthID = result.PendingAuthID
	} else {
		resp.Tokens = result.Tokens
	}
	c.JSON(http.StatusOK, resp)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.authUC.Verify2FA(c.Request.Context(), req.PendingAuthID, req.Code)
	if err != nil {
		s.logger.Warn("2FA verification failed",
			zap.Error(err),
			zap.String("pending_auth_id", req.PendingAuthID))

		switch err {
		case biz.ErrPendingAuthNotFound, biz.ErrPendingAuthExpired:
			c.JSON(http.StatusNotFound, gin.H{"error": "pending auth not found or expired, please login again"})
		case biz.ErrTooManyAttempts:
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many verification attempts, please login again"})
		case biz.ErrInvalid2FACode:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid 2FA code"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "2FA verification failed"})
		}
		return
	}

	c.JSON(http.StatusOK, Verify2FAResponse{
		Tokens: result.Tokens,
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokens, err := s.authUC.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		s.logger.Warn("token refresh failed", zap.Error(err))

		if err == biz.ErrInvalidToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "token refresh failed"})
		return
	}

	c.JSON(http.StatusOK, RefreshTokenResponse{TokenPair: tokens})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	setup, err := s.authUC.Enable2FA(c.Request.Context(), userID.(string))
	if err != nil {
		s.logger.Error("failed to enable 2FA", zap.Error(err), zap.String("user_id", userID.(string)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable 2FA"})
		return
	}

	// 将二维码保存到临时存储，生成下载 URL
	// 这里简化处理，直接返回 base64 编码的图片
	c.JSON(http.StatusOK, Enable2FAResponse{
		Secret:      setup.Secret,
		QRCodeURL:   "/auth/2fa/qrcode", // 前端需要再次请求获取二维码
		BackupCodes: setup.BackupCodes,
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 重新生成二维码（实际应该从缓存读取）
	setup, err := s.authUC.Enable2FA(c.Request.Context(), userID.(string))
	if err != nil {
		s.logger.Error("failed to get QR code", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get QR code"})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req Confirm2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.authUC.Confirm2FA(c.Request.Context(), userID.(string), req.Code); err != nil {
		s.logger.Warn("2FA confirmation failed", zap.Error(err), zap.String("user_id", userID.(string)))

		if err == biz.ErrInvalid2FACode {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid verification code"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm 2FA"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "2FA enabled successfully"})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req Disable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.authUC.Disable2FA(c.Request.Context(), userID.(string), req.Code); err != nil {
		s.logger.Warn("2FA disable failed", zap.Error(err), zap.String("user_id", userID.(string)))

		if err == biz.ErrInvalid2FACode {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid verification code"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable 2FA"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "2FA disabled successfully"})
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
