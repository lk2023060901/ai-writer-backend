package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/ai-writer-backend/internal/auth"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"go.uber.org/zap"
)

// JWTAuth JWT 认证中间件
func JWTAuth(jwtSecret string, log *logger.Logger) gin.HandlerFunc {
	jwtManager := auth.NewJWTManager(jwtSecret)

	return func(c *gin.Context) {
		var token string
		var err error

		// 优先从 Authorization header 获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			token, err = auth.ExtractTokenFromHeader(authHeader)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
				c.Abort()
				return
			}
		} else {
			// 如果 header 没有,尝试从查询参数获取 (用于 SSE)
			token = c.Query("token")
			if token == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
				c.Abort()
				return
			}
		}

		// 验证 token
		claims, err := jwtManager.VerifyAccessToken(token)
		if err != nil {
			log.Warn("invalid access token",
				zap.Error(err),
				zap.String("ip", c.ClientIP()))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// 将用户信息注入到上下文
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		c.Next()
	}
}

// OptionalJWTAuth 可选的 JWT 认证中间件（token 无效不拦截）
func OptionalJWTAuth(jwtSecret string, log *logger.Logger) gin.HandlerFunc {
	jwtManager := auth.NewJWTManager(jwtSecret)

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		token, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.Next()
			return
		}

		claims, err := jwtManager.VerifyAccessToken(token)
		if err != nil {
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}

// RequireRole 角色验证中间件（需要先经过 JWTAuth）
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户角色（需要在 JWT claims 中添加 role 字段）
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			c.Abort()
			return
		}

		// 检查角色
		roleStr := userRole.(string)
		for _, role := range roles {
			if roleStr == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		c.Abort()
	}
}

// GetUserID 从上下文获取用户 ID
func GetUserID(c *gin.Context) (int64, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	return userID.(int64), true
}

// GetEmail 从上下文获取用户邮箱
func GetEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get("email")
	if !exists {
		return "", false
	}
	return email.(string), true
}

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")

		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE, PATCH")
			c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
