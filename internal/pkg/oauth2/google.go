package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleTokenProvider 生产级 Google OAuth2 令牌提供者
// 特性：
// - 自动刷新过期 Token
// - 线程安全
// - 支持持久化存储
// - 错误重试机制
type GoogleTokenProvider struct {
	config       *Config
	oauth2Config *oauth2.Config
	tokenStore   TokenStore

	// 当前有效 Token 缓存
	currentToken *oauth2.Token
	mu           sync.RWMutex

	// 刷新控制
	refreshing   bool
	refreshCond  *sync.Cond
}

// NewGoogleTokenProvider 创建生产级 Google OAuth2 Token Provider
func NewGoogleTokenProvider(cfg *Config, store TokenStore) (*GoogleTokenProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("oauth2 config is required")
	}
	if store == nil {
		return nil, fmt.Errorf("token store is required")
	}

	// 验证必需配置
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("client_id and client_secret are required")
	}

	// 默认使用 Google OAuth2 端点
	endpoint := google.Endpoint
	if cfg.TokenURL != "" {
		endpoint.TokenURL = cfg.TokenURL
	}
	if cfg.AuthURL != "" {
		endpoint.AuthURL = cfg.AuthURL
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     endpoint,
		Scopes:       cfg.Scopes,
		RedirectURL:  cfg.RedirectURL,
	}

	provider := &GoogleTokenProvider{
		config:       cfg,
		oauth2Config: oauth2Config,
		tokenStore:   store,
	}
	provider.refreshCond = sync.NewCond(&provider.mu)

	// 从存储加载已有 Token
	if err := provider.loadTokenFromStore(context.Background()); err != nil {
		// 首次运行可能没有 Token，不返回错误
		// 仅记录日志
	}

	return provider, nil
}

// GetAccessToken 获取有效的访问令牌（产品级实现）
// 特性：
// - 自动检查过期并刷新
// - 并发请求时仅刷新一次
// - 失败重试机制
func (p *GoogleTokenProvider) GetAccessToken(ctx context.Context) (string, error) {
	p.mu.RLock()

	// 检查当前 Token 是否有效
	if p.currentToken != nil && p.currentToken.Valid() {
		token := p.currentToken.AccessToken
		p.mu.RUnlock()
		return token, nil
	}

	// Token 无效或即将过期，需要刷新
	// 检查是否已有其他 goroutine 在刷新
	if p.refreshing {
		// 等待刷新完成
		p.refreshCond.Wait()
		if p.currentToken != nil && p.currentToken.Valid() {
			token := p.currentToken.AccessToken
			p.mu.RUnlock()
			return token, nil
		}
		p.mu.RUnlock()
		return "", fmt.Errorf("token refresh failed")
	}

	p.mu.RUnlock()

	// 执行刷新
	return p.refreshTokenWithRetry(ctx, 3)
}

// refreshTokenWithRetry 带重试的 Token 刷新
func (p *GoogleTokenProvider) refreshTokenWithRetry(ctx context.Context, maxRetries int) (string, error) {
	p.mu.Lock()

	// 双重检查：可能其他 goroutine 已完成刷新
	if p.currentToken != nil && p.currentToken.Valid() {
		token := p.currentToken.AccessToken
		p.mu.Unlock()
		return token, nil
	}

	// 标记正在刷新
	p.refreshing = true
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.refreshing = false
		p.refreshCond.Broadcast() // 唤醒所有等待的 goroutine
		p.mu.Unlock()
	}()

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := p.RefreshToken(ctx); err != nil {
			lastErr = err
			if attempt < maxRetries {
				// 指数退避
				backoff := time.Duration(attempt) * time.Second
				time.Sleep(backoff)
				continue
			}
		} else {
			p.mu.RLock()
			token := p.currentToken.AccessToken
			p.mu.RUnlock()
			return token, nil
		}
	}

	return "", fmt.Errorf("failed to refresh token after %d attempts: %w", maxRetries, lastErr)
}

// RefreshToken 刷新 OAuth2 Token
func (p *GoogleTokenProvider) RefreshToken(ctx context.Context) error {
	p.mu.RLock()
	currentToken := p.currentToken
	p.mu.RUnlock()

	if currentToken == nil {
		return fmt.Errorf("no refresh token available, please authorize first")
	}

	// 使用 OAuth2 库自动刷新
	tokenSource := p.oauth2Config.TokenSource(ctx, currentToken)
	newToken, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("refresh token: %w", err)
	}

	// 更新缓存
	p.mu.Lock()
	p.currentToken = newToken
	p.mu.Unlock()

	// 持久化到存储
	if err := p.tokenStore.SaveToken(ctx, newToken); err != nil {
		// 持久化失败不影响当前使用，仅记录错误
		return fmt.Errorf("save token to store: %w", err)
	}

	return nil
}

// GetAuthURL 生成 OAuth2 授权 URL
func (p *GoogleTokenProvider) GetAuthURL(state string) string {
	opts := []oauth2.AuthCodeOption{}

	// 强制获取 Refresh Token
	opts = append(opts, oauth2.AccessTypeOffline)

	// 每次都提示用户同意（确保获取 Refresh Token）
	opts = append(opts, oauth2.ApprovalForce)

	return p.oauth2Config.AuthCodeURL(state, opts...)
}

// ExchangeCode 交换授权码获取 Token
func (p *GoogleTokenProvider) ExchangeCode(ctx context.Context, code string) error {
	token, err := p.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("exchange code: %w", err)
	}

	// 验证是否获取到 Refresh Token
	if token.RefreshToken == "" {
		return fmt.Errorf("no refresh token received, please revoke app access and re-authorize")
	}

	// 更新缓存
	p.mu.Lock()
	p.currentToken = token
	p.mu.Unlock()

	// 持久化到存储
	if err := p.tokenStore.SaveToken(ctx, token); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	return nil
}

// loadTokenFromStore 从存储加载 Token
func (p *GoogleTokenProvider) loadTokenFromStore(ctx context.Context) error {
	token, err := p.tokenStore.LoadToken(ctx)
	if err != nil {
		return err
	}

	p.mu.Lock()
	p.currentToken = token
	p.mu.Unlock()

	return nil
}

// IsAuthorized 检查是否已授权
func (p *GoogleTokenProvider) IsAuthorized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentToken != nil && p.currentToken.RefreshToken != ""
}

// RevokeToken 撤销 Token
func (p *GoogleTokenProvider) RevokeToken(ctx context.Context) error {
	p.mu.RLock()
	token := p.currentToken
	p.mu.RUnlock()

	if token == nil || token.AccessToken == "" {
		return fmt.Errorf("no token to revoke")
	}

	// 调用 Google Revoke API
	revokeURL := "https://oauth2.googleapis.com/revoke"
	req, err := http.NewRequestWithContext(ctx, "POST", revokeURL,
		strings.NewReader("token="+token.AccessToken))
	if err != nil {
		return fmt.Errorf("create revoke request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("revoke token failed: status %d", resp.StatusCode)
	}

	// 清除本地存储
	p.mu.Lock()
	p.currentToken = nil
	p.mu.Unlock()

	return p.tokenStore.DeleteToken(ctx)
}
