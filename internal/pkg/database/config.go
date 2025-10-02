package database

import (
	"errors"
	"fmt"
	"time"
)

// Config defines the database configuration
type Config struct {
	// Connection settings
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"` // disable, require, verify-ca, verify-full

	// Connection pool settings
	MaxIdleConns    int           `mapstructure:"maxidleconns"`    // Maximum idle connections
	MaxOpenConns    int           `mapstructure:"maxopenconns"`    // Maximum open connections
	ConnMaxLifetime time.Duration `mapstructure:"connmaxlifetime"` // Connection max lifetime
	ConnMaxIdleTime time.Duration `mapstructure:"connmaxidletime"` // Connection max idle time

	// GORM settings
	LogLevel        string `mapstructure:"loglevel"`        // silent, error, warn, info
	SlowThreshold   time.Duration `mapstructure:"slowthreshold"`   // Slow query threshold
	SkipDefaultTx   bool   `mapstructure:"skipdefaulttx"`   // Skip default transaction
	PrepareStmt     bool   `mapstructure:"preparestmt"`     // Prepare statement cache
	DisableForeignKey bool `mapstructure:"disableforeignkey"` // Disable foreign key constraints

	// Additional settings
	Timezone        string `mapstructure:"timezone"`        // Database timezone
	AutoMigrate     bool   `mapstructure:"automigrate"`     // Enable auto migration
	PreferSimpleProtocol bool `mapstructure:"prefersimpleprotocol"` // Prefer simple protocol
}

// DefaultConfig returns the default database configuration
func DefaultConfig() *Config {
	return &Config{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "postgres",
		SSLMode:  "disable",

		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,

		LogLevel:      "warn",
		SlowThreshold: 200 * time.Millisecond,
		SkipDefaultTx: false,
		PrepareStmt:   true,
		DisableForeignKey: false,

		Timezone:             "Asia/Shanghai",
		AutoMigrate:          false,
		PreferSimpleProtocol: false,
	}
}

// Validate validates the database configuration
func (c *Config) Validate() error {
	if c.Host == "" {
		return errors.New("database host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return errors.New("database port must be between 1 and 65535")
	}
	if c.User == "" {
		return errors.New("database user is required")
	}
	if c.DBName == "" {
		return errors.New("database name is required")
	}

	// Validate SSL mode
	validSSLModes := []string{"disable", "require", "verify-ca", "verify-full"}
	validSSLMode := false
	for _, mode := range validSSLModes {
		if c.SSLMode == mode {
			validSSLMode = true
			break
		}
	}
	if !validSSLMode {
		return errors.New("invalid SSL mode, must be one of: disable, require, verify-ca, verify-full")
	}

	// Validate log level
	validLogLevels := []string{"silent", "error", "warn", "info"}
	validLogLevel := false
	for _, level := range validLogLevels {
		if c.LogLevel == level {
			validLogLevel = true
			break
		}
	}
	if !validLogLevel {
		return errors.New("invalid log level, must be one of: silent, error, warn, info")
	}

	// Validate connection pool settings
	if c.MaxIdleConns < 0 {
		return errors.New("max idle connections must be >= 0")
	}
	if c.MaxOpenConns < 0 {
		return errors.New("max open connections must be >= 0")
	}
	if c.MaxIdleConns > c.MaxOpenConns && c.MaxOpenConns > 0 {
		return errors.New("max idle connections cannot exceed max open connections")
	}
	if c.ConnMaxLifetime < 0 {
		return errors.New("connection max lifetime must be >= 0")
	}
	if c.ConnMaxIdleTime < 0 {
		return errors.New("connection max idle time must be >= 0")
	}
	if c.SlowThreshold < 0 {
		return errors.New("slow threshold must be >= 0")
	}

	return nil
}

// DSN returns the PostgreSQL connection DSN
func (c *Config) DSN() string {
	dsn := "host=" + c.Host +
		" port=" + fmt.Sprintf("%d", c.Port) +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.DBName +
		" sslmode=" + c.SSLMode +
		" TimeZone=" + c.Timezone

	if c.PreferSimpleProtocol {
		dsn += " prefer_simple_protocol=true"
	}

	return dsn
}
