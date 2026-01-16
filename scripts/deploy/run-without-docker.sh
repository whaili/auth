#!/bin/bash

echo '========================================='
echo '无 Docker 部署 - 快速启动'
echo '========================================='
echo ''

# 检查 MongoDB
if ! pgrep -x mongod > /dev/null; then
    echo '⚠️  MongoDB 未运行'
    echo '   请先启动 MongoDB: sudo systemctl start mongod'
    echo '   或使用 Docker: docker run -d -p 27017:27017 --name mongodb mongo:7.0'
    exit 1
fi

echo '✅ MongoDB 正在运行'
echo ''

# 检查编译
if [ ! -f bin/tokenserv ]; then
    echo '编译服务...'
    go build -o bin/tokenserv cmd/server/main.go
fi

echo '✅ 二进制文件就绪'
echo ''

# 设置环境变量
export MONGO_URI="mongodb://localhost:27017"
export PORT="8080"
export ACCOUNT_FETCHER_MODE="local"
export QINIU_UID_MAPPER_MODE="simple"
export HMAC_TIMESTAMP_TOLERANCE="15m"

echo '启动服务...'
echo '端口: 8080'
echo 'MongoDB: mongodb://localhost:27017'
echo ''
echo '按 Ctrl+C 停止服务'
echo '========================================='
echo ''

# 运行服务
./bin/tokenserv
