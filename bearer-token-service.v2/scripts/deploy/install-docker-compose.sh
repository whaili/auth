#!/bin/bash

# ========================================
# 安装 Docker Compose 插件
# ========================================

set -e

echo "========================================="
echo "安装 Docker Compose 插件"
echo "========================================="
echo ""

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "❌ 错误: Docker 未安装"
    exit 1
fi

echo "✅ Docker 已安装: $(docker --version)"
echo ""

# 检查 Docker Compose 插件
if docker compose version &> /dev/null; then
    echo "✅ Docker Compose 插件已安装: $(docker compose version)"
    exit 0
fi

# 检查旧版 docker-compose
if command -v docker-compose &> /dev/null; then
    echo "⚠️  检测到旧版 docker-compose: $(docker-compose --version)"
    echo "建议安装新版插件"
    echo ""
fi

echo "开始安装 Docker Compose 插件..."
echo ""

# 安装插件
sudo apt-get update
sudo apt-get install -y docker-compose-plugin

echo ""
echo "========================================="
echo "✅ 安装完成！"
echo "========================================="
echo ""

# 验证
docker compose version

echo ""
echo "现在可以使用 'docker compose' 命令了"
echo ""
