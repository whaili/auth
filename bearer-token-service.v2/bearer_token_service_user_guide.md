# Bearer Token Service V2 - 用户接入手册

> 本手册面向业务系统开发者，介绍如何接入和使用 Bearer Token Service 进行认证和鉴权。

## 📚 目录

- [概述](#概述)
- [快速开始](#快速开始)
- [认证方式](#认证方式)
- [API 使用指南](#api-使用指南)
- [客户端开发](#客户端开发)
- [权限控制](#权限控制)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)
- [错误码参考](#错误码参考)

---

## 概述

### 什么是 Bearer Token Service?

Bearer Token Service 是**七牛云统一的 Bearer Token 鉴权服务**，在七牛内网环境运行，为所有内部业务系统提供统一的 Token 管理和验证能力。

**系统定位**：
- 🏢 **内网服务**: 部署在七牛内网，仅供内部业务系统访问
- 🔐 **统一鉴权**: 作为七牛云的统一 Bearer Token 鉴权中心
- 🌐 **服务地址**: `http://bearer.qiniu.io`（仅内网可访问）

**核心能力**：
- **Bearer Token 管理**: 创建、管理、验证 API Token
- **细粒度权限控制**: 基于 Scope 的权限系统
- **安全性保障**: Token 过期管理、状态控制

### 适用场景

- API 服务的访问认证
- 微服务之间的鉴权
- 第三方应用接入
- 移动端/前端应用认证
- 自动化脚本/CI/CD 工具认证

### ⚠️ 重要提示

**业务系统必须记录用户信息与 Token 的对应关系！**

Bearer Token Service 本身**不存储**用户业务信息（如用户名、邮箱、手机号等），仅管理 Token 的生命周期和权限。因此：

1. **业务系统职责**：
   - 在创建 Token 后，必须在自己的数据库中记录 `token_id` 或 `token` 与用户信息的映射关系
   - 建议记录：用户 ID、用户名、Token ID、创建时间、用途等

2. **为什么必须记录**：
   - Token 验证接口只返回 Token 是否有效和权限信息，不返回用户业务信息
   - 业务系统需要根据 Token 查询到对应的用户，才能执行业务逻辑
   - 便于后续审计、统计和管理

---

## 快速开始

### 前提条件

在开始使用之前，您需要确认您的七牛用户身份信息：

- **UID**: 七牛用户 ID
- **UT**: 用户类型（通常为 1）

用户通过七牛官网注册登录后，可以在密钥管理页面为各业务线创建和管理 Token

![用户创建Token流程](https://raw.githubusercontent.com/whaili/sharefile/refs/heads/main/%E7%94%A8%E6%88%B7%E5%88%9B%E5%BB%BAtoken.png)

### 第一步：创建 Bearer Token

使用 QiniuStub 认证创建 Token，无需签名，使用简单。

**Python 示例**：

```python
import json
import requests

BASE_URL = "http://bearer.qiniu.io"

# 七牛用户信息
QINIU_UID = "1369077332"  # 您的七牛 UID
QINIU_UT = "1"            # 用户类型

# 准备请求
uri = "/api/v2/tokens"
body_data = {
    "description": "Production API Token",
    "scope": ["storage:read", "storage:write"],
    "expires_in_seconds": 7776000  # 90天 = 90 * 24 * 3600
}

# 使用 QiniuStub 认证
headers = {
    "Authorization": f"QiniuStub uid={QINIU_UID}&ut={QINIU_UT}",
    "Content-Type": "application/json"
}

response = requests.post(f"{BASE_URL}{uri}", headers=headers, json=body_data)
token_info = response.json()

print(f"Token: {token_info['token']}")
print(f"Token ID: {token_info['token_id']}")
print(f"Expires At: {token_info['expires_at']}")
```

**curl 示例**：

```bash
curl -X POST http://bearer.qiniu.io/api/v2/tokens \
  -H "Authorization: QiniuStub uid=1369077332&ut=1" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Production API Token",
    "scope": ["storage:read", "storage:write"],
    "expires_in_seconds": 7776000
  }'
```

**响应示例**：

```json
{
  "token_id": "tk_9z8y7x6w5v4u",
  "token": "sk-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2",
  "account_id": "acc_1a2b3c4d5e6f",
  "description": "Production API Token",
  "scope": ["storage:read", "storage:write"],
  "created_at": "2025-12-30T10:00:00Z",
  "expires_at": "2026-03-30T10:00:00Z",
  "is_active": true,
  "status": "normal"
}
```

**⚠️ 重要：立即记录用户与 Token 的映射关系！**

Token 创建成功后，**必须立即在业务系统数据库中记录**用户信息与 Token 的对应关系：

```python
# 示例：创建 Token 后立即保存映射关系
token_info = response.json()

# 保存到业务系统数据库
save_user_token_mapping(
    user_id=current_user.id,           # 业务用户 ID
    username=current_user.username,    # 用户名
    token_id=token_info['token_id'],   # Token ID
    token=token_info['token'],         # Token 值（可选，建议加密存储）
    description=token_info['description'],
    expires_at=token_info['expires_at']
)

print(f"✅ Token 创建成功并已记录映射关系")
print(f"   User: {current_user.username}")
print(f"   Token ID: {token_info['token_id']}")
```

### 第二步：使用 Bearer Token 调用 API

现在可以使用 Bearer Token 调用您的业务 API 了。

```bash
curl -X POST http://your-business-api.com/api/upload \
  -H "Authorization: Bearer sk-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2" \
  -H "Content-Type: application/json" \
  -d '{"file": "image.png"}'
```

在您的业务 API 中，需要调用验证接口验证 Token：

```python
# 在您的业务 API 中
def verify_token(bearer_token, required_scope=None):
    """验证 Bearer Token"""
    validate_url = "http://bearer.qiniu.io/api/v2/validate"

    headers = {"Authorization": f"Bearer {bearer_token}"}
    payload = {}

    if required_scope:
        payload["required_scope"] = required_scope

    response = requests.post(validate_url, headers=headers, json=payload)

    if response.status_code == 200:
        result = response.json()
        return result.get("valid", False)

    return False

# 使用示例
token = request.headers.get("Authorization", "").replace("Bearer ", "")
if not verify_token(token, "storage:write"):
    return {"error": "Unauthorized"}, 401
```

---

## 认证方式

### QiniuStub 认证

七牛内部服务使用 QiniuStub 认证方式进行身份验证。

**请求头格式**：

```
Authorization: QiniuStub uid={用户ID}&ut={用户类型}
```

**示例**：
```bash
# 创建 Token 示例
curl -X POST http://bearer.qiniu.io/api/v2/tokens \
  -H "Authorization: QiniuStub uid=1369077332&ut=1" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Production API Token",
    "scope": ["storage:read", "storage:write"],
    "expires_in_seconds": 7776000
  }'
```

**参数说明**：
- `uid`: 七牛用户 ID
- `ut`: 用户类型（通常为 1）

---

## API 使用指南

### Token 管理

#### 1. 创建 Token

```bash
POST /api/v2/tokens
```

**请求体**：
```json
{
  "description": "Production API Token",
  "scope": ["storage:read", "storage:write", "cdn:refresh"],
  "expires_in_seconds": 7776000,
  "prefix": "sk-",
  "rate_limit": {
    "requests_per_minute": 1000
  }
}
```

**参数说明**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| description | string | 是 | Token 描述，建议说明用途 |
| scope | array | 是 | 权限范围，支持通配符 |
| expires_in_seconds | integer | 是 | 过期时间（秒），0 表示永不过期 |
| prefix | string | 否 | Token 前缀，默认 `sk-` |
| rate_limit | object | 否 | 频率限制配置 |

**过期时间参考**：

| 时长 | 秒数 | 适用场景 |
|------|------|----------|
| 5 分钟 | 300 | 临时测试 |
| 1 小时 | 3,600 | 临时访问 |
| 1 天 | 86,400 | 日常开发 |
| 7 天 | 604,800 | 短期项目 |
| 30 天 | 2,592,000 | 月度访问 |
| 90 天 | 7,776,000 | 季度访问 |
| 365 天 | 31,536,000 | 年度访问 |
| 永不过期 | 0 | 生产环境（慎用） |

#### 2. 列出 Tokens

```bash
GET /api/v2/tokens?active_only=true&limit=50&offset=0
```

**查询参数**：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| active_only | boolean | false | 仅显示激活的 Token |
| limit | integer | 50 | 返回数量（最大 100） |
| offset | integer | 0 | 偏移量 |

**响应**：
```json
{
  "account_id": "acc_1a2b3c4d5e6f",
  "tokens": [
    {
      "token_id": "tk_9z8y7x6w5v4u",
      "token_preview": "sk-a1b2c3d4e5f6g7******************************c9d0e1f2",
      "description": "Production API Token",
      "scope": ["storage:read", "storage:write"],
      "created_at": "2025-12-30T10:00:00Z",
      "expires_at": "2026-03-30T10:00:00Z",
      "is_active": true,
      "status": "normal",
      "total_requests": 12567,
      "last_used_at": "2025-12-30T09:45:00Z"
    }
  ],
  "total": 1
}
```

**Status 字段说明**：

- `normal`: 正常（未过期且已激活）
- `expired`: 已过期
- `disabled`: 已停用

#### 3. 获取 Token 详情

```bash
GET /api/v2/tokens/{token_id}
```

#### 4. 更新 Token 状态

启用或禁用 Token（不会删除）。

```bash
PUT /api/v2/tokens/{token_id}/status
```

**请求体**：
```json
{
  "is_active": false
}
```

#### 5. 删除 Token

永久删除 Token，无法恢复。

```bash
DELETE /api/v2/tokens/{token_id}
```

#### 6. 获取 Token 使用统计

```bash
GET /api/v2/tokens/{token_id}/stats
```

**响应**：
```json
{
  "token_id": "tk_9z8y7x6w5v4u",
  "total_requests": 12567,
  "last_used_at": "2025-12-30T09:45:00Z",
  "created_at": "2025-12-30T10:00:00Z"
}
```

### Token 验证

#### 验证 Token 有效性

在您的业务 API 中调用此接口验证 Token。

```bash
POST /api/v2/validate
```

**请求头**：
```
Authorization: Bearer sk-a1b2c3d4e5f6...
```

**请求体**（可选）：
```json
{
  "required_scope": "storage:write"
}
```

**响应（验证成功）**：
```json
{
  "valid": true,
  "message": "Token is valid",
  "token_info": {
    "token_id": "tk_9z8y7x6w5v4u",
    "uid": "1234567",
    "scope": ["storage:read", "storage:write"],
    "is_active": true,
    "expires_at": "2026-03-30T10:00:00Z"
  },
  "permission_check": {
    "requested": "storage:write",
    "granted": true
  }
}
```

**响应（验证失败）**：
```json
{
  "valid": false,
  "message": "Token has expired"
}
```

### 审计日志

#### 查询审计日志

```bash
GET /api/v2/audit-logs?action=create_token&limit=50
```

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| action | string | 过滤操作类型（如 `create_token`, `delete_token`） |
| resource_id | string | 过滤资源 ID |
| start_time | string | 开始时间（ISO 8601） |
| end_time | string | 结束时间（ISO 8601） |
| limit | integer | 返回数量（默认 50） |
| offset | integer | 偏移量（默认 0） |

**响应**：
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
      "timestamp": "2025-12-30T10:00:00Z"
    }
  ],
  "total": 1
}
```

---

## 客户端开发

### Python 客户端

完整的 Python 客户端示例：

```python
import json
import requests

class BearerTokenClient:
    """Bearer Token Service 客户端（使用 QiniuStub 认证）"""

    def __init__(self, qiniu_uid, qiniu_ut="1", base_url="http://bearer.qiniu.io"):
        """
        初始化客户端

        Args:
            qiniu_uid: 七牛用户 ID
            qiniu_ut: 用户类型（默认为 "1"）
            base_url: 服务地址
        """
        self.qiniu_uid = qiniu_uid
        self.qiniu_ut = qiniu_ut
        self.base_url = base_url

    def _get_headers(self):
        """获取 QiniuStub 认证请求头"""
        return {
            "Authorization": f"QiniuStub uid={self.qiniu_uid}&ut={self.qiniu_ut}",
            "Content-Type": "application/json"
        }

    def _request(self, method, uri, body=None):
        """发送请求"""
        url = f"{self.base_url}{uri}"
        headers = self._get_headers()

        if method == "GET":
            response = requests.get(url, headers=headers)
        elif method == "POST":
            response = requests.post(url, headers=headers, json=body)
        elif method == "PUT":
            response = requests.put(url, headers=headers, json=body)
        elif method == "DELETE":
            response = requests.delete(url, headers=headers)
        else:
            raise ValueError(f"Unsupported method: {method}")

        return response

    def create_token(self, description, scope, expires_in_seconds, prefix=None):
        """创建 Token"""
        uri = "/api/v2/tokens"
        payload = {
            "description": description,
            "scope": scope,
            "expires_in_seconds": expires_in_seconds
        }
        if prefix:
            payload["prefix"] = prefix

        response = self._request("POST", uri, payload)
        return response.json()

    def list_tokens(self, active_only=False, limit=50):
        """列出 Tokens"""
        uri = f"/api/v2/tokens?active_only={str(active_only).lower()}&limit={limit}"
        response = self._request("GET", uri)
        return response.json()

    def get_token(self, token_id):
        """获取 Token 详情"""
        uri = f"/api/v2/tokens/{token_id}"
        response = self._request("GET", uri)
        return response.json()

    def update_token_status(self, token_id, is_active):
        """更新 Token 状态"""
        uri = f"/api/v2/tokens/{token_id}/status"
        payload = {"is_active": is_active}
        response = self._request("PUT", uri, payload)
        return response.json()

    def delete_token(self, token_id):
        """删除 Token"""
        uri = f"/api/v2/tokens/{token_id}"
        response = self._request("DELETE", uri)
        return response.json()

    def validate_token(self, bearer_token, required_scope=None):
        """验证 Bearer Token"""
        url = f"{self.base_url}/api/v2/validate"
        headers = {"Authorization": f"Bearer {bearer_token}"}
        payload = {}

        if required_scope:
            payload["required_scope"] = required_scope

        response = requests.post(url, headers=headers, json=payload)
        return response.json()

# 使用示例
client = BearerTokenClient(
    qiniu_uid="1369077332",  # 您的七牛 UID
    qiniu_ut="1",
    base_url="http://bearer.qiniu.io"
)

# 创建 Token
token = client.create_token(
    description="API Token for Mobile App",
    scope=["storage:read", "storage:write"],
    expires_in_seconds=7776000  # 90天
)
print(f"Token created: {token['token']}")

# 列出所有 Tokens
tokens = client.list_tokens(active_only=True)
print(f"Total tokens: {tokens['total']}")

# 验证 Token
result = client.validate_token(token['token'], "storage:write")
print(f"Token valid: {result['valid']}")
```

### Go 客户端

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type BearerTokenClient struct {
    QiniuUID string
    QiniuUT  string
    BaseURL  string
}

func NewClient(qiniuUID, qiniuUT, baseURL string) *BearerTokenClient {
    return &BearerTokenClient{
        QiniuUID: qiniuUID,
        QiniuUT:  qiniuUT,
        BaseURL:  baseURL,
    }
}

func (c *BearerTokenClient) getHeaders() map[string]string {
    return map[string]string{
        "Authorization": fmt.Sprintf("QiniuStub uid=%s&ut=%s", c.QiniuUID, c.QiniuUT),
        "Content-Type":  "application/json",
    }
}

func (c *BearerTokenClient) request(method, uri string, body interface{}) (*http.Response, error) {
    url := c.BaseURL + uri

    var req *http.Request
    var err error

    if body != nil {
        bodyBytes, _ := json.Marshal(body)
        req, err = http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
    } else {
        req, err = http.NewRequest(method, url, nil)
    }

    if err != nil {
        return nil, err
    }

    headers := c.getHeaders()
    for k, v := range headers {
        req.Header.Set(k, v)
    }

    client := &http.Client{}
    return client.Do(req)
}

type CreateTokenRequest struct {
    Description      string   `json:"description"`
    Scope            []string `json:"scope"`
    ExpiresInSeconds int      `json:"expires_in_seconds"`
    Prefix           string   `json:"prefix,omitempty"`
}

type TokenResponse struct {
    TokenID   string    `json:"token_id"`
    Token     string    `json:"token"`
    AccountID string    `json:"account_id"`
    ExpiresAt time.Time `json:"expires_at"`
}

func (c *BearerTokenClient) CreateToken(req CreateTokenRequest) (*TokenResponse, error) {
    uri := "/api/v2/tokens"

    resp, err := c.request("POST", uri, req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var tokenResp TokenResponse
    if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
        return nil, err
    }

    return &tokenResp, nil
}

func (c *BearerTokenClient) ValidateToken(bearerToken, requiredScope string) (bool, error) {
    url := c.BaseURL + "/api/v2/validate"

    payload := map[string]string{}
    if requiredScope != "" {
        payload["required_scope"] = requiredScope
    }

    bodyBytes, _ := json.Marshal(payload)

    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
    req.Header.Set("Authorization", "Bearer "+bearerToken)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)

    return result["valid"].(bool), nil
}

func main() {
    client := NewClient(
        "1369077332",  // 您的七牛 UID
        "1",           // 用户类型
        "http://bearer.qiniu.io",
    )

    // 创建 Token
    token, err := client.CreateToken(CreateTokenRequest{
        Description:      "API Token for Backend Service",
        Scope:            []string{"storage:read", "storage:write"},
        ExpiresInSeconds: 7776000, // 90天
    })

    if err != nil {
        panic(err)
    }

    fmt.Printf("Token created: %s\n", token.Token)

    // 验证 Token
    valid, _ := client.ValidateToken(token.Token, "storage:write")
    fmt.Printf("Token valid: %v\n", valid)
}
```

### Node.js 客户端

```javascript
const axios = require('axios');

class BearerTokenClient {
    constructor(qiniuUID, qiniuUT = '1', baseURL = 'http://bearer.qiniu.io') {
        this.qiniuUID = qiniuUID;
        this.qiniuUT = qiniuUT;
        this.baseURL = baseURL;
    }

    getHeaders() {
        return {
            'Authorization': `QiniuStub uid=${this.qiniuUID}&ut=${this.qiniuUT}`,
            'Content-Type': 'application/json'
        };
    }

    async request(method, uri, body = null) {
        const url = `${this.baseURL}${uri}`;
        const headers = this.getHeaders();
        const config = { headers };

        if (method === 'GET') {
            return await axios.get(url, config);
        } else if (method === 'POST') {
            return await axios.post(url, body, config);
        } else if (method === 'PUT') {
            return await axios.put(url, body, config);
        } else if (method === 'DELETE') {
            return await axios.delete(url, config);
        }
    }

    async createToken(description, scope, expiresInSeconds, prefix = null) {
        const uri = '/api/v2/tokens';
        const payload = {
            description,
            scope,
            expires_in_seconds: expiresInSeconds
        };

        if (prefix) {
            payload.prefix = prefix;
        }

        const response = await this.request('POST', uri, payload);
        return response.data;
    }

    async listTokens(activeOnly = false, limit = 50) {
        const uri = `/api/v2/tokens?active_only=${activeOnly}&limit=${limit}`;
        const response = await this.request('GET', uri);
        return response.data;
    }

    async validateToken(bearerToken, requiredScope = null) {
        const url = `${this.baseURL}/api/v2/validate`;
        const headers = { 'Authorization': `Bearer ${bearerToken}` };
        const payload = {};

        if (requiredScope) {
            payload.required_scope = requiredScope;
        }

        const response = await axios.post(url, payload, { headers });
        return response.data;
    }
}

// 使用示例
(async () => {
    const client = new BearerTokenClient(
        '1369077332',  // 您的七牛 UID
        '1',           // 用户类型
        'http://bearer.qiniu.io'
    );

    // 创建 Token
    const token = await client.createToken(
        'API Token for Web App',
        ['storage:read', 'storage:write'],
        7776000  // 90天
    );
    console.log(`Token created: ${token.token}`);

    // 验证 Token
    const result = await client.validateToken(token.token, 'storage:write');
    console.log(`Token valid: ${result.valid}`);
})();
```

---

## 权限控制

### Scope 权限系统

Bearer Token 使用 Scope 进行细粒度权限控制。

#### Scope 格式

```
<resource>:<action>
```

示例：
- `storage:read` - 存储读权限
- `storage:write` - 存储写权限
- `cdn:refresh` - CDN 刷新权限
- `analytics:view` - 分析查看权限

#### 通配符支持

**前缀通配符**：
```json
{
  "scope": ["storage:*"]
}
```
匹配所有以 `storage:` 开头的权限：
- `storage:read` ✅
- `storage:write` ✅
- `storage:delete` ✅
- `cdn:refresh` ❌

**全局通配符**：
```json
{
  "scope": ["*"]
}
```
匹配所有权限（慎用，仅用于管理员 Token）。

#### 权限验证逻辑

验证时使用"最宽松匹配"原则：

1. **精确匹配**: Token 的 Scope 包含请求的权限
   ```
   Token Scope: ["storage:read", "storage:write"]
   Required: "storage:read"
   Result: ✅ Granted
   ```

2. **前缀通配匹配**: Token 的 Scope 包含通配符前缀
   ```
   Token Scope: ["storage:*"]
   Required: "storage:read"
   Result: ✅ Granted
   ```

3. **全局通配匹配**: Token 的 Scope 包含 `*`
   ```
   Token Scope: ["*"]
   Required: "any:permission"
   Result: ✅ Granted
   ```

4. **无匹配**: Token 没有对应权限
   ```
   Token Scope: ["storage:read"]
   Required: "storage:write"
   Result: ❌ Denied
   ```

#### 最佳实践

**按需授权原则**：
```json
// ✅ 推荐：精确授权
{
  "description": "Mobile App Read Token",
  "scope": ["storage:read", "analytics:view"]
}

// ❌ 不推荐：过度授权
{
  "description": "Mobile App Token",
  "scope": ["*"]
}
```

**分环境管理**：
```json
// 开发环境：较宽松
{
  "description": "Development Token",
  "scope": ["storage:*", "cdn:*"],
  "expires_in_seconds": 86400  // 1天
}

// 生产环境：最小权限
{
  "description": "Production Read-Only Token",
  "scope": ["storage:read"],
  "expires_in_seconds": 7776000  // 90天
}
```

**按角色授权**：
```json
// 管理员
{
  "scope": ["*"]
}

// 开发者
{
  "scope": ["storage:*", "cdn:refresh"]
}

// 只读用户
{
  "scope": ["storage:read", "analytics:view"]
}
```

---


## 错误码参考

### HTTP 状态码

| 状态码 | 说明 | 处理方式 |
|--------|------|----------|
| 200 | 成功 | - |
| 201 | 创建成功 | - |
| 400 | 请求参数错误 | 检查请求体格式和参数 |
| 401 | 认证失败 | 检查 QiniuStub 认证参数或 Bearer Token |
| 403 | 权限不足 | 检查 Token 的 Scope 权限 |
| 404 | 资源不存在 | 检查 Token ID 或资源 ID |
| 429 | 请求过于频繁 | 降低请求频率，稍后重试 |
| 500 | 服务器内部错误 | 联系管理员 |
| 503 | 服务不可用 | 稍后重试 |

### 业务错误码

**认证相关（4001-4099）**：

| 错误码 | 消息 | 说明 |
|--------|------|------|
| 4004 | Invalid bearer token | Bearer Token 无效 |
| 4005 | Token has expired | Token 已过期 |
| 4006 | Token is disabled | Token 已被禁用 |

**权限相关（4031-4099）**：

| 错误码 | 消息 | 说明 |
|--------|------|------|
| 4031 | Permission denied | 权限不足 |
| 4032 | Insufficient scope | Scope 权限不足 |

**资源相关（4041-4099）**：

| 错误码 | 消息 | 说明 |
|--------|------|------|
| 4041 | Token not found | Token 不存在 |
| 4042 | Account not found | 账户不存在 |

**请求相关（4001-4299）**：

| 错误码 | 消息 | 说明 |
|--------|------|------|
| 4291 | Too many requests | 请求频率超限 |
| 4292 | Rate limit exceeded | Token 频率限制超限 |
