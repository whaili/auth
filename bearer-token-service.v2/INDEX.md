# Bearer Token Service V2 - 实现索引

> 快速导航 - 查找接口定义和实现文件

---

## 📂 目录结构概览

```
v2/
├── ARCHITECTURE.md              # 架构设计文档
├── API.md                       # API 使用文档
├── INDEX.md                     # 本文件 - 实现索引
│
├── interfaces/                  # 接口定义（纯接口，无实现）
│   ├── models.go                # 数据模型定义
│   ├── repository.go            # Repository/Service 接口
│   └── api.go                   # API Handler 接口
│
├── auth/                        # 认证模块实现
│   ├── hmac.go                  # HMAC 签名实现
│   └── middleware.go            # 认证中间件实现
│
└── permission/                  # 权限模块实现
    └── scope.go                 # Scope 权限验证实现
```

---

## 🔍 接口定义文档

### 数据模型 (interfaces/models.go)

定义了所有数据结构和请求/响应模型：

| 模型 | 说明 |
|------|------|
| `Account` | 租户账户模型 |
| `Token` | Bearer Token 模型（带 Scope） |
| `AuditLog` | 审计日志模型 |
| `RateLimit` | API 频率限制配置 |
| `AccountRegisterRequest` | 账户注册请求 |
| `TokenCreateRequest` | Token 创建请求 |
| `TokenValidateRequest` | Token 验证请求 |

**查看**: `/root/src/auth/bearer-token-service.v1/v2/interfaces/models.go`

---

### Repository 接口 (interfaces/repository.go)

定义了数据访问层和服务层接口：

#### Repository 接口

| 接口 | 说明 | 方法数 |
|------|------|--------|
| `AccountRepository` | 账户数据访问 | 8 |
| `TokenRepository` | Token 数据访问 | 10 |
| `AuditLogRepository` | 审计日志数据访问 | 4 |

#### Service 接口

| 接口 | 说明 | 方法数 |
|------|------|--------|
| `AccountService` | 账户管理服务 | 5 |
| `TokenService` | Token 管理服务 | 6 |
| `ValidationService` | Token 验证服务 | 3 |
| `PermissionService` | 权限验证服务 | 3 |
| `AuditService` | 审计日志服务 | 3 |

#### 认证接口

| 接口 | 说明 |
|------|------|
| `HMACAuthenticator` | HMAC 签名认证 |
| `SignatureBuilder` | 签名构建器 |

**查看**: `/root/src/auth/bearer-token-service.v1/v2/interfaces/repository.go`

---

### API Handler 接口 (interfaces/api.go)

定义了 HTTP API 处理器接口：

| 接口 | 说明 | 端点数 |
|------|------|--------|
| `AccountHandler` | 账户管理 API | 3 |
| `TokenHandler` | Token 管理 API | 6 |
| `ValidationHandler` | Token 验证 API | 2 |
| `AuditHandler` | 审计日志 API | 1 |

#### Middleware 接口

| 接口 | 说明 |
|------|------|
| `HMACAuthMiddleware` | HMAC 认证中间件 |
| `CORSMiddleware` | CORS 中间件 |
| `RateLimitMiddleware` | 限流中间件 |
| `LoggingMiddleware` | 日志中间件 |

**查看**: `/root/src/auth/bearer-token-service.v1/v2/interfaces/api.go`

---

## ✅ 已实现模块

### 1. HMAC 签名认证 (auth/hmac.go)

**实现内容**:
- ✅ `SignatureBuilder` - 签名构建器
- ✅ `HMACAuthenticator` - HMAC 认证器
- ✅ `ClientSignatureGenerator` - 客户端签名生成器（用于测试）

**核心方法**:
```go
// 生成签名
GenerateSignature(secretKey, stringToSign) (signature, error)

// 验证签名
VerifySignature(secretKey, receivedSignature, stringToSign) (bool, error)

// 验证时间戳（防重放）
ValidateTimestamp(timestamp) error

// 构建待签名字符串
BuildStringToSign(method, uri, timestamp, body) string

// 解析 Authorization Header
ParseAuthHeader(authHeader) (accessKey, signature, error)
```

**查看**: `/root/src/auth/bearer-token-service.v1/v2/auth/hmac.go`

---

### 2. 认证中间件 (auth/middleware.go)

**实现内容**:
- ✅ `HMACMiddleware` - HMAC 认证中间件（适配 net/http）
- ✅ `ExtractAccountFromContext` - 从 Context 提取账户信息
- ✅ `ExtractAccountIDFromContext` - 从 Context 提取账户 ID

**认证流程**:
```
请求 → 提取 Authorization Header
     → 验证时间戳
     → 查询 Account
     → 验证签名
     → 注入 Account 到 Context
     → 调用下一个 Handler
```

**使用方式**:
```go
middleware := NewHMACMiddleware(accountFetcher, 15*time.Minute)
http.HandleFunc("/api/v2/tokens", middleware.Authenticate(handler))
```

**查看**: `/root/src/auth/bearer-token-service.v1/v2/auth/middleware.go`

---

### 3. Scope 权限验证 (permission/scope.go)

**实现内容**:
- ✅ `ScopeValidator` - Scope 权限验证器

**核心方法**:
```go
// 检查权限
HasPermission(tokenScopes, requiredScope) bool

// 验证 Scope 格式
ValidateScopes(scopes) error

// 展开通配符（用于显示）
ExpandWildcardScopes(scopes) []string

// 批量检查权限
MatchScopes(tokenScopes, requiredScopes) bool

// 获取缺失的权限
GetMissingScopes(tokenScopes, requiredScopes) []string
```

**权限匹配规则**:
1. **精确匹配**: `"storage:read"` == `"storage:read"`
2. **全局通配**: `"*"` 匹配所有权限
3. **前缀通配**: `"storage:*"` 匹配 `"storage:read"`, `"storage:write"` 等

**示例**:
```go
validator := NewScopeValidator()

// Token 拥有 ["storage:*", "cdn:refresh"]
validator.HasPermission(["storage:*"], "storage:read")  // true
validator.HasPermission(["storage:*"], "storage:write") // true
validator.HasPermission(["cdn:refresh"], "cdn:refresh") // true
validator.HasPermission(["cdn:refresh"], "cdn:purge")   // false
```

**查看**: `/root/src/auth/bearer-token-service.v1/v2/permission/scope.go`

---

## 📋 待实现模块

### Repository 实现

需要为以下 Repository 接口创建 MongoDB 实现：

- [ ] `MongoAccountRepository` - 实现 `AccountRepository` 接口
- [ ] `MongoTokenRepository` - 实现 `TokenRepository` 接口（带租户隔离）
- [ ] `MongoAuditLogRepository` - 实现 `AuditLogRepository` 接口

**建议目录**: `internal/repository/mongo/`

---

### Service 实现

需要实现以下业务逻辑服务：

- [ ] `AccountServiceImpl` - 实现 `AccountService` 接口
- [ ] `TokenServiceImpl` - 实现 `TokenService` 接口
- [ ] `ValidationServiceImpl` - 实现 `ValidationService` 接口
- [ ] `AuditServiceImpl` - 实现 `AuditService` 接口

**建议目录**: `internal/service/`

---

### Handler 实现

需要实现以下 HTTP API 处理器：

- [ ] `AccountHandlerImpl` - 实现 `AccountHandler` 接口
- [ ] `TokenHandlerImpl` - 实现 `TokenHandler` 接口
- [ ] `ValidationHandlerImpl` - 实现 `ValidationHandler` 接口
- [ ] `AuditHandlerImpl` - 实现 `AuditHandler` 接口

**建议目录**: `internal/handlers/`

---

### 其他中间件

需要实现的可选中间件：

- [ ] `CORSMiddleware` - CORS 跨域处理
- [ ] `RateLimitMiddleware` - API 限流（基于 Redis）
- [ ] `LoggingMiddleware` - 请求日志记录

**建议目录**: `internal/middleware/`

---

## 🔗 快速查找

### 按功能查找

| 功能 | 接口定义 | 实现 | 文档 |
|------|---------|------|------|
| **HMAC 签名** | `interfaces/repository.go:145-165` | `auth/hmac.go` | `API.md:认证方式` |
| **认证中间件** | `interfaces/api.go:73-78` | `auth/middleware.go` | `ARCHITECTURE.md:认证模块` |
| **Scope 权限** | `interfaces/repository.go:121-133` | `permission/scope.go` | `API.md:Scope权限说明` |
| **Account 模型** | `interfaces/models.go:12-22` | 待实现 | `ARCHITECTURE.md:数据模型` |
| **Token 模型** | `interfaces/models.go:24-37` | 待实现 | `ARCHITECTURE.md:数据模型` |
| **审计日志** | `interfaces/models.go:48-60` | 待实现 | `API.md:审计日志` |

---

### 按 API 端点查找

| API 端点 | Handler 方法 | 接口定义 | 文档 |
|----------|-------------|---------|------|
| `POST /api/v2/accounts/register` | `AccountHandler.Register` | `interfaces/api.go:20-25` | `API.md:注册账户` |
| `GET /api/v2/accounts/me` | `AccountHandler.GetAccountInfo` | `interfaces/api.go:27-31` | `API.md:获取账户信息` |
| `POST /api/v2/tokens` | `TokenHandler.CreateToken` | `interfaces/api.go:40-46` | `API.md:创建Token` |
| `GET /api/v2/tokens` | `TokenHandler.ListTokens` | `interfaces/api.go:48-53` | `API.md:列出Tokens` |
| `POST /api/v2/validate` | `ValidationHandler.ValidateToken` | `interfaces/api.go:82-88` | `API.md:验证Token` |

---

## 📖 相关文档

| 文档 | 说明 |
|------|------|
| `ARCHITECTURE.md` | 架构设计、目录结构、模块职责 |
| `API.md` | API 端点文档、请求/响应示例 |
| `CLOUD-VENDOR-DESIGN.md` | 原始设计方案（对比 v1 vs v2） |
| `INDEX.md` | 本文件 - 实现索引 |

---

## 🚀 开发流程建议

### Phase 1: 数据访问层

1. 实现 `MongoAccountRepository`
2. 实现 `MongoTokenRepository`（重点：租户隔离）
3. 实现 `MongoAuditLogRepository`
4. 编写单元测试

### Phase 2: 业务逻辑层

1. 实现 `AccountServiceImpl`（注册、AK/SK 生成）
2. 实现 `TokenServiceImpl`（Token CRUD）
3. 实现 `ValidationServiceImpl`（Token 验证 + Scope 检查）
4. 实现 `AuditServiceImpl`（审计日志）
5. 编写单元测试

### Phase 3: API 层

1. 实现 `AccountHandlerImpl`
2. 实现 `TokenHandlerImpl`
3. 实现 `ValidationHandlerImpl`
4. 集成 `HMACMiddleware`
5. 编写集成测试

### Phase 4: 增强功能

1. 实现 Rate Limiting（Redis）
2. 添加 Token 使用统计
3. 实现 AK/SK 轮换
4. 添加 Token 自动过期清理

---

## 🧪 测试建议

### 单元测试

- `auth/hmac_test.go` - HMAC 签名测试
- `permission/scope_test.go` - Scope 权限验证测试
- `repository/*_test.go` - Repository 测试
- `service/*_test.go` - Service 测试

### 集成测试

- 完整的 API 请求流程测试
- 认证中间件集成测试
- 租户隔离测试

---

**索引版本**: 1.0
**更新日期**: 2025-12-25
