#!/bin/bash
# ========================================
# Bearer Token Service - 统一管理脚本
# ========================================
# 查看状态、日志、重启等运维操作

set -e

C_RESET='\033[0m'
C_INFO='\033[36m'
C_SUCCESS='\033[32m'
C_ERROR='\033[31m'

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_DIR="$PROJECT_ROOT/deploy/docker-compose"
KUBECONFIG_TEST="$PROJECT_ROOT/_cust/kubeconfig-test"

declare -A DEVICE_HOSTS DEVICE_PATHS
DEVICE_HOSTS[vmxs1]="10.134.2.1"
DEVICE_PATHS[vmxs1]="/root/deploy"
DEVICE_HOSTS[vmxs2]="10.134.2.2"
DEVICE_PATHS[vmxs2]="/root/haili/deploy"

log_info() { echo -e "${C_INFO}[INFO]${C_RESET} $1"; }
log_success() { echo -e "${C_SUCCESS}[SUCCESS]${C_RESET} $1"; }
log_error() { echo -e "${C_ERROR}[ERROR]${C_RESET} $1"; }

show_help() {
    cat << EOF
Bearer Token Service - 统一管理脚本

用法: $0 <environment> <command>

环境:
  local        本地 Docker Compose 环境
  k8s-test     K8s 测试环境
  vmxs1        物理服务器 1
  vmxs2        物理服务器 2

命令:
  status       查看服务状态
  logs         查看实时日志
  health       健康检查
  restart      重启服务
  exec         进入容器

示例:
  $0 local status
  $0 local logs
  $0 vmxs1 status
  $0 vmxs1 logs
  $0 k8s-test status

EOF
}

# 本地环境管理
manage_local() {
    local cmd="$1"
    case "$cmd" in
        status)
            cd "$COMPOSE_DIR"
            docker-compose -f docker-compose.legacy.yml ps
            ;;
        logs)
            cd "$COMPOSE_DIR"
            docker-compose -f docker-compose.legacy.yml logs -f --tail=100
            ;;
        health)
            curl -s http://localhost/health || echo "健康检查失败"
            ;;
        restart)
            cd "$COMPOSE_DIR"
            docker-compose -f docker-compose.legacy.yml restart bearer-token-service
            ;;
        exec)
            cd "$COMPOSE_DIR"
            docker-compose -f docker-compose.legacy.yml exec bearer-token-service sh
            ;;
        *)
            log_error "未知命令: $cmd"
            exit 1
            ;;
    esac
}

# K8s 环境管理
manage_k8s() {
    local cmd="$1"
    case "$cmd" in
        status)
            KUBECONFIG="$KUBECONFIG_TEST" kubectl get all -n bearer-token-test
            ;;
        logs)
            KUBECONFIG="$KUBECONFIG_TEST" kubectl logs -f -n bearer-token-test -l app=bearer-token-service
            ;;
        health)
            log_info "端口转发到本地 8080..."
            KUBECONFIG="$KUBECONFIG_TEST" kubectl port-forward -n bearer-token-test svc/bearer-token-bearer-token-service 8080:8080
            ;;
        *)
            log_error "未知命令: $cmd"
            exit 1
            ;;
    esac
}

# 物理设备管理
manage_physical() {
    local device="$1"
    local cmd="$2"
    local host="${DEVICE_HOSTS[$device]}"
    local path="${DEVICE_PATHS[$device]}"

    case "$cmd" in
        status)
            log_info "设备状态: $device ($host)"
            echo ""
            ssh "$device" "cd $path && docker-compose -f docker-compose.legacy.yml ps"
            echo ""
            ssh "$device" "curl -s http://localhost/health" || log_error "健康检查失败"
            ;;
        logs)
            log_info "实时日志: $device"
            ssh "$device" "cd $path && docker-compose -f docker-compose.legacy.yml logs -f --tail=100 bearer-token-service-production"
            ;;
        health)
            log_info "健康检查: $device ($host)"
            if ssh "$device" "curl -sf http://localhost/health"; then
                log_success "服务正常"
            else
                log_error "服务异常"
            fi
            ;;
        restart)
            log_info "重启服务: $device"
            ssh "$device" "cd $path && docker-compose -f docker-compose.legacy.yml restart bearer-token-service-production"
            sleep 5
            $0 "$device" health
            ;;
        exec)
            log_info "进入容器: $device"
            ssh -t "$device" "cd $path && docker-compose -f docker-compose.legacy.yml exec bearer-token-service-production sh"
            ;;
        *)
            log_error "未知命令: $cmd"
            exit 1
            ;;
    esac
}

# 主逻辑
ENV="${1:-}"
CMD="${2:-status}"

case "$ENV" in
    local)
        manage_local "$CMD"
        ;;
    k8s-test)
        manage_k8s "$CMD"
        ;;
    vmxs1|vmxs2)
        manage_physical "$ENV" "$CMD"
        ;;
    help|--help|-h|"")
        show_help
        ;;
    *)
        log_error "未知环境: $ENV"
        show_help
        exit 1
        ;;
esac
