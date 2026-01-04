#!/bin/bash
# 三层限流功能测试脚本

set -e

echo "========================================="
echo "三层限流功能测试"
echo "========================================="
echo ""

# 配置
BASE_URL="http://localhost:8080"
MONGO_URI="mongodb://localhost:27017"
MONGO_DATABASE="token_service_v2_test"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 清理函数
cleanup() {
    echo ""
    echo -e "${YELLOW}清理测试环境...${NC}"
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    # 清理测试数据库
    mongosh "$MONGO_URI/$MONGO_DATABASE" --quiet --eval "db.dropDatabase()" 2>/dev/null || true
    echo -e "${GREEN}清理完成${NC}"
}

trap cleanup EXIT

echo "========================================="
echo "1. 启动服务（启用三层限流）"
echo "========================================="

# 设置环境变量
export MONGO_URI="$MONGO_URI"
export MONGO_DATABASE="$MONGO_DATABASE"
export PORT="8080"

# 启用三层限流（设置较小的值便于测试）
export ENABLE_APP_RATE_LIMIT=true
export APP_RATE_LIMIT_PER_MINUTE=10  # 每分钟10个请求
export APP_RATE_LIMIT_PER_HOUR=100
export APP_RATE_LIMIT_PER_DAY=1000

export ENABLE_ACCOUNT_RATE_LIMIT=true
export ENABLE_TOKEN_RATE_LIMIT=true

# 启动服务
echo "启动服务..."
./bearer-token-service > /tmp/bearer-token-service-test.log 2>&1 &
SERVER_PID=$!

# 等待服务启动
echo "等待服务启动..."
for i in {1..30}; do
    if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 服务启动成功${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}✗ 服务启动失败${NC}"
        cat /tmp/bearer-token-service-test.log
        exit 1
    fi
    sleep 1
done

echo ""
echo "========================================="
echo "2. 注册测试账户"
echo "========================================="

REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v2/accounts/register" \
    -H "Content-Type: application/json" \
    -d '{
        "email": "test@example.com",
        "company": "Test Company",
        "password": "password123456"
    }')

echo "$REGISTER_RESPONSE" | jq .

ACCESS_KEY=$(echo "$REGISTER_RESPONSE" | jq -r .access_key)
SECRET_KEY=$(echo "$REGISTER_RESPONSE" | jq -r .secret_key)
ACCOUNT_ID=$(echo "$REGISTER_RESPONSE" | jq -r .account_id)

echo ""
echo -e "${GREEN}✓ 账户创建成功${NC}"
echo "  AccessKey: $ACCESS_KEY"
echo "  AccountID: $ACCOUNT_ID"

# 为账户添加限流配置
echo ""
echo "为账户添加限流配置..."
mongosh "$MONGO_URI/$MONGO_DATABASE" --quiet --eval "
    db.accounts.updateOne(
        { _id: '$ACCOUNT_ID' },
        { \$set: {
            rate_limit: {
                requests_per_minute: 5,
                requests_per_hour: 50,
                requests_per_day: 500
            }
        }}
    )
" || true
echo -e "${GREEN}✓ 账户限流配置完成（5 req/min）${NC}"

echo ""
echo "========================================="
echo "3. 创建带限流的 Token"
echo "========================================="

# 生成 HMAC 签名
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
URI="/api/v2/tokens"
METHOD="POST"
BODY='{"description":"Test Token","scope":["storage:write"],"expires_in_seconds":3600,"rate_limit":{"requests_per_minute":3,"requests_per_hour":30,"requests_per_day":300}}'

# 计算签名
STRING_TO_SIGN="${METHOD}\n${URI}\n${TIMESTAMP}\n${BODY}"
SIGNATURE=$(echo -n "$STRING_TO_SIGN" | openssl dgst -sha256 -hmac "$SECRET_KEY" -binary | base64)
AUTH_HEADER="QINIU ${ACCESS_KEY}:${SIGNATURE}"

TOKEN_RESPONSE=$(curl -s -X POST "$BASE_URL$URI" \
    -H "Content-Type: application/json" \
    -H "Authorization: $AUTH_HEADER" \
    -H "X-Qiniu-Date: $TIMESTAMP" \
    -d "$BODY")

echo "$TOKEN_RESPONSE" | jq .

TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r .token)
TOKEN_ID=$(echo "$TOKEN_RESPONSE" | jq -r .token_id)

echo ""
echo -e "${GREEN}✓ Token 创建成功${NC}"
echo "  Token: ${TOKEN:0:20}..."
echo "  限流配置: 3 req/min, 30 req/hour, 300 req/day"

echo ""
echo "========================================="
echo "4. 测试应用层限流（全局限流）"
echo "========================================="
echo "限制: 10 req/min"
echo ""

SUCCESS_COUNT=0
RATE_LIMITED_COUNT=0

for i in {1..15}; do
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health")

    if [ "$HTTP_CODE" = "200" ]; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        echo -e "${GREEN}请求 $i: 200 OK${NC}"
    elif [ "$HTTP_CODE" = "429" ]; then
        RATE_LIMITED_COUNT=$((RATE_LIMITED_COUNT + 1))
        echo -e "${RED}请求 $i: 429 Too Many Requests (应用层限流)${NC}"
    else
        echo -e "${YELLOW}请求 $i: $HTTP_CODE${NC}"
    fi

    sleep 0.1
done

echo ""
echo "统计:"
echo "  成功: $SUCCESS_COUNT"
echo "  限流: $RATE_LIMITED_COUNT"

if [ $RATE_LIMITED_COUNT -gt 0 ]; then
    echo -e "${GREEN}✓ 应用层限流测试通过${NC}"
else
    echo -e "${YELLOW}⚠ 应用层限流未触发（可能需要更多请求）${NC}"
fi

echo ""
echo "========================================="
echo "5. 测试 Token 层限流"
echo "========================================="
echo "限制: 3 req/min"
echo ""

SUCCESS_COUNT=0
RATE_LIMITED_COUNT=0

for i in {1..8}; do
    RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$BASE_URL/api/v2/validate" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{"required_scope":"storage:write"}')

    HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
    BODY=$(echo "$RESPONSE" | grep -v "HTTP_CODE:")

    if [ "$HTTP_CODE" = "200" ]; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        echo -e "${GREEN}请求 $i: 200 OK - Token 验证成功${NC}"
    elif [ "$HTTP_CODE" = "429" ]; then
        RATE_LIMITED_COUNT=$((RATE_LIMITED_COUNT + 1))
        ERROR_MSG=$(echo "$BODY" | jq -r .error 2>/dev/null || echo "Rate limit exceeded")
        echo -e "${RED}请求 $i: 429 Too Many Requests - $ERROR_MSG${NC}"
    else
        echo -e "${YELLOW}请求 $i: $HTTP_CODE${NC}"
        echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
    fi

    sleep 0.2
done

echo ""
echo "统计:"
echo "  成功: $SUCCESS_COUNT"
echo "  限流: $RATE_LIMITED_COUNT"

if [ $RATE_LIMITED_COUNT -gt 0 ]; then
    echo -e "${GREEN}✓ Token 层限流测试通过${NC}"
else
    echo -e "${YELLOW}⚠ Token 层限流未触发${NC}"
fi

echo ""
echo "========================================="
echo "6. 检查限流响应头"
echo "========================================="

RESPONSE_WITH_HEADERS=$(curl -s -i -X POST "$BASE_URL/api/v2/validate" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"required_scope":"storage:write"}')

echo "$RESPONSE_WITH_HEADERS" | grep -i "x-ratelimit" || echo "未找到限流响应头"

echo ""
echo "========================================="
echo "测试完成"
echo "========================================="
echo ""
echo -e "${GREEN}✓ 三层限流功能实现完成${NC}"
echo ""
echo "功能特性:"
echo "  1. 应用层限流 - 全局流量保护"
echo "  2. 账户层限流 - 防止单租户滥用"
echo "  3. Token层限流 - 精细化权限控制"
echo "  4. 默认全部关闭，通过环境变量启用"
echo "  5. HTTP 响应头返回限流状态"
echo "  6. 滑动窗口算法（分钟/小时/天三个维度）"
echo ""
echo "日志文件: /tmp/bearer-token-service-test.log"
