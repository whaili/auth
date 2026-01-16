#!/bin/bash
set -e

echo "=========================================="
echo "Bearer Token Service V2 - 部署脚本"
echo "=========================================="

# 配置
PROJECT_DIR="/root/src/auth"
BINARY_NAME="bearer-token-service-v2"
SERVICE_PORT=8080
MONGO_URI="${MONGO_URI:-mongodb://localhost:27017}"
MONGO_DATABASE="${MONGO_DATABASE:-token_service_v2}"

echo ""
echo "1. 检查 MongoDB 连接..."
if ! mongosh --quiet --eval "db.adminCommand('ping')" "$MONGO_URI" >/dev/null 2>&1; then
    echo "❌ MongoDB 未运行，正在启动..."
    if ! systemctl is-active --quiet mongod; then
        sudo systemctl start mongod
        sleep 3
    fi

    # 如果还是不行，尝试 docker
    if ! mongosh --quiet --eval "db.adminCommand('ping')" "$MONGO_URI" >/dev/null 2>&1; then
        echo "尝试使用 Docker 启动 MongoDB..."
        docker ps -a | grep mongodb >/dev/null 2>&1 && docker rm -f mongodb
        docker run -d --name mongodb -p 27017:27017 mongo:latest
        sleep 5
    fi
fi

echo "✅ MongoDB 已就绪"

echo ""
echo "2. 编译服务..."
cd "$PROJECT_DIR"
go build -o "bin/$BINARY_NAME" cmd/server/main.go
echo "✅ 编译完成: bin/$BINARY_NAME"

echo ""
echo "3. 停止旧服务..."
pkill -f "$BINARY_NAME" 2>/dev/null || true
sleep 1

echo ""
echo "4. 启动新服务..."
export MONGO_URI="$MONGO_URI"
export MONGO_DATABASE="$MONGO_DATABASE"
export SERVER_PORT="$SERVICE_PORT"

nohup "./bin/$BINARY_NAME" > logs/service.log 2>&1 &
SERVICE_PID=$!

sleep 2

if ps -p $SERVICE_PID > /dev/null; then
    echo "✅ 服务启动成功 (PID: $SERVICE_PID)"
    echo ""
    echo "服务信息:"
    echo "  - 端口: $SERVICE_PORT"
    echo "  - MongoDB: $MONGO_URI"
    echo "  - Database: $MONGO_DATABASE"
    echo "  - 日志: logs/service.log"
    echo ""
    echo "查看日志: tail -f logs/service.log"
    echo "停止服务: pkill -f $BINARY_NAME"
else
    echo "❌ 服务启动失败"
    cat logs/service.log
    exit 1
fi

echo ""
echo "5. 等待服务就绪..."
for i in {1..10}; do
    if curl -s http://localhost:$SERVICE_PORT/health >/dev/null 2>&1; then
        echo "✅ 服务已就绪"
        exit 0
    fi
    sleep 1
done

echo "⚠️ 服务可能还在启动中，请检查日志"
