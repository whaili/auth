# Bearer Token Service V2

多租户 Token 认证服务，账户管理由外部系统（七牛）提供。

## 核心特性

- QiniuStub 认证（七牛内部用户系统）
- UID + IUID 支持（主账户 + IAM 子账户）
- Bearer Token 管理（CRUD）
- 秒级过期时间
- 审计日志
- 三层限流（应用/账户/Token）

## 架构

```
Handler (handlers/) → Service (service/) → Repository (repository/) → MongoDB
```

| 层 | 文件 |
|---|------|
| Handler | `handlers/token_handler.go`, `handlers/validation_handler.go` |
| Service | `service/token_service.go`, `service/validation_service.go`, `service/audit_service.go` |
| Repository | `repository/mongo_token_repo.go`, `repository/mongo_audit_repo.go` |
| Auth | `auth/qstub_middleware.go`, `auth/qiniu_uid_mapper.go`, `auth/context.go` |
| RateLimit | `ratelimit/limiter.go`, `ratelimit/middleware.go`, `config/ratelimit.go` |

## 数据模型

详见 `interfaces/models.go`

**Token**: ID(`tk_xxx`), AccountID(`qiniu_{uid}`), Token(`sk-xxx`), IUID, ExpiresAt, IsActive

**AuditLog**: AccountID, Action, ResourceID, IP, Result, Timestamp

## API

### 认证方式

```http
# QiniuStub（Token 管理 API）
Authorization: QiniuStub uid=1369077332&ut=1
Authorization: QiniuStub uid=1369077332&ut=1&iuid=8901234  # IAM 子账户

# Bearer（Token 验证 API）
Authorization: Bearer sk-abc123...
```

### 端点

| 端点 | 方法 | 认证 | 功能 |
|------|------|------|------|
| `/health` | GET | - | 健康检查 |
| `/api/v2/tokens` | POST | QiniuStub | 创建 Token |
| `/api/v2/tokens` | GET | QiniuStub | 列出 Token |
| `/api/v2/tokens/{id}` | GET | QiniuStub | 获取详情 |
| `/api/v2/tokens/{id}/status` | PUT | QiniuStub | 更新状态 |
| `/api/v2/tokens/{id}` | DELETE | QiniuStub | 删除 |
| `/api/v2/tokens/{id}/stats` | GET | QiniuStub | 统计信息 |
| `/api/v2/validate` | POST | Bearer | 验证 Token |

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `MONGO_URI` | `mongodb://admin:123456@localhost:27017` | MongoDB 连接 |
| `MONGO_DATABASE` | `token_service_v2` | 数据库名 |
| `PORT` | `8080` | 监听端口 |
| `QINIU_UID_MAPPER_MODE` | `simple` | UID 映射（simple/database） |
| `ENABLE_APP_RATE_LIMIT` | `false` | 应用层限流 |
| `ENABLE_ACCOUNT_RATE_LIMIT` | `false` | 账户层限流 |
| `ENABLE_TOKEN_RATE_LIMIT` | `false` | Token层限流 |

## 开发命令

```bash
make test           # 运行测试
make build          # 编译
make package        # 打包
bash tests/api/start_local.sh  # 本地启动
```

## 关键入口

- 启动: `cmd/server/main.go`
- Token 验证: `service/validation_service.go:ValidateToken()`
- Token 创建: `service/token_service.go:CreateToken()`
- 认证中间件: `auth/qstub_middleware.go:Authenticate()`
- 数据模型: `interfaces/models.go`, `interfaces/repository.go`
