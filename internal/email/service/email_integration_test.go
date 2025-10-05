// +build integration

package service_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/email/service"
	"github.com/lk2023060901/ai-writer-backend/internal/email/types"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
	oauth2pkg "github.com/lk2023060901/ai-writer-backend/internal/pkg/oauth2"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 集成测试说明:
// 1. 运行前确保配置文件 config.yaml 存在且包含正确的 OAuth2 配置
// 2. 首次运行需要完成 OAuth2 授权流程
// 3. 运行命令: go test -tags=integration -v ./internal/email/service/

var (
	testConfig = &types.EmailConfig{
		SMTPHost:       "smtp.gmail.com",
		SMTPPort:       587,
		FromAddr:       "lk2023060901@gmail.com",
		FromName:       "AI Writer Test",
		OAuth2Enabled:  true,
		MaxRetries:     3,
		RetryInterval:  2 * time.Second,
		ConnectTimeout: 10 * time.Second,
		SendTimeout:    30 * time.Second,
	}

	testOAuth2Config = &oauth2pkg.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  "http://localhost:8080/api/v1/email/oauth2/callback",
		Scopes:       []string{"https://mail.google.com/"},
	}

	testRecipient = "565434471@qq.com"
)

func setupTestEnvironment(t *testing.T) (*service.EmailService, oauth2pkg.TokenProvider) {
	// 初始化日志
	log, err := logger.New(&logger.Config{
		Level:  "debug",
		Format: "console",
		Output: "stdout",
	})
	require.NoError(t, err)

	// 初始化数据库
	dbConfig := &database.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "ai_writer"),
		SSLMode:  "disable",
	}

	db, err := database.New(dbConfig, log)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
	})

	// 创建 Token Store
	tokenStore, err := oauth2pkg.NewDatabaseTokenStore(db, "gmail")
	require.NoError(t, err)

	// 创建 Token Provider
	tokenProvider, err := oauth2pkg.NewGoogleTokenProvider(testOAuth2Config, tokenStore)
	require.NoError(t, err)

	// 创建邮件服务
	emailService, err := service.NewEmailService(testConfig, tokenProvider)
	require.NoError(t, err)

	return emailService, tokenProvider
}

func TestIntegration_OAuth2Flow(t *testing.T) {
	emailService, tokenProvider := setupTestEnvironment(t)

	t.Run("检查授权状态", func(t *testing.T) {
		authorized := emailService.IsAuthorized()
		t.Logf("当前授权状态: %v", authorized)

		if !authorized {
			t.Log("未授权，需要完成 OAuth2 授权流程")
			t.Log("请执行以下步骤:")
			t.Log("1. 启动服务器")
			t.Log("2. 访问 /api/v1/email/oauth2/auth-url 获取授权链接")
			t.Log("3. 在浏览器中打开链接并完成授权")
			t.Log("4. 授权成功后重新运行此测试")
			t.SkipNow()
		}
	})

	t.Run("刷新 Token", func(t *testing.T) {
		if !emailService.IsAuthorized() {
			t.Skip("未授权，跳过")
		}

		ctx := context.Background()
		err := tokenProvider.RefreshToken(ctx)
		assert.NoError(t, err)

		// 验证可以获取新的 Access Token
		accessToken, err := tokenProvider.GetAccessToken(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		t.Logf("成功刷新 Token，新 Access Token 长度: %d", len(accessToken))
	})
}

func TestIntegration_SendEmail(t *testing.T) {
	emailService, _ := setupTestEnvironment(t)

	if !emailService.IsAuthorized() {
		t.Skip("未授权，跳过邮件发送测试")
	}

	t.Run("发送纯文本邮件", func(t *testing.T) {
		email := &types.Email{
			To:      []string{testRecipient},
			Subject: "AI Writer - 集成测试邮件（纯文本）",
			Body:    "这是一封来自 AI Writer Backend 的测试邮件。\n\n测试时间: " + time.Now().Format(time.RFC3339),
			IsHTML:  false,
		}

		ctx := context.Background()
		status, err := emailService.SendEmail(ctx, email)
		require.NoError(t, err)
		require.NotNil(t, status)

		assert.NotEmpty(t, status.MessageID)
		assert.False(t, status.SentAt.IsZero())
		t.Logf("邮件发送成功 - MessageID: %s, 发送时间: %s",
			status.MessageID, status.SentAt.Format(time.RFC3339))
	})

	t.Run("发送 HTML 邮件", func(t *testing.T) {
		htmlBody := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>AI Writer 测试邮件</title>
</head>
<body style="font-family: Arial, sans-serif; padding: 20px;">
    <h1 style="color: #2c3e50;">AI Writer Backend</h1>
    <p>这是一封来自 AI Writer Backend 的 <strong>HTML 测试邮件</strong>。</p>
    <ul>
        <li>测试时间: ` + time.Now().Format(time.RFC3339) + `</li>
        <li>发送方式: OAuth2 XOAUTH2 认证</li>
        <li>SMTP 服务器: smtp.gmail.com:587</li>
    </ul>
    <hr>
    <p style="color: #7f8c8d; font-size: 12px;">
        此邮件由自动化测试发送，请勿回复。
    </p>
</body>
</html>
`

		email := &types.Email{
			To:      []string{testRecipient},
			Subject: "AI Writer - 集成测试邮件（HTML）",
			Body:    htmlBody,
			IsHTML:  true,
		}

		ctx := context.Background()
		status, err := emailService.SendEmail(ctx, email)
		require.NoError(t, err)
		require.NotNil(t, status)

		assert.NotEmpty(t, status.MessageID)
		t.Logf("HTML 邮件发送成功 - MessageID: %s", status.MessageID)
	})

	t.Run("发送多收件人邮件", func(t *testing.T) {
		email := &types.Email{
			To: []string{
				testRecipient,
				"lk2023060901@gmail.com", // 发件人自己
			},
			Cc:      []string{},
			Subject: "AI Writer - 多收件人测试",
			Body:    "此邮件发送给多个收件人。\n\n测试时间: " + time.Now().Format(time.RFC3339),
			IsHTML:  false,
		}

		ctx := context.Background()
		status, err := emailService.SendEmail(ctx, email)
		require.NoError(t, err)
		assert.NotNil(t, status)
		t.Logf("多收件人邮件发送成功")
	})

	t.Run("发送带自定义邮件头的邮件", func(t *testing.T) {
		email := &types.Email{
			To:      []string{testRecipient},
			Subject: "AI Writer - 自定义邮件头测试",
			Body:    "此邮件包含自定义邮件头。",
			IsHTML:  false,
			Headers: map[string]string{
				"X-Test-ID":       "integration-test-" + time.Now().Format("20060102150405"),
				"X-Test-Category": "email-service",
			},
		}

		ctx := context.Background()
		status, err := emailService.SendEmail(ctx, email)
		require.NoError(t, err)
		assert.NotNil(t, status)
		t.Logf("自定义邮件头邮件发送成功")
	})
}

func TestIntegration_SendEmailWithRetry(t *testing.T) {
	emailService, _ := setupTestEnvironment(t)

	if !emailService.IsAuthorized() {
		t.Skip("未授权，跳过重试测试")
	}

	t.Run("验证重试机制", func(t *testing.T) {
		email := &types.Email{
			To:      []string{testRecipient},
			Subject: "AI Writer - 重试机制测试",
			Body:    "此邮件用于测试重试机制。\n\n发送时间: " + time.Now().Format(time.RFC3339),
			IsHTML:  false,
		}

		ctx := context.Background()
		startTime := time.Now()
		status, err := emailService.SendEmail(ctx, email)
		duration := time.Since(startTime)

		require.NoError(t, err)
		assert.NotNil(t, status)
		t.Logf("邮件发送耗时: %v", duration)
	})
}

func TestIntegration_ConcurrentSend(t *testing.T) {
	emailService, _ := setupTestEnvironment(t)

	if !emailService.IsAuthorized() {
		t.Skip("未授权，跳过并发测试")
	}

	t.Run("并发发送邮件", func(t *testing.T) {
		concurrency := 3
		results := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(index int) {
				email := &types.Email{
					To:      []string{testRecipient},
					Subject: "AI Writer - 并发测试 #" + string(rune(index+1)),
					Body:    "并发测试邮件 #" + string(rune(index+1)) + "\n\n时间: " + time.Now().Format(time.RFC3339),
					IsHTML:  false,
				}

				ctx := context.Background()
				_, err := emailService.SendEmail(ctx, email)
				results <- err
			}(i)
		}

		// 等待所有邮件发送完成
		for i := 0; i < concurrency; i++ {
			err := <-results
			assert.NoError(t, err)
		}

		t.Logf("成功并发发送 %d 封邮件", concurrency)
	})
}

// getEnv 获取环境变量或默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
