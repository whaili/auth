#!/bin/bash

# ========================================
# Docker 和 Docker Compose 安装脚本
# ========================================
# 适用于 Ubuntu/Debian 系统

set -e

echo "========================================="
echo "Docker 和 Docker Compose 安装脚本"
echo "========================================="
echo ""

# 检查是否为 root 或有 sudo 权限
if [ "$EUID" -ne 0 ] && ! command -v sudo &> /dev/null; then
    echo "❌ 错误: 需要 root 权限或 sudo"
    exit 1
fi

SUDO=""
if [ "$EUID" -ne 0 ]; then
    SUDO="sudo"
fi

# 检查系统类型
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
    VERSION=$VERSION_ID
else
    echo "❌ 错误: 无法检测操作系统"
    exit 1
fi

echo "检测到系统: $OS $VERSION"
echo ""

# ========================================
# 1. 安装 Docker
# ========================================

echo "步骤 1: 检查 Docker 是否已安装..."
if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version)
    echo "✅ Docker 已安装: $DOCKER_VERSION"

    read -p "是否重新安装 Docker? (y/N): " reinstall
    if [ "$reinstall" != "y" ]; then
        echo "跳过 Docker 安装"
    else
        echo "开始重新安装 Docker..."
        INSTALL_DOCKER=true
    fi
else
    echo "Docker 未安装，开始安装..."
    INSTALL_DOCKER=true
fi

if [ "$INSTALL_DOCKER" = true ]; then
    echo ""
    echo "步骤 2: 卸载旧版本（如果存在）..."
    $SUDO apt-get remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true

    echo ""
    echo "步骤 3: 更新软件包索引..."
    $SUDO apt-get update

    echo ""
    echo "步骤 4: 安装必要的软件包..."
    $SUDO apt-get install -y \
        ca-certificates \
        curl \
        gnupg \
        lsb-release

    echo ""
    echo "步骤 5: 添加 Docker 官方 GPG 密钥..."
    $SUDO mkdir -p /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/$OS/gpg | $SUDO gpg --dearmor -o /etc/apt/keyrings/docker.gpg

    echo ""
    echo "步骤 6: 设置 Docker 仓库..."
    echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$OS \
      $(lsb_release -cs) stable" | $SUDO tee /etc/apt/sources.list.d/docker.list > /dev/null

    echo ""
    echo "步骤 7: 更新软件包索引..."
    $SUDO apt-get update

    echo ""
    echo "步骤 8: 安装 Docker Engine 和 Docker Compose 插件..."
    $SUDO apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

    echo ""
    echo "✅ Docker 安装完成！"
fi

# ========================================
# 2. 配置 Docker
# ========================================

echo ""
echo "步骤 9: 启动 Docker 服务..."
$SUDO systemctl start docker
$SUDO systemctl enable docker

echo ""
echo "步骤 10: 验证 Docker 安装..."
docker --version
docker compose version

echo ""
echo "步骤 11: 测试 Docker..."
if $SUDO docker run --rm hello-world > /dev/null 2>&1; then
    echo "✅ Docker 测试成功！"
else
    echo "⚠️  Docker 测试失败，但安装可能成功"
fi

# ========================================
# 3. 配置用户权限（可选）
# ========================================

echo ""
read -p "是否将当前用户添加到 docker 组？(y/N): " add_user
if [ "$add_user" = "y" ]; then
    CURRENT_USER=${SUDO_USER:-$USER}
    echo "添加用户 $CURRENT_USER 到 docker 组..."
    $SUDO usermod -aG docker $CURRENT_USER
    echo "✅ 用户已添加到 docker 组"
    echo ""
    echo "⚠️  重要: 需要重新登录才能生效！"
    echo "   或者运行: newgrp docker"
fi

# ========================================
# 完成
# ========================================

echo ""
echo "========================================="
echo "✅ 安装完成！"
echo "========================================="
echo ""
echo "Docker 版本:"
docker --version
docker compose version
echo ""
echo "下一步:"
echo "  1. 如果添加了用户到 docker 组，请重新登录或运行: newgrp docker"
echo "  2. 进入项目目录: cd /opt/src/auth/bearer-token-service.v2"
echo "  3. 运行部署: make deploy"
echo ""
