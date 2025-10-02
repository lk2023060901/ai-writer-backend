package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	agentservice "github.com/lk2023060901/ai-writer-backend/internal/agent/service"
	"github.com/lk2023060901/ai-writer-backend/internal/auth/middleware"
	authservice "github.com/lk2023060901/ai-writer-backend/internal/auth/service"
	"github.com/lk2023060901/ai-writer-backend/internal/conf"
	kbservice "github.com/lk2023060901/ai-writer-backend/internal/knowledge/service"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/redis"
	"github.com/lk2023060901/ai-writer-backend/internal/user/service"
	"go.uber.org/zap"
)

type HTTPServer struct {
	server          *http.Server
	logger          *logger.Logger
	userService     *service.UserService
	authService     *authservice.AuthService
	agentService    *agentservice.AgentService
	aiConfigService *kbservice.AIProviderService
	kbService       *kbservice.KnowledgeBaseService
	documentService *kbservice.DocumentService
}

func NewHTTPServer(
	config *conf.Config,
	log *logger.Logger,
	userService *service.UserService,
	authService *authservice.AuthService,
	agentService *agentservice.AgentService,
	aiConfigService *kbservice.AIProviderService,
	kbService *kbservice.KnowledgeBaseService,
	documentService *kbservice.DocumentService,
	redisClient *redis.Client,
) *HTTPServer {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(LoggerMiddleware(log))
	router.Use(middleware.CORS())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Public API routes (no authentication required)
	publicAPI := router.Group("/api/v1")
	{
		// Auth endpoints with rate limiting
		auth := publicAPI.Group("/auth")
		{
			auth.POST("/register",
				middleware.RegisterRateLimiter(redisClient, log),
				authService.Register)
			auth.POST("/login",
				middleware.LoginRateLimiter(redisClient, log),
				authService.Login)
			auth.POST("/2fa/verify", authService.Verify2FA)
			auth.POST("/refresh", authService.RefreshToken)
		}
	}

	// Protected API routes (authentication required)
	protectedAPI := router.Group("/api/v1")
	protectedAPI.Use(middleware.JWTAuth(config.Auth.JWTSecret, log))
	protectedAPI.Use(middleware.APIRateLimiter(redisClient, log))
	{
		// Protected auth endpoints
		auth := protectedAPI.Group("/auth")
		{
			auth.POST("/2fa/enable", authService.Enable2FA)
			auth.GET("/2fa/qrcode", authService.GetQRCode)
			auth.POST("/2fa/confirm", authService.Confirm2FA)
			auth.POST("/2fa/disable", authService.Disable2FA)
		}

		// User management (protected)
		userService.RegisterRoutes(protectedAPI)

		// Agent management (protected)
		agents := protectedAPI.Group("/agents")
		{
			agents.POST("", agentService.CreateAgent)
			agents.GET("", agentService.ListAgents)
			agents.GET("/:id", agentService.GetAgent)
			agents.PUT("/:id", agentService.UpdateAgent)
			agents.DELETE("/:id", agentService.DeleteAgent)
			agents.PATCH("/:id/enable", agentService.EnableAgent)
			agents.PATCH("/:id/disable", agentService.DisableAgent)
		}

		// AI Provider Config routes (protected)
		aiProviders := protectedAPI.Group("/ai-providers")
		{
			aiProviders.GET("", aiConfigService.ListAIProviderConfigs)
			aiProviders.POST("", aiConfigService.CreateAIProviderConfig)
			aiProviders.GET("/:id", aiConfigService.GetAIProviderConfig)
			aiProviders.PUT("/:id", aiConfigService.UpdateAIProviderConfig)
			aiProviders.DELETE("/:id", aiConfigService.DeleteAIProviderConfig)
		}

		// Knowledge Base routes (protected)
		kbs := protectedAPI.Group("/knowledge-bases")
		{
			kbs.GET("", kbService.ListKnowledgeBases)
			kbs.POST("", kbService.CreateKnowledgeBase)
			kbs.GET("/:id", kbService.GetKnowledgeBase)
			kbs.PUT("/:id", kbService.UpdateKnowledgeBase)
			kbs.DELETE("/:id", kbService.DeleteKnowledgeBase)

			// Document routes (nested under knowledge bases)
			kbs.POST("/:id/documents", documentService.UploadDocument)
			kbs.GET("/:id/documents", documentService.ListDocuments)
			kbs.GET("/:id/documents/:doc_id", documentService.GetDocument)
			kbs.DELETE("/:id/documents/:doc_id", documentService.DeleteDocument)
			kbs.POST("/:id/documents/:doc_id/reprocess", documentService.ReprocessDocument)
			kbs.POST("/:id/search", documentService.SearchDocuments)
		}
	}

	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)

	return &HTTPServer{
		server: &http.Server{
			Addr:    addr,
			Handler: router,
		},
		logger:          log,
		userService:     userService,
		authService:     authService,
		agentService:    agentService,
		aiConfigService: aiConfigService,
		kbService:       kbService,
		documentService: documentService,
	}
}

func (s *HTTPServer) Start() error {
	s.logger.Info("starting HTTP server", zap.String("addr", s.server.Addr))

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *HTTPServer) Stop(ctx context.Context) error {
	s.logger.Info("stopping HTTP server")
	return s.server.Shutdown(ctx)
}

func LoggerMiddleware(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)

		log.Info("HTTP request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
		)
	}
}
