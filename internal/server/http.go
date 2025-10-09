package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	agentservice "github.com/lk2023060901/ai-writer-backend/internal/agent/service"
	assistantservice "github.com/lk2023060901/ai-writer-backend/internal/assistant/service"
	"github.com/lk2023060901/ai-writer-backend/internal/auth/middleware"
	authservice "github.com/lk2023060901/ai-writer-backend/internal/auth/service"
	"github.com/lk2023060901/ai-writer-backend/internal/conf"
	emailhandler "github.com/lk2023060901/ai-writer-backend/internal/email/handler"
	kbservice "github.com/lk2023060901/ai-writer-backend/internal/knowledge/service"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/redis"
	"github.com/lk2023060901/ai-writer-backend/internal/user/service"
	"go.uber.org/zap"
)

type HTTPServer struct {
	server                  *http.Server
	logger                  *logger.Logger
	userService             *service.UserService
	authService             *authservice.AuthService
	agentService            *agentservice.AgentService
	aiConfigService         *kbservice.AIProviderService
	aiModelService          *kbservice.AIModelService
	documentProviderService *kbservice.DocumentProviderService
	kbService               *kbservice.KnowledgeBaseService
	documentService         *kbservice.DocumentService
	assistantService        *assistantservice.AssistantService
	topicService            *assistantservice.TopicService
	messageService          *assistantservice.MessageService
	favoriteService         *assistantservice.FavoriteService
	emailHandler            *emailhandler.EmailHandler
	oauth2Handler           *emailhandler.OAuth2Handler
}

func NewHTTPServer(
	config *conf.Config,
	log *logger.Logger,
	userService *service.UserService,
	authService *authservice.AuthService,
	agentService *agentservice.AgentService,
	aiConfigService *kbservice.AIProviderService,
	aiModelService *kbservice.AIModelService,
	documentProviderService *kbservice.DocumentProviderService,
	kbService *kbservice.KnowledgeBaseService,
	documentService *kbservice.DocumentService,
	assistantService *assistantservice.AssistantService,
	topicService *assistantservice.TopicService,
	messageService *assistantservice.MessageService,
	favoriteService *assistantservice.FavoriteService,
	emailHandler *emailhandler.EmailHandler,
	oauth2Handler *emailhandler.OAuth2Handler,
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

		// OAuth2 callback route (public - Google redirects here without JWT)
		email := publicAPI.Group("/email")
		{
			email.GET("/oauth2/callback", oauth2Handler.Callback)
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

			// Import routes
			agents.POST("/import/file", agentService.ImportFromFile)
			agents.POST("/import/url", agentService.ImportFromURL)
		}

		// AI Provider routes (只读，系统预设)
		aiProviders := protectedAPI.Group("/ai-providers")
		{
			aiProviders.GET("", aiConfigService.ListAIProviders)
			aiProviders.GET("/with-models", aiConfigService.ListAllProvidersWithModels) // 新增：获取所有服务商及其模型
			aiProviders.PATCH("/:id/status", aiConfigService.UpdateAIProviderStatus)     // 更新服务商启用状态
			aiProviders.PUT("/:id", aiConfigService.UpdateAIProvider)                    // 更新服务商配置（API Key 和 API 地址）

			// AI Models routes (nested under providers)
			aiProviders.GET("/:provider_id/models", aiModelService.HandleListModelsByProvider)
			aiProviders.POST("/:provider_id/models/sync", aiModelService.HandleSyncProviderModels)
			aiProviders.GET("/:provider_id/models/sync-history", aiModelService.HandleGetSyncHistory)
		}

		// AI Models routes (global)
		aiModels := protectedAPI.Group("/ai-models")
		{
			aiModels.GET("", aiModelService.HandleListAllModels)
			aiModels.GET("/:id", aiModelService.HandleGetModelByID)
			aiModels.GET("/capability/:type", aiModelService.HandleListModelsByCapability)
		}

		// Document Provider routes (只读，系统预设)
		documentProviders := protectedAPI.Group("/document-providers")
		{
			documentProviders.GET("", documentProviderService.ListDocumentProviders)
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
			kbs.POST("/:id/documents/upload", documentService.UploadDocument)              // 单文件上传（返回 JSON）
			kbs.POST("/:id/documents/batch-upload", documentService.BatchUploadDocuments) // 批量上传文档（返回 SSE）
			kbs.GET("/:id/documents", documentService.ListDocuments)
			kbs.POST("/:id/documents/batch-delete", documentService.BatchDeleteDocuments)  // 批量删除文档
			kbs.GET("/:id/document-stream/:doc_id", documentService.StreamDocumentStatus)  // SSE (独立路径避免冲突)
			kbs.GET("/:id/documents/:doc_id", documentService.GetDocument)
			kbs.DELETE("/:id/documents/:doc_id", documentService.DeleteDocument)
			kbs.POST("/:id/documents/:doc_id/reprocess", documentService.ReprocessDocument)
			kbs.POST("/:id/search", documentService.SearchDocuments)
		}

		// Topic routes (protected)
		topicService.RegisterRoutes(protectedAPI)

		// Message routes (protected)
		messageService.RegisterRoutes(protectedAPI)

		// Favorite routes (protected)
		favoriteService.RegisterRoutes(protectedAPI)

		// Chat routes (protected) - Multi-provider streaming chat
		chat := protectedAPI.Group("/chat")
		{
			chat.POST("/stream", assistantService.ChatStreamV2)
		}

		// Email routes (protected)
		email := protectedAPI.Group("/email")
		{
			email.POST("/send", emailHandler.SendEmail)
			email.GET("/oauth2/auth-url", oauth2Handler.GetAuthURL)
			email.GET("/oauth2/status", oauth2Handler.GetStatus)
			email.POST("/oauth2/revoke", oauth2Handler.RevokeAuthorization)
		}
	}

	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)

	return &HTTPServer{
		server: &http.Server{
			Addr:    addr,
			Handler: router,
		},
		logger:                  log,
		userService:             userService,
		authService:             authService,
		agentService:            agentService,
		aiConfigService:         aiConfigService,
		aiModelService:          aiModelService,
		documentProviderService: documentProviderService,
		kbService:               kbService,
		documentService:         documentService,
		assistantService:        assistantService,
		topicService:            topicService,
		messageService:          messageService,
		favoriteService:         favoriteService,
		emailHandler:            emailHandler,
		oauth2Handler:           oauth2Handler,
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
