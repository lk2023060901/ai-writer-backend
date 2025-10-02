.PHONY: run build test test-unit test-integration deps docker-up docker-down docker-logs clean

run:
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
