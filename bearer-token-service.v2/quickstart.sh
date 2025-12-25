#!/bin/bash

# ========================================
# Bearer Token Service V2 - 快速启动脚本
# ========================================

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Bearer Token Service V2 - Quick Start${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 1. 检查 MongoDB
log_info "Checking MongoDB..."
if docker ps | grep -q mongodb-test; then
    log_success "MongoDB is already running"
else
    log_info "Starting MongoDB..."
    docker run -d -p 27017:27017 --name mongodb-test mongo:latest
    sleep 3
    log_success "MongoDB started"
fi

# 2. 检查 Go 依赖
log_info "Checking Go dependencies..."
cd /root/src/auth/bearer-token-service.v1/v2
if [ ! -d "vendor" ] && [ ! -f "go.sum" ]; then
    log_info "Downloading Go dependencies..."
    go mod download
    log_success "Dependencies downloaded"
else
    log_success "Dependencies already installed"
fi

# 3. 启动服务（后台）
log_info "Starting Bearer Token Service V2..."
nohup go run cmd/server/main.go > /tmp/token-service-v2.log 2>&1 &
SERVICE_PID=$!
sleep 3

# 检查服务是否启动成功
if curl -s http://localhost:8080/health > /dev/null; then
    log_success "Service started successfully (PID: $SERVICE_PID)"
    echo "SERVICE_PID=$SERVICE_PID" > /tmp/token-service-v2.pid
else
    log_warning "Service may not be ready yet, checking logs..."
    tail -10 /tmp/token-service-v2.log
fi

# 4. 运行测试
echo ""
log_info "Running automated tests..."
sleep 2

cd tests
./test_api.sh

# 5. 总结
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}🎉 Quick Start Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
log_info "Service is running at: http://localhost:8080"
log_info "Service logs: /tmp/token-service-v2.log"
log_info "Service PID: $SERVICE_PID"
echo ""
log_info "To view logs: tail -f /tmp/token-service-v2.log"
log_info "To stop service: kill $SERVICE_PID"
echo ""
log_success "Test credentials saved to: /tmp/v2_test_credentials.env"
echo ""
