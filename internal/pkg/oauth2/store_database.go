package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

// OAuth2Token 数据库模型
type OAuth2Token struct {
	ID           uint      `gorm:"primaryKey"`
	Provider     string    `gorm:"uniqueIndex;not null;size:50;comment:OAuth2 提供商标识(如 gmail)"` // 唯一标识
	AccessToken  string    `gorm:"type:text;not null;comment:访问令牌"`
	TokenType    string    `gorm:"size:50;comment:令牌类型"`
	RefreshToken string    `gorm:"type:text;comment:刷新令牌"`
	Expiry       time.Time `gorm:"index;comment:过期时间"`
	TokenJSON    string    `gorm:"type:jsonb;not null;comment:完整令牌 JSON"`
	CreatedAt    time.Time `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime;comment:更新时间"`
}

// TableName 指定表名
func (OAuth2Token) TableName() string {
	return "oauth2_tokens"
}

// DatabaseTokenStore 基于 GORM 的 Token 存储实现
type DatabaseTokenStore struct {
	db       *database.DB
	provider string
}

// NewDatabaseTokenStore 创建数据库 Token 存储
func NewDatabaseTokenStore(db *database.DB, provider string) (*DatabaseTokenStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}

	store := &DatabaseTokenStore{
		db:       db,
		provider: provider,
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(&OAuth2Token{}); err != nil {
		return nil, fmt.Errorf("auto migrate oauth2_tokens table: %w", err)
	}

	return store, nil
}

// SaveToken 保存 Token 到数据库
func (s *DatabaseTokenStore) SaveToken(ctx context.Context, token *oauth2.Token) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}

	// 序列化完整 Token
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}

	model := &OAuth2Token{
		Provider:     s.provider,
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		TokenJSON:    string(tokenJSON),
	}

	// Upsert: 存在则更新，不存在则插入
	result := s.db.WithContext(ctx).
		Where("provider = ?", s.provider).
		Assign(map[string]interface{}{
			"access_token":  model.AccessToken,
			"token_type":    model.TokenType,
			"refresh_token": model.RefreshToken,
			"expiry":        model.Expiry,
			"token_json":    model.TokenJSON,
			"updated_at":    time.Now(),
		}).
		FirstOrCreate(&model)

	if result.Error != nil {
		return fmt.Errorf("save token: %w", result.Error)
	}

	return nil
}

// LoadToken 从数据库加载 Token
func (s *DatabaseTokenStore) LoadToken(ctx context.Context) (*oauth2.Token, error) {
	var model OAuth2Token

	result := s.db.WithContext(ctx).
		Where("provider = ?", s.provider).
		First(&model)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("token not found for provider: %s", s.provider)
		}
		return nil, fmt.Errorf("load token: %w", result.Error)
	}

	// 反序列化 Token
	var token oauth2.Token
	if err := json.Unmarshal([]byte(model.TokenJSON), &token); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}

	return &token, nil
}

// DeleteToken 从数据库删除 Token
func (s *DatabaseTokenStore) DeleteToken(ctx context.Context) error {
	result := s.db.WithContext(ctx).
		Where("provider = ?", s.provider).
		Delete(&OAuth2Token{})

	if result.Error != nil {
		return fmt.Errorf("delete token: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("token not found for provider: %s", s.provider)
	}

	return nil
}
