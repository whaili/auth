# 测试套件使用说明

> Bearer Token Service V2 完整测试工具集

---

## 📦 测试文件清单

| 文件 | 说明 | 用途 |
|------|------|------|
| `test_api.sh` | 自动化测试脚本 | 测试所有 API 端点 |
| `hmac_client.py` | HMAC 签名客户端 | Python 客户端库 + CLI 工具 |
| `TEST_GUIDE.md` | 详细测试指南 | 手动测试教程 |
| `README.md` | 本文件 | 快速参考 |

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

## 🧪 使用 Bash 测试脚本

### 完整测试

```bash
cd /root/src/auth/bearer-token-service.v1/v2/tests
./test_api.sh
```

### 自定义配置

```bash
# 指定服务地址
BASE_URL=http://localhost:8080 ./test_api.sh

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
    base_url="http://localhost:8080"
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

## 📋 测试场景

### 基础功能测试

```bash
# 1. 注册账户
curl -X POST http://localhost:8080/api/v2/accounts/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","company":"Test","password":"pass123"}'

# 2. 创建 Token
python3 hmac_client.py create_token "$AK" "$SK" "Test" '["*"]' 90

# 3. 验证 Token
curl -X POST http://localhost:8080/api/v2/validate \
  -H "Authorization: Bearer $TOKEN"
```

### Scope 权限测试

```bash
# 创建只读 Token
python3 hmac_client.py create_token "$AK" "$SK" \
  "Read-only" '["storage:read"]' 90

# 测试有权限的操作（应该成功）
curl -X POST http://localhost:8080/api/v2/validate \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"required_scope":"storage:read"}'

# 测试无权限的操作（应该失败）
curl -X POST http://localhost:8080/api/v2/validate \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"required_scope":"storage:write"}'
```

### 租户隔离测试

```bash
# 注册两个账户
# 账户 1
curl -X POST http://localhost:8080/api/v2/accounts/register \
  -d '{"email":"user1@test.com","company":"C1","password":"p1"}' \
  -H "Content-Type: application/json"
# 保存 AK1, SK1

# 账户 2
curl -X POST http://localhost:8080/api/v2/accounts/register \
  -d '{"email":"user2@test.com","company":"C2","password":"p2"}' \
  -H "Content-Type: application/json"
# 保存 AK2, SK2

# 验证租户 1 只能看到自己的 Tokens
python3 hmac_client.py list_tokens "$AK1" "$SK1"

# 验证租户 2 只能看到自己的 Tokens
python3 hmac_client.py list_tokens "$AK2" "$SK2"
```

---

## 🐛 故障排查

### 测试失败？

1. **检查服务是否运行**
   ```bash
   curl http://localhost:8080/health
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

## 📊 性能测试

### 使用 Apache Bench

```bash
# 安装
sudo apt-get install apache2-utils

# 测试验证端点
ab -n 1000 -c 10 \
  -H "Authorization: Bearer $TOKEN" \
  -p /dev/null \
  http://localhost:8080/api/v2/validate
```

### 使用 wrk

```bash
# 安装
sudo apt-get install wrk

# 测试
wrk -t4 -c100 -d30s \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v2/validate
```

---

## 📖 参考文档

- [API 文档](../API.md) - 完整的 API 参考
- [测试指南](./TEST_GUIDE.md) - 详细的测试教程
- [架构文档](../ARCHITECTURE.md) - 系统设计说明

---

## 🎯 测试检查清单

完成测试后，确认以下功能：

**基础功能**
- [ ] 账户注册
- [ ] 创建 Token
- [ ] 列出 Tokens
- [ ] 验证 Token
- [ ] 删除 Token

**安全功能**
- [ ] HMAC 签名认证
- [ ] 时间戳防重放（15分钟窗口）
- [ ] SecretKey 加密存储
- [ ] 租户数据隔离

**权限控制**
- [ ] Scope 精确匹配
- [ ] Scope 通配符匹配（`storage:*`）
- [ ] Scope 全局通配（`*`）
- [ ] 权限拒绝测试

**边界测试**
- [ ] 过期 Token 拒绝
- [ ] 禁用 Token 拒绝
- [ ] 跨租户访问拒绝
- [ ] 无效签名拒绝

---

**Happy Testing!** 🎉
