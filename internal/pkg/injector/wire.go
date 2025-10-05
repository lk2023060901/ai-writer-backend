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
	assistantbiz "github.com/lk2023060901/ai-writer-backend/internal/assistant/biz"
	assistantdata "github.com/lk2023060901/ai-writer-backend/internal/assistant/data"
	assistantservice "github.com/lk2023060901/ai-writer-backend/internal/assistant/service"
	authbiz "github.com/lk2023060901/ai-writer-backend/internal/auth/biz"
	authdata "github.com/lk2023060901/ai-writer-backend/internal/auth/data"
	authservice "github.com/lk2023060901/ai-writer-backend/internal/auth/service"
	"github.com/lk2023060901/ai-writer-backend/internal/conf"
	"github.com/lk2023060901/ai-writer-backend/internal/data"
	emailhandler "github.com/lk2023060901/ai-writer-backend/internal/email/handler"
	emailservice "github.com/lk2023060901/ai-writer-backend/internal/email/service"
	emailtypes "github.com/lk2023060901/ai-writer-backend/internal/email/types"
	kbbiz "github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	kbdata "github.com/lk2023060901/ai-writer-backend/internal/knowledge/data"
	kbembedding "github.com/lk2023060901/ai-writer-backend/internal/knowledge/embedding"
	kbprocessor "github.com/lk2023060901/ai-writer-backend/internal/knowledge/processor"
	kbqueue "github.com/lk2023060901/ai-writer-backend/internal/knowledge/queue"
	kbservice "github.com/lk2023060901/ai-writer-backend/internal/knowledge/service"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	oauth2pkg "github.com/lk2023060901/ai-writer-backend/internal/pkg/oauth2"
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
	provideTopicRepo,
	provideMessageRepo,
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
	assistantbiz.NewTopicUseCase,
	assistantbiz.NewMessageUseCase,
)

// Service providers
var serviceProviderSet = wire.NewSet(
	provideStorageService,
	provideVectorDBService,
	provideEmbeddingService,
	provideDocumentProcessor,
	provideEmailConfig,
	provideOAuth2Config,
	provideTokenStore,
	provideTokenProvider,
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
	assistantservice.NewTopicService,
	assistantservice.NewMessageService,
	provideEmailService,
	emailhandler.NewEmailHandler,
	emailhandler.NewOAuth2Handler,
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

func provideTopicRepo(d *data.Data) assistantbiz.TopicRepo {
	return assistantdata.NewTopicRepo(d.DBWrapper)
}

func provideMessageRepo(d *data.Data) assistantbiz.MessageRepo {
	return assistantdata.NewMessageRepo(d.DBWrapper)
}

// Service providers

func provideEmbeddingService() kbbiz.EmbeddingService {
	return kbembedding.NewEmbeddingService()
}

func provideDocumentProcessor() kbbiz.DocumentProcessor {
	return kbprocessor.NewDocumentProcessor()
}

// Email service providers

func provideEmailConfig(config *conf.Config) *emailtypes.EmailConfig {
	return &emailtypes.EmailConfig{
		SMTPHost:       config.Email.SMTPHost,
		SMTPPort:       config.Email.SMTPPort,
		FromAddr:       config.Email.FromAddr,
		FromName:       config.Email.FromName,
		OAuth2Enabled:  config.Email.OAuth2Enabled,
		MaxRetries:     config.Email.MaxRetries,
		RetryInterval:  config.Email.RetryInterval,
		ConnectTimeout: config.Email.ConnectTimeout,
		SendTimeout:    config.Email.SendTimeout,
	}
}

func provideOAuth2Config(config *conf.Config) *oauth2pkg.Config {
	return &oauth2pkg.Config{
		ClientID:     config.OAuth2.ClientID,
		ClientSecret: config.OAuth2.ClientSecret,
		RedirectURL:  config.OAuth2.RedirectURL,
		Scopes:       config.OAuth2.Scopes,
		AuthURL:      config.OAuth2.AuthURL,
		TokenURL:     config.OAuth2.TokenURL,
	}
}

func provideTokenStore(d *data.Data) (oauth2pkg.TokenStore, error) {
	return oauth2pkg.NewDatabaseTokenStore(d.DBWrapper, "gmail")
}

func provideTokenProvider(
	oauth2Config *oauth2pkg.Config,
	tokenStore oauth2pkg.TokenStore,
) (oauth2pkg.TokenProvider, error) {
	return oauth2pkg.NewGoogleTokenProvider(oauth2Config, tokenStore)
}

func provideEmailService(
	emailConfig *emailtypes.EmailConfig,
	tokenProvider oauth2pkg.TokenProvider,
) (*emailservice.EmailService, error) {
	return emailservice.NewEmailService(emailConfig, tokenProvider)
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
