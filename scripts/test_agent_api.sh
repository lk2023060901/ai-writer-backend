#!/bin/bash

# Test script for Agent REST API
# Usage: ./scripts/test_agent_api.sh

set -e

BASE_URL="http://localhost:8080/api/v1"
TOKEN=""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "=== Testing Agent REST API ==="
echo ""

# Step 1: Register a test user
echo "Step 1: Registering test user..."
REGISTER_RESPONSE=$(curl -s -X POST "${BASE_URL}/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test User",
    "email": "test@example.com",
    "password": "Test1234!"
  }')
echo "Register Response: $REGISTER_RESPONSE"
echo ""

# Step 2: Login to get JWT token
echo "Step 2: Logging in..."
LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "Test1234!"
  }')

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.tokens.access_token')
if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
  echo -e "${RED}❌ Failed to get token${NC}"
  echo "Login Response: $LOGIN_RESPONSE"
  exit 1
fi
echo -e "${GREEN}✓ Got JWT token${NC}"
echo ""

# Step 3: Create an agent
echo "Step 3: Creating an agent..."
CREATE_RESPONSE=$(curl -s -X POST "${BASE_URL}/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Test Agent",
    "emoji": "🤖",
    "prompt": "You are a helpful AI assistant that helps users with their tasks.",
    "knowledge_base_ids": [],
    "tags": ["testing", "demo"]
  }')

AGENT_ID=$(echo $CREATE_RESPONSE | jq -r '.data.id')
if [ "$AGENT_ID" == "null" ] || [ -z "$AGENT_ID" ]; then
  echo -e "${RED}❌ Failed to create agent${NC}"
  echo "Create Response: $CREATE_RESPONSE"
  exit 1
fi
echo -e "${GREEN}✓ Agent created with ID: $AGENT_ID${NC}"
echo "Response: $CREATE_RESPONSE" | jq '.'
echo ""

# Step 4: Get agent details
echo "Step 4: Getting agent details..."
GET_RESPONSE=$(curl -s -X GET "${BASE_URL}/agents/${AGENT_ID}" \
  -H "Authorization: Bearer $TOKEN")
echo "Response: $GET_RESPONSE" | jq '.'
echo ""

# Step 5: List agents
echo "Step 5: Listing agents..."
LIST_RESPONSE=$(curl -s -X GET "${BASE_URL}/agents?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN")
echo "Response: $LIST_RESPONSE" | jq '.'
echo ""

# Step 6: Update agent
echo "Step 6: Updating agent..."
UPDATE_RESPONSE=$(curl -s -X PUT "${BASE_URL}/agents/${AGENT_ID}" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Updated Test Agent",
    "emoji": "🚀",
    "tags": ["testing", "demo", "updated"]
  }')
echo "Response: $UPDATE_RESPONSE" | jq '.'
echo ""

# Step 7: Disable agent
echo "Step 7: Disabling agent..."
DISABLE_RESPONSE=$(curl -s -X PATCH "${BASE_URL}/agents/${AGENT_ID}/disable" \
  -H "Authorization: Bearer $TOKEN")
echo "Response: $DISABLE_RESPONSE" | jq '.'
echo ""

# Step 8: Enable agent
echo "Step 8: Enabling agent..."
ENABLE_RESPONSE=$(curl -s -X PATCH "${BASE_URL}/agents/${AGENT_ID}/enable" \
  -H "Authorization: Bearer $TOKEN")
echo "Response: $ENABLE_RESPONSE" | jq '.'
echo ""

# Step 9: Delete agent
echo "Step 9: Deleting agent..."
DELETE_RESPONSE=$(curl -s -X DELETE "${BASE_URL}/agents/${AGENT_ID}" \
  -H "Authorization: Bearer $TOKEN")
echo "Response: $DELETE_RESPONSE" | jq '.'
echo ""

echo -e "${GREEN}=== All tests completed successfully! ===${NC}"
