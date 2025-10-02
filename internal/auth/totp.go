package auth

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
)

const (
	// TOTPIssuer TOTP 发行者名称
	TOTPIssuer = "AI Writer Backend"
	// TOTPPeriod TOTP 有效期（秒）
	TOTPPeriod = 30
	// TOTPDigits TOTP 验证码位数
	TOTPDigits = 6
)

// TOTPManager TOTP 管理器
type TOTPManager struct {
	issuer string
}

// NewTOTPManager 创建 TOTP 管理器
func NewTOTPManager(issuer string) *TOTPManager {
	if issuer == "" {
		issuer = TOTPIssuer
	}
	return &TOTPManager{issuer: issuer}
}

// GenerateSecret 生成 TOTP 密钥
// 返回：base32 编码的密钥、OTP URL（用于生成二维码）、错误
func (m *TOTPManager) GenerateSecret(accountName string) (secret string, otpURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      m.issuer,
		AccountName: accountName,
		Period:      TOTPPeriod,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	return key.Secret(), key.URL(), nil
}

// GenerateSecretManual 手动生成 TOTP 密钥（不使用 totp.Generate）
func (m *TOTPManager) GenerateSecretManual() (string, error) {
	secret := make([]byte, 20) // 160 bits
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("failed to generate random secret: %w", err)
	}

	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// BuildOTPURL 构建 OTP URL（用于生成二维码）
func (m *TOTPManager) BuildOTPURL(accountName, secret string) string {
	v := url.Values{}
	v.Set("secret", secret)
	v.Set("issuer", m.issuer)
	v.Set("algorithm", "SHA1")
	v.Set("digits", "6")
	v.Set("period", "30")

	return fmt.Sprintf("otpauth://totp/%s:%s?%s",
		url.PathEscape(m.issuer),
		url.PathEscape(accountName),
		v.Encode(),
	)
}

// GenerateQRCode 生成二维码图片（PNG 格式）
// size: 二维码尺寸（像素），推荐 256
// 返回：PNG 图片字节数组
func (m *TOTPManager) GenerateQRCode(otpURL string, size int) ([]byte, error) {
	if size <= 0 {
		size = 256
	}

	// 使用 Medium 错误纠正级别（推荐）
	return qrcode.Encode(otpURL, qrcode.Medium, size)
}

// GenerateCode 生成当前时间的 TOTP 验证码（用于测试）
func (m *TOTPManager) GenerateCode(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}

// ValidateCode 验证 TOTP 验证码
func (m *TOTPManager) ValidateCode(secret, code string) bool {
	// 移除可能的空格
	code = strings.ReplaceAll(code, " ", "")

	// 验证当前时间的验证码（允许前后各一个时间窗口，即±30秒）
	return totp.Validate(code, secret)
}

// ValidateCodeWithWindow 验证 TOTP 验证码（自定义时间窗口）
// window: 允许的时间窗口数量（例如 window=1 表示允许前后各 30 秒）
func (m *TOTPManager) ValidateCodeWithWindow(secret, code string, window int) bool {
	code = strings.ReplaceAll(code, " ", "")

	t := time.Now()
	for i := -window; i <= window; i++ {
		testTime := t.Add(time.Duration(i) * time.Second * TOTPPeriod)
		validCode, err := totp.GenerateCode(secret, testTime)
		if err != nil {
			continue
		}
		if code == validCode {
			return true
		}
	}
	return false
}
