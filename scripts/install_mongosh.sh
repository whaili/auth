#!/bin/bash
# 快速安装 mongosh (MongoDB Shell)

set -e

echo "========================================="
echo "安装 MongoDB Shell (mongosh)"
echo "========================================="
echo ""

# 检查是否已安装
if command -v mongosh &> /dev/null; then
    echo "✓ mongosh 已安装"
    mongosh --version
    exit 0
fi

echo "检测操作系统..."
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
    VERSION=$VERSION_ID
    echo "  操作系统: $PRETTY_NAME"
else
    echo "✗ 无法检测操作系统"
    exit 1
fi

if [ "$OS" != "ubuntu" ]; then
    echo "✗ 此脚本仅支持 Ubuntu"
    exit 1
fi

echo ""
echo "添加 MongoDB 仓库..."

# 添加 GPG key
wget -qO - https://www.mongodb.org/static/pgp/server-7.0.asc | sudo apt-key add - 2>/dev/null
echo "  ✓ GPG key 已添加"

# 添加仓库
echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/7.0 multiverse" | \
    sudo tee /etc/apt/sources.list.d/mongodb-org-7.0.list
echo "  ✓ MongoDB 仓库已添加"

echo ""
echo "更新软件包列表..."
sudo apt-get update -qq

echo ""
echo "安装 mongosh..."
sudo apt-get install -y mongodb-mongosh

echo ""
echo "========================================="
echo "安装完成"
echo "========================================="
echo ""
mongosh --version
echo ""
echo "使用示例:"
echo "  mongosh mongodb://localhost:27017"
echo "  mongosh mongodb://admin:password@localhost:27017?authSource=admin"
