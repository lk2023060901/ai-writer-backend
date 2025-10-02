package biz

import (
	"context"
	"time"
)

// User represents the domain model
type User struct {
	ID        int64
	Name      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserRepo defines the interface for user data operations
type UserRepo interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	List(ctx context.Context, offset, limit int) ([]*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id int64) error
}

// UserUseCase contains business logic for user operations
type UserUseCase struct {
	repo UserRepo
}

func NewUserUseCase(repo UserRepo) *UserUseCase {
	return &UserUseCase{repo: repo}
}

func (uc *UserUseCase) CreateUser(ctx context.Context, name, email string) (*User, error) {
	user := &User{
		Name:      name,
		Email:     email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := uc.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (uc *UserUseCase) GetUser(ctx context.Context, id int64) (*User, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *UserUseCase) ListUsers(ctx context.Context, page, pageSize int) ([]*User, error) {
	offset := (page - 1) * pageSize
	return uc.repo.List(ctx, offset, pageSize)
}

func (uc *UserUseCase) UpdateUser(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now()
	return uc.repo.Update(ctx, user)
}

func (uc *UserUseCase) DeleteUser(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}
