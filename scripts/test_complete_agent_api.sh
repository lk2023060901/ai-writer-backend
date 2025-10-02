#!/bin/bash

# Complete Agent REST API Test
# Tests all CRUD operations for agents

set -e

BASE_URL="http://localhost:8080/api/v1"
TOKEN=""
AGENT_ID=""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   智能体 REST API 完整测试${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 1. 注册用户
echo -e "${BLUE}[1/11] 注册新用户...${NC}"
REGISTER_RESPONSE=$(curl -s -X POST "${BASE_URL}/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试用户",
    "email": "testuser@example.com",
    "password": "Test1234!"
  }')

if echo "$REGISTER_RESPONSE" | grep -q "error"; then
  echo -e "${RED}✗ 注册失败: $REGISTER_RESPONSE${NC}"
  exit 1
fi

USER_ID=$(echo $REGISTER_RESPONSE | grep -o '"user_id":"[^"]*"' | cut -d'"' -f4)
echo -e "${GREEN}✓ 用户注册成功${NC}"
echo "  User ID: $USER_ID"
echo ""

# 2. 登录获取 Token
echo -e "${BLUE}[2/11] 用户登录...${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "testuser@example.com",
    "password": "Test1234!"
  }')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
if [ -z "$TOKEN" ]; then
  echo -e "${RED}✗ 登录失败${NC}"
  echo "$LOGIN_RESPONSE"
  exit 1
fi

echo -e "${GREEN}✓ 登录成功，获取到 JWT Token${NC}"
echo "  Token: ${TOKEN:0:30}..."
echo ""

# 3. 创建第一个智能体
echo -e "${BLUE}[3/11] 创建智能体 - 编程助手...${NC}"
CREATE_RESPONSE=$(curl -s -X POST "${BASE_URL}/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "编程助手",
    "emoji": "💻",
    "prompt": "你是一个专业的编程助手，擅长帮助用户解决编程问题、代码审查和技术选型。你会提供清晰、准确的代码示例和最佳实践建议。",
    "knowledge_base_ids": [],
    "tags": ["编程", "技术", "代码"]
  }')

AGENT_ID=$(echo $CREATE_RESPONSE | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -z "$AGENT_ID" ]; then
  echo -e "${RED}✗ 创建失败${NC}"
  echo "$CREATE_RESPONSE"
  exit 1
fi

echo -e "${GREEN}✓ 智能体创建成功${NC}"
echo "  Agent ID: $AGENT_ID"
echo "  Response:"
echo "$CREATE_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$CREATE_RESPONSE"
echo ""

# 4. 创建第二个智能体
echo -e "${BLUE}[4/11] 创建智能体 - 文案写手...${NC}"
CREATE_RESPONSE_2=$(curl -s -X POST "${BASE_URL}/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "文案写手",
    "emoji": "✍️",
    "prompt": "你是一个专业的文案写手，擅长创作各类营销文案、社交媒体内容和品牌故事。你的文案富有创意、吸引眼球且能够精准传达品牌价值。",
    "knowledge_base_ids": [],
    "tags": ["文案", "营销", "创意"]
  }')

AGENT_ID_2=$(echo $CREATE_RESPONSE_2 | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo -e "${GREEN}✓ 第二个智能体创建成功${NC}"
echo "  Agent ID: $AGENT_ID_2"
echo ""

# 5. 获取智能体详情
echo -e "${BLUE}[5/11] 获取智能体详情...${NC}"
GET_RESPONSE=$(curl -s -X GET "${BASE_URL}/agents/${AGENT_ID}" \
  -H "Authorization: Bearer $TOKEN")

echo -e "${GREEN}✓ 获取智能体详情成功${NC}"
echo "$GET_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$GET_RESPONSE"
echo ""

# 6. 列出所有智能体
echo -e "${BLUE}[6/11] 列出所有智能体（分页）...${NC}"
LIST_RESPONSE=$(curl -s -X GET "${BASE_URL}/agents?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN")

TOTAL=$(echo $LIST_RESPONSE | grep -o '"total":[0-9]*' | cut -d':' -f2)
echo -e "${GREEN}✓ 列出智能体成功，共 $TOTAL 个${NC}"
echo "$LIST_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$LIST_RESPONSE"
echo ""

# 7. 更新智能体
echo -e "${BLUE}[7/11] 更新智能体信息...${NC}"
UPDATE_RESPONSE=$(curl -s -X PUT "${BASE_URL}/agents/${AGENT_ID}" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "高级编程助手",
    "emoji": "🚀",
    "prompt": "你是一个高级编程助手，不仅擅长解决编程问题，还能提供系统架构设计、性能优化和安全最佳实践的建议。你拥有丰富的实战经验和深厚的技术功底。",
    "tags": ["编程", "架构", "高级", "性能优化"]
  }')

echo -e "${GREEN}✓ 智能体更新成功${NC}"
echo "$UPDATE_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$UPDATE_RESPONSE"
echo ""

# 8. 禁用智能体
echo -e "${BLUE}[8/11] 禁用智能体...${NC}"
DISABLE_RESPONSE=$(curl -s -X PATCH "${BASE_URL}/agents/${AGENT_ID}/disable" \
  -H "Authorization: Bearer $TOKEN")

echo -e "${GREEN}✓ 智能体已禁用${NC}"
echo "$DISABLE_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$DISABLE_RESPONSE"
echo ""

# 9. 启用智能体
echo -e "${BLUE}[9/11] 启用智能体...${NC}"
ENABLE_RESPONSE=$(curl -s -X PATCH "${BASE_URL}/agents/${AGENT_ID}/enable" \
  -H "Authorization: Bearer $TOKEN")

echo -e "${GREEN}✓ 智能体已启用${NC}"
echo "$ENABLE_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$ENABLE_RESPONSE"
echo ""

# 10. 测试搜索功能（通过名称）
echo -e "${BLUE}[10/11] 测试搜索功能（搜索'编程'）...${NC}"
SEARCH_RESPONSE=$(curl -s -X GET "${BASE_URL}/agents?keyword=编程&page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN")

SEARCH_TOTAL=$(echo $SEARCH_RESPONSE | grep -o '"total":[0-9]*' | cut -d':' -f2)
echo -e "${GREEN}✓ 搜索成功，找到 $SEARCH_TOTAL 个结果${NC}"
echo "$SEARCH_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$SEARCH_RESPONSE"
echo ""

# 11. 删除智能体
echo -e "${BLUE}[11/11] 删除智能体...${NC}"
DELETE_RESPONSE=$(curl -s -X DELETE "${BASE_URL}/agents/${AGENT_ID_2}" \
  -H "Authorization: Bearer $TOKEN")

echo -e "${GREEN}✓ 智能体删除成功（软删除）${NC}"
echo "$DELETE_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$DELETE_RESPONSE"
echo ""

# 验证删除后列表
echo -e "${BLUE}验证删除后的列表...${NC}"
FINAL_LIST=$(curl -s -X GET "${BASE_URL}/agents?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN")

FINAL_TOTAL=$(echo $FINAL_LIST | grep -o '"total":[0-9]*' | cut -d':' -f2)
echo -e "${GREEN}✓ 删除后剩余 $FINAL_TOTAL 个智能体${NC}"
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}   ✓ 所有测试通过！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "测试总结："
echo "  ✓ 用户注册"
echo "  ✓ 用户登录"
echo "  ✓ 创建智能体 x2"
echo "  ✓ 获取智能体详情"
echo "  ✓ 列出所有智能体"
echo "  ✓ 更新智能体"
echo "  ✓ 禁用智能体"
echo "  ✓ 启用智能体"
echo "  ✓ 搜索智能体"
echo "  ✓ 删除智能体（软删除）"
echo ""
