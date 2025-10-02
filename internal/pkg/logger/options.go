package logger

// Option defines a function to modify logger configuration
type Option func(*Config)

// WithLevel sets the log level
func WithLevel(level string) Option {
	return func(c *Config) {
		c.Level = level
	}
}

// WithFormat sets the log format (json or console)
func WithFormat(format string) Option {
	return func(c *Config) {
		c.Format = format
	}
}

// WithOutput sets the log output (console, file, or both)
func WithOutput(output string) Option {
	return func(c *Config) {
		c.Output = output
	}
}

// WithFilename sets the log file name
func WithFilename(filename string) Option {
	return func(c *Config) {
		c.File.Filename = filename
	}
}

// WithMaxSize sets the maximum size in megabytes before rotation
func WithMaxSize(maxSize int) Option {
	return func(c *Config) {
		c.File.MaxSize = maxSize
	}
}

// WithMaxAge sets the maximum number of days to retain old log files
func WithMaxAge(maxAge int) Option {
	return func(c *Config) {
		c.File.MaxAge = maxAge
	}
}

// WithMaxBackups sets the maximum number of old log files to retain
func WithMaxBackups(maxBackups int) Option {
	return func(c *Config) {
		c.File.MaxBackups = maxBackups
	}
}

// WithCompress enables or disables compression of rotated log files
func WithCompress(compress bool) Option {
	return func(c *Config) {
		c.File.Compress = compress
	}
}

// WithCaller enables or disables caller information
func WithCaller(enabled bool) Option {
	return func(c *Config) {
		c.EnableCaller = enabled
	}
}

// WithStacktrace enables or disables stacktrace for error level
func WithStacktrace(enabled bool) Option {
	return func(c *Config) {
		c.EnableStacktrace = enabled
	}
}

// NewWithOptions creates a new logger with options
func NewWithOptions(opts ...Option) (*Logger, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return New(cfg)
}

// Development returns a logger optimized for development
// - Console output
// - Debug level
// - Console format with colors
// - Caller and stacktrace enabled
func Development() (*Logger, error) {
	return NewWithOptions(
		WithLevel("debug"),
		WithFormat("console"),
		WithOutput("console"),
		WithCaller(true),
		WithStacktrace(true),
	)
}

// Production returns a logger optimized for production
// - File output
// - Info level
// - JSON format
// - Caller enabled, stacktrace for errors
func Production(filename string) (*Logger, error) {
	return NewWithOptions(
		WithLevel("info"),
		WithFormat("json"),
		WithOutput("file"),
		WithFilename(filename),
		WithMaxSize(100),
		WithMaxAge(30),
		WithMaxBackups(10),
		WithCompress(true),
		WithCaller(true),
		WithStacktrace(true),
	)
}

