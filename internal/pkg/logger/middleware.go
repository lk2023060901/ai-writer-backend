package logger

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GinLogger returns a gin middleware for logging HTTP requests
func GinLogger(logger *Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add request ID to context
		ctx := WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		// Set request ID in response header
		c.Header("X-Request-ID", requestID)

		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		statusCode := c.Writer.Status()

		// Build log fields
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// Add error if present
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// Log based on status code
		switch {
		case statusCode >= 500:
			logger.Error("HTTP Request", fields...)
		case statusCode >= 400:
			logger.Warn("HTTP Request", fields...)
		default:
			logger.Info("HTTP Request", fields...)
		}
	}
}

// GinRecovery returns a gin middleware for recovering from panics
func GinRecovery(logger *Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get request ID from context
				requestID := GetRequestID(c.Request.Context())

				// Log the panic
				logger.Error("Panic recovered",
					zap.String("request_id", requestID),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.Any("error", err),
					zap.Stack("stacktrace"),
				)

				// Return 500 error
				c.AbortWithStatus(500)
			}
		}()

		c.Next()
	}
}

// MiddlewareOptions configures the logger middleware
type MiddlewareOptions struct {
	// SkipPaths is a list of paths to skip logging
	SkipPaths []string
	// SkipPathPrefixes is a list of path prefixes to skip logging
	SkipPathPrefixes []string
	// EnableRequestBody enables logging request body
	EnableRequestBody bool
	// EnableResponseBody enables logging response body
	EnableResponseBody bool
}

// GinLoggerWithConfig returns a gin middleware with custom configuration
func GinLoggerWithConfig(logger *Logger, opts MiddlewareOptions) gin.HandlerFunc {
	// Build skip path map for fast lookup
	skipPaths := make(map[string]bool)
	for _, path := range opts.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Generate request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add request ID to context
		ctx := WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		// Set request ID in response header
		c.Header("X-Request-ID", requestID)

		path := c.Request.URL.Path

		// Check if path should be skipped
		if skipPaths[path] {
			c.Next()
			return
		}

		// Check path prefixes
		for _, prefix := range opts.SkipPathPrefixes {
			if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
				c.Next()
				return
			}
		}

		// Start timer
		start := time.Now()
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		statusCode := c.Writer.Status()

		// Build log fields
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// Add error if present
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// Log based on status code
		switch {
		case statusCode >= 500:
			logger.Error("HTTP Request", fields...)
		case statusCode >= 400:
			logger.Warn("HTTP Request", fields...)
		default:
			logger.Info("HTTP Request", fields...)
		}
	}
}
