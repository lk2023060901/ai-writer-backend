#!/bin/bash

# 配置 API Keys 并测试同步功能

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== AI 模型同步测试脚本 ===${NC}\n"

# 1. 配置 API Keys（请替换为真实的 API Keys）
echo -e "${YELLOW}步骤 1: 配置 API Keys${NC}"
echo "请在数据库中手动配置 API Keys:"
echo ""
echo "UPDATE ai_providers SET api_key = 'your-siliconflow-key' WHERE provider_type = 'siliconflow';"
echo "UPDATE ai_providers SET api_key = 'your-anthropic-key' WHERE provider_type = 'anthropic';"
echo "UPDATE ai_providers SET api_key = 'your-zhipu-key' WHERE provider_type = 'zhipu';"
echo ""
echo -e "${GREEN}配置完成后按回车继续...${NC}"
read

# 2. 注册测试用户
echo -e "\n${YELLOW}步骤 2: 注册测试用户${NC}"
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  --data-raw '{"name":"Test Admin","email":"test_admin@example.com","password":"Test123456"}')

echo "$REGISTER_RESPONSE" | jq '.'

# 3. 登录获取 Token
echo -e "\n${YELLOW}步骤 3: 登录获取 JWT Token${NC}"
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  --data-raw '{"account":"test_admin@example.com","password":"Test123456"}' | jq -r '.data.token // empty')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo -e "${RED}❌ 登录失败${NC}"
  exit 1
fi

echo -e "${GREEN}✅ 登录成功${NC}"

# 4. 获取 AI Providers
echo -e "\n${YELLOW}步骤 4: 获取 AI Providers 列表${NC}"
PROVIDERS=$(curl -s -X GET http://localhost:8080/api/v1/ai-providers \
  -H "Authorization: Bearer $TOKEN")

echo "$PROVIDERS" | jq '.data.items[] | {id, provider_type, provider_name}'

# 5. 测试每个 Provider 的同步
echo "$PROVIDERS" | jq -r '.data.items[] | @base64' | while read provider; do
  _jq() {
    echo ${provider} | base64 --decode | jq -r ${1}
  }

  PROVIDER_ID=$(_jq '.id')
  PROVIDER_NAME=$(_jq '.provider_name')
  PROVIDER_TYPE=$(_jq '.provider_type')

  # 跳过 OpenAI（没有 API Key）
  if [ "$PROVIDER_TYPE" = "openai" ]; then
    echo -e "\n${YELLOW}⏭️  跳过 OpenAI (没有 API Key)${NC}"
    continue
  fi

  echo -e "\n${YELLOW}========================================${NC}"
  echo -e "${YELLOW}测试: $PROVIDER_NAME ($PROVIDER_TYPE)${NC}"
  echo -e "${YELLOW}========================================${NC}"

  # 触发同步
  echo -e "\n${GREEN}触发模型同步...${NC}"
  SYNC_RESULT=$(curl -s -X POST \
    "http://localhost:8080/api/v1/ai-providers/$PROVIDER_ID/models/sync" \
    -H "Authorization: Bearer $TOKEN")

  echo "$SYNC_RESULT" | jq '.'

  # 获取同步后的模型列表
  echo -e "\n${GREEN}获取同步后的模型列表...${NC}"
  MODELS=$(curl -s -X GET \
    "http://localhost:8080/api/v1/ai-providers/$PROVIDER_ID/models" \
    -H "Authorization: Bearer $TOKEN")

  MODEL_COUNT=$(echo "$MODELS" | jq '.data.total')
  echo -e "${GREEN}✅ 同步完成，共 $MODEL_COUNT 个模型${NC}"

  # 显示前5个模型
  echo -e "\n${GREEN}前5个模型:${NC}"
  echo "$MODELS" | jq '.data.items[0:5] | .[] | {model_name, capabilities: [.capabilities[].capability_type]}'

  # 获取同步历史
  echo -e "\n${GREEN}同步历史:${NC}"
  HISTORY=$(curl -s -X GET \
    "http://localhost:8080/api/v1/ai-providers/$PROVIDER_ID/models/sync-history?limit=3" \
    -H "Authorization: Bearer $TOKEN")

  echo "$HISTORY" | jq '.data.items[] | {sync_type, new_models_count, synced_at}'

done

# 6. 查询所有模型
echo -e "\n${YELLOW}========================================${NC}"
echo -e "${YELLOW}步骤 6: 查询所有同步的模型${NC}"
echo -e "${YELLOW}========================================${NC}"

ALL_MODELS=$(curl -s -X GET http://localhost:8080/api/v1/ai-models \
  -H "Authorization: Bearer $TOKEN")

TOTAL=$(echo "$ALL_MODELS" | jq '.data.total')
echo -e "${GREEN}✅ 总共 $TOTAL 个模型${NC}"

# 7. 按能力类型查询
echo -e "\n${GREEN}Embedding 模型:${NC}"
EMBEDDING=$(curl -s -X GET http://localhost:8080/api/v1/ai-models/capability/embedding \
  -H "Authorization: Bearer $TOKEN")
echo "$EMBEDDING" | jq '.data.items[] | {model_name, dimensions: .capabilities[0].embedding_dimensions}'

echo -e "\n${GREEN}Chat 模型:${NC}"
CHAT=$(curl -s -X GET http://localhost:8080/api/v1/ai-models/capability/chat \
  -H "Authorization: Bearer $TOKEN")
echo "$CHAT" | jq '.data.items[0:5] | .[] | {model_name, vision: .capabilities[0].supports_vision}'

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}✅ 测试完成！${NC}"
echo -e "${GREEN}========================================${NC}"
