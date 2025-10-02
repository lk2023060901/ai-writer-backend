package data

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/conf"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
	pkglogger "github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	pkgmilvus "github.com/lk2023060901/ai-writer-backend/internal/pkg/milvus"
	pkgminio "github.com/lk2023060901/ai-writer-backend/internal/pkg/minio"
	pkgredis "github.com/lk2023060901/ai-writer-backend/internal/pkg/redis"
	"github.com/lk2023060901/ai-writer-backend/internal/user/data"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Data struct {
	DB           *gorm.DB
	DBWrapper    *database.DB
	RedisClient  *pkgredis.Client
	MinIOClient  *pkgminio.Client
	MilvusClient *pkgmilvus.Client
	Logger       *zap.Logger
}

func NewData(config *conf.Config, log *zap.Logger) (*Data, func(), error) {
	// Initialize PostgreSQL
	db, err := initDB(config, log)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init database: %w", err)
	}

	// Initialize Redis using internal wrapper
	wrapperLog, err := pkglogger.New(&pkglogger.Config{
		Level:  "info",
		Format: "json",
		Output: "console",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create redis logger: %w", err)
	}

	redisClient, err := initRedis(config, wrapperLog)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init redis: %w", err)
	}

	// Initialize MinIO
	minioClient, err := initMinIO(config, log)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init minio: %w", err)
	}

	// Initialize Milvus
	milvusClient, err := initMilvus(config, wrapperLog)
	if err != nil {
		log.Warn("failed to init milvus (this is optional)", zap.Error(err))
	}

	// Initialize database wrapper
	dbWrapper, err := database.New(&database.Config{
		Host:     config.Database.Host,
		Port:     config.Database.Port,
		User:     config.Database.User,
		Password: config.Database.Password,
		DBName:   config.Database.DBName,
		SSLMode:  config.Database.SSLMode,
		LogLevel: "warn",
		Timezone: "Asia/Shanghai",
	}, wrapperLog)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init database wrapper: %w", err)
	}

	d := &Data{
		DB:           db,
		DBWrapper:    dbWrapper,
		RedisClient:  redisClient,
		MinIOClient:  minioClient,
		MilvusClient: milvusClient,
		Logger:       log,
	}

	cleanup := func() {
		log.Info("cleaning up data resources")

		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}

		if redisClient != nil {
			redisClient.Close()
		}

		if milvusClient != nil {
			milvusClient.Close(context.Background())
		}
	}

	return d, cleanup, nil
}

func initDB(config *conf.Config, log *zap.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(config.Database.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto migrate
	if err := db.AutoMigrate(&data.UserPO{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	log.Info("database initialized successfully")
	return db, nil
}

func initRedis(config *conf.Config, log *pkglogger.Logger) (*pkgredis.Client, error) {
	redisCfg := &pkgredis.Config{
		Mode:         pkgredis.ModeSingle,
		MasterAddr:   fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port),
		Password:     config.Redis.Password,
		DB:           config.Redis.DB,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	}
	return pkgredis.New(redisCfg, log)
}

func initMinIO(config *conf.Config, log *zap.Logger) (*pkgminio.Client, error) {
	minioCfg := &pkgminio.Config{
		Endpoint:        config.MinIO.Endpoint,
		AccessKeyID:     config.MinIO.AccessKey,
		SecretAccessKey: config.MinIO.SecretKey,
		UseSSL:          config.MinIO.UseSSL,
		BucketLookup:    pkgminio.BucketLookupAuto,
	}

	minioClient, err := pkgminio.NewClient(minioCfg, log)
	if err != nil {
		return nil, err
	}

	// Create bucket if not exists
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, config.MinIO.Bucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, config.MinIO.Bucket, pkgminio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
	}

	return minioClient, nil
}

func initMilvus(config *conf.Config, log *pkglogger.Logger) (*pkgmilvus.Client, error) {
	addr := fmt.Sprintf("%s:%d", config.Milvus.Host, config.Milvus.Port)
	milvusCfg := &pkgmilvus.Config{
		Address:  addr,
		Database: "default",
	}

	return pkgmilvus.New(context.Background(), milvusCfg, log)
}
