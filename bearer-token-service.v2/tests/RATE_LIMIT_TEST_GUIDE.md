# 限流测试快速指南

## 🚀 快速开始

### 1. 确保 MongoDB 运行
```bash
# 检查 MongoDB 是否运行
docker ps | grep mongo

# 如果没有运行，启动 MongoDB
docker run -d \
  --name mongo-test \
  -p 27017:27017 \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=123456 \
  mongo:latest
```

### 2. 编译服务
```bash
cd /root/src/auth/bearer-token-service.v2
go build -o bearer-token-service ./cmd/server
```

### 3. 运行测试
```bash
./tests/test_rate_limit.sh
```

---

## ⚠️ 常见问题

### 问题 1: mongosh: command not found
**解决**：
```bash
./scripts/install_mongosh.sh
```

### 问题 2: email already registered
**解决**：测试脚本已自动清理数据库，如果还出现此问题，手动清理：
```bash
mongosh "mongodb://admin:123456@localhost:27017?authSource=admin" --eval "
  use token_service_v2_test;
  db.dropDatabase();
"
```

### 问题 3: 服务启动失败
**检查**：
```bash
# 检查端口是否被占用
lsof -i :8081

# 查看服务日志
tail -f /tmp/bearer-token-service-test.log
```

### 问题 4: MongoDB 连接失败
**检查**：
```bash
# 测试连接
mongosh "mongodb://admin:123456@localhost:27017?authSource=admin" --eval "db.version()"

# 检查 MongoDB 容器
docker logs mongo-test
```

---

## 📊 预期输出

```
=========================================
三层限流功能测试
=========================================

检查依赖...
✓ 所有依赖已安装

=========================================
1. 启动服务（启用三层限流）
=========================================
启动服务...
等待服务启动...
✓ 服务启动成功 (PID: 12345)

配置信息：
  应用层限流: 5 req/min
  账户层限流: 将设置为 3 req/min
  Token层限流: 将设置为 2 req/min

=========================================
1.5. 清理测试数据库（如果存在）
=========================================
Database cleaned
✓ 数据库已清理

=========================================
2. 注册测试账户
=========================================
{
  "account_id": "...",
  "email": "test@example.com",
  "company": "Test Company",
  "access_key": "AK_...",
  "secret_key": "SK_...",
  "created_at": "2026-01-04T..."
}

✓ 账户创建成功
  AccessKey: AK_...
  AccountID: ...

为账户添加限流配置...
Updated account rate limit
✓ 账户限流配置完成（3 req/min）

=========================================
3. 创建带限流的 Token
=========================================
{
  "token_id": "tk_...",
  "token": "sk-...",
  ...
}

✓ Token 创建成功
  Token ID: tk_...
  Token: sk-abc...
  限流配置: 2 req/min, 30 req/hour, 300 req/day

=========================================
4. 测试应用层限流（全局限流）
=========================================
限制: 5 req/min
测试: 发送 10 个请求，预期第 6 个开始触发限流

请求 1: 200 OK
请求 2: 200 OK
请求 3: 200 OK
请求 4: 200 OK
请求 5: 200 OK
请求 6: 429 Too Many Requests (应用层限流) ✓
请求 7: 429 Too Many Requests (应用层限流) ✓
请求 8: 429 Too Many Requests (应用层限流) ✓
请求 9: 429 Too Many Requests (应用层限流) ✓
请求 10: 429 Too Many Requests (应用层限流) ✓

统计:
  成功: 5
  限流: 5
✓✓✓ 应用层限流测试通过 - 成功触发限流！

等待 65 秒，让限流窗口重置...

=========================================
5. 测试 Token 层限流
=========================================
限制: 2 req/min
测试: 发送 5 个 Token 验证请求，预期第 3 个开始触发限流

请求 1: 200 OK - Token 验证成功
请求 2: 200 OK - Token 验证成功
请求 3: 429 Too Many Requests - Token rate limit exceeded ✓
请求 4: 429 Too Many Requests - Token rate limit exceeded ✓
请求 5: 429 Too Many Requests - Token rate limit exceeded ✓

统计:
  成功: 2
  限流: 3
✓✓✓ Token 层限流测试通过 - 成功触发限流！

...

╔════════════════════════════════════════╗
║  ✓✓✓ 三层限流功能测试全部通过！  ║
╚════════════════════════════════════════╝
```

---

## 🔧 手动测试

### 启动服务（启用限流）
```bash
export MONGO_URI="mongodb://admin:123456@localhost:27017/token_service_v2?authSource=admin"
export MONGO_DATABASE="token_service_v2"
export PORT="8081"
export ENABLE_APP_RATE_LIMIT=true
export APP_RATE_LIMIT_PER_MINUTE=5
export ENABLE_ACCOUNT_RATE_LIMIT=true
export ENABLE_TOKEN_RATE_LIMIT=true
./bearer-token-service
```

### 测试应用层限流
```bash
# 快速发送 10 个请求
for i in {1..10}; do
  echo "请求 $i:"
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8081/health
done
```

预期：前 5 个返回 200，后 5 个返回 429

---

## 📝 测试日志

测试日志保存在：`/tmp/bearer-token-service-test.log`

查看日志：
```bash
tail -f /tmp/bearer-token-service-test.log
```

---

## 🎯 验收标准

- ✅ 应用层限流：10 个请求 → 5 成功 + 5 限流
- ✅ Token 层限流：5 个请求 → 2 成功 + 3 限流
- ✅ 账户层限流：6 个请求 → 3 成功 + 3 限流
- ✅ 响应头包含 X-RateLimit-* 信息
- ✅ 响应头包含 Retry-After
- ✅ 错误消息清晰准确

---

更多信息请参考：
- `docs/RATE_LIMIT.md` - 完整限流文档
- `tests/README.md` - 测试套件说明
