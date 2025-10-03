# 数据库迁移文件

## 目录结构

```
migrations/
├── 00001_init_all_tables.sql       # 完整的数据库初始化文件（一次性创建所有表）
├── schemas/                        # 按模块分类的独立 schema（仅供参考，不参与迁移）
│   ├── 01_users.sql               # 用户认证模块
│   ├── 02_agents.sql              # 智能体模块（agents + official_agents）
│   ├── 03_ai_providers.sql        # AI 服务商配置模块
│   ├── 04_knowledge_bases.sql     # 知识库模块
│   ├── 05_documents.sql           # 文档和分块模块
│   └── 06_conversations.sql       # 对话模块（topics + messages）
├── deprecated/                     # 已废弃的旧迁移文件
└── utils/                          # 工具脚本
```

## 设计理念

### 1. 单文件初始化
- **00001_init_all_tables.sql** 是完整的数据库初始化文件
- 包含所有表的创建语句，按依赖关系排序
- 新项目只需运行一个文件即可完成数据库初始化

### 2. 模块化参考 schema
- `schemas/` 目录下的文件**不参与 goose 迁移**
- 仅供开发者查看和理解各模块的表结构
- 便于后续维护和文档化

### 3. 增量迁移
- 未来的数据库变更创建新的迁移文件：`00002_*.sql`, `00003_*.sql` 等
- 保持向前兼容，遵循 goose 迁移规范

## 表依赖关系

```
users (基础)
  ├── agents (用户智能体)
  ├── ai_provider_configs (AI 配置)
  │   └── knowledge_bases (知识库)
  │       ├── documents (文档)
  │       │   └── chunks (分块)
  └── official_agents (官方智能体)

topics (对话主题)
  └── messages (消息)
```

## 使用方法

### 初始化数据库

```bash
# 从项目根目录执行
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=aiwriter sslmode=disable" up
```

### 回滚迁移

```bash
# 回滚到版本 0
goose -dir migrations postgres "..." down-to 0
```

### 查看迁移状态

```bash
goose -dir migrations postgres "..." status
```

## 模块说明

### 1. 用户认证模块
- **表**: `users`
- **功能**: 用户注册、登录、密码认证、2FA、JWT Refresh Token

### 2. 智能体模块
- **表**: `agents`, `official_agents`
- **功能**: 用户创建私有智能体，系统提供官方智能体

### 3. AI 服务商配置
- **表**: `ai_provider_configs`
- **功能**: 支持多种 AI 服务商（OpenAI、Anthropic、本地模型等）

### 4. 知识库模块
- **表**: `knowledge_bases`
- **功能**: 管理文档集合，关联 Milvus collection 和 AI 配置

### 5. 文档和分块模块
- **表**: `documents`, `chunks`
- **功能**: 文档上传、解析、分块、向量化存储

### 6. 对话模块
- **表**: `topics`, `messages`
- **功能**: 用户与智能体的对话记录，支持复杂消息格式（text、thinking、tool_use 等）

## 注意事项

1. **schemas/ 目录文件不参与迁移** - 仅供开发参考
2. **deprecated/ 目录保留旧文件** - 以防需要回滚
3. **生产环境** - 不要直接删除数据重建，应使用增量迁移
4. **时间戳类型** - 统一使用 `TIMESTAMPTZ`（带时区）
5. **UUID 生成** - 由应用层生成 UUID v7，而非数据库 `gen_random_uuid()`

## 开发流程

### 添加新表或修改表结构

1. 在 `schemas/` 目录创建或修改相应模块的 SQL 文件（便于查看）
2. 创建新的迁移文件：`00002_add_xxx.sql`
3. 在新迁移文件中编写 Up 和 Down 语句
4. 测试迁移：`goose up` 和 `goose down`
5. 提交代码

### 示例：添加新字段

```sql
-- 00002_add_user_avatar.sql
-- +goose Up
ALTER TABLE users ADD COLUMN avatar_url VARCHAR(500);

-- +goose Down
ALTER TABLE users DROP COLUMN avatar_url;
```

## 工具脚本

- `utils/check_slow_queries.sql` - 检查慢查询
- `utils/benchmark_users_table.sql` - 性能测试
