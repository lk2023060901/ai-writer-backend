package logger

import (
	"context"
	"os"
	"testing"

	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
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
			name: "console output",
			config: &Config{
				Level:  "info",
				Format: "console",
				Output: "console",
			},
			wantErr: false,
		},
		{
			name: "file output",
			config: &Config{
				Level:  "debug",
				Format: "json",
				Output: "file",
				File: FileConfig{
					Filename:   "/tmp/test.log",
					MaxSize:    10,
					MaxAge:     7,
					MaxBackups: 3,
					Compress:   true,
				},
			},
			wantErr: false,
		},
		{
			name: "both output",
			config: &Config{
				Level:  "warn",
				Format: "json",
				Output: "both",
				File: FileConfig{
					Filename:   "/tmp/test2.log",
					MaxSize:    10,
					MaxAge:     7,
					MaxBackups: 3,
					Compress:   false,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			config: &Config{
				Level:  "invalid",
				Format: "json",
				Output: "console",
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			config: &Config{
				Level:  "info",
				Format: "invalid",
				Output: "console",
			},
			wantErr: true,
		},
		{
			name: "invalid output",
			config: &Config{
				Level:  "info",
				Format: "json",
				Output: "invalid",
			},
			wantErr: true,
		},
		{
			name: "file output without filename",
			config: &Config{
				Level:  "info",
				Format: "json",
				Output: "file",
				File: FileConfig{
					Filename: "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger == nil {
				t.Error("New() returned nil logger")
			}
			if logger != nil {
				logger.Sync()
			}
		})
	}

	// Cleanup test files
	os.Remove("/tmp/test.log")
	os.Remove("/tmp/test2.log")
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid level",
			config: &Config{
				Level:  "invalid",
				Format: "json",
				Output: "console",
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			config: &Config{
				Level:  "info",
				Format: "invalid",
				Output: "console",
			},
			wantErr: true,
		},
		{
			name: "invalid output",
			config: &Config{
				Level:  "info",
				Format: "json",
				Output: "invalid",
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

func TestLogger_With(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	childLogger := logger.With(zap.String("key", "value"))
	if childLogger == nil {
		t.Error("With() returned nil logger")
	}

	// Test that child logger works
	childLogger.Info("test message")
}

func TestLogger_Named(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	namedLogger := logger.Named("test")
	if namedLogger == nil {
		t.Error("Named() returned nil logger")
	}

	// Test that named logger works
	namedLogger.Info("test message")
}

func TestContext(t *testing.T) {
	logger, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Test context without any values
	ctx := context.Background()
	ctxLogger := logger.WithContext(ctx)
	if ctxLogger == nil {
		t.Error("WithContext() returned nil logger")
	}

	// Test context with request ID
	ctx = WithRequestID(ctx, "test-request-id")
	ctxLogger = logger.WithContext(ctx)
	if ctxLogger == nil {
		t.Error("WithContext() returned nil logger")
	}

	// Test FromContext
	fromCtxLogger := FromContext(ctx)
	if fromCtxLogger == nil {
		t.Error("FromContext() returned nil logger")
	}

	// Test ToContext
	ctx = ToContext(ctx, logger)
	if ctx == nil {
		t.Error("ToContext() returned nil context")
	}

	// Test GetRequestID
	requestID := GetRequestID(ctx)
	if requestID != "test-request-id" {
		t.Errorf("GetRequestID() = %v, want %v", requestID, "test-request-id")
	}

	// Test GetTraceID
	ctx = WithTraceID(ctx, "test-trace-id")
	traceID := GetTraceID(ctx)
	if traceID != "test-trace-id" {
		t.Errorf("GetTraceID() = %v, want %v", traceID, "test-trace-id")
	}

	// Test GetUserID
	ctx = WithUserID(ctx, "test-user-id")
	userID := GetUserID(ctx)
	if userID != "test-user-id" {
		t.Errorf("GetUserID() = %v, want %v", userID, "test-user-id")
	}
}

func TestGlobalLogger(t *testing.T) {
	// Test default global logger
	logger := L()
	if logger == nil {
		t.Error("L() returned nil logger")
	}

	// Test InitGlobal
	config := DefaultConfig()
	err := InitGlobal(config)
	if err != nil {
		t.Errorf("InitGlobal() error = %v", err)
	}

	// Test global logger functions
	Debug("debug message", zap.String("key", "value"))
	Info("info message", zap.String("key", "value"))
	Warn("warn message", zap.String("key", "value"))
	Error("error message", zap.String("key", "value"))

	// Test sync (ignore stdout/stderr sync errors in tests)
	_ = Sync()
}

func TestNewWithOptions(t *testing.T) {
	logger, err := NewWithOptions(
		WithLevel("debug"),
		WithFormat("console"),
		WithOutput("console"),
		WithCaller(true),
		WithStacktrace(true),
	)
	if err != nil {
		t.Errorf("NewWithOptions() error = %v", err)
	}
	if logger == nil {
		t.Error("NewWithOptions() returned nil logger")
	}
	if logger != nil {
		logger.Sync()
	}
}

func TestDevelopment(t *testing.T) {
	logger, err := Development()
	if err != nil {
		t.Errorf("Development() error = %v", err)
	}
	if logger == nil {
		t.Error("Development() returned nil logger")
	}
	if logger != nil {
		logger.Info("test development logger")
		logger.Sync()
	}
}

func TestProduction(t *testing.T) {
	logger, err := Production("/tmp/prod-test.log")
	if err != nil {
		t.Errorf("Production() error = %v", err)
	}
	if logger == nil {
		t.Error("Production() returned nil logger")
	}
	if logger != nil {
		logger.Info("test production logger")
		logger.Sync()
	}

	// Cleanup
	os.Remove("/tmp/prod-test.log")
}

func TestContextLogging(t *testing.T) {
	err := InitGlobal(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to initialize global logger: %v", err)
	}

	ctx := context.Background()
	ctx = WithRequestID(ctx, "test-req")
	ctx = WithTraceID(ctx, "test-trace")
	ctx = WithUserID(ctx, "test-user")

	// Test context-aware logging functions
	DebugContext(ctx, "debug message")
	InfoContext(ctx, "info message")
	WarnContext(ctx, "warn message")
	ErrorContext(ctx, "error message")

	Sync()
}
