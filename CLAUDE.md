# Bearer Token Service V2 - AI Context Guide

> 本文档用于帮助 Claude Code 快速理解项目上下文，保持代码修改与架构一致性。

## 项目定位

**多租户 Token 认证服务**，**不负责账户管理**，账户管理由外部系统（七牛）提供。

### 核心特性
- 🔐 **QiniuStub 认证**：使用七牛内部用户系统认证
- 🎯 **UID + IUID 支持**：支持主账户（UID）和 IAM 子账户（UID + IUID）
- 🎫 **Bearer Token 管理**：创建、列表、更新、删除 Token
- ⏱️ **秒级过期**：Token 支持秒级精度的过期时间
- 📊 **审计合规**：完整的操作日志记录
- 🚦 **限流功能**：三层限流（应用/账户/Token）

---

## ⚠️ 重要变更（相比早期版本）

1. **移除账户管理功能**：
   - ❌ 不再支持账户注册（`/api/v2/accounts/register`）
   - ❌ 不再支持账户信息查询（`/api/v2/accounts/me`）
   - ❌ 不再支持 SecretKey 重新生成（`/api/v2/accounts/regenerate-sk`）
   - ✅ 账户管理由外部系统负责

2. **移除 HMAC 签名认证**：
   - ❌ 不再支持 HMAC-SHA256 签名认证
   - ✅ 只使用 QiniuStub 认证（`Authorization: QiniuStub uid=xxx&ut=x`）

3. **移除 Scope 权限控制**：
   - ❌ 不再支持 Scope 权限检查
   - ❌ 不再支持 `/api/v2/permissions` 端点
   - ✅ Token 验证仅检查有效性和过期时间

4. **认证方式简化**：
   - 所有 Token 管理 API 使用 QiniuStub 认证
   - Token 验证 API 使用 Bearer Token 认证

---

## 架构设计

### 三层架构
```
┌─────────────────────────────────────────────────────────┐
│  Handler 层 (HTTP API)                                  │
│  handlers/token_handler.go                              │
│  handlers/validation_handler.go                         │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│  Service 层 (业务逻辑)                                   │
│  service/token_service.go                               │
│  service/validation_service.go                          │
│  service/audit_service.go                               │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│  Repository 层 (数据访问)                                │
│  repository/mongo_account_repo.go (仅用于 UID 映射)      │
│  repository/mongo_token_repo.go                         │
│  repository/mongo_audit_repo.go                         │
└─────────────────────────────────────────────────────────┘
                         ↓
                    MongoDB
```

### 认证模块
```
auth/
├── qstub_middleware.go        # QiniuStub 认证中间件
├── qiniu_uid_mapper.go        # 七牛 UID 映射
└── context.go                 # Context 辅助函数
```

### 限流模块
```
ratelimit/
├── limiter.go                 # 限流器核心实现（滑动窗口算法）
├── middleware.go              # 三层限流中间件
config/
└── ratelimit.go               # 限流配置管理
```

---

## 核心数据模型

### Token（Bearer Token）
```go
Token {
    ID            string     // tk_xxx 格式
    AccountID     string     // 关联账户（qiniu_{uid} 格式）
    Token         string     // sk-xxx 前缀
    Description   string     // Token 描述
    RateLimit     *RateLimit // 频率限制配置
    IUID          string     // IAM 子账户 ID（可选）
    ExpiresAt     *time.Time // 过期时间（nil=永不过期，支持秒级精度）
    IsActive      bool       // 启用状态
    TotalRequests int64      // 使用次数统计
    LastUsedAt    *time.Time // 最后使用时间
    CreatedAt     time.Time
}
```

### AuditLog（审计日志）
```go
AuditLog {
    ID          string      // 日志 ID
    AccountID   string      // 操作者账户
    Action      string      // create_token/delete_token/update_token 等
    ResourceID  string      // 操作对象 ID
    IP          string      // 请求 IP
    UserAgent   string      // User-Agent
    RequestData interface{} // 请求参数
    Result      string      // success/failure
    Timestamp   time.Time   // 操作时间
}
```

---

## API 设计

### 认证方式

#### QiniuStub 认证（所有 Token 管理 API）

**主账户格式**：
```http
Authorization: QiniuStub uid=1369077332&ut=1
```

**IAM 子账户格式**：
```http
Authorization: QiniuStub uid=1369077332&ut=1&iuid=8901234
```

#### Bearer Token 认证（Token 验证 API）
```http
Authorization: Bearer sk-abc123def456...
```

---

### 核心 API 端点

| 端点 | 方法 | 认证 | 功能 | 文件位置 |
|------|------|------|------|----------|
| `/health` | GET | ❌ | 健康检查 | cmd/server/main.go |
| **Token 管理** |
| `/api/v2/tokens` | POST | QiniuStub | 创建 Token | handlers/token_handler.go |
| `/api/v2/tokens` | GET | QiniuStub | 列出所有 Token | handlers/token_handler.go |
| `/api/v2/tokens/{id}` | GET | QiniuStub | 获取 Token 详情 | handlers/token_handler.go |
| `/api/v2/tokens/{id}/status` | PUT | QiniuStub | 更新 Token 状态 | handlers/token_handler.go |
| `/api/v2/tokens/{id}` | DELETE | QiniuStub | 删除 Token | handlers/token_handler.go |
| `/api/v2/tokens/{id}/stats` | GET | QiniuStub | 获取 Token 统计 | handlers/token_handler.go |
| **Token 验证** |
| `/api/v2/validate` | POST | Bearer | 验证 Token（核心接口） | handlers/validation_handler.go |

---

## 关键业务流程

### 1. Token 创建流程（主账户）
```
POST /api/v2/tokens (QiniuStub uid=xxx&ut=1)
  ↓
QstubAuthMiddleware 验证
  ├─ 解析 Authorization 头
  ├─ 提取 uid
  ├─ 映射为 account_id（qiniu_{uid}）
  └─ 注入到 Context
  ↓
TokenHandler.CreateToken()
  ↓
TokenService.CreateToken()
  ├─ 生成 Token ID（tk_ + 随机字符串）
  ├─ 生成 Token 值（sk- + 随机字符串）
  ├─ 计算过期时间（当前时间 + expires_in_seconds 秒）
  ├─ 保存到 MongoDB（关联 account_id）
  └─ 记录审计日志
  ↓
返回完整 Token（明文，仅此一次！）
```

### 2. Token 创建流程（IAM 子账户）
```
POST /api/v2/tokens (QiniuStub uid=xxx&ut=1&iuid=yyy)
  ↓
QstubAuthMiddleware 验证
  ├─ 解析 Authorization 头
  ├─ 提取 uid 和 iuid
  ├─ 映射为 account_id（qiniu_{uid}）
  ├─ 保存 iuid 到 Context
  └─ 注入到 Context
  ↓
TokenHandler.CreateToken()
  ↓
（与主账户流程相同，但 Token 会关联 iuid）
```

### 3. Token 验证流程
```
POST /api/v2/validate (Bearer sk-xxx)
  ↓
ValidationHandler.ValidateToken()
  ↓
ValidationService.ValidateToken()
  ├─ TokenRepository.GetByTokenValue()  # 查询 Token
  ├─ 检查 IsActive 状态
  ├─ 检查过期时间（ExpiresAt）
  └─ 异步更新使用统计（IncrementUsage）
  ↓
返回验证结果 + Token 信息（包含 uid 和 iuid）
```

---

## 配置管理

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `MONGO_URI` | `mongodb://admin:123456@localhost:27017` | MongoDB 连接 |
| `MONGO_DATABASE` | `token_service_v2` | 数据库名 |
| `PORT` | `8080` | 监听端口 |
| `QINIU_UID_MAPPER_MODE` | `simple` | UID 映射方式（simple/database） |
| `QINIU_UID_AUTO_CREATE` | `false` | 自动创建账户映射 |
| `SKIP_INDEX_CREATION` | `false` | 跳过索引创建 |
| **限流配置** |
| `ENABLE_APP_RATE_LIMIT` | `false` | 应用层限流开关 |
| `ENABLE_ACCOUNT_RATE_LIMIT` | `false` | 账户层限流开关 |
| `ENABLE_TOKEN_RATE_LIMIT` | `false` | Token层限流开关 |

---

## 开发指南

### UID 映射机制

**SimpleQiniuUIDMapper（默认）**：
- 将七牛 UID 直接转换为 `qiniu_{uid}` 格式
- 不需要数据库查询
- 适用于大部分场景

**DatabaseQiniuUIDMapper**：
- 在 accounts 集合中查询或创建 UID 映射
- 支持自动创建账户
- 适用于需要额外账户信息的场景

### IAM 子账户处理

1. **创建 Token 时**：
   - QiniuStub 头包含 `iuid` 参数
   - Token 记录关联的 `iuid`
   - 审计日志记录 `iuid` 信息

2. **验证 Token 时**：
   - 验证响应中返回 `uid` 和 `iuid`（如果有）
   - 调用方可根据 `iuid` 识别子账户操作

---

## 测试

### 运行测试
```bash
make test          # 运行完整测试
make compile       # 编译服务
make clean         # 清理编译产物
```

### 测试脚本
```bash
tests/test_qstub_api.sh    # QiniuStub API 测试
```

---

## 关键文件索引

### 启动入口
- `cmd/server/main.go` - 服务启动、依赖注入、路由配置

### 核心业务
- `service/validation_service.go:ValidateToken()` - Token 验证核心逻辑
- `service/token_service.go:CreateToken()` - Token 创建逻辑

### 认证安全
- `auth/qstub_middleware.go:Authenticate()` - QiniuStub 认证中间件
- `auth/qiniu_uid_mapper.go` - UID 映射实现

### 数据模型
- `interfaces/models.go` - 所有数据结构定义
- `interfaces/repository.go` - Repository 接口定义

---

## 快速命令

```bash
# 本地开发
bash tests/start_local.sh     # 启动服务
make test                     # 运行测试

# 编译
make build                    # 编译二进制
make package                  # 打包部署文件

# 测试
curl http://localhost:8081/health  # 健康检查

# MongoDB 操作
docker exec -it bearer-token-service-mongo-1 mongosh
> use token_service_v2
> db.tokens.find().pretty()
```

---

**最后更新**: 2026-01-12
**版本**: V2（简化版 - 移除账户管理、HMAC 认证和 Scope 权限控制）
