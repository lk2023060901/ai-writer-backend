package oauth2

import (
	"context"

	"golang.org/x/oauth2"
)

// TokenProvider 定义获取访问令牌的接口
type TokenProvider interface {
	// GetAccessToken 获取有效的访问令牌（自动刷新）
	GetAccessToken(ctx context.Context) (string, error)

	// RefreshToken 强制刷新令牌
	RefreshToken(ctx context.Context) error

	// GetAuthURL 生成授权 URL
	GetAuthURL(state string) string

	// ExchangeCode 交换授权码获取 Token
	ExchangeCode(ctx context.Context, code string) error

	// IsAuthorized 检查是否已授权
	IsAuthorized() bool

	// RevokeToken 撤销 Token
	RevokeToken(ctx context.Context) error
}

// TokenStore Token 持久化存储接口
type TokenStore interface {
	// SaveToken 保存 Token
	SaveToken(ctx context.Context, token *oauth2.Token) error

	// LoadToken 加载 Token
	LoadToken(ctx context.Context) (*oauth2.Token, error)

	// DeleteToken 删除 Token
	DeleteToken(ctx context.Context) error
}

// Config OAuth2 配置
type Config struct {
	// Web Application OAuth 配置
	ClientID     string   `yaml:"client_id" json:"client_id"`
	ClientSecret string   `yaml:"client_secret" json:"client_secret"`
	RedirectURL  string   `yaml:"redirect_url" json:"redirect_url"`
	Scopes       []string `yaml:"scopes" json:"scopes"`

	// 可选：自定义端点（默认使用 Google 端点）
	AuthURL  string `yaml:"auth_url" json:"auth_url,omitempty"`
	TokenURL string `yaml:"token_url" json:"token_url,omitempty"`
}
