# 知识库模块实施指南

> **当前状态**：已完成数据库设计和错误定义，待实现业务逻辑层和 HTTP 层
> **最后更新**：2025-10-02
> **会话 Token**：已使用 92K+，建议新对话继续

---

## 一、已完成的工作 ✅

### 1. 数据库迁移文件
**文件**：`migrations/00003_create_knowledge_and_ai_provider_tables.sql`

**包含 4 个表**：
- `ai_provider_configs` - AI 服务商配置（官方+用户）
- `knowledge_bases` - 知识库（官方+用户）
- `documents` - 文档
- `chunks` - 分块

**关键设计**：
- 官方资源通过 `owner_id = '00000000-0000-0000-0000-000000000000'` 识别
- API Key 测试阶段明文存储（生产环境需加密）
- 知识库通过 `ai_provider_config_id` 关联 AI 配置
- 完整的索引和外键约束

### 2. 业务错误定义
**文件**：`internal/knowledge/biz/errors.go`

**包含**：
- `SystemOwnerID` 常量
- AI 配置相关错误
- 知识库相关错误
- 文档相关错误
- 权限相关错误
- Milvus 相关错误

---

## 二、待完成的工作清单 📋

### 阶段 1：数据访问层（Data Layer）

#### 1.1 创建 AI 服务商配置数据层
**文件**：`internal/knowledge/data/ai_provider_config.go`

**需要实现**：
```go
type AIProviderConfigRepo struct {
    db *database.DB
}

// 必需方法
- Create(ctx, config) error
- GetByID(ctx, id, userID) (*biz.AIProviderConfig, error)
- GetUserDefault(ctx, userID) (*biz.AIProviderConfig, error)
- GetOfficialDefault(ctx) (*biz.AIProviderConfig, error)
- List(ctx, userID) ([]*biz.AIProviderConfig, error)
- Update(ctx, config) error
- Delete(ctx, id, ownerID) error
- SetDefault(ctx, id, userID) error
```

**关键点**：
- 使用 `database.DB` 包装器（与 Agent 模块一致）
- `GetByID` 需要验证：官方配置 或 用户自己的配置
- `List` 返回：官方配置 + 用户自己的配置
- `SetDefault` 需要先取消用户的其他默认配置

#### 1.2 创建知识库数据层
**文件**：`internal/knowledge/data/knowledge.go`

**需要实现**：
```go
type KnowledgeBaseRepo struct {
    db *database.DB
}

// 必需方法
- Create(ctx, kb) error
- GetByID(ctx, id, userID) (*biz.KnowledgeBase, error)
- List(ctx, req) ([]*biz.KnowledgeBase, int64, error)
- Update(ctx, kb) error
- Delete(ctx, id, ownerID) error
- IncrementDocumentCount(ctx, id, delta) error
```

**关键点**：
- 支持分页查询
- `GetByID` 验证权限（官方或用户自己的）
- `List` 需要合并官方和用户知识库

#### 1.3 创建官方知识库数据层（可选）
**文件**：`internal/knowledge/data/official_knowledge.go`

**类似智能体模块的设计**：
```go
type OfficialKnowledgeBaseRepo struct {
    db *database.DB
}

- GetByID(ctx, id) (*biz.KnowledgeBase, error)
- List(ctx, req) ([]*biz.KnowledgeBase, int64, error)
```

**说明**：也可以在 `KnowledgeBaseRepo` 中通过 `owner_id` 过滤，不单独创建此文件。

---

### 阶段 2：业务逻辑层（Biz Layer）

#### 2.1 创建 AI 服务商配置业务逻辑
**文件**：`internal/knowledge/biz/ai_provider_config.go`

**核心结构**：
```go
type AIProviderConfig struct {
    ID                  string
    OwnerID             string  // SystemOwnerID = 官方
    ProviderType        string  // openai, anthropic
    ProviderName        string
    APIKey              string
    APIBaseURL          string
    EmbeddingModel      string
    EmbeddingDimensions int
    IsEnabled           bool
    IsDefault           bool
    CreatedAt           time.Time
    UpdatedAt           time.Time
}

func (c *AIProviderConfig) IsOfficial() bool {
    return c.OwnerID == SystemOwnerID
}

type AIProviderConfigUseCase struct {
    repo AIProviderConfigRepo
}

// 必需方法
- CreateAIProviderConfig(ctx, userID, req) (*AIProviderConfig, error)
- GetAIProviderConfig(ctx, id, userID) (*AIProviderConfig, error)
- ListAIProviderConfigs(ctx, userID) ([]*AIProviderConfig, error)
- UpdateAIProviderConfig(ctx, id, userID, req) (*AIProviderConfig, error)
- DeleteAIProviderConfig(ctx, id, userID) error
- SetDefaultAIProviderConfig(ctx, id, userID) error
```

**权限控制逻辑**：
```go
// 更新时检查
if config.IsOfficial() {
    return nil, ErrCannotEditOfficialResource
}
if config.OwnerID != userID {
    return nil, ErrUnauthorized
}

// 删除时检查
if config.IsOfficial() {
    return ErrCannotDeleteOfficialResource
}
// 检查是否有知识库在使用此配置
count := repo.CountByAIProviderConfig(ctx, id)
if count > 0 {
    return ErrConfigInUse
}
```

#### 2.2 创建知识库业务逻辑
**文件**：`internal/knowledge/biz/knowledge.go`

**核心结构**：
```go
type KnowledgeBase struct {
    ID                  string
    OwnerID             string  // SystemOwnerID = 官方
    Name                string
    AIProviderConfigID  string
    ChunkSize           int
    ChunkOverlap        int
    ChunkStrategy       string
    MilvusCollection    string
    DocumentCount       int64
    CreatedAt           time.Time
    UpdatedAt           time.Time
}

func (kb *KnowledgeBase) IsOfficial() bool {
    return kb.OwnerID == SystemOwnerID
}

type KnowledgeBaseUseCase struct {
    kbRepo         KnowledgeBaseRepo
    aiConfigRepo   AIProviderConfigRepo
    // milvusStore    storage.VectorStore  // 阶段 3 再添加
}

// 必需方法
- CreateKnowledgeBase(ctx, userID, req) (*KnowledgeBase, error)
- GetKnowledgeBase(ctx, id, userID) (*KnowledgeBase, error)
- ListKnowledgeBases(ctx, userID, req) ([]*KnowledgeBase, int64, error)
- UpdateKnowledgeBase(ctx, id, userID, req) (*KnowledgeBase, error)
- DeleteKnowledgeBase(ctx, id, userID) error
```

**创建知识库的核心逻辑**：
```go
func (uc *KnowledgeBaseUseCase) CreateKnowledgeBase(ctx, userID, req) {
    // 1. 解析 AI 配置
    var aiConfig *AIProviderConfig
    if req.AIProviderConfigID != "" {
        config := uc.aiConfigRepo.GetByID(ctx, req.AIProviderConfigID, userID)
        // 验证：必须是官方配置 或 用户自己的配置
        if !config.IsOfficial() && config.OwnerID != userID {
            return ErrUnauthorized
        }
        aiConfig = config
    } else {
        // 优先用户默认，否则用官方默认
        aiConfig = uc.aiConfigRepo.GetUserDefault(ctx, userID)
        if aiConfig == nil {
            aiConfig = uc.aiConfigRepo.GetOfficialDefault(ctx)
        }
    }

    // 2. 生成 Milvus Collection 名称
    collectionName := fmt.Sprintf("kb_%s_%s",
        userID[:8], uuid.New().String()[:8])

    // 3. 【阶段 3】在 Milvus 创建 Collection
    // uc.milvusStore.CreateCollection(...)

    // 4. 创建知识库
    kb := &KnowledgeBase{
        OwnerID:            userID,
        Name:               req.Name,
        AIProviderConfigID: aiConfig.ID,
        MilvusCollection:   collectionName,
        // ...
    }
    return uc.kbRepo.Create(ctx, kb)
}
```

---

### 阶段 3：HTTP 服务层（Service Layer）

#### 3.1 创建 HTTP DTO
**文件**：`internal/knowledge/service/types.go`

**AI 服务商配置 DTO**：
```go
// 创建请求
type CreateAIProviderConfigRequest struct {
    ProviderType        string `json:"provider_type" binding:"required"`
    ProviderName        string `json:"provider_name" binding:"required"`
    APIKey              string `json:"api_key" binding:"required"`
    APIBaseURL          string `json:"api_base_url"`
    EmbeddingModel      string `json:"embedding_model" binding:"required"`
    EmbeddingDimensions int    `json:"embedding_dimensions" binding:"required,min=1"`
    IsDefault           bool   `json:"is_default"`
}

// 更新请求
type UpdateAIProviderConfigRequest struct {
    ProviderName *string `json:"provider_name"`
    APIKey       *string `json:"api_key"`
    APIBaseURL   *string `json:"api_base_url"`
}

// 响应
type AIProviderConfigResponse struct {
    ID                  string `json:"id"`
    OwnerID             string `json:"owner_id"`
    ProviderType        string `json:"provider_type"`
    ProviderName        string `json:"provider_name"`
    EmbeddingModel      string `json:"embedding_model"`
    EmbeddingDimensions int    `json:"embedding_dimensions"`
    IsOfficial          bool   `json:"is_official"`
    IsDefault           bool   `json:"is_default"`
    IsEnabled           bool   `json:"is_enabled"`
    CanEdit             bool   `json:"can_edit"`
    CanDelete           bool   `json:"can_delete"`
    CreatedAt           string `json:"created_at"`
}
```

**知识库 DTO**：
```go
// 创建请求
type CreateKnowledgeBaseRequest struct {
    Name               string `json:"name" binding:"required"`
    AIProviderConfigID string `json:"ai_provider_config_id"`
    ChunkSize          int    `json:"chunk_size"`
    ChunkOverlap       int    `json:"chunk_overlap"`
    ChunkStrategy      string `json:"chunk_strategy"`
}

// 更新请求
type UpdateKnowledgeBaseRequest struct {
    Name *string `json:"name"`
}

// 响应
type KnowledgeBaseResponse struct {
    ID                string                      `json:"id"`
    OwnerID           string                      `json:"owner_id"`
    Name              string                      `json:"name"`
    AIProviderConfig  *AIProviderConfigResponse   `json:"ai_provider_config"`
    ChunkSize         int                         `json:"chunk_size"`
    ChunkOverlap      int                         `json:"chunk_overlap"`
    ChunkStrategy     string                      `json:"chunk_strategy"`
    MilvusCollection  string                      `json:"milvus_collection"`
    DocumentCount     int64                       `json:"document_count"`
    IsOfficial        bool                        `json:"is_official"`
    CanEdit           bool                        `json:"can_edit"`
    CanDelete         bool                        `json:"can_delete"`
    CreatedAt         string                      `json:"created_at"`
    UpdatedAt         string                      `json:"updated_at"`
}

// 列表响应
type ListKnowledgeBasesResponse struct {
    Items      []*KnowledgeBaseResponse `json:"items"`
    Pagination *PaginationResponse      `json:"pagination"`
}
```

#### 3.2 创建 AI 服务商配置 HTTP Handler
**文件**：`internal/knowledge/service/ai_provider.go`

**核心结构**：
```go
type AIProviderService struct {
    uc     *biz.AIProviderConfigUseCase
    logger *logger.Logger
}

// HTTP Handlers
- CreateAIProviderConfig(c *gin.Context)
- ListAIProviderConfigs(c *gin.Context)
- GetAIProviderConfig(c *gin.Context)
- UpdateAIProviderConfig(c *gin.Context)
- DeleteAIProviderConfig(c *gin.Context)
- SetDefaultAIProviderConfig(c *gin.Context)
```

**统一响应格式**（与 Agent 模块一致）：
```go
// 成功
response.Success(c, data)       // 200
response.Created(c, data)       // 201

// 错误
response.BadRequest(c, msg)     // 400
response.Unauthorized(c, msg)   // 401
response.Forbidden(c, msg)      // 403
response.NotFound(c, msg)       // 404
response.InternalError(c, msg)  // 500
```

#### 3.3 创建知识库 HTTP Handler
**文件**：`internal/knowledge/service/knowledge.go`

**核心结构**：
```go
type KnowledgeBaseService struct {
    uc     *biz.KnowledgeBaseUseCase
    logger *logger.Logger
}

// HTTP Handlers
- CreateKnowledgeBase(c *gin.Context)
- ListKnowledgeBases(c *gin.Context)
- GetKnowledgeBase(c *gin.Context)
- UpdateKnowledgeBase(c *gin.Context)
- DeleteKnowledgeBase(c *gin.Context)
```

---

### 阶段 4：集成到主服务

#### 4.1 更新依赖注入
**文件**：`cmd/server/main.go`

**添加代码**：
```go
import (
    kbbiz "github.com/.../internal/knowledge/biz"
    kbdata "github.com/.../internal/knowledge/data"
    kbservice "github.com/.../internal/knowledge/service"
)

// 在 main() 中添加：

// Initialize knowledge repositories
aiConfigRepo := kbdata.NewAIProviderConfigRepo(d.DBWrapper)
kbRepo := kbdata.NewKnowledgeBaseRepo(d.DBWrapper)

// Initialize knowledge use cases
aiConfigUseCase := kbbiz.NewAIProviderConfigUseCase(aiConfigRepo)
kbUseCase := kbbiz.NewKnowledgeBaseUseCase(kbRepo, aiConfigRepo)

// Initialize knowledge services
aiConfigService := kbservice.NewAIProviderService(aiConfigUseCase, log)
kbService := kbservice.NewKnowledgeBaseService(kbUseCase, log)

// Update HTTPServer initialization
httpServer := server.NewHTTPServer(
    config, log,
    userService, authService, agentService,
    aiConfigService, kbService,  // 新增
    d.RedisClient,
)
```

#### 4.2 更新路由注册
**文件**：`internal/server/http.go`

**添加字段**：
```go
type HTTPServer struct {
    // ...
    aiConfigService *kbservice.AIProviderService
    kbService       *kbservice.KnowledgeBaseService
}
```

**注册路由**：
```go
// Protected API routes
protectedAPI.Use(middleware.JWTAuth(config.Auth.JWTSecret, log))
{
    // AI Provider Config routes
    aiProviders := protectedAPI.Group("/ai-providers")
    {
        aiProviders.GET("", aiConfigService.ListAIProviderConfigs)
        aiProviders.POST("", aiConfigService.CreateAIProviderConfig)
        aiProviders.GET("/:id", aiConfigService.GetAIProviderConfig)
        aiProviders.PUT("/:id", aiConfigService.UpdateAIProviderConfig)
        aiProviders.DELETE("/:id", aiConfigService.DeleteAIProviderConfig)
        aiProviders.PATCH("/:id/set-default", aiConfigService.SetDefaultAIProviderConfig)
    }

    // Knowledge Base routes
    kbs := protectedAPI.Group("/knowledge-bases")
    {
        kbs.GET("", kbService.ListKnowledgeBases)
        kbs.POST("", kbService.CreateKnowledgeBase)
        kbs.GET("/:id", kbService.GetKnowledgeBase)
        kbs.PUT("/:id", kbService.UpdateKnowledgeBase)
        kbs.DELETE("/:id", kbService.DeleteKnowledgeBase)
    }
}
```

---

### 阶段 5：测试与验证

#### 5.1 运行数据库迁移
```bash
cd /Volumes/work/coding/golang/ai-writer-backend

# 运行迁移
goose -dir migrations postgres \
  "user=postgres password=postgres dbname=aiwriter host=localhost port=5432 sslmode=disable" \
  up

# 验证表创建
docker exec -i $(docker ps -q -f name=postgres) \
  psql -U postgres -d aiwriter -c "\dt"
```

#### 5.2 清空测试数据
```bash
# 清空数据库
docker exec -i $(docker ps -q -f name=postgres) \
  psql -U postgres -d aiwriter \
  -c "TRUNCATE TABLE chunks, documents, knowledge_bases, ai_provider_configs, agents, official_agents, users CASCADE;"

# 清空 Redis
docker exec -i $(docker ps -q -f name=redis) redis-cli FLUSHDB
```

#### 5.3 编译测试
```bash
# 编译检查
go build -o /dev/null ./cmd/server/

# 启动服务
go run cmd/server/main.go -config=config.yaml
```

#### 5.4 API 测试脚本
**文件**：`scripts/test_knowledge_api.sh`

**测试流程**：
1. 注册用户并登录
2. 列出 AI 服务商配置（应该看到官方配置）
3. 创建用户自己的 AI 配置
4. 创建知识库（使用官方配置）
5. 创建知识库（使用用户配置）
6. 列出知识库（官方 + 用户）
7. 更新知识库
8. 删除知识库
9. 尝试编辑官方知识库（应该失败）

---

## 三、关键设计决策 📝

### 1. 官方资源标识
- **方式**：`owner_id = '00000000-0000-0000-0000-000000000000'`
- **一致性**：与 Agent 模块完全一致

### 2. 权限控制
- **官方资源**：所有用户可查看和使用，不可编辑/删除
- **用户资源**：仅所有者可查看、编辑、删除、使用

### 3. 默认配置
- **用户级默认**：每个用户只能有一个默认 AI 配置
- **官方默认**：全局只有一个官方默认配置
- **创建知识库时**：优先用户默认 → 官方默认

### 4. 数据库设计
- **单表设计**：知识库使用单表（与智能体的双表不同）
- **关联关系**：知识库 → AI 配置（RESTRICT 删除）

### 5. API 响应格式
```json
{
  "code": 200,
  "message": "",
  "data": {}
}
```
- `code` 使用 HTTP 状态码
- `message` 仅在错误时填充
- `data` 成功时返回实际数据，失败时返回 `{}`

---

## 四、分阶段实施建议 🚀

### 简化版（第一阶段）- 本次实施
- ✅ 数据库迁移
- ✅ 业务错误定义
- ⏳ AI 配置 CRUD
- ⏳ 知识库 CRUD
- ⏳ 权限控制
- ❌ 暂不实现文档上传
- ❌ 暂不实现向量搜索

**目标**：验证架构设计，测试权限控制

### 完整版（第二阶段）- 后续对话
- 文档上传 HTTP Handler
- 异步文档处理（loader/chunker/embedder）
- Milvus Collection 管理
- 向量搜索 API
- 文档重新处理
- 配额管理（可选）

---

## 五、注意事项 ⚠️

### 1. API Key 安全
- ✅ 测试阶段：明文存储（当前）
- ❌ 生产环境：必须使用 AES-256-GCM 加密

### 2. Milvus Collection
- 第一阶段：仅生成名称，不实际创建
- 第二阶段：集成 Milvus SDK 创建 Collection

### 3. 错误处理
- 统一使用 `internal/pkg/response` 包
- 业务错误映射到 HTTP 状态码

### 4. 测试数据
- 需要预置官方 AI 配置（手动或通过 SQL）
- 官方知识库可由管理员后期创建

---

## 六、快速启动命令 🎯

**新对话开始时，运行以下命令**：

```bash
# 1. 查看已完成的文件
ls -la internal/knowledge/biz/
ls -la migrations/00003*

# 2. 查看待创建的目录
mkdir -p internal/knowledge/data
mkdir -p internal/knowledge/service

# 3. 开始实施（按顺序）
# 先创建 data 层
# 再创建 biz 层
# 最后创建 service 层

# 4. 编译检查
go build -o /dev/null ./cmd/server/

# 5. 运行迁移
goose -dir migrations postgres "..." up

# 6. 启动测试
go run cmd/server/main.go -config=config.yaml
```

---

## 七、TODO 清单 ✅

### 高优先级（本次实施）
- [ ] 创建 `internal/knowledge/data/ai_provider_config.go`
- [ ] 创建 `internal/knowledge/data/knowledge.go`
- [ ] 创建 `internal/knowledge/biz/ai_provider_config.go`
- [ ] 创建 `internal/knowledge/biz/knowledge.go`
- [ ] 创建 `internal/knowledge/service/types.go`
- [ ] 创建 `internal/knowledge/service/ai_provider.go`
- [ ] 创建 `internal/knowledge/service/knowledge.go`
- [ ] 更新 `cmd/server/main.go`
- [ ] 更新 `internal/server/http.go`
- [ ] 运行迁移并测试
- [ ] 编写测试脚本

### 中优先级（第二阶段）
- [ ] 文档上传功能
- [ ] 异步文档处理
- [ ] Milvus 集成
- [ ] 向量搜索 API

### 低优先级（后期优化）
- [ ] API Key 加密
- [ ] 配额管理
- [ ] 文档预览
- [ ] 批量操作

---

## 八、相关文档链接 🔗

- Agent 模块实现：`internal/agent/`
- 数据库包装器：`internal/pkg/database/`
- 统一响应格式：`internal/pkg/response/`
- 现有知识库组件：`internal/knowledge/loader/`, `chunker/`, `embedding/`

---

**交接完成！新对话开始时，直接参考本文档继续实施即可。** 🚀
