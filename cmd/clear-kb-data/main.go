package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	PostgresDSN       string
	RedisAddr         string
	RedisPassword     string
	MinioEndpoint     string
	MinioAccessKey    string
	MinioSecretKey    string
	MinioBucket       string
	MinioUseSSL       bool
	MilvusHost        string
	MilvusPort        string
}

func main() {
	cfg := loadConfig()

	fmt.Println("==========================================")
	fmt.Println("清理所有知识库文件数据")
	fmt.Println("==========================================\n")

	ctx := context.Background()

	// 1. 连接数据库
	fmt.Println("1. 连接 PostgreSQL...")
	db, err := connectPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("连接 PostgreSQL 失败: %v", err)
	}
	fmt.Println("   ✓ PostgreSQL 连接成功\n")

	// 2. 统计清理前数据
	fmt.Println("2. 统计清理前数据...")
	printStats(db)

	// 3. 清理 PostgreSQL
	fmt.Println("\n3. 清理 PostgreSQL 数据...")
	chunkCount, docCount, fileCount := clearPostgres(db)
	fmt.Printf("   ✓ 已删除 %d 个分块\n", chunkCount)
	fmt.Printf("   ✓ 已删除 %d 个文档\n", docCount)
	fmt.Printf("   ✓ 已删除 %d 个文件存储记录\n", fileCount)

	// 4. 清理 Redis
	fmt.Println("\n4. 清理 Redis 缓存...")
	clearRedis(ctx, cfg)

	// 5. 清理 Milvus
	fmt.Println("\n5. 清理 Milvus 向量数据...")
	clearMilvus(ctx, cfg)

	// 6. 清理 MinIO
	fmt.Println("\n6. 清理 MinIO 文件存储...")
	minioCount := clearMinio(ctx, cfg)
	fmt.Printf("   ✓ 已删除 %d 个文件\n", minioCount)

	// 7. 更新知识库文档数量
	fmt.Println("\n7. 更新知识库文档数量...")
	updateKnowledgeBaseCount(db)

	// 8. 验证清理结果
	fmt.Println("\n8. 验证清理结果...")
	printStats(db)

	fmt.Println("\n==========================================")
	fmt.Println("清理完成！")
	fmt.Println("==========================================")
	fmt.Printf("\n清理汇总:\n")
	fmt.Printf("  - PostgreSQL 文档: %d\n", docCount)
	fmt.Printf("  - PostgreSQL 分块: %d\n", chunkCount)
	fmt.Printf("  - PostgreSQL 文件存储: %d\n", fileCount)
	fmt.Printf("  - MinIO 文件: %d\n", minioCount)
	fmt.Printf("  - Milvus 向量: 已清理所有集合\n")
	fmt.Printf("  - Redis 缓存: 已清理所有知识库相关缓存\n")
	fmt.Printf("  - 知识库文档计数: 已全部重置为 0\n\n")
}

func loadConfig() *Config {
	return &Config{
		PostgresDSN: getEnv("POSTGRES_DSN",
			"host=localhost port=5432 user=postgres password=postgres dbname=aiwriter sslmode=disable"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:    getEnv("MINIO_BUCKET", "aiwriter"),
		MinioUseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
		MilvusHost:     getEnv("MILVUS_HOST", "localhost"),
		MilvusPort:     getEnv("MILVUS_PORT", "19530"),
	}
}

func connectPostgres(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

func printStats(db *gorm.DB) {
	var stats []struct {
		TableName string
		Count     int64
	}

	db.Raw(`
		SELECT 'documents' as table_name, COUNT(*) as count FROM documents
		UNION ALL
		SELECT 'chunks' as table_name, COUNT(*) as count FROM chunks
		UNION ALL
		SELECT 'file_storage' as table_name, COUNT(*) as count FROM file_storage
	`).Scan(&stats)

	for _, s := range stats {
		fmt.Printf("   %s: %d\n", s.TableName, s.Count)
	}
}

func clearPostgres(db *gorm.DB) (int64, int64, int64) {
	var chunkCount, docCount, fileCount int64

	// 删除分块
	result := db.Exec("DELETE FROM chunks")
	chunkCount = result.RowsAffected

	// 删除文档
	result = db.Exec("DELETE FROM documents")
	docCount = result.RowsAffected

	// 删除文件存储
	result = db.Exec("DELETE FROM file_storage")
	fileCount = result.RowsAffected

	return chunkCount, docCount, fileCount
}

func clearRedis(ctx context.Context, cfg *Config) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})
	defer rdb.Close()

	// 清理所有 kb: 开头的 key
	iter := rdb.Scan(ctx, 0, "kb:*", 0).Iterator()
	count := 0
	for iter.Next(ctx) {
		rdb.Del(ctx, iter.Val())
		count++
	}
	if err := iter.Err(); err != nil {
		fmt.Printf("   ⚠ Redis 清理失败: %v\n", err)
		return
	}

	fmt.Printf("   ✓ 已清理 %d 个 Redis key\n", count)
}

func clearMilvus(ctx context.Context, cfg *Config) {
	c, err := client.NewClient(ctx, client.Config{
		Address: fmt.Sprintf("%s:%s", cfg.MilvusHost, cfg.MilvusPort),
	})
	if err != nil {
		fmt.Printf("   ⚠ Milvus 连接失败: %v\n", err)
		return
	}
	defer c.Close()

	// 获取所有集合
	collections, err := c.ListCollections(ctx)
	if err != nil {
		fmt.Printf("   ⚠ 获取 Milvus 集合失败: %v\n", err)
		return
	}

	fmt.Printf("   找到 %d 个 Milvus 集合\n", len(collections))

	// 删除所有集合
	for _, coll := range collections {
		err = c.DropCollection(ctx, coll.Name)
		if err != nil {
			fmt.Printf("   ✗ 删除集合 %s 失败: %v\n", coll.Name, err)
		} else {
			fmt.Printf("   ✓ 已删除集合: %s\n", coll.Name)
		}
	}
}

func clearMinio(ctx context.Context, cfg *Config) int {
	minioClient, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		fmt.Printf("   ⚠ MinIO 连接失败: %v\n", err)
		return 0
	}

	// 检查 bucket 是否存在
	exists, err := minioClient.BucketExists(ctx, cfg.MinioBucket)
	if err != nil || !exists {
		fmt.Printf("   ⚠ Bucket '%s' 不存在\n", cfg.MinioBucket)
		return 0
	}

	count := 0

	// 清理 knowledge_bases 目录
	count += removePrefix(ctx, minioClient, cfg.MinioBucket, "knowledge_bases/")

	// 清理 files 目录
	count += removePrefix(ctx, minioClient, cfg.MinioBucket, "files/")

	return count
}

func removePrefix(ctx context.Context, client *minio.Client, bucket, prefix string) int {
	count := 0
	objectCh := client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			continue
		}

		err := client.RemoveObject(ctx, bucket, object.Key, minio.RemoveObjectOptions{})
		if err == nil {
			count++
		}
	}

	return count
}

func updateKnowledgeBaseCount(db *gorm.DB) {
	db.Exec("UPDATE knowledge_bases SET document_count = 0")

	var kbs []struct {
		ID            string
		Name          string
		DocumentCount int64
		ActualCount   int64
	}

	db.Raw(`
		SELECT 
			LEFT(id::TEXT, 8) as id,
			LEFT(name, 30) as name,
			document_count,
			(SELECT COUNT(*) FROM documents WHERE knowledge_base_id = knowledge_bases.id) as actual_count
		FROM knowledge_bases
		ORDER BY name
	`).Scan(&kbs)

	for _, kb := range kbs {
		fmt.Printf("   %s | %s | 文档数: %d | 实际数: %d\n",
			kb.ID, kb.Name, kb.DocumentCount, kb.ActualCount)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
