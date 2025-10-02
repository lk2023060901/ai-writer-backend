package data

import (
	"context"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/auth"
	"github.com/lk2023060901/ai-writer-backend/internal/auth/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
	"github.com/lk2023060901/ai-writer-backend/internal/user/data"
	"gorm.io/gorm"
)

// AuthUserRepo 认证用户仓库
// 使用 internal/pkg/database 封装
type AuthUserRepo struct {
	db *database.DB
}

// NewAuthUserRepo 创建认证用户仓库
func NewAuthUserRepo(db *database.DB) biz.UserRepo {
	return &AuthUserRepo{db: db}
}

// Create 创建用户
func (r *AuthUserRepo) Create(ctx context.Context, user *biz.User) error {
	po := r.toUserPO(user)
	if err := r.db.WithContext(ctx).GetDB().Create(po).Error; err != nil {
		return err
	}
	user.ID = po.ID
	return nil
}

// GetByID 根据 ID 获取用户
func (r *AuthUserRepo) GetByID(ctx context.Context, id string) (*biz.User, error) {
	var po data.UserPO
	if err := r.db.WithContext(ctx).GetDB().
		Where("id = ? AND deleted_at IS NULL", id).
		First(&po).Error; err != nil {
		if database.IsRecordNotFoundError(err) {
			return nil, biz.ErrUserNotFound
		}
		return nil, err
	}
	return r.toBizUser(&po), nil
}

// GetByEmail 根据邮箱获取用户
func (r *AuthUserRepo) GetByEmail(ctx context.Context, email string) (*biz.User, error) {
	var po data.UserPO
	if err := r.db.WithContext(ctx).GetDB().
		Where("email = ? AND deleted_at IS NULL", email).
		First(&po).Error; err != nil {
		if database.IsRecordNotFoundError(err) {
			return nil, biz.ErrUserNotFound
		}
		return nil, err
	}
	return r.toBizUser(&po), nil
}

// Update 更新用户
func (r *AuthUserRepo) Update(ctx context.Context, user *biz.User) error {
	po := r.toUserPO(user)
	return r.db.WithContext(ctx).GetDB().
		Model(&data.UserPO{}).
		Where("id = ?", user.ID).
		Updates(po).Error
}

// UpdateLoginInfo 更新登录信息
func (r *AuthUserRepo) UpdateLoginInfo(ctx context.Context, userID string, ip string) error {
	now := time.Now()
	return r.db.WithContext(ctx).GetDB().
		Model(&data.UserPO{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"last_login_at": now,
			"last_login_ip": ip,
			"updated_at":    now,
		}).Error
}

// IncrementFailedLogins 增加失败登录次数
func (r *AuthUserRepo) IncrementFailedLogins(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).GetDB().
		Model(&data.UserPO{}).
		Where("id = ?", userID).
		UpdateColumn("failed_login_attempts", gorm.Expr("failed_login_attempts + 1")).Error
}

// ResetFailedLogins 重置失败登录次数
func (r *AuthUserRepo) ResetFailedLogins(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).GetDB().
		Model(&data.UserPO{}).
		Where("id = ?", userID).
		Update("failed_login_attempts", 0).Error
}

// LockAccount 锁定账户
func (r *AuthUserRepo) LockAccount(ctx context.Context, userID string, duration time.Duration) error {
	lockedUntil := time.Now().Add(duration)
	return r.db.WithContext(ctx).GetDB().
		Model(&data.UserPO{}).
		Where("id = ?", userID).
		Update("locked_until", lockedUntil).Error
}

// GetByRefreshToken 根据 refresh token 获取用户
func (r *AuthUserRepo) GetByRefreshToken(ctx context.Context, refreshToken string) (*biz.User, error) {
	var po data.UserPO
	if err := r.db.WithContext(ctx).GetDB().
		Where("refresh_token = ? AND deleted_at IS NULL", refreshToken).
		First(&po).Error; err != nil {
		if database.IsRecordNotFoundError(err) {
			return nil, biz.ErrInvalidToken
		}
		return nil, err
	}
	return r.toBizUser(&po), nil
}

// toUserPO 业务模型转数据模型
func (r *AuthUserRepo) toUserPO(user *biz.User) *data.UserPO {
	if user == nil {
		return nil
	}

	return &data.UserPO{
		ID:                         user.ID,
		Name:                       user.Name,
		Email:                      user.Email,
		PasswordHash:               user.PasswordHash,
		EmailVerified:              user.EmailVerified,
		RefreshToken:               user.RefreshToken,
		RefreshTokenExpiresAt:      user.RefreshTokenExpiresAt,
		TwoFactorEnabled:           user.TwoFactorEnabled,
		TwoFactorSecret:            user.TwoFactorSecret,
		TwoFactorBackupCodes:       data.BackupCodesJSON(user.TwoFactorBackupCodes),
		LastLoginAt:                user.LastLoginAt,
		LastLoginIP:                user.LastLoginIP,
		FailedLoginAttempts:        user.FailedLoginAttempts,
		LockedUntil:                user.LockedUntil,
		EmailVerificationToken:     user.EmailVerificationToken,
		EmailVerificationExpiresAt: user.EmailVerificationExpiresAt,
		PasswordResetToken:         user.PasswordResetToken,
		PasswordResetExpiresAt:     user.PasswordResetExpiresAt,
		CreatedAt:                  user.CreatedAt,
		UpdatedAt:                  user.UpdatedAt,
	}
}

// toBizUser 数据模型转业务模型
func (r *AuthUserRepo) toBizUser(po *data.UserPO) *biz.User {
	if po == nil {
		return nil
	}

	return &biz.User{
		ID:                         po.ID,
		Name:                       po.Name,
		Email:                      po.Email,
		PasswordHash:               po.PasswordHash,
		EmailVerified:              po.EmailVerified,
		TwoFactorEnabled:           po.TwoFactorEnabled,
		TwoFactorSecret:            po.TwoFactorSecret,
		TwoFactorBackupCodes:       []auth.BackupCode(po.TwoFactorBackupCodes),
		FailedLoginAttempts:        po.FailedLoginAttempts,
		LockedUntil:                po.LockedUntil,
		LastLoginAt:                po.LastLoginAt,
		LastLoginIP:                po.LastLoginIP,
		RefreshToken:               po.RefreshToken,
		RefreshTokenExpiresAt:      po.RefreshTokenExpiresAt,
		EmailVerificationToken:     po.EmailVerificationToken,
		EmailVerificationExpiresAt: po.EmailVerificationExpiresAt,
		PasswordResetToken:         po.PasswordResetToken,
		PasswordResetExpiresAt:     po.PasswordResetExpiresAt,
		CreatedAt:                  po.CreatedAt,
		UpdatedAt:                  po.UpdatedAt,
	}
}
