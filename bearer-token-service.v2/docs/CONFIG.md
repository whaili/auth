# 配置说明

Bearer Token Service V2 支持通过环境变量进行灵活配置，方便在不同环境下切换数据源和认证策略。

## 📋 配置项总览

| 环境变量 | 说明 | 可选值 | 默认值 | 必填 |
|---------|------|--------|--------|------|
| `MONGO_URI` | MongoDB 连接地址 | 连接字符串 | `mongodb://localhost:27017` | 否 |
| `PORT` | 服务监听端口 | 端口号 | `8080` | 否 |
| `ACCOUNT_FETCHER_MODE` | 账户查询方式 | `local` / `external` | `local` | 否 |
| `EXTERNAL_ACCOUNT_API_URL` | 外部账户 API 地址 | URL | - | 当 MODE=external 时必填 |
| `EXTERNAL_ACCOUNT_API_TOKEN` | 外部 API 认证 Token | Token 字符串 | - | 否 |
| `QINIU_UID_MAPPER_MODE` | UID 映射方式 | `simple` / `database` | `simple` | 否 |
| `QINIU_UID_AUTO_CREATE` | 自动创建账户 | `true` / `false` | `false` | 否 |
| `HMAC_TIMESTAMP_TOLERANCE` | 时间戳容忍度 | Duration (如 `15m`) | `15m` | 否 |
| `SKIP_INDEX_CREATION` | 跳过索引创建 | `true` / `false` | `false` | 否 |

---

## 🔧 配置详解

### 0. MongoDB 连接配置

#### 本地 MongoDB

```bash
# 单机模式
export MONGO_URI=mongodb://localhost:27017

# 带认证
export MONGO_URI=mongodb://user:password@localhost:27017/dbname?authSource=admin
```

#### 外部 MongoDB 副本集（生产环境）

```bash
# 副本集连接字符串格式
mongodb://用户名:密码@主节点:端口,从节点1:端口,从节点2:端口/数据库名?replicaSet=副本集名&authSource=admin&其他参数
```

**重要参数说明**:

| 参数 | 必填 | 说明 | 推荐值 |
|------|------|------|--------|
| `/数据库名` | ✅ | 数据库名称，**必须指定** | `/bearer_token_service` |
| `replicaSet` | ✅ | 副本集名称 | `rs0` (根据实际配置) |
| `authSource` | ✅ | 认证数据库 | `admin` |
| `readPreference` | ❌ | 读偏好 | `primaryPreferred` |
| `retryWrites` | ❌ | 自动重试写入 | `true` |
| `w` | ❌ | 写确认级别 | `majority` |
| `maxPoolSize` | ❌ | 连接池大小 | `50` |

**完整示例（1主2备）**:

```bash
# 基础配置
MONGO_URI=mongodb://bearer_token_wr:password@10.70.65.39:27019,10.70.65.40:27019,10.70.65.41:27019/bearer_token_service?replicaSet=rs0&authSource=admin

# 高可用配置（推荐）
MONGO_URI=mongodb://bearer_token_wr:password@10.70.65.39:27019,10.70.65.40:27019,10.70.65.41:27019/bearer_token_service?replicaSet=rs0&authSource=admin&readPreference=primaryPreferred&retryWrites=true&w=majority&maxPoolSize=50
```

**常见错误**:

❌ 错误：缺少数据库名称
```bash
# 会连接到默认的 test 数据库，导致权限错误
mongodb://user:pass@host:27019/?authSource=admin
```

✅ 正确：包含数据库名称
```bash
mongodb://user:pass@host:27019/bearer_token_service?authSource=admin
```

### 1. 账户查询配置 (AccountFetcher)

控制 HMAC 认证时如何查询 AccessKey 对应的账户信息（特别是 SecretKey）。

#### 模式 1: 本地 MongoDB 模式（默认）

```bash
# 不设置或设置为 local
export ACCOUNT_FETCHER_MODE=local
export MONGO_URI=mongodb://localhost:27017
```

**适用场景**：
- ✅ 开发环境
- ✅ 独立部署的服务
- ✅ 账户信息存储在本地 MongoDB

**工作原理**：
```
HMAC 请求 → 解析 AccessKey → 查询本地 MongoDB → 返回 SecretKey → 验证签名
```

---

#### 模式 2: 外部 API 模式

```bash
export ACCOUNT_FETCHER_MODE=external
export EXTERNAL_ACCOUNT_API_URL=https://account-service.qiniu.com
export EXTERNAL_ACCOUNT_API_TOKEN=your_api_token_here  # 可选
```

**适用场景**：
- ✅ **生产环境（推荐）**
- ✅ 使用共用的外部账户系统
- ✅ 账户信息集中管理

**工作原理**：
```
HMAC 请求 → 解析 AccessKey
           ↓
HTTP GET: https://account-service.qiniu.com/api/accounts?access_key={AK}
Authorization: Bearer {API_TOKEN}
           ↓
返回 {id, email, access_key, secret_key, status}
           ↓
验证签名
```

**外部 API 接口规范**：

请求：
```http
GET /api/accounts?access_key={AccessKey}
Authorization: Bearer {API_TOKEN}
```

响应（成功）：
```json
{
  "id": "acc_123",
  "email": "user@qiniu.com",
  "access_key": "AK_xxx",
  "secret_key": "SK_xxx",
  "status": "active"
}
```

响应（未找到）：
```http
HTTP/1.1 404 Not Found
```

---

### 2. 七牛 UID 映射配置 (QiniuUIDMapper)

控制 QiniuStub 认证时如何将七牛 UID 映射为系统的 account_id。

#### 模式 1: 简单模式（默认）

```bash
# 不设置或设置为 simple
export QINIU_UID_MAPPER_MODE=simple
```

**映射规则**：
```
UID: 12345  →  account_id: "qiniu_12345"
UID: 67890  →  account_id: "qiniu_67890"
```

**适用场景**：
- ✅ **快速集成（推荐新项目）**
- ✅ 不需要关联到已有账户系统
- ✅ 性能要求高（无数据库查询，~0.001ms）

**优缺点**：
- ✅ 简单快速，无需数据库
- ✅ 性能极高
- ❌ 无法存储额外账户信息
- ❌ 无法关联到已有账户

---

#### 模式 2: 数据库模式

```bash
export QINIU_UID_MAPPER_MODE=database
export QINIU_UID_AUTO_CREATE=true  # 或 false
```

**工作原理**：
```
QiniuStub 请求 → 解析 UID = 12345
           ↓
查询 MongoDB: SELECT account_id FROM accounts WHERE qiniu_uid = 12345
           ↓
找到了: 返回 account_id = "acc_001"
未找到:
  - 如果 AUTO_CREATE=true: 创建新账户并返回
  - 如果 AUTO_CREATE=false: 返回错误
```

**适用场景**：
- ✅ 需要关联到已有账户系统
- ✅ 需要存储账户元数据（邮箱、创建时间等）
- ✅ 需要账户管理功能

**优缺点**：
- ✅ 灵活，支持账户管理
- ✅ 可关联到已有系统
- ❌ 性能较低（需要数据库查询，~5ms）

---

### 3. HMAC 签名配置

#### 时间戳容忍度

```bash
export HMAC_TIMESTAMP_TOLERANCE=15m  # 默认 15 分钟
# 其他示例:
# export HMAC_TIMESTAMP_TOLERANCE=5m   # 5 分钟（更严格）
# export HMAC_TIMESTAMP_TOLERANCE=30m  # 30 分钟（更宽松）
```

**说明**：允许客户端时间与服务器时间的最大偏差，用于防止重放攻击。

---

## 🚀 常见配置场景

### 场景 1: 开发环境（默认配置）

```bash
# 最简配置，所有功能使用默认值
export MONGO_URI=mongodb://localhost:27017
go run cmd/server/main.go
```

输出：
```
✅ Using Local MongoDB AccountFetcher
✅ Using SimpleQiniuUIDMapper (format: qiniu_{uid})
✅ Unified authentication middleware initialized (HMAC + QiniuStub, tolerance=15m0s)
```

---

### 场景 2: 生产环境（使用外部账户系统）

```bash
# 账户信息从外部 API 查询
export ACCOUNT_FETCHER_MODE=external
export EXTERNAL_ACCOUNT_API_URL=https://account-service.qiniu.com
export EXTERNAL_ACCOUNT_API_TOKEN=prod_token_xyz123

# UID 映射使用数据库模式（不自动创建）
export QINIU_UID_MAPPER_MODE=database
export QINIU_UID_AUTO_CREATE=false

# MongoDB 用于存储 Token 和映射关系
export MONGO_URI=mongodb://prod-mongo:27017

# 生产环境配置
export PORT=8080
export HMAC_TIMESTAMP_TOLERANCE=10m

go run cmd/server/main.go
```

输出：
```
✅ Using External AccountFetcher (API: https://account-service.qiniu.com)
✅ Using DatabaseQiniuUIDMapper (autoCreate=false)
✅ Unified authentication middleware initialized (HMAC + QiniuStub, tolerance=10m0s)
```

---

### 场景 3: 混合模式（外部账户 + 简单 UID 映射）

```bash
# HMAC 认证使用外部 API
export ACCOUNT_FETCHER_MODE=external
export EXTERNAL_ACCOUNT_API_URL=https://account-service.qiniu.com
export EXTERNAL_ACCOUNT_API_TOKEN=your_token

# QiniuStub 认证使用简单映射（快速）
export QINIU_UID_MAPPER_MODE=simple

go run cmd/server/main.go
```

**适用于**：
- HMAC 用户来自统一账户系统
- QiniuStub 用户是临时/测试用户，不需要持久化

---

### 场景 4: 全本地模式（自动创建账户）

```bash
# 所有功能使用本地 MongoDB
export ACCOUNT_FETCHER_MODE=local
export QINIU_UID_MAPPER_MODE=database
export QINIU_UID_AUTO_CREATE=true  # 首次访问自动创建账户

export MONGO_URI=mongodb://localhost:27017

go run cmd/server/main.go
```

**适用于**：
- 独立部署的服务
- 需要完整的账户管理功能
- 接受首次访问自动创建账户

---

## 🔍 配置验证

启动服务时，日志会显示当前使用的配置：

```
🚀 Bearer Token Service V2 - Starting...
✅ Connected to MongoDB
📊 Creating database indexes...
✅ Database indexes created
✅ Services initialized
✅ Handlers initialized
✅ Using External AccountFetcher (API: https://account-service.qiniu.com)
✅ Using DatabaseQiniuUIDMapper (autoCreate=false)
✅ Unified authentication middleware initialized (HMAC + QiniuStub, tolerance=10m0s)
✅ Routes configured
🌐 Server starting on http://localhost:8080
✨ Bearer Token Service V2 is ready!
```

检查以下几行确认配置正确：
- `Using External AccountFetcher` 或 `Using Local MongoDB AccountFetcher`
- `Using DatabaseQiniuUIDMapper` 或 `Using SimpleQiniuUIDMapper`
- `tolerance=...` 确认时间戳容忍度

---

## ⚠️ 注意事项

### 1. 外部 API 模式的限制

- 外部 API 必须可达，否则服务启动失败
- 建议配置超时时间（当前默认 5 秒）
- 生产环境建议添加缓存（如 Redis）减少 API 调用

### 2. 数据库模式的性能

- 每次 Qstub 认证都会查询数据库
- 建议为 `accounts.qiniu_uid` 字段创建索引（已自动创建）
- 高并发场景建议添加缓存

### 3. 自动创建账户的安全性

- `QINIU_UID_AUTO_CREATE=true` 意味着任何有效的七牛 UID 都能创建账户
- 生产环境建议设置为 `false`，通过其他方式预先创建账户

### 4. 时间戳容忍度

- 设置过大会降低安全性（增加重放攻击窗口）
- 设置过小会导致客户端时间偏差较大时认证失败
- 建议：开发环境 15-30 分钟，生产环境 5-10 分钟

---

## 🔄 配置迁移指南

### 从本地 MongoDB 迁移到外部 API

1. 确保外部 API 已准备好
2. 测试外部 API 连通性
3. 修改环境变量：
```bash
export ACCOUNT_FETCHER_MODE=external
export EXTERNAL_ACCOUNT_API_URL=https://your-api-url
export EXTERNAL_ACCOUNT_API_TOKEN=your_token
```
4. 重启服务
5. 验证日志输出

### 从简单映射迁移到数据库映射

1. 确保 MongoDB 中已有账户数据或允许自动创建
2. 修改环境变量：
```bash
export QINIU_UID_MAPPER_MODE=database
export QINIU_UID_AUTO_CREATE=true  # 或 false
```
3. 重启服务
4. 验证现有 UID 能正确映射

---

## 📚 相关文档

- [架构设计文档](./ARCHITECTURE_DUAL_AUTH.md)
- [API 文档](./API.md)
- [部署指南](./DEPLOYMENT_SUMMARY.txt)
- [快速参考](./QUICK_REF.md)
