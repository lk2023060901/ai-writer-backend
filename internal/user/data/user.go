package data

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/auth"
	"github.com/lk2023060901/ai-writer-backend/internal/user/biz"
	"gorm.io/gorm"
)

// BackupCodesJSON 自定义 JSONB 类型（用于存储备用恢复码）
type BackupCodesJSON []auth.BackupCode

func (j *BackupCodesJSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

func (j BackupCodesJSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// UserPO represents the database model
type UserPO struct {
	ID        string         `gorm:"type:uuid;primarykey"`
	Name      string         `gorm:"size:100;not null"`
	Email     string         `gorm:"size:255;not null;uniqueIndex:idx_users_email,where:deleted_at IS NULL"`
	EmailVerified bool       `gorm:"not null;default:false"`

	// 认证信息
	PasswordHash string `gorm:"size:255;not null"`

	// JWT Refresh Token
	RefreshToken         *string    `gorm:"size:512"`
	RefreshTokenExpiresAt *time.Time

	// 双因子认证
	TwoFactorEnabled     bool            `gorm:"not null;default:false"`
	TwoFactorSecret      *string         `gorm:"size:32"`
	TwoFactorBackupCodes BackupCodesJSON `gorm:"type:jsonb"`

	// 登录追踪
	LastLoginAt         *time.Time
	LastLoginIP         *string `gorm:"size:45"`
	FailedLoginAttempts int     `gorm:"not null;default:0"`
	LockedUntil         *time.Time

	// 邮箱验证
	EmailVerificationToken     *string    `gorm:"size:64;index:idx_users_email_verification_token,where:email_verification_token IS NOT NULL"`
	EmailVerificationExpiresAt *time.Time

	// 密码重置
	PasswordResetToken     *string    `gorm:"size:64;index:idx_users_password_reset_token,where:password_reset_token IS NOT NULL"`
	PasswordResetExpiresAt *time.Time

	// 时间戳
	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (UserPO) TableName() string {
	return "users"
}

// UserRepo implements biz.UserRepo interface
type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) biz.UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *biz.User) error {
	po := &UserPO{
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}

	user.ID = po.ID
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*biz.User, error) {
	var po UserPO
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		return nil, err
	}

	return r.toUser(&po), nil
}

func (r *UserRepo) List(ctx context.Context, offset, limit int) ([]*biz.User, error) {
	var pos []UserPO
	if err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&pos).Error; err != nil {
		return nil, err
	}

	users := make([]*biz.User, len(pos))
	for i, po := range pos {
		users[i] = r.toUser(&po)
	}

	return users, nil
}

func (r *UserRepo) Update(ctx context.Context, user *biz.User) error {
	po := &UserPO{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		UpdatedAt: user.UpdatedAt,
	}

	return r.db.WithContext(ctx).Model(po).Updates(po).Error
}

func (r *UserRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&UserPO{}).Error
}

func (r *UserRepo) toUser(po *UserPO) *biz.User {
	return &biz.User{
		ID:        po.ID,
		Name:      po.Name,
		Email:     po.Email,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}
}
