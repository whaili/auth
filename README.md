# Bearer Token Service V2

> 多租户 Token 认证服务 - 基于 QiniuStub 认证

## 核心特性

- QiniuStub 认证（七牛内部用户系统）
- UID + IUID 支持（主账户 + IAM 子账户）
- 秒级过期时间精度
- 三层限流（应用/账户/Token）
- 审计日志

## 快速开始

### 本地开发

```bash
# 编译和打包
make compile    # 编译二进制
make build      # 构建镜像
make package    # 打包（镜像 + Helm Chart）

# 运行测试
make test
```

### 部署（所有部署操作已移至独立脚本）

**重要**: 部署脚本会自动从 `_cust/credentials.env` 提取 Qconf 配置并同步到目标环境。

```bash
# 本地测试环境（自带 MongoDB + Redis）
./deploy/scripts/deploy.sh local start
./deploy/scripts/deploy.sh local test      # 运行 API 测试验证
./deploy/scripts/manage.sh local status
./deploy/scripts/manage.sh local logs

# K8s 测试环境
./deploy/scripts/deploy.sh k8s-test deploy
./deploy/scripts/deploy.sh k8s-test test   # 运行 API 测试验证
./deploy/scripts/manage.sh k8s-test status

# 物理服务器生产环境（自动同步 Qconf 配置）
make package  # 先打包
./deploy/scripts/deploy.sh physical vmxs1  # 会自动从 credentials.env 提取配置
./deploy/scripts/deploy.sh physical vmxs1 test  # 运行 API 测试（通过 SSH 远程执行）
./deploy/scripts/manage.sh vmxs1 status
./deploy/scripts/manage.sh vmxs1 logs
```

查看所有部署选项：
```bash
./deploy/scripts/deploy.sh --help
./deploy/scripts/manage.sh --help
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
| `/api/v2/validateu` | POST | Bearer | 验证 Token（含用户信息） |

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
│   ├── scripts/         # 部署和管理脚本
│   │   ├── deploy.sh    # 统一部署脚本
│   │   └── manage.sh    # 统一管理脚本
│   ├── docker-compose/  # Docker Compose 配置
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
| `QCONF_ENABLED` | `false` | 是否启用 qconfapi RPC |
| `ENABLE_APP_RATE_LIMIT` | `false` | 应用层限流 |
| `ENABLE_ACCOUNT_RATE_LIMIT` | `false` | 账户层限流 |
| `ENABLE_TOKEN_RATE_LIMIT` | `false` | Token 层限流 |

完整配置说明见 [CLAUDE.md](CLAUDE.md)

## 部署环境

| 环境 | 方式 | 地址 | 部署命令 |
|------|------|------|----------|
| 本地测试 | Docker Compose | localhost | `./deploy/scripts/deploy.sh local start` |
| K8s 测试 | Helm | bearer-token-test.jfcs-k8s-qa1.qiniu.io | `./deploy/scripts/deploy.sh k8s-test deploy` |
| 物理服务器（生产） | Docker Compose | vmxs1(10.134.2.1), vmxs2(10.134.2.2) | `./deploy/scripts/deploy.sh physical vmxs1` |

详细部署信息见 [_cust/DEPLOYMENT.md](_cust/DEPLOYMENT.md)

## 文档

- [项目指南](CLAUDE.md) - 架构、API、配置详解
- [部署环境信息](_cust/DEPLOYMENT.md) - 各环境配置和访问方式
- [API 文档](docs/api/API.md) - API 接口说明
- [限流配置](docs/RATE_LIMIT.md) - 三层限流详解
- [测试说明](tests/api/TESTING.md) - 测试指南

## 常用命令

```bash
# 构建和测试
make help       # 查看所有命令
make package    # 打包镜像和 Helm Chart
make test       # 运行测试

# 部署和验证（本地测试）
./deploy/scripts/deploy.sh local start
./deploy/scripts/deploy.sh local test    # API 功能测试
./deploy/scripts/manage.sh local logs

# 部署和验证（K8s 测试）
./deploy/scripts/deploy.sh k8s-test deploy
./deploy/scripts/deploy.sh k8s-test test  # API 功能测试
./deploy/scripts/manage.sh k8s-test status

# 部署和验证（物理服务器生产环境）
./deploy/scripts/deploy.sh physical vmxs1
./deploy/scripts/deploy.sh physical vmxs1 test  # API 功能测试（SSH 远程执行）
./deploy/scripts/manage.sh vmxs1 status
./deploy/scripts/manage.sh vmxs1 logs
./deploy/scripts/manage.sh vmxs1 health
```

## 许可证

MIT License
