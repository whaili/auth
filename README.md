# Bearer Token Service V2

> 多租户 Token 认证服务 - 基于 QiniuStub 认证、秒级过期时间精度

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![MongoDB](https://img.shields.io/badge/MongoDB-4.0+-green.svg)](https://www.mongodb.com)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## 📖 文档导航

- **API 文档**: [docs/api/API.md](docs/api/API.md) - 完整 API 参考
- **架构说明**: [CLAUDE.md](CLAUDE.md) - 系统架构和开发指南
- **配置说明**: [docs/CONFIG.md](docs/CONFIG.md) - 环境变量配置
- **测试说明**: [tests/TESTING.md](tests/TESTING.md) - 测试指南

---

## ✨ 核心特性

- **QiniuStub 认证**: 使用七牛内部用户系统认证（UID + IUID）
- **多租户隔离**: 完全的数据隔离，支持 SaaS 化部署
- **秒级过期时间**: ⭐ 支持秒级精度的 Token 过期时间设置
- **IAM 子账户支持**: 支持主账户（UID）和 IAM 子账户（UID + IUID）
- **审计日志**: 完整的操作审计记录
- **限流功能**: 三层限流（应用/账户/Token）
- **生产就绪**: Docker 部署、日志管理、性能优化

---

## 🚀 快速开始

### 1. 使用 Docker Compose（推荐）

```bash
# 启动服务（包含 MongoDB）
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### 2. 本地开发

```bash
# 启动 MongoDB
docker run -d -p 27017:27017 --name mongodb \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=123456 \
  mongo:latest

# 启动服务
bash tests/start_local.sh

# 或使用 make
make run
```

### 3. 编译和测试

```bash
# 编译
make build

# 运行测试
make test

# 清理
make clean
```

---

## 🔐 认证方式

### QiniuStub 认证（Token 管理 API）

**主账户**：
```bash
curl -X POST "http://localhost:8081/api/v2/tokens" \
  -H "Authorization: QiniuStub uid=1369077332&ut=1" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "My upload token",
    "expires_in_seconds": 3600
  }'
```

**IAM 子账户**：
```bash
curl -X POST "http://localhost:8081/api/v2/tokens" \
  -H "Authorization: QiniuStub uid=1369077332&ut=1&iuid=8901234" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "IAM user token",
    "expires_in_seconds": 3600
  }'
```

### Bearer Token 认证（Token 验证 API）

```bash
curl -X POST "http://localhost:8081/api/v2/validate" \
  -H "Authorization: Bearer sk-abc123def456..." \
  -H "Content-Type: application/json"
```

---

## 📚 API 端点

| 端点 | 方法 | 认证 | 说明 |
|------|------|------|------|
| `/health` | GET | ❌ | 健康检查 |
| `/api/v2/tokens` | POST | QiniuStub | 创建 Token |
| `/api/v2/tokens` | GET | QiniuStub | 列出 Tokens |
| `/api/v2/tokens/{id}` | GET | QiniuStub | 获取 Token 详情 |
| `/api/v2/tokens/{id}/status` | PUT | QiniuStub | 更新 Token 状态 |
| `/api/v2/tokens/{id}` | DELETE | QiniuStub | 删除 Token |
| `/api/v2/tokens/{id}/stats` | GET | QiniuStub | 获取 Token 统计 |
| `/api/v2/validate` | POST | Bearer | 验证 Token |

完整 API 文档: [docs/api/API.md](docs/api/API.md)

---

## 🏗️ 项目结构

```
bearer-token-service.v2/
├── cmd/server/              # 服务入口
│   └── main.go
├── auth/                    # 认证模块
│   ├── qstub_middleware.go  # QiniuStub 认证
│   ├── qiniu_uid_mapper.go  # UID 映射
│   └── context.go           # Context 辅助
├── ratelimit/               # 限流模块
│   ├── limiter.go           # 限流器
│   └── middleware.go        # 限流中间件
├── service/                 # 业务逻辑层
│   ├── token_service.go
│   ├── validation_service.go
│   └── audit_service.go
├── repository/              # 数据访问层
│   ├── mongo_token_repo.go
│   └── mongo_audit_repo.go
├── handlers/                # HTTP 处理层
│   ├── token_handler.go
│   └── validation_handler.go
├── interfaces/              # 接口定义
│   ├── models.go
│   └── repository.go
├── config/                  # 配置管理
├── docs/                    # 文档
├── tests/                   # 测试
└── docker-compose.yml       # Docker 编排
```

---

## ⚙️ 配置

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | `8080` | 服务端口 |
| `MONGO_URI` | `mongodb://admin:123456@localhost:27017` | MongoDB 连接 |
| `MONGO_DATABASE` | `token_service_v2` | 数据库名 |
| `QINIU_UID_MAPPER_MODE` | `simple` | UID 映射模式 |
| `ENABLE_APP_RATE_LIMIT` | `false` | 应用层限流 |
| `ENABLE_ACCOUNT_RATE_LIMIT` | `false` | 账户层限流 |
| `ENABLE_TOKEN_RATE_LIMIT` | `false` | Token 层限流 |

完整配置: [docs/CONFIG.md](docs/CONFIG.md)

---

## 🧪 测试

```bash
# 运行完整测试
make test

# 只运行服务（不测试）
make run

# 停止测试服务
make test-stop
```

测试覆盖：
- ✅ Token 创建（主账户 + IAM 子账户）
- ✅ Token 列表、详情、更新、删除
- ✅ Token 验证（包含 IUID）

---

## 📊 数据库

### MongoDB 集合

- `tokens` - Token 数据
- `accounts` - 账户映射（UID 映射）
- `audit_logs` - 审计日志

### 索引策略

- `tokens.token` (unique) - Token 值唯一索引
- `tokens.account_id + is_active` - 账户查询优化
- `tokens.expires_at` - 过期清理
- `audit_logs.account_id + timestamp` - 审计日志查询

---

## 🚀 部署

### Docker 部署

```bash
# 构建镜像
docker build -t bearer-token-service:v2 .

# 运行
docker run -d -p 8080:8080 \
  -e MONGO_URI="mongodb://mongo:27017" \
  bearer-token-service:v2
```

### Makefile 命令

```bash
make build      # 编译二进制
make package    # 打包部署文件
make clean      # 清理
make run        # 运行服务
make test       # 运行测试
```

---

## 📝 重要变更

### V2 简化版（当前版本）

**移除的功能**：
- ❌ 账户注册和管理（由外部系统负责）
- ❌ HMAC 签名认证（只使用 QiniuStub）
- ❌ Scope 权限控制（简化设计）

**保留的功能**：
- ✅ Token 完整生命周期管理
- ✅ Bearer Token 验证
- ✅ 限流功能
- ✅ 审计日志

**认证方式**：
- Token 管理 API: QiniuStub（UID + IUID）
- Token 验证 API: Bearer Token

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---

## 📄 许可证

MIT License

---

**最后更新**: 2026-01-12
**版本**: V2（简化版）
