#!/bin/bash
# 三层限流功能快速测试脚本（无需等待窗口重置）

echo "========================================="
echo "三层限流功能快速测试"
echo "========================================="
echo ""

# 配置
BASE_URL="http://localhost:8081"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 检查服务是否运行
echo "检查服务是否运行..."
if ! curl -s "$BASE_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}✗ 服务未运行，请先启动服务${NC}"
    echo "启动命令："
    echo "  export ENABLE_APP_RATE_LIMIT=true"
    echo "  export APP_RATE_LIMIT_PER_MINUTE=5"
    echo "  export ENABLE_TOKEN_RATE_LIMIT=true"
    echo "  ./bearer-token-service"
    exit 1
fi
echo -e "${GREEN}✓ 服务正在运行${NC}"

echo ""
echo "========================================="
echo "测试 1: 应用层限流"
echo "========================================="
echo -e "${BLUE}预期: 如果启用了应用层限流（如 5 req/min），第 6 个请求应该被限流${NC}"
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
        echo -e "${RED}请求 $i: 429 Too Many Requests ✓${NC}"
    else
        echo -e "${YELLOW}请求 $i: $HTTP_CODE${NC}"
    fi
done

echo ""
echo "统计: 成功=$SUCCESS_COUNT, 限流=$RATE_LIMITED_COUNT"

if [ $RATE_LIMITED_COUNT -gt 0 ]; then
    echo -e "${GREEN}✓ 应用层限流工作正常${NC}"
else
    echo -e "${YELLOW}⚠ 应用层限流未触发（可能未启用或限制过高）${NC}"
fi

echo ""
echo "========================================="
echo "测试 2: Token 层限流（需要先创建 Token）"
echo "========================================="
echo ""

# 读取用户输入的 Token
echo -e "${YELLOW}请输入测试 Token（或按 Enter 跳过）:${NC}"
read -r TEST_TOKEN

if [ -z "$TEST_TOKEN" ]; then
    echo -e "${YELLOW}跳过 Token 层限流测试${NC}"
else
    echo ""
    echo -e "${BLUE}使用 Token 发送 5 个验证请求...${NC}"
    echo ""

    SUCCESS_COUNT=0
    RATE_LIMITED_COUNT=0

    for i in {1..5}; do
        RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$BASE_URL/api/v2/validate" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $TEST_TOKEN" \
            -d '{"required_scope":"storage:write"}')

        HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)

        if [ "$HTTP_CODE" = "200" ]; then
            SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
            echo -e "${GREEN}请求 $i: 200 OK${NC}"
        elif [ "$HTTP_CODE" = "429" ]; then
            RATE_LIMITED_COUNT=$((RATE_LIMITED_COUNT + 1))
            echo -e "${RED}请求 $i: 429 Too Many Requests ✓${NC}"
        elif [ "$HTTP_CODE" = "401" ]; then
            echo -e "${YELLOW}请求 $i: 401 Unauthorized (Token 无效)${NC}"
            break
        else
            echo -e "${YELLOW}请求 $i: $HTTP_CODE${NC}"
        fi
    done

    echo ""
    echo "统计: 成功=$SUCCESS_COUNT, 限流=$RATE_LIMITED_COUNT"

    if [ $RATE_LIMITED_COUNT -gt 0 ]; then
        echo -e "${GREEN}✓ Token 层限流工作正常${NC}"
    else
        echo -e "${YELLOW}⚠ Token 层限流未触发（可能未启用或限制过高）${NC}"
    fi
fi

echo ""
echo "========================================="
echo "测试 3: 检查限流响应头"
echo "========================================="
echo ""

RESPONSE=$(curl -s -i "$BASE_URL/health" 2>&1)

echo "限流相关响应头："
echo "$RESPONSE" | grep -i "x-ratelimit" || echo "  (未找到限流响应头)"

echo ""
echo "========================================="
echo "快速测试完成"
echo "========================================="
echo ""
echo "提示："
echo "  1. 应用层限流通过环境变量 ENABLE_APP_RATE_LIMIT=true 启用"
echo "  2. Token 层限流通过环境变量 ENABLE_TOKEN_RATE_LIMIT=true 启用"
echo "  3. 账户层限流通过环境变量 ENABLE_ACCOUNT_RATE_LIMIT=true 启用"
echo ""
echo "完整测试请运行: ./tests/test_rate_limit_improved.sh"
