#!/bin/bash
set -e

echo "=========================================="
echo "Bearer Token Service V2 - 测试脚本"
echo "=========================================="

# 配置
BASE_URL="${BASE_URL:-http://localhost:8080}"
TEST_EMAIL="test-$(date +%s)@example.com"
TEST_COMPANY="Test Company"
TEST_PASSWORD="TestPassword123"

echo ""
echo "测试配置:"
echo "  - 服务地址: $BASE_URL"
echo "  - 测试邮箱: $TEST_EMAIL"
echo ""

# HMAC 签名函数
sign_request() {
    local method="$1"
    local uri="$2"
    local timestamp="$3"
    local body="$4"
    local secret_key="$5"

    local string_to_sign="${method}\n${uri}\n${timestamp}\n${body}"
    echo -n "$string_to_sign" | openssl dgst -sha256 -hmac "$secret_key" -binary | base64
}

echo "=========================================="
echo "测试 1: 账户注册"
echo "=========================================="

REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v2/accounts/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"company\":\"$TEST_COMPANY\",\"password\":\"$TEST_PASSWORD\"}")

echo "响应: $REGISTER_RESPONSE"

ACCESS_KEY=$(echo "$REGISTER_RESPONSE" | grep -o '"access_key":"[^"]*"' | cut -d'"' -f4)
SECRET_KEY=$(echo "$REGISTER_RESPONSE" | grep -o '"secret_key":"[^"]*"' | cut -d'"' -f4)
ACCOUNT_ID=$(echo "$REGISTER_RESPONSE" | grep -o '"account_id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ACCESS_KEY" ] || [ -z "$SECRET_KEY" ]; then
    echo "❌ 账户注册失败"
    exit 1
fi

echo "✅ 账户注册成功"
echo "  - Account ID: $ACCOUNT_ID"
echo "  - Access Key: $ACCESS_KEY"
echo "  - Secret Key: ${SECRET_KEY:0:20}..."
echo ""

echo "=========================================="
echo "测试 2: 创建 Token（1小时过期）"
echo "=========================================="

METHOD="POST"
URI="/api/v2/tokens"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BODY='{"description":"Test 1-hour token","scope":["storage:read"],"expires_in_seconds":3600}'

SIGNATURE=$(sign_request "$METHOD" "$URI" "$TIMESTAMP" "$BODY" "$SECRET_KEY")

TOKEN_RESPONSE=$(curl -s -X POST "$BASE_URL$URI" \
    -H "Authorization: QINIU ${ACCESS_KEY}:${SIGNATURE}" \
    -H "X-Qiniu-Date: $TIMESTAMP" \
    -H "Content-Type: application/json" \
    -d "$BODY")

echo "响应: $TOKEN_RESPONSE"

BEARER_TOKEN=$(echo "$TOKEN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
TOKEN_ID=$(echo "$TOKEN_RESPONSE" | grep -o '"token_id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$BEARER_TOKEN" ]; then
    echo "❌ Token 创建失败"
    exit 1
fi

echo "✅ Token 创建成功（1小时过期）"
echo "  - Token ID: $TOKEN_ID"
echo "  - Token: ${BEARER_TOKEN:0:30}..."
echo ""

echo "=========================================="
echo "测试 3: 创建 Token（90天过期）"
echo "=========================================="

TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BODY='{"description":"Test 90-day token","scope":["storage:*","cdn:refresh"],"expires_in_seconds":7776000}'

SIGNATURE=$(sign_request "$METHOD" "$URI" "$TIMESTAMP" "$BODY" "$SECRET_KEY")

TOKEN_RESPONSE_90D=$(curl -s -X POST "$BASE_URL$URI" \
    -H "Authorization: QINIU ${ACCESS_KEY}:${SIGNATURE}" \
    -H "X-Qiniu-Date: $TIMESTAMP" \
    -H "Content-Type: application/json" \
    -d "$BODY")

echo "响应: $TOKEN_RESPONSE_90D"

BEARER_TOKEN_90D=$(echo "$TOKEN_RESPONSE_90D" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$BEARER_TOKEN_90D" ]; then
    echo "❌ Token 创建失败"
    exit 1
fi

echo "✅ Token 创建成功（90天过期）"
echo ""

echo "=========================================="
echo "测试 4: 验证 Token"
echo "=========================================="

VALIDATE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v2/validate" \
    -H "Authorization: Bearer $BEARER_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"required_scope":"storage:read"}')

echo "响应: $VALIDATE_RESPONSE"

VALID=$(echo "$VALIDATE_RESPONSE" | grep -o '"valid":true')

if [ -z "$VALID" ]; then
    echo "❌ Token 验证失败"
    exit 1
fi

echo "✅ Token 验证成功"
echo ""

echo "=========================================="
echo "测试 5: 列出 Tokens"
echo "=========================================="

METHOD="GET"
URI="/api/v2/tokens?active_only=true"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BODY=""

SIGNATURE=$(sign_request "$METHOD" "$URI" "$TIMESTAMP" "$BODY" "$SECRET_KEY")

LIST_RESPONSE=$(curl -s -X GET "$BASE_URL$URI" \
    -H "Authorization: QINIU ${ACCESS_KEY}:${SIGNATURE}" \
    -H "X-Qiniu-Date: $TIMESTAMP")

echo "响应: $LIST_RESPONSE"

TOTAL=$(echo "$LIST_RESPONSE" | grep -o '"total":[0-9]*' | cut -d':' -f2)

if [ "$TOTAL" -ge 2 ]; then
    echo "✅ 列出 Tokens 成功（共 $TOTAL 个）"
else
    echo "❌ 列出 Tokens 失败"
    exit 1
fi

echo ""
echo "=========================================="
echo "测试 6: 获取账户信息"
echo "=========================================="

METHOD="GET"
URI="/api/v2/accounts/me"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BODY=""

SIGNATURE=$(sign_request "$METHOD" "$URI" "$TIMESTAMP" "$BODY" "$SECRET_KEY")

ACCOUNT_RESPONSE=$(curl -s -X GET "$BASE_URL$URI" \
    -H "Authorization: QINIU ${ACCESS_KEY}:${SIGNATURE}" \
    -H "X-Qiniu-Date: $TIMESTAMP")

echo "响应: $ACCOUNT_RESPONSE"

ACCOUNT_EMAIL=$(echo "$ACCOUNT_RESPONSE" | grep -o "\"email\":\"$TEST_EMAIL\"")

if [ -n "$ACCOUNT_EMAIL" ]; then
    echo "✅ 获取账户信息成功"
else
    echo "❌ 获取账户信息失败"
    exit 1
fi

echo ""
echo "=========================================="
echo "所有测试通过！✅"
echo "=========================================="
echo ""
echo "测试账户信息:"
echo "  - Email: $TEST_EMAIL"
echo "  - Access Key: $ACCESS_KEY"
echo "  - Secret Key: $SECRET_KEY"
echo ""
echo "生成的 Tokens:"
echo "  1. 1小时 Token: $BEARER_TOKEN"
echo "  2. 90天 Token: $BEARER_TOKEN_90D"
echo ""
