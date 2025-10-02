package logger

import (
	"errors"
	"strings"
)

// Config defines the logger configuration
type Config struct {
	Level      string     `mapstructure:"level"`      // debug, info, warn, error
	Format     string     `mapstructure:"format"`     // json, console
	Output     string     `mapstructure:"output"`     // console, file, both
	File       FileConfig `mapstructure:"file"`
	EnableCaller bool      `mapstructure:"enablecaller"` // enable caller info
	EnableStacktrace bool  `mapstructure:"enablestacktrace"` // enable stacktrace for error level
}

// FileConfig defines file output configuration
type FileConfig struct {
	Filename   string `mapstructure:"filename"`   // log file path
	MaxSize    int    `mapstructure:"maxsize"`    // max size in MB
	MaxAge     int    `mapstructure:"maxage"`     // max age in days
	MaxBackups int    `mapstructure:"maxbackups"` // max backup files
	Compress   bool   `mapstructure:"compress"`   // compress rotated files
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:            "info",
		Format:           "json",
		Output:           "console",
		EnableCaller:     true,
		EnableStacktrace: true,
		File: FileConfig{
			Filename:   "logs/app.log",
			MaxSize:    100,
			MaxAge:     30,
			MaxBackups: 10,
			Compress:   true,
		},
	}
}

// Validate validates the logger configuration
func (c *Config) Validate() error {
	// Validate level
	validLevels := []string{"debug", "info", "warn", "error", "dpanic", "panic", "fatal"}
	levelValid := false
	for _, level := range validLevels {
		if strings.ToLower(c.Level) == level {
			levelValid = true
			break
		}
	}
	if !levelValid {
		return errors.New("invalid log level, must be one of: debug, info, warn, error, dpanic, panic, fatal")
	}

	// Validate format
	if c.Format != "json" && c.Format != "console" {
		return errors.New("invalid log format, must be 'json' or 'console'")
	}

	// Validate output
	if c.Output != "console" && c.Output != "file" && c.Output != "both" {
		return errors.New("invalid log output, must be 'console', 'file' or 'both'")
	}

	// Validate file config when output is file or both
	if c.Output == "file" || c.Output == "both" {
		if c.File.Filename == "" {
			return errors.New("log file filename is required when output is 'file' or 'both'")
		}
		if c.File.MaxSize <= 0 {
			return errors.New("log file maxsize must be greater than 0")
		}
		if c.File.MaxAge <= 0 {
			return errors.New("log file maxage must be greater than 0")
		}
		if c.File.MaxBackups < 0 {
			return errors.New("log file maxbackups must be greater than or equal to 0")
		}
	}

	return nil
}
