package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/redis"
)

const (
	// PendingAuthTTL pending auth 的过期时间（5分钟）
	PendingAuthTTL = 5 * time.Minute

	// MaxVerifyAttempts 最大验证尝试次数
	MaxVerifyAttempts = 3

	// PendingAuthKeyPrefix Redis key 前缀
	PendingAuthKeyPrefix = "pending_auth:"
)

// PendingAuth 待验证的认证信息
type PendingAuth struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"` // UUID
	Email     string    `json:"email"`
	IP        string    `json:"ip"`
	Attempts  int       `json:"attempts"`   // 已尝试次数
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PendingAuthRepo pending auth 仓储接口
type PendingAuthRepo interface {
	// Create 创建 pending auth
	Create(ctx context.Context, auth *PendingAuth) error

	// Get 获取 pending auth
	Get(ctx context.Context, id string) (*PendingAuth, error)

	// IncrementAttempts 增加尝试次数
	IncrementAttempts(ctx context.Context, id string) error

	// Delete 删除 pending auth
	Delete(ctx context.Context, id string) error
}

// RedisPendingAuthRepo Redis 实现
type RedisPendingAuthRepo struct {
	client *redis.Client
}

// NewRedisPendingAuthRepo 创建 Redis pending auth repo
func NewRedisPendingAuthRepo(client *redis.Client) PendingAuthRepo {
	return &RedisPendingAuthRepo{
		client: client,
	}
}

// Create 创建 pending auth
func (r *RedisPendingAuthRepo) Create(ctx context.Context, auth *PendingAuth) error {
	data, err := json.Marshal(auth)
	if err != nil {
		return fmt.Errorf("failed to marshal pending auth: %w", err)
	}

	key := PendingAuthKeyPrefix + auth.ID
	return r.client.Set(ctx, key, string(data), PendingAuthTTL)
}

// Get 获取 pending auth
func (r *RedisPendingAuthRepo) Get(ctx context.Context, id string) (*PendingAuth, error) {
	key := PendingAuthKeyPrefix + id

	data, err := r.client.Get(ctx, key)
	if err != nil {
		if redis.IsNil(err) {
			return nil, ErrPendingAuthNotFound
		}
		return nil, fmt.Errorf("failed to get pending auth: %w", err)
	}

	var auth PendingAuth
	if err := json.Unmarshal([]byte(data), &auth); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pending auth: %w", err)
	}

	// 检查是否过期
	if time.Now().After(auth.ExpiresAt) {
		_ = r.Delete(ctx, id)
		return nil, ErrPendingAuthExpired
	}

	return &auth, nil
}

// IncrementAttempts 增加尝试次数
func (r *RedisPendingAuthRepo) IncrementAttempts(ctx context.Context, id string) error {
	auth, err := r.Get(ctx, id)
	if err != nil {
		return err
	}

	auth.Attempts++

	// 如果超过最大尝试次数，删除记录
	if auth.Attempts >= MaxVerifyAttempts {
		return r.Delete(ctx, id)
	}

	// 更新记录
	return r.Create(ctx, auth)
}

// Delete 删除 pending auth
func (r *RedisPendingAuthRepo) Delete(ctx context.Context, id string) error {
	key := PendingAuthKeyPrefix + id
	_, err := r.client.Del(ctx, key)
	return err
}

// NewPendingAuth 创建新的 pending auth
func NewPendingAuth(userID string, email, ip string) *PendingAuth {
	now := time.Now()
	return &PendingAuth{
		ID:        uuid.New().String(),
		UserID:    userID,
		Email:     email,
		IP:        ip,
		Attempts:  0,
		CreatedAt: now,
		ExpiresAt: now.Add(PendingAuthTTL),
	}
}
