#!/bin/bash
# check_mysql_config.sh - 检查 MySQL 配置

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[✓]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[!]${NC} $1"; }
log_error() { echo -e "${RED}[✗]${NC} $1"; }

echo "===================================================================="
echo "MySQL 配置检查工具"
echo "===================================================================="
echo ""

# 检查 Docker Compose 配置
if [ -f "deploy/docker-compose/.env" ]; then
    echo "1. Docker Compose 配置:"
    if grep -q "^MYSQL_HOST=" deploy/docker-compose/.env 2>/dev/null; then
        MYSQL_HOST=$(grep "^MYSQL_HOST=" deploy/docker-compose/.env | cut -d'=' -f2)
        MYSQL_DATABASE=$(grep "^MYSQL_DATABASE=" deploy/docker-compose/.env | cut -d'=' -f2)
        log_info "已配置 MySQL: ${MYSQL_HOST} / ${MYSQL_DATABASE}"
    else
        log_warn "未配置 MySQL（可选）"
    fi
else
    log_warn "未找到 .env 文件"
fi

echo ""

# 检查 Helm values
echo "2. Helm 配置:"

if [ -f "deploy/helm/bearer-token-service/values-prod.yaml" ]; then
    if grep -q "enabled: true" deploy/helm/bearer-token-service/values-prod.yaml | grep -A5 "externalMysql" >/dev/null 2>&1; then
        PROD_HOST=$(grep -A10 "externalMysql:" deploy/helm/bearer-token-service/values-prod.yaml | grep "host:" | head -1 | awk '{print $2}' | tr -d '"')
        log_info "生产环境已配置 MySQL: ${PROD_HOST}"
    else
        log_warn "生产环境未启用 MySQL"
    fi
fi

if [ -f "deploy/helm/bearer-token-service/values-test.yaml" ]; then
    if grep -q "enabled: true" deploy/helm/bearer-token-service/values-test.yaml | grep -A5 "externalMysql" >/dev/null 2>&1; then
        TEST_HOST=$(grep -A10 "externalMysql:" deploy/helm/bearer-token-service/values-test.yaml | grep "host:" | head -1 | awk '{print $2}' | tr -d '"')
        log_info "测试环境已配置 MySQL: ${TEST_HOST}"
    else
        log_warn "测试环境未启用 MySQL"
    fi
fi

echo ""
echo "===================================================================="
echo "配置文件位置:"
echo "  Docker Compose: deploy/docker-compose/.env.example"
echo "  Helm 生产:      deploy/helm/bearer-token-service/values-prod.yaml"
echo "  Helm 测试:      deploy/helm/bearer-token-service/values-test.yaml"
echo "===================================================================="
