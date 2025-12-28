#!/bin/bash

# ========================================
# Bearer Token Service V2 - API 测试脚本
# ========================================

set -e  # 遇到错误立即退出

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
BASE_URL="${BASE_URL:-http://localhost:8080}"
TEST_EMAIL="test$(date +%s)@example.com"
TEST_COMPANY="Test Company"
TEST_PASSWORD="testPassword123"

# 临时文件存储响应
RESPONSE_FILE=$(mktemp)
trap "rm -f $RESPONSE_FILE" EXIT

# ========================================
# 辅助函数
# ========================================

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
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

# ========================================
# 主要测试流程
# ========================================

main() {
    log_info "Starting Bearer Token Service V2 API Tests"
    log_info "Base URL: $BASE_URL"
    echo ""

    # 0. 健康检查
    test_step "0. Health Check"
    test_health_check

    # 1. 注册账户
    test_step "1. Register Account"
    test_register_account

    # 2. 获取账户信息
    test_step "2. Get Account Info"
    test_get_account_info

    # 3. 创建 Token（不同 Scope）
    test_step "3. Create Tokens with Different Scopes"
    test_create_tokens

    # 4. 列出 Tokens
    test_step "4. List Tokens"
    test_list_tokens

    # 5. 获取 Token 详情
    test_step "5. Get Token Info"
    test_get_token_info

    # 6. 验证 Bearer Token
    test_step "6. Validate Bearer Token"
    test_validate_token

    # 7. 验证 Scope 权限
    test_step "7. Validate Token with Scope"
    test_validate_token_with_scope

    # 8. 更新 Token 状态
    test_step "8. Update Token Status"
    test_update_token_status

    # 9. 获取 Token 统计
    test_step "9. Get Token Stats"
    test_get_token_stats

    # 10. 重新生成 SecretKey
    test_step "10. Regenerate Secret Key"
    test_regenerate_secret_key

    # 11. 删除 Token
    test_step "11. Delete Token"
    test_delete_token

    # ========================================
    # Qstub 认证测试
    # ========================================

    # 12. Qstub 认证 - 创建账户
    test_step "12. Qstub Authentication - Create Account"
    test_qstub_create_account

    # 13. Qstub 认证 - 创建 Token
    test_step "13. Qstub Authentication - Create Token"
    test_qstub_create_token

    # 14. Qstub 认证 - 列出 Tokens
    test_step "14. Qstub Authentication - List Tokens"
    test_qstub_list_tokens

    # 15. Qstub 认证 - 获取 Token 详情
    test_step "15. Qstub Authentication - Get Token Info"
    test_qstub_get_token

    # 16. 验证 Account ID 映射（在删除前验证）
    test_step "16. Verify Qstub Account ID Mapping"
    test_qstub_account_mapping

    # 17. Qstub 认证 - 更新 Token
    test_step "17. Qstub Authentication - Update Token Status"
    test_qstub_update_token

    # 18. Qstub 认证 - 删除 Token
    test_step "18. Qstub Authentication - Delete Token"
    test_qstub_delete_token

    # 测试总结
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}🎉 All Tests Passed!${NC}"
    echo -e "${GREEN}  - HMAC Authentication ✓${NC}"
    echo -e "${GREEN}  - Qstub Authentication ✓${NC}"
    echo -e "${GREEN}========================================${NC}"
}

# ========================================
# 测试用例
# ========================================

test_health_check() {
    log_info "Testing health check endpoint..."

    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/health")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "200" ]; then
        log_success "Health check passed: $body"
    else
        log_error "Health check failed with status $http_code"
        exit 1
    fi
}

test_register_account() {
    log_info "Registering new account: $TEST_EMAIL"

    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v2/accounts/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$TEST_EMAIL\",
            \"company\": \"$TEST_COMPANY\",
            \"password\": \"$TEST_PASSWORD\"
        }")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "201" ]; then
        # 提取 AccessKey 和 SecretKey
        ACCESS_KEY=$(echo "$body" | grep -o '"access_key":"[^"]*"' | cut -d'"' -f4)
        SECRET_KEY=$(echo "$body" | grep -o '"secret_key":"[^"]*"' | cut -d'"' -f4)
        ACCOUNT_ID=$(echo "$body" | grep -o '"account_id":"[^"]*"' | cut -d'"' -f4)

        # 导出为全局变量
        export ACCESS_KEY
        export SECRET_KEY
        export ACCOUNT_ID

        log_success "Account registered successfully"
        log_info "Account ID: $ACCOUNT_ID"
        log_info "Access Key: $ACCESS_KEY"
        log_info "Secret Key: ${SECRET_KEY:0:20}*** (hidden)"

        # 保存到文件供后续测试使用
        echo "ACCESS_KEY=$ACCESS_KEY" > /tmp/v2_test_credentials.env
        echo "SECRET_KEY=$SECRET_KEY" >> /tmp/v2_test_credentials.env
        echo "ACCOUNT_ID=$ACCOUNT_ID" >> /tmp/v2_test_credentials.env
    else
        log_error "Account registration failed with status $http_code"
        echo "$body"
        exit 1
    fi
}

test_get_account_info() {
    log_info "Getting account information..."

    # 使用 HMAC 签名（需要 Python 脚本辅助）
    response=$(python3 - <<EOF
import sys
import requests
import hmac
import hashlib
import base64
from datetime import datetime

access_key = "$ACCESS_KEY"
secret_key = "$SECRET_KEY"
method = "GET"
uri = "/api/v2/accounts/me"
timestamp = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
body = ""

string_to_sign = f"{method}\n{uri}\n{timestamp}\n{body}"
signature = base64.b64encode(
    hmac.new(secret_key.encode(), string_to_sign.encode(), hashlib.sha256).digest()
).decode()

headers = {
    "Authorization": f"QINIU {access_key}:{signature}",
    "X-Qiniu-Date": timestamp
}

r = requests.get("$BASE_URL" + uri, headers=headers)
print(r.status_code)
print(r.text)
EOF
)

    http_code=$(echo "$response" | head -n1)
    body=$(echo "$response" | tail -n +2)

    if [ "$http_code" = "200" ]; then
        log_success "Account info retrieved successfully"
        echo "$body" | python3 -m json.tool
    else
        log_error "Failed to get account info with status $http_code"
        echo "$body"
        exit 1
    fi
}

test_create_tokens() {
    log_info "Creating tokens with different scopes..."

    # Token 1: Read-only
    TOKEN1=$(python3 ./hmac_client.py create_token \
        "$ACCESS_KEY" "$SECRET_KEY" \
        "Read-only token" '["storage:read"]' 90)

    export TOKEN1
    log_success "Token 1 created (Read-only)"
    echo "$TOKEN1" | grep -o '"token":"[^"]*"' | cut -d'"' -f4 | head -c 30
    echo "***"

    # Token 2: Full permissions
    TOKEN2=$(python3 ./hmac_client.py create_token \
        "$ACCESS_KEY" "$SECRET_KEY" \
        "Full permissions token" '["storage:*","cdn:*"]' 180)

    export TOKEN2
    log_success "Token 2 created (Full permissions)"

    # Token 3: Admin
    TOKEN3=$(python3 ./hmac_client.py create_token \
        "$ACCESS_KEY" "$SECRET_KEY" \
        "Admin token" '["*"]' 365)

    export TOKEN3
    export TOKEN3_ID=$(echo "$TOKEN3" | grep -o '"token_id":"[^"]*"' | cut -d'"' -f4)
    log_success "Token 3 created (Admin)"

    # 保存 Token 值
    TOKEN3_VALUE=$(echo "$TOKEN3" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    export TOKEN3_VALUE
    echo "TOKEN3_VALUE=$TOKEN3_VALUE" >> /tmp/v2_test_credentials.env
    echo "TOKEN3_ID=$TOKEN3_ID" >> /tmp/v2_test_credentials.env

    # Token 4: Custom prefix
    log_info "Creating token with custom prefix..."
    TOKEN4=$(python3 ./hmac_client.py create_token \
        "$ACCESS_KEY" "$SECRET_KEY" \
        "Custom prefix token" '["storage:read"]' 90 "custom_bearer_")

    export TOKEN4
    TOKEN4_VALUE=$(echo "$TOKEN4" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    export TOKEN4_VALUE

    # 验证前缀
    if echo "$TOKEN4_VALUE" | grep -q "^custom_bearer_"; then
        log_success "Token 4 created with custom prefix: ${TOKEN4_VALUE:0:20}***"
    else
        log_error "Token 4 does not have the expected custom prefix!"
        echo "Expected prefix: custom_bearer_"
        echo "Actual token: $TOKEN4_VALUE"
        exit 1
    fi
}

test_list_tokens() {
    log_info "Listing all tokens..."

    response=$(python3 ./hmac_client.py list_tokens \
        "$ACCESS_KEY" "$SECRET_KEY")

    if echo "$response" | grep -q '"tokens"'; then
        log_success "Tokens listed successfully"
        echo "$response" | python3 -m json.tool
    else
        log_error "Failed to list tokens"
        exit 1
    fi
}

test_get_token_info() {
    log_info "Getting token info for Token 3..."

    response=$(python3 ./hmac_client.py get_token \
        "$ACCESS_KEY" "$SECRET_KEY" "$TOKEN3_ID")

    if echo "$response" | grep -q '"token_id"'; then
        log_success "Token info retrieved successfully"
        echo "$response" | python3 -m json.tool
    else
        log_error "Failed to get token info"
        exit 1
    fi
}

test_validate_token() {
    log_info "Validating Bearer Token..."

    response=$(curl -s -X POST "$BASE_URL/api/v2/validate" \
        -H "Authorization: Bearer $TOKEN3_VALUE" \
        -H "Content-Type: application/json")

    if echo "$response" | grep -q '"valid":true'; then
        log_success "Token validation passed"
        echo "$response" | python3 -m json.tool
    else
        log_error "Token validation failed"
        echo "$response"
        exit 1
    fi
}

test_validate_token_with_scope() {
    log_info "Validating Token with required scope..."

    # 测试有权限的 Scope
    response=$(curl -s -X POST "$BASE_URL/api/v2/validate" \
        -H "Authorization: Bearer $TOKEN3_VALUE" \
        -H "Content-Type: application/json" \
        -d '{"required_scope": "storage:read"}')

    if echo "$response" | grep -q '"granted":true'; then
        log_success "Scope validation passed (storage:read)"
    else
        log_error "Scope validation failed"
        echo "$response"
        exit 1
    fi
}

test_update_token_status() {
    log_info "Disabling Token 3..."

    response=$(python3 ./hmac_client.py update_token_status \
        "$ACCESS_KEY" "$SECRET_KEY" "$TOKEN3_ID" false)

    if echo "$response" | grep -q "success"; then
        log_success "Token disabled successfully"
    else
        log_error "Failed to disable token"
        exit 1
    fi

    # 重新启用
    log_info "Re-enabling Token 3..."
    python3 ./hmac_client.py update_token_status \
        "$ACCESS_KEY" "$SECRET_KEY" "$TOKEN3_ID" true > /dev/null
    log_success "Token re-enabled"
}

test_get_token_stats() {
    log_info "Getting token statistics..."

    response=$(python3 ./hmac_client.py get_token_stats \
        "$ACCESS_KEY" "$SECRET_KEY" "$TOKEN3_ID")

    if echo "$response" | grep -q '"total_requests"'; then
        log_success "Token stats retrieved successfully"
        echo "$response" | python3 -m json.tool
    else
        log_error "Failed to get token stats"
        exit 1
    fi
}

test_regenerate_secret_key() {
    log_info "Regenerating Secret Key..."
    log_warning "This will invalidate the old Secret Key!"

    response=$(python3 ./hmac_client.py regenerate_sk \
        "$ACCESS_KEY" "$SECRET_KEY")

    if echo "$response" | grep -q '"secret_key"'; then
        NEW_SECRET_KEY=$(echo "$response" | grep -o '"secret_key":"[^"]*"' | cut -d'"' -f4)
        export SECRET_KEY="$NEW_SECRET_KEY"
        log_success "Secret Key regenerated successfully"
        log_info "New Secret Key: ${NEW_SECRET_KEY:0:20}*** (hidden)"
    else
        log_error "Failed to regenerate Secret Key"
        exit 1
    fi
}

test_delete_token() {
    log_info "Deleting Token 3..."

    response=$(python3 ./hmac_client.py delete_token \
        "$ACCESS_KEY" "$SECRET_KEY" "$TOKEN3_ID")

    if echo "$response" | grep -q "success"; then
        log_success "Token deleted successfully"
    else
        log_error "Failed to delete token"
        exit 1
    fi
}

# ========================================
# Qstub 认证测试用例
# ========================================

test_qstub_create_account() {
    log_info "Creating Qstub authentication context..."

    # 构建 Qstub Token
    # Qstub Token 格式: Base64({"uid":"12345","email":"user@qiniu.com","name":"User Name"})
    QSTUB_UID="12345"
    QSTUB_EMAIL="qstub-test@qiniu.com"
    QSTUB_NAME="Qstub Test User"

    USER_INFO="{\"uid\":\"$QSTUB_UID\",\"email\":\"$QSTUB_EMAIL\",\"name\":\"$QSTUB_NAME\"}"
    QSTUB_TOKEN=$(echo -n "$USER_INFO" | base64 -w 0)

    # 导出全局变量
    export QSTUB_TOKEN
    export QSTUB_UID
    export QSTUB_EMAIL

    log_success "Qstub authentication context created"
    log_info "Qstub UID: $QSTUB_UID"
    log_info "Qstub Email: $QSTUB_EMAIL"
    log_info "Qstub Token: ${QSTUB_TOKEN:0:30}..."

    # 保存到文件
    echo "QSTUB_TOKEN=$QSTUB_TOKEN" >> /tmp/v2_test_credentials.env
    echo "QSTUB_UID=$QSTUB_UID" >> /tmp/v2_test_credentials.env
}

test_qstub_create_token() {
    log_info "Creating Bearer Token using Qstub authentication..."

    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v2/tokens" \
        -H "Authorization: Bearer $QSTUB_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "description": "Qstub test token",
            "scope": ["storage:read", "storage:write"],
            "expires_in_seconds": 3600
        }')

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "201" ]; then
        # 提取 Token ID 和值
        QSTUB_TOKEN_ID=$(echo "$body" | grep -o '"token_id":"[^"]*"' | cut -d'"' -f4)
        QSTUB_BEARER_TOKEN=$(echo "$body" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        QSTUB_ACCOUNT_ID=$(echo "$body" | grep -o '"account_id":"[^"]*"' | cut -d'"' -f4)

        export QSTUB_TOKEN_ID
        export QSTUB_BEARER_TOKEN
        export QSTUB_ACCOUNT_ID

        log_success "Token created via Qstub authentication"
        log_info "Token ID: $QSTUB_TOKEN_ID"
        log_info "Account ID: $QSTUB_ACCOUNT_ID"
        log_info "Bearer Token: ${QSTUB_BEARER_TOKEN:0:30}..."

        # 保存到文件
        echo "QSTUB_TOKEN_ID=$QSTUB_TOKEN_ID" >> /tmp/v2_test_credentials.env
        echo "QSTUB_BEARER_TOKEN=$QSTUB_BEARER_TOKEN" >> /tmp/v2_test_credentials.env
        echo "QSTUB_ACCOUNT_ID=$QSTUB_ACCOUNT_ID" >> /tmp/v2_test_credentials.env
    else
        log_error "Failed to create token via Qstub with status $http_code"
        echo "$body"
        exit 1
    fi
}

test_qstub_list_tokens() {
    log_info "Listing tokens using Qstub authentication..."

    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v2/tokens?limit=10" \
        -H "Authorization: Bearer $QSTUB_TOKEN")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "200" ]; then
        log_success "Tokens listed successfully via Qstub"
        TOKEN_COUNT=$(echo "$body" | grep -o '"total":[0-9]*' | cut -d':' -f2)
        log_info "Total tokens: $TOKEN_COUNT"
        echo "$body" | python3 -m json.tool 2>/dev/null || echo "$body"
    else
        log_error "Failed to list tokens via Qstub with status $http_code"
        echo "$body"
        exit 1
    fi
}

test_qstub_get_token() {
    log_info "Getting token info using Qstub authentication..."

    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/v2/tokens/$QSTUB_TOKEN_ID" \
        -H "Authorization: Bearer $QSTUB_TOKEN")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "200" ]; then
        log_success "Token info retrieved via Qstub"
        echo "$body" | python3 -m json.tool 2>/dev/null || echo "$body"
    else
        log_error "Failed to get token info via Qstub with status $http_code"
        echo "$body"
        exit 1
    fi
}

test_qstub_update_token() {
    log_info "Updating token status using Qstub authentication..."

    # 禁用 Token
    response=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/api/v2/tokens/$QSTUB_TOKEN_ID/status" \
        -H "Authorization: Bearer $QSTUB_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"enabled": false}')

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "200" ]; then
        log_success "Token disabled via Qstub"

        # 重新启用
        log_info "Re-enabling token..."
        response=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/api/v2/tokens/$QSTUB_TOKEN_ID/status" \
            -H "Authorization: Bearer $QSTUB_TOKEN" \
            -H "Content-Type: application/json" \
            -d '{"enabled": true}')

        http_code=$(echo "$response" | tail -n1)
        if [ "$http_code" = "200" ]; then
            log_success "Token re-enabled via Qstub"
        else
            log_error "Failed to re-enable token"
            exit 1
        fi
    else
        log_error "Failed to update token status via Qstub with status $http_code"
        echo "$body"
        exit 1
    fi
}

test_qstub_delete_token() {
    log_info "Deleting token using Qstub authentication..."

    response=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/api/v2/tokens/$QSTUB_TOKEN_ID" \
        -H "Authorization: Bearer $QSTUB_TOKEN")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "200" ]; then
        log_success "Token deleted via Qstub"
    else
        log_error "Failed to delete token via Qstub with status $http_code"
        echo "$body"
        exit 1
    fi
}

test_qstub_account_mapping() {
    log_info "Verifying Qstub Account ID mapping..."

    # 验证 account_id 格式应该是 qiniu_{uid}
    EXPECTED_ACCOUNT_ID="qiniu_$QSTUB_UID"

    if [ "$QSTUB_ACCOUNT_ID" = "$EXPECTED_ACCOUNT_ID" ]; then
        log_success "Account ID mapping is correct"
        log_info "Expected: $EXPECTED_ACCOUNT_ID"
        log_info "Actual:   $QSTUB_ACCOUNT_ID"
    else
        log_error "Account ID mapping mismatch!"
        log_info "Expected: $EXPECTED_ACCOUNT_ID"
        log_info "Actual:   $QSTUB_ACCOUNT_ID"
        exit 1
    fi

    # 测试使用生成的 Bearer Token 访问 API
    log_info "Testing Bearer Token created via Qstub..."
    response=$(curl -s -X POST "$BASE_URL/api/v2/validate" \
        -H "Authorization: Bearer $QSTUB_BEARER_TOKEN" \
        -H "Content-Type: application/json")

    if echo "$response" | grep -q '"valid":true'; then
        log_success "Bearer Token validation passed"
        echo "$response" | python3 -m json.tool 2>/dev/null || echo "$response"
    else
        log_error "Bearer Token validation failed"
        echo "$response"
        exit 1
    fi
}

# ========================================
# 运行测试
# ========================================

main "$@"
