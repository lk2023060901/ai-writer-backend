//go:build wireinject
// +build wireinject

package injector

import (
	"context"

	pb "github.com/lk2023060901/ai-writer-backend/api/auth/v1"
	"github.com/google/wire"
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
	pkgredis "github.com/lk2023060901/ai-writer-backend/internal/pkg/redis"
	"github.com/lk2023060901/ai-writer-backend/internal/server"
	userbiz "github.com/lk2023060901/ai-writer-backend/internal/user/biz"
	userdata "github.com/lk2023060901/ai-writer-backend/internal/user/data"
	userservice "github.com/lk2023060901/ai-writer-backend/internal/user/service"
	"go.uber.org/zap"
)

// ProviderSet is the Wire provider set for all dependencies
var ProviderSet = wire.NewSet(
	// Data layer
	dataProviderSet,

	// Repositories
	repositoryProviderSet,

	// Use cases
	useCaseProviderSet,

	// Services
	serviceProviderSet,

	// HTTP/gRPC services
	httpServiceProviderSet,

	// Servers
	serverProviderSet,
)

// Data layer providers
var dataProviderSet = wire.NewSet(
	provideData,
	provideRedisClient,
)

// Repository providers
var repositoryProviderSet = wire.NewSet(
	provideUserRepo,
	provideAuthUserRepo,
	providePendingAuthRepo,
	provideAgentRepo,
	provideOfficialAgentRepo,
	provideAIProviderConfigRepo,
	provideKnowledgeBaseRepo,
	provideDocumentRepo,
	provideChunkRepo,
)

// Use case providers
var useCaseProviderSet = wire.NewSet(
	provideZapLogger,
	userbiz.NewUserUseCase,
	provideAuthUseCase,
	agentbiz.NewAgentUseCase,
	kbbiz.NewAIProviderConfigUseCase,
	kbbiz.NewKnowledgeBaseUseCase,
	kbbiz.NewDocumentUseCase,
)

// Service providers
var serviceProviderSet = wire.NewSet(
	provideStorageService,
	provideVectorDBService,
	provideEmbeddingService,
	provideDocumentProcessor,
)

// HTTP/gRPC service providers
var httpServiceProviderSet = wire.NewSet(
	userservice.NewUserService,
	authservice.NewAuthService,
	provideGRPCAuthService,
	agentservice.NewAgentService,
	kbservice.NewAIProviderService,
	kbservice.NewKnowledgeBaseService,
	kbservice.NewDocumentService,
)

// Server providers
var serverProviderSet = wire.NewSet(
	server.NewHTTPServer,
	server.NewGRPCServer,
	provideDocumentWorkerWithStart,
)

// InitializeApp initializes the application with Wire
func InitializeApp(config *conf.Config, log *logger.Logger) (*App, func(), error) {
	wire.Build(ProviderSet, newApp)
	return nil, nil, nil
}

// Provider functions for complex dependencies

func provideAuthUseCase(
	userRepo authbiz.UserRepo,
	pendingRepo authbiz.PendingAuthRepo,
	config *conf.Config,
) *authbiz.AuthUseCase {
	return authbiz.NewAuthUseCase(
		userRepo,
		pendingRepo,
		config.Auth.JWTSecret,
		config.Auth.TOTPIssuer,
	)
}

func provideStorageService(
	d *data.Data,
	config *conf.Config,
) kbbiz.StorageService {
	return kbdata.NewMinIOStorageService(d.MinIOClient, config.MinIO.Bucket)
}

func provideVectorDBService(d *data.Data) kbbiz.VectorDBService {
	return kbdata.NewMilvusVectorDBService(d.MilvusClient)
}

func provideDocumentWorkerWithStart(
	d *data.Data,
	docUseCase *kbbiz.DocumentUseCase,
	log *logger.Logger,
) (*kbqueue.Worker, error) {
	worker := kbqueue.NewWorker(d.RedisClient, docUseCase, log.Logger, 5)
	if err := worker.Start(context.Background()); err != nil {
		return nil, err
	}
	return worker, nil
}

func provideGRPCAuthService(
	authUC *authbiz.AuthUseCase,
	log *logger.Logger,
) pb.AuthServiceServer {
	return authservice.NewGRPCAuthService(authUC, log)
}

// Data layer helpers

func provideData(config *conf.Config, log *logger.Logger) (*data.Data, func(), error) {
	return data.NewData(config, log.Logger)
}

func provideRedisClient(d *data.Data) *pkgredis.Client {
	return d.RedisClient
}

func provideZapLogger(log *logger.Logger) *zap.Logger {
	return log.Logger
}

// Repository providers

func provideUserRepo(d *data.Data) userbiz.UserRepo {
	return userdata.NewUserRepo(d.DB)
}

func provideAuthUserRepo(d *data.Data) authbiz.UserRepo {
	return authdata.NewAuthUserRepo(d.DBWrapper)
}

func providePendingAuthRepo(d *data.Data) authbiz.PendingAuthRepo {
	return authbiz.NewRedisPendingAuthRepo(d.RedisClient)
}

func provideAgentRepo(d *data.Data) agentbiz.AgentRepo {
	return agentdata.NewAgentRepo(d.DBWrapper)
}

func provideOfficialAgentRepo(d *data.Data) agentbiz.OfficialAgentRepo {
	return agentdata.NewOfficialAgentRepo(d.DBWrapper)
}

func provideAIProviderConfigRepo(d *data.Data) kbbiz.AIProviderConfigRepo {
	return kbdata.NewAIProviderConfigRepo(d.DBWrapper)
}

func provideKnowledgeBaseRepo(d *data.Data) kbbiz.KnowledgeBaseRepo {
	return kbdata.NewKnowledgeBaseRepo(d.DBWrapper)
}

func provideDocumentRepo(d *data.Data) kbbiz.DocumentRepo {
	return kbdata.NewDocumentRepo(d.DBWrapper)
}

func provideChunkRepo(d *data.Data) kbbiz.ChunkRepo {
	return kbdata.NewChunkRepo(d.DBWrapper)
}

// Service providers

func provideEmbeddingService() kbbiz.EmbeddingService {
	return kbembedding.NewEmbeddingService()
}

func provideDocumentProcessor() kbbiz.DocumentProcessor {
	return kbprocessor.NewDocumentProcessor()
}

func newApp(
	config *conf.Config,
	log *logger.Logger,
	httpServer *server.HTTPServer,
	grpcServer *server.GRPCServer,
	documentWorker *kbqueue.Worker,
) (*App, func()) {
	// Cleanup function combines worker and data cleanup
	cleanup := func() {
		if documentWorker != nil {
			documentWorker.Stop()
		}
	}

	return &App{
		Config:         config,
		Logger:         log,
		HTTPServer:     httpServer,
		GRPCServer:     grpcServer,
		DocumentWorker: documentWorker,
		cleanup:        cleanup,
	}, cleanup
}
