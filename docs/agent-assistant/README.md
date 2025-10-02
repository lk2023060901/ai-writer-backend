# Agent & Assistant 系统文档

基于 Cherry Studio 架构的 AI 智能体管理系统。

## 概述

该系统提供完整的 AI 智能体管理功能：
- **Agent（模板）**: 预设的 AI 角色模板，如"产品经理"、"SEO专家"
- **Assistant（助手）**: 用户实例化的 AI 助手，基于 Agent 创建
- **Topic（对话）**: 每个 Assistant 支持多个对话主题

## 架构

```
internal/
├── agent/                  # Agent 领域
│   ├── types/             # 类型定义
│   ├── models/            # GORM 数据模型
│   ├── biz/               # 业务逻辑
│   ├── data/              # 数据访问
│   ├── service/           # HTTP API
│   └── seed/              # 内置数据
│
└── assistant/              # Assistant 领域
    ├── types/             # 类型定义
    ├── models/            # GORM 数据模型
    ├── biz/               # 业务逻辑
    ├── data/              # 数据访问
    └── service/           # HTTP API
```

## 快速开始

### 1. 运行数据库迁移

```go
import (
    agentModels "ai-writer-backend/internal/agent/models"
    assistantModels "ai-writer-backend/internal/assistant/models"
)

// 在 main.go 中添加
agentModels.AutoMigrate(db)
assistantModels.AutoMigrate(db)
```

### 2. 初始化内置 Agent

```go
import (
    agentData "ai-writer-backend/internal/agent/data"
    agentSeed "ai-writer-backend/internal/agent/seed"
)

agentRepo := agentData.NewAgentRepo(db)
agentSeed.SeedBuiltinAgents(ctx, agentRepo)
```

### 3. 注册路由

```go
import (
    agentService "ai-writer-backend/internal/agent/service"
    assistantService "ai-writer-backend/internal/assistant/service"
)

// 初始化服务
agentSvc := agentService.NewAgentService(agentUseCase)
assistantSvc := assistantService.NewAssistantService(assistantUseCase)
topicSvc := assistantService.NewTopicService(topicUseCase)

// 注册路由
v1 := router.Group("/api/v1")
agentSvc.RegisterRoutes(v1)
assistantSvc.RegisterRoutes(v1)
topicSvc.RegisterRoutes(v1)
```

## API 文档

### Agent API

#### 列出所有 Agent
```http
GET /api/v1/agents
Query Parameters:
  - group: 按分组筛选
  - is_builtin: 是否内置（true/false）
  - keyword: 搜索关键词
```

#### 获取 Agent 详情
```http
GET /api/v1/agents/:id
```

#### 创建 Agent（管理员）
```http
POST /api/v1/agents
Content-Type: application/json

{
  "name": "新角色",
  "description": "角色描述",
  "emoji": "🤖",
  "prompt": "系统提示词...",
  "group": ["自定义"],
  "settings": {
    "temperature": 0.7,
    "max_tokens": 2000,
    "context_count": 10,
    "enable_web_search": false,
    "tool_use_mode": "function"
  }
}
```

#### 列出所有分组
```http
GET /api/v1/agents/groups
```

#### 按分组列出 Agent
```http
GET /api/v1/agents/group/:name
```

### Assistant API

#### 创建 Assistant
```http
POST /api/v1/assistants
Content-Type: application/json

{
  "name": "我的助手",
  "emoji": "🤖",
  "prompt": "自定义提示词",
  "tags": ["工作", "编程"],
  "model_id": "gpt-4",
  "settings": {
    "temperature": 0.7,
    "stream_output": true
  },
  "enable_web_search": true,
  "web_search_provider_id": "tavily"
}
```

#### 从 Agent 创建 Assistant
```http
POST /api/v1/assistants/from-agent/:agent_id
```

#### 列出用户的 Assistants
```http
GET /api/v1/assistants
Query Parameters:
  - keyword: 搜索关键词
```

#### 更新 Assistant
```http
PUT /api/v1/assistants/:id
Content-Type: application/json

{
  "name": "新名称",
  "emoji": "🚀",
  "tags": ["更新的标签"]
}
```

#### 更新设置
```http
PUT /api/v1/assistants/:id/settings
Content-Type: application/json

{
  "temperature": 0.8,
  "max_tokens": 4000,
  "context_count": 20,
  "stream_output": true
}
```

#### 删除 Assistant
```http
DELETE /api/v1/assistants/:id
```

### Topic API

#### 创建 Topic
```http
POST /api/v1/assistants/:assistant_id/topics
Content-Type: application/json

{
  "name": "新对话"
}
```

#### 列出 Topics
```http
GET /api/v1/assistants/:assistant_id/topics
```

#### 更新 Topic
```http
PUT /api/v1/assistants/:assistant_id/topics/:topic_id
Content-Type: application/json

{
  "name": "更新的名称"
}
```

#### 删除 Topic
```http
DELETE /api/v1/assistants/:assistant_id/topics/:topic_id
```

#### 清空所有 Topics
```http
DELETE /api/v1/assistants/:assistant_id/topics
```

## 内置 Agent

系统内置以下 Agent 模板：

1. **产品经理** (pm-001) - 职业、商业、工具
2. **SEO专家** (seo-001) - 职业、营销
3. **文案专家** (writer-001) - 写作、营销
4. **全栈开发** (dev-001) - 技术、职业
5. **教育导师** (teacher-001) - 教育、职业
6. **翻译专家** (translator-001) - 工具、写作

## 使用示例

完整示例代码见: `examples/agent-assistant/main.go`

```go
// 1. 列出所有 Agent
agents, _ := agentUseCase.ListAgents(ctx, nil)

// 2. 从 Agent 创建 Assistant
assistant, _ := assistantUseCase.CreateFromAgent(ctx, userID, "seo-001")

// 3. 创建新对话
topic, _ := topicUseCase.CreateTopic(ctx, assistant.ID, "SEO策略讨论")

// 4. 更新 Assistant
assistantUseCase.UpdateAssistant(ctx, assistant.ID, userID, &UpdateRequest{
    Name: "我的SEO助手",
})
```

## 数据库表结构

### agents
```sql
- id: varchar(36) PK
- name: varchar(255)
- description: text
- emoji: varchar(10)
- prompt: text
- groups: json
- settings: json
- is_builtin: boolean
- created_at, updated_at
```

### assistants
```sql
- id: varchar(36) PK
- user_id: varchar(36)
- agent_id: varchar(36)
- name: varchar(255)
- emoji: varchar(10)
- prompt: text
- type: varchar(50)
- tags: json
- model_id: varchar(100)
- settings: json
- enable_web_search: boolean
- web_search_provider_id: varchar(50)
- enable_memory: boolean
- enable_knowledge: boolean
- knowledge_base_ids: json
- created_at, updated_at
```

### topics
```sql
- id: varchar(36) PK
- assistant_id: varchar(36) FK
- name: varchar(255)
- is_name_manually_edited: boolean
- created_at, updated_at
```

## 扩展功能

### 集成 Web Search
```go
assistant.EnableWebSearch = true
assistant.WebSearchProviderID = "tavily"
```

### 集成知识库
```go
assistant.EnableKnowledge = true
assistant.KnowledgeBaseIDs = []string{"kb-001", "kb-002"}
```

### 集成记忆功能
```go
assistant.EnableMemory = true
```

## 最佳实践

1. **Agent 作为模板**: Agent 是不可变的模板，不要频繁修改
2. **Assistant 个性化**: 用户可以自由定制 Assistant 的设置
3. **Topic 隔离**: 不同对话使用不同 Topic，保持上下文清晰
4. **标签管理**: 使用标签对 Assistant 进行分类管理
5. **软删除**: Assistant 和 Topic 使用软删除，可恢复

## 许可证

MIT License
