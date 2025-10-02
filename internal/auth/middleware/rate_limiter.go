package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/redis"
	"go.uber.org/zap"
)

// RateLimiterConfig 限流配置
type RateLimiterConfig struct {
	// 时间窗口内允许的最大请求数
	MaxRequests int
	// 时间窗口（秒）
	WindowSeconds int
	// 限流策略：user, endpoint, ip（默认）
	Strategy string
}

// RateLimiter 基于 Redis 的滑动窗口限流中间件
func RateLimiter(redisClient *redis.Client, cfg RateLimiterConfig, log *logger.Logger) gin.HandlerFunc {
	if cfg.MaxRequests <= 0 {
		cfg.MaxRequests = 100
	}
	if cfg.WindowSeconds <= 0 {
		cfg.WindowSeconds = 60
	}
	if cfg.Strategy == "" {
		cfg.Strategy = "ip"
	}

	return func(c *gin.Context) {
		// 构建限流 key
		key := buildRateLimitKey(c, cfg.Strategy)

		ctx := c.Request.Context()
		allowed, remaining, resetTime, err := checkRateLimit(ctx, redisClient, key, cfg)

		if err != nil {
			log.Error("rate limiter error", zap.Error(err), zap.String("key", key))
			// 限流器故障时，降级允许请求通过
			c.Next()
			return
		}

		// 设置响应头
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.MaxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))

		if !allowed {
			c.Header("Retry-After", fmt.Sprintf("%d", cfg.WindowSeconds))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": fmt.Sprintf("too many requests, please try again in %d seconds", cfg.WindowSeconds),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// buildRateLimitKey 构建限流 key
func buildRateLimitKey(c *gin.Context, strategy string) string {
	prefix := "rate_limit"

	switch strategy {
	case "user":
		// 基于用户 ID 限流（需要先经过认证中间件）
		if userID, exists := c.Get("user_id"); exists {
			return fmt.Sprintf("%s:user:%v", prefix, userID)
		}
		// 未认证用户回退到 IP 限流
		return fmt.Sprintf("%s:ip:%s", prefix, c.ClientIP())

	case "endpoint":
		// 基于端点 + IP 限流
		return fmt.Sprintf("%s:endpoint:%s:%s", prefix, c.Request.URL.Path, c.ClientIP())

	default:
		// 默认使用 IP 限流（包括显式指定 "ip" 和任何未知策略）
		return fmt.Sprintf("%s:ip:%s", prefix, c.ClientIP())
	}
}

// checkRateLimit 使用 Redis 滑动窗口算法检查限流
func checkRateLimit(ctx context.Context, redisClient *redis.Client, key string, cfg RateLimiterConfig) (allowed bool, remaining int, resetTime int64, err error) {
	now := time.Now().Unix()

	// Lua 脚本实现原子性滑动窗口限流
	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local window_start = now - window

		-- 删除窗口外的记录
		redis.call('ZREMRANGEBYSCORE', key, 0, window_start)

		-- 获取当前窗口内的请求数
		local current = redis.call('ZCARD', key)

		if current < limit then
			-- 未超限，记录本次请求
			redis.call('ZADD', key, now, now)
			redis.call('EXPIRE', key, window)
			return {1, limit - current - 1, now + window}
		else
			-- 超限
			local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')[2]
			local reset_time = tonumber(oldest) + window
			return {0, 0, reset_time}
		end
	`

	// 使用 internal/pkg/redis 封装的方法执行 Lua 脚本
	result, err := redisClient.Eval(ctx, script, []string{key}, now, cfg.WindowSeconds, cfg.MaxRequests)
	if err != nil {
		return false, 0, 0, err
	}

	// 解析结果
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) != 3 {
		return false, 0, 0, fmt.Errorf("invalid rate limit result")
	}

	allowedInt, _ := resultSlice[0].(int64)
	remainingInt, _ := resultSlice[1].(int64)
	resetTimeInt, _ := resultSlice[2].(int64)

	return allowedInt == 1, int(remainingInt), resetTimeInt, nil
}

// LoginRateLimiter 登录端点专用限流（更严格）
// 5 次请求 / 5 分钟（基于 IP）
func LoginRateLimiter(redisClient *redis.Client, log *logger.Logger) gin.HandlerFunc {
	return RateLimiter(redisClient, RateLimiterConfig{
		MaxRequests:   5,
		WindowSeconds: 300,
		Strategy:      "ip",
	}, log)
}

// RegisterRateLimiter 注册端点专用限流
// 3 次请求 / 1 小时（基于 IP）
func RegisterRateLimiter(redisClient *redis.Client, log *logger.Logger) gin.HandlerFunc {
	return RateLimiter(redisClient, RateLimiterConfig{
		MaxRequests:   3,
		WindowSeconds: 3600,
		Strategy:      "ip",
	}, log)
}

// APIRateLimiter 通用 API 限流
// 100 次请求 / 1 分钟（基于用户 ID）
func APIRateLimiter(redisClient *redis.Client, log *logger.Logger) gin.HandlerFunc {
	return RateLimiter(redisClient, RateLimiterConfig{
		MaxRequests:   100,
		WindowSeconds: 60,
		Strategy:      "user",
	}, log)
}
