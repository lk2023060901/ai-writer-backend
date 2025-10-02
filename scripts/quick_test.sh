#!/bin/bash

# Get token
TOKEN=$(curl -s http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"Test1234!"}' | jq -r '.tokens.access_token')

echo "✓ Got token"

# List agents
echo ""
echo "=== Listing agents ==="
curl -s http://localhost:8080/api/v1/agents \
  -H "Authorization: Bearer $TOKEN" | jq '.'
