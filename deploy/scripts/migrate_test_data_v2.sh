#!/bin/bash
# migrate_test_data_v2.sh - 改进版本，解决SSH和字符集问题
set -euo pipefail

# 加载敏感配置（从 _cust/ 目录）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CREDENTIALS_FILE="${PROJECT_ROOT}/_cust/credentials.env"

if [[ ! -f "${CREDENTIALS_FILE}" ]]; then
    echo "错误: 找不到凭据配置文件: ${CREDENTIALS_FILE}"
    echo "请先创建此文件（可以从 credentials.env.example 复制）"
    exit 1
fi

source "${CREDENTIALS_FILE}"

# 配置
PROD_SERVER="${SSH_PROXY_HOST:-vmyzh165}"
PROD_HOST="${PROD_MYSQL_HOST}"
PROD_PORT="${PROD_MYSQL_PORT}"
PROD_USER="${PROD_MYSQL_USER}"
PROD_PASSWORD="${PROD_MYSQL_PASSWORD}"
PROD_DATABASE="${PROD_MYSQL_DATABASE}"
PROD_TABLE="auth"

TEST_HOST="${TEST_MYSQL_HOST}"
TEST_PORT="${TEST_MYSQL_PORT}"
TEST_USER="${TEST_MYSQL_USER}"
TEST_PASSWORD="${TEST_MYSQL_PASSWORD}"
TEST_DATABASE="${1:-${TEST_MYSQL_DATABASE}}"
EXPORT_LIMIT="${2:-500}"

GREEN='\033[0;32m'
BLUE='\033[0;36m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[✓]${NC} $1"; }
log_step() { echo -e "${BLUE}[STEP]${NC} $1"; }

echo "===================================================================="
log_info "MySQL 测试数据迁移脚本 v2"
echo "===================================================================="
echo ""

# 1. 在远程服务器导出
log_step "从生产环境导出数据 (${EXPORT_LIMIT} 条记录)..."
REMOTE_FILE=$(ssh ${PROD_SERVER} bash -s ${EXPORT_LIMIT} ${PROD_HOST} ${PROD_PORT} ${PROD_USER} ${PROD_PASSWORD} ${PROD_DATABASE} << 'REMOTE_SCRIPT'
LIMIT=$1
MYSQL_HOST=$2
MYSQL_PORT=$3
MYSQL_USER=$4
MYSQL_PASS=$5
MYSQL_DB=$6
OUTPUT="/tmp/auth_export_$(date +%s).sql"
mysqldump -h "${MYSQL_HOST}" -P "${MYSQL_PORT}" -u "${MYSQL_USER}" -p"${MYSQL_PASS}" \
    --complete-insert \
    --skip-extended-insert \
    --where="1=1 LIMIT ${LIMIT}" \
    "${MYSQL_DB}" auth > "${OUTPUT}" 2>/dev/null && echo "${OUTPUT}"
REMOTE_SCRIPT
)
log_info "远程导出完成: ${REMOTE_FILE}"

# 2. 下载到本地
log_step "下载导出文件..."
LOCAL_EXPORT="/tmp/auth_export_$(date +%s).sql"
scp ${PROD_SERVER}:"${REMOTE_FILE}" "${LOCAL_EXPORT}" >/dev/null 2>&1
log_info "下载完成: ${LOCAL_EXPORT}"

# 3. 混淆数据
log_step "混淆敏感数据..."
LOCAL_OBFUSCATED="/tmp/auth_obfuscated_$(date +%s).sql"
python3 "${SCRIPT_DIR}/obfuscate_data.py" "${LOCAL_EXPORT}" "${LOCAL_OBFUSCATED}" >/dev/null
log_info "数据混淆完成"

# 4. 修复字符集兼容性
log_step "修复字符集兼容性..."
sed -i 's/utf8mb4_0900_ai_ci/utf8mb4_unicode_ci/g' "${LOCAL_OBFUSCATED}"
sed -i 's/utf8mb3_general_ci/utf8_general_ci/g' "${LOCAL_OBFUSCATED}"
log_info "字符集修复完成"

# 5. 创建测试数据库
log_step "创建测试数据库: ${TEST_DATABASE}"
mysql -h "${TEST_HOST}" -P "${TEST_PORT}" -u "${TEST_USER}" -p"${TEST_PASSWORD}" \
  -e "DROP DATABASE IF EXISTS \`${TEST_DATABASE}\`; CREATE DATABASE \`${TEST_DATABASE}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>/dev/null
log_info "数据库创建完成"

# 6. 导入数据
log_step "导入数据到测试数据库..."
mysql -h "${TEST_HOST}" -P "${TEST_PORT}" -u "${TEST_USER}" -p"${TEST_PASSWORD}" \
  "${TEST_DATABASE}" < "${LOCAL_OBFUSCATED}" 2>/dev/null
log_info "数据导入完成"

# 7. 验证
log_step "验证导入数据..."
RECORD_COUNT=$(mysql -h "${TEST_HOST}" -P "${TEST_PORT}" -u "${TEST_USER}" -p"${TEST_PASSWORD}" \
  "${TEST_DATABASE}" -sse "SELECT COUNT(*) FROM ${PROD_TABLE};" 2>/dev/null)
log_info "成功导入 ${RECORD_COUNT} 条记录"

# 8. 显示示例数据
echo ""
echo "示例数据（前3条）:"
mysql -h "${TEST_HOST}" -P "${TEST_PORT}" -u "${TEST_USER}" -p"${TEST_PASSWORD}" \
  "${TEST_DATABASE}" -e "SELECT id, username, email FROM ${PROD_TABLE} LIMIT 3;" 2>/dev/null | sed 's/^/  /'

# 9. 清理
log_step "清理临时文件..."
rm -f "${LOCAL_EXPORT}" "${LOCAL_OBFUSCATED}"
ssh ${PROD_SERVER} "rm -f ${REMOTE_FILE}" 2>/dev/null || true
log_info "清理完成"

echo ""
echo "===================================================================="
log_info "✅ 迁移完成！"
echo "===================================================================="
echo ""
echo "测试数据库配置:"
echo "  export MYSQL_HOST=\"${TEST_HOST}\""
echo "  export MYSQL_PORT=\"${TEST_PORT}\""
echo "  export MYSQL_USER=\"${TEST_USER}\""
echo "  export MYSQL_PASSWORD=\"${TEST_PASSWORD}\""
echo "  export MYSQL_DATABASE=\"${TEST_DATABASE}\""
echo ""
echo "连接命令:"
echo "  mysql -h ${TEST_HOST} -P ${TEST_PORT} -u ${TEST_USER} -p'${TEST_PASSWORD}' ${TEST_DATABASE}"
echo ""
