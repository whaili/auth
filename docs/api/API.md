# Bearer Token Service V2 - API 文档

> 多租户 Token 认证服务 API 参考

---

## 认证方式

###  QiniuStub 认证

所有 Token 管理 API 都需要使用 QiniuStub 认证。

#### 认证格式

```http
Authorization: QiniuStub uid={uid}&ut={user_type}
```

或

```http
Authorization: QiniuStub uid={uid}&ut={user_type}&iuid={iam_uid}
```

#### 参数说明

| 参数 | 必需 | 说明 | 示例 |
|------|------|------|------|
| `uid` | ✅ | 七牛主账户 UID | `1369077332` |
| `ut` | ✅ | 用户类型 | `1` |
| `iuid` | ❌ | IAM 子账户 ID（如有） | `8901234` |
| `app` | ❌ | 应用 ID（保留字段） | `100` |
| `ak` | ❌ | AccessKey（保留字段） | `test_ak` |
| `eu` | ❌ | 终端用户（保留字段） | `user@example.com` |

#### 认证示例

**主账户认证**:
```bash
curl -X POST "http://bearer.qiniu.io/api/v2/tokens" \
  -H "Authorization: QiniuStub uid=1369077332&ut=1" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Upload token",
    "expires_in_seconds": 3600
  }'
```

**IAM 子账户认证**:
```bash
curl -X POST "http://bearer.qiniu.io/api/v2/tokens" \
  -H "Authorization: QiniuStub uid=1369077332&ut=1&iuid=8901234" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "IAM upload token",
    "expires_in_seconds": 3600
  }'
```

---

## API 端点

### Token 管理

#### 1. 创建 Token

创建新的 Bearer Token。

**请求**

```http
POST /api/v2/tokens
Authorization: QiniuStub uid=1369077332&ut=1
Content-Type: application/json

{
  "description": "Upload token",
  "expires_in_seconds": 3600,
  "prefix": "sk-"
}
```

**请求参数**

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `description` | string | ✅ | Token 描述 |
| `expires_in_seconds` | int | ❌ | 过期时间（秒），不传则永不过期 |
| `prefix` | string | ❌ | Token 前缀，默认 `sk-` |
| `rate_limit` | object | ❌ | 限流配置 |

**响应**

```json
{
  "token_id": "tk_abc123",
  "token": "sk-abc123def456...",
  "account_id": "qiniu_1369077332",
  "description": "Upload token",
  "rate_limit": null,
  "created_at": "2026-01-12T10:00:00Z",
  "expires_at": "2026-01-12T11:00:00Z",
  "is_active": true
}
```

**注意**:
- 完整的 `token` 值仅在创建时返回一次，请妥善保存
- `account_id` 格式为 `qiniu_{uid}`，由系统自动生成
- 如果请求中包含 IUID，Token 会关联该 IAM 子账户

---

#### 2. 列出 Tokens

列出当前账户下的所有 Tokens。

**请求**

```http
GET /api/v2/tokens?active_only=true&limit=50&offset=0
Authorization: QiniuStub uid=1369077332&ut=1
```

**查询参数**

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `active_only` | bool | ❌ | 只返回激活的 Token（默认 `false`） |
| `limit` | int | ❌ | 返回数量（默认 50） |
| `offset` | int | ❌ | 偏移量（默认 0） |

**响应**

```json
{
  "account_id": "qiniu_1369077332",
  "tokens": [
    {
      "token_id": "tk_abc123",
      "token_preview": "sk-a1b2c3d4****e5f6g7h8",
      "description": "Upload token",
      "rate_limit": null,
      "created_at": "2026-01-12T10:00:00Z",
      "expires_at": "2026-01-12T11:00:00Z",
      "is_active": true,
      "status": "normal",
      "total_requests": 1250,
      "last_used_at": "2026-01-12T10:30:00Z"
    }
  ],
  "total": 1
}
```

**Token 状态说明**:
- `normal`: 正常（未过期且已激活）
- `expired`: 已过期
- `disabled`: 已停用

---

#### 3. 获取 Token 详情

获取指定 Token 的详细信息。

**请求**

```http
GET /api/v2/tokens/{token_id}
Authorization: QiniuStub uid=1369077332&ut=1
```

**响应**

```json
{
  "token_id": "tk_abc123",
  "account_id": "qiniu_1369077332",
  "token": "sk-abc123def456...",
  "description": "Upload token",
  "rate_limit": null,
  "iuid": "",
  "created_at": "2026-01-12T10:00:00Z",
  "expires_at": "2026-01-12T11:00:00Z",
  "is_active": true,
  "total_requests": 1250,
  "last_used_at": "2026-01-12T10:30:00Z"
}
```

---

#### 4. 更新 Token 状态

启用或停用 Token。

**请求**

```http
PUT /api/v2/tokens/{token_id}/status
Authorization: QiniuStub uid=1369077332&ut=1
Content-Type: application/json

{
  "is_active": false
}
```

**请求参数**

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `is_active` | bool | ✅ | Token 激活状态 |

**响应**

```json
{
  "message": "Token status updated successfully"
}
```

---

#### 5. 删除 Token

删除指定的 Token。

**请求**

```http
DELETE /api/v2/tokens/{token_id}
Authorization: QiniuStub uid=1369077332&ut=1
```

**响应**

```json
{
  "message": "Token deleted successfully"
}
```

---

#### 6. 获取 Token 使用统计

获取指定 Token 的使用统计信息。

**请求**

```http
GET /api/v2/tokens/{token_id}/stats
Authorization: QiniuStub uid=1369077332&ut=1
```

**响应**

```json
{
  "token_id": "tk_abc123",
  "total_requests": 1250,
  "last_used_at": "2026-01-12T10:30:00Z",
  "created_at": "2026-01-12T10:00:00Z",
  "daily_stats": []
}
```

---

### Token 验证

#### 验证 Token

验证 Bearer Token 的有效性。

**请求**

```http
POST /api/v2/validate
Authorization: Bearer sk-abc123def456...
Content-Type: application/json
```

**响应（验证成功）**

```json
{
  "valid": true,
  "message": "Token is valid",
  "token_info": {
    "token_id": "tk_abc123",
    "uid": "1369077332",
    "iuid": "",
    "is_active": true,
    "expires_at": "2026-01-12T11:00:00Z",
    "last_used_at": "2026-01-12T10:30:00Z"
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

**Token 信息字段说明**:

| 字段 | 类型 | 说明 |
|------|------|------|
| `token_id` | string | Token ID |
| `uid` | string | 主账户 UID（从 account_id 提取） |
| `iuid` | string | IAM 子账户 ID（如果创建时包含） |
| `is_active` | bool | Token 是否激活 |
| `expires_at` | string | 过期时间（null 表示永不过期） |
| `last_used_at` | string | 最后使用时间 |

**注意**:
- 验证成功会异步记录使用统计，不影响响应速度
- `uid` 字段仅在 QiniuStub 认证创建的 Token 中返回
- `iuid` 字段仅在 IAM 子账户创建的 Token 中返回

---

## 健康检查

#### GET /health

检查服务健康状态（无需认证）。

**请求**

```http
GET /health
```

**响应**

```json
{
  "status": "ok"
}
```

---

## 错误响应

所有错误响应统一格式：

```json
{
  "error": "错误描述",
  "code": 400,
  "timestamp": "2026-01-12T10:00:00Z"
}
```

### 常见错误码

| HTTP 状态码 | 说明 |
|------------|------|
| `400` | 请求参数错误 |
| `401` | 未认证或认证失败 |
| `403` | 无权限访问 |
| `404` | 资源不存在 |
| `409` | 资源冲突 |
| `429` | 请求限流 |
| `500` | 服务器内部错误 |

---

## 限流配置

Token 可以配置限流规则：

```json
{
  "rate_limit": {
    "requests_per_minute": 100,
    "requests_per_hour": 5000,
    "requests_per_day": 100000
  }
}
```

限流触发时返回 `429 Too Many Requests`，并包含以下响应头：

| 响应头 | 说明 |
|-------|------|
| `X-RateLimit-Limit-{Layer}` | 限流上限 |
| `X-RateLimit-Remaining-{Layer}` | 剩余请求数 |
| `X-RateLimit-Reset-{Layer}` | 限流重置时间（Unix 时间戳） |
| `Retry-After` | 建议重试时间（秒） |

`{Layer}` 可能的值：
- `App`: 应用层限流
- `Account`: 账户层限流
- `Token`: Token 层限流

---

## 数据模型

### Token

| 字段 | 类型 | 说明 |
|------|------|------|
| `token_id` | string | Token 唯一标识 |
| `account_id` | string | 关联账户 ID |
| `token` | string | Token 值（sk-开头） |
| `description` | string | Token 描述 |
| `rate_limit` | object | 限流配置 |
| `iuid` | string | IAM 子账户 ID（可选） |
| `created_at` | datetime | 创建时间 |
| `expires_at` | datetime | 过期时间（null=永不过期） |
| `is_active` | bool | 是否激活 |
| `total_requests` | int | 总请求次数 |
| `last_used_at` | datetime | 最后使用时间 |

---

## 完整使用示例

### 1. 主账户创建并使用 Token

```bash
# 1. 创建 Token
curl -X POST "http://localhost:8081/api/v2/tokens" \
  -H "Authorization: QiniuStub uid=1369077332&ut=1" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Upload token",
    "expires_in_seconds": 3600
  }'

# 响应：
# {
#   "token_id": "tk_abc123",
#   "token": "sk-abc123def456...",
#   "account_id": "qiniu_1369077332",
#   ...
# }

# 2. 使用 Token 进行验证
curl -X POST "http://localhost:8081/api/v2/validate" \
  -H "Authorization: Bearer sk-abc123def456..." \
  -H "Content-Type: application/json"

# 3. 列出所有 Tokens
curl -X GET "http://localhost:8081/api/v2/tokens" \
  -H "Authorization: QiniuStub uid=1369077332&ut=1"

# 4. 停用 Token
curl -X PUT "http://localhost:8081/api/v2/tokens/tk_abc123/status" \
  -H "Authorization: QiniuStub uid=1369077332&ut=1" \
  -H "Content-Type: application/json" \
  -d '{
    "is_active": false
  }'

# 5. 删除 Token
curl -X DELETE "http://localhost:8081/api/v2/tokens/tk_abc123" \
  -H "Authorization: QiniuStub uid=1369077332&ut=1"
```

### 2. IAM 子账户创建并使用 Token

```bash
# 1. IAM 子账户创建 Token（包含 iuid）
curl -X POST "http://localhost:8081/api/v2/tokens" \
  -H "Authorization: QiniuStub uid=1369077332&ut=1&iuid=8901234" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "IAM upload token",
    "expires_in_seconds": 3600
  }'

# 响应：
# {
#   "token_id": "tk_def456",
#   "token": "sk-def456ghi789...",
#   "account_id": "qiniu_1369077332",
#   ...
# }

# 2. 验证 Token（返回包含 iuid）
curl -X POST "http://localhost:8081/api/v2/validate" \
  -H "Authorization: Bearer sk-def456ghi789..." \
  -H "Content-Type: application/json"

# 响应：
# {
#   "valid": true,
#   "message": "Token is valid",
#   "token_info": {
#     "token_id": "tk_def456",
#     "uid": "1369077332",
#     "iuid": "8901234",
#     ...
#   }
# }
```

---

**最后更新**: 2026-01-12
**版本**: V2（简化版）
