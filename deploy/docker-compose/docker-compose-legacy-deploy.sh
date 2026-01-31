#!/bin/bash
# ========================================
# Bearer Token Service - Legacy Deployment Script
# ========================================
# 适用于 docker-compose 1.25.0+
# 因为旧版本不支持 depends_on 的 service_healthy condition
# 需要手动等待服务就绪
#
# 用法:
#   ./docker-compose-legacy-deploy.sh [prod|test]
#
# 环境说明:
#   prod (默认) - 生产环境，不启动 MongoDB，需要外部 MongoDB
#   test        - 测试环境，启动完整栈（包括 MongoDB）

set -e

# ========================================
# 参数解析
# ========================================
DEPLOY_ENV="${1:-prod}"

if [[ "$DEPLOY_ENV" != "prod" && "$DEPLOY_ENV" != "test" ]]; then
    echo "❌ 错误: 无效的环境参数"
    echo "用法: $0 [prod|test]"
    echo ""
    echo "  prod - 生产环境（不启动 MongoDB）"
    echo "  test - 测试环境（启动 MongoDB）"
    exit 1
fi

COMPOSE_FILE="docker-compose.legacy.yml"
ENV_FILE=".env"

echo "========================================"
echo "Bearer Token Service - Legacy Deployment"
echo "环境: $DEPLOY_ENV"
echo "========================================"
echo ""

# 检查 docker-compose 版本
COMPOSE_VERSION=$(docker-compose --version | grep -oP '\d+\.\d+\.\d+' || echo "unknown")
echo "检测到 docker-compose 版本: $COMPOSE_VERSION"
echo ""

# 检查环境变量文件
if [ ! -f "$ENV_FILE" ]; then
    echo "❌ 错误: 找不到 .env 文件"
    echo "请先复制模板: cp .env.example .env"
    exit 1
fi

echo "✅ 环境变量文件已找到"
echo ""

# 生产环境检查 MONGO_URI
if [[ "$DEPLOY_ENV" == "prod" ]]; then
    source "$ENV_FILE"
    if [[ -z "$MONGO_URI" ]]; then
        echo "❌ 错误: 生产环境必须配置外部 MONGO_URI"
        echo "请在 .env 文件中设置: MONGO_URI=mongodb://user:pass@host:port/database"
        exit 1
    fi
    echo "✅ 已配置外部 MongoDB: ${MONGO_URI%%@*}@***"
    echo ""
fi

# ========================================
# 部署流程
# ========================================

STEP=1
TOTAL_STEPS=5

# 启动 Redis（生产和测试环境都需要）
echo "步骤 $STEP/$TOTAL_STEPS: 启动 Redis..."
docker-compose -f "$COMPOSE_FILE" up -d --pull never redis

echo ""
echo "等待 Redis 就绪（最多等待 30 秒）..."
for i in {1..15}; do
    if docker exec bearer-token-redis redis-cli ping 2>/dev/null | grep -q PONG; then
        echo "✅ Redis 已就绪"
        break
    fi

    if [ $i -eq 15 ]; then
        echo "⚠️  警告: Redis 启动超时"
        echo "查看日志: docker-compose -f $COMPOSE_FILE logs redis"
        read -p "是否继续部署? (y/N): " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi

    echo -n "."
    sleep 2
done
((STEP++))

echo ""

# 测试环境：启动 MongoDB
if [[ "$DEPLOY_ENV" == "test" ]]; then
    echo "步骤 $STEP/$TOTAL_STEPS: 启动 MongoDB..."
    docker-compose -f "$COMPOSE_FILE" up -d --pull never mongodb
    ((STEP++))

    echo ""
    echo "步骤 $STEP/$TOTAL_STEPS: 等待 MongoDB 就绪（最多等待 60 秒）..."
    for i in {1..30}; do
        if docker-compose -f "$COMPOSE_FILE" exec -T mongodb mongosh --quiet --eval "db.runCommand({ping:1})" &>/dev/null; then
            echo "✅ MongoDB 已就绪"
            break
        fi

        if [ $i -eq 30 ]; then
            echo "❌ 错误: MongoDB 启动超时"
            echo "查看日志: docker-compose -f $COMPOSE_FILE logs mongodb"
            exit 1
        fi

        echo -n "."
        sleep 2
    done
    ((STEP++))

    echo ""
    echo "步骤 $STEP/$TOTAL_STEPS: 初始化数据库索引..."
    docker-compose -f "$COMPOSE_FILE" run --rm mongodb-init

    if [ $? -eq 0 ]; then
        echo "✅ 数据库索引创建完成"
    else
        echo "⚠️  警告: 索引创建失败（可能已存在）"
        echo "查看日志: docker-compose -f $COMPOSE_FILE logs mongodb-init"
        read -p "是否继续部署? (y/N): " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    ((STEP++))
else
    echo "生产环境：跳过 MongoDB 启动（使用外部数据库）"
    echo ""
fi

# 启动 Bearer Token Service
echo "步骤 $STEP/$TOTAL_STEPS: 启动 Bearer Token Service..."
if [[ "$DEPLOY_ENV" == "prod" ]]; then
    # 生产环境：启动 bearer-token-service-production（连接外部 MongoDB）
    docker-compose -f "$COMPOSE_FILE" up -d --pull never --no-deps bearer-token-service-production
else
    # 测试环境：启动 bearer-token-service（连接本地 MongoDB）
    docker-compose -f "$COMPOSE_FILE" up -d --pull never bearer-token-service
fi
((STEP++))

echo ""
echo "步骤 $STEP/$TOTAL_STEPS: 等待服务就绪（最多等待 30 秒）..."

# 根据环境选择容器名称
if [[ "$DEPLOY_ENV" == "prod" ]]; then
    SERVICE_NAME="bearer-token-service-production"
    CONTAINER_NAME="bearer-token-service-prod"
else
    SERVICE_NAME="bearer-token-service"
    CONTAINER_NAME="bearer-token-service"
fi

for i in {1..15}; do
    if docker exec $CONTAINER_NAME curl -sf http://localhost:8080/health &>/dev/null; then
        echo "✅ Bearer Token Service 已就绪"
        break
    fi

    if [ $i -eq 15 ]; then
        echo "⚠️  警告: 服务健康检查超时"
        echo "查看日志: docker-compose -f $COMPOSE_FILE logs $SERVICE_NAME"
        echo "服务可能仍在启动中..."
    fi

    echo -n "."
    sleep 2
done
((STEP++))

# 启动 Nginx（可选）
echo ""
echo "步骤 $STEP/$TOTAL_STEPS: 启动 Nginx（可选）..."
read -p "是否启动 Nginx? (y/N): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if [[ "$DEPLOY_ENV" == "prod" ]]; then
        # 生产环境：不启动依赖服务
        docker-compose -f "$COMPOSE_FILE" up -d --pull never --no-deps nginx
    else
        # 测试环境：正常启动
        docker-compose -f "$COMPOSE_FILE" up -d --pull never nginx
    fi

    echo "等待 Nginx 就绪..."
    sleep 5

    if curl -sf http://localhost/health &>/dev/null; then
        echo "✅ Nginx 已就绪"
    else
        echo "⚠️  警告: Nginx 健康检查失败"
    fi
else
    echo "跳过 Nginx 启动"
fi

echo ""
echo "========================================"
echo "🎉 部署完成！"
echo "========================================"
echo ""
echo "部署环境: $DEPLOY_ENV"
echo ""
echo "查看服务状态:"
echo "  docker-compose -f $COMPOSE_FILE ps"
echo ""
echo "查看日志:"
echo "  docker-compose -f $COMPOSE_FILE logs -f"
echo ""
echo "测试服务:"
echo "  curl http://localhost:8080/health"
echo ""

if [[ "$DEPLOY_ENV" == "prod" ]]; then
    echo "生产环境提示:"
    echo "  - 已连接外部 MongoDB: ${MONGO_URI%%@*}@***"
    echo "  - 已启动本地 Redis 缓存"
    echo "  - 确保数据库已初始化索引（运行 scripts/init/init-db.sh）"
else
    echo "测试环境提示:"
    echo "  - MongoDB 已启动并初始化"
    echo "  - Redis 缓存已启动"
    echo "  - 数据存储在 Docker volume: mongodb_data, redis_data"
fi

echo ""
echo "停止服务:"
echo "  docker-compose -f $COMPOSE_FILE down"
echo ""
