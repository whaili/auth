# Bearer Token Service V2

> 云厂商级多租户认证服务 - 基于 HMAC 签名、Scope 权限控制、租户隔离

---

## 🚀 快速开始

### 1. 启动 MongoDB

```bash
# Docker 方式
docker run -d -p 27017:27017 --name mongodb mongo:latest

# 或使用已有的 MongoDB 实例
export MONGO_URI="mongodb://localhost:27017"
```

### 2. 运行服务

```bash
cd /root/src/auth/bearer-token-service.v1/v2

# 安装依赖
go mod download

# 运行服务
go run cmd/server/main.go

# 或编译后运行
go build -o bin/token-service cmd/server/main.go
./bin/token-service
```

### 3. 测试 API

```bash
# 注册账户
curl -X POST http://localhost:8080/api/v2/accounts/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "company": "Example Inc",
    "password": "securePassword123"
  }'

# 响应会包含 AccessKey 和 SecretKey，保存好！
```

---

## 📖 完整文档

| 文档 | 说明 |
|------|------|
| [API.md](./API.md) | API 使用文档 |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | 架构设计文档 |
| [INDEX.md](./INDEX.md) | 实现索引 |
| [CLOUD-VENDOR-DESIGN.md](../CLOUD-VENDOR-DESIGN.md) | 设计对比 |

---

## 🏗️ 项目结构

```
v2/
├── cmd/server/          # 服务入口
│   └── main.go
├── auth/                # 认证模块
│   ├── hmac.go          # HMAC 签名
│   └── middleware.go    # 认证中间件
├── permission/          # 权限模块
│   └── scope.go         # Scope 验证
├── repository/          # 数据访问层
│   ├── mongo_account_repo.go
│   ├── mongo_token_repo.go
│   └── mongo_audit_repo.go
├── service/             # 业务逻辑层
│   ├── account_service.go
│   ├── token_service.go
│   ├── validation_service.go
│   └── audit_service.go
├── handlers/            # HTTP 处理器
│   ├── account_handler.go
│   ├── token_handler.go
│   └── validation_handler.go
└── interfaces/          # 接口定义
    ├── models.go
    ├── repository.go
    └── api.go
```

---

## ✨ 核心特性

### 1. 多租户隔离
```go
// 所有 Token 查询自动添加租户过滤
filter := bson.M{
    "account_id": accountID,  // 强制租户隔离
}
```

### 2. HMAC 签名认证
```bash
# 时间戳防重放（15 分钟窗口）
Authorization: QINIU {AccessKey}:{Signature}
X-Qiniu-Date: 2025-12-25T10:00:00Z
```

### 3. Scope 权限控制
```json
{
  "scope": ["storage:read", "storage:write", "cdn:*"]
}
```

支持：
- 精确匹配：`storage:read`
- 前缀通配：`storage:*`
- 全局通配：`*`

---

## 🔐 使用示例

### Python 客户端

```python
import hmac
import hashlib
import base64
from datetime import datetime
import requests

class TokenClient:
    def __init__(self, access_key, secret_key):
        self.access_key = access_key
        self.secret_key = secret_key
        self.base_url = "http://localhost:8080"

    def _sign(self, method, uri, timestamp, body):
        string_to_sign = f"{method}\n{uri}\n{timestamp}\n{body}"
        signature = hmac.new(
            self.secret_key.encode(),
            string_to_sign.encode(),
            hashlib.sha256
        ).digest()
        return base64.b64encode(signature).decode()

    def create_token(self, description, scope, expires_in_days):
        uri = "/api/v2/tokens"
        timestamp = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
        body = json.dumps({
            "description": description,
            "scope": scope,
            "expires_in_days": expires_in_days
        })

        signature = self._sign("POST", uri, timestamp, body)

        headers = {
            "Authorization": f"QINIU {self.access_key}:{signature}",
            "X-Qiniu-Date": timestamp,
            "Content-Type": "application/json"
        }

        response = requests.post(f"{self.base_url}{uri}", headers=headers, data=body)
        return response.json()

# 使用
client = TokenClient(
    access_key="AK_...",
    secret_key="SK_..."
)

token = client.create_token(
    description="Production token",
    scope=["storage:read", "cdn:refresh"],
    expires_in_days=90
)
print(token)
```

---

## 🧪 测试

```bash
# 单元测试
go test ./...

# 测试覆盖率
go test -cover ./...

# 集成测试（需要 MongoDB）
go test -tags=integration ./...
```

---

## 🚧 待实现功能

- [ ] Rate Limiting（基于 Redis）
- [ ] Token 使用统计详情（每日统计）
- [ ] AK/SK 自动轮换
- [ ] IP 白名单
- [ ] Webhook 通知
- [ ] 子账户（IAM Users）

---

## 📝 环境变量

```bash
# MongoDB 连接
export MONGO_URI="mongodb://localhost:27017"

# 服务端口
export PORT="8080"

# HMAC 时间戳容忍度（可选）
export TIMESTAMP_TOLERANCE="15m"
```

---

## 🆚 与 V1 的区别

| 特性 | V1 | V2 |
|------|----|----|
| 认证方式 | Basic Auth | HMAC 签名 |
| 租户隔离 | ❌ 无 | ✅ 完全隔离 |
| 权限控制 | ❌ 无 | ✅ Scope 权限 |
| 防重放攻击 | ❌ 无 | ✅ 时间戳验证 |
| 审计日志 | ❌ 无 | ✅ 完整审计 |
| 生产就绪 | ⚠️ 内部使用 | ✅ 云服务级别 |

---

## 📄 License

MIT

---

**版本**: 2.0
**更新日期**: 2025-12-25
**参考标准**: AWS Signature V4, Qiniu Qbox Auth, OAuth 2.0
