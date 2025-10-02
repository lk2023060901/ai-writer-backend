package models

import (
	"context"
	"fmt"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
	"go.uber.org/zap"
)

// AutoMigrate 自动迁移所有知识库相关表
func AutoMigrate(ctx context.Context, db *database.DB) error {
	// 按依赖顺序迁移表
	models := []interface{}{
		&KnowledgeBase{},
		&Document{},
		&Chunk{},
	}

	for _, model := range models {
		if err := db.WithContext(ctx).AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	// 创建额外的索引
	if err := createIndexes(ctx, db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// createIndexes 创建额外的索引
func createIndexes(ctx context.Context, db *database.DB) error {
	// 为知识库表创建复合索引
	if err := db.WithContext(ctx).Exec(`
		CREATE INDEX IF NOT EXISTS idx_kb_user_created
		ON knowledge_bases(user_id, created_at DESC)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	// 为文档表创建复合索引
	if err := db.WithContext(ctx).Exec(`
		CREATE INDEX IF NOT EXISTS idx_doc_kb_status
		ON documents(knowledge_base_id, status)
	`).Error; err != nil {
		return err
	}

	if err := db.WithContext(ctx).Exec(`
		CREATE INDEX IF NOT EXISTS idx_doc_kb_created
		ON documents(knowledge_base_id, created_at DESC)
	`).Error; err != nil {
		return err
	}

	// 为分块表创建复合索引
	if err := db.WithContext(ctx).Exec(`
		CREATE INDEX IF NOT EXISTS idx_chunk_doc_index
		ON chunks(document_id, chunk_index)
	`).Error; err != nil {
		return err
	}

	return nil
}

// DropTables 删除所有知识库相关表（危险操作，仅用于测试）
func DropTables(ctx context.Context, db *database.DB) error {
	// 按相反顺序删除表
	models := []interface{}{
		&Chunk{},
		&Document{},
		&KnowledgeBase{},
	}

	for _, model := range models {
		if err := db.WithContext(ctx).Migrator().DropTable(model); err != nil {
			return fmt.Errorf("failed to drop table %T: %w", model, err)
		}
	}

	return nil
}

// MigrateWithLog 带日志的迁移
func MigrateWithLog(ctx context.Context, db *database.DB, logger *zap.Logger) error {
	logger.Info("starting knowledge base schema migration")

	if err := AutoMigrate(ctx, db); err != nil {
		logger.Error("schema migration failed", zap.Error(err))
		return err
	}

	logger.Info("knowledge base schema migration completed successfully")
	return nil
}
