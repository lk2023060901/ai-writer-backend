package database

import (
	"context"
	"testing"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"gorm.io/gorm"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "missing host",
			config: &Config{
				Host:     "",
				Port:     5432,
				User:     "user",
				DBName:   "test",
				SSLMode:  "disable",
				LogLevel: "warn",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: &Config{
				Host:     "localhost",
				Port:     0,
				User:     "user",
				DBName:   "test",
				SSLMode:  "disable",
				LogLevel: "warn",
			},
			wantErr: true,
		},
		{
			name: "invalid SSL mode",
			config: &Config{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				DBName:   "test",
				SSLMode:  "invalid",
				LogLevel: "warn",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: &Config{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				DBName:   "test",
				SSLMode:  "disable",
				LogLevel: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid connection pool",
			config: &Config{
				Host:         "localhost",
				Port:         5432,
				User:         "user",
				DBName:       "test",
				SSLMode:      "disable",
				LogLevel:     "warn",
				MaxIdleConns: 100,
				MaxOpenConns: 10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPaginate(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		wantPage     int
		wantPageSize int
	}{
		{
			name:         "valid pagination",
			page:         2,
			pageSize:     10,
			wantPage:     2,
			wantPageSize: 10,
		},
		{
			name:         "page less than 1",
			page:         0,
			pageSize:     10,
			wantPage:     1,
			wantPageSize: 10,
		},
		{
			name:         "page size less than 1",
			page:         1,
			pageSize:     0,
			wantPage:     1,
			wantPageSize: 10,
		},
		{
			name:         "page size exceeds max",
			page:         1,
			pageSize:     200,
			wantPage:     1,
			wantPageSize: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Full testing would require a real database connection
			// This is a basic structure test
			scope := Paginate(tt.page, tt.pageSize)
			if scope == nil {
				t.Error("Paginate() returned nil")
			}
		})
	}
}

func TestOrderBy(t *testing.T) {
	tests := []struct {
		name  string
		field string
		desc  bool
	}{
		{
			name:  "ascending order",
			field: "created_at",
			desc:  false,
		},
		{
			name:  "descending order",
			field: "created_at",
			desc:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := OrderBy(tt.field, tt.desc)
			if scope == nil {
				t.Error("OrderBy() returned nil")
			}
		})
	}
}

func TestWhereIf(t *testing.T) {
	tests := []struct {
		name      string
		condition bool
		query     string
		args      []interface{}
	}{
		{
			name:      "condition true",
			condition: true,
			query:     "status = ?",
			args:      []interface{}{"active"},
		},
		{
			name:      "condition false",
			condition: false,
			query:     "status = ?",
			args:      []interface{}{"active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := WhereIf(tt.condition, tt.query, tt.args...)
			if scope == nil {
				t.Error("WhereIf() returned nil")
			}
		})
	}
}

func TestIsRecordNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		// Note: Testing actual GORM errors would require database connection
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRecordNotFoundError(tt.err); got != tt.want {
				t.Errorf("IsRecordNotFoundError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueryBuilder(t *testing.T) {
	// Note: This is a structure test only
	// Full testing would require actual database connection
	t.Run("create query builder", func(t *testing.T) {
		qb := &QueryBuilder{
			scopes: make([]func(*gorm.DB) *gorm.DB, 0),
		}

		qb = qb.Where("status = ?", "active").
			Order("created_at DESC").
			Limit(10).
			Offset(0)

		if qb == nil {
			t.Error("QueryBuilder is nil")
		}

		if len(qb.scopes) != 4 {
			t.Errorf("Expected 4 scopes, got %d", len(qb.scopes))
		}
	})
}

func TestPageResult(t *testing.T) {
	result := &PageResult{
		Data:       []string{"a", "b", "c"},
		Total:      100,
		Page:       1,
		PageSize:   10,
		TotalPages: 10,
	}

	if result.Total != 100 {
		t.Errorf("PageResult.Total = %v, want %v", result.Total, 100)
	}
	if result.TotalPages != 10 {
		t.Errorf("PageResult.TotalPages = %v, want %v", result.TotalPages, 10)
	}
}

func TestTransactionManager(t *testing.T) {
	log, err := logger.Development()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	cfg := DefaultConfig()
	db := &DB{
		config: cfg,
		logger: log,
	}

	tm := NewTransactionManager(db)
	if tm == nil {
		t.Error("NewTransactionManager() returned nil")
	}

	if tm.db != db {
		t.Error("TransactionManager.db is not set correctly")
	}
}

func TestContextWithTransaction(t *testing.T) {
	ctx := context.Background()

	// Note: This test doesn't use real transaction, just tests context functionality
	ctx = ContextWithTransaction(ctx, nil)

	_, ok := TransactionFromContext(ctx)
	if !ok {
		t.Error("Failed to retrieve transaction from context")
	}
}

func TestConfigDSN(t *testing.T) {
	cfg := &Config{
		Host:                 "localhost",
		Port:                 5432,
		User:                 "testuser",
		Password:             "testpass",
		DBName:               "testdb",
		SSLMode:              "disable",
		Timezone:             "UTC",
		PreferSimpleProtocol: true,
	}

	dsn := cfg.DSN()
	if dsn == "" {
		t.Error("DSN is empty")
	}

	// Check if DSN contains expected parts
	expectedParts := []string{"host=", "user=", "password=", "dbname=", "sslmode=", "TimeZone="}
	for _, part := range expectedParts {
		if !contains(dsn, part) {
			t.Errorf("DSN missing expected part: %s", part)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Host != "localhost" {
		t.Errorf("DefaultConfig.Host = %v, want localhost", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("DefaultConfig.Port = %v, want 5432", cfg.Port)
	}
	if cfg.MaxIdleConns != 10 {
		t.Errorf("DefaultConfig.MaxIdleConns = %v, want 10", cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns != 100 {
		t.Errorf("DefaultConfig.MaxOpenConns = %v, want 100", cfg.MaxOpenConns)
	}
	if cfg.ConnMaxLifetime != time.Hour {
		t.Errorf("DefaultConfig.ConnMaxLifetime = %v, want %v", cfg.ConnMaxLifetime, time.Hour)
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		// Note: Testing actual retryable errors would require specific error instances
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableError(tt.err); got != tt.want {
				t.Errorf("isRetryableError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
