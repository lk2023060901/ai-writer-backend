package mineru

import (
	"errors"
	"time"
)

// Config MinerU 配置
type Config struct {
	// BaseURL API 基础地址
	BaseURL string `mapstructure:"base_url" yaml:"base_url"`

	// APIKey API 密钥
	APIKey string `mapstructure:"api_key" yaml:"api_key"`

	// Timeout 请求超时时间
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout"`

	// MaxRetries 最大重试次数
	MaxRetries int `mapstructure:"max_retries" yaml:"max_retries"`

	// DefaultLanguage 默认文档语言
	DefaultLanguage string `mapstructure:"default_language" yaml:"default_language"`

	// EnableFormula 默认是否启用公式识别
	EnableFormula bool `mapstructure:"enable_formula" yaml:"enable_formula"`

	// EnableTable 默认是否启用表格识别
	EnableTable bool `mapstructure:"enable_table" yaml:"enable_table"`

	// ModelVersion 模型版本 (pipeline/vlm)
	ModelVersion string `mapstructure:"model_version" yaml:"model_version"`
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return errors.New("mineru: base_url is required")
	}

	if c.APIKey == "" {
		return errors.New("mineru: api_key is required")
	}

	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}

	if c.MaxRetries <= 0 {
		c.MaxRetries = 3
	}

	if c.DefaultLanguage == "" {
		c.DefaultLanguage = "ch"
	}

	if c.ModelVersion == "" {
		c.ModelVersion = "pipeline"
	} else if c.ModelVersion != "pipeline" && c.ModelVersion != "vlm" {
		return errors.New("mineru: model_version must be 'pipeline' or 'vlm'")
	}

	return nil
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		BaseURL:         "https://mineru.net",
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		DefaultLanguage: "ch",
		EnableFormula:   true,
		EnableTable:     true,
		ModelVersion:    "pipeline",
	}
}
