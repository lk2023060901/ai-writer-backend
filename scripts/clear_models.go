package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 连接数据库
	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=aiwriter sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 查看当前 AI Providers
	fmt.Println("=== Current AI Providers ===")
	var providers []struct {
		ID           string
		Name         string
		ProviderType string
	}
	db.Table("ai_providers").Select("id, name, provider_type").Order("name").Find(&providers)
	for _, p := range providers {
		fmt.Printf("ID: %s, Name: %s, Type: %s\n", p.ID, p.Name, p.ProviderType)
	}

	// 查看当前数据量
	var modelCount, capCount, syncLogCount int64
	db.Table("ai_models").Count(&modelCount)
	db.Table("ai_model_capabilities").Count(&capCount)
	db.Table("ai_model_sync_logs").Count(&syncLogCount)

	fmt.Printf("\n=== Before Deletion ===\n")
	fmt.Printf("AI Models: %d\n", modelCount)
	fmt.Printf("Capabilities: %d\n", capCount)
	fmt.Printf("Sync Logs: %d\n", syncLogCount)

	// 清空数据
	fmt.Println("\n=== Deleting data ===")

	if err := db.Exec("DELETE FROM ai_model_capabilities").Error; err != nil {
		log.Fatal("Failed to delete capabilities:", err)
	}
	fmt.Println("✅ Deleted ai_model_capabilities")

	if err := db.Exec("DELETE FROM ai_model_sync_logs").Error; err != nil {
		log.Fatal("Failed to delete sync logs:", err)
	}
	fmt.Println("✅ Deleted ai_model_sync_logs")

	if err := db.Exec("DELETE FROM ai_models").Error; err != nil {
		log.Fatal("Failed to delete models:", err)
	}
	fmt.Println("✅ Deleted ai_models")

	// 验证清空
	db.Table("ai_models").Count(&modelCount)
	db.Table("ai_model_capabilities").Count(&capCount)
	db.Table("ai_model_sync_logs").Count(&syncLogCount)

	fmt.Printf("\n=== After Deletion ===\n")
	fmt.Printf("AI Models: %d\n", modelCount)
	fmt.Printf("Capabilities: %d\n", capCount)
	fmt.Printf("Sync Logs: %d\n", syncLogCount)

	fmt.Println("\n✅ All model data cleared successfully!")
}
