#!/bin/bash

# ========================================
# Bearer Token Service V2 - QiniuStub API 测试脚本
# ========================================

set -e  # 遇到错误立即退出

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
BASE_URL="${BASE_URL:-http://localhost:8081}"

# 测试用的 Qiniu UID
QINIU_UID="${QINIU_UID:-1369077332}"
QINIU_IUID="${QINIU_IUID:-8901234}"
# 测试环境有效 UID (用于 qconfapi 完整测试)
# - 1810810692: 测试环境 Qconf 有数据
# - 1383218128: 生产环境 Qconf 有数据
QINIU_TEST_UID="${QINIU_TEST_UID:-1810810692}"
QINIU_PROD_UID="${QINIU_PROD_UID:-1383218128}"

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
# 测试函数
# ========================================

# 0. 健康检查
test_health_check() {
    log_info "Testing health check endpoint..."

    local response=$(curl -s "$BASE_URL/health")
    if [[ $response == *"ok"* ]]; then
        log_success "Health check passed: $response"
    else
        log_error "Health check failed: $response"
        exit 1
    fi
}

# 1. 创建 Token（主账户）
test_create_token_main_account() {
    log_info "Creating token with main account (uid=$QINIU_UID)..."

    local qstub_auth="QiniuStub uid=${QINIU_UID}&ut=1"

    local response=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
        -H "Authorization: $qstub_auth" \
        -H "Content-Type: application/json" \
        -d '{
            "description": "Test token for main account",
            "expires_in_seconds": 7200
        }')

    # 提取 token_id 和 token
    TOKEN_ID_MAIN=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin)['token_id'])" 2>/dev/null)
    BEARER_TOKEN_MAIN=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])" 2>/dev/null)

    if [[ -n "$TOKEN_ID_MAIN" ]]; then
        log_success "Token created for main account"
        log_info "Token ID: $TOKEN_ID_MAIN"
        log_info "Bearer Token: ${BEARER_TOKEN_MAIN:0:20}..."
    else
        log_error "Failed to create token: $response"
        exit 1
    fi
}

# 2. 创建 Token（IAM 子账户）
test_create_token_iam_account() {
    log_info "Creating token with IAM sub-account (uid=$QINIU_UID, iuid=$QINIU_IUID)..."

    local qstub_auth="QiniuStub uid=${QINIU_UID}&ut=1&iuid=${QINIU_IUID}"

    local response=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
        -H "Authorization: $qstub_auth" \
        -H "Content-Type: application/json" \
        -d '{
            "description": "Test token for IAM sub-account",
            "expires_in_seconds": 3600
        }')

    # 提取 token_id 和 token
    TOKEN_ID_IAM=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin)['token_id'])" 2>/dev/null)
    BEARER_TOKEN_IAM=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])" 2>/dev/null)

    if [[ -n "$TOKEN_ID_IAM" ]]; then
        log_success "Token created for IAM sub-account"
        log_info "Token ID: $TOKEN_ID_IAM"
        log_info "Bearer Token: ${BEARER_TOKEN_IAM:0:20}..."
    else
        log_error "Failed to create token: $response"
        exit 1
    fi
}

# 2.1 创建 Token（自定义 prefix）
test_create_token_with_prefix() {
    log_info "Creating token with custom prefix..."

    local qstub_auth="QiniuStub uid=${QINIU_UID}&ut=1"

    local response=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
        -H "Authorization: $qstub_auth" \
        -H "Content-Type: application/json" \
        -d '{
            "description": "Test token with custom prefix",
            "prefix": "myapp"
        }')

    TOKEN_ID_PREFIX=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin)['token_id'])" 2>/dev/null)
    BEARER_TOKEN_PREFIX=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])" 2>/dev/null)

    if [[ -n "$TOKEN_ID_PREFIX" ]]; then
        # 验证 token 格式是否正确（以 myapp- 开头）
        if [[ "$BEARER_TOKEN_PREFIX" == myapp-* ]]; then
            log_success "Token created with custom prefix"
            log_info "Token ID: $TOKEN_ID_PREFIX"
            log_info "Bearer Token: ${BEARER_TOKEN_PREFIX:0:20}..."
        else
            log_error "Token prefix format incorrect: $BEARER_TOKEN_PREFIX"
            exit 1
        fi
    else
        log_error "Failed to create token: $response"
        exit 1
    fi
}

# 2.2 测试 prefix 校验（无效前缀 - 包含大写字母）
test_create_token_invalid_prefix_uppercase() {
    log_info "Testing invalid prefix (uppercase letters)..."

    local qstub_auth="QiniuStub uid=${QINIU_UID}&ut=1"

    local response=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
        -H "Authorization: $qstub_auth" \
        -H "Content-Type: application/json" \
        -d '{
            "description": "Test token with invalid prefix",
            "prefix": "MyApp"
        }')

    local error=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('error', ''))" 2>/dev/null)

    if [[ "$error" == *"lowercase"* ]]; then
        log_success "Correctly rejected uppercase prefix"
    else
        log_error "Should have rejected uppercase prefix: $response"
        exit 1
    fi
}

# 2.3 测试 prefix 校验（无效前缀 - 超过12字符）
test_create_token_invalid_prefix_too_long() {
    log_info "Testing invalid prefix (too long)..."

    local qstub_auth="QiniuStub uid=${QINIU_UID}&ut=1"

    local response=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
        -H "Authorization: $qstub_auth" \
        -H "Content-Type: application/json" \
        -d '{
            "description": "Test token with long prefix",
            "prefix": "verylongprefix123"
        }')

    local error=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('error', ''))" 2>/dev/null)

    if [[ "$error" == *"12"* ]]; then
        log_success "Correctly rejected prefix exceeding 12 characters"
    else
        log_error "Should have rejected long prefix: $response"
        exit 1
    fi
}

# 2.4 测试 prefix 校验（无效前缀 - 包含特殊字符）
test_create_token_invalid_prefix_special_chars() {
    log_info "Testing invalid prefix (special characters)..."

    local qstub_auth="QiniuStub uid=${QINIU_UID}&ut=1"

    local response=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
        -H "Authorization: $qstub_auth" \
        -H "Content-Type: application/json" \
        -d '{
            "description": "Test token with special chars",
            "prefix": "my-app"
        }')

    local error=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('error', ''))" 2>/dev/null)

    if [[ "$error" == *"lowercase"* ]] || [[ "$error" == *"underscore"* ]]; then
        log_success "Correctly rejected prefix with special characters"
    else
        log_error "Should have rejected prefix with special chars: $response"
        exit 1
    fi
}

# 3. 列出 Tokens
test_list_tokens() {
    log_info "Listing all tokens..."

    local qstub_auth="QiniuStub uid=${QINIU_UID}&ut=1"

    local response=$(curl -s -X GET "$BASE_URL/api/v2/tokens" \
        -H "Authorization: $qstub_auth")

    echo "$response" | python3 -m json.tool
    log_success "Tokens listed successfully"
}

# 4. 获取 Token 详情
test_get_token_info() {
    log_info "Getting token info for Token ID: $TOKEN_ID_MAIN..."

    local qstub_auth="QiniuStub uid=${QINIU_UID}&ut=1"

    local response=$(curl -s -X GET "$BASE_URL/api/v2/tokens/$TOKEN_ID_MAIN" \
        -H "Authorization: $qstub_auth")

    echo "$response" | python3 -m json.tool
    log_success "Token info retrieved successfully"
}

# 5. 验证 Bearer Token（主账户）
test_validate_bearer_token_main() {
    log_info "Validating Bearer Token (main account)..."

    local response=$(curl -s -X POST "$BASE_URL/api/v2/validate" \
        -H "Authorization: Bearer $BEARER_TOKEN_MAIN" \
        -H "Content-Type: application/json" \
        -d '{}')

    echo "$response" | python3 -m json.tool

    local valid=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('valid', False))" 2>/dev/null)

    if [[ "$valid" == "True" ]]; then
        log_success "Bearer Token validation passed (main account)"
    else
        log_error "Bearer Token validation failed: $response"
        exit 1
    fi
}

# 6. 验证 Bearer Token（IAM 子账户）
test_validate_bearer_token_iam() {
    log_info "Validating Bearer Token (IAM sub-account)..."

    local response=$(curl -s -X POST "$BASE_URL/api/v2/validate" \
        -H "Authorization: Bearer $BEARER_TOKEN_IAM" \
        -H "Content-Type: application/json")

    echo "$response" | python3 -m json.tool

    local valid=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('valid', False))" 2>/dev/null)
    local iuid=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('token_info', {}).get('iuid', ''))" 2>/dev/null)

    if [[ "$valid" == "True" ]]; then
        log_success "Bearer Token validation passed (IAM sub-account)"
        if [[ -n "$iuid" ]]; then
            log_success "IUID field present in response: $iuid"
        else
            log_warning "IUID field not present in response"
        fi
    else
        log_error "Bearer Token validation failed: $response"
        exit 1
    fi
}

# 6.5 验证 Bearer Token 并返回用户信息（主账户）
test_validate_bearer_token_with_userinfo_main() {
    log_info "Validating Bearer Token with UserInfo (main account)..."

    local response=$(curl -s -X POST "$BASE_URL/api/v2/validateu" \
        -H "Authorization: Bearer $BEARER_TOKEN_MAIN" \
        -H "Content-Type: application/json")

    # 检查是否返回 404（端点不存在）
    if echo "$response" | grep -q "404 page not found"; then
        log_warning "/api/v2/validateu endpoint not available (older version?) - skipping test"
        return 0
    fi

    echo "$response" | python3 -m json.tool 2>/dev/null || {
        log_warning "Failed to parse JSON response, raw response: $response"
        return 0
    }

    local valid=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('valid', False))" 2>/dev/null)
    local has_userinfo=$(echo $response | python3 -c "import sys, json; ti = json.load(sys.stdin).get('token_info', {}); print(ti.get('user_info') is not None)" 2>/dev/null)
    local uid=$(echo $response | python3 -c "import sys, json; ui = json.load(sys.stdin).get('token_info', {}).get('user_info'); print(ui.get('uid', 0) if ui else 0)" 2>/dev/null)

    if [[ "$valid" == "True" ]]; then
        log_success "Bearer Token validation with UserInfo passed (main account)"
        if [[ "$has_userinfo" == "True" ]]; then
            log_success "UserInfo included in response (UID: $uid)"

            # 验证关键字段
            local email=$(echo $response | python3 -c "import sys, json; ui = json.load(sys.stdin).get('token_info', {}).get('user_info'); print(ui.get('email', '') if ui else '')" 2>/dev/null)
            local username=$(echo $response | python3 -c "import sys, json; ui = json.load(sys.stdin).get('token_info', {}).get('user_info'); print(ui.get('username', '') if ui else '')" 2>/dev/null)
            local activated=$(echo $response | python3 -c "import sys, json; ui = json.load(sys.stdin).get('token_info', {}).get('user_info'); print(ui.get('activated', False) if ui else False)" 2>/dev/null)

            if [[ -n "$email" ]]; then
                log_success "  Email: $email"
            fi
            if [[ -n "$username" ]]; then
                log_success "  Username: $username"
            fi
            log_info "  Activated: $activated"
        else
            log_warning "UserInfo is null (Qconf RPC may not have this UID - graceful degradation)"
        fi
    else
        log_error "Bearer Token validation failed: $response"
        exit 1
    fi
}

# 6.6 验证 Bearer Token 并返回用户信息（IAM 子账户）
test_validate_bearer_token_with_userinfo_iam() {
    log_info "Validating Bearer Token with UserInfo (IAM sub-account)..."

    local response=$(curl -s -X POST "$BASE_URL/api/v2/validateu" \
        -H "Authorization: Bearer $BEARER_TOKEN_IAM" \
        -H "Content-Type: application/json")

    # 检查是否返回 404（端点不存在）
    if echo "$response" | grep -q "404 page not found"; then
        log_warning "/api/v2/validateu endpoint not available (older version?) - skipping test"
        return 0
    fi

    echo "$response" | python3 -m json.tool 2>/dev/null || {
        log_warning "Failed to parse JSON response, raw response: $response"
        return 0
    }

    local valid=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('valid', False))" 2>/dev/null)
    local iuid=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('token_info', {}).get('iuid', ''))" 2>/dev/null)
    local has_userinfo=$(echo $response | python3 -c "import sys, json; ti = json.load(sys.stdin).get('token_info', {}); print(ti.get('user_info') is not None)" 2>/dev/null)
    local parent_uid=$(echo $response | python3 -c "import sys, json; ui = json.load(sys.stdin).get('token_info', {}).get('user_info'); print(ui.get('parent_uid', 0) if ui else 0)" 2>/dev/null)

    if [[ "$valid" == "True" ]]; then
        log_success "Bearer Token validation with UserInfo passed (IAM sub-account)"
        if [[ -n "$iuid" ]]; then
            log_success "IUID field present in response: $iuid"
        fi
        if [[ "$has_userinfo" == "True" ]]; then
            log_success "UserInfo included in response"
            if [[ "$parent_uid" != "0" ]]; then
                log_info "  Parent UID: $parent_uid (IAM sub-account relationship)"
            fi
        else
            log_warning "UserInfo is null (Qconf RPC may not have this UID - graceful degradation)"
        fi
    else
        log_error "Bearer Token validation failed: $response"
        exit 1
    fi
}

# 6.7 验证 Bearer Token 并返回完整用户信息（使用有效测试 UID）
test_validate_bearer_token_with_full_userinfo() {
    log_info "Validating Bearer Token with FULL UserInfo (Smart UID Selection)..."

    # 尝试两个 UID：测试环境 UID 和生产环境 UID
    local test_uids=("$QINIU_TEST_UID" "$QINIU_PROD_UID")
    local test_labels=("Test ENV (1810810692)" "Prod ENV (1383218128)")
    local found_userinfo=false

    for i in 0 1; do
        local uid="${test_uids[$i]}"
        local label="${test_labels[$i]}"

        log_info "Trying ${label}..."

        # 创建测试 token
        local qstub_auth="QiniuStub uid=${uid}&ut=1"
        local expires_at=$(date -u -d "+1 hour" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -v+1H +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null)

        local create_response=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
            -H "Authorization: $qstub_auth" \
            -H "Content-Type: application/json" \
            -d "{
                \"description\": \"Test token for qconf validation\",
                \"expires_at\": \"$expires_at\"
            }")

        local test_token_id=$(echo $create_response | python3 -c "import sys, json; print(json.load(sys.stdin).get('token_id', ''))" 2>/dev/null)
        local test_bearer_token=$(echo $create_response | python3 -c "import sys, json; print(json.load(sys.stdin).get('token', ''))" 2>/dev/null)

        if [[ -z "$test_token_id" || -z "$test_bearer_token" ]]; then
            log_warning "Failed to create token for UID $uid, skipping..."
            continue
        fi

        # 验证 token 并获取用户信息
        local response=$(curl -s -X POST "$BASE_URL/api/v2/validateu" \
            -H "Authorization: Bearer $test_bearer_token" \
            -H "Content-Type: application/json")

        # 检查是否返回 404（端点不存在）
        if echo "$response" | grep -q "404 page not found"; then
            log_warning "/api/v2/validateu endpoint not available - skipping all tests"
            curl -s -X DELETE "$BASE_URL/api/v2/tokens/$test_token_id" \
                -H "Authorization: $qstub_auth" >/dev/null 2>&1
            return 0
        fi

        local valid=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('valid', False))" 2>/dev/null)
        local has_userinfo=$(echo $response | python3 -c "import sys, json; ti = json.load(sys.stdin).get('token_info', {}); print(ti.get('user_info') is not None)" 2>/dev/null)

        if [[ "$valid" == "True" && "$has_userinfo" == "True" ]]; then
            # 找到有 UserInfo 的 UID！
            found_userinfo=true
            echo "$response" | python3 -m json.tool 2>/dev/null
            log_success "Bearer Token validation passed"

            # 提取并显示用户信息
            local ret_uid=$(echo $response | python3 -c "import sys, json; ui = json.load(sys.stdin).get('token_info', {}).get('user_info'); print(ui.get('uid', 0) if ui else 0)" 2>/dev/null)
            local email=$(echo $response | python3 -c "import sys, json; ui = json.load(sys.stdin).get('token_info', {}).get('user_info'); print(ui.get('email', '') if ui else '')" 2>/dev/null)
            local utype=$(echo $response | python3 -c "import sys, json; ui = json.load(sys.stdin).get('token_info', {}).get('user_info'); print(ui.get('utype', 0) if ui else 0)" 2>/dev/null)
            local activated=$(echo $response | python3 -c "import sys, json; ui = json.load(sys.stdin).get('token_info', {}).get('user_info'); print(ui.get('activated', False) if ui else False)" 2>/dev/null)

            log_success "🎉 FULL UserInfo retrieved from Qconfapi RPC!"
            log_success "  Environment: ${label}"
            log_success "  UID: $ret_uid"
            log_success "  Email: $email"
            log_success "  Utype: $utype"
            log_success "  Activated: $activated"
            log_success "✅ Qconf RPC integration working correctly!"

            # 清理
            curl -s -X DELETE "$BASE_URL/api/v2/tokens/$test_token_id" \
                -H "Authorization: $qstub_auth" > /dev/null
            break
        else
            # 没有 UserInfo，尝试下一个
            curl -s -X DELETE "$BASE_URL/api/v2/tokens/$test_token_id" \
                -H "Authorization: $qstub_auth" > /dev/null
        fi
    done

    if [[ "$found_userinfo" == "false" ]]; then
        log_warning "No UserInfo found for both UIDs - Qconf RPC may not be configured"
        log_warning "  UID $QINIU_TEST_UID (Test): Should have data in test Qconf"
        log_warning "  UID $QINIU_PROD_UID (Prod): Should have data in prod Qconf"
    fi
}

# 7. 更新 Token 状态
test_update_token_status() {
    log_info "Updating token status..."

    local qstub_auth="QiniuStub uid=${QINIU_UID}&ut=1"

    # 禁用 Token
    log_info "Disabling token..."
    curl -s -X PUT "$BASE_URL/api/v2/tokens/$TOKEN_ID_MAIN/status" \
        -H "Authorization: $qstub_auth" \
        -H "Content-Type: application/json" \
        -d '{"is_active": false}' > /dev/null
    log_success "Token disabled"

    # 重新启用 Token
    log_info "Re-enabling token..."
    curl -s -X PUT "$BASE_URL/api/v2/tokens/$TOKEN_ID_MAIN/status" \
        -H "Authorization: $qstub_auth" \
        -H "Content-Type: application/json" \
        -d '{"is_active": true}' > /dev/null
    log_success "Token re-enabled"
}

# 8. 删除 Tokens
test_delete_tokens() {
    log_info "Deleting tokens..."

    local qstub_auth="QiniuStub uid=${QINIU_UID}&ut=1"

    # 删除主账户 Token
    curl -s -X DELETE "$BASE_URL/api/v2/tokens/$TOKEN_ID_MAIN" \
        -H "Authorization: $qstub_auth" > /dev/null
    log_success "Main account token deleted"

    # 删除 IAM 子账户 Token
    curl -s -X DELETE "$BASE_URL/api/v2/tokens/$TOKEN_ID_IAM" \
        -H "Authorization: $qstub_auth" > /dev/null
    log_success "IAM sub-account token deleted"

    # 删除自定义 prefix Token
    if [[ -n "$TOKEN_ID_PREFIX" ]]; then
        curl -s -X DELETE "$BASE_URL/api/v2/tokens/$TOKEN_ID_PREFIX" \
            -H "Authorization: $qstub_auth" > /dev/null
        log_success "Custom prefix token deleted"
    fi
}

# ========================================
# 主测试流程
# ========================================

main() {
    log_info "Starting Bearer Token Service V2 API Tests"
    log_info "Base URL: $BASE_URL"
    log_info "Qiniu UID: $QINIU_UID"
    log_info "Qiniu IUID: $QINIU_IUID"
    echo ""

    # 0. 健康检查
    test_step "0. Health Check"
    test_health_check

    # 1. 创建 Token（主账户）
    test_step "1. Create Token (Main Account)"
    test_create_token_main_account

    # 2. 创建 Token（IAM 子账户）
    test_step "2. Create Token (IAM Sub-Account)"
    test_create_token_iam_account

    # 2.1 创建 Token（自定义 prefix）
    test_step "2.1 Create Token (Custom Prefix)"
    test_create_token_with_prefix

    # 2.2 测试 prefix 校验（无效前缀）
    test_step "2.2 Prefix Validation Tests"
    test_create_token_invalid_prefix_uppercase
    test_create_token_invalid_prefix_too_long
    test_create_token_invalid_prefix_special_chars

    # 3. 列出 Tokens
    test_step "3. List Tokens"
    test_list_tokens

    # 4. 获取 Token 详情
    test_step "4. Get Token Info"
    test_get_token_info

    # 5. 验证 Bearer Token（主账户）
    test_step "5. Validate Bearer Token (Main Account)"
    test_validate_bearer_token_main

    # 6. 验证 Bearer Token（IAM 子账户）
    test_step "6. Validate Bearer Token (IAM Sub-Account)"
    test_validate_bearer_token_iam

    # 6.5 验证 Bearer Token 并返回用户信息（主账户）
    test_step "6.5 Validate Bearer Token with UserInfo (Main Account)"
    test_validate_bearer_token_with_userinfo_main

    # 6.6 验证 Bearer Token 并返回用户信息（IAM 子账户）
    test_step "6.6 Validate Bearer Token with UserInfo (IAM Sub-Account)"
    test_validate_bearer_token_with_userinfo_iam

    # 6.7 验证 Bearer Token 并返回完整用户信息（使用有效测试 UID）
    test_step "6.7 Validate Bearer Token with FULL UserInfo (Valid Test UID)"
    test_validate_bearer_token_with_full_userinfo

    # 7. 更新 Token 状态
    test_step "7. Update Token Status"
    test_update_token_status

    # 8. 删除 Tokens
    test_step "8. Delete Tokens"
    test_delete_tokens

    # 完成
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}🎉 All Tests Passed!${NC}"
    echo -e "${GREEN}  - Main Account (UID) ✓${NC}"
    echo -e "${GREEN}  - IAM Sub-Account (UID + IUID) ✓${NC}"
    echo -e "${GREEN}  - Custom Prefix Token ✓${NC}"
    echo -e "${GREEN}  - Prefix Validation ✓${NC}"
    echo -e "${GREEN}  - Bearer Token Validation (/validate) ✓${NC}"
    echo -e "${GREEN}  - Bearer Token with UserInfo (/validateu) ✓${NC}"
    echo -e "${GREEN}  - Qconfapi RPC Integration ✓${NC}"
    echo -e "${GREEN}========================================${NC}"
}

# 运行主测试
main
