#!/bin/bash
# ========================================
# Bearer Token Service - Legacy 运维脚本
# ========================================
# 适用于 docker-compose 1.25.0+
# 简化常用运维操作

COMPOSE_FILE="docker-compose.legacy.yml"
DEPLOY_ENV="${DEPLOY_ENV:-prod}"  # 默认生产环境

show_help() {
    cat << EOF
Bearer Token Service - 运维工具 (Legacy)

用法: $0 <命令> [选项]

环境控制:
  DEPLOY_ENV=prod     生产环境（默认，不启动 MongoDB）
  DEPLOY_ENV=test     测试环境（启动 MongoDB）

命令列表:
  deploy          完整部署（根据环境启动服务）
  start           启动所有服务
  stop            停止所有服务
  restart         重启所有服务
  status          查看服务状态
  logs            查看所有日志
  logs-service    查看应用日志
  logs-mongo      查看 MongoDB 日志
  logs-nginx      查看 Nginx 日志
  health          健康检查
  shell           进入应用容器
  shell-mongo     进入 MongoDB
  backup          备份 MongoDB 数据
  update          更新服务（需要新镜像）
  clean           清理停止的容器
  help            显示此帮助

示例:
  $0 deploy                    # 生产环境部署（默认）
  DEPLOY_ENV=test $0 deploy    # 测试环境部署
  $0 start                     # 启动服务
  $0 logs                      # 查看日志
  $0 health                    # 健康检查

EOF
}

wait_for_mongo() {
    echo "等待 MongoDB 就绪..."
    for i in {1..30}; do
        if docker-compose -f "$COMPOSE_FILE" exec -T mongodb mongosh --quiet --eval "db.runCommand({ping:1})" &>/dev/null; then
            echo "✅ MongoDB 已就绪"
            return 0
        fi
        echo -n "."
        sleep 2
    done
    echo ""
    echo "❌ MongoDB 启动超时"
    return 1
}

wait_for_service() {
    echo "等待应用服务就绪..."
    for i in {1..30}; do
        if curl -sf http://localhost:8080/health &>/dev/null; then
            echo "✅ 应用服务已就绪"
            return 0
        fi
        echo -n "."
        sleep 2
    done
    echo ""
    echo "⚠️  应用服务启动超时（可能仍在启动中）"
    return 1
}

cmd_deploy() {
    echo "=========================================="
    echo "开始部署 Bearer Token Service"
    echo "环境: $DEPLOY_ENV"
    echo "=========================================="
    echo ""

    if [ ! -f ".env" ]; then
        echo "❌ 错误: 未找到 .env 文件"
        echo "请先执行: cp .env.example .env"
        exit 1
    fi

    # 根据环境决定是否启动 MongoDB
    if [[ "$DEPLOY_ENV" == "test" ]]; then
        echo "步骤 1/4: 启动 MongoDB..."
        docker-compose -f "$COMPOSE_FILE" up -d mongodb
        wait_for_mongo || exit 1

        echo ""
        echo "步骤 2/4: 初始化数据库..."
        docker-compose -f "$COMPOSE_FILE" run --rm mongodb-init
    else
        echo "生产环境：跳过 MongoDB 启动（使用外部数据库）"

        # 检查 MONGO_URI
        source .env
        if [[ -z "$MONGO_URI" ]]; then
            echo "❌ 错误: 生产环境必须配置外部 MONGO_URI"
            echo "请在 .env 文件中设置 MONGO_URI"
            exit 1
        fi
        echo "✅ 已配置外部 MongoDB"
    fi

    echo ""
    echo "步骤 3/4: 启动应用服务..."
    if [[ "$DEPLOY_ENV" == "prod" ]]; then
        docker-compose -f "$COMPOSE_FILE" up -d --no-deps bearer-token-service-production
    else
        docker-compose -f "$COMPOSE_FILE" up -d bearer-token-service
    fi
    wait_for_service

    echo ""
    read -p "是否启动 Nginx? (y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "步骤 4/4: 启动 Nginx..."
        if [[ "$DEPLOY_ENV" == "prod" ]]; then
            docker-compose -f "$COMPOSE_FILE" up -d --no-deps nginx
        else
            docker-compose -f "$COMPOSE_FILE" up -d nginx
        fi
        sleep 5
        echo "✅ Nginx 已启动"
    else
        echo "跳过 Nginx"
    fi

    echo ""
    echo "=========================================="
    echo "🎉 部署完成！"
    echo "=========================================="
    cmd_status
}

cmd_start() {
    echo "启动服务（环境: $DEPLOY_ENV）..."

    if [[ "$DEPLOY_ENV" == "test" ]]; then
        docker-compose -f "$COMPOSE_FILE" up -d mongodb
        wait_for_mongo
        docker-compose -f "$COMPOSE_FILE" up -d bearer-token-service
    else
        docker-compose -f "$COMPOSE_FILE" up -d --no-deps bearer-token-service-production
    fi

    wait_for_service
    docker-compose -f "$COMPOSE_FILE" up -d --no-deps nginx
    echo "✅ 所有服务已启动"
}

cmd_stop() {
    echo "停止服务..."
    docker-compose -f "$COMPOSE_FILE" down
    echo "✅ 所有服务已停止"
}

cmd_restart() {
    echo "重启服务（环境: $DEPLOY_ENV）..."

    if [[ "$DEPLOY_ENV" == "test" ]]; then
        echo "1. 重启 MongoDB..."
        docker-compose -f "$COMPOSE_FILE" restart mongodb
        wait_for_mongo
    fi

    echo "2. 重启应用服务..."
    if [[ "$DEPLOY_ENV" == "prod" ]]; then
        docker-compose -f "$COMPOSE_FILE" restart bearer-token-service-production
    else
        docker-compose -f "$COMPOSE_FILE" restart bearer-token-service
    fi
    wait_for_service

    echo "3. 重启 Nginx..."
    docker-compose -f "$COMPOSE_FILE" restart nginx
    echo "✅ 所有服务已重启"
}

cmd_status() {
    echo "=========================================="
    echo "服务状态"
    echo "=========================================="
    docker-compose -f "$COMPOSE_FILE" ps
    echo ""
    cmd_health
}

cmd_logs() {
    docker-compose -f "$COMPOSE_FILE" logs -f --tail=100
}

cmd_logs_service() {
    docker-compose -f "$COMPOSE_FILE" logs -f --tail=100 bearer-token-service
}

cmd_logs_mongo() {
    docker-compose -f "$COMPOSE_FILE" logs -f --tail=100 mongodb
}

cmd_logs_nginx() {
    docker-compose -f "$COMPOSE_FILE" logs -f --tail=100 nginx
}

cmd_health() {
    echo "健康检查:"
    echo "---"

    # 检查 MongoDB
    if docker-compose -f "$COMPOSE_FILE" exec -T mongodb mongosh --quiet --eval "db.runCommand({ping:1})" &>/dev/null; then
        echo "✅ MongoDB: 正常"
    else
        echo "❌ MongoDB: 异常"
    fi

    # 检查应用服务
    if curl -sf http://localhost:8080/health &>/dev/null; then
        echo "✅ Bearer Token Service: 正常"
    else
        echo "❌ Bearer Token Service: 异常"
    fi

    # 检查 Nginx
    if curl -sf http://localhost/health &>/dev/null 2>&1; then
        echo "✅ Nginx: 正常"
    else
        echo "⚠️  Nginx: 未启动或异常"
    fi
}

cmd_shell() {
    if [[ "$DEPLOY_ENV" == "prod" ]]; then
        docker-compose -f "$COMPOSE_FILE" exec bearer-token-service-production sh
    else
        docker-compose -f "$COMPOSE_FILE" exec bearer-token-service sh
    fi
}

cmd_shell_mongo() {
    docker-compose -f "$COMPOSE_FILE" exec mongodb mongosh -u "${MONGO_ROOT_USERNAME:-admin}"
}

cmd_backup() {
    BACKUP_DIR="backup"
    BACKUP_FILE="mongodb-backup-$(date +%Y%m%d-%H%M%S).archive"

    mkdir -p "$BACKUP_DIR"

    echo "开始备份 MongoDB..."
    docker-compose -f "$COMPOSE_FILE" exec -T mongodb mongodump \
        --username="${MONGO_ROOT_USERNAME:-admin}" \
        --password="${MONGO_ROOT_PASSWORD:-changeme}" \
        --archive > "$BACKUP_DIR/$BACKUP_FILE"

    gzip "$BACKUP_DIR/$BACKUP_FILE"

    echo "✅ 备份完成: $BACKUP_DIR/$BACKUP_FILE.gz"
}

cmd_update() {
    echo "更新服务（环境: $DEPLOY_ENV）..."
    echo "⚠️  请确保已加载新的 Docker 镜像"
    read -p "是否继续? (y/N): " -n 1 -r
    echo ""

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "取消更新"
        exit 0
    fi

    echo "重启应用服务..."
    if [[ "$DEPLOY_ENV" == "prod" ]]; then
        docker-compose -f "$COMPOSE_FILE" up -d --no-deps bearer-token-service-production
    else
        docker-compose -f "$COMPOSE_FILE" up -d bearer-token-service
    fi
    wait_for_service
    echo "✅ 服务已更新"
}

cmd_clean() {
    echo "清理停止的容器..."
    docker-compose -f "$COMPOSE_FILE" rm -f
    echo "✅ 清理完成"
}

# 主逻辑
case "${1:-}" in
    deploy)         cmd_deploy ;;
    start)          cmd_start ;;
    stop)           cmd_stop ;;
    restart)        cmd_restart ;;
    status)         cmd_status ;;
    logs)           cmd_logs ;;
    logs-service)   cmd_logs_service ;;
    logs-mongo)     cmd_logs_mongo ;;
    logs-nginx)     cmd_logs_nginx ;;
    health)         cmd_health ;;
    shell)          cmd_shell ;;
    shell-mongo)    cmd_shell_mongo ;;
    backup)         cmd_backup ;;
    update)         cmd_update ;;
    clean)          cmd_clean ;;
    help|--help|-h) show_help ;;
    *)
        echo "❌ 未知命令: ${1:-}"
        echo ""
        show_help
        exit 1
        ;;
esac
