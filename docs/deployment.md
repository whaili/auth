# Bearer Token Service V2 - 部署指南

## 部署方式概览

| 场景 | 方式 | 配置位置 |
|------|------|----------|
| 本地开发测试 | Docker Compose | `deploy/docker-compose/` |
| 测试环境 (K8s) | Helm Chart | `deploy/helm/` + `_cust/kubeconfig-test` |
| 生产环境 (物理机) | Docker Compose | `deploy/docker-compose/` |
| 生产环境 (K8s) | Helm Chart | `deploy/helm/` + `_cust/kubeconfig-yzh` |

**可观测性**: 使用公共 Prometheus + Grafana 服务，服务暴露 `/metrics` 端点。

---

## Docker Compose 部署

### 本地测试

```bash
# 使用 Makefile
make up-test

# 或直接执行
cd deploy/docker-compose
./docker-compose-legacy-deploy.sh test
```

自动启动内置 MongoDB 和 Redis。

### 生产环境 (物理机)

```bash
cd deploy/docker-compose

# 1. 配置环境变量
cp .env.example .env
vim .env  # 配置外部 MongoDB

# 2. 部署
./docker-compose-legacy-deploy.sh prod

# 或使用 Makefile
make up-prod
```

`.env` 关键配置:

```bash
MONGO_URI=mongodb://user:pass@mongo-host:27017/dbname?authSource=admin
REDIS_ADDR=redis-host:6379
REDIS_PASSWORD=your_password
ENABLE_APP_RATE_LIMIT=true
ENABLE_ACCOUNT_RATE_LIMIT=true
ENABLE_TOKEN_RATE_LIMIT=true
```

### 管理命令

```bash
make status    # 查看状态
make logs      # 查看日志
make health    # 健康检查
make down      # 停止服务
```

---

## Helm 部署 (K8s)

Helm Chart 位于 `deploy/helm/bearer-token-service/`。

### Kubeconfig 配置

| 环境 | kubeconfig 路径 |
|------|-----------------|
| 测试 | `_cust/kubeconfig-test` |
| 生产 | `_cust/kubeconfig-yzh` |

### 测试环境

```bash
make helm-deploy-test

# 或直接使用 Helm
KUBECONFIG=_cust/kubeconfig-test helm upgrade --install bearer-token deploy/helm/bearer-token-service \
  -f deploy/helm/bearer-token-service/values-test.yaml \
  -n bearer-token --create-namespace
```

测试环境会自动部署内置 MongoDB 和 Redis。

### 生产环境

```bash
make helm-deploy-prod \
  MONGO_URI='mongodb://user:pass@mongo-host:27017/dbname?authSource=admin' \
  REDIS_ADDR='redis-host:6379'

# 或直接使用 Helm
KUBECONFIG=_cust/kubeconfig-yzh helm upgrade --install bearer-token deploy/helm/bearer-token-service \
  -f deploy/helm/bearer-token-service/values-prod.yaml \
  --set externalMongodb.uri='...' \
  --set externalRedis.addr='...' \
  -n bearer-token --create-namespace
```

### 管理命令

```bash
make helm-status-test   # 测试环境状态
make helm-status-prod   # 生产环境状态
make helm-port-forward-test  # 端口转发
make helm-delete-test   # 删除部署
```

---

## 配置说明

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `PORT` | 服务端口 | 8080 |
| `MONGO_URI` | MongoDB 连接字符串 | - |
| `MONGO_DATABASE` | 数据库名 | token_service_v2 |
| `REDIS_ENABLED` | 启用 Redis 缓存 | true |
| `REDIS_ADDR` | Redis 地址 | redis:6379 |
| `LOG_LEVEL` | 日志级别 | info |

### 限流配置

| 变量 | 说明 |
|------|------|
| `ENABLE_APP_RATE_LIMIT` | 应用层限流 |
| `ENABLE_ACCOUNT_RATE_LIMIT` | 账户层限流 |
| `ENABLE_TOKEN_RATE_LIMIT` | Token 层限流 |

---

## 健康检查

```bash
curl http://localhost:8080/health
# {"status":"ok"}

curl http://localhost:8080/metrics
# Prometheus 指标
```
