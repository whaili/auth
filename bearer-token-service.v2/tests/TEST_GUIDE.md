# 测试指南

> Bearer Token Service V2 完整测试流程

---

## 🚀 快速测试

### 1. 启动服务

```bash
# 终端 1：启动 MongoDB
docker run -d -p 27017:27017 --name mongodb-test mongo:latest

# 终端 2：启动服务
cd /root/src/auth/bearer-token-service.v1/v2
go run cmd/server/main.go
```

### 2. 运行自动化测试

```bash
# 终端 3：运行测试脚本
cd /root/src/auth/bearer-token-service.v1/v2/tests

# 添加执行权限
chmod +x test_api.sh
chmod +x hmac_client.py

# 运行完整测试
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

## 🔧 手动测试

### 测试 1：注册账户

```bash
curl -X POST http://localhost:8080/api/v2/accounts/register \
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

### 测试 2：创建 Token（使用 Python 客户端）

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
curl -X POST http://localhost:8080/api/v2/validate \
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

## 🧪 Scope 权限测试

### 测试不同的 Scope 组合

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

### 验证权限

```bash
# 测试 storage:read 权限（应该成功）
curl -X POST http://localhost:8080/api/v2/validate \
  -H "Authorization: Bearer <read-only-token>" \
  -d '{"required_scope": "storage:read"}'

# 测试 storage:write 权限（应该失败）
curl -X POST http://localhost:8080/api/v2/validate \
  -H "Authorization: Bearer <read-only-token>" \
  -d '{"required_scope": "storage:write"}'
```

---

## 🔐 HMAC 签名测试

### 测试防重放攻击

```bash
# 1. 创建一个有效请求
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
echo "Current timestamp: $TIMESTAMP"

# 2. 使用旧时间戳（20 分钟前）
OLD_TIMESTAMP=$(date -u -d '20 minutes ago' +"%Y-%m-%dT%H:%M:%SZ")
echo "Old timestamp: $OLD_TIMESTAMP"

# 3. 尝试使用旧时间戳（应该失败）
# 需要手动计算签名，或修改 Python 客户端
```

---

## 📊 租户隔离测试

### 验证租户隔离

```bash
# 1. 注册两个账户
curl -X POST http://localhost:8080/api/v2/accounts/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user1@example.com", "company": "Company1", "password": "pass1"}'

curl -X POST http://localhost:8080/api/v2/accounts/register \
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

## 🚨 错误场景测试

### 1. 无效签名

```bash
curl -X GET http://localhost:8080/api/v2/accounts/me \
  -H "Authorization: QINIU INVALID_AK:INVALID_SIGNATURE" \
  -H "X-Qiniu-Date: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"

# 期望：401 Unauthorized
```

### 2. 缺少时间戳

```bash
curl -X GET http://localhost:8080/api/v2/accounts/me \
  -H "Authorization: QINIU $AK:signature"

# 期望：401 Unauthorized - missing X-Qiniu-Date header
```

### 3. 过期 Token

```bash
# 创建一个 1 天过期的 Token
python3 tests/hmac_client.py create_token "$AK" "$SK" "Short-lived" '["*"]' 1

# 等待 2 天后验证（或手动修改数据库）
# 期望：Token has expired
```

### 4. 禁用的 Token

```bash
# 禁用 Token
python3 tests/hmac_client.py update_token_status "$AK" "$SK" "$TOKEN_ID" false

# 尝试验证
curl -X POST http://localhost:8080/api/v2/validate \
  -H "Authorization: Bearer $TOKEN"

# 期望：Token is inactive
```

---

## 📈 性能测试

### 使用 Apache Bench

```bash
# 安装 ab
sudo apt-get install apache2-utils

# 测试 Token 验证端点（1000 请求，并发 10）
ab -n 1000 -c 10 \
  -H "Authorization: Bearer $TOKEN" \
  -p /dev/null \
  -T "application/json" \
  http://localhost:8080/api/v2/validate
```

### 使用 wrk

```bash
# 安装 wrk
sudo apt-get install wrk

# 测试健康检查端点
wrk -t4 -c100 -d30s http://localhost:8080/health

# 测试 Token 验证
wrk -t4 -c100 -d30s \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v2/validate
```

---

## 🐛 调试技巧

### 1. 查看详细请求信息

```bash
# 使用 curl 的 -v 参数
curl -v -X POST http://localhost:8080/api/v2/validate \
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

## ✅ 测试检查清单

### 基础功能
- [ ] 账户注册成功
- [ ] 获取账户信息
- [ ] 创建 Token
- [ ] 列出 Tokens
- [ ] 验证 Bearer Token
- [ ] 删除 Token

### 安全功能
- [ ] HMAC 签名验证
- [ ] 时间戳防重放
- [ ] 租户隔离（不能访问其他租户的 Token）
- [ ] SecretKey 加密存储
- [ ] Token 只在创建时显示完整值

### 权限控制
- [ ] 精确 Scope 匹配
- [ ] 通配符 Scope 匹配
- [ ] 权限拒绝测试

### 边界情况
- [ ] 无效签名拒绝
- [ ] 过期 Token 拒绝
- [ ] 禁用 Token 拒绝
- [ ] 缺少时间戳拒绝
- [ ] 跨租户访问拒绝

---

## 📞 遇到问题？

1. 检查 MongoDB 是否运行：`docker ps | grep mongodb`
2. 检查服务是否启动：`curl http://localhost:8080/health`
3. 查看服务日志：检查终端输出
4. 验证环境变量：`echo $MONGO_URI`

---

**测试愉快！** 🎉
