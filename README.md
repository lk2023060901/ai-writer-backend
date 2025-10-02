# AI Writer Backend

A DDD-based monolithic backend service built with Go, Gin, PostgreSQL, Redis, MinIO, and Milvus.

## Architecture

This project follows Domain-Driven Design (DDD) principles with a clean architecture:

```
├── cmd/
│   └── server/          # Application entry point
├── internal/
│   ├── conf/            # Configuration management
│   ├── data/            # Data layer initialization
│   ├── server/          # HTTP server setup
│   └── user/            # User domain (example)
│       ├── biz/         # Business logic layer
│       ├── data/        # Data access layer
│       └── service/     # Service layer (HTTP handlers)
├── config.yaml          # Configuration file
└── docker-compose.yaml  # Infrastructure dependencies
```

### Layers

- **Biz (Business Logic)**: Contains domain models and use cases
- **Data**: Implements repository interfaces and database operations
- **Service**: HTTP handlers and API layer

## Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Make (optional)

## Quick Start

### 1. Start Infrastructure Dependencies

```bash
# Start PostgreSQL, Redis, MinIO, and Milvus
make docker-up

# Or use docker-compose directly
docker-compose up -d

# Check services status
docker-compose ps
```

### 2. Configure Application

Copy the example config and adjust if needed:

```bash
cp .env.example .env
```

The default [config.yaml](config.yaml) works with the Docker Compose setup.

### 3. Install Dependencies

```bash
make deps
```

### 4. Run the Application

```bash
# Run directly
make run

# Or build and run
make build
./bin/server -config=config.yaml
```

The server will start on `http://localhost:8080`

## API Endpoints

### Health Check
```bash
curl http://localhost:8080/health
```

### Authentication

#### Register
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test User",
    "email": "test@example.com",
    "password": "SecurePassword123"
  }'
```

**Response:**
```json
{
  "user_id": 1,
  "email": "test@example.com",
  "message": "Registration successful. Please verify your email."
}
```

#### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "SecurePassword123"
  }'
```

**Response:**
```json
{
  "require_2fa": false,
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "36dc50258b45dc6a4f9bcba22e062b83...",
    "expires_in": 900
  }
}
```

#### Refresh Token
```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "36dc50258b45dc6a4f9bcba22e062b83..."
  }'
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "36dc50258b45dc6a4f9bcba22e062b83...",
  "expires_in": 900
}
```

#### Enable 2FA (Protected)
```bash
curl -X POST http://localhost:8080/api/v1/auth/2fa/enable \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

**Response:**
```json
{
  "secret": "PPDYJHBIXMF4IEZGOW7EFJPFFZ4OBPPG",
  "qr_code_url": "/auth/2fa/qrcode",
  "backup_codes": [
    "a7eb-6ef9-11eb-c73b",
    "e315-a3f2-00ce-c2b8",
    "..."
  ]
}
```

#### Get QR Code (Protected)
```bash
curl -X GET http://localhost:8080/api/v1/auth/2fa/qrcode \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

Returns a PNG image of the QR code to scan with authenticator app.

#### Confirm 2FA (Protected)
```bash
curl -X POST http://localhost:8080/api/v1/auth/2fa/confirm \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "code": "123456"
  }'
```

#### Verify 2FA (During Login)
```bash
curl -X POST http://localhost:8080/api/v1/auth/2fa/verify \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "code": "123456"
  }'
```

#### Disable 2FA (Protected)
```bash
curl -X POST http://localhost:8080/api/v1/auth/2fa/disable \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "password": "SecurePassword123"
  }'
```

### User Management (Protected)

All user management endpoints require authentication. Include the access token in the Authorization header:
```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Create User**
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com"
  }'
```

**Get User**
```bash
curl http://localhost:8080/api/v1/users/1 \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

**List Users**
```bash
curl "http://localhost:8080/api/v1/users?page=1&page_size=10" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

**Update User**
```bash
curl -X PUT http://localhost:8080/api/v1/users/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "name": "John Smith",
    "email": "john.smith@example.com"
  }'
```

**Delete User**
```bash
curl -X DELETE http://localhost:8080/api/v1/users/1 \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## Infrastructure Services

### PostgreSQL
- **URL**: `localhost:5432`
- **User**: `postgres`
- **Password**: `postgres`
- **Database**: `aiwriter`

### Redis
- **URL**: `localhost:6379`
- **Password**: (empty)

### MinIO
- **API**: `localhost:9000`
- **Console**: `http://localhost:9001`
- **User**: `minioadmin`
- **Password**: `minioadmin`

### Milvus
- **URL**: `localhost:19530`
- **Management**: `http://localhost:9091`

## Development

### Run Tests
```bash
make test
```

### View Logs
```bash
make docker-logs
```

### Stop Services
```bash
make docker-down
```

### Clean Build Artifacts
```bash
make clean
```

## Project Structure Details

### Adding a New Domain

To add a new domain (e.g., "article"):

1. Create domain structure:
```
internal/article/
├── biz/
│   └── article.go      # Domain model and use cases
├── data/
│   └── article.go      # Repository implementation
└── service/
    └── article.go      # HTTP handlers
```

2. Define domain model in `biz/article.go`:
```go
type Article struct {
    ID      int64
    Title   string
    Content string
}

type ArticleRepo interface {
    Create(ctx context.Context, article *Article) error
    // ... other methods
}

type ArticleUseCase struct {
    repo ArticleRepo
}
```

3. Implement repository in `data/article.go`

4. Create HTTP handlers in `service/article.go`

5. Wire dependencies in [cmd/server/main.go](cmd/server/main.go)

## Configuration

Configuration is managed through [config.yaml](config.yaml). You can override values using environment variables:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "localhost"
  port: 5432
  # ... other settings
```

## License

MIT License - see [LICENSE](LICENSE) file for details.