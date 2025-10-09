package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/conf"
	"github.com/lk2023060901/ai-writer-backend/internal/data"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	kbdata "github.com/lk2023060901/ai-writer-backend/internal/knowledge/data"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func initZapLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	config.OutputPaths = []string{"stdout"}
	return config.Build()
}

func main() {
	fmt.Println("🚀 AI 模型同步工具启动...")
	fmt.Println()

	// Load config
	cfg, err := conf.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	// Initialize logger
	zapLogger, err := initZapLogger()
	if err != nil {
		log.Fatalf("❌ 初始化日志失败: %v", err)
	}
	defer zapLogger.Sync()

	// Initialize data layer (includes DB, Redis, Milvus, etc.)
	d, cleanup, err := data.NewData(cfg, zapLogger)
	if err != nil {
		log.Fatalf("❌ 初始化数据层失败: %v", err)
	}
	defer cleanup()

	// Create repositories
	providerRepo := kbdata.NewAIProviderRepo(d.DBWrapper)
	capabilityRepo := kbdata.NewModelCapabilityRepo(d.DBWrapper)
	modelRepo := kbdata.NewAIModelRepo(d.DBWrapper, capabilityRepo)
	syncLogRepo := kbdata.NewModelSyncLogRepo(d.DBWrapper)

	// Create use case
	syncUseCase := biz.NewModelSyncUseCase(providerRepo, modelRepo, capabilityRepo, syncLogRepo)

	// Get all enabled providers
	ctx := context.Background()
	providers, err := providerRepo.ListAll(ctx)
	if err != nil {
		log.Fatalf("❌ 获取 AI 服务商列表失败: %v", err)
	}

	fmt.Printf("📋 发现 %d 个启用的 AI 服务商\n\n", len(providers))

	// Sync each provider
	totalNew := 0
	totalDeprecated := 0
	totalUpdated := 0

	for _, provider := range providers {
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("🔄 同步服务商: %s (%s)\n", provider.ProviderName, provider.ProviderType)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

		if provider.APIKey == "" {
			fmt.Printf("⚠️  跳过: 未配置 API Key\n\n")
			continue
		}

		startTime := time.Now()

		req := &biz.ModelSyncRequest{
			ProviderID: provider.ID,
			SyncedBy:   "cli-tool",
			SyncType:   "manual",
		}

		result, err := syncUseCase.SyncProviderModels(ctx, req)
		if err != nil {
			fmt.Printf("❌ 同步失败: %v\n\n", err)
			continue
		}

		duration := time.Since(startTime)

		// Print results
		fmt.Printf("✅ 同步成功 (耗时: %.2fs)\n", duration.Seconds())
		fmt.Printf("\n📊 同步统计:\n")
		fmt.Printf("   • 新增模型: %d\n", len(result.NewModels))
		fmt.Printf("   • 弃用模型: %d\n", len(result.DeprecatedModels))
		fmt.Printf("   • 更新模型: %d\n", len(result.UpdatedModels))

		// Show sample new models
		if len(result.NewModels) > 0 {
			fmt.Printf("\n📝 新增模型示例 (前10个):\n")
			for i, model := range result.NewModels {
				if i >= 10 {
					fmt.Printf("   ... 还有 %d 个模型\n", len(result.NewModels)-10)
					break
				}
				capabilities := ""
				for j, cap := range model.Capabilities {
					if j > 0 {
						capabilities += ", "
					}
					capabilities += string(cap.CapabilityType)
					if cap.CapabilityType == biz.CapabilityTypeEmbedding && cap.EmbeddingDimensions != nil {
						capabilities += fmt.Sprintf("(%d维)", *cap.EmbeddingDimensions)
					}
				}
				fmt.Printf("   %d. %s\n", i+1, model.ModelName)
				if capabilities != "" {
					fmt.Printf("      能力: %s\n", capabilities)
				}
			}
		}

		// Show deprecated models
		if len(result.DeprecatedModels) > 0 {
			fmt.Printf("\n⚠️  弃用模型 (前5个):\n")
			for i, model := range result.DeprecatedModels {
				if i >= 5 {
					fmt.Printf("   ... 还有 %d 个模型\n", len(result.DeprecatedModels)-5)
					break
				}
				fmt.Printf("   %d. %s\n", i+1, model.ModelName)
			}
		}

		totalNew += len(result.NewModels)
		totalDeprecated += len(result.DeprecatedModels)
		totalUpdated += len(result.UpdatedModels)

		fmt.Println()
	}

	// Final summary
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✨ 总体统计\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("   • 总新增: %d\n", totalNew)
	fmt.Printf("   • 总弃用: %d\n", totalDeprecated)
	fmt.Printf("   • 总更新: %d\n", totalUpdated)
	fmt.Printf("   • 完成时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println()
	fmt.Println("✅ 所有服务商同步完成！")
}
