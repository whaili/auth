#!/bin/bash

# ========================================
# Bearer Token Service V2 - Redis 缓存功能测试脚本
# ========================================

set -e  # 遇到错误立即退出

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
# 默认端口 8081，可通过环境变量覆盖: BASE_URL=http://localhost:8081 ./test_redis_cache.sh
BASE_URL="${BASE_URL:-http://localhost:8081}"
REDIS_CLI="${REDIS_CLI:-redis-cli}"
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_CONTAINER="${REDIS_CONTAINER:-bearer-token-redis}"

# 测试用的 Qiniu UID
QINIU_UID="${QINIU_UID:-1369077332}"

# 测试计数
TESTS_PASSED=0
TESTS_FAILED=0

# ========================================
# 辅助函数
# ========================================

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

test_step() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

redis_cmd() {
    docker exec "$REDIS_CONTAINER" redis-cli "$@"
}

# ========================================
# 前置检查
# ========================================

check_prerequisites() {
    log_info "Checking prerequisites..."

    # 检查服务是否运行
    local health=$(curl -s "$BASE_URL/health" 2>/dev/null)
    if [[ $health != *"ok"* ]]; then
        log_error "Service is not running at $BASE_URL"
        log_info "Please start the service with REDIS_ENABLED=true"
        exit 1
    fi
    log_success "Service is running"

    # 检查 Redis 是否可用
    local pong
    pong=$(docker exec "$REDIS_CONTAINER" redis-cli PING 2>&1) || true
    if [[ "$pong" != "PONG" ]]; then
        log_error "Redis is not available: $pong"
        log_info "Please start Redis: docker run -d --name bearer-token-redis -p 6379:6379 redis:7.2-alpine"
        exit 1
    fi
    log_success "Redis is available"
}

# ========================================
# 测试函数
# ========================================

# 1. 测试创建 Token 后不写入缓存
test_create_token_no_cache() {
    log_info "Creating token and checking cache is empty..."

    # 清空缓存
    redis_cmd FLUSHALL > /dev/null

    # 创建 Token
    local response=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
        -H "Authorization: QiniuStub uid=${QINIU_UID}&ut=1" \
        -H "Content-Type: application/json" \
        -d '{"description":"Cache test token","expires_in_seconds":3600}')

    TOKEN_ID=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin)['token_id'])" 2>/dev/null)
    TOKEN_VALUE=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])" 2>/dev/null)

    if [[ -z "$TOKEN_ID" ]]; then
        log_error "Failed to create token: $response"
        return 1
    fi

    log_info "Token created: $TOKEN_ID"

    # 检查缓存应该为空
    local keys=$(redis_cmd KEYS "token:*")
    if [[ -z "$keys" ]]; then
        log_success "Cache is empty after token creation (as expected)"
    else
        log_error "Cache should be empty after creation, but found: $keys"
        return 1
    fi
}

# 2. 测试首次验证写入缓存
test_first_validation_cache_write() {
    log_info "Testing first validation writes to cache..."

    # 验证 Token
    local response=$(curl -s -X POST "$BASE_URL/api/v2/validate" \
        -H "Authorization: Bearer $TOKEN_VALUE")

    local valid=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('valid', False))" 2>/dev/null)

    if [[ "$valid" != "True" ]]; then
        log_error "Token validation failed: $response"
        return 1
    fi

    # 等待异步缓存写入
    sleep 0.5

    # 检查缓存已写入
    local cache_key="token:val:$TOKEN_VALUE"
    local cached=$(redis_cmd GET "$cache_key")

    if [[ -n "$cached" ]]; then
        log_success "Cache written after first validation"
        log_info "Cache key: token:val:${TOKEN_VALUE:0:20}..."
    else
        log_error "Cache not written after validation"
        return 1
    fi
}

# 3. 测试缓存命中（响应时间比较）
test_cache_hit_performance() {
    log_info "Testing cache hit performance..."

    # 第一次验证（可能有缓存）
    local start1=$(date +%s%N)
    curl -s -X POST "$BASE_URL/api/v2/validate" \
        -H "Authorization: Bearer $TOKEN_VALUE" > /dev/null
    local end1=$(date +%s%N)
    local time1=$(( (end1 - start1) / 1000000 ))

    # 第二次验证（一定命中缓存）
    local start2=$(date +%s%N)
    curl -s -X POST "$BASE_URL/api/v2/validate" \
        -H "Authorization: Bearer $TOKEN_VALUE" > /dev/null
    local end2=$(date +%s%N)
    local time2=$(( (end2 - start2) / 1000000 ))

    # 第三次验证
    local start3=$(date +%s%N)
    curl -s -X POST "$BASE_URL/api/v2/validate" \
        -H "Authorization: Bearer $TOKEN_VALUE" > /dev/null
    local end3=$(date +%s%N)
    local time3=$(( (end3 - start3) / 1000000 ))

    log_info "Response times: ${time1}ms, ${time2}ms, ${time3}ms"

    # 缓存命中的响应应该很快（通常 < 10ms）
    if [[ $time2 -lt 50 && $time3 -lt 50 ]]; then
        log_success "Cache hit performance is good (< 50ms)"
    else
        log_warning "Cache hit response time higher than expected"
    fi
}

# 4. 测试禁用 Token 后缓存失效
test_disable_token_cache_invalidation() {
    log_info "Testing cache invalidation when token is disabled..."

    # 确认缓存存在
    local cache_key="token:val:$TOKEN_VALUE"
    local cached_before=$(redis_cmd GET "$cache_key")

    if [[ -z "$cached_before" ]]; then
        log_warning "Cache was empty before test, triggering validation first..."
        curl -s -X POST "$BASE_URL/api/v2/validate" \
            -H "Authorization: Bearer $TOKEN_VALUE" > /dev/null
        sleep 0.5
    fi

    # 禁用 Token
    local response=$(curl -s -X PUT "$BASE_URL/api/v2/tokens/$TOKEN_ID/status" \
        -H "Authorization: QiniuStub uid=${QINIU_UID}&ut=1" \
        -H "Content-Type: application/json" \
        -d '{"is_active": false}')

    log_info "Token disabled: $response"

    # 检查缓存已被清除
    local cached_after=$(redis_cmd GET "$cache_key")

    if [[ -z "$cached_after" ]]; then
        log_success "Cache invalidated after token disabled"
    else
        log_error "Cache should be cleared after token disabled"
        return 1
    fi

    # 验证 Token 应该失败
    local validate_response=$(curl -s -X POST "$BASE_URL/api/v2/validate" \
        -H "Authorization: Bearer $TOKEN_VALUE")

    local valid=$(echo $validate_response | python3 -c "import sys, json; print(json.load(sys.stdin).get('valid', False))" 2>/dev/null)

    if [[ "$valid" == "False" ]]; then
        log_success "Disabled token validation returns invalid"
    else
        log_error "Disabled token should return invalid: $validate_response"
        return 1
    fi
}

# 5. 测试重新启用 Token 后缓存更新
test_enable_token_cache_update() {
    log_info "Testing cache update when token is re-enabled..."

    # 重新启用 Token
    curl -s -X PUT "$BASE_URL/api/v2/tokens/$TOKEN_ID/status" \
        -H "Authorization: QiniuStub uid=${QINIU_UID}&ut=1" \
        -H "Content-Type: application/json" \
        -d '{"is_active": true}' > /dev/null

    log_info "Token re-enabled"

    # 验证 Token 应该成功
    local response=$(curl -s -X POST "$BASE_URL/api/v2/validate" \
        -H "Authorization: Bearer $TOKEN_VALUE")

    local valid=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('valid', False))" 2>/dev/null)

    if [[ "$valid" == "True" ]]; then
        log_success "Re-enabled token validation returns valid"
    else
        log_error "Re-enabled token should return valid: $response"
        return 1
    fi

    # 等待缓存写入
    sleep 0.5

    # 检查新缓存已写入
    local cache_key="token:val:$TOKEN_VALUE"
    local cached=$(redis_cmd GET "$cache_key")

    if [[ -n "$cached" ]]; then
        log_success "New cache written after token re-enabled"
    else
        log_warning "Cache not written after re-enabling (may be async)"
    fi
}

# 6. 测试删除 Token 后缓存失效
test_delete_token_cache_invalidation() {
    log_info "Testing cache invalidation when token is deleted..."

    # 删除 Token
    curl -s -X DELETE "$BASE_URL/api/v2/tokens/$TOKEN_ID" \
        -H "Authorization: QiniuStub uid=${QINIU_UID}&ut=1" > /dev/null

    log_info "Token deleted"

    # 检查缓存已被清除
    local cache_key="token:val:$TOKEN_VALUE"
    local cached=$(redis_cmd GET "$cache_key")

    if [[ -z "$cached" ]]; then
        log_success "Cache invalidated after token deleted"
    else
        log_error "Cache should be cleared after token deleted"
        return 1
    fi

    # 验证 Token 应该失败
    local response=$(curl -s -X POST "$BASE_URL/api/v2/validate" \
        -H "Authorization: Bearer $TOKEN_VALUE")

    local valid=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('valid', False))" 2>/dev/null)

    if [[ "$valid" == "False" ]]; then
        log_success "Deleted token validation returns invalid"
    else
        log_error "Deleted token should return invalid: $response"
        return 1
    fi
}

# 7. 测试空对象缓存（防穿透）
test_null_cache_penetration_protection() {
    log_info "Testing null cache for penetration protection..."

    # 清空缓存
    redis_cmd FLUSHALL > /dev/null

    # 使用不存在的 Token 验证
    local fake_token="sk-nonexistent1234567890abcdef1234567890abcdef1234567890abcdef12345678"

    local response=$(curl -s -X POST "$BASE_URL/api/v2/validate" \
        -H "Authorization: Bearer $fake_token")

    local valid=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('valid', False))" 2>/dev/null)

    if [[ "$valid" == "False" ]]; then
        log_info "Non-existent token correctly returns invalid"
    fi

    # 等待缓存写入
    sleep 0.5

    # 检查空对象缓存
    local cache_key="token:val:$fake_token"
    local cached=$(redis_cmd GET "$cache_key")

    if [[ "$cached" == "null" ]]; then
        log_success "Null cache written for non-existent token (penetration protection)"
    else
        log_warning "Null cache not found (might be disabled or async)"
    fi
}

# 8. 测试缓存 TTL
test_cache_ttl() {
    log_info "Testing cache TTL..."

    # 创建新 Token
    local response=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
        -H "Authorization: QiniuStub uid=${QINIU_UID}&ut=1" \
        -H "Content-Type: application/json" \
        -d '{"description":"TTL test token","expires_in_seconds":3600}')

    local token_id=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin)['token_id'])" 2>/dev/null)
    local token_value=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])" 2>/dev/null)

    # 验证触发缓存写入
    curl -s -X POST "$BASE_URL/api/v2/validate" \
        -H "Authorization: Bearer $token_value" > /dev/null

    sleep 0.5

    # 检查 TTL
    local cache_key="token:val:$token_value"
    local ttl=$(redis_cmd TTL "$cache_key")

    if [[ $ttl -gt 0 ]]; then
        log_success "Cache TTL is set: ${ttl}s"
        # TTL 应该在 4.5-5.5 分钟之间（5分钟 ± 10% 抖动）
        if [[ $ttl -ge 270 && $ttl -le 330 ]]; then
            log_success "TTL is within expected range (270-330s)"
        else
            log_warning "TTL is outside expected range: ${ttl}s"
        fi
    else
        log_error "Cache TTL is not set"
    fi

    # 清理
    curl -s -X DELETE "$BASE_URL/api/v2/tokens/$token_id" \
        -H "Authorization: QiniuStub uid=${QINIU_UID}&ut=1" > /dev/null
}

# ========================================
# 主测试流程
# ========================================

main() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  Redis 缓存功能测试${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    log_info "Base URL: $BASE_URL"
    log_info "Redis: $REDIS_HOST:$REDIS_PORT"
    echo ""

    # 前置检查
    test_step "0. Prerequisites Check"
    check_prerequisites

    # 1. 创建 Token 后不写入缓存
    test_step "1. Create Token (No Cache Write)"
    test_create_token_no_cache

    # 2. 首次验证写入缓存
    test_step "2. First Validation (Cache Write)"
    test_first_validation_cache_write

    # 3. 缓存命中性能
    test_step "3. Cache Hit Performance"
    test_cache_hit_performance

    # 4. 禁用 Token 缓存失效
    test_step "4. Disable Token (Cache Invalidation)"
    test_disable_token_cache_invalidation

    # 5. 重新启用 Token
    test_step "5. Re-enable Token (Cache Update)"
    test_enable_token_cache_update

    # 6. 删除 Token 缓存失效
    test_step "6. Delete Token (Cache Invalidation)"
    test_delete_token_cache_invalidation

    # 7. 空对象缓存防穿透
    test_step "7. Null Cache (Penetration Protection)"
    test_null_cache_penetration_protection

    # 8. 缓存 TTL 测试
    test_step "8. Cache TTL"
    test_cache_ttl

    # 测试结果
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  测试结果${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo -e "${GREEN}  通过: $TESTS_PASSED${NC}"
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "${RED}  失败: $TESTS_FAILED${NC}"
        echo -e "${RED}========================================${NC}"
        exit 1
    else
        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}🎉 All Redis Cache Tests Passed!${NC}"
        echo -e "${GREEN}========================================${NC}"
    fi
}

# 运行主测试
main
