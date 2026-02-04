# 部署和配置指南

Bearer Token Service V2 完整的部署和配置文档。

## 目录

- [概述](#概述)
- [环境变量配置](#环境变量配置)
- [Qconf RPC 配置](#qconf-rpc-配置)
- [Docker Compose 部署](#docker-compose-部署)
- [Helm (Kubernetes) 部署](#helm-kubernetes-部署)
- [环境对比](#环境对比)
- [故障排查](#故障排查)
- [Make 命令参考](#make-命令参考)

---

## 概述

### 部署方式

| 场景 | 方式 | 配置位置 |
|------|------|----------|
| 本地开发测试 | Docker Compose | `deploy/docker-compose/` |
| 测试环境 (K8s) | Helm Chart | `deploy/helm/` + `values-test.yaml` |
| 生产环境 (K8s) | Helm Chart | `deploy/helm/` + `values-prod.yaml` |

### 用户信息查询

系统使用 **Qconfapi RPC** 获取用户详细信息（用于 `/api/v2/validateu` 端点）：

- **Qconfapi RPC** - 七牛内部 RPC 服务
- **优雅降级** - 如果 Qconf 不可用，返回基本 token 信息，user_info 为 null

---

## 环境变量配置

### 基础配置

| 变量 | 说明 | 默认值 | 必填 |
|------|------|--------|------|
| `PORT` | 服务监听端口 | `8080` | 否 |
| `LOG_LEVEL` | 日志级别 (debug/info/warn/error) | `info` | 否 |
| `LOG_FORMAT` | 日志格式 (json/text) | `json` | 否 |
| `LOG_FILE` | 日志文件路径 | - | 否 |

### MongoDB 配置

| 变量 | 说明 | 默认值 | 必填 |
|------|------|--------|------|
| `MONGO_URI` | MongoDB 连接字符串 | `mongodb://localhost:27017` | 是 |
| `MONGO_DATABASE` | 数据库名 | `token_service_v2` | 否 |
| `SKIP_INDEX_CREATION` | 跳过索引创建（多实例部署） | `false` | 否 |

**MongoDB URI 格式**:
```
mongodb://user:pass@host1:27017,host2:27017/dbname?authSource=admin&replicaSet=rs0
```

### Redis 配置（可选，推荐启用）

| 变量 | 说明 | 默认值 | 必填 |
|------|------|--------|------|
| `REDIS_ENABLED` | 是否启用 Redis 缓存 | `false` | 否 |
| `REDIS_ADDR` | Redis 地址 | `localhost:6379` | 否 |
| `REDIS_PASSWORD` | Redis 密码 | - | 否 |
| `REDIS_DB` | Redis 数据库编号 (0-15) | `0` | 否 |
| `CACHE_TOKEN_TTL` | Token 缓存过期时间 | `5m` | 否 |

### Qconf RPC 配置（用户信息查询）

| 变量 | 说明 | 默认值 | 必填 |
|------|------|--------|------|
| `QCONF_ENABLED` | 是否启用 qconfapi | `false` | 否 |
| `QCONF_ACCESS_KEY` | Qconf AccessKey | - | 是* |
| `QCONF_SECRET_KEY` | Qconf SecretKey | - | 是* |
| `QCONF_MASTER_HOSTS` | Master 节点列表（逗号分隔） | - | 是* |
| `QCONF_MC_HOSTS` | Memcache 节点列表（可选） | - | 否 |
| `QCONF_LC_EXPIRES_MS` | 本地缓存过期时间（毫秒） | `300000` | 否 |
| `QCONF_LC_DURATION_MS` | 本地缓存刷新间隔（毫秒） | `60000` | 否 |
| `QCONF_LC_CHAN_BUFSIZE` | 消息队列缓冲区大小 | `1000` | 否 |
| `QCONF_MC_RW_TIMEOUT_MS` | Memcache 读写超时（毫秒） | `1000` | 否 |

*仅当 `QCONF_ENABLED=true` 时必填

### QiniuStub 认证配置

| 变量 | 说明 | 默认值 | 必填 |
|------|------|--------|------|
| `QINIU_UID_MAPPER_MODE` | UID 映射方式 (simple/database) | `simple` | 否 |
| `QINIU_UID_AUTO_CREATE` | 自动创建账户（仅 database 模式） | `false` | 否 |
| `HMAC_TIMESTAMP_TOLERANCE` | 时间戳容忍度（防重放攻击） | `15m` | 否 |

**UID 映射模式说明**:
- `simple`: 直接映射为 `qiniu_{uid}`（推荐，性能高）
- `database`: 查询 MongoDB 获取 account_id（灵活，支持账户管理）

### 限流配置

| 变量 | 说明 | 默认值 | 必填 |
|------|------|--------|------|
| `ENABLE_APP_RATE_LIMIT` | 应用层限流 | `false` | 否 |
| `ENABLE_ACCOUNT_RATE_LIMIT` | 账户层限流 | `false` | 否 |
| `ENABLE_TOKEN_RATE_LIMIT` | Token 层限流 | `false` | 否 |
| `APP_LIMIT_PER_MINUTE` | 应用层每分钟限制 | `1000` | 否 |
| `APP_LIMIT_PER_HOUR` | 应用层每小时限制 | `50000` | 否 |
| `APP_LIMIT_PER_DAY` | 应用层每天限制 | `1000000` | 否 |

详细限流配置见 [RATE_LIMIT.md](./RATE_LIMIT.md)

---

## Qconf RPC 配置

### 什么是 Qconf

Qconfapi 是七牛内部的 RPC 服务，用于获取用户账户信息。Bearer Token Service V2 使用它来支持 `/api/v2/validateu` 端点，返回完整的用户信息。

### 测试环境配置

```bash
QCONF_ENABLED=true
QCONF_ACCESS_KEY=-lnWyW53aRF1AYUj5D2oBBub377cTMYawZdOT25z
QCONF_SECRET_KEY=FOXDmvIMttefUbUO8RWFIJ8JJdbPwvQGwoqZQL3O
QCONF_MASTER_HOSTS=http://kodo-dev.confg.jfcs-k8s-qa2.qiniu.io
```

**测试说明**:
- 接入点: `http://kodo-dev.confg.jfcs-k8s-qa2.qiniu.io`
- 数据范围: 有限的测试用户数据
- 测试 UID: `1810810692`（已验证可用）

### 生产环境配置

```bash
QCONF_ENABLED=true
QCONF_ACCESS_KEY=ppca4hFBYQ_ykozmLUcSIJi8eLnYhFahE0OF5MoZ
QCONF_SECRET_KEY=<联系运维获取>
QCONF_MASTER_HOSTS=http://10.34.35.42:8510,http://10.34.35.43:8510
```

**生产说明**:
- 接入点: `http://10.34.35.42:8510,http://10.34.35.43:8510`
- 数据范围: 完整的生产用户数据
- SecretKey: 联系运维团队获取

### 工作流程

```
/api/v2/validateu 请求
    ↓
1. 验证 Bearer Token
    ↓
2. 提取 UID
    ↓
3. 调用 qconfapi.GetAccountInfo(uid)
    ↓
4. 返回 token_info + user_info
```

如果 Qconf 调用失败（Not Found、网络错误等），系统会优雅降级：

```json
{
  "valid": true,
  "message": "Token is valid",
  "token_info": {
    "token_id": "tk_xxx",
    "uid": "1369077332",
    "is_active": true,
    "user_info": null  // ← 降级，但 token 验证仍然成功
  }
}
```

### 缓存策略

Qconfapi 内置两层缓存：

1. **本地缓存** (Local Cache)
   - 过期时间: `QCONF_LC_EXPIRES_MS` (默认 5 分钟)
   - 刷新间隔: `QCONF_LC_DURATION_MS` (默认 1 分钟)
   - 作用: 减少 RPC 调用，提升性能

2. **Memcache**（可选）
   - 配置: `QCONF_MC_HOSTS`
   - 超时: `QCONF_MC_RW_TIMEOUT_MS` (默认 1 秒)
   - 作用: 多实例共享缓存

---

## Docker Compose 部署

### 测试环境快速启动

```bash
# 1. 进入项目目录
cd /root/src/auth

# 2. 启动测试环境（包含 MongoDB + Redis）
make up-test

# 3. 查看日志
make logs

# 4. 运行测试
make test-qconf

# 5. 停止服务
make down
```

### 配置文件

配置文件位置: `deploy/docker-compose/.env`

**测试环境示例**:

```bash
# MongoDB
MONGO_ROOT_USERNAME=admin
MONGO_ROOT_PASSWORD=123456

# Qconf RPC
QCONF_ENABLED=true
QCONF_ACCESS_KEY=-lnWyW53aRF1AYUj5D2oBBub377cTMYawZdOT25z
QCONF_SECRET_KEY=FOXDmvIMttefUbUO8RWFIJ8JJdbPwvQGwoqZQL3O
QCONF_MASTER_HOSTS=http://kodo-dev.confg.jfcs-k8s-qa2.qiniu.io

# Redis
REDIS_ENABLED=true
REDIS_ADDR=redis:6379

# 服务
HOST_PORT=8081
PORT=8080
LOG_LEVEL=debug
LOG_FORMAT=text
```

**生产环境示例**:

```bash
# MongoDB（外部副本集）
MONGO_URI=mongodb://bearer_token_wr:<PASSWORD>@10.70.65.41:27019,10.70.65.34:27019,10.70.65.39:27019/bearer_token_main?authSource=admin

# Qconf RPC
QCONF_ENABLED=true
QCONF_ACCESS_KEY=ppca4hFBYQ_ykozmLUcSIJi8eLnYhFahE0OF5MoZ
QCONF_SECRET_KEY=<YOUR_SECRET>
QCONF_MASTER_HOSTS=http://10.34.35.42:8510,http://10.34.35.43:8510

# Redis
REDIS_ENABLED=true
REDIS_ADDR=redis:6379

# 限流（生产启用）
ENABLE_APP_RATE_LIMIT=true
ENABLE_ACCOUNT_RATE_LIMIT=true
ENABLE_TOKEN_RATE_LIMIT=true

# 服务
HOST_PORT=8080
PORT=8080
LOG_LEVEL=info
LOG_FORMAT=json
SKIP_INDEX_CREATION=true
```

### 服务验证

```bash
# 健康检查
curl http://localhost:8081/health

# 查看 Qconf 连接状态
docker logs bearer-token-service | grep qconfapi

# 测试 validateu 端点（需要先创建 token）
make test-qconf
```

---

## Helm (Kubernetes) 部署

### 测试环境部署

**配置文件**: `deploy/helm/bearer-token-service/values-test.yaml`

```yaml
replicaCount: 1

# 内置 MongoDB
mongodb:
  enabled: true
  auth:
    username: admin
    password: <YOUR_PASSWORD>
    database: token_service_v2

# 内置 Redis
redis:
  enabled: true

# Qconf RPC
qconf:
  enabled: true
  accessKey: "-lnWyW53aRF1AYUj5D2oBBub377cTMYawZdOT25z"
  secretKey: "<YOUR_SECRET>"
  masterHosts: "http://kodo-dev.confg.jfcs-k8s-qa2.qiniu.io"

# Ingress
ingress:
  enabled: true
  host: bearer-token-test.qiniu.io

config:
  logLevel: "debug"
  skipIndexCreation: "false"
  rateLimit:
    app: false
    account: false
    token: false
```

**部署命令**:

```bash
# 部署到测试环境
make helm-deploy-test

# 查看状态
make helm-status-test

# 查看 Pod 日志
kubectl logs -n bearer-token -l app.kubernetes.io/name=bearer-token-service

# 端口转发（本地测试）
make helm-port-forward-test
```

### 生产环境部署

**配置文件**: `deploy/helm/bearer-token-service/values-prod.yaml`

```yaml
replicaCount: 4

image:
  repository: registry-kubesphere-hd.qiniu.io/miku-stream/bearer-token-service
  tag: "v2.0.0"
  pullPolicy: Always

# 禁用内置 MongoDB（使用外部）
mongodb:
  enabled: false

# 外部 MongoDB
externalMongodb:
  uri: "mongodb://bearer_token_wr:<PASSWORD>@10.70.65.41:27019,10.70.65.34:27019,10.70.65.39:27019/bearer_token_main?authSource=admin"

# 内置 Redis
redis:
  enabled: true
  storage: 5Gi

# Qconf RPC
qconf:
  enabled: true
  accessKey: "ppca4hFBYQ_ykozmLUcSIJi8eLnYhFahE0OF5MoZ"
  secretKey: "<YOUR_SECRET>"
  masterHosts: "http://10.34.35.42:8510,http://10.34.35.43:8510"

# Nginx 反向代理
nginx:
  enabled: true
  replicaCount: 2

# HPA
hpa:
  enabled: true
  minReplicas: 4
  maxReplicas: 20
  targetCPUUtilizationPercentage: 70

# Ingress
ingress:
  enabled: true
  host: bearer-token.qiniu.io

config:
  ginMode: "release"
  logLevel: "info"
  skipIndexCreation: "true"
  rateLimit:
    app: true
    account: true
    token: true

resources:
  requests:
    cpu: 1000m
    memory: 1Gi
  limits:
    cpu: 2000m
    memory: 2Gi
```

**部署命令**:

```bash
# 构建并推送镜像
make build
make push

# 部署到生产环境
make helm-deploy-prod

# 查看状态
make helm-status-prod

# 滚动更新
helm upgrade bearer-token deploy/helm/bearer-token-service \
  -f deploy/helm/bearer-token-service/values-prod.yaml \
  -n bearer-token
```

---

## 环境对比

### 配置差异

| 配置项 | 测试环境 | 生产环境 |
|--------|----------|----------|
| **MongoDB** | 内置容器 | 外部副本集 (3节点) |
| **Redis** | 内置容器 | 内置容器（5Gi存储） |
| **Qconf 端点** | `kodo-dev.confg.jfcs-k8s-qa2.qiniu.io` | `10.34.35.42:8510,10.34.35.43:8510` |
| **副本数** | 1 | 4（HPA: 4-20） |
| **Nginx** | 禁用 | 启用（2副本） |
| **限流** | 全部禁用 | 全部启用 |
| **日志级别** | debug | info |
| **日志格式** | text | json |
| **索引创建** | 启用 | 跳过（启动时） |
| **资源配额** | 200m CPU / 256Mi | 1000m CPU / 1Gi |

### 网络访问

**测试环境**:
- Qconf: ✅ 可通过公网域名访问
- MongoDB: ✅ K8s 内部服务
- Redis: ✅ K8s 内部服务

**生产环境**:
- Qconf: ⚠️ 内网 IP，需要 VPN 或 K8s 内部访问
- MongoDB: ⚠️ 内网副本集，需要配置网络策略
- Redis: ✅ K8s 内部服务

---

## 故障排查

### 问题 1: Qconfapi 连接失败

**症状**:
```
{"level":"ERROR","msg":"Failed to initialize qconfapi"}
```

**排查步骤**:
1. 检查网络连通性
   ```bash
   curl http://kodo-dev.confg.jfcs-k8s-qa2.qiniu.io
   ```
2. 验证凭据是否正确
   ```bash
   docker logs bearer-token-service | grep "access_key"
   ```
3. 检查 QCONF_MASTER_HOSTS 格式（逗号分隔，无空格）

**解决方案**:
- 修正网络配置或防火墙规则
- 更新正确的 qconf 凭据
- 确认 `QCONF_ENABLED=true`

### 问题 2: user_info 为 null

**症状**:
```json
{
  "valid": true,
  "token_info": {
    "user_info": null
  }
}
```

**排查步骤**:
1. 检查日志中是否有 "GetAccountInfo failed" 错误
   ```bash
   docker logs bearer-token-service | grep "GetAccountInfo"
   ```
2. 确认 UID 是否在 qconf 数据库中存在
3. 验证 qconfapi 连接是否正常

**解决方案**:
- 测试环境：使用已知有效的 UID（如 `1810810692`）
- 生产环境：确认用户 UID 存在于生产 qconf
- 如果是合法的 "Not Found"，这是正常的优雅降级行为

### 问题 3: MongoDB 连接失败

**症状**:
```
{"level":"ERROR","msg":"Failed to connect to MongoDB"}
```

**排查步骤**:
1. 检查 MONGO_URI 格式是否正确
2. 验证 MongoDB 是否运行
   ```bash
   docker ps | grep mongodb
   ```
3. 测试连接
   ```bash
   mongosh "mongodb://admin:123456@localhost:27017"
   ```

**解决方案**:
- 检查 MongoDB 容器状态：`docker logs bearer-token-mongodb`
- 确认用户名密码正确
- 验证网络连通性

### 问题 4: Redis 缓存不生效

**症状**: Token 验证很慢，每次都查询 MongoDB

**排查步骤**:
1. 确认 Redis 已启用
   ```bash
   docker logs bearer-token-service | grep "Redis"
   ```
2. 检查 Redis 连接
   ```bash
   docker exec bearer-token-redis redis-cli ping
   ```

**解决方案**:
- 确认 `REDIS_ENABLED=true`
- 检查 Redis 地址配置
- 查看 Redis 日志：`docker logs bearer-token-redis`

### 问题 5: 测试失败

**症状**: `make test-qconf` 失败

**解决方案**:
```bash
# 重启服务
make down
make up-test

# 等待服务就绪
sleep 10

# 检查健康状态
curl http://localhost:8081/health

# 重新运行测试
make test-qconf
```

---

## Make 命令参考

### 测试相关

```bash
make test          # 运行所有测试（单元测试 + API 测试）
make test-unit     # 仅运行单元测试
make test-api      # 仅运行 API 集成测试
make test-qconf    # 测试 qconfapi 集成（使用有效 UID）
```

### Docker Compose

```bash
make up-test       # 启动测试环境（MongoDB + Redis + Service）
make down          # 停止所有服务
make logs          # 查看服务日志（实时）
make status        # 查看服务状态
make health        # 健康检查
```

### Helm (K8s)

```bash
# 部署
make helm-deploy-test   # 部署到测试环境
make helm-deploy-prod   # 部署到生产环境

# 状态查看
make helm-status        # 查看所有环境状态
make helm-status-test   # 查看测试环境状态
make helm-status-prod   # 查看生产环境状态

# 端口转发
make helm-port-forward-test   # 转发测试环境端口到本地
make helm-port-forward-prod   # 转发生产环境端口到本地

# 清理
make helm-delete-test   # 删除测试环境
make helm-delete-prod   # 删除生产环境（需确认）
```

### 构建与打包

```bash
make compile       # 编译 Go 二进制文件
make build         # 构建 Docker 镜像
make push          # 推送镜像到仓库
make package       # 打包 Helm Chart
```

### 其他

```bash
make help          # 显示所有可用命令
make clean         # 清理构建产物
```

---

## 监控建议

### 关键指标

1. **Qconfapi 调用成功率**
   - 目标: >99% (生产环境)
   - 监控: `qconfapi_call_success / qconfapi_call_total`

2. **user_info 可用率**
   - 目标: >95% (生产环境)
   - 监控: `user_info_non_null / validateu_requests_total`

3. **响应延迟**
   - `/api/v2/validateu` P95: <50ms
   - `/api/v2/validate` P95: <30ms

4. **优雅降级频率**
   - user_info=null 的请求比例
   - 生产环境应该 <1%

### 日志查询

```bash
# Qconfapi 连接状态
docker logs bearer-token-service | grep "qconfapi"

# GetAccountInfo 错误
docker logs bearer-token-service | grep "GetAccountInfo failed"

# 优雅降级情况
docker logs bearer-token-service | grep "user_info.*null"

# 错误统计
docker logs bearer-token-service | grep "ERROR" | wc -l
```

---

## 参考文档

- [CLAUDE.md](../CLAUDE.md) - 项目结构和开发指南
- [README.md](../README.md) - 项目快速开始
- [RATE_LIMIT.md](./RATE_LIMIT.md) - 限流配置详解
- [API 文档](./api/openapi.yaml) - OpenAPI 规范
