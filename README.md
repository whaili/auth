# Bearer Token Service V2

> 多租户 Token 认证服务 - 基于 QiniuStub 认证

## 文档导航

- [部署和配置指南](docs/DEPLOYMENT.md) - 完整的部署和配置文档
- [限流配置](docs/RATE_LIMIT.md) - 三层限流详解
- [API 文档](docs/api/API.md) - API 接口说明
- [测试说明](tests/api/TESTING.md) - 测试指南
- [项目指南](CLAUDE.md) - 项目结构和开发指南

## 核心特性

- QiniuStub 认证（七牛内部用户系统）
- UID + IUID 支持（主账户 + IAM 子账户）
- 秒级过期时间精度
- 三层限流（应用/账户/Token）
- 审计日志

## 快速开始

### 本地开发

```bash
# 启动测试环境（含 MongoDB + Redis + Nginx）
make up-test

# 运行测试（通过 Nginx 80 端口）
BASE_URL=http://localhost make test

# 查看日志
make logs

# 停止服务
make down
```

### 端口说明

| 服务 | 端口 | 说明 |
|------|------|------|
| Nginx | 80/443 | 反向代理入口（推荐） |
| App | 8080 | 服务直接端口（容器内部） |

测试脚本默认使用 8081 端口，通过 Docker Compose 部署时需指定：
```bash
BASE_URL=http://localhost make test      # 通过 Nginx
BASE_URL=http://localhost:8080 make test # 直接访问（需暴露端口）
```

### 编译构建

```bash
make build      # 编译二进制
make package    # 打包（镜像 + Helm Chart）
```

## 认证方式

### QiniuStub（Token 管理 API）

```bash
# 主账户
curl -X POST "http://localhost:8080/api/v2/tokens" \
  -H "Authorization: QiniuStub uid=1369077332&ut=1" \
  -H "Content-Type: application/json" \
  -d '{"description": "My token", "expires_in_seconds": 3600}'

# IAM 子账户
curl -X POST "http://localhost:8080/api/v2/tokens" \
  -H "Authorization: QiniuStub uid=1369077332&ut=1&iuid=8901234" \
  -H "Content-Type: application/json" \
  -d '{"description": "IAM token", "expires_in_seconds": 3600}'
```

### Bearer Token（验证 API）

```bash
curl -X POST "http://localhost:8080/api/v2/validate" \
  -H "Authorization: Bearer sk-abc123..."
```

## API 端点

| 端点 | 方法 | 认证 | 说明 |
|------|------|------|------|
| `/health` | GET | - | 健康检查 |
| `/metrics` | GET | - | Prometheus 指标 |
| `/api/v2/tokens` | POST | QiniuStub | 创建 Token |
| `/api/v2/tokens` | GET | QiniuStub | 列出 Tokens |
| `/api/v2/tokens/{id}` | GET | QiniuStub | 获取详情 |
| `/api/v2/tokens/{id}/status` | PUT | QiniuStub | 更新状态 |
| `/api/v2/tokens/{id}` | DELETE | QiniuStub | 删除 Token |
| `/api/v2/validate` | POST | Bearer | 验证 Token |

## 项目结构

```
├── cmd/server/          # 服务入口
├── auth/                # 认证模块
├── service/             # 业务逻辑层
├── repository/          # 数据访问层
├── handlers/            # HTTP 处理层
├── interfaces/          # 接口和模型定义
├── ratelimit/           # 限流模块
├── observability/       # 可观测性（日志、指标）
├── config/              # 配置管理
├── deploy/              # 部署配置
│   ├── docker-compose/  # Docker Compose 部署
│   ├── helm/            # Kubernetes Helm Chart
│   └── init/            # 数据库初始化脚本
├── tests/               # 测试
│   ├── api/             # API 功能测试
│   └── load/            # 负载测试 (k6)
└── docs/                # 文档
```

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | `8080` | 服务端口 |
| `MONGO_URI` | - | MongoDB 连接字符串 |
| `MONGO_DATABASE` | `token_service_v2` | 数据库名 |
| `REDIS_ADDR` | `redis:6379` | Redis 地址 |
| `ENABLE_APP_RATE_LIMIT` | `false` | 应用层限流 |
| `ENABLE_ACCOUNT_RATE_LIMIT` | `false` | 账户层限流 |
| `ENABLE_TOKEN_RATE_LIMIT` | `false` | Token 层限流 |

## 部署

详见 [部署指南](docs/deployment.md)

| 场景 | 方式 | 命令 |
|------|------|------|
| 本地测试 | Docker Compose | `make up-test` |
| 生产（物理机） | Docker Compose | `make up-prod` |
| 测试（K8s） | Helm | `make helm-deploy-test` |
| 生产（K8s） | Helm | `make helm-deploy-prod` |

## 许可证

MIT License

## 测试数据迁移

本地开发和测试环境无法直接访问生产 MySQL，需要使用测试 MySQL (10.210.31.54)。

### 快速开始

```bash
# 1. 从生产环境导出并混淆数据
./deploy/scripts/migrate_test_data_v2.sh bearer_token_test_<用户名> 500

# 2. 更新环境变量（使用脚本输出的配置）
export MYSQL_HOST="10.210.31.54"
export MYSQL_DATABASE="bearer_token_test_<your_db>"

# 3. 启动服务
go run cmd/server/main.go
```

### 文档

- **快速开始**: [deploy/scripts/USAGE.md](deploy/scripts/USAGE.md)
- **完整文档**: [deploy/scripts/README.md](deploy/scripts/README.md)
- **技术方案**: [deploy/MYSQL_CONFIG.md](deploy/MYSQL_CONFIG.md)

### 特性

- ✅ 自动从生产环境导出数据
- ✅ 自动混淆敏感信息（邮箱、用户名、密码、IP、手机号）
- ✅ 支持自定义数据库名和导出数量
- ✅ 完整的错误检查和数据验证

详细说明请查看 [deploy/scripts/](deploy/scripts/) 目录。

