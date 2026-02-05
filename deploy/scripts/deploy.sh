#!/bin/bash
# ========================================
# Bearer Token Service - 统一部署脚本
# ========================================
# 支持多种部署方式：
# - local: 本地测试环境 (Docker Compose + MongoDB)
# - k8s-test: K8s 测试环境 (Helm)
# - physical: 物理服务器生产环境 (vmxs1, vmxs2)

set -e

# 颜色定义
C_RESET='\033[0m'
C_INFO='\033[36m'
C_SUCCESS='\033[32m'
C_WARNING='\033[33m'
C_ERROR='\033[31m'

# 项目配置
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_DIR="$PROJECT_ROOT/deploy/docker-compose"
HELM_CHART="$PROJECT_ROOT/deploy/helm/bearer-token-service"
KUBECONFIG_TEST="$PROJECT_ROOT/_cust/kubeconfig-test"
CREDENTIALS_FILE="$PROJECT_ROOT/_cust/credentials.env"

# 物理设备配置
declare -A DEVICE_HOSTS DEVICE_PATHS
DEVICE_HOSTS[vmxs1]="10.134.2.1"
DEVICE_PATHS[vmxs1]="/root/deploy"
DEVICE_HOSTS[vmxs2]="10.134.2.2"
DEVICE_PATHS[vmxs2]="/root/haili/deploy"

log_info() { echo -e "${C_INFO}[INFO]${C_RESET} $1"; }
log_success() { echo -e "${C_SUCCESS}[SUCCESS]${C_RESET} $1"; }
log_warning() { echo -e "${C_WARNING}[WARNING]${C_RESET} $1"; }
log_error() { echo -e "${C_ERROR}[ERROR]${C_RESET} $1"; }

# 从 credentials.env 提取配置
extract_credentials() {
    local env_type="$1"  # TEST 或 PROD

    if [ ! -f "$CREDENTIALS_FILE" ]; then
        log_warning "credentials.env 不存在: $CREDENTIALS_FILE"
        return 1
    fi

    # 提取 Qconf 配置
    export QCONF_ACCESS_KEY=$(grep "^${env_type}_QCONF_ACCESS_KEY=" "$CREDENTIALS_FILE" | cut -d'=' -f2- | tr -d '"')
    export QCONF_SECRET_KEY=$(grep "^${env_type}_QCONF_SECRET_KEY=" "$CREDENTIALS_FILE" | cut -d'=' -f2- | tr -d '"')
    export QCONF_MASTER_HOSTS=$(grep "^${env_type}_QCONF_MASTER_HOSTS=" "$CREDENTIALS_FILE" | cut -d'=' -f2- | tr -d '"')

    if [ -z "$QCONF_ACCESS_KEY" ] || [ -z "$QCONF_SECRET_KEY" ] || [ -z "$QCONF_MASTER_HOSTS" ]; then
        log_warning "无法从 credentials.env 提取完整的 ${env_type} Qconf 配置"
        return 1
    fi

    log_info "已从 credentials.env 提取 ${env_type} Qconf 配置"
    return 0
}

show_help() {
    cat << EOF
Bearer Token Service - 统一部署脚本

用法: $0 <environment> [options]

环境类型:
  local                本地测试环境（Docker Compose + MongoDB）
  k8s-test             K8s 测试环境（Helm）
  physical <device>    物理服务器生产环境（vmxs1 或 vmxs2）

示例:
  # 本地测试
  $0 local start
  $0 local stop
  $0 local logs
  $0 local test        # 运行 API 测试验证

  # K8s 测试环境
  $0 k8s-test deploy
  $0 k8s-test status
  $0 k8s-test test     # 运行 API 测试验证

  # 物理服务器生产环境
  $0 physical vmxs1
  $0 physical vmxs1 /path/to/image.tar

操作命令（local）:
  start     启动服务
  stop      停止服务
  restart   重启服务
  logs      查看日志
  status    查看状态
  test      运行 API 测试验证服务功能

操作命令（k8s-test）:
  deploy    部署到集群
  delete    删除部署
  status    查看状态
  test      运行 API 测试验证服务功能

操作命令（physical）:
  <device>            部署到指定物理服务器
  <device> test       在服务器上运行 API 测试验证（通过 SSH 远程执行）
  <device> <image>    使用指定镜像部署

注意:
  - physical 部署会自动从 _cust/credentials.env 提取 Qconf 配置并同步到服务器
  - physical test 通过 SSH 在服务器上远程执行，无需本地网络可达

相关命令:
  # 构建打包
  make package

  # 服务管理
  ./deploy/scripts/manage.sh <env> <command>

EOF
}

# ========================================
# Docker Compose 部署
# ========================================

deploy_local() {
    local action="$1"

    case "$action" in
        start)
            log_info "启动本地测试环境..."
            cd "$COMPOSE_DIR"
            ./docker-compose-legacy-deploy.sh test
            ;;
        stop)
            log_info "停止服务..."
            cd "$COMPOSE_DIR"
            docker-compose -f docker-compose.legacy.yml down
            ;;
        restart)
            $0 local stop
            sleep 2
            $0 local start
            ;;
        logs)
            cd "$COMPOSE_DIR"
            docker-compose -f docker-compose.legacy.yml logs -f --tail=100
            ;;
        status)
            cd "$COMPOSE_DIR"
            docker-compose -f docker-compose.legacy.yml ps
            ;;
        test)
            log_info "运行 API 测试（本地环境）..."
            export BASE_URL="http://localhost:8081"
            bash "$PROJECT_ROOT/tests/api/test_qstub_api.sh"
            if [ $? -eq 0 ]; then
                log_success "API 测试通过"
            else
                log_error "API 测试失败"
                exit 1
            fi
            ;;
        *)
            log_error "未知操作: $action"
            echo "可用操作: start, stop, restart, logs, status, test"
            exit 1
            ;;
    esac
}

# ========================================
# K8s 部署
# ========================================

deploy_k8s_test() {
    local action="$1"

    if [ ! -f "$KUBECONFIG_TEST" ]; then
        log_error "kubeconfig 不存在: $KUBECONFIG_TEST"
        exit 1
    fi

    case "$action" in
        deploy)
            log_info "部署到 K8s 测试环境..."
            KUBECONFIG="$KUBECONFIG_TEST" helm upgrade --install bearer-token "$HELM_CHART" \
                -f "$HELM_CHART/values-test.yaml" \
                -n bearer-token-test
            log_success "部署完成"
            ;;
        delete)
            log_warning "删除测试环境..."
            KUBECONFIG="$KUBECONFIG_TEST" helm uninstall bearer-token -n bearer-token-test --ignore-not-found
            log_success "删除完成"
            ;;
        status)
            log_info "K8s 测试环境状态:"
            KUBECONFIG="$KUBECONFIG_TEST" kubectl get all -n bearer-token-test
            ;;
        test)
            log_info "运行 API 测试（K8s 测试环境）..."
            export BASE_URL="http://bearer-token-test.jfcs-k8s-qa1.qiniu.io"
            bash "$PROJECT_ROOT/tests/api/test_qstub_api.sh"
            if [ $? -eq 0 ]; then
                log_success "API 测试通过"
            else
                log_error "API 测试失败"
                exit 1
            fi
            ;;
        *)
            log_error "未知操作: $action"
            echo "可用操作: deploy, delete, status, test"
            exit 1
            ;;
    esac
}

# ========================================
# 物理设备部署
# ========================================

deploy_physical() {
    local device="$1"
    local param2="$2"

    # 判断第二个参数是 test 还是镜像路径
    local run_test=false
    local image_file="$PROJECT_ROOT/dist/bearer-token-service-latest.tar"

    if [ "$param2" = "test" ]; then
        run_test=true
    elif [ -n "$param2" ] && [ "$param2" != "test" ]; then
        image_file="$param2"
    fi

    if [ -z "${DEVICE_HOSTS[$device]}" ]; then
        log_error "未知设备: $device"
        echo "可用设备: vmxs1, vmxs2"
        exit 1
    fi

    local host="${DEVICE_HOSTS[$device]}"
    local path="${DEVICE_PATHS[$device]}"

    # 如果只是运行测试，不需要镜像文件
    if [ "$run_test" = false ]; then
        if [ ! -f "$image_file" ]; then
            log_error "镜像文件不存在: $image_file"
            echo "请先运行: make package"
            exit 1
        fi

        log_info "部署到物理设备: $device ($host)"
        echo "  镜像: $image_file"
        echo ""

        # 二次确认
        if [ "$NO_CONFIRM" != "true" ]; then
            read -p "确认部署到 $device? 输入设备名确认: " confirm
            if [ "$confirm" != "$device" ]; then
                log_error "已取消"
                exit 1
            fi
        fi

        # 提取并同步 Qconf 配置
        log_info "[1/5] 同步 Qconf 配置..."
        if extract_credentials "PROD"; then
            ssh "$device" "bash -s" <<EOF
                # 删除旧的 Qconf 配置（如果存在）
                sed -i '/^# .*Qconf RPC 配置/,/^QCONF_MC_RW_TIMEOUT_MS=/d' $path/.env 2>/dev/null || true

                # 添加新配置
                cat >> $path/.env << 'QCONF_EOF'

# ========================================
# Qconf RPC 配置（生产环境 - 用于 /api/v2/validateu 获取用户信息）
# ========================================
QCONF_ENABLED=true
QCONF_ACCESS_KEY=$QCONF_ACCESS_KEY
QCONF_SECRET_KEY=$QCONF_SECRET_KEY
QCONF_MASTER_HOSTS=$QCONF_MASTER_HOSTS
QCONF_MC_HOSTS=
QCONF_LC_EXPIRES_MS=600000
QCONF_LC_DURATION_MS=5000
QCONF_LC_CHAN_BUFSIZE=16000
QCONF_MC_RW_TIMEOUT_MS=100
QCONF_EOF
EOF
            if [ $? -eq 0 ]; then
                log_success "Qconf 配置已同步"
            else
                log_error "同步配置失败"
                exit 1
            fi
        else
            log_warning "跳过 Qconf 配置同步"
        fi

        log_info "[2/5] 上传镜像..."
        scp "$image_file" "$device:/tmp/" || { log_error "上传失败"; exit 1; }

        log_info "[3/5] 加载镜像..."
        ssh "$device" "docker load -i /tmp/$(basename "$image_file")" || { log_error "加载失败"; exit 1; }

        log_info "[4/5] 重启服务..."
        ssh "$device" "cd $path && docker-compose -f docker-compose.legacy.yml up -d --no-deps bearer-token-service-production" || { log_error "重启失败"; exit 1; }

        log_info "[5/5] 健康检查..."
        sleep 5
        if ssh "$device" "curl -sf http://localhost/health" >/dev/null 2>&1; then
            log_success "部署完成，服务正常"
        else
            log_warning "服务可能仍在启动，请稍后检查"
        fi
    fi

    # 运行测试
    if [ "$run_test" = true ]; then
        log_info "运行 API 测试（$device）..."

        # 上传测试脚本到远程服务器
        local test_script="$PROJECT_ROOT/tests/api/test_qstub_api.sh"
        log_info "上传测试脚本到 $device..."
        scp "$test_script" "$device:/tmp/test_qstub_api.sh" >/dev/null || { log_error "上传测试脚本失败"; exit 1; }

        # 在远程服务器上执行测试（使用 localhost）
        log_info "在 $device 上执行测试..."
        if ssh "$device" "BASE_URL=http://localhost bash /tmp/test_qstub_api.sh"; then
            log_success "API 测试通过"
            # 清理临时文件
            ssh "$device" "rm -f /tmp/test_qstub_api.sh" >/dev/null 2>&1 || true
        else
            log_error "API 测试失败"
            # 清理临时文件
            ssh "$device" "rm -f /tmp/test_qstub_api.sh" >/dev/null 2>&1 || true
            exit 1
        fi
    fi
}

# ========================================
# 主逻辑
# ========================================

ENV="${1:-}"
ACTION="${2:-start}"

case "$ENV" in
    local)
        deploy_local "$ACTION"
        ;;
    k8s-test)
        deploy_k8s_test "$ACTION"
        ;;
    physical)
        if [ -z "$2" ]; then
            log_error "缺少设备参数"
            show_help
            exit 1
        fi
        shift
        deploy_physical "$@"
        ;;
    help|--help|-h|"")
        show_help
        ;;
    *)
        log_error "未知环境: $ENV"
        echo ""
        show_help
        exit 1
        ;;
esac
