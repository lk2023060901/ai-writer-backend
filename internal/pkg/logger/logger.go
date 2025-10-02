package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps zap.Logger with additional functionality
type Logger struct {
	*zap.Logger
	config *Config
}

// New creates a new logger instance with the given configuration
func New(cfg *Config) (*Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid logger configuration: %w", err)
	}

	// Parse log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("failed to parse log level: %w", err)
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Choose encoder based on format
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Create writers based on output configuration
	var writers []zapcore.WriteSyncer

	switch cfg.Output {
	case "console":
		writers = append(writers, zapcore.AddSync(os.Stdout))
	case "file":
		fileWriter := getFileWriter(&cfg.File)
		writers = append(writers, zapcore.AddSync(fileWriter))
	case "both":
		writers = append(writers, zapcore.AddSync(os.Stdout))
		fileWriter := getFileWriter(&cfg.File)
		writers = append(writers, zapcore.AddSync(fileWriter))
	}

	// Create core
	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writers...),
		level,
	)

	// Build logger options
	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1), // Skip one level to show the actual caller
	}

	if cfg.EnableStacktrace {
		opts = append(opts, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	// Create zap logger
	zapLogger := zap.New(core, opts...)

	return &Logger{
		Logger: zapLogger,
		config: cfg,
	}, nil
}

// getFileWriter creates a lumberjack file writer with rotation
func getFileWriter(cfg *FileConfig) io.Writer {
	// Ensure log directory exists
	dir := filepath.Dir(cfg.Filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log directory: %v\n", err)
	}

	return &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
		MaxBackups: cfg.MaxBackups,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}
}

// With creates a child logger with additional fields
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{
		Logger: l.Logger.With(fields...),
		config: l.config,
	}
}

// Named creates a named logger
func (l *Logger) Named(name string) *Logger {
	return &Logger{
		Logger: l.Logger.Named(name),
		config: l.config,
	}
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// Config returns the logger configuration
func (l *Logger) Config() *Config {
	return l.config
}

// Global logger instance
var globalLogger *Logger

// InitGlobal initializes the global logger
func InitGlobal(cfg *Config) error {
	logger, err := New(cfg)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// L returns the global logger instance
func L() *Logger {
	if globalLogger == nil {
		// Create default logger if not initialized
		logger, _ := New(DefaultConfig())
		globalLogger = logger
	}
	return globalLogger
}

// Convenience methods for global logger
func Debug(msg string, fields ...zap.Field) {
	L().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	L().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	L().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	L().Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	L().Fatal(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	L().Panic(msg, fields...)
}

func Sync() error {
	return L().Sync()
}
