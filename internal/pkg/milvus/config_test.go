package milvus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "localhost:19530", cfg.Address)
	assert.Equal(t, "default", cfg.Database)
	assert.Equal(t, 10*time.Second, cfg.DialTimeout)
	assert.Equal(t, 30*time.Second, cfg.RequestTimeout)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, time.Second, cfg.RetryDelay)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Address: "localhost:19530",
			},
			wantErr: false,
		},
		{
			name: "empty address",
			cfg: &Config{
				Address: "",
			},
			wantErr: true,
		},
		{
			name: "negative dial timeout",
			cfg: &Config{
				Address:     "localhost:19530",
				DialTimeout: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative request timeout",
			cfg: &Config{
				Address:        "localhost:19530",
				RequestTimeout: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative max retries",
			cfg: &Config{
				Address:    "localhost:19530",
				MaxRetries: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_SetDefaults(t *testing.T) {
	cfg := &Config{
		Address: "localhost:19530",
	}

	cfg.SetDefaults()

	assert.Equal(t, "default", cfg.Database)
	assert.Equal(t, 10*time.Second, cfg.DialTimeout)
	assert.Equal(t, 30*time.Second, cfg.RequestTimeout)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, time.Second, cfg.RetryDelay)
	assert.Equal(t, 10, cfg.MaxIdleConns)
	assert.Equal(t, 100, cfg.MaxOpenConns)
	assert.Equal(t, 30*time.Minute, cfg.ConnMaxLifetime)
	assert.Equal(t, 30*time.Second, cfg.KeepAlive)
}

func TestConfig_Clone(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Username = "test_user"
	cfg.Password = "test_pass"

	cloned := cfg.Clone()

	assert.Equal(t, cfg.Address, cloned.Address)
	assert.Equal(t, cfg.Username, cloned.Username)
	assert.Equal(t, cfg.Password, cloned.Password)
	assert.Equal(t, cfg.Database, cloned.Database)

	// 修改克隆的配置不应影响原配置
	cloned.Address = "changed:19530"
	assert.NotEqual(t, cfg.Address, cloned.Address)
}

func TestConfig_WithOptions(t *testing.T) {
	cfg := &Config{
		Address: "localhost:19530",
	}

	// 测试设置数据库
	cfg.Database = "test_db"
	assert.Equal(t, "test_db", cfg.Database)

	// 测试设置认证
	cfg.Username = "admin"
	cfg.Password = "secret"
	assert.Equal(t, "admin", cfg.Username)
	assert.Equal(t, "secret", cfg.Password)

	// 测试设置超时
	cfg.DialTimeout = 5 * time.Second
	cfg.RequestTimeout = 60 * time.Second
	assert.Equal(t, 5*time.Second, cfg.DialTimeout)
	assert.Equal(t, 60*time.Second, cfg.RequestTimeout)

	// 测试设置重试
	cfg.MaxRetries = 5
	cfg.RetryDelay = 2 * time.Second
	assert.Equal(t, 5, cfg.MaxRetries)
	assert.Equal(t, 2*time.Second, cfg.RetryDelay)
}

func TestConfig_ConnectionPool(t *testing.T) {
	cfg := &Config{
		Address:         "localhost:19530",
		MaxIdleConns:    20,
		MaxOpenConns:    200,
		ConnMaxLifetime: 60 * time.Minute,
	}

	cfg.SetDefaults()

	assert.Equal(t, 20, cfg.MaxIdleConns)
	assert.Equal(t, 200, cfg.MaxOpenConns)
	assert.Equal(t, 60*time.Minute, cfg.ConnMaxLifetime)
}

func TestConfig_TLS(t *testing.T) {
	cfg := &Config{
		Address:   "localhost:19530",
		EnableTLS: true,
		TLSMode:   "mutual",
	}

	assert.True(t, cfg.EnableTLS)
	assert.Equal(t, "mutual", cfg.TLSMode)
}

func TestConfig_ValidateEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "zero timeout is valid",
			cfg: &Config{
				Address:        "localhost:19530",
				DialTimeout:    0,
				RequestTimeout: 0,
			},
			wantErr: false,
		},
		{
			name: "zero retries is valid",
			cfg: &Config{
				Address:    "localhost:19530",
				MaxRetries: 0,
			},
			wantErr: false,
		},
		{
			name: "empty database will use default",
			cfg: &Config{
				Address:  "localhost:19530",
				Database: "",
			},
			wantErr: false,
		},
		{
			name: "address with port",
			cfg: &Config{
				Address: "milvus.example.com:19530",
			},
			wantErr: false,
		},
		{
			name: "address without port",
			cfg: &Config{
				Address: "milvus.example.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
