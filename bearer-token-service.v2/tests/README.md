# 测试套件使用说明

> Bearer Token Service V2 完整测试工具集

---

## 📦 测试文件清单

| 文件 | 说明 | 用途 |
|------|------|------|
| `test_api.sh` | 自动化测试脚本 | 测试所有 API 端点 |
| `test_rate_limit_improved.sh` | **限流完整测试（推荐）** | 测试三层限流功能 |
| `test_rate_limit_quick.sh` | 限流快速测试 | 快速验证限流是否工作 |
| `test_rate_limit.sh` | 限流基础测试 | 原始版本（已过时） |
| `hmac_client.py` | HMAC 签名客户端 | Python 客户端库 + CLI 工具 |
| `README.md` | 本文件 | 完整测试指南 |

---

## 🚀 快速开始

### 方式 1：一键启动和测试

```bash
cd /root/src/auth/bearer-token-service.v1/v2
./quickstart.sh
```

这个脚本会自动：
1. 启动 MongoDB
2. 启动服务
3. 运行完整测试
4. 保存测试凭证

### 方式 2：手动启动

```bash
# 1. 启动 MongoDB
docker run -d -p 27017:27017 --name mongodb-test mongo:latest

# 2. 启动服务
cd /root/src/auth/bearer-token-service.v1/v2
go run cmd/server/main.go

# 3. 新终端运行测试
cd tests
./test_api.sh
```

---

## 📋 测试覆盖

自动化测试脚本 (`test_api.sh`) 会测试以下功能：

| # | 测试项 | API 端点 | 说明 |
|---|--------|---------|------|
| 0 | 健康检查 | `GET /health` | 服务状态 |
| 1 | 注册账户 | `POST /api/v2/accounts/register` | 获取 AK/SK |
| 2 | 获取账户信息 | `GET /api/v2/accounts/me` | HMAC 认证测试 |
| 3 | 创建 Token | `POST /api/v2/tokens` | 不同 Scope |
| 4 | 列出 Tokens | `GET /api/v2/tokens` | 租户隔离 |
| 5 | 获取 Token 详情 | `GET /api/v2/tokens/{id}` | - |
| 6 | 验证 Token | `POST /api/v2/validate` | Bearer Token |
| 7 | Scope 权限检查 | `POST /api/v2/validate` | 权限验证 |
| 8 | 更新 Token 状态 | `PUT /api/v2/tokens/{id}/status` | 启用/禁用 |
| 9 | Token 统计 | `GET /api/v2/tokens/{id}/stats` | 使用统计 |
| 10 | 重新生成 SK | `POST /api/v2/accounts/regenerate-sk` | 密钥轮换 |
| 11 | 删除 Token | `DELETE /api/v2/tokens/{id}` | - |

---

## 🧪 使用 Bash 测试脚本

### 完整测试

```bash
cd /root/src/auth/bearer-token-service.v1/v2/tests
./test_api.sh
```

### 自定义配置

```bash
# 指定服务地址
BASE_URL=http://localhost:8081 ./test_api.sh

# 使用不同端口
BASE_URL=http://localhost:9090 ./test_api.sh
```

### 测试输出

测试脚本会显示：
- ✅ 成功的测试（绿色）
- ❌ 失败的测试（红色）
- ℹ️  信息提示（蓝色）
- ⚠️  警告信息（黄色）

测试凭证保存在：`/tmp/v2_test_credentials.env`

---

## 🐍 使用 Python 客户端

### 作为命令行工具

```bash
cd /root/src/auth/bearer-token-service.v1/v2/tests

# 创建 Token
python3 hmac_client.py create_token \
  "AK_xxx" \
  "SK_xxx" \
  "My token" \
  '["storage:read"]' \
  90

# 列出 Tokens
python3 hmac_client.py list_tokens \
  "AK_xxx" \
  "SK_xxx"

# 获取 Token 详情
python3 hmac_client.py get_token \
  "AK_xxx" \
  "SK_xxx" \
  "tk_xxx"

# 更新 Token 状态
python3 hmac_client.py update_token_status \
  "AK_xxx" \
  "SK_xxx" \
  "tk_xxx" \
  false

# 删除 Token
python3 hmac_client.py delete_token \
  "AK_xxx" \
  "SK_xxx" \
  "tk_xxx"
```

### 作为 Python 库

```python
from hmac_client import HMACClient

# 创建客户端
client = HMACClient(
    access_key="AK_xxx",
    secret_key="SK_xxx",
    base_url="http://localhost:8081"
)

# 创建 Token
token = client.create_token(
    description="Production token",
    scope=["storage:read", "cdn:refresh"],
    expires_in_days=90
)
print(token)

# 列出 Tokens
tokens = client.list_tokens()
print(tokens)

# 获取账户信息
account = client.get_account_info()
print(account)
```

---

## 🔧 手动测试教程

### 测试 1：注册账户

```bash
curl -X POST http://localhost:8081/api/v2/accounts/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "company": "Test Company",
    "password": "password123"
  }'
```

**期望响应**：
```json
{
  "account_id": "acc_xxx",
  "email": "test@example.com",
  "company": "Test Company",
  "access_key": "AK_xxx",
  "secret_key": "SK_xxx",
  "created_at": "2025-12-25T10:00:00Z"
}
```

⚠️ **保存 `access_key` 和 `secret_key`！**

---

### 测试 2：创建 Token

```bash
# 使用 HMAC 客户端
python3 tests/hmac_client.py create_token \
  "AK_xxx" \
  "SK_xxx" \
  "My first token" \
  '["storage:read","cdn:refresh"]' \
  90
```

**期望响应**：
```json
{
  "token_id": "tk_xxx",
  "token": "sk-xxx",
  "account_id": "acc_xxx",
  "description": "My first token",
  "scope": ["storage:read", "cdn:refresh"],
  "expires_at": "2026-03-25T10:00:00Z",
  "is_active": true
}
```

⚠️ **保存 `token` 值！**

---

### 测试 3：验证 Bearer Token

```bash
curl -X POST http://localhost:8081/api/v2/validate \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{"required_scope": "storage:read"}'
```

**期望响应**：
```json
{
  "valid": true,
  "message": "Token is valid",
  "token_info": {
    "token_id": "tk_xxx",
    "account_id": "acc_xxx",
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

---

### 测试 4：列出所有 Tokens

```bash
python3 tests/hmac_client.py list_tokens \
  "AK_xxx" \
  "SK_xxx"
```

---

## 🧪 高级测试场景

### 1. Scope 权限测试

#### 创建不同权限的 Token

```bash
# 1. 只读权限
python3 tests/hmac_client.py create_token \
  "$AK" "$SK" \
  "Read-only token" \
  '["storage:read"]' \
  90

# 2. 读写权限
python3 tests/hmac_client.py create_token \
  "$AK" "$SK" \
  "Read-write token" \
  '["storage:read","storage:write"]' \
  90

# 3. 通配符权限
python3 tests/hmac_client.py create_token \
  "$AK" "$SK" \
  "Storage all permissions" \
  '["storage:*"]' \
  90

# 4. 全部权限
python3 tests/hmac_client.py create_token \
  "$AK" "$SK" \
  "Admin token" \
  '["*"]' \
  365
```

#### 验证权限

```bash
# 测试 storage:read 权限（应该成功）
curl -X POST http://localhost:8081/api/v2/validate \
  -H "Authorization: Bearer <read-only-token>" \
  -H "Content-Type: application/json" \
  -d '{"required_scope": "storage:read"}'

# 测试 storage:write 权限（应该失败）
curl -X POST http://localhost:8081/api/v2/validate \
  -H "Authorization: Bearer <read-only-token>" \
  -H "Content-Type: application/json" \
  -d '{"required_scope": "storage:write"}'
```

---

### 2. HMAC 签名与防重放测试

```bash
# 1. 创建一个有效请求
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
echo "Current timestamp: $TIMESTAMP"

# 2. 使用旧时间戳（20 分钟前）
OLD_TIMESTAMP=$(date -u -d '20 minutes ago' +"%Y-%m-%dT%H:%M:%SZ")
echo "Old timestamp: $OLD_TIMESTAMP"

# 3. 尝试使用旧时间戳（应该失败：timestamp expired）
# Python 客户端会自动处理时间戳，手动测试需要修改客户端代码
```

---

### 3. 租户隔离测试

```bash
# 1. 注册两个账户
curl -X POST http://localhost:8081/api/v2/accounts/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user1@example.com", "company": "Company1", "password": "pass1"}'

curl -X POST http://localhost:8081/api/v2/accounts/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user2@example.com", "company": "Company2", "password": "pass2"}'

# 2. 分别创建 Token
python3 tests/hmac_client.py create_token "$AK1" "$SK1" "User1 token" '["*"]' 90
python3 tests/hmac_client.py create_token "$AK2" "$SK2" "User2 token" '["*"]' 90

# 3. 验证租户 1 只能看到自己的 Tokens
python3 tests/hmac_client.py list_tokens "$AK1" "$SK1"

# 4. 验证租户 2 只能看到自己的 Tokens
python3 tests/hmac_client.py list_tokens "$AK2" "$SK2"

# 5. 尝试用租户 1 的 AK/SK 访问租户 2 的 Token（应该失败）
python3 tests/hmac_client.py get_token "$AK1" "$SK1" "<user2_token_id>"
```

---

### 4. 错误场景测试

#### 无效签名

```bash
curl -X GET http://localhost:8081/api/v2/accounts/me \
  -H "Authorization: QINIU INVALID_AK:INVALID_SIGNATURE" \
  -H "X-Qiniu-Date: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"

# 期望：401 Unauthorized
```

#### 缺少时间戳

```bash
curl -X GET http://localhost:8081/api/v2/accounts/me \
  -H "Authorization: QINIU $AK:signature"

# 期望：401 Unauthorized - missing X-Qiniu-Date header
```

#### 过期 Token

```bash
# 创建一个 1 天过期的 Token
python3 tests/hmac_client.py create_token "$AK" "$SK" "Short-lived" '["*"]' 1

# 等待 2 天后验证（或手动修改数据库）
# 期望：Token has expired
```

#### 禁用的 Token

```bash
# 禁用 Token
python3 tests/hmac_client.py update_token_status "$AK" "$SK" "$TOKEN_ID" false

# 尝试验证
curl -X POST http://localhost:8081/api/v2/validate \
  -H "Authorization: Bearer $TOKEN"

# 期望：Token is inactive
```

---

## 📊 性能测试

### 使用 Apache Bench

```bash
# 安装
sudo apt-get install apache2-utils

# 测试验证端点（1000 请求，并发 10）
ab -n 1000 -c 10 \
  -H "Authorization: Bearer $TOKEN" \
  -p /dev/null \
  -T "application/json" \
  http://localhost:8081/api/v2/validate
```

### 使用 wrk

```bash
# 安装
sudo apt-get install wrk

# 测试健康检查端点
wrk -t4 -c100 -d30s http://localhost:8081/health

# 测试 Token 验证
wrk -t4 -c100 -d30s \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8081/api/v2/validate
```

---

## 🐛 调试技巧

### 1. 查看详细请求信息

```bash
# 使用 curl 的 -v 参数
curl -v -X POST http://localhost:8081/api/v2/validate \
  -H "Authorization: Bearer $TOKEN"
```

### 2. 检查 MongoDB 数据

```bash
# 进入 MongoDB
docker exec -it mongodb-test mongosh

# 切换数据库
use token_service_v2

# 查看账户
db.accounts.find().pretty()

# 查看 Tokens
db.tokens.find().pretty()

# 查看审计日志
db.audit_logs.find().sort({timestamp: -1}).limit(10).pretty()
```

### 3. 查看服务日志

服务启动时会输出详细的日志信息，包括：
- MongoDB 连接状态
- 索引创建状态
- 路由配置
- 请求处理日志

---

## 🐛 故障排查

### 测试失败？

1. **检查服务是否运行**
   ```bash
   curl http://localhost:8081/health
   ```

2. **检查 MongoDB**
   ```bash
   docker ps | grep mongodb
   ```

3. **查看服务日志**
   ```bash
   # 如果使用 quickstart.sh 启动
   tail -f /tmp/token-service-v2.log

   # 如果手动启动，查看终端输出
   ```

4. **验证 Python 依赖**
   ```bash
   pip3 install requests
   ```

### 常见错误

| 错误 | 原因 | 解决方案 |
|------|------|---------|
| `Connection refused` | 服务未启动 | 启动服务 |
| `401 Unauthorized` | 签名错误 | 检查 AK/SK |
| `timestamp expired` | 时间戳过期 | 检查系统时间 |
| `token not found` | Token 不存在 | 重新创建 Token |

---

## 🎯 测试检查清单

完成测试后，确认以下功能：

### 基础功能
- [ ] 账户注册成功
- [ ] 获取账户信息
- [ ] 创建 Token
- [ ] 列出 Tokens
- [ ] 获取 Token 详情
- [ ] 验证 Bearer Token
- [ ] 更新 Token 状态
- [ ] 删除 Token

### 安全功能
- [ ] HMAC 签名认证
- [ ] 时间戳防重放（15分钟窗口）
- [ ] SecretKey 加密存储
- [ ] 租户数据隔离
- [ ] Token 只在创建时显示完整值

### 权限控制
- [ ] Scope 精确匹配
- [ ] Scope 通配符匹配（`storage:*`）
- [ ] Scope 全局通配（`*`）
- [ ] 权限拒绝测试

### 边界测试
- [ ] 过期 Token 拒绝
- [ ] 禁用 Token 拒绝
- [ ] 跨租户访问拒绝
- [ ] 无效签名拒绝
- [ ] 缺少时间戳拒绝

---

## 📖 参考文档

- [API 文档](../API.md) - 完整的 API 参考
- [架构文档](../ARCHITECTURE.md) - 系统设计说明

---

**Happy Testing!** 🎉

---

## 🚦 限流功能测试

### 测试脚本说明

#### 1. test_rate_limit_improved.sh（完整测试 - 推荐）

**自动化完整测试**，验证所有三层限流功能。

**特点**：
- ✅ 自动启动服务（带限流配置）
- ✅ 测试应用层限流（5 req/min）
- ✅ 测试 Token 层限流（2 req/min）
- ✅ 测试账户层限流（3 req/min）
- ✅ 验证限流响应头
- ✅ 自动清理测试环境
- ⚠️ 需要等待限流窗口重置（约 3 分钟）

**运行方式**：
```bash
cd /root/src/auth/bearer-token-service.v2
./tests/test_rate_limit_improved.sh
```

**预期输出**：
```
✓✓✓ 应用层限流测试通过 - 成功触发限流！
✓✓✓ Token 层限流测试通过 - 成功触发限流！
✓✓✓ 账户层限流测试通过 - 成功触发限流！
╔════════════════════════════════════════╗
║  ✓✓✓ 三层限流功能测试全部通过！  ║
╚════════════════════════════════════════╝
```

---

#### 2. test_rate_limit_quick.sh（快速测试）

**快速验证**限流是否工作，无需等待窗口重置。

**特点**：
- ✅ 快速检查应用层限流
- ✅ 可选测试 Token 层限流
- ✅ 无需等待窗口重置
- ⚠️ 需要手动启动服务

**运行方式**：
```bash
# 1. 先启动服务（启用限流）
export MONGO_URI="mongodb://admin:123456@localhost:27017/token_service_v2?authSource=admin"
export MONGO_DATABASE="token_service_v2"
export PORT="8081"
export ENABLE_APP_RATE_LIMIT=true
export APP_RATE_LIMIT_PER_MINUTE=5
export ENABLE_TOKEN_RATE_LIMIT=true
./bearer-token-service

# 2. 在另一个终端运行测试
./tests/test_rate_limit_quick.sh
```

---

### 限流配置说明

#### 应用层限流（全局）
```bash
export ENABLE_APP_RATE_LIMIT=true
export APP_RATE_LIMIT_PER_MINUTE=5    # 每分钟 5 个请求
export APP_RATE_LIMIT_PER_HOUR=100    # 每小时 100 个请求
export APP_RATE_LIMIT_PER_DAY=1000    # 每天 1000 个请求
```

#### 账户层限流（单租户）
通过数据库配置：
```javascript
db.accounts.updateOne(
  { _id: "account_id" },
  {
    $set: {
      rate_limit: {
        requests_per_minute: 3,
        requests_per_hour: 50,
        requests_per_day: 500
      }
    }
  }
)
```

#### Token 层限流（单 Token）
创建 Token 时指定：
```json
{
  "description": "Test Token",
  "scope": ["storage:write"],
  "expires_in_seconds": 3600,
  "rate_limit": {
    "requests_per_minute": 2,
    "requests_per_hour": 30,
    "requests_per_day": 300
  }
}
```

---

### 限流响应示例

#### 成功响应（带限流头）
```http
HTTP/1.1 200 OK
X-RateLimit-Limit-App: 5
X-RateLimit-Remaining-App: 3
X-RateLimit-Reset-App: 1735992400
X-RateLimit-Limit-Token: 2
X-RateLimit-Remaining-Token: 1
X-RateLimit-Reset-Token: 1735992400
```

#### 限流触发
```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
X-RateLimit-Limit-App: 5
X-RateLimit-Remaining-App: 0
X-RateLimit-Reset-App: 1735992400
Retry-After: 45

{
  "error": "Application rate limit exceeded",
  "code": 429,
  "timestamp": "2026-01-04T10:30:00Z"
}
```

---

### 验收标准

#### ✅ 应用层限流
- 发送 10 个请求，前 5 个返回 200，后 5 个返回 429
- 响应头包含 `X-RateLimit-Limit-App`
- 响应头包含 `Retry-After`

#### ✅ Token 层限流
- 发送 5 个 Token 验证请求，前 2 个返回 200，后 3 个返回 429
- 响应头包含 `X-RateLimit-Limit-Token`
- 错误消息为 "Token rate limit exceeded"

#### ✅ 账户层限流
- 发送 6 个 HMAC 认证请求，前 3 个返回 200，后 3 个返回 429
- 响应头包含 `X-RateLimit-Limit-Account`
- 错误消息为 "Account rate limit exceeded"

---

### 故障排查

#### 1. 限流未触发
**原因**：
- 限流功能未启用（环境变量未设置）
- 限流阈值设置过高
- 滑动窗口还未累积足够的请求

**解决**：
```bash
# 检查环境变量
echo $ENABLE_APP_RATE_LIMIT

# 降低限流阈值
export APP_RATE_LIMIT_PER_MINUTE=3

# 连续快速发送请求
for i in {1..10}; do curl http://localhost:8081/health; done
```

#### 2. 服务启动失败
**原因**：
- MongoDB 未运行
- 端口被占用
- 编译失败

**解决**：
```bash
# 检查 MongoDB
mongosh mongodb://admin:123456@localhost:27017

# 检查端口
lsof -i :8081

# 重新编译
go build -o bearer-token-service ./cmd/server
```

---

更多信息请参考 `docs/RATE_LIMIT.md`

