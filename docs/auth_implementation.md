# 认证模块完整实现文档

## 概述

本文档描述了 AI Writer Backend 项目的完整认证系统实现，包括 JWT 认证、双因子认证（2FA）、备用恢复码和基于 Redis 的限流。

## 架构概览

```
┌─────────────────────────────────────────────────────────┐
│                   API Layer (Gin)                       │
│  ┌────────────┬──────────────┬────────────────────┐    │
│  │  Register  │    Login     │  2FA Enable/Verify │    │
│  └────────────┴──────────────┴────────────────────┘    │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                Middleware Layer                          │
│  ┌──────────┬─────────────┬──────────────────────┐     │
│  │ JWT Auth │ Rate Limiter│       CORS           │     │
│  └──────────┴─────────────┴──────────────────────┘     │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│              Business Logic Layer                        │
│  ┌──────────────┬────────────────┬──────────────┐      │
│  │ AuthUseCase  │  TOTPManager   │ JWTManager   │      │
│  └──────────────┴────────────────┴──────────────┘      │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                  Data Layer                              │
│  ┌──────────────────────────────────────────────┐       │
│  │  AuthUserRepo (internal/pkg/database)        │       │
│  └──────────────────────────────────────────────┘       │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│             PostgreSQL + Redis                           │
└──────────────────────────────────────────────────────────┘
```

## 核心功能

### 1. JWT 认证

**文件**: [internal/auth/jwt.go](../internal/auth/jwt.go)

**功能**:
- ✅ Access Token 生成（15 分钟有效期）
- ✅ Refresh Token 生成（14 天有效期）
- ✅ Token 验证
- ✅ 从 Authorization Header 提取 Token

**使用示例**:
```go
jwtManager := auth.NewJWTManager("your-secret-key")

// 生成 Access Token
accessToken, _ := jwtManager.GenerateAccessToken(userID, email)

// 生成 Refresh Token
refreshToken, _ := jwtManager.GenerateRefreshToken()

// 验证 Token
claims, _ := jwtManager.VerifyAccessToken(accessToken)
```

### 2. TOTP 双因子认证

**文件**: [internal/auth/totp.go](../internal/auth/totp.go)

**功能**:
- ✅ 生成 TOTP 密钥（base32 编码）
- ✅ 生成二维码（PNG 格式，服务端生成）
- ✅ 验证 6 位验证码（30 秒刷新）
- ✅ 自定义时间窗口验证

**使用示例**:
```go
totpManager := auth.NewTOTPManager("AI Writer")

// 生成密钥和二维码
secret, otpURL, _ := totpManager.GenerateSecret("user@example.com")
qrCode, _ := totpManager.GenerateQRCode(otpURL, 256)

// 验证验证码
valid := totpManager.ValidateCode(secret, "123456")
```

### 3. 备用恢复码

**文件**: [internal/auth/backup_codes.go](../internal/auth/backup_codes.go)

**功能**:
- ✅ 生成 16 位十六进制恢复码（格式：`a3f2-9d7c-4e1b-8a6f`）
- ✅ bcrypt 哈希存储（cost=12）
- ✅ 一次性使用 + 审计日志
- ✅ 统计剩余数量

**安全特性**:
- 64 bits 熵（符合 NIST 标准）
- GitHub 同款方案
- 支持大小写、空格、分隔符容错

**使用示例**:
```go
// 生成 8 个恢复码
plainCodes, backupCodes, _ := auth.GenerateBackupCodes(8)

// 验证恢复码
index, valid, _ := auth.VerifyBackupCode(backupCodes, userInput)
if valid {
    auth.MarkBackupCodeAsUsed(backupCodes, index, &userIP)
}
```

### 4. 基于 Redis 的限流

**文件**: [internal/auth/middleware/rate_limiter.go](../internal/auth/middleware/rate_limiter.go)

**功能**:
- ✅ 滑动窗口算法（Lua 脚本原子操作）
- ✅ 三种限流策略：IP / 用户 ID / 端点
- ✅ 自动降级（Redis 故障时允许请求通过）
- ✅ 标准 HTTP 响应头

**预设限流器**:
```go
// 登录限流：5 次/5 分钟（基于 IP）
LoginRateLimiter(redisClient, log)

// 注册限流：3 次/1 小时（基于 IP）
RegisterRateLimiter(redisClient, log)

// API 限流：100 次/1 分钟（基于用户 ID）
APIRateLimiter(redisClient, log)
```

## API 端点

### 公开端点

| 方法 | 路径 | 描述 | 限流 |
|------|------|------|------|
| POST | `/auth/register` | 用户注册 | 3 次/小时 |
| POST | `/auth/login` | 用户登录 | 5 次/5 分钟 |
| POST | `/auth/2fa/verify` | 验证 2FA 代码 | - |
| POST | `/auth/refresh` | 刷新 Access Token | - |

### 需要认证的端点

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/auth/2fa/enable` | 启用双因子认证 |
| GET | `/auth/2fa/qrcode` | 获取 2FA 二维码 |
| POST | `/auth/2fa/confirm` | 确认启用 2FA |
| POST | `/auth/2fa/disable` | 禁用双因子认证 |

## 数据库表结构

**用户表**: [migrations/00001_create_users_table.sql](../migrations/00001_create_users_table.sql)

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,

    -- 基础信息
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,

    -- 认证信息
    password_hash VARCHAR(255) NOT NULL,

    -- JWT Refresh Token
    refresh_token VARCHAR(512),
    refresh_token_expires_at TIMESTAMPTZ,

    -- 双因子认证
    two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    two_factor_secret VARCHAR(32),
    two_factor_backup_codes JSONB,

    -- 登录追踪
    last_login_at TIMESTAMPTZ,
    last_login_ip INET,
    failed_login_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,

    -- 邮箱验证/密码重置
    email_verification_token VARCHAR(64),
    email_verification_expires_at TIMESTAMPTZ,
    password_reset_token VARCHAR(64),
    password_reset_expires_at TIMESTAMPTZ,

    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);
```

**JSONB 备用恢复码格式**:
```json
[
  {
    "hash": "$2a$12$abc...xyz",
    "used": false,
    "used_at": null,
    "used_ip": null
  },
  {
    "hash": "$2a$12$def...uvw",
    "used": true,
    "used_at": "2025-10-02T10:30:00Z",
    "used_ip": "192.168.1.100"
  }
]
```

## 认证流程

### 1. 注册流程

```
1. POST /auth/register
   ├─ 验证邮箱格式（Go 代码层）
   ├─ 检查邮箱是否已存在
   ├─ bcrypt 哈希密码（cost=12）
   ├─ 生成邮箱验证 token（24 小时有效）
   └─ 返回用户 ID

2. （后续）发送验证邮件

3. （后续）GET /auth/verify-email?token=xxx
   └─ 标记 email_verified = true
```

### 2. 登录流程（无 2FA）

```
1. POST /auth/login
   ├─ 检查账户是否锁定
   ├─ 验证密码
   ├─ 重置失败次数
   ├─ 生成 Access Token（15 分钟）
   ├─ 生成 Refresh Token（14 天）
   ├─ 更新登录信息（时间、IP）
   └─ 返回 tokens
```

### 3. 登录流程（有 2FA）

```
1. POST /auth/login
   ├─ 验证密码
   └─ 返回 { "require_2fa": true, "user_id": 123 }

2. POST /auth/2fa/verify
   ├─ 验证 TOTP 验证码（优先）
   ├─ 或验证备用恢复码
   ├─ 标记恢复码为已使用（如果使用）
   ├─ 生成 tokens
   └─ 返回 tokens
```

### 4. 启用 2FA 流程

```
1. POST /auth/2fa/enable (需要 JWT)
   ├─ 生成 TOTP 密钥
   ├─ 生成二维码（PNG）
   ├─ 生成 8 个备用恢复码
   ├─ 保存到数据库（two_factor_enabled = false）
   └─ 返回 { secret, qr_code_url, backup_codes }

2. GET /auth/2fa/qrcode (需要 JWT)
   └─ 返回二维码图片（PNG）

3. POST /auth/2fa/confirm (需要 JWT)
   ├─ 验证用户输入的验证码
   ├─ 设置 two_factor_enabled = true
   └─ 返回成功
```

### 5. Token 刷新流程

```
POST /auth/refresh
├─ 验证 refresh_token
├─ 检查是否过期
├─ 生成新的 Access Token
└─ 返回 { access_token, refresh_token (复用), expires_in }
```

## 安全特性

### 1. 密码安全
- ✅ bcrypt 哈希（cost=12）
- ✅ 最小长度 8 字符
- ✅ 最大长度 72 字符（bcrypt 限制）

### 2. 账户锁定
- ✅ 5 次登录失败 → 锁定 15 分钟
- ✅ 成功登录后重置失败次数
- ✅ 锁定期间拒绝所有登录尝试

### 3. Token 安全
- ✅ Access Token 短期有效（15 分钟）
- ✅ Refresh Token 长期有效（14 天）
- ✅ Refresh Token 轮换（可选）
- ✅ Token 存储在数据库（可撤销）

### 4. 限流保护
- ✅ 登录：5 次/5 分钟
- ✅ 注册：3 次/1 小时
- ✅ API：100 次/1 分钟
- ✅ 故障降级（Redis 不可用时允许请求）

### 5. 2FA 安全
- ✅ TOTP 标准（RFC 6238）
- ✅ 6 位数字码，30 秒刷新
- ✅ 备用恢复码：64 bits 熵
- ✅ 一次性使用 + 审计日志

## 使用示例

### 1. 初始化认证模块

```go
import (
    "github.com/lk2023060901/ai-writer-backend/internal/auth/biz"
    "github.com/lk2023060901/ai-writer-backend/internal/auth/data"
    "github.com/lk2023060901/ai-writer-backend/internal/auth/service"
    "github.com/lk2023060901/ai-writer-backend/internal/auth/middleware"
    "github.com/lk2023060901/ai-writer-backend/internal/pkg/database"
    "github.com/lk2023060901/ai-writer-backend/internal/pkg/redis"
    "github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
)

// 初始化依赖
db, _ := database.New(dbConfig, log)
redisClient, _ := redis.New(redisConfig, log)

// 创建仓库
userRepo := data.NewAuthUserRepo(db)

// 创建业务逻辑
authUC := biz.NewAuthUseCase(userRepo, "jwt-secret-key", "AI Writer")

// 创建服务
authService := service.NewAuthService(authUC, log)

// 注册路由
router := gin.New()
api := router.Group("/api")
authService.RegisterRoutes(api)
```

### 2. 添加认证中间件

```go
// 全局中间件
router.Use(middleware.CORS())

// 公开 API
publicAPI := router.Group("/api")
{
    // 注册限流
    publicAPI.POST("/auth/register",
        middleware.RegisterRateLimiter(redisClient, log),
        authService.Register)

    // 登录限流
    publicAPI.POST("/auth/login",
        middleware.LoginRateLimiter(redisClient, log),
        authService.Login)
}

// 需要认证的 API
protectedAPI := router.Group("/api")
protectedAPI.Use(middleware.JWTAuth("jwt-secret-key", log))
protectedAPI.Use(middleware.APIRateLimiter(redisClient, log))
{
    protectedAPI.POST("/auth/2fa/enable", authService.Enable2FA)
    protectedAPI.POST("/auth/2fa/confirm", authService.Confirm2FA)
    protectedAPI.POST("/auth/2fa/disable", authService.Disable2FA)
}
```

### 3. 前端集成示例

```javascript
// 注册
const register = async (name, email, password) => {
  const response = await fetch('/api/auth/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, email, password })
  });
  return response.json();
};

// 登录
const login = async (email, password) => {
  const response = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password })
  });
  const data = await response.json();

  if (data.require_2fa) {
    // 需要 2FA，跳转到验证页面
    return { require2FA: true, userID: data.user_id };
  }

  // 保存 tokens
  localStorage.setItem('access_token', data.tokens.access_token);
  localStorage.setItem('refresh_token', data.tokens.refresh_token);
  return { require2FA: false };
};

// 验证 2FA
const verify2FA = async (userID, code) => {
  const response = await fetch('/api/auth/2fa/verify', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ user_id: userID, code })
  });
  const data = await response.json();

  localStorage.setItem('access_token', data.tokens.access_token);
  localStorage.setItem('refresh_token', data.tokens.refresh_token);
};

// 刷新 Token
const refreshToken = async () => {
  const refreshToken = localStorage.getItem('refresh_token');
  const response = await fetch('/api/auth/refresh', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken })
  });
  const data = await response.json();

  localStorage.setItem('access_token', data.access_token);
};

// 带认证的 API 请求
const apiRequest = async (url, options = {}) => {
  const token = localStorage.getItem('access_token');
  const response = await fetch(url, {
    ...options,
    headers: {
      ...options.headers,
      'Authorization': `Bearer ${token}`
    }
  });

  if (response.status === 401) {
    // Token 过期，刷新后重试
    await refreshToken();
    return apiRequest(url, options);
  }

  return response.json();
};
```

## 测试

### 单元测试
```bash
# 测试备用恢复码
go test -v ./internal/auth/ -run TestBackupCode

# 测试覆盖率
go test -cover ./internal/auth/
```

### 基准测试
```bash
# 运行基准测试
go test -bench=. -benchmem ./internal/auth/
```

### 集成测试
```bash
# 需要先启动数据库
make docker-up

# 运行集成测试
go test -v -tags=integration ./internal/auth/...
```

## 性能监控

### 慢查询检测
```bash
# 检查用户表慢查询
make db-check-slow-queries

# 运行性能基准测试
make db-benchmark
```

## 下一步

- [ ] 实现邮箱验证
- [ ] 实现密码重置
- [ ] 添加 OAuth2 登录（Google/GitHub）
- [ ] 实现 Remember Me 功能
- [ ] 添加设备管理（查看/撤销登录设备）
- [ ] 实现日志审计
- [ ] 添加 WebAuthn 支持

## 参考资料

- [RFC 6238 - TOTP](https://datatracker.ietf.org/doc/html/rfc6238)
- [NIST SP 800-63B - Digital Identity Guidelines](https://pages.nist.gov/800-63-3/sp800-63b.html)
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)
