# Docker Compose 生产部署指南

> Bearer Token Service V2 - 容器化部署完全手册

---

## 📋 目录

- [部署优势](#-部署优势)
- [前置要求](#-前置要求)
- [快速开始](#-快速开始)
- [生产部署](#-生产部署)
- [运维操作](#-运维操作)
- [监控与日志](#-监控与日志)
- [故障排查](#-故障排查)
- [安全最佳实践](#-安全最佳实践)
- [性能优化](#-性能优化)

---

## 🌟 部署优势

相比传统 systemd 部署,Docker Compose 提供:

| 特性 | systemd | Docker Compose |
|------|---------|----------------|
| **环境一致性** | ⚠️ 依赖系统环境 | ✅ 完全隔离 |
| **依赖管理** | ❌ 手动安装 MongoDB | ✅ 自动编排 |
| **快速部署** | ⚠️ 多步骤 | ✅ 一条命令 |
| **版本回滚** | ❌ 困难 | ✅ 简单 |
| **多环境部署** | ⚠️ 配置冲突 | ✅ 配置隔离 |
| **资源隔离** | ❌ 共享系统 | ✅ 容器隔离 |
| **水平扩展** | ❌ 困难 | ✅ 简单 |

---

## 📦 前置要求

### 1. 安装 Docker

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com | bash

# CentOS/RHEL
sudo yum install -y docker-ce docker-ce-cli containerd.io

# 启动 Docker
sudo systemctl start docker
sudo systemctl enable docker

# 验证安装
docker --version
```

### 2. 安装 Docker Compose

```bash
# 方式 1: Docker 插件（推荐）
sudo apt-get install docker-compose-plugin

# 方式 2: 独立二进制
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" \
  -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# 验证安装
docker compose version  # 或 docker-compose --version
```

### 3. 配置非 root 用户（可选）

```bash
sudo usermod -aG docker $USER
# 重新登录生效
```

---

## 🚀 快速开始

### 1. 编译服务

在 **vm-test** 或开发环境中编译:

```bash
cd /root/src/auth/bearer-token-service.v2

# 编译二进制文件
go build -o bin/tokenserv cmd/server/main.go

# 验证编译
ls -lh bin/tokenserv
```

### 2. 配置环境变量

```bash
# 复制配置模板
cp .env.example .env

# 编辑配置（使用默认值可快速开始）
vim .env
```

### 3. 启动服务

```bash
# 构建并启动所有服务
docker compose up -d

# 查看日志
docker compose logs -f

# 验证服务
curl http://localhost:8080/health
```

**预期输出**:
```json
{"status":"ok"}
```

### 4. 测试 API

```bash
# 注册账户
curl -X POST http://localhost:8080/api/v2/accounts/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "company": "Test Inc",
    "password": "test123456"
  }'

# 保存返回的 AccessKey 和 SecretKey
```

---

## 🏭 生产部署

### 1. 修改生产配置

编辑 `.env` 文件:

```bash
# ========================================
# 生产环境配置
# ========================================

# 版本控制
VERSION=v2.0.0

# MongoDB 强密码（务必修改！）
MONGO_ROOT_USERNAME=prod_admin
MONGO_ROOT_PASSWORD=YourSecurePassword_123!@#

# 外部账户系统（如果使用）
ACCOUNT_FETCHER_MODE=external
EXTERNAL_ACCOUNT_API_URL=https://account-api.yourcompany.com
EXTERNAL_ACCOUNT_API_TOKEN=prod_token_xyz123456

# UID 映射（生产推荐 database 模式）
QINIU_UID_MAPPER_MODE=database
QINIU_UID_AUTO_CREATE=false

# 更严格的安全配置
HMAC_TIMESTAMP_TOLERANCE=10m

# 自定义端口（如果需要）
HOST_PORT=18080
```

### 2. 修改 Docker Compose（生产增强）

编辑 `docker-compose.yml`:

```yaml
# 取消 MongoDB 端口暴露（安全）
# ports:
#   - "27017:27017"

# 启用资源限制
services:
  bearer-token-service:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
    # 启用日志轮转
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### 3. 数据持久化检查

```bash
# 查看数据卷
docker volume ls

# 备份数据卷
docker run --rm -v bearer-token-service_mongodb_data:/data \
  -v $(pwd)/backup:/backup \
  alpine tar czf /backup/mongodb-backup-$(date +%Y%m%d).tar.gz /data
```

### 4. 启动生产服务

```bash
# 构建镜像（带版本标签）
docker compose build --no-cache

# 启动服务
docker compose up -d

# 查看服务状态
docker compose ps

# 查看实时日志
docker compose logs -f bearer-token-service
```

---

## 🔧 运维操作

### 常用命令

```bash
# ========================================
# 服务管理
# ========================================

# 启动服务
docker compose up -d

# 停止服务
docker compose down

# 重启服务
docker compose restart

# 停止并删除所有资源（包括数据卷，慎用！）
docker compose down -v

# ========================================
# 查看状态
# ========================================

# 查看运行状态
docker compose ps

# 查看日志
docker compose logs -f                          # 所有服务
docker compose logs -f bearer-token-service     # 指定服务
docker compose logs --tail=100 bearer-token-service  # 最近 100 行

# 查看资源使用
docker stats

# ========================================
# 进入容器
# ========================================

# 进入服务容器
docker compose exec bearer-token-service sh

# 进入 MongoDB 容器
docker compose exec mongodb mongosh -u admin -p changeme

# ========================================
# 更新服务
# ========================================

# 1. 在 vm-test 重新编译
go build -o bin/tokenserv cmd/server/main.go

# 2. 重新构建镜像
docker compose build bearer-token-service

# 3. 重启服务（滚动更新）
docker compose up -d bearer-token-service

# 4. 验证新版本
docker compose logs -f bearer-token-service
curl http://localhost:8080/health

# ========================================
# 版本回滚
# ========================================

# 切换到旧版本镜像
docker tag bearer-token-service:v2.0.0 bearer-token-service:latest
docker compose up -d bearer-token-service
```

### 使用 Makefile（推荐）

```bash
# 查看所有命令
make help

# 常用操作
make build          # 构建镜像
make up             # 启动服务
make down           # 停止服务
make restart        # 重启服务
make logs           # 查看日志
make status         # 查看状态
make clean          # 清理资源
```

---

## 📊 监控与日志

### 1. 健康检查

```bash
# 手动检查
curl http://localhost:8080/health

# 持续监控
watch -n 5 'curl -s http://localhost:8080/health | jq'

# Docker 健康状态
docker compose ps
# healthy 表示健康，unhealthy 表示异常
```

### 2. 日志管理

```bash
# 实时日志
docker compose logs -f

# 按时间过滤
docker compose logs --since 30m bearer-token-service

# 导出日志
docker compose logs --no-color > logs/service-$(date +%Y%m%d).log

# 查看应用日志（容器内）
docker compose exec bearer-token-service cat /app/logs/service.log
```

### 3. 性能监控

```bash
# 资源使用
docker stats --no-stream

# 容器详细信息
docker compose exec bearer-token-service cat /proc/meminfo
docker compose exec bearer-token-service cat /proc/cpuinfo
```

### 4. 集成监控工具（可选）

推荐集成 Prometheus + Grafana:

```yaml
# docker-compose.monitoring.yml
services:
  prometheus:
    image: prom/prometheus
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana
    ports:
      - "3000:3000"
```

---

## 🔍 故障排查

### 问题 1: 服务无法启动

```bash
# 1. 查看日志
docker compose logs bearer-token-service

# 2. 检查端口占用
sudo netstat -tlnp | grep 8080

# 3. 检查环境变量
docker compose config

# 4. 重新构建
docker compose down
docker compose build --no-cache
docker compose up -d
```

### 问题 2: MongoDB 连接失败

```bash
# 1. 检查 MongoDB 状态
docker compose ps mongodb

# 2. 查看 MongoDB 日志
docker compose logs mongodb

# 3. 手动连接测试
docker compose exec mongodb mongosh -u admin -p changeme

# 4. 检查网络
docker compose exec bearer-token-service ping mongodb
```

### 问题 3: 健康检查失败

```bash
# 1. 容器内测试
docker compose exec bearer-token-service curl http://localhost:8080/health

# 2. 检查端口监听
docker compose exec bearer-token-service netstat -tlnp

# 3. 查看进程
docker compose exec bearer-token-service ps aux
```

### 问题 4: 外部无法访问

```bash
# 1. 检查端口映射
docker compose ps

# 2. 检查防火墙
sudo ufw status
sudo iptables -L -n

# 3. 测试本机访问
curl http://localhost:8080/health

# 4. 测试外部访问
curl http://SERVER_IP:8080/health
```

---

## 🔐 安全最佳实践

### 1. 强密码策略

```bash
# 生成强密码
openssl rand -base64 32

# 更新 .env
MONGO_ROOT_PASSWORD=$(openssl rand -base64 32)
```

### 2. 限制网络访问

```yaml
# docker-compose.yml
services:
  mongodb:
    # 不暴露端口到宿主机
    # ports:
    #   - "27017:27017"
    expose:
      - "27017"  # 仅容器间访问
```

### 3. 使用 Docker Secrets（Swarm 模式）

```yaml
secrets:
  mongo_password:
    external: true

services:
  mongodb:
    secrets:
      - mongo_password
    environment:
      MONGO_INITDB_ROOT_PASSWORD_FILE: /run/secrets/mongo_password
```

### 4. 定期更新

```bash
# 更新基础镜像
docker compose pull

# 重建服务
docker compose up -d --build
```

### 5. 限制容器权限

```yaml
services:
  bearer-token-service:
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs:
      - /tmp
```

---

## ⚡ 性能优化

### 1. 资源限制

```yaml
services:
  bearer-token-service:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '1.0'
          memory: 1G
```

### 2. MongoDB 调优

```yaml
services:
  mongodb:
    command: mongod --wiredTigerCacheSizeGB 1.5
```

### 3. 网络优化

```yaml
networks:
  bearer-token-net:
    driver: bridge
    driver_opts:
      com.docker.network.driver.mtu: 1450
```

### 4. 日志优化

```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

---

## 📚 附录

### A. 完整部署流程（生产）

```bash
# 1. 编译服务
cd /root/src/auth/bearer-token-service.v2
go build -o bin/tokenserv cmd/server/main.go

# 2. 配置环境
cp .env.example .env
vim .env  # 修改生产配置

# 3. 构建镜像
docker compose build

# 4. 启动服务
docker compose up -d

# 5. 验证部署
docker compose ps
docker compose logs -f
curl http://localhost:8080/health

# 6. 测试 API
./tests/test_api.sh
```

### B. 备份与恢复

```bash
# 备份 MongoDB
docker compose exec mongodb mongodump \
  --uri="mongodb://admin:changeme@localhost:27017" \
  --out=/backup

# 恢复 MongoDB
docker compose exec mongodb mongorestore \
  --uri="mongodb://admin:changeme@localhost:27017" \
  /backup
```

### C. 迁移到生产服务器

```bash
# 1. 导出镜像
docker save bearer-token-service:latest | gzip > bearer-token-service.tar.gz

# 2. 传输到生产服务器
scp bearer-token-service.tar.gz user@prod-server:/tmp/

# 3. 在生产服务器导入
ssh user@prod-server
docker load < /tmp/bearer-token-service.tar.gz

# 4. 复制配置文件
scp .env docker-compose.yml user@prod-server:/opt/bearer-token-service/

# 5. 启动服务
ssh user@prod-server
cd /opt/bearer-token-service
docker compose up -d
```

---

## 🎯 快速命令参考

```bash
# 启动
docker compose up -d

# 停止
docker compose down

# 重启
docker compose restart

# 查看日志
docker compose logs -f

# 查看状态
docker compose ps

# 进入容器
docker compose exec bearer-token-service sh

# 更新服务
docker compose up -d --build

# 清理资源
docker compose down -v
docker system prune -a
```

---

**文档版本**: v1.0
**更新日期**: 2025-12-26
**适用版本**: Bearer Token Service V2.0.0
