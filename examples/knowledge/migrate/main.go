package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/models"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
)

func main() {
	// 初始化日志
	logCfg := logger.DefaultConfig()
	if err := logger.InitGlobal(logCfg); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Knowledge base migration example")

	// 数据库配置
	dbCfg := &database.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "ai_writer",
		SSLMode:  "disable",
	}

	// 连接数据库
	db, err := database.New(dbCfg, logger.L())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if sqlDB, err := db.DB.DB(); err == nil {
			sqlDB.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 执行迁移
	logger.Info("starting migration...")
	if err := models.MigrateWithLog(ctx, db, logger.L().Logger); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	logger.Info("migration completed successfully!")
	fmt.Println("\n✅ Database schema migrated successfully")
	fmt.Println("\nCreated tables:")
	fmt.Println("  - knowledge_bases")
	fmt.Println("  - documents")
	fmt.Println("  - chunks")
}
