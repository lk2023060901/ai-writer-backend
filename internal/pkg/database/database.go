package database

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DB wraps gorm.DB with additional functionality
type DB struct {
	*gorm.DB
	config *Config
	logger *logger.Logger
}

// New creates a new database connection
func New(cfg *Config, log *logger.Logger) (*DB, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid database configuration: %w", err)
	}

	// Create custom GORM logger
	gormLog := newGormLogger(log, cfg)

	// Open database connection
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger:                 gormLog,
		SkipDefaultTransaction: cfg.SkipDefaultTx,
		PrepareStmt:            cfg.PrepareStmt,
		DisableForeignKeyConstraintWhenMigrating: cfg.DisableForeignKey,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("database connected successfully",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.DBName),
	)

	return &DB{
		DB:     db,
		config: cfg,
		logger: log,
	}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}

	db.logger.Info("closing database connection")
	return sqlDB.Close()
}

// HealthCheck checks if the database connection is healthy
func (db *DB) HealthCheck(ctx context.Context) error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	return sqlDB.PingContext(ctx)
}

// Stats returns database connection pool statistics
func (db *DB) Stats() map[string]interface{} {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections":   stats.MaxOpenConnections,
		"open_connections":       stats.OpenConnections,
		"in_use":                 stats.InUse,
		"idle":                   stats.Idle,
		"wait_count":             stats.WaitCount,
		"wait_duration":          stats.WaitDuration.String(),
		"max_idle_closed":        stats.MaxIdleClosed,
		"max_lifetime_closed":    stats.MaxLifetimeClosed,
		"max_idle_time_closed":   stats.MaxIdleTimeClosed,
	}
}

// Config returns the database configuration
func (db *DB) Config() *Config {
	return db.config
}

// Logger returns the logger instance
func (db *DB) Logger() *logger.Logger {
	return db.logger
}

// WithContext returns a new DB instance with the given context
func (db *DB) WithContext(ctx context.Context) *DB {
	return &DB{
		DB:     db.DB.WithContext(ctx),
		config: db.config,
		logger: db.logger,
	}
}

// WithLogger returns a new DB instance with the given logger
func (db *DB) WithLogger(log *logger.Logger) *DB {
	return &DB{
		DB:     db.DB,
		config: db.config,
		logger: log,
	}
}

// AutoMigrate runs auto migration for the given models
func (db *DB) AutoMigrate(models ...interface{}) error {
	if !db.config.AutoMigrate {
		db.logger.Warn("auto migration is disabled in configuration")
		return nil
	}

	db.logger.Info("running auto migration", zap.Int("models", len(models)))

	if err := db.DB.AutoMigrate(models...); err != nil {
		return fmt.Errorf("auto migration failed: %w", err)
	}

	db.logger.Info("auto migration completed successfully")
	return nil
}

// Exec executes raw SQL
func (db *DB) Exec(sql string, values ...interface{}) *gorm.DB {
	return db.DB.Exec(sql, values...)
}

// Raw executes raw SQL query
func (db *DB) Raw(sql string, values ...interface{}) *gorm.DB {
	return db.DB.Raw(sql, values...)
}

// IsRecordNotFoundError checks if the error is a record not found error
func IsRecordNotFoundError(err error) bool {
	return err == gorm.ErrRecordNotFound
}

// IsDuplicateKeyError checks if the error is a duplicate key error
func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL duplicate key error code: 23505
	return err.Error() == "ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)" ||
		err.Error() == "UNIQUE constraint failed" ||
		err.Error() == "Duplicate entry"
}

// GetDB returns the underlying gorm.DB instance
func (db *DB) GetDB() *gorm.DB {
	return db.DB
}

// newGormLogger creates a custom GORM logger that integrates with our logger
func newGormLogger(log *logger.Logger, cfg *Config) gormlogger.Interface {
	var logLevel gormlogger.LogLevel
	switch cfg.LogLevel {
	case "silent":
		logLevel = gormlogger.Silent
	case "error":
		logLevel = gormlogger.Error
	case "warn":
		logLevel = gormlogger.Warn
	case "info":
		logLevel = gormlogger.Info
	default:
		logLevel = gormlogger.Warn
	}

	return &customGormLogger{
		logger:        log,
		logLevel:      logLevel,
		slowThreshold: cfg.SlowThreshold,
	}
}

// customGormLogger implements gorm logger.Interface
type customGormLogger struct {
	logger        *logger.Logger
	logLevel      gormlogger.LogLevel
	slowThreshold time.Duration
}

func (l *customGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

func (l *customGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Info {
		l.logger.WithContext(ctx).Info(fmt.Sprintf(msg, data...))
	}
}

func (l *customGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Warn {
		l.logger.WithContext(ctx).Warn(fmt.Sprintf(msg, data...))
	}
}

func (l *customGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Error {
		l.logger.WithContext(ctx).Error(fmt.Sprintf(msg, data...))
	}
}

func (l *customGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.logLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []zap.Field{
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	}

	switch {
	case err != nil && l.logLevel >= gormlogger.Error:
		fields = append(fields, zap.Error(err))
		l.logger.WithContext(ctx).Error("database query error", fields...)
	case elapsed > l.slowThreshold && l.slowThreshold != 0 && l.logLevel >= gormlogger.Warn:
		fields = append(fields, zap.Duration("threshold", l.slowThreshold))
		l.logger.WithContext(ctx).Warn("slow SQL query", fields...)
	case l.logLevel >= gormlogger.Info:
		l.logger.WithContext(ctx).Info("database query", fields...)
	}
}
