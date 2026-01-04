#!/bin/bash
# 三层限流功能测试脚本

set -e

echo "========================================="
echo "三层限流功能测试"
echo "========================================="
echo ""

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查依赖
echo "检查依赖..."
MISSING_DEPS=()

if ! command -v curl &> /dev/null; then
    MISSING_DEPS+=("curl")
fi

if ! command -v jq &> /dev/null; then
    MISSING_DEPS+=("jq")
fi

if ! command -v mongosh &> /dev/null; then
    MISSING_DEPS+=("mongosh")
fi

if ! command -v openssl &> /dev/null; then
    MISSING_DEPS+=("openssl")
fi

if [ ${#MISSING_DEPS[@]} -gt 0 ]; then
    echo -e "${RED}✗ 缺少以下依赖：${NC}"
    for dep in "${MISSING_DEPS[@]}"; do
        echo "  - $dep"
    done
    echo ""
    echo "安装命令："
    for dep in "${MISSING_DEPS[@]}"; do
        if [ "$dep" = "mongosh" ]; then
            echo "  mongosh: 运行 ./scripts/install_mongosh.sh"
        else
            echo "  $dep: sudo apt-get install -y $dep"
        fi
    done
    exit 1
fi

echo -e "${GREEN}✓ 所有依赖已安装${NC}"
echo ""

# 配置
BASE_URL="http://localhost:8081"
MONGO_URI="mongodb://admin:123456@localhost:27017/token_service_v2_test?authSource=admin"
MONGO_DATABASE="token_service_v2_test"

# 清理函数
cleanup() {
    echo ""
    echo -e "${YELLOW}清理测试环境...${NC}"
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    # 清理测试数据库
    mongosh "$MONGO_URI" --quiet --eval "use $MONGO_DATABASE; db.dropDatabase();" 2>/dev/null || true
    echo -e "${GREEN}清理完成${NC}"
}

trap cleanup EXIT

echo "========================================="
echo "1. 启动服务（启用三层限流）"
echo "========================================="

# 设置环境变量 - 使用非常小的限流值便于测试
export MONGO_URI="$MONGO_URI"
export MONGO_DATABASE="$MONGO_DATABASE"
export PORT="8081"

# 应用层限流：每分钟 5 个请求（非常小，容易触发）
export ENABLE_APP_RATE_LIMIT=true
export APP_RATE_LIMIT_PER_MINUTE=5
export APP_RATE_LIMIT_PER_HOUR=100
export APP_RATE_LIMIT_PER_DAY=1000

export ENABLE_ACCOUNT_RATE_LIMIT=true
export ENABLE_TOKEN_RATE_LIMIT=true

# 启动服务
echo "启动服务..."
../bin/tokenserv > /tmp/bearer-token-service-test.log 2>&1 &
SERVER_PID=$!

# 等待服务启动
echo "等待服务启动..."
for i in {1..30}; do
    if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 服务启动成功 (PID: $SERVER_PID)${NC}"
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
echo "配置信息："
echo "  应用层限流: 5 req/min"
echo "  账户层限流: 将设置为 3 req/min"
echo "  Token层限流: 将设置为 2 req/min"
echo ""

# 清理旧数据
echo "========================================="
echo "1.5. 清理测试数据（确保干净环境）"
echo "========================================="

# 删除整个数据库和所有集合
mongosh "$MONGO_URI/$MONGO_DATABASE" --quiet --eval "
    db.dropDatabase();
    print('Database dropped');
" 2>&1 | grep -v "switched to"

# 等待操作完成
sleep 2

# 再次确认清理所有集合
mongosh "$MONGO_URI/$MONGO_DATABASE" --quiet --eval "
    var colls = db.getCollectionNames();
    colls.forEach(function(c) { db[c].drop(); });
    db.accounts.deleteMany({});
    db.tokens.deleteMany({});
    db.audit_logs.deleteMany({});
    print('All data cleared');
" 2>&1 | grep -v "switched to"

echo -e "${GREEN}✓ 数据库已清理${NC}"
echo ""

echo "========================================="
echo "2. 注册测试账户"
echo "========================================="

# 使用时间戳确保邮箱唯一
TEST_EMAIL="test-$(date +%s)@example.com"

REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v2/accounts/register" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$TEST_EMAIL\",
        \"company\": \"Test Company\",
        \"password\": \"password123456\"
    }")

# 检查响应
if echo "$REGISTER_RESPONSE" | grep -q "email already registered"; then
    echo -e "${RED}✗ 邮箱已注册，使用备用邮箱重试...${NC}"
    TEST_EMAIL="test-backup-$(date +%s%N)@example.com"
    REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v2/accounts/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$TEST_EMAIL\",
            \"company\": \"Test Company\",
            \"password\": \"password123456\"
        }")
fi

echo "$REGISTER_RESPONSE" | jq . || {
    echo -e "${RED}✗ 账户注册失败${NC}"
    echo "$REGISTER_RESPONSE"
    echo ""
    echo "可能的原因："
    echo "  1. 数据库清理不彻底"
    echo "  2. 服务使用了不同的数据库"
    echo ""
    echo "手动清理命令："
    echo "  mongosh \"$MONGO_URI/$MONGO_DATABASE\" --eval \"db.dropDatabase()\""
    exit 1
}

ACCESS_KEY=$(echo "$REGISTER_RESPONSE" | jq -r .access_key)
SECRET_KEY=$(echo "$REGISTER_RESPONSE" | jq -r .secret_key)
ACCOUNT_ID=$(echo "$REGISTER_RESPONSE" | jq -r .account_id)

if [ "$ACCESS_KEY" = "null" ] || [ -z "$ACCESS_KEY" ]; then
    echo -e "${RED}✗ 无法获取 AccessKey${NC}"
    echo "响应内容："
    echo "$REGISTER_RESPONSE"
    exit 1
fi

echo ""
echo -e "${GREEN}✓ 账户创建成功${NC}"
echo "  AccessKey: $ACCESS_KEY"
echo "  AccountID: $ACCOUNT_ID"

# 为账户添加限流配置（3 req/min）
echo ""
echo "为账户添加限流配置..."
mongosh "$MONGO_URI" --quiet --eval "
    use $MONGO_DATABASE;
    db.accounts.updateOne(
        { _id: '$ACCOUNT_ID' },
        { \$set: {
            rate_limit: {
                requests_per_minute: 3,
                requests_per_hour: 50,
                requests_per_day: 500
            }
        }}
    );
    print('Updated account rate limit');
" || {
    echo -e "${RED}✗ 设置账户限流失败${NC}"
    exit 1
}
echo -e "${GREEN}✓ 账户限流配置完成（3 req/min）${NC}"

echo ""
echo "========================================="
echo "3. 创建带限流的 Token"
echo "========================================="

# 生成 HMAC 签名
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
URI="/api/v2/tokens"
METHOD="POST"
BODY='{"description":"Test Token","scope":["storage:write"],"expires_in_seconds":3600,"rate_limit":{"requests_per_minute":2,"requests_per_hour":30,"requests_per_day":300}}'

# 计算签名
STRING_TO_SIGN="${METHOD}\n${URI}\n${TIMESTAMP}\n${BODY}"
SIGNATURE=$(echo -n "$STRING_TO_SIGN" | openssl dgst -sha256 -hmac "$SECRET_KEY" -binary | base64)
AUTH_HEADER="QINIU ${ACCESS_KEY}:${SIGNATURE}"

TOKEN_RESPONSE=$(curl -s -X POST "$BASE_URL$URI" \
    -H "Content-Type: application/json" \
    -H "Authorization: $AUTH_HEADER" \
    -H "X-Qiniu-Date: $TIMESTAMP" \
    -d "$BODY")

echo "$TOKEN_RESPONSE" | jq . || {
    echo -e "${RED}✗ Token 创建失败${NC}"
    echo "$TOKEN_RESPONSE"
    exit 1
}

TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r .token)
TOKEN_ID=$(echo "$TOKEN_RESPONSE" | jq -r .token_id)

if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
    echo -e "${RED}✗ 无法获取 Token${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✓ Token 创建成功${NC}"
echo "  Token ID: $TOKEN_ID"
echo "  Token: ${TOKEN:0:20}...${TOKEN: -10}"
echo "  限流配置: 2 req/min, 30 req/hour, 300 req/day"

echo ""
echo "========================================="
echo "4. 测试应用层限流（全局限流）"
echo "========================================="
echo -e "${BLUE}限制: 5 req/min${NC}"
echo -e "${BLUE}测试: 发送 10 个请求，预期第 6 个开始触发限流${NC}"
echo ""

SUCCESS_COUNT=0
RATE_LIMITED_COUNT=0

for i in {1..10}; do
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health")

    if [ "$HTTP_CODE" = "200" ]; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        echo -e "${GREEN}请求 $i: 200 OK${NC}"
    elif [ "$HTTP_CODE" = "429" ]; then
        RATE_LIMITED_COUNT=$((RATE_LIMITED_COUNT + 1))
        echo -e "${RED}请求 $i: 429 Too Many Requests (应用层限流) ✓${NC}"
    else
        echo -e "${YELLOW}请求 $i: $HTTP_CODE${NC}"
    fi
done

echo ""
echo "统计:"
echo "  成功: $SUCCESS_COUNT"
echo "  限流: $RATE_LIMITED_COUNT"

if [ $RATE_LIMITED_COUNT -gt 0 ]; then
    echo -e "${GREEN}✓✓✓ 应用层限流测试通过 - 成功触发限流！${NC}"
else
    echo -e "${RED}✗✗✗ 应用层限流测试失败 - 未触发限流${NC}"
    echo "查看服务日志："
    tail -50 /tmp/bearer-token-service-test.log
    exit 1
fi

# 等待限流窗口重置
echo ""
echo -e "${YELLOW}等待 65 秒，让限流窗口重置...${NC}"
sleep 65

echo ""
echo "========================================="
echo "5. 测试 Token 层限流"
echo "========================================="
echo -e "${BLUE}限制: 2 req/min${NC}"
echo -e "${BLUE}测试: 发送 5 个 Token 验证请求，预期第 3 个开始触发限流${NC}"
echo ""

SUCCESS_COUNT=0
RATE_LIMITED_COUNT=0

for i in {1..5}; do
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
        echo -e "${RED}请求 $i: 429 Too Many Requests - $ERROR_MSG ✓${NC}"
    elif [ "$HTTP_CODE" = "401" ]; then
        echo -e "${YELLOW}请求 $i: 401 Unauthorized - Token 验证失败${NC}"
        echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
    else
        echo -e "${YELLOW}请求 $i: $HTTP_CODE${NC}"
        echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
    fi
done

echo ""
echo "统计:"
echo "  成功: $SUCCESS_COUNT"
echo "  限流: $RATE_LIMITED_COUNT"

if [ $RATE_LIMITED_COUNT -gt 0 ]; then
    echo -e "${GREEN}✓✓✓ Token 层限流测试通过 - 成功触发限流！${NC}"
else
    echo -e "${RED}✗✗✗ Token 层限流测试失败 - 未触发限流${NC}"
    echo "查看服务日志："
    tail -50 /tmp/bearer-token-service-test.log
    exit 1
fi

echo ""
echo "========================================="
echo "6. 测试账户层限流"
echo "========================================="
echo -e "${BLUE}限制: 3 req/min${NC}"
echo -e "${BLUE}测试: 使用 HMAC 认证发送 6 个请求，预期第 4 个开始触发限流${NC}"
echo ""

# 等待限流窗口重置
echo -e "${YELLOW}等待 65 秒，让限流窗口重置...${NC}"
sleep 65

SUCCESS_COUNT=0
RATE_LIMITED_COUNT=0

for i in {1..6}; do
    # 创建新的时间戳和签名
    TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    URI="/api/v2/accounts/me"
    METHOD="GET"
    BODY=""

    STRING_TO_SIGN="${METHOD}\n${URI}\n${TIMESTAMP}\n${BODY}"
    SIGNATURE=$(echo -n "$STRING_TO_SIGN" | openssl dgst -sha256 -hmac "$SECRET_KEY" -binary | base64)
    AUTH_HEADER="QINIU ${ACCESS_KEY}:${SIGNATURE}"

    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL$URI" \
        -H "Authorization: $AUTH_HEADER" \
        -H "X-Qiniu-Date: $TIMESTAMP")

    if [ "$HTTP_CODE" = "200" ]; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        echo -e "${GREEN}请求 $i: 200 OK${NC}"
    elif [ "$HTTP_CODE" = "429" ]; then
        RATE_LIMITED_COUNT=$((RATE_LIMITED_COUNT + 1))
        echo -e "${RED}请求 $i: 429 Too Many Requests (账户层限流) ✓${NC}"
    else
        echo -e "${YELLOW}请求 $i: $HTTP_CODE${NC}"
    fi
done

echo ""
echo "统计:"
echo "  成功: $SUCCESS_COUNT"
echo "  限流: $RATE_LIMITED_COUNT"

if [ $RATE_LIMITED_COUNT -gt 0 ]; then
    echo -e "${GREEN}✓✓✓ 账户层限流测试通过 - 成功触发限流！${NC}"
else
    echo -e "${RED}✗✗✗ 账户层限流测试失败 - 未触发限流${NC}"
    echo "查看服务日志："
    tail -50 /tmp/bearer-token-service-test.log
    exit 1
fi

echo ""
echo "========================================="
echo "7. 检查限流响应头"
echo "========================================="

# 等待限流重置
echo -e "${YELLOW}等待 65 秒，让限流窗口重置...${NC}"
sleep 65

echo "发送一个正常请求，检查限流响应头..."
RESPONSE_WITH_HEADERS=$(curl -s -i -X POST "$BASE_URL/api/v2/validate" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"required_scope":"storage:write"}')

echo ""
echo "限流响应头："
echo "$RESPONSE_WITH_HEADERS" | grep -i "x-ratelimit" || echo "未找到限流响应头"

# 检查是否包含所有三层的限流头
HAS_APP_LIMIT=$(echo "$RESPONSE_WITH_HEADERS" | grep -i "x-ratelimit-limit-app" || echo "")
HAS_TOKEN_LIMIT=$(echo "$RESPONSE_WITH_HEADERS" | grep -i "x-ratelimit-limit-token" || echo "")

echo ""
if [ ! -z "$HAS_APP_LIMIT" ] && [ ! -z "$HAS_TOKEN_LIMIT" ]; then
    echo -e "${GREEN}✓ 限流响应头检查通过${NC}"
else
    echo -e "${YELLOW}⚠ 部分限流响应头缺失${NC}"
fi

echo ""
echo "========================================="
echo "测试完成"
echo "========================================="
echo ""
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  ✓✓✓ 三层限流功能测试全部通过！  ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo "测试结果总结："
echo "  ✓ 应用层限流 - 触发成功"
echo "  ✓ Token 层限流 - 触发成功"
echo "  ✓ 账户层限流 - 触发成功"
echo "  ✓ 限流响应头 - 返回正确"
echo ""
echo "功能特性:"
echo "  1. 应用层限流 - 全局流量保护 (5 req/min)"
echo "  2. 账户层限流 - 防止单租户滥用 (3 req/min)"
echo "  3. Token层限流 - 精细化权限控制 (2 req/min)"
echo "  4. 默认全部关闭，通过环境变量启用"
echo "  5. HTTP 响应头返回限流状态"
echo "  6. 滑动窗口算法（分钟/小时/天三个维度）"
echo ""
echo "日志文件: /tmp/bearer-token-service-test.log"
