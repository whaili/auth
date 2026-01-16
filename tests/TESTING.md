# Bearer Token Service V2 - 测试指南

## 🧪 测试概述

本项目提供完整的 API 测试套件，覆盖所有核心功能。

---

## 🚀 快速运行测试

### 使用 Makefile（推荐）

```bash
# 完整测试流程（启动服务 + 运行测试）
make test

# 只编译
make compile

# 停止测试服务
make test-stop
```

### 手动运行测试

```bash
# 1. 确保 MongoDB 运行
docker-compose up -d mongodb

# 2. 启动服务
bash tests/start_local.sh

# 3. 运行测试
bash tests/test_qstub_api.sh
```

---

## 📋 测试覆盖

### 1. 健康检查
- ✅ `/health` 端点

### 2. Token 管理（主账户）
- ✅ 创建 Token（QiniuStub `uid=xxx&ut=1`）
- ✅ 列出 Tokens
- ✅ 获取 Token 详情
- ✅ 更新 Token 状态（启用/禁用）
- ✅ 删除 Token

### 3. Token 管理（IAM 子账户）
- ✅ 创建 Token（QiniuStub `uid=xxx&ut=1&iuid=yyy`）
- ✅ 验证 IUID 字段正确返回

### 4. Token 验证
- ✅ Bearer Token 验证（主账户）
- ✅ Bearer Token 验证（IAM 子账户，包含 IUID）
- ✅ Scope 权限验证

### 5. 权限系统
- ✅ 获取权限列表

### 6. Redis 缓存
- ✅ Token 创建后不写入缓存
- ✅ 首次验证写入缓存
- ✅ 缓存命中加速响应
- ✅ Token 禁用后缓存失效
- ✅ Token 删除后缓存失效
- ✅ 空对象缓存防穿透
- ✅ 缓存 TTL 验证

---

## 🔧 测试脚本说明

### test_qstub_api.sh

完整的 QiniuStub API 测试脚本。

**环境变量**：
```bash
BASE_URL=http://localhost:8081  # 服务地址
QINIU_UID=1369077332            # 测试用 UID
QINIU_IUID=8901234              # 测试用 IUID
```

**测试流程**：
1. 健康检查
2. 创建 Token（主账户）
3. 创建 Token（IAM 子账户）
4. 列出 Tokens
5. 获取 Token 详情
6. 验证 Bearer Token（主账户）
7. 验证 Bearer Token（IAM 子账户 + IUID）
8. 更新 Token 状态
9. 获取权限列表
10. 删除 Tokens

### test_redis_cache.sh

Redis 缓存功能测试脚本。

**前置条件**：
- 服务启动时需设置 `REDIS_ENABLED=true`
- Redis 容器运行中（`bearer-token-redis`）

**环境变量**：
```bash
BASE_URL=http://localhost:8081  # 服务地址
REDIS_HOST=localhost            # Redis 地址
REDIS_PORT=6379                 # Redis 端口
```

**测试流程**：
1. 前置检查（服务、Redis 可用性）
2. 创建 Token（验证不写入缓存）
3. 首次验证（验证写入缓存）
4. 缓存命中性能测试
5. 禁用 Token（验证缓存失效）
6. 重新启用 Token（验证缓存更新）
7. 删除 Token（验证缓存失效）
8. 空对象缓存测试（防穿透）
9. 缓存 TTL 测试

**运行方式**：
```bash
# 确保服务启用 Redis 缓存
REDIS_ENABLED=true PORT=8081 ./bin/server &

# 运行缓存测试
BASE_URL=http://localhost:8081 bash tests/test_redis_cache.sh
```

---

## 📊 测试结果示例

```bash
$ make test

========================================
准备测试环境...
========================================

✅ MongoDB 容器运行中
✅ 二进制文件存在
✅ 服务运行中 (http://localhost:8081)

========================================
开始运行测试...
========================================

========================================
0. Health Check
========================================
ℹ️  Testing health check endpoint...
✅ Health check passed: {"status":"ok"}

========================================
1. Create Token (Main Account)
========================================
ℹ️  Creating token with main account (uid=1369077332)...
✅ Token created for main account
ℹ️  Token ID: tk_xxx
ℹ️  Bearer Token: sk-xxx...

... (更多测试输出)

========================================
🎉 All Tests Passed!
  - Main Account (UID) ✓
  - IAM Sub-Account (UID + IUID) ✓
========================================
```

---

## 🐛 调试测试

### 查看服务日志

```bash
# 查看最后 30 行日志
tail -30 tokenserv_test.log

# 实时查看日志
tail -f tokenserv_test.log
```

### 手动测试单个接口

```bash
# 创建 Token（主账户）
curl -X POST "http://localhost:8081/api/v2/tokens" \
  -H "Authorization: QiniuStub uid=1369077332&ut=1" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Test token",
    "scope": ["storage:*"],
    "expires_in_seconds": 3600
  }'

# 创建 Token（IAM 子账户）
curl -X POST "http://localhost:8081/api/v2/tokens" \
  -H "Authorization: QiniuStub uid=1369077332&ut=1&iuid=8901234" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "IAM token",
    "scope": ["storage:read"],
    "expires_in_seconds": 3600
  }'

# 验证 Bearer Token
curl -X POST "http://localhost:8081/api/v2/validate" \
  -H "Authorization: Bearer sk-xxx..." \
  -H "Content-Type: application/json" \
  -d '{
    "required_scope": "storage:read"
  }'
```

---

## 🔍 常见问题

### 测试失败：服务未启动

**问题**：`❌ 服务启动失败`

**解决方案**：
```bash
# 检查 MongoDB 是否运行
docker ps | grep mongo

# 查看服务日志
tail -50 tokenserv_test.log

# 手动启动服务测试
bash tests/start_local.sh
```

### 测试失败：MongoDB 连接错误

**问题**：`(Unauthorized) Command insert requires authentication`

**解决方案**：
```bash
# 确保 MongoDB 认证信息正确
export MONGO_URI="mongodb://admin:123456@localhost:27017"

# 或修改 tests/start_local.sh 中的 MONGO_URI
```

### Token 创建失败

**问题**：`invalid qstub token`

**解决方案**：
- 检查 Authorization 头格式
- 确保 UID 参数存在
- 验证 QiniuStub 格式：`uid=xxx&ut=1`

---

## 📝 添加新测试

### 测试脚本结构

```bash
# 测试函数命名规范
test_<功能名称>() {
    log_info "Testing <功能描述>..."

    # 执行 API 调用
    local response=$(curl -s ...)

    # 验证响应
    if [[ 验证条件 ]]; then
        log_success "测试通过"
    else
        log_error "测试失败"
        exit 1
    fi
}

# 在 main() 中添加测试步骤
test_step "X. <测试名称>"
test_<功能名称>
```

---

## 🎯 持续集成

测试脚本可以集成到 CI/CD 流程：

```yaml
# GitHub Actions 示例
- name: Run Tests
  run: |
    make test
```

---

**最后更新**: 2026-01-14
