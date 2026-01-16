# Docker 安装指南

## 问题：docker: command not found

如果你遇到 `docker: command not found` 或 `docker compose` 不可用的错误，说明 Docker 还没有安装。

---

## 🚀 快速安装（一键脚本）

### 使用安装脚本（推荐）

```bash
# 1. 运行安装脚本
cd /opt/src/auth/bearer-token-service.v2
bash scripts/install-docker.sh

# 2. 重新登录或刷新用户组
newgrp docker

# 3. 验证安装
docker --version
docker compose version

# 4. 继续部署
make deploy
```

---

## 📦 手动安装

### Ubuntu/Debian 系统

```bash
# 1. 更新软件包
sudo apt-get update

# 2. 安装依赖
sudo apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

# 3. 添加 Docker 官方 GPG 密钥
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

# 4. 设置仓库
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# 5. 安装 Docker 和 Docker Compose
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 6. 启动 Docker
sudo systemctl start docker
sudo systemctl enable docker

# 7. 添加用户到 docker 组（避免每次都用 sudo）
sudo usermod -aG docker $USER

# 8. 重新登录或刷新用户组
newgrp docker

# 9. 验证安装
docker --version
docker compose version
```

### CentOS/RHEL 系统

```bash
# 1. 安装依赖
sudo yum install -y yum-utils

# 2. 添加仓库
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo

# 3. 安装 Docker
sudo yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 4. 启动 Docker
sudo systemctl start docker
sudo systemctl enable docker

# 5. 添加用户到 docker 组
sudo usermod -aG docker $USER

# 6. 重新登录
newgrp docker
```

---

## ✅ 验证安装

```bash
# 检查 Docker 版本
docker --version
# 预期输出: Docker version 24.0.x, build xxxxx

# 检查 Docker Compose 版本
docker compose version
# 预期输出: Docker Compose version v2.x.x

# 测试 Docker
docker run --rm hello-world

# 检查 Docker 服务状态
sudo systemctl status docker
```

---

## 🔧 常见问题

### 问题 1: Got permission denied while trying to connect to the Docker daemon socket

**原因**: 当前用户没有 Docker 权限

**解决**:
```bash
# 添加用户到 docker 组
sudo usermod -aG docker $USER

# 重新登录或刷新用户组
newgrp docker

# 或者重新登录系统
exit
# 然后重新 SSH 登录
```

### 问题 2: docker compose: command not found

**原因**: Docker Compose 插件未安装

**解决**:
```bash
# Ubuntu/Debian
sudo apt-get install -y docker-compose-plugin

# CentOS/RHEL
sudo yum install -y docker-compose-plugin

# 验证
docker compose version
```

### 问题 3: Cannot connect to the Docker daemon

**原因**: Docker 服务未启动

**解决**:
```bash
# 启动 Docker 服务
sudo systemctl start docker

# 设置开机自启
sudo systemctl enable docker

# 检查状态
sudo systemctl status docker
```

### 问题 4: 旧版本 docker-compose (v1.x)

**问题**: 系统上安装的是旧版本 `docker-compose` (带连字符)

**解决**:
```bash
# 卸载旧版本
sudo apt-get remove docker-compose

# 安装新版本插件
sudo apt-get install -y docker-compose-plugin

# 验证（注意是空格，不是连字符）
docker compose version
```

---

## 🚀 安装完成后

### 1. 验证环境

```bash
cd /opt/src/auth/bearer-token-service.v2

# 检查 Docker
docker --version
docker compose version

# 检查编译后的二进制
ls -lh bin/tokenserv
```

### 2. 继续部署

```bash
# 方式 1: 使用 Makefile
make deploy

# 方式 2: 直接使用 docker compose
docker compose up -d

# 验证服务
curl http://localhost/health
```

### 3. 查看服务状态

```bash
# 查看容器状态
docker compose ps

# 查看日志
docker compose logs -f

# 查看特定服务日志
docker compose logs -f nginx
docker compose logs -f bearer-token-service
```

---

## 📚 Docker Compose 版本说明

### V1 (旧版，已弃用)

```bash
docker-compose --version    # 带连字符
docker-compose up -d
```

### V2 (新版，推荐)

```bash
docker compose version      # 空格
docker compose up -d
```

**重要**: 本项目使用 Docker Compose V2 (空格版本)。

---

## 🔍 卸载 Docker（如果需要）

```bash
# Ubuntu/Debian
sudo apt-get purge -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo rm -rf /var/lib/docker
sudo rm -rf /var/lib/containerd

# CentOS/RHEL
sudo yum remove -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo rm -rf /var/lib/docker
sudo rm -rf /var/lib/containerd
```

---

## 📞 获取帮助

如果遇到其他问题:

1. 查看 Docker 官方文档: https://docs.docker.com/engine/install/
2. 运行安装脚本: `bash scripts/install-docker.sh`
3. 查看系统日志: `sudo journalctl -u docker.service`

---

**版本**: v1.0
**更新日期**: 2025-12-26
**适用系统**: Ubuntu 20.04+, Debian 10+, CentOS 7+
