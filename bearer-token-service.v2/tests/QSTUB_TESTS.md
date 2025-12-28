# Qstub 认证测试说明

## 概述

`test_api.sh` 脚本现在包含完整的 **Qstub 认证方式** 测试用例，与原有的 HMAC 认证测试一起，全面覆盖 Bearer Token Service V2 的双认证模式。

## 新增测试用例

### 12. Qstub 认证 - 创建账户 (test_qstub_create_account)
**功能**：构建 Qstub 认证上下文
- 创建测试用户信息：UID=12345, Email=qstub-test@qiniu.com
- 生成 Qstub Token（Base64 编码的 JSON）
- 导出全局变量供后续测试使用

**Qstub Token 格式**：
```bash
# 原始 JSON
{"uid":"12345","email":"qstub-test@qiniu.com","name":"Qstub Test User"}

# Base64 编码后
eyJ1aWQiOiIxMjM0NSIsImVtYWlsIjoicXN0dWItdGVzdEBxaW5pdS5jb20iLCJuYW1lIjoiUXN0dWIgVGVzdCBVc2VyIn0=
```

### 13. Qstub 认证 - 创建 Token (test_qstub_create_token)
**功能**：使用 Qstub 认证创建 Bearer Token
- HTTP 请求头：`Authorization: Bearer {QSTUB_TOKEN}`
- 创建的 Token 具有 `storage:read` 和 `storage:write` 权限
- 验证响应中的 `account_id` 格式

**期望结果**：
- HTTP 状态码：201
- 返回 token_id, token, account_id
- account_id 格式：`qiniu_{uid}` (例如: qiniu_12345)

### 14. Qstub 认证 - 列出 Tokens (test_qstub_list_tokens)
**功能**：使用 Qstub 认证列出所有 Token
- 测试 GET `/api/v2/tokens?limit=10`
- 验证分页和 total 字段

### 15. Qstub 认证 - 获取 Token 详情 (test_qstub_get_token)
**功能**：使用 Qstub 认证获取单个 Token 的详细信息
- 测试 GET `/api/v2/tokens/{token_id}`
- 验证返回的 Token 详情与创建时一致

### 16. Qstub 认证 - 更新 Token (test_qstub_update_token)
**功能**：使用 Qstub 认证更新 Token 状态
- 禁用 Token：PUT `/api/v2/tokens/{token_id}/status` with `{"enabled": false}`
- 重新启用 Token：PUT with `{"enabled": true}`

### 17. Qstub 认证 - 删除 Token (test_qstub_delete_token)
**功能**：使用 Qstub 认证删除 Token
- 测试 DELETE `/api/v2/tokens/{token_id}`
- 验证删除成功

### 18. 验证 Account ID 映射 (test_qstub_account_mapping)
**功能**：验证 Qstub 认证的 Account ID 映射逻辑
- 检查 account_id 是否为 `qiniu_{uid}` 格式
- 测试生成的 Bearer Token 是否可用于 `/api/v2/validate` 端点

## 运行测试

### 前置条件
1. 服务已启动：`./bin/tokenserv` (默认端口 8080)
2. MongoDB 已运行
3. Python 3 已安装（用于 HMAC 测试部分）
4. 安装依赖：`pip3 install requests`

### 执行测试
```bash
cd bearer-token-service.v2/tests
chmod +x test_api.sh
./test_api.sh
```

### 环境变量（可选）
```bash
# 自定义服务地址
BASE_URL=http://localhost:9000 ./test_api.sh
```

## 测试覆盖范围

### HMAC 认证测试 (步骤 1-11)
- ✅ 账户注册
- ✅ 获取账户信息（HMAC 签名）
- ✅ 创建 Token（多种 Scope）
- ✅ 列出、查询、更新、删除 Token
- ✅ 重新生成 Secret Key

### Qstub 认证测试 (步骤 12-18)
- ✅ Qstub Token 构建
- ✅ 使用 Qstub 创建 Bearer Token
- ✅ 使用 Qstub 管理 Token（列出、查询、更新、删除）
- ✅ Account ID 映射验证
- ✅ Bearer Token 验证

## Qstub 认证原理

### 认证流程
```
1. 客户端生成 Qstub Token
   JSON: {"uid":"12345","email":"user@qiniu.com","name":"User Name"}
   ↓
   Base64 编码
   ↓
   QSTUB_TOKEN

2. 请求 API
   Authorization: Bearer {QSTUB_TOKEN}
   ↓
   服务端解析 Base64
   ↓
   提取 UID

3. UID 映射为 Account ID
   - 模式 1 (simple): account_id = "qiniu_{uid}"
   - 模式 2 (database): 查询数据库映射关系
   ↓
   认证成功
```

### 与 HMAC 认证的区别

| 特性 | HMAC 认证 | Qstub 认证 |
|------|----------|-----------|
| **认证头** | `Authorization: QINIU {ak}:{sig}` | `Authorization: Bearer {base64}` |
| **需要注册** | ✅ 需要先调用 `/register` | ❌ 无需注册，直接使用 |
| **Account ID** | 自动生成 UUID | `qiniu_{uid}` 映射 |
| **签名计算** | ✅ HMAC-SHA256 签名 | ❌ 无签名，仅 Base64 |
| **适用场景** | 外部客户端，公网 API | 七牛内部服务，内网调用 |
| **安全性** | 高（密钥签名） | 低（仅编码，需内网保护） |

## 输出示例

### 成功输出
```
========================================
12. Qstub Authentication - Create Account
========================================
ℹ️  Creating Qstub authentication context...
✅ Qstub authentication context created
ℹ️  Qstub UID: 12345
ℹ️  Qstub Email: qstub-test@qiniu.com

========================================
13. Qstub Authentication - Create Token
========================================
ℹ️  Creating Bearer Token using Qstub authentication...
✅ Token created via Qstub authentication
ℹ️  Token ID: tok_abc123...
ℹ️  Account ID: qiniu_12345

...

========================================
18. Verify Qstub Account ID Mapping
========================================
✅ Account ID mapping is correct
ℹ️  Expected: qiniu_12345
ℹ️  Actual:   qiniu_12345
✅ Bearer Token validation passed

========================================
🎉 All Tests Passed!
  - HMAC Authentication ✓
  - Qstub Authentication ✓
========================================
```

## 故障排查

### 常见错误

**1. Qstub Token 解析失败**
```
❌ Failed to create token via Qstub with status 401
{"error":"invalid qstub token: invalid character..."}
```
- 原因：Base64 编码错误或 JSON 格式错误
- 解决：使用 `echo -n` 避免换行符，确保 JSON 格式正确

**2. Account ID 映射错误**
```
❌ Account ID mapping mismatch!
ℹ️  Expected: qiniu_12345
ℹ️  Actual:   some_other_id
```
- 原因：服务端 `QINIU_UID_MAPPER_MODE` 配置错误
- 解决：检查环境变量，确保为 `simple` 或正确配置数据库模式

**3. Bearer Token 验证失败**
```
❌ Bearer Token validation failed
{"error":"invalid token"}
```
- 原因：Token 已过期或被禁用
- 解决：检查 Token 的 `expires_at` 和 `enabled` 状态

## 相关文件

- **测试脚本**：`tests/test_api.sh`
- **Qstub 中间件**：`auth/unified_middleware.go`
- **UID 映射器**：`auth/qiniu_uid_mapper.go`
- **配置文档**：`CONFIG.md`
- **架构文档**：`DUAL_AUTH_GUIDE.md`

## 配置选项

### Qstub 相关环境变量

```bash
# UID 映射模式
QINIU_UID_MAPPER_MODE=simple        # 简单模式：qiniu_{uid}
QINIU_UID_MAPPER_MODE=database      # 数据库模式：查询映射表

# 自动创建账户（仅 database 模式）
QINIU_UID_AUTO_CREATE=true          # 首次访问时自动创建
QINIU_UID_AUTO_CREATE=false         # 必须预先存在映射关系
```

## 总结

通过添加 Qstub 认证测试，`test_api.sh` 现在提供：
- ✅ **完整的双认证测试**：HMAC + Qstub
- ✅ **端到端验证**：从认证到 Token 管理全流程
- ✅ **Account ID 映射测试**：确保 UID 正确转换
- ✅ **自动化测试**：一键运行，快速验证功能

这使得开发和部署过程中可以快速验证双认证模式是否正常工作。
