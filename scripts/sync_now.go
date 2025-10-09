package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/biz"
	"github.com/lk2023060901/ai-writer-backend/internal/knowledge/data"
	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== AI 模型同步执行 ===\n")

	// 连接数据库
	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=aiwriter sslmode=disable"
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// 创建 DB wrapper
	db := &database.DB{}
	sqlDB, _ := gormDB.DB()
	db.Init(sqlDB)

	ctx := context.Background()

	// 创建 repositories
	providerRepo := data.NewAIProviderRepo(db)
	capabilityRepo := data.NewModelCapabilityRepo(db)
	modelRepo := data.NewAIModelRepo(db, capabilityRepo)
	syncLogRepo := data.NewModelSyncLogRepo(db)

	// 创建 use case
	syncUseCase := biz.NewModelSyncUseCase(
		providerRepo,
		modelRepo,
		capabilityRepo,
		syncLogRepo,
	)

	// 获取所有 providers
	providers, err := providerRepo.ListAll(ctx)
	if err != nil {
		log.Fatal("获取 Providers 失败:", err)
	}

	fmt.Printf("找到 %d 个 AI Provider\n\n", len(providers))

	// 对每个 provider 执行同步
	for _, provider := range providers {
		// 跳过没有 API Key 的
		if provider.APIKey == "" {
			fmt.Printf("⏭️  跳过 %s (没有 API Key)\n\n", provider.ProviderName)
			continue
		}

		fmt.Printf("========================================\n")
		fmt.Printf("同步: %s (%s)\n", provider.ProviderName, provider.ProviderType)
		fmt.Printf("========================================\n")

		// 执行同步
		req := &biz.ModelSyncRequest{
			ProviderID: provider.ID,
			SyncedBy:   "admin-script",
			SyncType:   "manual",
		}

		result, err := syncUseCase.SyncProviderModels(ctx, req)
		if err != nil {
			fmt.Printf("❌ 同步失败: %v\n\n", err)
			continue
		}

		// 显示结果
		fmt.Printf("✅ 同步完成:\n")
		fmt.Printf("  新增: %d 个模型\n", len(result.NewModels))
		fmt.Printf("  弃用: %d 个模型\n", len(result.DeprecatedModels))
		fmt.Printf("  更新: %d 个模型\n", len(result.UpdatedModels))
		fmt.Printf("  错误: %d 个\n", len(result.Errors))

		if len(result.NewModels) > 0 {
			fmt.Println("\n前5个新增模型:")
			for i, m := range result.NewModels {
				if i >= 5 {
					fmt.Printf("  ... 还有 %d 个\n", len(result.NewModels)-5)
					break
				}
				caps := []string{}
				for _, c := range m.Capabilities {
					caps = append(caps, c.CapabilityType)
				}
				fmt.Printf("  - %s [%v]\n", m.ModelName, caps)
			}
		}

		if len(result.Errors) > 0 {
			fmt.Println("\n错误:")
			for i, e := range result.Errors {
				if i >= 3 {
					break
				}
				fmt.Printf("  - %v\n", e)
			}
		}

		fmt.Println()
	}

	// 统计
	fmt.Println("========================================")
	fmt.Println("同步统计")
	fmt.Println("========================================")

	allModels, _ := modelRepo.ListAll(ctx)
	fmt.Printf("总模型数: %d\n\n", len(allModels))

	embeddingModels, _ := modelRepo.ListByCapabilityType(ctx, biz.CapabilityTypeEmbedding)
	chatModels, _ := modelRepo.ListByCapabilityType(ctx, biz.CapabilityTypeChat)

	fmt.Printf("按能力分类:\n")
	fmt.Printf("  Embedding: %d 个\n", len(embeddingModels))
	fmt.Printf("  Chat: %d 个\n", len(chatModels))

	if len(embeddingModels) > 0 {
		fmt.Println("\n前3个 Embedding 模型:")
		for i, m := range embeddingModels {
			if i >= 3 {
				break
			}
			dim := "?"
			for _, cap := range m.Capabilities {
				if cap.CapabilityType == biz.CapabilityTypeEmbedding && cap.EmbeddingDimensions != nil {
					dim = fmt.Sprintf("%d", *cap.EmbeddingDimensions)
				}
			}
			fmt.Printf("  - %s (维度: %s)\n", m.ModelName, dim)
		}
	}

	fmt.Println("\n✅ 同步完成！")
}
