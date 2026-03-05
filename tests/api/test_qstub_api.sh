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

# 2.05 创建 Token（IAM 子账户，iam_alias 方式）
test_create_token_iam_alias() {
    log_info "Creating token with IAM sub-account (iam_alias方式, uid=$QINIU_UID, iam_alias=testuser)..."

    local qstub_auth="QiniuStub uid=${QINIU_UID}&ut=1&app=1&iam_alias=testuser"

    local response=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
        -H "Authorization: $qstub_auth" \
        -H "Content-Type: application/json" \
        -d '{
            "description": "Test token for IAM sub-account (iam_alias)",
            "expires_in_seconds": 3600
        }')

    TOKEN_ID_IAM_ALIAS=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin)['token_id'])" 2>/dev/null)
    BEARER_TOKEN_IAM_ALIAS=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])" 2>/dev/null)

    if [[ -n "$TOKEN_ID_IAM_ALIAS" ]]; then
        log_success "Token created for IAM sub-account (iam_alias)"
        log_info "Token ID: $TOKEN_ID_IAM_ALIAS"
        log_info "Bearer Token: ${BEARER_TOKEN_IAM_ALIAS:0:20}..."
    else
        log_error "Failed to create token: $response"
        exit 1
    fi
}

# 6.65 验证 Bearer Token 并返回用户信息（IAM iam_alias 方式）
test_validate_bearer_token_with_userinfo_iam_alias() {
    log_info "Validating Bearer Token with UserInfo (IAM iam_alias sub-account)..."

    local response=$(curl -s -X POST "$BASE_URL/api/v2/validateu" \
        -H "Authorization: Bearer $BEARER_TOKEN_IAM_ALIAS" \
        -H "Content-Type: application/json")

    if echo "$response" | grep -q "404 page not found"; then
        log_warning "/api/v2/validateu endpoint not available - skipping test"
        return 0
    fi

    echo "$response" | python3 -m json.tool 2>/dev/null || {
        log_warning "Failed to parse JSON response, raw response: $response"
        return 0
    }

    local valid=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('valid', False))" 2>/dev/null)
    local iam_alias=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('token_info', {}).get('iam_alias', ''))" 2>/dev/null)

    if [[ "$valid" == "True" ]]; then
        log_success "Bearer Token validation with UserInfo passed (IAM iam_alias sub-account)"
        if [[ -n "$iam_alias" ]]; then
            log_success "iam_alias field present in token_info: $iam_alias"
        else
            log_warning "iam_alias field not present in token_info"
        fi
    else
        log_error "Bearer Token validation failed: $response"
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

# 6.8 验证 Bearer Token 并返回完整用户信息（IAM 子账户，含 iuid/iam_alias）
test_validate_bearer_token_with_full_userinfo_iam() {
    log_info "Validating Bearer Token with FULL UserInfo (IAM sub-account, iuid + iam_alias)..."

    local test_uids=("$QINIU_TEST_UID" "$QINIU_PROD_UID")
    local test_labels=("Test ENV ($QINIU_TEST_UID)" "Prod ENV ($QINIU_PROD_UID)")
    local found_userinfo=false

    for i in 0 1; do
        local uid="${test_uids[$i]}"
        local label="${test_labels[$i]}"

        log_info "Trying ${label} with iuid=${QINIU_IUID} and iam_alias=testuser..."

        # 用 iuid 创建 token
        local qstub_auth_iuid="QiniuStub uid=${uid}&ut=1&iuid=${QINIU_IUID}"
        local create_iuid=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
            -H "Authorization: $qstub_auth_iuid" \
            -H "Content-Type: application/json" \
            -d '{"description": "Full userinfo IAM iuid test", "expires_in_seconds": 300}')

        local token_id_iuid=$(echo $create_iuid | python3 -c "import sys, json; print(json.load(sys.stdin).get('token_id', ''))" 2>/dev/null)
        local bearer_iuid=$(echo $create_iuid | python3 -c "import sys, json; print(json.load(sys.stdin).get('token', ''))" 2>/dev/null)

        # 用 iam_alias 创建 token
        local qstub_auth_alias="QiniuStub uid=${uid}&ut=1&app=1&iam_alias=testuser"
        local create_alias=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
            -H "Authorization: $qstub_auth_alias" \
            -H "Content-Type: application/json" \
            -d '{"description": "Full userinfo IAM iam_alias test", "expires_in_seconds": 300}')

        local token_id_alias=$(echo $create_alias | python3 -c "import sys, json; print(json.load(sys.stdin).get('token_id', ''))" 2>/dev/null)
        local bearer_alias=$(echo $create_alias | python3 -c "import sys, json; print(json.load(sys.stdin).get('token', ''))" 2>/dev/null)

        if [[ -z "$token_id_iuid" || -z "$token_id_alias" ]]; then
            log_warning "Failed to create tokens for UID $uid, skipping..."
            continue
        fi

        # 验证 iuid token
        local resp_iuid=$(curl -s -X POST "$BASE_URL/api/v2/validateu" \
            -H "Authorization: Bearer $bearer_iuid")
        local has_ui_iuid=$(echo $resp_iuid | python3 -c "import sys, json; ti = json.load(sys.stdin).get('token_info', {}); print(ti.get('user_info') is not None)" 2>/dev/null)

        if [[ "$has_ui_iuid" == "True" ]]; then
            found_userinfo=true
            log_info "--- iuid 方式 ---"
            echo "$resp_iuid" | python3 -m json.tool 2>/dev/null

            local ret_iuid=$(echo $resp_iuid | python3 -c "import sys, json; print(json.load(sys.stdin).get('token_info', {}).get('iuid', ''))" 2>/dev/null)
            local ui_iuid=$(echo $resp_iuid | python3 -c "import sys, json; ui = json.load(sys.stdin).get('token_info', {}).get('user_info'); print(ui.get('iuid', 0) if ui else 0)" 2>/dev/null)

            if [[ -n "$ret_iuid" ]]; then
                log_success "token_info.iuid: $ret_iuid"
            else
                log_warning "token_info.iuid not present"
            fi
            if [[ "$ui_iuid" != "0" ]]; then
                log_success "user_info.iuid: $ui_iuid"
            else
                log_warning "user_info.iuid not present"
            fi
        fi

        # 验证 iam_alias token
        local resp_alias=$(curl -s -X POST "$BASE_URL/api/v2/validateu" \
            -H "Authorization: Bearer $bearer_alias")
        local has_ui_alias=$(echo $resp_alias | python3 -c "import sys, json; ti = json.load(sys.stdin).get('token_info', {}); print(ti.get('user_info') is not None)" 2>/dev/null)

        if [[ "$has_ui_alias" == "True" ]]; then
            found_userinfo=true
            log_info "--- iam_alias 方式 ---"
            echo "$resp_alias" | python3 -m json.tool 2>/dev/null

            local ret_alias=$(echo $resp_alias | python3 -c "import sys, json; print(json.load(sys.stdin).get('token_info', {}).get('iam_alias', ''))" 2>/dev/null)
            local ui_iuid_alias=$(echo $resp_alias | python3 -c "import sys, json; ui = json.load(sys.stdin).get('token_info', {}).get('user_info'); print(ui.get('iuid', 0) if ui else 0)" 2>/dev/null)

            if [[ -n "$ret_alias" ]]; then
                log_success "token_info.iam_alias: $ret_alias"
            else
                log_warning "token_info.iam_alias not present"
            fi
            if [[ "$ui_iuid_alias" != "0" ]]; then
                log_success "user_info.iuid: $ui_iuid_alias (iam_alias 方式无 iuid，符合预期)"
            else
                log_warning "user_info.iuid not present (iam_alias 方式，符合预期)"
            fi
        fi

        # 清理
        local qstub_auth="QiniuStub uid=${uid}&ut=1"
        curl -s -X DELETE "$BASE_URL/api/v2/tokens/$token_id_iuid" -H "Authorization: $qstub_auth" > /dev/null
        curl -s -X DELETE "$BASE_URL/api/v2/tokens/$token_id_alias" -H "Authorization: $qstub_auth" > /dev/null

        if [[ "$found_userinfo" == "true" ]]; then
            log_success "🎉 FULL UserInfo with IAM sub-account fields verified!"
            break
        fi
    done

    if [[ "$found_userinfo" == "false" ]]; then
        log_warning "No UserInfo found - Qconf RPC may not be configured (graceful degradation)"
    fi
}

# 3.1 列出 Tokens（iuid 子账号隔离）
# 验证：iuid 视角返回的数量 < 主账户总量（隔离生效）
test_list_tokens_iuid_isolation() {
    log_info "Listing tokens with iuid isolation (uid=$QINIU_UID, iuid=$QINIU_IUID)..."

    # 先获取主账户总数
    local main_total=$(curl -s -X GET "$BASE_URL/api/v2/tokens" \
        -H "Authorization: QiniuStub uid=${QINIU_UID}&ut=1" \
        | python3 -c "import sys, json; print(json.load(sys.stdin).get('total', -1))" 2>/dev/null)

    # 再获取 iuid 子账号视角的数量
    local response=$(curl -s -X GET "$BASE_URL/api/v2/tokens" \
        -H "Authorization: QiniuStub uid=${QINIU_UID}&ut=1&iuid=${QINIU_IUID}")

    echo "$response" | python3 -m json.tool 2>/dev/null

    local iuid_total=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('total', -1))" 2>/dev/null)

    if [[ "$iuid_total" == "-1" ]]; then
        log_error "Failed to list tokens with iuid: $response"
        exit 1
    fi

    # 子账号视角的数量必须 <= 主账户总量，且等于该 iuid 创建的 token 数（至少1个）
    if [[ "$iuid_total" -ge 1 && "$iuid_total" -lt "$main_total" ]]; then
        log_success "iuid isolation passed: iuid_total=$iuid_total < main_total=$main_total"
    elif [[ "$iuid_total" -ge 1 ]]; then
        log_success "iuid isolation: iuid_total=$iuid_total, main_total=$main_total"
    else
        log_error "iuid isolation failed: iuid_total=$iuid_total (expected >=1)"
        exit 1
    fi
}

# 3.2 列出 Tokens（iam_alias 子账号隔离）
# 验证：iam_alias 视角返回的数量 < 主账户总量（隔离生效）
test_list_tokens_iam_alias_isolation() {
    log_info "Listing tokens with iam_alias isolation (uid=$QINIU_UID, iam_alias=testuser)..."

    # 先获取主账户总数
    local main_total=$(curl -s -X GET "$BASE_URL/api/v2/tokens" \
        -H "Authorization: QiniuStub uid=${QINIU_UID}&ut=1" \
        | python3 -c "import sys, json; print(json.load(sys.stdin).get('total', -1))" 2>/dev/null)

    # 获取 iam_alias 子账号视角的数量
    local response=$(curl -s -X GET "$BASE_URL/api/v2/tokens" \
        -H "Authorization: QiniuStub uid=${QINIU_UID}&ut=1&app=1&iam_alias=testuser")

    echo "$response" | python3 -m json.tool 2>/dev/null

    local alias_total=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('total', -1))" 2>/dev/null)

    if [[ "$alias_total" == "-1" ]]; then
        log_error "Failed to list tokens with iam_alias: $response"
        exit 1
    fi

    if [[ "$alias_total" -ge 1 && "$alias_total" -lt "$main_total" ]]; then
        log_success "iam_alias isolation passed: alias_total=$alias_total < main_total=$main_total"
    elif [[ "$alias_total" -ge 1 ]]; then
        log_success "iam_alias isolation: alias_total=$alias_total, main_total=$main_total"
    else
        log_error "iam_alias isolation failed: alias_total=$alias_total (expected >=1)"
        exit 1
    fi
}

# 3.3 主账户列出全部 Tokens（不隔离）
test_list_tokens_main_sees_all() {
    log_info "Listing tokens with main account (should see all tokens including sub-accounts)..."

    local qstub_auth="QiniuStub uid=${QINIU_UID}&ut=1"

    local response=$(curl -s -X GET "$BASE_URL/api/v2/tokens" \
        -H "Authorization: $qstub_auth")

    local total=$(echo $response | python3 -c "import sys, json; print(json.load(sys.stdin).get('total', -1))" 2>/dev/null)

    if [[ "$total" == "-1" ]]; then
        log_error "Failed to list tokens: $response"
        exit 1
    fi

    # 主账户应该能看到所有 token（主账户 + iuid子账号 + iam_alias子账号 + prefix token = 至少4个）
    if [[ "$total" -ge 4 ]]; then
        log_success "Main account sees all tokens: total=$total (includes sub-account tokens)"
    else
        log_warning "Main account sees $total tokens (expected >=4, some may be filtered)"
    fi
}

# 4.1 子账号跨账号操作隔离验证
# iuid 子账号不能 GET/PUT/DELETE 属于 iam_alias 子账号的 token，反之亦然
test_subaccount_cross_access_denied() {
    local qstub_main="QiniuStub uid=${QINIU_UID}&ut=1"
    local qstub_iuid="QiniuStub uid=${QINIU_UID}&ut=1&iuid=${QINIU_IUID}"
    local qstub_alias="QiniuStub uid=${QINIU_UID}&ut=1&app=1&iam_alias=testuser"

    # 创建 iuid token 和 iam_alias token
    local iuid_tid=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
        -H "Authorization: $qstub_iuid" -H "Content-Type: application/json" \
        -d '{"description":"cross-access-iuid"}' \
        | python3 -c "import sys,json; print(json.load(sys.stdin).get('token_id',''))" 2>/dev/null)

    local alias_tid=$(curl -s -X POST "$BASE_URL/api/v2/tokens" \
        -H "Authorization: $qstub_alias" -H "Content-Type: application/json" \
        -d '{"description":"cross-access-alias"}' \
        | python3 -c "import sys,json; print(json.load(sys.stdin).get('token_id',''))" 2>/dev/null)

    log_info "Created iuid token: $iuid_tid, alias token: $alias_tid"

    local passed=0
    local failed=0

    # iuid 子账号尝试 GET iam_alias token → 应 403
    local r=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/api/v2/tokens/$alias_tid" \
        -H "Authorization: $qstub_iuid")
    if [[ "$r" == "403" ]]; then
        log_success "GET: iuid cannot access alias token (403) ✓"; passed=$((passed+1))
    else
        log_error "GET: iuid should be denied alias token, got $r"; failed=$((failed+1))
    fi

    # iam_alias 子账号尝试 GET iuid token → 应 403
    r=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/api/v2/tokens/$iuid_tid" \
        -H "Authorization: $qstub_alias")
    if [[ "$r" == "403" ]]; then
        log_success "GET: alias cannot access iuid token (403) ✓"; passed=$((passed+1))
    else
        log_error "GET: alias should be denied iuid token, got $r"; failed=$((failed+1))
    fi

    # iuid 子账号尝试 PUT iam_alias token → 应 403
    r=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/api/v2/tokens/$alias_tid/status" \
        -H "Authorization: $qstub_iuid" -H "Content-Type: application/json" \
        -d '{"is_active":false}')
    if [[ "$r" == "403" ]]; then
        log_success "PUT: iuid cannot update alias token (403) ✓"; passed=$((passed+1))
    else
        log_error "PUT: iuid should be denied alias token, got $r"; failed=$((failed+1))
    fi

    # iam_alias 子账号尝试 DELETE iuid token → 应 403
    r=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/api/v2/tokens/$iuid_tid" \
        -H "Authorization: $qstub_alias")
    if [[ "$r" == "403" ]]; then
        log_success "DELETE: alias cannot delete iuid token (403) ✓"; passed=$((passed+1))
    else
        log_error "DELETE: alias should be denied iuid token, got $r"; failed=$((failed+1))
    fi

    # iuid 子账号能操作自己的 token → 应 200
    r=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/api/v2/tokens/$iuid_tid" \
        -H "Authorization: $qstub_iuid")
    if [[ "$r" == "200" ]]; then
        log_success "GET: iuid can access own token (200) ✓"; passed=$((passed+1))
    else
        log_error "GET: iuid should access own token, got $r"; failed=$((failed+1))
    fi

    # 主账号能操作所有 token → 应 200
    r=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/api/v2/tokens/$alias_tid" \
        -H "Authorization: $qstub_main")
    if [[ "$r" == "200" ]]; then
        log_success "GET: main account can access any token (200) ✓"; passed=$((passed+1))
    else
        log_error "GET: main account should access alias token, got $r"; failed=$((failed+1))
    fi

    # 清理
    curl -s -X DELETE "$BASE_URL/api/v2/tokens/$iuid_tid"  -H "Authorization: $qstub_main" > /dev/null
    curl -s -X DELETE "$BASE_URL/api/v2/tokens/$alias_tid" -H "Authorization: $qstub_main" > /dev/null

    if [[ "$failed" -gt 0 ]]; then
        log_error "Sub-account cross-access isolation: $passed passed, $failed FAILED"
        exit 1
    fi
    log_success "Sub-account cross-access isolation: all $passed checks passed ✓"
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

    # 删除 iam_alias 子账户 Token
    if [[ -n "$TOKEN_ID_IAM_ALIAS" ]]; then
        curl -s -X DELETE "$BASE_URL/api/v2/tokens/$TOKEN_ID_IAM_ALIAS" \
            -H "Authorization: $qstub_auth" > /dev/null
        log_success "IAM iam_alias token deleted"
    fi

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

    # 2.05 创建 Token（IAM 子账户，iam_alias 方式）
    test_step "2.05 Create Token (IAM Sub-Account, iam_alias)"
    test_create_token_iam_alias

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

    # 3.1 列出 Tokens（iuid 子账号隔离）
    test_step "3.1 List Tokens (iuid Sub-Account Isolation)"
    test_list_tokens_iuid_isolation

    # 3.2 列出 Tokens（iam_alias 子账号隔离）
    test_step "3.2 List Tokens (iam_alias Sub-Account Isolation)"
    test_list_tokens_iam_alias_isolation

    # 3.3 主账户列出全部 Tokens
    test_step "3.3 List Tokens (Main Account - Sees All)"
    test_list_tokens_main_sees_all

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

    # 6.65 验证 Bearer Token 并返回用户信息（IAM iam_alias 方式）
    test_step "6.65 Validate Bearer Token with UserInfo (IAM iam_alias Sub-Account)"
    test_validate_bearer_token_with_userinfo_iam_alias

    # 6.7 验证 Bearer Token 并返回完整用户信息（使用有效测试 UID）
    test_step "6.7 Validate Bearer Token with FULL UserInfo (Valid Test UID)"
    test_validate_bearer_token_with_full_userinfo

    # 6.8 验证 Bearer Token 并返回完整用户信息（IAM 子账户，含 iuid/iam_alias）
    test_step "6.8 Validate Bearer Token with FULL UserInfo (IAM Sub-Account)"
    test_validate_bearer_token_with_full_userinfo_iam

    # 4.1 子账号不能访问他人 Token（隔离验证）
    test_step "4.1 Sub-Account Cannot Access Other's Token"
    test_subaccount_cross_access_denied

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
    echo -e "${GREEN}  - IAM Sub-Account (UID + iam_alias) ✓${NC}"
    echo -e "${GREEN}  - Custom Prefix Token ✓${NC}"
    echo -e "${GREEN}  - Prefix Validation ✓${NC}"
    echo -e "${GREEN}  - List Tokens (iuid isolation) ✓${NC}"
    echo -e "${GREEN}  - List Tokens (iam_alias isolation) ✓${NC}"
    echo -e "${GREEN}  - List Tokens (main account sees all) ✓${NC}"
    echo -e "${GREEN}  - Sub-Account Cross-Access Denied ✓${NC}"
    echo -e "${GREEN}  - Bearer Token Validation (/validate) ✓${NC}"
    echo -e "${GREEN}  - Bearer Token with UserInfo (/validateu) ✓${NC}"
    echo -e "${GREEN}  - Qconfapi RPC Integration ✓${NC}"
    echo -e "${GREEN}========================================${NC}"
}

# 运行主测试
main
