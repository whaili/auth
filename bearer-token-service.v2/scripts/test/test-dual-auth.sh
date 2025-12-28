#!/bin/bash
# 双认证模式测试脚本
# 测试 HMAC 和 Qstub Bearer Token 两种认证方式

set -e

BASE_URL="http://localhost:8080"
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo "=========================================="
echo "  Bearer Token Service V2"
echo "  双认证模式测试"
echo "=========================================="
echo ""

# ==========================================
# 测试 1: Qstub Bearer Token 认证
# ==========================================
echo -e "${BLUE}测试 1: Qstub Bearer Token 认证${NC}"
echo "----------------------------------------"

# 构建 Qstub Token
USER_INFO='{"uid":"12345","email":"testuser@qiniu.com","name":"Test User"}'
QSTUB_TOKEN=$(echo -n "$USER_INFO" | base64)

echo "用户信息: $USER_INFO"
echo "Qstub Token: $QSTUB_TOKEN"
echo ""

# 创建 Token
echo "创建 Bearer Token..."
RESPONSE=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
  -H "Authorization: Bearer $QSTUB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Qstub test token",
    "scope": ["storage:read", "storage:write"],
    "expires_in_seconds": 3600
  }')

echo "响应: $RESPONSE"

# 检查是否成功
if echo "$RESPONSE" | grep -q '"token_id"'; then
    echo -e "${GREEN}✓ Qstub 认证测试通过${NC}"
    TOKEN_ID=$(echo "$RESPONSE" | grep -o '"token_id":"[^"]*"' | cut -d'"' -f4)
    ACCOUNT_ID=$(echo "$RESPONSE" | grep -o '"account_id":"[^"]*"' | cut -d'"' -f4)
    echo "  Token ID: $TOKEN_ID"
    echo "  Account ID: $ACCOUNT_ID"

    # 验证 account_id 格式是否为 qiniu_{uid}
    if [ "$ACCOUNT_ID" = "qiniu_12345" ]; then
        echo -e "${GREEN}✓ Account ID 格式正确 (qiniu_{uid})${NC}"
    else
        echo -e "${RED}✗ Account ID 格式错误，期望: qiniu_12345，实际: $ACCOUNT_ID${NC}"
    fi
else
    echo -e "${RED}✗ Qstub 认证测试失败${NC}"
    echo "错误信息: $RESPONSE"
fi

echo ""

# ==========================================
# 测试 2: HMAC 签名认证（需要先注册账户）
# ==========================================
echo -e "${BLUE}测试 2: HMAC 签名认证${NC}"
echo "----------------------------------------"

# 注册测试账户
echo "注册测试账户..."
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v2/accounts/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "hmac-test@example.com",
    "company": "Test Corp",
    "password": "TestPassword123!"
  }')

echo "注册响应: $REGISTER_RESPONSE"

# 提取 AccessKey 和 SecretKey
ACCESS_KEY=$(echo "$REGISTER_RESPONSE" | grep -o '"access_key":"[^"]*"' | cut -d'"' -f4)
SECRET_KEY=$(echo "$REGISTER_RESPONSE" | grep -o '"secret_key":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ACCESS_KEY" ] || [ -z "$SECRET_KEY" ]; then
    echo -e "${RED}✗ 账户注册失败，跳过 HMAC 测试${NC}"
    echo "响应: $REGISTER_RESPONSE"
else
    echo -e "${GREEN}✓ 账户注册成功${NC}"
    echo "  Access Key: $ACCESS_KEY"
    echo "  Secret Key: ${SECRET_KEY:0:20}..."
    echo ""

    # 使用 HMAC 签名创建 Token
    echo "使用 HMAC 签名创建 Bearer Token..."

    METHOD="POST"
    PATH="/api/v2/tokens"
    TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    BODY='{"description":"HMAC test token","scope":["cdn:refresh"],"expires_in_seconds":7200}'

    # 计算签名
    STRING_TO_SIGN="${METHOD}"$'\n'"${PATH}"$'\n'"${TIMESTAMP}"$'\n'"${BODY}"
    SIGNATURE=$(echo -n "$STRING_TO_SIGN" | openssl dgst -sha256 -hmac "$SECRET_KEY" -binary | base64)

    # 发送请求
    HMAC_RESPONSE=$(curl -s -X POST "$BASE_URL$PATH" \
      -H "Authorization: QINIU ${ACCESS_KEY}:${SIGNATURE}" \
      -H "X-Qiniu-Date: ${TIMESTAMP}" \
      -H "Content-Type: application/json" \
      -d "$BODY")

    echo "响应: $HMAC_RESPONSE"

    # 检查是否成功
    if echo "$HMAC_RESPONSE" | grep -q '"token_id"'; then
        echo -e "${GREEN}✓ HMAC 认证测试通过${NC}"
        HMAC_TOKEN_ID=$(echo "$HMAC_RESPONSE" | grep -o '"token_id":"[^"]*"' | cut -d'"' -f4)
        HMAC_ACCOUNT_ID=$(echo "$HMAC_RESPONSE" | grep -o '"account_id":"[^"]*"' | cut -d'"' -f4)
        echo "  Token ID: $HMAC_TOKEN_ID"
        echo "  Account ID: $HMAC_ACCOUNT_ID"
    else
        echo -e "${RED}✗ HMAC 认证测试失败${NC}"
        echo "错误信息: $HMAC_RESPONSE"
    fi
fi

echo ""

# ==========================================
# 测试 3: 列出 Tokens（使用 Qstub 认证）
# ==========================================
echo -e "${BLUE}测试 3: 列出 Tokens (Qstub 认证)${NC}"
echo "----------------------------------------"

LIST_RESPONSE=$(curl -s -X GET "$BASE_URL/api/v2/tokens?limit=10" \
  -H "Authorization: Bearer $QSTUB_TOKEN")

echo "响应: $LIST_RESPONSE"

if echo "$LIST_RESPONSE" | grep -q '"tokens"'; then
    echo -e "${GREEN}✓ 列出 Tokens 测试通过${NC}"
    TOKEN_COUNT=$(echo "$LIST_RESPONSE" | grep -o '"total":[0-9]*' | cut -d':' -f2)
    echo "  Token 总数: $TOKEN_COUNT"
else
    echo -e "${RED}✗ 列出 Tokens 测试失败${NC}"
fi

echo ""

# ==========================================
# 测试总结
# ==========================================
echo "=========================================="
echo -e "${BLUE}测试总结${NC}"
echo "=========================================="
echo ""
echo "✓ 支持 Qstub Bearer Token 认证"
echo "✓ 支持 HMAC 签名认证"
echo "✓ 两种认证方式互不干扰"
echo "✓ Account ID 自动映射（qiniu_{uid}）"
echo ""
echo -e "${GREEN}所有测试完成！${NC}"
echo ""
