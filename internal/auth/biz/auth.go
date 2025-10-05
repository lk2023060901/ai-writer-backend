package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lk2023060901/ai-writer-backend/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrUserNotFound        = errors.New("user not found")
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrAccountLocked       = errors.New("account is locked due to too many failed login attempts")
	ErrInvalid2FACode      = errors.New("invalid 2FA code")
	ErrEmailNotVerified    = errors.New("email not verified")
	ErrInvalidToken        = errors.New("invalid or expired token")
	ErrPendingAuthNotFound = errors.New("pending auth not found or expired")
	ErrPendingAuthExpired  = errors.New("pending auth expired")
	ErrTooManyAttempts     = errors.New("too many verification attempts")
)

// User 认证相关的用户模型
type User struct {
	ID                  string // UUID v7
	Name                string
	Email               string
	PasswordHash        string
	EmailVerified       bool
	TwoFactorEnabled    bool
	TwoFactorSecret     *string
	TwoFactorBackupCodes []auth.BackupCode
	FailedLoginAttempts int
	LockedUntil         *time.Time
	LastLoginAt         *time.Time
	LastLoginIP         *string
	RefreshToken        *string
	RefreshTokenExpiresAt *time.Time
	EmailVerificationToken *string
	EmailVerificationExpiresAt *time.Time
	PasswordResetToken *string
	PasswordResetExpiresAt *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// UserRepo 用户仓库接口
type UserRepo interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByEmailOrName(ctx context.Context, account string) (*User, error) // 通过邮箱或姓名查找
	GetByRefreshToken(ctx context.Context, refreshToken string) (*User, error)
	Update(ctx context.Context, user *User) error
	UpdateLoginInfo(ctx context.Context, userID string, ip string) error
	IncrementFailedLogins(ctx context.Context, userID string) error
	ResetFailedLogins(ctx context.Context, userID string) error
	LockAccount(ctx context.Context, userID string, duration time.Duration) error
}

// AuthUseCase 认证业务逻辑
type AuthUseCase struct {
	userRepo        UserRepo
	pendingAuthRepo PendingAuthRepo
	jwtManager      *auth.JWTManager
	totpManager     *auth.TOTPManager
}

func NewAuthUseCase(userRepo UserRepo, pendingAuthRepo PendingAuthRepo, jwtSecret string, issuer string) *AuthUseCase {
	return &AuthUseCase{
		userRepo:        userRepo,
		pendingAuthRepo: pendingAuthRepo,
		jwtManager:      auth.NewJWTManager(jwtSecret),
		totpManager:     auth.NewTOTPManager(issuer),
	}
}

// Register 用户注册
func (uc *AuthUseCase) Register(ctx context.Context, name, email, password string) (*User, error) {
	// 检查邮箱是否已存在
	existingUser, err := uc.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	// 哈希密码
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 生成邮箱验证 token
	verificationToken, err := auth.GenerateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification token: %w", err)
	}

	// 生成 UUID v7 (时间有序)
	userID := uuid.Must(uuid.NewV7()).String()

	expiresAt := time.Now().Add(24 * time.Hour)

	user := &User{
		ID:                         userID,
		Name:                       name,
		Email:                      email,
		PasswordHash:               string(passwordHash),
		EmailVerified:              false,
		EmailVerificationToken:     &verificationToken,
		EmailVerificationExpiresAt: &expiresAt,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login 用户登录（第一步：验证密码）
func (uc *AuthUseCase) Login(ctx context.Context, account, password, ip string, rememberMe bool) (*LoginResult, error) {
	// 通过邮箱或姓名查找用户
	user, err := uc.userRepo.GetByEmailOrName(ctx, account)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 检查账户是否被锁定
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, ErrAccountLocked
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		// 密码错误，增加失败次数
		uc.userRepo.IncrementFailedLogins(ctx, user.ID)

		// 5 次失败后锁定 15 分钟
		if user.FailedLoginAttempts+1 >= 5 {
			uc.userRepo.LockAccount(ctx, user.ID, 15*time.Minute)
		}

		return nil, ErrInvalidCredentials
	}

	// 检查是否需要 2FA
	if user.TwoFactorEnabled {
		// 创建 pending auth
		pendingAuth := NewPendingAuth(user.ID, user.Email, ip)
		if err := uc.pendingAuthRepo.Create(ctx, pendingAuth); err != nil {
			return nil, fmt.Errorf("failed to create pending auth: %w", err)
		}

		return &LoginResult{
			Require2FA:    true,
			PendingAuthID: pendingAuth.ID,
			UserID:        user.ID,
		}, nil
	}

	// 不需要 2FA，直接生成 token
	return uc.generateTokens(ctx, user, ip, rememberMe)
}

// Verify2FA 验证 2FA 代码（第二步）
func (uc *AuthUseCase) Verify2FA(ctx context.Context, pendingAuthID, code string) (*LoginResult, error) {
	// 从 Redis 获取 pending auth
	pendingAuth, err := uc.pendingAuthRepo.Get(ctx, pendingAuthID)
	if err != nil {
		if err == ErrPendingAuthNotFound || err == ErrPendingAuthExpired {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get pending auth: %w", err)
	}

	// 检查尝试次数
	if pendingAuth.Attempts >= MaxVerifyAttempts {
		_ = uc.pendingAuthRepo.Delete(ctx, pendingAuthID)
		return nil, ErrTooManyAttempts
	}

	// 获取用户信息
	user, err := uc.userRepo.GetByID(ctx, pendingAuth.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if !user.TwoFactorEnabled || user.TwoFactorSecret == nil {
		return nil, ErrInvalid2FACode
	}

	// 先尝试 TOTP 验证
	if uc.totpManager.ValidateCode(*user.TwoFactorSecret, code) {
		// TOTP 验证成功，删除 pending auth
		_ = uc.pendingAuthRepo.Delete(ctx, pendingAuthID)
		return uc.generateTokens(ctx, user, pendingAuth.IP, false) // 2FA验证时暂不支持记住我
	}

	// 尝试备用恢复码
	index, valid, err := auth.VerifyBackupCode(user.TwoFactorBackupCodes, code)
	if err != nil {
		return nil, err
	}

	if valid {
		// 标记恢复码为已使用
		auth.MarkBackupCodeAsUsed(user.TwoFactorBackupCodes, index, &pendingAuth.IP)
		user.UpdatedAt = time.Now()
		if err := uc.userRepo.Update(ctx, user); err != nil {
			return nil, err
		}

		// 删除 pending auth
		_ = uc.pendingAuthRepo.Delete(ctx, pendingAuthID)
		return uc.generateTokens(ctx, user, pendingAuth.IP, false) // 2FA验证时暂不支持记住我
	}

	// 验证失败，增加尝试次数
	_ = uc.pendingAuthRepo.IncrementAttempts(ctx, pendingAuthID)
	uc.userRepo.IncrementFailedLogins(ctx, user.ID)
	return nil, ErrInvalid2FACode
}

// RefreshAccessToken 刷新 Access Token
func (uc *AuthUseCase) RefreshAccessToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	user, err := uc.userRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// 检查 refresh token 是否过期
	if user.RefreshTokenExpiresAt == nil || user.RefreshTokenExpiresAt.Before(time.Now()) {
		return nil, ErrInvalidToken
	}

	// 生成新的 access token
	accessToken, err := uc.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken, // 复用原有的 refresh token
		ExpiresIn:    int(auth.AccessTokenDuration.Seconds()),
	}, nil
}

// Enable2FA 启用双因子认证
func (uc *AuthUseCase) Enable2FA(ctx context.Context, userID string) (*TwoFactorSetup, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// 生成 TOTP 密钥
	secret, otpURL, err := uc.totpManager.GenerateSecret(user.Email)
	if err != nil {
		return nil, err
	}

	// 生成二维码
	qrCode, err := uc.totpManager.GenerateQRCode(otpURL, 256)
	if err != nil {
		return nil, err
	}

	// 生成备用恢复码
	plainCodes, backupCodes, err := auth.GenerateBackupCodes(auth.BackupCodeCount)
	if err != nil {
		return nil, err
	}

	// 保存到数据库（但尚未启用）
	user.TwoFactorSecret = &secret
	user.TwoFactorBackupCodes = backupCodes
	user.TwoFactorEnabled = false // 需要用户验证后才启用
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return &TwoFactorSetup{
		Secret:      secret,
		QRCode:      qrCode,
		BackupCodes: plainCodes,
	}, nil
}

// Confirm2FA 确认启用 2FA（用户输入第一个验证码）
func (uc *AuthUseCase) Confirm2FA(ctx context.Context, userID string, code string) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if user.TwoFactorSecret == nil {
		return errors.New("2FA not initialized")
	}

	// 验证验证码
	if !uc.totpManager.ValidateCode(*user.TwoFactorSecret, code) {
		return ErrInvalid2FACode
	}

	// 启用 2FA
	user.TwoFactorEnabled = true
	user.UpdatedAt = time.Now()

	return uc.userRepo.Update(ctx, user)
}

// Disable2FA 禁用双因子认证
func (uc *AuthUseCase) Disable2FA(ctx context.Context, userID string, code string) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if !user.TwoFactorEnabled || user.TwoFactorSecret == nil {
		return errors.New("2FA not enabled")
	}

	// 验证验证码
	if !uc.totpManager.ValidateCode(*user.TwoFactorSecret, code) {
		return ErrInvalid2FACode
	}

	// 禁用 2FA
	user.TwoFactorEnabled = false
	user.TwoFactorSecret = nil
	user.TwoFactorBackupCodes = nil
	user.UpdatedAt = time.Now()

	return uc.userRepo.Update(ctx, user)
}

// generateTokens 生成 token 对
func (uc *AuthUseCase) generateTokens(ctx context.Context, user *User, ip string, rememberMe bool) (*LoginResult, error) {
	// 生成 access token
	accessToken, err := uc.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	// 生成 refresh token
	refreshToken, err := uc.jwtManager.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// 保存 refresh token - 如果勾选"记住我",则90天有效期,否则7天
	var tokenDuration time.Duration
	if rememberMe {
		tokenDuration = 90 * 24 * time.Hour // 90天
	} else {
		tokenDuration = auth.RefreshTokenDuration // 默认7天
	}

	expiresAt := time.Now().Add(tokenDuration)
	user.RefreshToken = &refreshToken
	user.RefreshTokenExpiresAt = &expiresAt

	// 更新登录信息
	now := time.Now()
	user.LastLoginAt = &now
	user.LastLoginIP = &ip
	user.FailedLoginAttempts = 0
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return &LoginResult{
		Require2FA: false,
		Tokens: &TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    int(auth.AccessTokenDuration.Seconds()),
		},
	}, nil
}

// LoginResult 登录结果
type LoginResult struct {
	Require2FA     bool
	PendingAuthID  string // 当 Require2FA=true 时返回
	UserID         string
	Tokens         *TokenPair
}

// TokenPair token 对
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // 秒
}

// TwoFactorSetup 2FA 设置信息
type TwoFactorSetup struct {
	Secret      string
	QRCode      []byte   // PNG 图片
	BackupCodes []string // 明文恢复码（仅显示一次）
}

