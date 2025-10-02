package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	agentbiz "github.com/lk2023060901/ai-writer-backend/internal/agent/biz"
	agentdata "github.com/lk2023060901/ai-writer-backend/internal/agent/data"
	agentservice "github.com/lk2023060901/ai-writer-backend/internal/agent/service"
	authbiz "github.com/lk2023060901/ai-writer-backend/internal/auth/biz"
	authdata "github.com/lk2023060901/ai-writer-backend/internal/auth/data"
	authservice "github.com/lk2023060901/ai-writer-backend/internal/auth/service"
	"github.com/lk2023060901/ai-writer-backend/internal/conf"
	"github.com/lk2023060901/ai-writer-backend/internal/data"
	kbbiz "github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	kbdata "github.com/lk2023060901/ai-writer-backend/internal/knowledge/data"
	kbembedding "github.com/lk2023060901/ai-writer-backend/internal/knowledge/embedding"
	kbprocessor "github.com/lk2023060901/ai-writer-backend/internal/knowledge/processor"
	kbqueue "github.com/lk2023060901/ai-writer-backend/internal/knowledge/queue"
	kbservice "github.com/lk2023060901/ai-writer-backend/internal/knowledge/service"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"github.com/lk2023060901/ai-writer-backend/internal/server"
	"github.com/lk2023060901/ai-writer-backend/internal/user/biz"
	userdata "github.com/lk2023060901/ai-writer-backend/internal/user/data"
	"github.com/lk2023060901/ai-writer-backend/internal/user/service"
	"go.uber.org/zap"
)

var (
	configFile = flag.String("config", "config.yaml", "config file path")
)

func main() {
	flag.Parse()

	// Load configuration
	config, err := conf.LoadConfig(*configFile)
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Initialize logger with config
	logConfig := &logger.Config{
		Level:            config.Log.Level,
		Format:           config.Log.Format,
		Output:           config.Log.Output,
		EnableCaller:     config.Log.EnableCaller,
		EnableStacktrace: config.Log.EnableStacktrace,
		File: logger.FileConfig{
			Filename:   config.Log.File.Filename,
			MaxSize:    config.Log.File.MaxSize,
			MaxAge:     config.Log.File.MaxAge,
			MaxBackups: config.Log.File.MaxBackups,
			Compress:   config.Log.File.Compress,
		},
	}

	log, err := logger.New(logConfig)
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer log.Sync()

	// Initialize global logger
	if err := logger.InitGlobal(logConfig); err != nil {
		log.Fatal("failed to initialize global logger", zap.Error(err))
	}

	log.Info("config loaded successfully")

	// Initialize data layer
	d, cleanup, err := data.NewData(config, log.Logger)
	if err != nil {
		log.Fatal("failed to initialize data layer", zap.Error(err))
	}
	defer cleanup()

	// Initialize repositories
	userRepo := userdata.NewUserRepo(d.DB)
	authUserRepo := authdata.NewAuthUserRepo(d.DBWrapper)
	pendingAuthRepo := authbiz.NewRedisPendingAuthRepo(d.RedisClient)
	agentRepo := agentdata.NewAgentRepo(d.DBWrapper)
	officialAgentRepo := agentdata.NewOfficialAgentRepo(d.DBWrapper)
	aiConfigRepo := kbdata.NewAIProviderConfigRepo(d.DBWrapper)
	kbRepo := kbdata.NewKnowledgeBaseRepo(d.DBWrapper)
	documentRepo := kbdata.NewDocumentRepo(d.DBWrapper)
	chunkRepo := kbdata.NewChunkRepo(d.DBWrapper)

	// Initialize use cases
	userUseCase := biz.NewUserUseCase(userRepo)
	authUseCase := authbiz.NewAuthUseCase(
		authUserRepo,
		pendingAuthRepo,
		config.Auth.JWTSecret,
		config.Auth.TOTPIssuer,
	)
	agentUseCase := agentbiz.NewAgentUseCase(agentRepo, officialAgentRepo)
	aiConfigUseCase := kbbiz.NewAIProviderConfigUseCase(aiConfigRepo)
	kbUseCase := kbbiz.NewKnowledgeBaseUseCase(kbRepo, aiConfigRepo)

	// Initialize document services
	storageService := kbdata.NewMinIOStorageService(d.MinIOClient, config.MinIO.Bucket)
	vectorDBService := kbdata.NewMilvusVectorDBService(d.MilvusClient)
	embeddingService := kbembedding.NewEmbeddingService()
	documentProcessor := kbprocessor.NewDocumentProcessor()

	documentUseCase := kbbiz.NewDocumentUseCase(
		documentRepo,
		chunkRepo,
		kbRepo,
		aiConfigRepo,
		storageService,
		vectorDBService,
		embeddingService,
		documentProcessor,
	)

	// Initialize document worker
	documentWorker := kbqueue.NewWorker(
		d.RedisClient,
		documentUseCase,
		log.Logger,
		5, // worker count
	)

	// Start worker
	if err := documentWorker.Start(context.Background()); err != nil {
		log.Fatal("failed to start document worker", zap.Error(err))
	}
	defer documentWorker.Stop()

	// Initialize services
	userService := service.NewUserService(userUseCase, log.Logger)
	authService := authservice.NewAuthService(authUseCase, log)
	grpcAuthService := authservice.NewGRPCAuthService(authUseCase, log)
	agentService := agentservice.NewAgentService(agentUseCase, log)
	aiConfigService := kbservice.NewAIProviderService(aiConfigUseCase, log)
	kbService := kbservice.NewKnowledgeBaseService(kbUseCase, aiConfigUseCase, log)
	documentService := kbservice.NewDocumentService(documentUseCase, documentWorker, log.Logger)

	// Initialize servers
	httpServer := server.NewHTTPServer(config, log, userService, authService, agentService, aiConfigService, kbService, documentService, d.RedisClient)
	grpcServer := server.NewGRPCServer(config, log, grpcAuthService)

	// Start servers in goroutines
	go func() {
		if err := httpServer.Start(); err != nil {
			log.Fatal("failed to start HTTP server", zap.Error(err))
		}
	}()

	go func() {
		if err := grpcServer.Start(); err != nil {
			log.Fatal("failed to start gRPC server", zap.Error(err))
		}
	}()

	log.Info("servers started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down servers...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Stop gRPC server
	grpcServer.Stop()

	// Stop HTTP server
	if err := httpServer.Stop(ctx); err != nil {
		log.Error("HTTP server forced to shutdown", zap.Error(err))
	}

	log.Info("servers exited")
}
