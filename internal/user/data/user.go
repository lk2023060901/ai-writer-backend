package data

import (
	"context"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/user/biz"
	"gorm.io/gorm"
)

// UserPO represents the database model
type UserPO struct {
	ID        int64          `gorm:"primarykey"`
	Name      string         `gorm:"size:100;not null"`
	Email     string         `gorm:"size:255;uniqueIndex;not null"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
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

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*biz.User, error) {
	var po UserPO
	if err := r.db.WithContext(ctx).First(&po, id).Error; err != nil {
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

func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&UserPO{}, id).Error
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
