# Bearer Token Service V2 - API 文档

> 云厂商级多租户认证服务 API 参考

---

## 认证方式

### HMAC 签名认证

所有账户管理和 Token 管理 API 都需要使用 HMAC-SHA256 签名认证。

#### 签名步骤

```bash
# 1. 构建待签名字符串
STRING_TO_SIGN = "<HTTP_METHOD>\n<URI_PATH>\n<TIMESTAMP>\n<REQUEST_BODY>"

# 2. 使用 SecretKey 生成 HMAC-SHA256 签名
SIGNATURE = Base64(HMAC-SHA256(STRING_TO_SIGN, SECRET_KEY))

# 3. 构造 Authorization Header
Authorization: QINIU <ACCESS_KEY>:<SIGNATURE>
X-Qiniu-Date: <TIMESTAMP>
```

#### 示例

```bash
# 请求参数
METHOD="POST"
URI="/api/v2/tokens"
TIMESTAMP="2025-12-25T10:00:00Z"
ACCESS_KEY="AK_f8e7d6c5b4a392817a6b5c4d3e2f1a0b9c8d7e6f5a4b3c2d1e0f9a8b7c6d5e4"
SECRET_KEY="SK_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2"
BODY='{"description":"test token","scope":["storage:read"],"expires_in_seconds":7776000}'  # 90天 = 90*24*3600秒

# 构建待签名字符串
STRING_TO_SIGN="${METHOD}\n${URI}\n${TIMESTAMP}\n${BODY}"

# 生成签名
SIGNATURE=$(echo -n "$STRING_TO_SIGN" | openssl dgst -sha256 -hmac "$SECRET_KEY" -binary | base64)

# 发送请求
curl -X POST "https://api.example.com/api/v2/tokens" \
  -H "Authorization: QINIU ${ACCESS_KEY}:${SIGNATURE}" \
  -H "X-Qiniu-Date: ${TIMESTAMP}" \
  -H "Content-Type: application/json" \
  -d "$BODY"
```

---

## API 端点

### 账户管理

#### 1. 注册账户

创建新的租户账户并获取 AccessKey/SecretKey。

**请求**

```http
POST /api/v2/accounts/register
Content-Type: application/json

{
  "email": "user@example.com",
  "company": "Example Inc",
  "password": "securePassword123"
}
```

**响应**

```json
{
  "account_id": "acc_1a2b3c4d5e6f",
  "email": "user@example.com",
  "company": "Example Inc",
  "access_key": "AK_f8e7d6c5b4a392817a6b5c4d3e2f1a0b9c8d7e6f5a4b3c2d1e0f9a8b7c6d5e4",
  "secret_key": "SK_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2",
  "created_at": "2025-12-25T10:00:00Z"
}
```

**注意**：`secret_key` 仅在注册时返回一次，请妥善保存！

---

#### 2. 获取账户信息

获取当前登录账户的信息。

**请求**

```http
GET /api/v2/accounts/me
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: 2025-12-25T10:00:00Z
```

**响应**

```json
{
  "id": "acc_1a2b3c4d5e6f",
  "email": "user@example.com",
  "company": "Example Inc",
  "access_key": "AK_f8e7d6c5b4a392817a6b5c4d3e2f1a0b9c8d7e6f5a4b3c2d1e0f9a8b7c6d5e4",
  "status": "active",
  "created_at": "2025-12-25T10:00:00Z",
  "updated_at": "2025-12-25T10:00:00Z"
}
```

---

#### 3. 重新生成 SecretKey

出于安全考虑，定期轮换 SecretKey。

**请求**

```http
POST /api/v2/accounts/regenerate-sk
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: 2025-12-25T10:00:00Z
```

**响应**

```json
{
  "access_key": "AK_f8e7d6c5b4a392817a6b5c4d3e2f1a0b9c8d7e6f5a4b3c2d1e0f9a8b7c6d5e4",
  "secret_key": "SK_b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2a3",
  "updated_at": "2025-12-25T10:30:00Z"
}
```

**注意**：新的 `secret_key` 仅显示一次！旧的 SecretKey 立即失效。

---

### Token 管理

#### 4. 创建 Bearer Token

创建一个新的 Bearer Token，并指定权限范围（Scope）。

**请求**

```http
POST /api/v2/tokens
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: 2025-12-25T10:00:00Z
Content-Type: application/json

{
  "description": "Production read-only token",
  "scope": ["storage:read", "cdn:refresh"],
  "expires_in_seconds": 7776000,
  "prefix": "custom_bearer_",
  "rate_limit": {
    "requests_per_minute": 1000
  }
}
```

**响应**

```json
{
  "token_id": "tk_9z8y7x6w5v4u",
  "token": "custom_bearer_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2",
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

**参数说明**

- `description`: Token 描述（必填）
- `scope`: 权限范围，支持通配符（必填）
  - `"storage:read"` - 存储读权限
  - `"storage:*"` - 存储所有权限
  - `"*"` - 全部权限
- `expires_in_seconds`: 过期时间（秒），0 表示永不过期（可选）
  - 示例：`3600`（1小时）、`86400`（1天）、`7776000`（90天）
- `prefix`: 自定义 Token 前缀（可选，默认 `sk-`，保持与 V1 兼容）
  - 示例：`"custom_bearer_"`, `"my_token_"`, `"prod_"`
- `rate_limit`: API 频率限制（可选）

---

#### 5. 列出 Tokens

列出当前账户的所有 Tokens。

**请求**

```http
GET /api/v2/tokens?active_only=true&limit=50&offset=0
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: 2025-12-25T10:00:00Z
```

**查询参数**

- `active_only`: 仅显示激活的 Token（默认 false）
- `limit`: 返回数量（默认 50，最大 100）
- `offset`: 偏移量（默认 0）

**响应**

```json
{
  "account_id": "acc_1a2b3c4d5e6f",
  "tokens": [
    {
      "token_id": "tk_9z8y7x6w5v4u",
      "token_preview": "sk-a1b2c3d4e5f6g7******************************c9d0e1f2",
      "description": "Production read-only token",
      "scope": ["storage:read", "cdn:refresh"],
      "created_at": "2025-12-25T10:00:00Z",
      "expires_at": "2026-03-25T10:00:00Z",
      "is_active": true,
      "total_requests": 12567,
      "last_used_at": "2025-12-25T09:45:00Z"
    }
  ],
  "total": 1
}
```

---

#### 6. 获取 Token 详情

获取单个 Token 的详细信息。

**请求**

```http
GET /api/v2/tokens/{token_id}
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: 2025-12-25T10:00:00Z
```

**响应**

```json
{
  "token_id": "tk_9z8y7x6w5v4u",
  "token": "sk-a1b2c3d4e5f6g7******************************c9d0e1f2",
  "account_id": "acc_1a2b3c4d5e6f",
  "description": "Production read-only token",
  "scope": ["storage:read", "cdn:refresh"],
  "created_at": "2025-12-25T10:00:00Z",
  "expires_at": "2026-03-25T10:00:00Z",
  "is_active": true,
  "total_requests": 12567,
  "last_used_at": "2025-12-25T09:45:00Z"
}
```

---

#### 7. 更新 Token 状态

启用或禁用 Token。

**请求**

```http
PUT /api/v2/tokens/{token_id}/status
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: 2025-12-25T10:00:00Z
Content-Type: application/json

{
  "is_active": false
}
```

**响应**

```json
{
  "token_id": "tk_9z8y7x6w5v4u",
  "is_active": false,
  "updated_at": "2025-12-25T10:30:00Z"
}
```

---

#### 8. 删除 Token

永久删除 Token。

**请求**

```http
DELETE /api/v2/tokens/{token_id}
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: 2025-12-25T10:00:00Z
```

**响应**

```json
{
  "message": "Token deleted successfully"
}
```

---

#### 9. 获取 Token 使用统计

查看 Token 的使用情况。

**请求**

```http
GET /api/v2/tokens/{token_id}/stats
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: 2025-12-25T10:00:00Z
```

**响应**

```json
{
  "token_id": "tk_9z8y7x6w5v4u",
  "total_requests": 12567,
  "last_used_at": "2025-12-25T09:45:00Z",
  "created_at": "2025-12-25T10:00:00Z"
}
```

---

### Token 验证

#### 10. 验证 Bearer Token

验证 Token 的有效性，并可选地检查特定权限。

**请求**

```http
POST /api/v2/validate
Authorization: Bearer sk-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2
Content-Type: application/json

{
  "required_scope": "storage:read"
}
```

**参数说明**

- `required_scope`: 可选，要求的权限

**响应（验证成功）**

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

**响应（验证失败）**

```json
{
  "valid": false,
  "message": "Token has expired"
}
```

---

### 审计日志

#### 11. 查询审计日志

查询当前账户的操作审计日志。

**请求**

```http
GET /api/v2/audit-logs?action=create_token&limit=50&offset=0
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: 2025-12-25T10:00:00Z
```

**查询参数**

- `action`: 过滤操作类型（可选）
- `resource_id`: 过滤资源 ID（可选）
- `start_time`: 开始时间（可选，ISO 8601）
- `end_time`: 结束时间（可选，ISO 8601）
- `limit`: 返回数量（默认 50）
- `offset`: 偏移量（默认 0）

**响应**

```json
{
  "account_id": "acc_1a2b3c4d5e6f",
  "logs": [
    {
      "id": "log_xyz123",
      "account_id": "acc_1a2b3c4d5e6f",
      "action": "create_token",
      "resource_id": "tk_9z8y7x6w5v4u",
      "ip": "203.0.113.42",
      "user_agent": "Mozilla/5.0...",
      "result": "success",
      "timestamp": "2025-12-25T10:00:00Z"
    }
  ],
  "total": 1
}
```

---

## 错误响应

所有 API 错误响应格式统一：

```json
{
  "code": 4001,
  "message": "Invalid signature",
  "details": "The provided signature does not match the expected signature",
  "request_id": "req_abc123"
}
```

### 常见错误码

| 错误码 | 说明 |
|-------|------|
| 400 | 请求参数错误 |
| 401 | 认证失败 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 429 | 请求过于频繁 |
| 4001 | 签名无效 |
| 4002 | 时间戳过期 |
| 4003 | AccessKey 不存在 |
| 4004 | 账户已暂停 |
| 4031 | 权限不足 |
| 4032 | Scope 未授权 |
| 4041 | Token 不存在 |
| 4042 | Token 已过期 |
| 4043 | Token 已禁用 |

---

## Scope 权限说明

### 权限格式

- `resource:action` - 资源:操作
- `resource:*` - 资源的所有操作
- `*` - 全部权限

### 预定义权限

| Scope | 说明 |
|-------|------|
| `storage:read` | 存储读权限 |
| `storage:write` | 存储写权限 |
| `storage:delete` | 存储删除权限 |
| `storage:*` | 存储所有权限 |
| `cdn:refresh` | CDN 刷新权限 |
| `cdn:purge` | CDN 清除权限 |
| `cdn:*` | CDN 所有权限 |
| `*` | 全部权限 |

### 权限匹配规则

1. **精确匹配**: `storage:read` 匹配 `storage:read`
2. **通配符匹配**: `storage:*` 匹配 `storage:read`, `storage:write` 等
3. **全局通配**: `*` 匹配所有权限

---

## 使用示例

### Python 客户端示例

```python
import hmac
import hashlib
import base64
from datetime import datetime
import requests

class QiniuTokenClient:
    def __init__(self, access_key, secret_key, base_url):
        self.access_key = access_key
        self.secret_key = secret_key
        self.base_url = base_url

    def _sign(self, method, uri, timestamp, body):
        string_to_sign = f"{method}\n{uri}\n{timestamp}\n{body}"
        signature = hmac.new(
            self.secret_key.encode(),
            string_to_sign.encode(),
            hashlib.sha256
        ).digest()
        return base64.b64encode(signature).decode()

    def create_token(self, description, scope, expires_in_seconds):
        uri = "/api/v2/tokens"
        timestamp = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
        body = json.dumps({
            "description": description,
            "scope": scope,
            "expires_in_seconds": expires_in_seconds
        })

        signature = self._sign("POST", uri, timestamp, body)

        headers = {
            "Authorization": f"QINIU {self.access_key}:{signature}",
            "X-Qiniu-Date": timestamp,
            "Content-Type": "application/json"
        }

        response = requests.post(
            f"{self.base_url}{uri}",
            headers=headers,
            data=body
        )

        return response.json()

# 使用示例
client = QiniuTokenClient(
    access_key="AK_f8e7d6c5b4a392817a6b5c4d3e2f1a0b9c8d7e6f5a4b3c2d1e0f9a8b7c6d5e4",
    secret_key="SK_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2",
    base_url="https://api.example.com"
)

token = client.create_token(
    description="Production token",
    scope=["storage:read", "cdn:refresh"],
    expires_in_seconds=7776000  # 90天 = 90*24*3600秒
)

print(token)
```

---

**API 版本**: 2.0
**文档日期**: 2025-12-25
**更新日志**: 初始版本
