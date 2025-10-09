//go:build wireinject
// +build wireinject

package injector

import (
	"context"
	"time"

	pb "github.com/lk2023060901/ai-writer-backend/api/auth/v1"
	"github.com/google/wire"
	agentbiz "github.com/lk2023060901/ai-writer-backend/internal/agent/biz"
	agentdata "github.com/lk2023060901/ai-writer-backend/internal/agent/data"
	agentservice "github.com/lk2023060901/ai-writer-backend/internal/agent/service"
	assistantbiz "github.com/lk2023060901/ai-writer-backend/internal/assistant/biz"
	assistantdata "github.com/lk2023060901/ai-writer-backend/internal/assistant/data"
	"github.com/lk2023060901/ai-writer-backend/internal/assistant/llm"
	llmproviders "github.com/lk2023060901/ai-writer-backend/internal/assistant/llm/providers"
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
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/mineru"
	oauth2pkg "github.com/lk2023060901/ai-writer-backend/internal/pkg/oauth2"
	pkgredis "github.com/lk2023060901/ai-writer-backend/internal/pkg/redis"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/sse"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/workerpool"
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
	provideAIProviderRepo,
	provideAIModelRepo,
	provideModelSyncLogRepo,
	provideDocumentProviderRepo,
	provideKnowledgeBaseRepo,
	provideDocumentRepo,
	provideChunkRepo,
	provideFileStorageRepo,
	provideAssistantRepo,
	provideTopicRepo,
	provideMessageRepo,
	provideFavoriteRepo,
)

// Use case providers
var useCaseProviderSet = wire.NewSet(
	provideZapLogger,
	userbiz.NewUserUseCase,
	provideAuthUseCase,
	agentbiz.NewAgentUseCase,
	kbbiz.NewAIProviderUseCase,
	kbbiz.NewAIModelUseCase,
	kbbiz.NewModelSyncUseCase,
	kbbiz.NewDocumentProviderUseCase,
	kbbiz.NewKnowledgeBaseUseCase,
	provideDocumentUseCase,
	assistantbiz.NewAssistantUseCase,
	assistantbiz.NewTopicUseCase,
	assistantbiz.NewMessageUseCase,
	assistantbiz.NewFavoriteUseCase,
)

// Service providers
var serviceProviderSet = wire.NewSet(
	provideStorageService,
	provideVectorDBService,
	provideEmbeddingService,
	provideMinerUClient,
	provideDocumentProcessor,
	provideEmailConfig,
	provideOAuth2Config,
	provideTokenStore,
	provideTokenProvider,
	provideSSEHub,
	provideProviderFactory,
	provideOrchestrator,
	provideUploadWorkerPool,
)

// HTTP/gRPC service providers
var httpServiceProviderSet = wire.NewSet(
	userservice.NewUserService,
	authservice.NewAuthService,
	provideGRPCAuthService,
	agentservice.NewAgentService,
	kbservice.NewAIProviderService,
	kbservice.NewAIModelService,
	kbservice.NewDocumentProviderService,
	kbservice.NewKnowledgeBaseService,
	kbservice.NewDocumentService,
	assistantservice.NewAssistantService,
	assistantservice.NewTopicService,
	assistantservice.NewMessageService,
	assistantservice.NewFavoriteService,
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

func provideSSEHub() *sse.Hub {
	return sse.NewHub()
}

func provideDocumentWorkerWithStart(
	d *data.Data,
	docUseCase *kbbiz.DocumentUseCase,
	sseHub *sse.Hub,
	log *logger.Logger,
) (*kbqueue.Worker, error) {
	worker := kbqueue.NewWorker(d.RedisClient, docUseCase, sseHub, log.Logger, 5)
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

func provideDocumentUseCase(
	documentRepo kbbiz.DocumentRepo,
	chunkRepo kbbiz.ChunkRepo,
	kbRepo kbbiz.KnowledgeBaseRepo,
	aiModelRepo kbbiz.AIModelRepo,
	aiProviderRepo kbbiz.AIProviderRepo,
	fileStorageRepo kbbiz.FileStorageRepo,
	storage kbbiz.StorageService,
	vectorDB kbbiz.VectorDBService,
	embedder kbbiz.EmbeddingService,
	processor kbbiz.DocumentProcessor,
	log *logger.Logger,
) *kbbiz.DocumentUseCase {
	return kbbiz.NewDocumentUseCase(
		documentRepo,
		chunkRepo,
		kbRepo,
		aiModelRepo,
		aiProviderRepo,
		fileStorageRepo,
		storage,
		vectorDB,
		embedder,
		processor,
		log,
	)
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

func provideAIProviderRepo(d *data.Data) kbbiz.AIProviderRepo {
	return kbdata.NewAIProviderRepo(d.DBWrapper)
}

func provideAIModelRepo(d *data.Data) kbbiz.AIModelRepo {
	return kbdata.NewAIModelRepo(d.DBWrapper)
}

func provideDocumentProviderRepo(d *data.Data) kbbiz.DocumentProviderRepo {
	return kbdata.NewDocumentProviderRepo(d.DBWrapper)
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

func provideFileStorageRepo(d *data.Data) kbbiz.FileStorageRepo {
	kbrepo := kbdata.NewFileStorageRepository(d.DBWrapper)
	return kbdata.NewFileStorageRepo(kbrepo)
}

func provideAssistantRepo(d *data.Data) assistantbiz.AssistantRepo {
	return assistantdata.NewAssistantRepo(d.DB)
}

func provideTopicRepo(d *data.Data) assistantbiz.TopicRepo {
	return assistantdata.NewTopicRepo(d.DBWrapper)
}

func provideMessageRepo(d *data.Data) assistantbiz.MessageRepo {
	return assistantdata.NewMessageRepo(d.DBWrapper)
}

func provideFavoriteRepo(d *data.Data) assistantbiz.FavoriteRepo {
	return assistantdata.NewFavoriteRepo(d.DBWrapper)
}

func provideModelSyncLogRepo(d *data.Data) kbbiz.ModelSyncLogRepo {
	return kbdata.NewModelSyncLogRepo(d.DBWrapper)
}

// Service providers

func provideEmbeddingService() kbbiz.EmbeddingService {
	return kbembedding.NewEmbeddingService()
}

func provideMinerUClient(config *conf.Config, log *logger.Logger) (*mineru.Client, error) {
	cfg := &mineru.Config{
		BaseURL:         config.MinerU.BaseURL,
		APIKey:          config.MinerU.APIKey,
		Timeout:         config.MinerU.Timeout,
		MaxRetries:      config.MinerU.MaxRetries,
		DefaultLanguage: config.MinerU.DefaultLanguage,
		EnableFormula:   config.MinerU.EnableFormula,
		EnableTable:     config.MinerU.EnableTable,
		ModelVersion:    config.MinerU.ModelVersion,
	}
	return mineru.New(cfg, log)
}

func provideDocumentProcessor(client *mineru.Client, log *logger.Logger) kbbiz.DocumentProcessor {
	return kbprocessor.NewMinerUProcessor(client, log)
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

// provideProviderFactory 提供 AI Provider 工厂
func provideProviderFactory(
	aiProviderUseCase *kbbiz.AIProviderUseCase,
	zapLogger *zap.Logger,
) llm.ProviderFactory {
	return llmproviders.NewDatabaseProviderFactory(aiProviderUseCase, zapLogger)
}

// provideOrchestrator 提供多服务商编排器
func provideOrchestrator(
	providerFactory llm.ProviderFactory,
	docUseCase *kbbiz.DocumentUseCase,
	zapLogger *zap.Logger,
) llm.MultiProviderOrchestrator {
	// 创建知识库适配器
	knowledgeSearcher := llm.NewKnowledgeAdapter(docUseCase)

	// 创建 Orchestrator
	return llm.NewOrchestrator(
		providerFactory,
		nil, // contextManager
		nil, // webSearch
		nil, // fileProcessor
		nil, // errorHandler
		nil, // metricsCollector
		knowledgeSearcher,
		zapLogger,
	)
}

// provideUploadWorkerPool 提供上传文件 Worker Pool
func provideUploadWorkerPool(
	config *conf.Config,
	log *logger.Logger,
) (*workerpool.Pool, error) {
	// 配置 Worker Pool（限制最大并发为 2）
	poolConfig := &workerpool.Config{
		InitialWorkers: 2,      // 初始 2 个 worker
		QueueSize:      1000,   // 队列大小 1000
		EnablePriority: false,  // 暂不启用优先级队列
		AutoScaling: &workerpool.AutoScalingConfig{
			Enable:                    true,
			MinWorkers:                2,
			MaxWorkers:                2,    // 最大 2 个 worker（固定并发为 2）
			ScaleUpQueueThreshold:     800,  // 队列 > 800 时扩容
			ScaleUpUtilizationRatio:   0.8,  // 利用率 > 80% 扩容
			ScaleDownUtilizationRatio: 0.2,  // 利用率 < 20% 缩容
			ScaleUpStep:               1,    // 每次扩容 1 个
			ScaleDownStep:             1,    // 每次缩容 1 个
			CooldownPeriod:            30 * time.Second, // 30 秒冷却期
			EnablePredictive:          true, // 启用预测式扩容
		},
	}

	// 创建 Worker Pool（自动启动）
	return workerpool.New(poolConfig, log.Logger)
}

func newApp(
	config *conf.Config,
	log *logger.Logger,
	httpServer *server.HTTPServer,
	grpcServer *server.GRPCServer,
	documentWorker *kbqueue.Worker,
	uploadPool *workerpool.Pool,
) (*App, func()) {
	// Cleanup function combines worker and data cleanup
	cleanup := func() {
		if documentWorker != nil {
			documentWorker.Stop()
		}
		if uploadPool != nil {
			uploadPool.Shutdown()
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
