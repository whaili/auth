#!/bin/bash

# 检查并停止已运行的服务
if pgrep -f "bin/tokenserv" > /dev/null; then
    echo "Stopping existing service..."
    pkill -f "bin/tokenserv" 2>/dev/null || true
    sleep 1
fi

export PORT=8081
export MONGO_URI=mongodb://admin:123456@localhost:27017
export ACCOUNT_FETCHER_MODE=local
export QINIU_UID_MAPPER_MODE=simple
export QINIU_UID_AUTO_CREATE=false
export HMAC_TIMESTAMP_TOLERANCE=15m

# Redis 缓存配置
export REDIS_ENABLED=true
export REDIS_ADDR=localhost:6379
export REDIS_DB=0
export CACHE_TOKEN_TTL=5m

echo "Starting Bearer Token Service on port 8081..."
echo "  Redis cache: ${REDIS_ENABLED:-false}"
./bin/tokenserv
