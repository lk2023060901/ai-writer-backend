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
	fmt.Println("=== AI 模型同步工具 ===\n")

	// 1. 连接数据库
	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=aiwriter sslmode=disable"
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// 使用 database wrapper
	db := &database.DB{}
	// 使用反射或直接访问内部字段
	db.WithContext(context.Background()).GetDB() // 初始化

	// 直接创建repositories，传入 gormDB
	type dbWrapper struct{ db *gorm.DB }
	func (d *dbWrapper) WithContext(ctx context.Context) *database.DB { return &database.DB{} }
	func (d *dbWrapper) GetDB() *gorm.DB { return d.db }
	
	wrapper := &dbWrapper{db: gormDB}
	
	// 实际上我们可以直接用匿名结构体实现接口
	dbImpl := struct{
		*gorm.DB
	}{gormDB}

	// 简化：直接查询
	fmt.Println("测试数据库连接...")
	
	var count int64
	gormDB.Table("ai_providers").Count(&count)
	fmt.Printf("✅ 数据库连接成功，找到 %d 个 AI Provider\n\n", count)

	// 不使用脚本，改为输出 SQL 让用户手动执行
	fmt.Println("由于权限问题，请您手动配置 API Keys:")
	fmt.Println("")
	fmt.Println("-- 配置硅基流动 API Key")
	fmt.Println("UPDATE ai_providers SET api_key = 'YOUR_SILICONFLOW_KEY' WHERE provider_type = 'siliconflow';")
	fmt.Println("")
	fmt.Println("-- 配置 Anthropic API Key")
	fmt.Println("UPDATE ai_providers SET api_key = 'YOUR_ANTHROPIC_KEY' WHERE provider_type = 'anthropic';")
	fmt.Println("")
	fmt.Println("-- 配置智谱 AI API Key")
	fmt.Println("UPDATE ai_providers SET api_key = 'YOUR_ZHIPU_KEY' WHERE provider_type = 'zhipu';")
	fmt.Println("")
	fmt.Println("配置完成后，请使用 HTTP API 触发同步:")
	fmt.Println("POST /api/v1/ai-providers/{provider_id}/models/sync")
}
