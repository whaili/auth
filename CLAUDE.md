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
Handler (handlers/) → Service (service/) → Repository (repository/) → MongoDB / MySQL
```

| 层 | 文件 |
|---|------|
| Handler | `handlers/token_handler.go`, `handlers/validation_handler.go` |
| Service | `service/token_service.go`, `service/validation_service.go`, `service/audit_service.go` |
| Repository | `repository/mongo_token_repo.go`, `repository/mongo_audit_repo.go`, `repository/mysql_userinfo_repo.go` |
| Auth | `auth/qstub_middleware.go`, `auth/qiniu_uid_mapper.go`, `auth/context.go` |
| RateLimit | `ratelimit/limiter.go`, `ratelimit/middleware.go`, `config/ratelimit.go` |
| Config | `config/ratelimit.go`, `config/mysql.go` |

## 数据模型

详见 `interfaces/models.go`

**Token**: ID(`tk_xxx`), AccountID(`qiniu_{uid}`), Token(`sk-xxx`), IUID, ExpiresAt, IsActive

**AuditLog**: AccountID, Action, ResourceID, IP, Result, Timestamp

**UserInfo**: UID, Email, Username, Utype (位掩码), Activated, DisabledType, ParentUID, CreatedAt, UpdatedAt, LastLoginAt

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
| `/api/v2/validateu` | POST | Bearer | 验证 Token（扩展用户信息） |

## 环境变量

### MongoDB 配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `MONGO_URI` | `mongodb://admin:123456@localhost:27017` | MongoDB 连接 |
| `MONGO_DATABASE` | `token_service_v2` | 数据库名 |

### Qconf 配置（用户信息查询）

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `QCONF_ENABLED` | `false` | 是否启用 qconfapi RPC |
| `QCONF_ACCESS_KEY` | - | Qconf AccessKey |
| `QCONF_SECRET_KEY` | - | Qconf SecretKey |
| `QCONF_MASTER_HOSTS` | - | Qconf Master 节点（逗号分隔） |
| `QCONF_LC_EXPIRES_MS` | `300000` | 本地缓存过期时间（毫秒） |
| `QCONF_LC_DURATION_MS` | `60000` | 本地缓存刷新间隔（毫秒） |

### 服务配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | `8080` | 监听端口 |
| `QINIU_UID_MAPPER_MODE` | `simple` | UID 映射（simple/database） |

### 限流配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `ENABLE_APP_RATE_LIMIT` | `false` | 应用层限流 |
| `ENABLE_ACCOUNT_RATE_LIMIT` | `false` | 账户层限流 |
| `ENABLE_TOKEN_RATE_LIMIT` | `false` | Token层限流 |

**注意**:
- Qconf 配置用于 `/api/v2/validateu` 端点获取用户详细信息
- 如果 Qconf 连接失败，接口会优雅降级，返回基本 Token 信息（`user_info` 为 `null`）
- 测试环境 Qconf: `http://kodo-dev.confg.jfcs-k8s-qa2.qiniu.io`
- 生产环境 Qconf: `http://10.34.35.42:8510,http://10.34.35.43:8510`

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
- Token 验证（扩展用户信息）: `service/validation_service.go:ValidateTokenWithUserInfo()`
- Token 创建: `service/token_service.go:CreateToken()`
- 认证中间件: `auth/qstub_middleware.go:Authenticate()`
- 数据模型: `interfaces/models.go`, `interfaces/repository.go`
- Qconf 配置: `config/qconf.go`
- RPC 用户信息查询: `repository/rpc_userinfo_repo.go`

## 功能说明

### /api/v2/validateu 端点

新增的 Token 验证端点，返回扩展的用户信息。

**特性**:
- 验证 Bearer Token 有效性（与 `/api/v2/validate` 相同）
- 从 Qconfapi RPC 查询用户详细信息（UID、Email、Username、Utype 等）
- **优雅降级**: Qconf 查询失败时仍返回基本 Token 信息，`user_info` 字段为 `null`

**请求示例**:
```http
POST /api/v2/validateu
Authorization: Bearer sk-abc123...
```

**响应示例（成功）**:
```json
{
  "valid": true,
  "message": "Token is valid",
  "token_info": {
    "token_id": "tk_9z8y7x6w5v4u",
    "uid": "1369077332",
    "iuid": "8901234",
    "is_active": true,
    "expires_at": "2026-03-25T10:00:00Z",
    "user_info": {
      "uid": 1369077332,
      "email": "user@example.com",
      "username": "测试用户",
      "utype": 268435460,
      "activated": true,
      "disabled_type": 0,
      "disabled_reason": "",
      "parent_uid": 0,
      "created_at": 1609459200,
      "updated_at": 1704067200,
      "last_login_at": 1735689600
    }
  }
}
```

**响应示例（优雅降级）**:
```json
{
  "valid": true,
  "message": "Token is valid",
  "token_info": {
    "token_id": "tk_9z8y7x6w5v4u",
    "uid": "1369077332",
    "iuid": "8901234",
    "is_active": true,
    "expires_at": "2026-03-25T10:00:00Z",
    "user_info": null
  }
}
```

**Utype 字段说明**:
- `utype` 是位掩码，表示用户类型和权限
- 使用位运算判断，不能用 `==` 比较
- 详见 `/root/src/aitoken-aigc-agent/chatnio/_cust/GetBOUserInfo_FIELDS.md`

