# Bearer Token Service V2 - 云厂商级架构设计

> 基于多租户、HMAC 签名认证、Scope 权限控制的企业级架构

---

## 📖 目录

1. [架构设计理念](#架构设计理念)
2. [V1 vs V2 对比](#v1-vs-v2-对比)
3. [三层认证架构](#三层认证架构)
4. [核心模块设计](#核心模块设计)
5. [数据模型](#数据模型)
6. [API 端点设计](#api-端点设计)
7. [安全增强](#安全增强)
8. [与云厂商对比](#与知名云厂商对比)
9. [实现路径](#实现路径)

---

## 架构设计理念

### 设计目标

1. **多租户隔离**: 每个客户管理自己的 tokens，数据完全隔离
2. **安全认证**: 使用 Access Key/Secret Key HMAC 签名认证
3. **权限控制**: Token 支持细粒度的 scope/权限范围
4. **可扩展性**: 支持子账户、角色、策略等高级功能
5. **云服务级别**: 对标七牛云、AWS、阿里云的认证架构

---

## V1 vs V2 对比

### V1 架构局限性（单租户模式）

```
系统管理员 (admin:adminpassword)
    └─> 管理所有 Tokens (Basic Auth)
         └─> 创建的 Token 供所有人使用
              └─> 无法区分 Token 属于哪个客户
```

**V1 存在的问题:**
1. ❌ **单一管理员**: 所有客户共用一个管理员账户
2. ❌ **无租户隔离**: 无法区分 token 属于哪个客户/租户
3. ❌ **无权限控制**: 所有 token 权限相同
4. ❌ **不可扩展**: 无法支持多客户场景
5. ❌ **Basic Auth**: 不适合生产环境的 API 认证

**V1 适用场景:**
- ✅ 内部系统使用
- ✅ 单一组织/团队
- ✅ 简单的 API Key 管理

### 架构对比表

| 维度 | V1 实现 | V2 实现 (云厂商级) |
|------|---------|------------------|
| **租户模型** | 单租户 | 多租户 (Account-based) |
| **管理员认证** | Basic Auth (admin/pass) | Access Key + Secret Key (HMAC 签名) |
| **Token 归属** | 无归属 | 关联到租户账户 |
| **Token 前缀** | `sk-` | `sk-` (可自定义) |
| **Token 隐藏** | 中间隐藏 30 字符 | 中间隐藏 30 字符 |
| **权限控制** | 无 | Scope/Permissions/Policy |
| **租户隔离** | 无 | 严格隔离 (数据库级) |
| **API 认证** | Basic Auth | HMAC-SHA256 签名 |
| **子账户** | 不支持 | 支持 (IAM Users) |
| **审计日志** | 无 | 完整审计 (谁、何时、做了什么) |

---

## 三层认证架构

```
┌─────────────────────────────────────────────────────────────┐
│                     云厂商 Token 服务                         │
└─────────────────────────────────────────────────────────────┘

第一层：账户注册与管理
┌──────────────────────────────────────────────────────────┐
│  用户注册 → 创建租户账户 (Account)                         │
│  └─> 生成 Access Key (AK) + Secret Key (SK)             │
│       例: AK=AK_xxx, SK=SK_yyy              │
└──────────────────────────────────────────────────────────┘

第二层：租户使用 AK/SK 管理 Bearer Tokens
┌──────────────────────────────────────────────────────────┐
│  租户 A (AK_A/SK_A) ─┐                                    │
│                      ├─> 创建 Token-A1 (scope: read)     │
│                      ├─> 创建 Token-A2 (scope: write)    │
│                      └─> 列出/禁用/删除自己的 Tokens      │
├──────────────────────────────────────────────────────────┤
│  租户 B (AK_B/SK_B) ─┐                                    │
│                      ├─> 创建 Token-B1 (scope: admin)    │
│                      └─> 只能看到自己的 Tokens            │
└──────────────────────────────────────────────────────────┘

第三层：第三方使用 Bearer Token 访问资源
┌──────────────────────────────────────────────────────────┐
│  外部应用/用户 ─> 使用 Token-A1 (Bearer)                  │
│                └─> 只能执行 Token 允许的操作 (scope)      │
└──────────────────────────────────────────────────────────┘
```

---

## 核心模块设计

### 目录结构

```
bearer-token-service.v2/
├── cmd/
│   └── server/
│       └── main.go                 # 服务入口
├── interfaces/                     # 接口与模型定义
│   ├── models.go                   # 数据模型
│   ├── repository.go               # Repository 接口
│   └── api.go                      # API 接口定义
├── auth/                           # 认证模块
│   ├── hmac.go                     # HMAC 签名认证
│   └── middleware.go               # 认证中间件
├── permission/                     # 权限模块
│   └── scope.go                    # Scope 权限验证
├── repository/                     # 数据访问层
│   ├── mongo_account_repo.go       # Account Repository
│   ├── mongo_token_repo.go         # Token Repository
│   └── mongo_audit_repo.go         # AuditLog Repository
├── service/                        # 业务逻辑层
│   ├── account_service.go          # 账户管理服务
│   ├── token_service.go            # Token 管理服务
│   ├── validation_service.go       # Token 验证服务
│   └── audit_service.go            # 审计日志服务
├── handlers/                       # HTTP 处理器
│   ├── account_handler.go          # 账户相关 API
│   ├── token_handler.go            # Token 管理 API
│   └── validation_handler.go       # Token 验证 API
├── tests/                          # 测试脚本
│   ├── hmac_client.py              # HMAC 签名客户端
│   └── test_api.sh                 # 集成测试脚本
├── go.mod
└── go.sum
```

### 1. 认证模块 (auth/)

#### HMAC 签名认证流程

```
Client Request
    │
    ├─> Authorization: QINIU {AccessKey}:{Signature}
    ├─> X-Qiniu-Date: 2025-12-25T10:00:00Z
    ├─> Content-Type: application/json
    └─> Body: {...}
    │
    ▼
[HMACAuthMiddleware]
    │
    ├─> 1. 提取 AccessKey
    ├─> 2. 查询 Account & SecretKey
    ├─> 3. 重新计算签名
    ├─> 4. 对比签名一致性
    ├─> 5. 验证时间戳（防重放）
    └─> 6. 注入 Account 到 Context
    │
    ▼
[Business Handler]
```

#### HMAC 签名算法实现

**客户端签名生成:**
```bash
# 请求参数
METHOD="POST"
URI="/api/v2/tokens"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
ACCESS_KEY="AK_f8e7d6c5b4a39281"
SECRET_KEY="SK_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
BODY='{"description":"Token","scope":["storage:read"],"expires_in_days":90}'

# 构造签名字符串 (只签名 path，不包含 query)
STRING_TO_SIGN="${METHOD}\n${URI}\n${TIMESTAMP}\n${BODY}"

# 生成 HMAC-SHA256 签名
SIGNATURE=$(echo -n "$STRING_TO_SIGN" | openssl dgst -sha256 -hmac "$SECRET_KEY" -binary | base64)

# 发送请求
curl -X POST "http://localhost:8080/api/v2/tokens" \
  -H "Authorization: QINIU ${ACCESS_KEY}:${SIGNATURE}" \
  -H "X-Qiniu-Date: ${TIMESTAMP}" \
  -H "Content-Type: application/json" \
  -d "$BODY"
```

**服务端签名验证:**
```go
func VerifyHMACSignature(r *http.Request, secretKey string) (bool, error) {
    // 1. 提取签名
    authHeader := r.Header.Get("Authorization")
    // Format: "QINIU {AccessKey}:{Signature}"
    parts := strings.Split(authHeader, ":")
    receivedSignature := parts[1]

    // 2. 重新计算签名
    stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
        r.Method,
        r.URL.Path,  // 只使用 path，不包含 query
        r.Header.Get("X-Qiniu-Date"),
        getBodyString(r),
    )

    mac := hmac.New(sha256.New, []byte(secretKey))
    mac.Write([]byte(stringToSign))
    expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

    // 3. 对比签名
    return hmac.Equal([]byte(expectedSignature), []byte(receivedSignature)), nil
}
```

### 2. 权限模块 (permission/)

#### Scope 权限验证算法

```go
// 权限匹配规则：
// 1. 精确匹配: "storage:read" == "storage:read"
// 2. 全局通配: "*" 匹配所有权限
// 3. 前缀通配: "storage:*" 匹配 "storage:read", "storage:write" 等

支持的 Scope 示例:
- storage:read         # 存储读权限
- storage:write        # 存储写权限
- storage:*            # 存储所有权限
- cdn:refresh          # CDN 刷新权限
- cdn:*                # CDN 所有权限
- *                    # 全部权限

// 权限验证实现
func HasPermission(tokenScopes []string, requiredScope string) bool {
    for _, scope := range tokenScopes {
        if scope == requiredScope || scope == "*" {
            return true
        }

        // 支持通配符: storage:* 包含 storage:read
        if strings.HasSuffix(scope, ":*") {
            prefix := strings.TrimSuffix(scope, "*")
            if strings.HasPrefix(requiredScope, prefix) {
                return true
            }
        }
    }
    return false
}
```

### 3. 租户隔离策略

#### 数据隔离

```go
// 所有 Token 查询自动添加租户过滤
db.Find(bson.M{
    "account_id": accountID,  // 强制租户隔离
    "is_active": true,
})

// 创建 Token 时自动关联账户
token := &Token{
    AccountID: extractAccountIDFromContext(ctx),
    Token:     generateToken(),
    Scope:     []string{"storage:read"},
}
```

#### 上下文传递

```go
// 认证中间件注入账户信息到 Context
ctx := context.WithValue(r.Context(), "account", account)

// 业务层从 Context 提取账户
account := ctx.Value("account").(*models.Account)
```

---

## 数据模型

### 1. Account (租户账户)

```go
type Account struct {
    ID          string    `bson:"_id,omitempty" json:"id"`                // acc_xxx
    Email       string    `bson:"email" json:"email"`
    Company     string    `bson:"company" json:"company"`
    Password    string    `bson:"password" json:"-"`                      // bcrypt 加密
    AccessKey   string    `bson:"access_key" json:"access_key"`           // AK_xxx
    SecretKey   string    `bson:"secret_key" json:"-"`                    // 明文存储（HMAC 签名需要）
    Status      string    `bson:"status" json:"status"`                   // active, suspended
    CreatedAt   time.Time `bson:"created_at" json:"created_at"`
    UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}
```

**安全要求:**
- 🔒 SecretKey 只在创建/重新生成时显示一次
- 🔒 SecretKey 明文存储（HMAC 签名验证需要明文）
- 🔒 Password 使用 bcrypt 加密存储
- 🔒 支持 AK/SK 轮换

### 2. Token (Bearer Token)

```go
type Token struct {
    ID          string     `bson:"_id,omitempty" json:"token_id"`          // tk_xxx
    AccountID   string     `bson:"account_id" json:"account_id"`           // 关联到账户
    Token       string     `bson:"token" json:"token"`                     // sk-xxx...
    Description string     `bson:"description" json:"description"`
    Scope       []string   `bson:"scope" json:"scope"`                     // 权限范围
    RateLimit   *RateLimit `bson:"rate_limit,omitempty" json:"rate_limit,omitempty"`
    CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
    ExpiresAt   time.Time  `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
    IsActive    bool       `bson:"is_active" json:"is_active"`
    Prefix      string     `bson:"-" json:"-"`                             // 自定义前缀（不存储）

    // 使用统计
    TotalRequests int64     `bson:"total_requests" json:"total_requests"`
    LastUsedAt    time.Time `bson:"last_used_at,omitempty" json:"last_used_at,omitempty"`
}

type RateLimit struct {
    RequestsPerMinute int `bson:"requests_per_minute" json:"requests_per_minute"`
    RequestsPerHour   int `bson:"requests_per_hour,omitempty" json:"requests_per_hour,omitempty"`
    RequestsPerDay    int `bson:"requests_per_day,omitempty" json:"requests_per_day,omitempty"`
}
```

**Token 格式:**
- 默认前缀: `sk-`
- 自定义前缀: 支持（如 `custom_bearer_`）
- 总长度: 67 字符 (前缀 + 64 字符 hex)
- 隐藏格式: `sk-abc123...******************************...xyz789` (中间 30 个字符隐藏)

### 3. AuditLog (审计日志)

```go
type AuditLog struct {
    ID          string                 `bson:"_id,omitempty"`
    AccountID   string                 `bson:"account_id"`
    Action      string                 `bson:"action"`      // create_token, delete_token, etc.
    ResourceID  string                 `bson:"resource_id"` // token_id
    IP          string                 `bson:"ip"`
    UserAgent   string                 `bson:"user_agent"`
    RequestData map[string]interface{} `bson:"request_data"`
    Result      string                 `bson:"result"`      // success, failure
    Timestamp   time.Time              `bson:"timestamp"`
}
```

---

## API 端点设计

### 租户账户管理

| 端点 | 方法 | 认证 | 功能 | 示例 |
|------|------|------|------|------|
| `/api/v2/accounts/register` | POST | 无 | 注册账户，获取 AK/SK | [示例](#账户注册) |
| `/api/v2/accounts/me` | GET | HMAC | 获取当前账户信息 | [示例](#获取账户信息) |
| `/api/v2/accounts/regenerate-sk` | POST | HMAC | 重新生成 Secret Key | [示例](#重新生成-sk) |

#### 账户注册

**请求:**
```http
POST /api/v2/accounts/register
Content-Type: application/json

{
  "email": "customer@example.com",
  "company": "Example Inc",
  "password": "secure_password"
}
```

**响应:**
```json
{
  "account_id": "acc_1a2b3c4d5e6f",
  "email": "customer@example.com",
  "company": "Example Inc",
  "access_key": "AK_f8e7d6c5b4a39281",
  "secret_key": "SK_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "status": "active",
  "created_at": "2025-12-25T10:00:00Z"
}
```

### Token 管理（租户操作）

| 端点 | 方法 | 认证 | 功能 |
|------|------|------|------|
| `/api/v2/tokens` | POST | HMAC | 创建 Bearer Token（带 Scope）|
| `/api/v2/tokens` | GET | HMAC | 列出自己的 Tokens |
| `/api/v2/tokens/{id}` | GET | HMAC | 获取单个 Token 详情 |
| `/api/v2/tokens/{id}/status` | PUT | HMAC | 启用/禁用 Token |
| `/api/v2/tokens/{id}` | DELETE | HMAC | 删除 Token |
| `/api/v2/tokens/{id}/stats` | GET | HMAC | 获取 Token 使用统计 |

#### 创建 Bearer Token

**请求:**
```http
POST /api/v2/tokens
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: 2025-12-25T10:00:00Z
Content-Type: application/json

{
  "description": "Production read-only token",
  "scope": ["storage:read", "cdn:refresh"],
  "expires_in_days": 90,
  "prefix": "custom_bearer_",
  "rate_limit": {
    "requests_per_minute": 1000
  }
}
```

**响应:**
```json
{
  "token_id": "tk_9z8y7x6w5v4u",
  "token": "custom_bearer_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0",
  "account_id": "acc_1a2b3c4d5e6f",
  "description": "Production read-only token",
  "scope": ["storage:read", "cdn:refresh"],
  "rate_limit": {
    "requests_per_minute": 1000
  },
  "created_at": "2025-12-25T10:00:00Z",
  "expires_at": "2026-03-25T10:00:00Z",
  "is_active": true
}
```

#### 列出 Tokens（租户隔离）

**请求:**
```http
GET /api/v2/tokens?active_only=true&limit=50
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: 2025-12-25T10:00:00Z
```

**响应:**
```json
{
  "account_id": "acc_1a2b3c4d5e6f",
  "tokens": [
    {
      "token_id": "tk_9z8y7x6w5v4u",
      "token_preview": "sk-a1b2c3d4e5f6******************************v2w3x4y5z6",
      "description": "Production read-only token",
      "scope": ["storage:read", "cdn:refresh"],
      "created_at": "2025-12-25T10:00:00Z",
      "expires_at": "2026-03-25T10:00:00Z",
      "is_active": true,
      "total_requests": 125678,
      "last_used_at": "2025-12-25T09:45:00Z"
    }
  ],
  "total": 1
}
```

**租户隔离保证:**
- 🔒 只返回当前租户（AccessKey 对应的 account_id）的 tokens
- 🔒 数据库查询自动添加 `WHERE account_id = ?` 过滤

### Token 验证（公开 API）

| 端点 | 方法 | 认证 | 功能 |
|------|------|------|------|
| `/api/v2/validate` | POST | Bearer Token | 验证 Token + 权限检查 |

#### 验证 Bearer Token（带权限检查）

**请求:**
```http
POST /api/v2/validate
Authorization: Bearer sk-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
Content-Type: application/json

{
  "required_scope": "storage:read"
}
```

**响应:**
```json
{
  "valid": true,
  "message": "Token is valid",
  "token_info": {
    "token_id": "tk_9z8y7x6w5v4u",
    "account_id": "acc_1a2b3c4d5e6f",
    "scope": ["storage:read", "cdn:refresh"],
    "is_active": true,
    "expires_at": "2026-03-25T10:00:00Z"
  },
  "permission_check": {
    "requested": "storage:read",
    "granted": true
  }
}
```

**验证逻辑:**
```go
1. Token 存在
2. Token.is_active == true
3. Token.expires_at > now()
4. 如果请求了特定 scope:
   └─> 检查 Token.scope 是否包含请求的权限
```

---

## 安全增强

### 1. HMAC 签名防篡改

```
StringToSign =
    HTTP_METHOD + "\n" +
    URI_PATH + "\n" +        # 只使用 path，不包含 query
    TIMESTAMP + "\n" +
    REQUEST_BODY

Signature = Base64(HMAC-SHA256(StringToSign, SecretKey))
```

**请求头格式:**
```
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: {ISO8601 Timestamp}
```

### 2. 时间戳验证（防重放攻击）

```go
// 允许 ±15 分钟的时钟偏差
func ValidateTimestamp(timestamp string) error {
    requestTime, err := time.Parse(time.RFC3339, timestamp)
    if err != nil {
        return err
    }

    timeDiff := time.Since(requestTime).Abs()
    if timeDiff > 15*time.Minute {
        return errors.New("timestamp expired")
    }

    return nil
}
```

### 3. SecretKey 安全存储

```go
// SecretKey 明文存储（HMAC 签名验证需要明文）
// 安全依赖：
// 1. MongoDB 传输层加密（TLS）
// 2. MongoDB 访问控制（Authentication + Authorization）
// 3. 网络隔离（仅内网访问）
// 4. 定期轮换 SecretKey

// 验证时使用 constant-time 比较
hmac.Equal(expectedSignature, receivedSignature)
```

### 4. Token 隐藏显示

```go
// hideToken 隐藏 Token 的中间部分，保留前后明文
// 示例: sk-abc123...******************************...xyz789
func hideToken(token string) string {
    const (
        hiddenStart  = 15 // 从第 15 个字符开始隐藏
        hiddenLength = 30 // 隐藏 30 个字符
    )

    if len(token) < hiddenStart+hiddenLength {
        return token // Token 太短，直接返回
    }

    // 将中间部分替换为星号
    bytes := []byte(token)
    for i := hiddenStart; i < hiddenStart+hiddenLength && i < len(bytes); i++ {
        bytes[i] = '*'
    }
    return string(bytes)
}
```

---

## 与知名云厂商对比

### Qiniu (七牛云)

```bash
# 使用 AK/SK 上传文件 (Qbox 认证)
ACCESS_KEY="your_access_key"
SECRET_KEY="your_secret_key"

# 构造上传凭证 (类似本方案的 HMAC 签名)
POLICY=$(echo '{"scope":"bucket","deadline":1672531199}' | base64)
SIGN=$(echo -n "$POLICY" | openssl dgst -sha1 -hmac "$SECRET_KEY" -binary | base64)
UPLOAD_TOKEN="${ACCESS_KEY}:${SIGN}:${POLICY}"

curl -X POST http://upload.qiniup.com \
  -F "token=${UPLOAD_TOKEN}" \
  -F "file=@example.jpg"
```

**对比:**
- ✅ 本方案使用 HMAC-SHA256（比七牛的 SHA1 更安全）
- ✅ 本方案支持时间戳验证（防重放攻击）
- ✅ 本方案支持 Scope 权限控制

### AWS (Amazon Web Services)

```bash
# AWS Signature V4 (类似本方案的 HMAC 方式)
AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

# AWS CLI 自动处理签名
aws s3 ls s3://my-bucket \
  --profile production
```

**对比:**
- ✅ 本方案签名算法与 AWS Signature V4 类似
- ✅ 本方案更简化（适合中小规模应用）

### 阿里云 (Aliyun)

```bash
# 使用 AccessKey/SecretKey 调用 API
aliyun oss ls oss://bucket-name \
  --access-key-id=LTAI4G... \
  --access-key-secret=xxx
```

**对比:**
- ✅ 本方案的 AK/SK 机制与阿里云一致
- ✅ 本方案支持多租户隔离

---

## 错误处理规范

### 错误代码定义

```go
// 认证错误 (4001-4099)
ErrInvalidSignature     = 4001  // 签名无效
ErrTimestampExpired     = 4002  // 时间戳过期
ErrAccessKeyNotFound    = 4003  // AccessKey 不存在
ErrAccountSuspended     = 4004  // 账户已暂停

// 权限错误 (4031-4099)
ErrPermissionDenied     = 4031  // 权限不足
ErrScopeNotGranted      = 4032  // Scope 未授权

// Token 错误 (4041-4099)
ErrTokenNotFound        = 4041  // Token 不存在
ErrTokenExpired         = 4042  // Token 已过期
ErrTokenInactive        = 4043  // Token 已禁用

// 业务错误 (5001-5099)
ErrDuplicateEmail       = 5001  // 邮箱已存在
ErrInvalidScope         = 5002  // 无效的 Scope 格式
```

---

## 配置管理

### 环境变量

```yaml
# 服务配置
SERVER_PORT: 8080
SERVER_ENV: production

# MongoDB 配置
MONGO_URI: mongodb://localhost:27017
MONGO_DATABASE: token_service_v2

# 安全配置
HMAC_TIMESTAMP_TOLERANCE: 15m
SECRET_KEY_ROTATION_DAYS: 90
TOKEN_DEFAULT_EXPIRY_DAYS: 365

# Rate Limiting (未来)
RATE_LIMIT_ENABLED: true
RATE_LIMIT_REDIS_URI: redis://localhost:6379
```

---

## 性能优化

### 1. 数据库索引

```javascript
// MongoDB 索引策略
db.accounts.createIndex({ "access_key": 1 }, { unique: true })
db.accounts.createIndex({ "email": 1 }, { unique: true })

db.tokens.createIndex({ "account_id": 1, "is_active": 1 })
db.tokens.createIndex({ "token": 1 }, { unique: true })
db.tokens.createIndex({ "expires_at": 1 })

db.audit_logs.createIndex({ "account_id": 1, "timestamp": -1 })
```

### 2. 缓存策略（未来）

```go
// Token 验证缓存 (Redis)
cache.Set("token:"+tokenValue, tokenInfo, 5*time.Minute)

// Account 信息缓存
cache.Set("account:"+accessKey, account, 10*time.Minute)
```

---

## 实现路径

### Phase 1: MVP（最小可行产品）✅ 已完成

- ✅ 数据模型定义（Account, Token, AuditLog）
- ✅ HMAC 签名认证（auth/hmac.go）
- ✅ 租户隔离的 Token 管理
- ✅ Scope 权限验证（permission/scope.go）
- ✅ Bearer Token 验证 API
- ✅ 自定义 Token 前缀支持
- ✅ 与 V1 格式兼容（token 前缀 `sk-`，中间隐藏）

### Phase 2: 增强功能

- ✅ Token 使用统计（TotalRequests, LastUsedAt）
- ✅ 审计日志记录（AuditLog）
- ⏳ AK/SK 轮换
- ⏳ Rate Limiting（Redis）
- ⏳ Token 自动过期清理

### Phase 3: 企业级功能

- 📅 子账户（IAM Users）
- 📅 基于策略的权限控制（Policy）
- 📅 IP 白名单
- 📅 Webhook 通知
- 📅 多区域部署

---

## 兼容性

### V1 到 V2 迁移

```go
// 保留 V1 API 路径
router.HandleFunc("/api/tokens", v1Handler.CreateToken)

// 新增 V2 API 路径
router.HandleFunc("/api/v2/tokens", v2Handler.CreateToken)

// 数据迁移脚本
// 1. 创建默认 Account
// 2. 将所有 V1 Tokens 关联到默认 Account
// 3. 为 V1 Tokens 添加默认 Scope: ["*"]
```

### 向后兼容保证

| 特性 | V1 | V2 | 兼容性 |
|------|----|----|--------|
| Token 默认前缀 | `sk-` | `sk-` | ✅ 完全兼容 |
| Token 隐藏格式 | 中间隐藏 30 字符 | 中间隐藏 30 字符 | ✅ 完全兼容 |
| 自定义前缀 | ✅ 支持 | ✅ 支持 | ✅ 完全兼容 |
| Bearer Token 验证 | `/api/validate` | `/api/v2/validate` | ⚠️ 需要更新 endpoint |

---

## 总结

### 核心优势

1. **企业级多租户**: 完全的租户隔离，支持 SaaS 化部署
2. **云厂商级认证**: HMAC-SHA256 签名，对标七牛云、AWS、阿里云
3. **细粒度权限**: Scope 权限控制，支持通配符
4. **高安全性**: 签名防篡改、时间戳防重放、常量时间比较
5. **高可扩展**: 模块化设计，支持子账户、策略、审计等扩展
6. **生产就绪**: 完整的错误处理、审计日志、性能优化

### 下一步计划

1. **Rate Limiting**: 基于 Redis 的 API 频率限制
2. **AK/SK 轮换**: 支持无缝更新 SecretKey
3. **子账户系统**: IAM Users 管理
4. **策略引擎**: 基于策略的权限控制（Policy-based）
5. **多区域部署**: 数据同步与全球加速

---

**架构版本**: 2.0
**设计日期**: 2025-12-25
**参考标准**: AWS Signature V4, Qiniu Qbox Auth, OAuth 2.0
**实现状态**: Phase 1 已完成，Phase 2 进行中
