# 脚本说明

## 📁 目录结构

```
scripts/
├── init/          # 数据库初始化脚本
│   ├── init-db.sh
│   └── init-indexes.js
├── deploy/        # 部署相关脚本
│   ├── deploy.sh
│   ├── setup-ssl.sh
│   ├── run-without-docker.sh
│   ├── install-docker.sh
│   ├── install-docker-compose.sh
│   └── systemd/
├── test/          # 测试脚本
│   ├── test.sh
│   ├── test-dual-auth.sh
│   ├── test-hmac.sh
│   ├── test-hmac.py
│   └── test.py
└── README.md      # 本文件
```

---

## 🔧 init/ - 数据库初始化

### init-db.sh
**数据库初始化脚本**

**用途**：在负载均衡多实例部署前，统一初始化 MongoDB 索引

**使用方法**：
```bash
# 使用默认配置
./scripts/init/init-db.sh

# 自定义配置
MONGO_URI="mongodb://user:pass@host:27017" \
MONGO_DATABASE="custom_db" \
./scripts/init/init-db.sh
```

**前置条件**：
- MongoDB 服务已启动
- 已安装 `mongosh` 或 `mongo` 命令

**输出**：创建所有必需的数据库索引

---

### init-indexes.js
**MongoDB 索引定义脚本**

**用途**：定义所有集合的索引结构

**使用方法**：
```bash
# 直接执行（不推荐）
mongosh mongodb://localhost:27017/token_service_v2 scripts/init/init-indexes.js

# 通过 init-db.sh 执行（推荐）
./scripts/init/init-db.sh
```

**索引列表**：
- **accounts** 集合：5个索引（含2个唯一索引）
- **tokens** 集合：5个索引（含1个唯一索引）
- **audit_logs** 集合：5个索引（含1个TTL索引）

---

## 🚀 deploy/ - 部署脚本

### deploy.sh
Docker Compose 一键部署脚本

### setup-ssl.sh
配置 SSL/HTTPS 证书（Let's Encrypt）

### run-without-docker.sh
不使用 Docker 的原生部署

### install-docker.sh / install-docker-compose.sh
Docker 环境安装脚本

### systemd/
Systemd 服务单元文件

---

## 🧪 test/ - 测试脚本

### test.sh
完整的 API 功能测试

### test-dual-auth.sh
双认证模式测试（HMAC + Qstub）

### test-hmac.sh / test-hmac.py
HMAC 签名认证测试

---

## 📚 详细文档

- [数据库初始化指南](../docs/DATABASE_INIT.md)
- [生产部署指南](../PRODUCTION_DEPLOYMENT.md)

## 🔍 脚本依赖关系

```
init/init-db.sh (Shell 脚本)
    │
    ├─> 检查 mongosh/mongo 命令
    ├─> 测试 MongoDB 连接
    └─> 执行 init/init-indexes.js (JavaScript)
            │
            └─> 创建所有索引
```

## ⚠️ 注意事项

1. **首次部署必须执行**：确保索引在服务启动前创建完成
2. **幂等操作**：可以安全地重复执行，不会覆盖数据
3. **生产环境**：建议在部署流程中自动执行此脚本

## 🚀 快速开始

```bash
# 1. 初始化数据库
./scripts/init/init-db.sh

# 2. 启动服务（跳过索引创建）
SKIP_INDEX_CREATION=true ./bin/server
```
