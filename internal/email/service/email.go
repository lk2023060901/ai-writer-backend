package service

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/email/types"
	oauth2pkg "github.com/lk2023060901/ai-writer-backend/internal/pkg/oauth2"
	"github.com/wneessen/go-mail"
)

// EmailService 生产级邮件服务
type EmailService struct {
	config        *types.EmailConfig
	tokenProvider oauth2pkg.TokenProvider
}

// NewEmailService 创建邮件服务
func NewEmailService(cfg *types.EmailConfig, tokenProvider oauth2pkg.TokenProvider) (*EmailService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("email config is required")
	}

	// OAuth2 模式下必须提供 TokenProvider
	if cfg.OAuth2Enabled && tokenProvider == nil {
		return nil, fmt.Errorf("token provider is required when oauth2 is enabled")
	}

	// 设置默认值
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryInterval == 0 {
		cfg.RetryInterval = 2 * time.Second
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}
	if cfg.SendTimeout == 0 {
		cfg.SendTimeout = 30 * time.Second
	}

	return &EmailService{
		config:        cfg,
		tokenProvider: tokenProvider,
	}, nil
}

// SendEmail 发送邮件
func (s *EmailService) SendEmail(ctx context.Context, email *types.Email) (*types.EmailStatus, error) {
	if email == nil {
		return nil, fmt.Errorf("email is required")
	}

	// 验证邮件
	if err := s.validateEmail(email); err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	// 创建邮件客户端
	client, err := s.createClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create mail client: %w", err)
	}
	defer client.Close()

	// 构建邮件消息
	msg, err := s.buildMessage(email)
	if err != nil {
		return nil, fmt.Errorf("build message: %w", err)
	}

	// 发送邮件（带重试）
	var lastErr error
	for attempt := 1; attempt <= s.config.MaxRetries; attempt++ {
		sendCtx, cancel := context.WithTimeout(ctx, s.config.SendTimeout)
		err := client.DialAndSendWithContext(sendCtx, msg)
		cancel()

		if err == nil {
			// 发送成功
			return &types.EmailStatus{
				MessageID: msg.GetGenHeader("Message-ID")[0],
				SentAt:    time.Now(),
			}, nil
		}

		lastErr = err
		if attempt < s.config.MaxRetries {
			time.Sleep(s.config.RetryInterval)
		}
	}

	return nil, fmt.Errorf("failed to send email after %d attempts: %w", s.config.MaxRetries, lastErr)
}

// createClient 创建邮件客户端
func (s *EmailService) createClient(ctx context.Context) (*mail.Client, error) {
	opts := []mail.Option{
		mail.WithPort(s.config.SMTPPort),
		mail.WithTimeout(s.config.ConnectTimeout),
		mail.WithSMTPAuth(mail.SMTPAuthNoAuth), // 默认无认证
	}

	// OAuth2 认证
	if s.config.OAuth2Enabled {
		// 获取访问令牌
		accessToken, err := s.tokenProvider.GetAccessToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("get access token: %w", err)
		}

		// 使用 XOAUTH2 认证
		opts = append(opts,
			mail.WithSMTPAuth(mail.SMTPAuthXOAUTH2),
			mail.WithUsername(s.config.FromAddr),
			mail.WithPassword(accessToken),
		)
	}

	// 创建客户端
	client, err := mail.NewClient(s.config.SMTPHost, opts...)
	if err != nil {
		return nil, fmt.Errorf("create mail client: %w", err)
	}

	return client, nil
}

// buildMessage 构建邮件消息
func (s *EmailService) buildMessage(email *types.Email) (*mail.Msg, error) {
	msg := mail.NewMsg()

	// 设置发件人
	if err := msg.From(s.formatAddress(s.config.FromAddr, s.config.FromName)); err != nil {
		return nil, fmt.Errorf("set from: %w", err)
	}

	// 设置收件人
	if err := msg.To(email.To...); err != nil {
		return nil, fmt.Errorf("set to: %w", err)
	}

	// 设置抄送
	if len(email.Cc) > 0 {
		if err := msg.Cc(email.Cc...); err != nil {
			return nil, fmt.Errorf("set cc: %w", err)
		}
	}

	// 设置密送
	if len(email.Bcc) > 0 {
		if err := msg.Bcc(email.Bcc...); err != nil {
			return nil, fmt.Errorf("set bcc: %w", err)
		}
	}

	// 设置主题
	msg.Subject(email.Subject)

	// 设置正文
	if email.IsHTML {
		msg.SetBodyString(mail.TypeTextHTML, email.Body)
	} else {
		msg.SetBodyString(mail.TypeTextPlain, email.Body)
	}

	// 添加附件
	for _, att := range email.Attachments {
		msg.AttachReader(att.Filename, &bytesReader{data: att.Content},
			mail.WithFileContentType(mail.ContentType(att.ContentType)))
	}

	// 添加自定义邮件头
	for key, value := range email.Headers {
		msg.SetGenHeader(mail.Header(key), value)
	}

	// 设置标准邮件头
	msg.SetGenHeader(mail.HeaderXMailer, "AI-Writer-Backend")
	msg.SetDate()
	msg.SetMessageID()

	return msg, nil
}

// validateEmail 验证邮件
func (s *EmailService) validateEmail(email *types.Email) error {
	if len(email.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	if email.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if email.Body == "" {
		return fmt.Errorf("body is required")
	}
	return nil
}

// formatAddress 格式化邮件地址
func (s *EmailService) formatAddress(addr, name string) string {
	if name == "" {
		return addr
	}
	return fmt.Sprintf("%s <%s>", name, addr)
}

// bytesReader 实现 io.Reader 接口用于附件
type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	if r.pos >= len(r.data) {
		return n, fmt.Errorf("EOF")
	}
	return n, nil
}

// IsAuthorized 检查 OAuth2 是否已授权
func (s *EmailService) IsAuthorized() bool {
	if !s.config.OAuth2Enabled || s.tokenProvider == nil {
		return false
	}
	return s.tokenProvider.IsAuthorized()
}

// GetAuthURL 获取 OAuth2 授权 URL
func (s *EmailService) GetAuthURL(state string) (string, error) {
	if !s.config.OAuth2Enabled || s.tokenProvider == nil {
		return "", fmt.Errorf("oauth2 is not enabled")
	}
	return s.tokenProvider.GetAuthURL(state), nil
}

// Authorize 完成 OAuth2 授权
func (s *EmailService) Authorize(ctx context.Context, code string) error {
	if !s.config.OAuth2Enabled || s.tokenProvider == nil {
		return fmt.Errorf("oauth2 is not enabled")
	}
	return s.tokenProvider.ExchangeCode(ctx, code)
}

// RevokeAuthorization 撤销 OAuth2 授权
func (s *EmailService) RevokeAuthorization(ctx context.Context) error {
	if !s.config.OAuth2Enabled || s.tokenProvider == nil {
		return fmt.Errorf("oauth2 is not enabled")
	}
	return s.tokenProvider.RevokeToken(ctx)
}
