package data

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/conf"
	"github.com/lk2023060901/ai-writer-backend/internal/user/data"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Data struct {
	DB           *gorm.DB
	RedisClient  *redis.Client
	MinIOClient  *minio.Client
	MilvusClient client.Client
	Logger       *zap.Logger
}

func NewData(config *conf.Config, log *zap.Logger) (*Data, func(), error) {
	// Initialize PostgreSQL
	db, err := initDB(config, log)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init database: %w", err)
	}

	// Initialize Redis
	redisClient := initRedis(config)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// Initialize MinIO
	minioClient, err := initMinIO(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init minio: %w", err)
	}

	// Initialize Milvus
	milvusClient, err := initMilvus(config)
	if err != nil {
		log.Warn("failed to init milvus (this is optional)", zap.Error(err))
	}

	d := &Data{
		DB:           db,
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
			milvusClient.Close()
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

func initRedis(config *conf.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port),
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})
}

func initMinIO(config *conf.Config) (*minio.Client, error) {
	minioClient, err := minio.New(config.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.MinIO.AccessKey, config.MinIO.SecretKey, ""),
		Secure: config.MinIO.UseSSL,
	})
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
		err = minioClient.MakeBucket(ctx, config.MinIO.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
	}

	return minioClient, nil
}

func initMilvus(config *conf.Config) (client.Client, error) {
	addr := fmt.Sprintf("%s:%d", config.Milvus.Host, config.Milvus.Port)
	return client.NewClient(context.Background(), client.Config{
		Address: addr,
	})
}
