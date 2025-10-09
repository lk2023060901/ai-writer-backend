.PHONY: run dev build test test-unit test-integration deps docker-up docker-down docker-logs clean migrate-status migrate-up migrate-down migrate-reset migrate-create

# 开发模式 - 使用优雅退出脚本（推荐）
dev:
	@./scripts/dev.sh config.yaml

# 直接运行 - 使用 trap 捕获信号
run:
	@echo "Starting server... (Press Ctrl+C to stop)"
	@trap 'echo "Shutting down..."; pkill -P $$; exit' INT TERM; \
	go run cmd/server/main.go -config=config.yaml

build:
	@mkdir -p bin
	go build -o bin/server cmd/server/main.go

# Run all tests
test:
	go test -v ./...

# Run unit tests only
test-unit:
	go test -v -short ./...

# Run integration tests with docker
test-integration: docker-up
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Running integration tests..."
	go test -v -tags=integration ./internal/pkg/database/
	@$(MAKE) docker-down

# Run database integration tests only
test-db-integration:
	go test -v -tags=integration ./internal/pkg/database/

deps:
	go mod download
	go mod tidy

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Wait for postgres to be ready
docker-wait-postgres:
	@echo "Waiting for PostgreSQL to be ready..."
	@until docker exec aiwriter-postgres pg_isready -U postgres > /dev/null 2>&1; do \
		echo "Waiting..."; \
		sleep 1; \
	done
	@echo "PostgreSQL is ready!"

clean:
	rm -rf bin/
	go clean

# 数据库迁移配置
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
DB_PASS ?= postgres
DB_NAME ?= ai_writer
DB_STRING = "host=$(DB_HOST) port=$(DB_PORT) user=$(DB_USER) password=$(DB_PASS) dbname=$(DB_NAME) sslmode=disable"

# 查看迁移状态
migrate-status:
	goose -dir migrations postgres $(DB_STRING) status

# 应用所有迁移
migrate-up:
	goose -dir migrations postgres $(DB_STRING) up

# 应用下一个迁移
migrate-up-one:
	goose -dir migrations postgres $(DB_STRING) up-by-one

# 回滚一次迁移
migrate-down:
	goose -dir migrations postgres $(DB_STRING) down

# 重置所有迁移（危险！）
migrate-reset:
	@echo "⚠️  This will drop all tables! Press Ctrl+C to cancel..."
	@sleep 5
	goose -dir migrations postgres $(DB_STRING) reset

# 创建新迁移文件
migrate-create:
	@read -p "Migration name: " name; \
	goose -dir migrations create $$name sql

# 验证迁移文件
migrate-validate:
	goose -dir migrations validate

# 检查慢查询（需要先启用 pg_stat_statements）
db-check-slow-queries:
	@echo "检测 users 表慢查询..."
	psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f scripts/check_slow_queries.sql

# 性能基准测试
db-benchmark:
	@echo "运行 users 表性能基准测试..."
	psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f scripts/benchmark_users_table.sql

# 启用 pg_stat_statements 扩展
db-enable-stats:
	@echo "启用 pg_stat_statements 扩展..."
	psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -c "CREATE EXTENSION IF NOT EXISTS pg_stat_statements;"

# 重置统计数据
db-reset-stats:
	@echo "重置 pg_stat_statements 统计数据..."
	psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -c "SELECT pg_stat_statements_reset();"

# 更新 users 表统计信息
db-analyze-users:
	@echo "更新 users 表统计信息..."
	psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -c "ANALYZE users;"
