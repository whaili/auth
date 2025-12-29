# 数据库初始化指南

## 📋 概述

Bearer Token Service V2 在负载均衡多实例部署时，为了避免每个实例重复创建索引，提供了**独立的数据库初始化脚本**。

## 🎯 适用场景

### 使用数据库初始化脚本的场景

✅ **生产环境多实例负载均衡部署**
```
      负载均衡器 (Nginx/HAProxy)
            |
   +--------+--------+
   |        |        |
 实例1    实例2    实例3  ← 共享同一个 MongoDB
   |        |        |
   +--------+--------+
            |
        MongoDB
```

### 直接启动服务的场景

❌ **开发环境或单实例部署**
- 无需手动初始化数据库
- 程序启动时自动创建索引

---

## 🚀 快速开始

### 部署模式检测

初始化脚本会自动检测部署模式：

1. **外部 MongoDB**: 检测到 `MONGO_URI` 环境变量
2. **Docker MongoDB**: 未设置 `MONGO_URI`，自动连接 Docker 容器

---

### 模式 1: 外部 MongoDB（生产环境）

适用于使用外部 MongoDB 副本集（1主2备）的生产环境。

#### 步骤 1: 配置环境变量

```bash
# 设置副本集连接字符串
export MONGO_URI="mongodb://bearer_token_wr:password@10.70.65.39:27019,10.70.65.40:27019,10.70.65.41:27019/bearer_token_service?replicaSet=rs0&authSource=admin"
```

**重要**: MONGO_URI 必须包含：
- ✅ 所有副本集节点地址（主节点+从节点）
- ✅ 数据库名称（如 `/bearer_token_service`）
- ✅ 副本集名称（如 `?replicaSet=rs0`）
- ✅ 认证数据库（如 `&authSource=admin`）

#### 步骤 2: 安装 mongosh

```bash
# Ubuntu/Debian
sudo apt install mongodb-mongosh

# CentOS/RHEL
sudo yum install mongodb-mongosh

# 验证安装
mongosh --version
```

#### 步骤 3: 执行初始化

```bash
# 进入部署目录
cd /opt/src/auth/bearer-token-service.v2/dist/deploy

# 执行初始化脚本
./scripts/init/init-db.sh
```

**预期输出**：
```
========================================
Bearer Token Service V2 - 数据库初始化
========================================

🌐 检测到外部 MongoDB 配置
📋 配置信息:
   MONGO_URI: mongodb://***:***@10.70.65.39:27019,...

✅ 找到 mongosh 命令
✅ MongoDB 连接成功
🚀 开始创建索引...
========================================
✅ 数据库初始化成功！
========================================
```

---

### 模式 2: Docker MongoDB（本地开发/测试）

适用于使用 Docker Compose 内置 MongoDB 容器的环境。

#### 步骤 1: 初始化数据库

在**部署服务实例之前**，先运行初始化脚本：

```bash
# 进入项目目录（宿主机）
cd /root/src/auth/bearer-token-service.v2

# 不设置 MONGO_URI（自动检测 Docker 容器）
# 可选：设置数据库名称
export MONGO_DATABASE="token_service_v2"

# 执行初始化脚本
./scripts/init/init-db.sh
```

**输出示例**：
```
========================================
Bearer Token Service V2 - 数据库初始化
========================================

📋 配置信息:
   MONGO_URI: mongodb://localhost:27017
   MONGO_DATABASE: token_service_v2

🔍 检查依赖...
✅ 找到 mongosh 命令

🔌 测试 MongoDB 连接...
✅ MongoDB 连接成功

🚀 开始创建索引...

📊 创建 accounts 集合索引...
  ✅ 创建 email 唯一索引
  ✅ 创建 access_key 唯一索引
  ✅ 创建 status 索引
  ✅ 创建 qiniu_uid 唯一稀疏索引
  ✅ 创建 created_at 索引
✅ accounts 集合索引创建完成

📊 创建 tokens 集合索引...
  ✅ 创建 token 唯一索引
  ✅ 创建 account_id + is_active 复合索引（租户隔离）
  ✅ 创建 account_id + created_at 复合索引（查询优化）
  ✅ 创建 expires_at 索引（过期清理）
  ✅ 创建 last_used_at 索引（统计分析）
✅ tokens 集合索引创建完成

📊 创建 audit_logs 集合索引...
  ✅ 创建 account_id + timestamp 复合索引
  ✅ 创建 account_id + action 复合索引
  ✅ 创建 account_id + resource_id 复合索引
  ✅ 创建 timestamp 索引
  ✅ 创建 timestamp TTL 索引（90天自动删除）
✅ audit_logs 集合索引创建完成

=====================================
✅ 数据库初始化成功！
=====================================
```

---

### 步骤 2: 启动服务实例

初始化完成后，启动所有服务实例，并设置 `SKIP_INDEX_CREATION=true`：

```bash
# 启动实例 1
PORT=8080 SKIP_INDEX_CREATION=true ./bin/server

# 启动实例 2
PORT=8081 SKIP_INDEX_CREATION=true ./bin/server

# 启动实例 3
PORT=8082 SKIP_INDEX_CREATION=true ./bin/server
```

**日志输出**：
```
🚀 Bearer Token Service V2 - Starting...
✅ Connected to MongoDB
⏭️  Skipping index creation (SKIP_INDEX_CREATION=true)
ℹ️  Ensure indexes are created by running: scripts/init/init-db.sh
✅ Services initialized
...
✨ Bearer Token Service V2 is ready!
```

---

## 📊 数据库结构

### 数据库名称
- 默认：`token_service_v2`
- 可通过 `MONGO_DATABASE` 环境变量自定义

### 集合（Collections）

| 集合名称 | 用途 | 重要索引 |
|---------|------|---------|
| `accounts` | 账户信息 | `email` (unique), `access_key` (unique), `qiniu_uid` (unique, sparse) |
| `tokens` | Bearer Token | `token` (unique), `account_id + is_active` (租户隔离) |
| `audit_logs` | 审计日志 | `account_id + timestamp`, `timestamp` (TTL 90天) |

### 索引详情

#### accounts 集合
```javascript
{
  "email": 1              // 唯一索引
  "access_key": 1         // 唯一索引
  "status": 1
  "qiniu_uid": 1          // 唯一稀疏索引
  "created_at": -1
}
```

#### tokens 集合
```javascript
{
  "token": 1                      // 唯一索引
  "account_id": 1, "is_active": 1 // 租户隔离（核心）
  "account_id": 1, "created_at": -1
  "expires_at": 1
  "last_used_at": -1
}
```

#### audit_logs 集合
```javascript
{
  "account_id": 1, "timestamp": -1
  "account_id": 1, "action": 1
  "account_id": 1, "resource_id": 1
  "timestamp": -1                  // TTL 索引（90天自动删除）
}
```

---

## 🔧 环境变量配置

### 必需配置

| 变量名 | 说明 | 示例 |
|-------|------|------|
| `MONGO_URI` | MongoDB 连接字符串 | `mongodb://localhost:27017` |
| `SKIP_INDEX_CREATION` | 跳过启动时创建索引 | `true` (生产), `false` (开发) |

### 可选配置

| 变量名 | 说明 | 默认值 |
|-------|------|--------|
| `MONGO_DATABASE` | 数据库名称 | `token_service_v2` |
| `ACCOUNT_FETCHER_MODE` | 账户查询模式 | `local` |
| `QINIU_UID_MAPPER_MODE` | 七牛UID映射模式 | `simple` |

完整配置参考：`.env.production.example`

---

## 🐳 Docker Compose 部署示例

### docker-compose.yml 配置

```yaml
version: '3.8'

services:
  # MongoDB 服务
  mongodb:
    image: mongo:latest
    container_name: mongodb
    ports:
      - "27017:27017"
    volumes:
      - mongo_data:/data/db

  # 初始化数据库（仅执行一次）
  init-db:
    image: mongo:latest
    depends_on:
      - mongodb
    environment:
      MONGO_URI: mongodb://mongodb:27017
      MONGO_DATABASE: token_service_v2
    volumes:
      - ./scripts:/scripts
    command: >
      bash -c "
        sleep 5 &&
        mongosh mongodb://mongodb:27017/token_service_v2 /scripts/init/init-indexes.js
      "

  # 服务实例 1
  app1:
    image: bearer-token-service:v2
    depends_on:
      - init-db
    environment:
      PORT: 8080
      MONGO_URI: mongodb://mongodb:27017
      SKIP_INDEX_CREATION: "true"
    ports:
      - "8080:8080"

  # 服务实例 2
  app2:
    image: bearer-token-service:v2
    depends_on:
      - init-db
    environment:
      PORT: 8080
      MONGO_URI: mongodb://mongodb:27017
      SKIP_INDEX_CREATION: "true"
    ports:
      - "8081:8080"

  # 服务实例 3
  app3:
    image: bearer-token-service:v2
    depends_on:
      - init-db
    environment:
      PORT: 8080
      MONGO_URI: mongodb://mongodb:27017
      SKIP_INDEX_CREATION: "true"
    ports:
      - "8082:8080"

  # Nginx 负载均衡器
  nginx:
    image: nginx:alpine
    depends_on:
      - app1
      - app2
      - app3
    ports:
      - "80:80"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf

volumes:
  mongo_data:
```

### Nginx 负载均衡配置

```nginx
upstream bearer_token_service {
    server app1:8080;
    server app2:8080;
    server app3:8080;
}

server {
    listen 80;

    location / {
        proxy_pass http://bearer_token_service;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

---

## 🛠️ 手动管理索引

### 查看现有索引

```bash
mongosh mongodb://localhost:27017/token_service_v2 --eval "
  db.accounts.getIndexes();
  db.tokens.getIndexes();
  db.audit_logs.getIndexes();
"
```

### 删除所有索引（慎用！）

```bash
mongosh mongodb://localhost:27017/token_service_v2 --eval "
  db.accounts.dropIndexes();
  db.tokens.dropIndexes();
  db.audit_logs.dropIndexes();
"
```

### 重新初始化索引

```bash
./scripts/init/init-db.sh
```

---

## ❓ 常见问题

### Q1: 初始化脚本可以重复执行吗？

**A**: 可以。MongoDB 的 `createIndex` 是幂等操作，重复执行不会报错。

### Q2: 如果忘记初始化数据库直接启动服务怎么办？

**A**: 如果 `SKIP_INDEX_CREATION=false`（默认），服务会自动创建索引。但在多实例部署时，建议使用脚本统一初始化。

### Q3: 数据库升级时如何添加新索引？

**A**:
1. 更新 `scripts/init/init-indexes.js` 添加新索引
2. 执行 `./scripts/init/init-db.sh` 创建新索引
3. 重启服务实例（无需停机）

### Q4: 生产环境推荐的部署流程？

**A**:
```bash
# 1. 初始化数据库（首次部署或升级）
./scripts/init/init-db.sh

# 2. 构建服务（如果需要）
make build

# 3. 启动所有实例
SKIP_INDEX_CREATION=true ./bin/server &

# 4. 配置负载均衡器（Nginx/HAProxy）
```

---

## 📚 相关文档

- [API 文档](../API.md)
- [部署指南](../README.md#部署)
- [环境变量配置](.env.production.example)
- [架构设计](../ARCHITECTURE.md)

---

## 📞 技术支持

如有问题，请提交 Issue 或联系开发团队。
