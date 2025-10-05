package types

import "time"

// EmailConfig 邮件服务配置
type EmailConfig struct {
	// SMTP 配置
	SMTPHost string `yaml:"smtp_host" json:"smtp_host"`
	SMTPPort int    `yaml:"smtp_port" json:"smtp_port"`
	FromAddr string `yaml:"from_addr" json:"from_addr"` // 发件人地址
	FromName string `yaml:"from_name" json:"from_name"` // 发件人名称

	// OAuth2 配置
	OAuth2Enabled bool `yaml:"oauth2_enabled" json:"oauth2_enabled"`

	// 重试配置
	MaxRetries    int           `yaml:"max_retries" json:"max_retries"`
	RetryInterval time.Duration `yaml:"retry_interval" json:"retry_interval"`

	// 超时配置
	ConnectTimeout time.Duration `yaml:"connect_timeout" json:"connect_timeout"`
	SendTimeout    time.Duration `yaml:"send_timeout" json:"send_timeout"`
}

// Email 邮件结构
type Email struct {
	To          []string          // 收件人
	Cc          []string          // 抄送
	Bcc         []string          // 密送
	Subject     string            // 主题
	Body        string            // 正文（纯文本或 HTML）
	IsHTML      bool              // 是否为 HTML 正文
	Attachments []Attachment      // 附件
	Headers     map[string]string // 自定义邮件头
}

// Attachment 邮件附件
type Attachment struct {
	Filename    string // 文件名
	ContentType string // MIME 类型
	Content     []byte // 文件内容
}

// EmailStatus 邮件发送状态
type EmailStatus struct {
	MessageID string    // 邮件 ID
	SentAt    time.Time // 发送时间
	Error     error     // 错误信息
}
