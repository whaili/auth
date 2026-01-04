# Bearer Token Service V2 - AI Context Guide

> 本文档用于帮助 Claude Code 快速理解项目上下文，保持代码修改与架构一致性。

## 项目定位

**云厂商级的多租户 Token 认证与授权服务**，对标 AWS IAM / 七牛云 / 阿里云的认证体系。

### 核心特性
- 🏢 **多租户隔离**：每个客户独立的 AccessKey/SecretKey，数据完全隔离
- 🔐 **双重认证**：HMAC 签名（管理）+ Bearer Token（验证）
- ⏱️ **秒级过期**：比 V1 的天级更灵活（关键升级点）
- 🎯 **Scope 权限**：细粒度权限控制（`resource:action` 格式）
- 📊 **审计合规**：完整的操作日志记录

---

## 架构设计

### 三层架构
```
┌─────────────────────────────────────────────────────────┐
│  Handler 层 (HTTP API)                                  │
│  handlers/account_handler.go                            │
│  handlers/token_handler.go                              │
│  handlers/validation_handler.go                         │
│  handlers/permission_handler.go                         │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│  Service 层 (业务逻辑)                                   │
│  service/account_service.go                             │
│  service/token_service.go                               │
│  service/validation_service.go                          │
│  service/audit_service.go                               │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│  Repository 层 (数据访问)                                │
│  repository/mongo_account_repo.go                       │
│  repository/mongo_token_repo.go                         │
│  repository/mongo_audit_repo.go                         │
└─────────────────────────────────────────────────────────┘
                         ↓
                    MongoDB
```

### 认证模块
```
auth/
├── hmac.go                    # HMAC-SHA256 签名验证
├── middleware.go              # HMAC 中间件
├── unified_middleware.go      # 统一认证（HMAC + Qstub）
└── qiniu_uid_mapper.go        # 七牛 UID 映射
```

### 权限模块
```
permission/
├── definitions.go             # 权限定义（storage、cdn、user、token）
└── scope.go                   # Scope 验证逻辑（支持通配符）
```

### 限流模块（新增）
```
ratelimit/
├── limiter.go                 # 限流器核心实现（滑动窗口算法）
├── middleware.go              # 三层限流中间件
config/
└── ratelimit.go               # 限流配置管理
```

**三层限流架构**：
1. **应用层限流**：全局请求限流（保护整个服务）
2. **账户层限流**：单个账户的请求限流（防止单租户滥用）
3. **Token层限流**：单个Token的请求限流（精细化控制）

**特性**：
- 滑动窗口算法（支持分钟/小时/天三个维度）
- 内存存储（高性能，自动清理）
- 默认全部关闭（通过环境变量启用）
- HTTP 响应头返回限流状态

---

## 核心数据模型

> 位置: `interfaces/models.go`

### Account（租户账户）
```go
Account {
    ID         string     // 账户唯一标识
    Email      string     // 邮箱地址（唯一索引）
    Company    string     // 公司名称
    AccessKey  string     // AK_xxx 格式（唯一索引）
    SecretKey  string     // bcrypt 加密存储
    Status     string     // active/suspended
    RateLimit  *RateLimit // 账户级限流配置（新增）
    QiniuUID   string     // 七牛 UID（可选关联）
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

**关键点**：
- AccessKey/SecretKey 对类似 AWS
- SecretKey 使用 bcrypt 加密存储（不可逆）
- 支持 SecretKey 轮换（重新生成）
- **RateLimit 字段**：账户级限流配置（可选）

### Token（Bearer Token）
```go
Token {
    ID            string     // tk_xxx 格式
    AccountID     string     // 关联账户（租户隔离关键！）
    Token         string     // sk-xxx 前缀
    Description   string     // Token 描述
    Scope         []string   // 权限范围（如 ["storage:read", "cdn:*"]）
    RateLimit     int        // 频率限制
    ExpiresAt     *time.Time // 过期时间（nil=永不过期，支持秒级精度）
    IsActive      bool       // 启用状态
    TotalRequests int64      // 使用次数统计
    LastUsedAt    *time.Time // 最后使用时间
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

**关键点**：
- 每次验证都强制检查 `account_id`（多租户隔离）
- `ExpiresAt` 支持秒级精度（V2 核心升级）
- Token 查询时自动隐藏中间 30 个字符（安全性）

### AuditLog（审计日志）
```go
AuditLog {
    ID          string      // 日志 ID
    AccountID   string      // 操作者账户
    Action      string      // create_token/delete_token/update_token 等
    ResourceID  string      // 操作对象 ID
    IP          string      // 请求 IP
    UserAgent   string      // User-Agent
    RequestData interface{} // 请求参数
    Result      string      // success/failure
    Timestamp   time.Time   // 操作时间
}
```

---

## API 设计

### 认证方式

#### 1. HMAC 签名认证（管理类 API）
```http
POST /api/v2/tokens
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: 2026-01-04T10:00:00Z
Content-Type: application/json

{
  "description": "Upload token",
  "scope": ["storage:write"],
  "expires_in": 3600
}
```

**签名算法**：
```
StringToSign = METHOD + "\n" + URI + "\n" + TIMESTAMP + "\n" + BODY
Signature = Base64(HMAC-SHA256(StringToSign, SecretKey))
```

**防重放攻击**：
- 检查时间戳与服务器时间差（默认 ±15 分钟）
- 可通过 `HMAC_TIMESTAMP_TOLERANCE` 环境变量调整

#### 2. Bearer Token 认证（验证 API）
```http
POST /api/v2/validate
Authorization: Bearer sk-abc123def456...
Content-Type: application/json

{
  "required_scope": "storage:write"
}
```

### 核心 API 端点

| 端点 | 方法 | 认证 | 功能 | 文件位置 |
|------|------|------|------|----------|
| `/health` | GET | ❌ | 健康检查 | cmd/server/main.go:114 |
| **账户管理** |
| `/api/v2/accounts/register` | POST | ❌ | 注册新账户 | handlers/account_handler.go |
| `/api/v2/accounts/me` | GET | HMAC | 获取当前账户信息 | handlers/account_handler.go |
| `/api/v2/accounts/regenerate-sk` | POST | HMAC | 重新生成 SecretKey | handlers/account_handler.go |
| **Token 管理** |
| `/api/v2/tokens` | POST | HMAC | 创建 Token | handlers/token_handler.go |
| `/api/v2/tokens` | GET | HMAC | 列出所有 Token | handlers/token_handler.go |
| `/api/v2/tokens/{id}` | GET | HMAC | 获取 Token 详情 | handlers/token_handler.go |
| `/api/v2/tokens/{id}/status` | PUT | HMAC | 更新 Token 状态 | handlers/token_handler.go |
| `/api/v2/tokens/{id}` | DELETE | HMAC | 删除 Token | handlers/token_handler.go |
| **Token 验证** |
| `/api/v2/validate` | POST | Bearer | 验证 Token（核心接口） | handlers/validation_handler.go |
| **权限查询** |
| `/api/v2/permissions` | GET | ❌ | 获取所有权限定义 | handlers/permission_handler.go |

---

## 权限系统

### Scope 格式
```
{resource}:{action}
```

### 预定义权限
```go
// permission/definitions.go

storage:read      // 读取存储资源
storage:write     // 写入存储资源
storage:delete    // 删除存储资源
storage:*         // 所有存储权限

cdn:refresh       // CDN 刷新
cdn:*             // 所有 CDN 权限

user:read         // 读取用户信息
user:write        // 修改用户信息
user:*            // 所有用户权限

token:create      // 创建 Token
token:read        // 读取 Token
token:delete      // 删除 Token
token:*           // 所有 Token 权限

*                 // 所有权限（超级权限）
```

### 权限验证逻辑
```go
// permission/scope.go:HasPermission()

验证规则：
1. 完全匹配：token.scope = ["storage:read"] 可以访问 "storage:read"
2. 前缀通配：token.scope = ["storage:*"] 可以访问 "storage:read", "storage:write" 等
3. 全局通配：token.scope = ["*"] 可以访问所有权限
4. 多权限：token.scope = ["storage:read", "cdn:refresh"] 可以访问两者
```

---

## 关键业务流程

### 1. 用户注册流程
```
POST /api/v2/accounts/register
  ↓
AccountHandler.Register()
  ↓
AccountService.Register()
  ├─ 检查邮箱是否已存在
  ├─ 生成 AccessKey（AK_ + 随机字符串）
  ├─ 生成 SecretKey（SK_ + 随机字符串）
  ├─ bcrypt 加密 SecretKey
  └─ 保存到 MongoDB
  ↓
返回 AccessKey/SecretKey（仅此一次明文返回！）
```

### 2. Token 创建流程
```
POST /api/v2/tokens (HMAC 认证)
  ↓
HMAC 中间件验证签名
  ├─ 解析 Authorization 头
  ├─ 验证时间戳（防重放）
  ├─ 重新计算签名
  └─ 匹配签名 → 获取 AccountID
  ↓
TokenHandler.CreateToken()
  ↓
TokenService.CreateToken()
  ├─ 生成 Token ID（tk_ + 随机字符串）
  ├─ 生成 Token 值（sk- + 随机字符串）
  ├─ 计算过期时间（当前时间 + expires_in 秒）
  ├─ 保存到 MongoDB（关联 account_id）
  └─ 记录审计日志
  ↓
返回完整 Token（明文，仅此一次！）
```

### 3. Token 验证流程（最关键）
```
POST /api/v2/validate (Bearer 认证)
  ↓
ValidationHandler.ValidateToken()
  ↓
ValidationService.ValidateToken()
  ├─ TokenRepository.GetByTokenValue()  # 查询 Token
  ├─ 检查 IsActive 状态
  ├─ 检查过期时间（ExpiresAt）
  ├─ ScopeValidator.HasPermission()     # 检查权限
  └─ 异步更新使用统计（IncrementUsage）
  ↓
返回验证结果 + Token 信息
```

---

## 配置管理

### 环境变量

| 变量 | 默认值 | 说明 | 修改影响 |
|------|--------|------|----------|
| `MONGO_URI` | `mongodb://admin:123456@localhost:27017/token_service_v2?authSource=admin` | MongoDB 连接 | 本地/云部署 |
| `MONGO_DATABASE` | `token_service_v2` | 数据库名 | 数据隔离 |
| `PORT` | `8081` | 监听端口 | 服务访问（本地开发） |
| `ACCOUNT_FETCHER_MODE` | `local` | 账户查询方式 | local/external |
| `EXTERNAL_ACCOUNT_API_URL` | - | 外部 API 地址 | 生产环境 |
| `QINIU_UID_MAPPER_MODE` | `simple` | UID 映射方式 | simple/database |
| `QINIU_UID_AUTO_CREATE` | `false` | 自动创建账户 | 集成灵活性 |
| `HMAC_TIMESTAMP_TOLERANCE` | `15m` | 时间容差 | 防重放强度 |
| `SKIP_INDEX_CREATION` | `false` | 跳过索引创建 | 负载均衡部署 |
| **限流配置** |
| `ENABLE_APP_RATE_LIMIT` | `false` | 应用层限流开关 | **默认关闭** |
| `ENABLE_ACCOUNT_RATE_LIMIT` | `false` | 账户层限流开关 | **默认关闭** |
| `ENABLE_TOKEN_RATE_LIMIT` | `false` | Token层限流开关 | **默认关闭** |
| `APP_RATE_LIMIT_PER_MINUTE` | `1000` | 应用层分钟限流 | 全局流量保护 |
| `APP_RATE_LIMIT_PER_HOUR` | `50000` | 应用层小时限流 | 全局流量保护 |
| `APP_RATE_LIMIT_PER_DAY` | `1000000` | 应用层天级限流 | 全局流量保护 |

**限流配置说明**：
- 三层限流默认全部关闭，需手动启用
- 账户层和 Token 层的限流配置存储在数据库中（Account.RateLimit / Token.RateLimit）
- 限流触发时返回 `429 Too Many Requests`，并设置 `Retry-After` 头
- 响应头包含限流状态：`X-RateLimit-Limit-*`, `X-RateLimit-Remaining-*`, `X-RateLimit-Reset-*`

### 部署模式

#### 本地开发模式
```bash
docker-compose up -d
# MongoDB + Service 完整栈
```

#### 生产部署模式
```bash
# 外部 MongoDB + 外部账户 API
export MONGO_URI="mongodb://prod-cluster:27017"
export ACCOUNT_FETCHER_MODE="external"
export EXTERNAL_ACCOUNT_API_URL="https://account.example.com"
./bearer-token-service
```

#### 启用限流模式
```bash
# 启用应用层限流（全局保护）
export ENABLE_APP_RATE_LIMIT=true
export APP_RATE_LIMIT_PER_MINUTE=1000
export APP_RATE_LIMIT_PER_HOUR=50000
export APP_RATE_LIMIT_PER_DAY=1000000

# 启用账户层限流（防止单租户滥用）
export ENABLE_ACCOUNT_RATE_LIMIT=true

# 启用 Token 层限流（精细化控制）
export ENABLE_TOKEN_RATE_LIMIT=true

./bearer-token-service
```

**限流使用示例**：

1. **创建带限流的 Token**：
```bash
POST /api/v2/tokens
{
  "description": "Limited upload token",
  "scope": ["storage:write"],
  "expires_in_seconds": 3600,
  "rate_limit": {
    "requests_per_minute": 100,
    "requests_per_hour": 5000,
    "requests_per_day": 100000
  }
}
```

2. **限流触发的响应**：
```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
X-RateLimit-Limit-Token: 100
X-RateLimit-Remaining-Token: 0
X-RateLimit-Reset-Token: 1735992345
Retry-After: 45

{
  "error": "Token rate limit exceeded",
  "code": 429,
  "timestamp": "2026-01-04T10:30:00Z"
}
```

3. **正常响应中的限流头**：
```http
HTTP/1.1 200 OK
X-RateLimit-Limit-App: 1000
X-RateLimit-Remaining-App: 856
X-RateLimit-Reset-App: 1735992400
X-RateLimit-Limit-Account: 500
X-RateLimit-Remaining-Account: 234
X-RateLimit-Reset-Account: 1735992400
X-RateLimit-Limit-Token: 100
X-RateLimit-Remaining-Token: 67
X-RateLimit-Reset-Token: 1735992400
```

---

## 数据库设计

### 集合（Collections）

#### accounts
```javascript
{
  _id: ObjectId,
  email: "user@example.com",       // 唯一索引
  company: "Example Inc",
  access_key: "AK_xxx",             // 唯一索引
  secret_key: "$2a$10$...",         // bcrypt 哈希
  status: "active",
  rate_limit: {                     // 账户级限流配置（新增）
    requests_per_minute: 500,
    requests_per_hour: 30000,
    requests_per_day: 500000
  },
  qiniu_uid: "qiniu_123",           // 索引
  created_at: ISODate,
  updated_at: ISODate
}
```

#### tokens
```javascript
{
  _id: ObjectId,
  token_id: "tk_xxx",               // 唯一索引
  account_id: "acc_xxx",            // 索引（租户隔离关键）
  token: "sk-xxx...",               // 唯一索引（验证用）
  description: "Upload token",
  scope: ["storage:read", "storage:write"],
  rate_limit: 1000,
  expires_at: ISODate,              // 索引（清理过期 Token）
  is_active: true,                  // 索引
  total_requests: 1250,
  last_used_at: ISODate,
  created_at: ISODate,
  updated_at: ISODate
}
```

#### audit_logs
```javascript
{
  _id: ObjectId,
  account_id: "acc_xxx",            // 索引
  action: "create_token",           // 索引
  resource_id: "tk_xxx",
  ip: "192.168.1.1",
  user_agent: "curl/7.64.1",
  request_data: {...},
  result: "success",
  timestamp: ISODate                // 索引（清理旧日志）
}
```

### 索引策略
```go
// repository/mongo_account_repo.go:CreateIndexes()
accounts:
  - email (unique)
  - access_key (unique)
  - qiniu_uid

// repository/mongo_token_repo.go:CreateIndexes()
tokens:
  - token_id (unique)
  - token (unique)
  - account_id + is_active (复合索引，查询优化)
  - expires_at (TTL 索引，自动清理)

// repository/mongo_audit_repo.go:CreateIndexes()
audit_logs:
  - account_id + timestamp (复合索引，查询优化)
  - action
  - timestamp (TTL 索引，保留 90 天)
```

---

## 关键文件索引

### 启动入口
- `cmd/server/main.go` - 服务启动、依赖注入、路由配置

### 核心业务
- `service/validation_service.go:ValidateToken()` - Token 验证核心逻辑
- `service/token_service.go:CreateToken()` - Token 创建逻辑
- `service/account_service.go:Register()` - 账户注册逻辑

### 认证安全
- `auth/hmac.go:VerifySignature()` - HMAC 签名验证算法
- `auth/middleware.go:Authenticate()` - HMAC 中间件
- `permission/scope.go:HasPermission()` - 权限验证引擎

### 数据模型
- `interfaces/models.go` - 所有数据结构定义
- `interfaces/repository.go` - Repository 接口定义

### 数据访问
- `repository/mongo_token_repo.go:GetByTokenValue()` - Token 查询（高频）
- `repository/mongo_account_repo.go:GetByAccessKey()` - 账户查询（高频）

---

## 开发指南

### 添加新 API 的标准流程

1. **定义数据模型**（如果需要）
   ```go
   // interfaces/models.go
   type NewResource struct {
       ID        string
       AccountID string  // 务必添加！多租户隔离
       // ...其他字段
   }
   ```

2. **添加 Repository 接口**
   ```go
   // interfaces/repository.go
   type NewResourceRepository interface {
       Create(ctx context.Context, resource *NewResource) error
       GetByID(ctx context.Context, accountID, resourceID string) (*NewResource, error)
       // 注意：所有查询都要带 accountID 参数！
   }
   ```

3. **实现 Repository**
   ```go
   // repository/mongo_newresource_repo.go
   func (r *MongoNewResourceRepository) GetByID(ctx context.Context, accountID, resourceID string) (*NewResource, error) {
       filter := bson.M{
           "resource_id": resourceID,
           "account_id": accountID,  // 强制租户隔离！
       }
       // ...
   }
   ```

4. **实现 Service**
   ```go
   // service/newresource_service.go
   type NewResourceService struct {
       repo repository.NewResourceRepository
   }

   func (s *NewResourceService) CreateResource(ctx context.Context, accountID string, req *CreateResourceRequest) error {
       // 业务逻辑
       resource := &interfaces.NewResource{
           AccountID: accountID,  // 从认证中间件获取
           // ...
       }
       return s.repo.Create(ctx, resource)
   }
   ```

5. **实现 Handler**
   ```go
   // handlers/newresource_handler.go
   func (h *NewResourceHandler) CreateResource(w http.ResponseWriter, r *http.Request) {
       accountID := r.Context().Value("account_id").(string)  // 从中间件获取
       // ...
       err := h.service.CreateResource(r.Context(), accountID, req)
       // ...
   }
   ```

6. **注册路由**
   ```go
   // cmd/server/main.go
   authRouter.HandleFunc("/api/v2/resources", resourceHandler.CreateResource).Methods("POST")
   ```

7. **添加审计日志**（如果是重要操作）
   ```go
   // service/newresource_service.go
   auditLog := &interfaces.AuditLog{
       AccountID:  accountID,
       Action:     "create_resource",
       ResourceID: resource.ID,
       Result:     "success",
   }
   s.auditService.LogAction(ctx, auditLog)
   ```

### 关键设计原则

#### ✅ DO
- 所有查询都带 `account_id` 参数（多租户隔离）
- 使用 bcrypt 存储密码和密钥
- 使用 `crypto/rand` 生成随机数
- 重要操作记录审计日志
- 使用依赖注入（便于测试）
- 错误信息避免泄露敏感信息

#### ❌ DON'T
- 不要跨租户查询数据
- 不要在日志中打印敏感信息（Token、SecretKey）
- 不要使用 `math/rand`（不安全）
- 不要在 Token 响应中返回完整 Token（除创建时）
- 不要绕过认证中间件

### 安全检查清单

- [ ] 所有管理 API 使用 HMAC 认证
- [ ] 所有查询强制检查 `account_id`
- [ ] Token 过期时间正确处理（`nil` vs 具体时间）
- [ ] 敏感数据加密存储（bcrypt）
- [ ] 防重放攻击（时间戳验证）
- [ ] 权限检查（Scope 验证）
- [ ] 审计日志记录（重要操作）
- [ ] 错误信息不泄露内部细节

---

## 常见问题排查

### Token 验证失败
1. 检查 Token 是否过期（`expires_at`）
2. 检查 Token 是否激活（`is_active`）
3. 检查 Token 是否存在（数据库查询）
4. 检查 Scope 权限是否匹配

### HMAC 签名失败
1. 检查时间戳是否在容差范围内（默认 ±15 分钟）
2. 检查签名算法是否正确（METHOD + URI + TIMESTAMP + BODY）
3. 检查 SecretKey 是否正确
4. 检查请求体是否被修改

### 多租户隔离问题
1. 确认所有查询都带 `account_id` 过滤
2. 检查 Repository 层的过滤条件
3. 查看审计日志确认操作者

### MongoDB 连接问题
1. 检查 `MONGO_URI` 环境变量
2. 检查网络连通性
3. 检查 MongoDB 用户权限
4. 查看 MongoDB 日志

---

## V1 vs V2 对比（重要升级点）

| 维度 | V1 | V2 | 影响 |
|------|----|----|------|
| **租户模式** | 单租户 | 多租户 | 架构重构 |
| **认证方式** | Basic Auth | HMAC 签名 | 安全性提升 |
| **过期精度** | 天级 | 秒级 | 灵活性提升 |
| **权限控制** | 无 | Scope 权限 | 功能增强 |
| **防重放** | 无 | 时间戳验证 | 安全性提升 |
| **审计日志** | 简单 | 完整 | 合规性提升 |

---

## 技术债务 & 改进建议

### 当前已知限制
1. Token 使用统计是异步更新（可能延迟）
2. 过期 Token 需要定期清理（未实现自动任务）
3. 频率限制功能未完全实现
4. 缺少 Token 使用详细日志（只有总次数）

### 未来改进方向
1. 添加 Redis 缓存（减少 MongoDB 查询）
2. 实现真正的频率限制（基于 Redis）
3. Token 使用详细日志（每次验证记录）
4. WebSocket 实时通知（Token 状态变更）
5. 监控指标（Prometheus）
6. 性能测试和基准测试

---

## 快速命令

```bash
# 本地开发
docker-compose up -d              # 启动服务
docker-compose logs -f            # 查看日志
docker-compose down               # 停止服务

# 编译
make build                        # 编译二进制
make package                      # 打包部署文件

# 测试
curl http://localhost:8080/health # 健康检查
# 更多测试命令见 README.md

# MongoDB 操作
docker exec -it bearer-token-service-mongo-1 mongosh
> use token_service_v2
> db.accounts.find().pretty()
> db.tokens.find().pretty()
> db.audit_logs.find().sort({timestamp: -1}).limit(10)
```

---

## 联系信息

- 项目文档：`docs/` 目录
- API 文档：`docs/api/API.md`
- 配置说明：`docs/CONFIG.md`
- 用户手册：`docs/bearer_token_service_user_guide.md`

---

**最后更新**: 2026-01-04
**版本**: V2
**维护者**: Claude Code Generated
