# 双认证模式使用指南

Bearer Token Service V2 现在支持**两种认证方式**来创建和管理 Token：

## 🔐 认证方式

### 1. HMAC 签名认证（标准方式）

使用 AccessKey/SecretKey 进行 HMAC-SHA256 签名认证。

**适用场景**：
- 外部客户端调用
- 需要高安全性的场景
- 使用自己的账户系统

**请求示例**：

```http
POST /api/v2/tokens
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: 2025-12-26T10:00:00Z
Content-Type: application/json

{
  "description": "Production read-only token",
  "scope": ["storage:read", "cdn:refresh"],
  "expires_in_seconds": 7776000,
  "rate_limit": {
    "requests_per_minute": 1000
  }
}
```

**签名计算**：
```
StringToSign = <Method>\n<Path>\n<Timestamp>\n<Body>
Signature = Base64(HMAC-SHA256(SecretKey, StringToSign))
```

---

### 2. Qstub Bearer Token 认证（七牛内部）

使用 Base64 编码的用户信息进行认证。

**适用场景**：
- 七牛内部服务调用
- 已有七牛用户系统
- 快速集成

**用户信息格式**：

```json
{
  "uid": "12345",
  "email": "user@example.com",
  "name": "张三"
}
```

**Token 生成**：

```bash
# 原始 JSON
echo '{"uid":"12345","email":"user@qiniu.com"}' | base64

# 输出（示例）
eyJ1aWQiOiIxMjM0NSIsImVtYWlsIjoidXNlckBxaW5pdS5jb20ifQ==
```

**请求示例**：

```http
POST /api/v2/tokens
Authorization: Bearer eyJ1aWQiOiIxMjM0NSIsImVtYWlsIjoidXNlckBxaW5pdS5jb20ifQ==
Content-Type: application/json

{
  "description": "My dev token",
  "scope": ["storage:*"],
  "expires_in_seconds": 86400
}
```

---

## 🔄 认证方式识别规则

系统会自动识别认证方式：

| 特征 | 认证方式 |
|------|---------|
| 有 `X-Qiniu-Date` 头 | HMAC 签名认证 |
| `Authorization: Bearer <token>` | Qstub Token 认证 |
| `Authorization: QINIU <ak>:<sig>` 但无 `X-Qiniu-Date` | 错误：缺少时间戳 |
| 其他 | 错误：不支持的认证方式 |

---

## 📋 完整示例

### 方式 1: HMAC 签名认证

```bash
#!/bin/bash
# 配置
ACCESS_KEY="ak_1a2b3c4d5e6f"
SECRET_KEY="sk_fedcba9876543210"  # 实际应从安全存储读取
METHOD="POST"
PATH="/api/v2/tokens"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BODY='{"description":"test token","scope":["storage:read"],"expires_in_seconds":3600}'

# 计算签名
STRING_TO_SIGN="${METHOD}\n${PATH}\n${TIMESTAMP}\n${BODY}"
SIGNATURE=$(echo -n "$STRING_TO_SIGN" | openssl dgst -sha256 -hmac "$SECRET_KEY" -binary | base64)

# 发送请求
curl -X POST http://localhost:8080/api/v2/tokens \
  -H "Authorization: QINIU ${ACCESS_KEY}:${SIGNATURE}" \
  -H "X-Qiniu-Date: ${TIMESTAMP}" \
  -H "Content-Type: application/json" \
  -d "$BODY"
```

**响应**：

```json
{
  "token_id": "tk_abc123",
  "token": "sk_1234567890abcdef...",
  "account_id": "acc_xyz789",
  "description": "test token",
  "scope": ["storage:read"],
  "created_at": "2025-12-26T10:00:00Z",
  "expires_at": "2025-12-26T11:00:00Z",
  "is_active": true
}
```

---

### 方式 2: Qstub Bearer Token

```bash
#!/bin/bash
# 构建用户信息
USER_INFO='{"uid":"12345","email":"user@qiniu.com"}'
QSTUB_TOKEN=$(echo -n "$USER_INFO" | base64)

# 发送请求
curl -X POST http://localhost:8080/api/v2/tokens \
  -H "Authorization: Bearer ${QSTUB_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "My dev token",
    "scope": ["storage:*"],
    "expires_in_seconds": 86400
  }'
```

**响应**：

```json
{
  "token_id": "tk_def456",
  "token": "sk_fedcba0987654321...",
  "account_id": "qiniu_12345",
  "description": "My dev token",
  "scope": ["storage:*"],
  "created_at": "2025-12-26T10:00:00Z",
  "expires_at": "2025-12-27T10:00:00Z",
  "is_active": true
}
```

注意：`account_id` 格式为 `qiniu_{uid}`（简单映射模式）

---

## 🎯 账户 ID 映射策略

### 简单映射模式（默认）

七牛 UID 直接转换为 `account_id`：

```
UID: 12345  →  account_id: qiniu_12345
UID: 67890  →  account_id: qiniu_67890
```

**优点**：
- 无需数据库查询
- 快速集成
- 无需账户管理

**缺点**：
- 无法关联到已有账户系统

### 数据库映射模式（高级）

通过数据库查询或创建账户映射关系：

```go
// 在 main.go 中替换为数据库映射器
qiniuUIDMapper := auth.NewDatabaseQiniuUIDMapper(accountRepo, true)  // 自动创建账户
```

**优点**：
- 可关联到已有账户系统
- 支持账户元数据（email、状态等）
- 统一的账户管理

**缺点**：
- 需要额外数据库查询
- 需要实现 `AccountRepository` 接口

---

## 🔧 自定义认证方式

如果您需要其他认证方式（OAuth2、JWT 等），可以实现自己的中间件：

```go
func CustomAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. 验证您的认证方式
        userID := validateYourAuth(r)

        // 2. 将 account_id 注入到 Context（关键）
        ctx := context.WithValue(r.Context(), "account_id", userID)

        // 3. 继续执行
        next.ServeHTTP(w, r.WithContext(ctx))
    }
}

// 在路由中使用
router.HandleFunc("/api/v2/tokens", CustomAuthMiddleware(tokenHandler.CreateToken))
```

**核心要求**：只需在 Context 中设置 `account_id` 即可！

---

## 🚀 常见问题

### Q1: 两种认证方式可以同时使用吗？

可以！系统会自动识别：
- 如果有 `X-Qiniu-Date` 头，使用 HMAC
- 如果 Authorization 以 `Bearer ` 开头，使用 Qstub
- 两者互不干扰

### Q2: Qstub 认证是否需要注册账户？

**不需要**！使用简单映射模式时，系统会自动将七牛 UID 转换为 `account_id`。

### Q3: 如何切换到数据库映射模式？

修改 `cmd/server/main.go`：

```go
// 替换这一行
qiniuUIDMapper := auth.NewSimpleQiniuUIDMapper()

// 为
qiniuUIDMapper := auth.NewDatabaseQiniuUIDMapper(accountRepo, true)
```

然后实现 `GetAccountByQiniuUID` 和 `CreateAccountForQiniuUID` 方法。

### Q4: 如何禁用某种认证方式？

**禁用 HMAC**：直接使用 Qstub 中间件
```go
qstubMiddleware := auth.NewQstubMiddleware(qiniuUIDMapper)
```

**禁用 Qstub**：直接使用 HMAC 中间件
```go
hmacMiddleware := auth.NewHMACMiddleware(accountFetcher, 15*time.Minute)
```

---

## 📊 架构对比

| 特性 | HMAC 认证 | Qstub 认证 |
|------|---------|-----------|
| 安全性 | ⭐⭐⭐⭐⭐ 高（加密签名） | ⭐⭐⭐ 中（Base64 编码） |
| 性能 | ⭐⭐⭐ 中（需计算签名） | ⭐⭐⭐⭐ 好（仅解码） |
| 集成难度 | ⭐⭐⭐ 中（需签名计算） | ⭐⭐⭐⭐⭐ 易（直接 Base64） |
| 账户管理 | 需要注册账户 | 可选（简单模式免注册） |
| 适用场景 | 外部客户端 | 内部服务 |

---

## ✅ 总结

Bearer Token Service V2 现在支持**灵活的双认证模式**：

1. ✅ 保留原有 HMAC 签名认证（高安全性）
2. ✅ 新增 Qstub Bearer Token 认证（快速集成）
3. ✅ 自动识别认证方式（无需配置）
4. ✅ 支持自定义认证中间件（完全解耦）

**无需修改任何 Token 管理代码，即可同时支持两种认证方式！**
