# Bearer Token Service V2 - 环境与部署手册

## 环境总览

| 环境 | URL | 基础设施 | 访问方式 |
|------|-----|----------|----------|
| 本地开发 | `http://localhost:8081` | 本地进程 + 本地 MongoDB/Redis | 直接 |
| 测试环境 (QA) | `http://bearer-token-test.jfcs-k8s-qa1.qiniu.io` | k8s (内置 MongoDB + Redis) | 直接 |
| 生产-国内 xs 物理机 | — | vmxs1 / vmxs2 Docker Compose | SSH 10.134.2.1 |
| 生产-国内 yzh k8s | `http://bearer-token.qiniu.io` | k8s + 外部 MongoDB + 独立 Redis | 直接 |
| 生产-海外 sufy k8s | `https://bo-api-key.sufy.dev` | k8s + 共享 MongoDB + 共享 Redis | `ssh nsg4` 中转 |

---

## 本地开发

### 前置依赖

- Go 1.21+
- 本地 MongoDB: `mongodb://admin:123456@localhost:27017`
- 本地 Redis: `localhost:6379`

### 编译与启动

```bash
# 编译
make compile          # 输出 bin/tokenserv

# 启动（端口 8081）
bash tests/api/start_local.sh
```

`start_local.sh` 关键配置：

```
PORT=8081
MONGO_URI=mongodb://admin:123456@localhost:27017
REDIS_ENABLED=true
REDIS_ADDR=localhost:6379
QINIU_UID_MAPPER_MODE=simple
QINIU_UID_AUTO_CREATE=false
```

### 单元测试

```bash
make test
# 等价于: go test -v ./handlers/... ./service/... ./repository/...
```

---

## 测试环境 (QA k8s)

### 基本信息

- **URL**: `http://bearer-token-test.jfcs-k8s-qa1.qiniu.io`
- **Namespace**: `bearer-token-test`
- **kubeconfig**: `_cust/kubeconfig-test`（不入 git）
- **镜像仓库**: `aslan-spock-register.qiniu.io/miku-stream/bearer-token-service`
- **MongoDB**: helm 内置（`token_service_v2` 库）
- **Redis**: helm 内置
- **Qconf**: `http://kodo-dev.confg.jfcs-k8s-qa2.qiniu.io`
- **限流**: 全部关闭（方便测试）

### 部署

```bash
# 一键：编译 + 构建 + 推送镜像 + helm upgrade
make deploy-k8s-test

# 等价拆分步骤
make push   # 构建并推送镜像
KUBECONFIG=_cust/kubeconfig-test helm upgrade --install bearer-token \
    deploy/helm/bearer-token-service \
    -f deploy/helm/bearer-token-service/values-test.yaml \
    -n bearer-token-test
```

### 验证

```bash
# 健康检查
curl http://bearer-token-test.jfcs-k8s-qa1.qiniu.io/health

# API 功能测试
BASE_URL=http://bearer-token-test.jfcs-k8s-qa1.qiniu.io \
    bash tests/api/test_qstub_api.sh
```

---

## 生产-国内 xs 物理机

### 基本信息

| 设备 | IP | 部署路径 |
|------|----|----------|
| vmxs1 | 10.134.2.1 | `/root/deploy` |
| vmxs2 | 10.134.2.2 | `/root/haili/deploy` |

- **部署方式**: Docker Compose
- **配置文件**: `deploy/docker-compose/`

### 部署

```bash
# 部署到 vmxs1
./deploy/scripts/deploy.sh physical vmxs1

# 部署到 vmxs2
./deploy/scripts/deploy.sh physical vmxs2
```

### 调试

```bash
ssh 10.134.2.1
cd /root/deploy
docker-compose logs -f bearer-token-service
docker-compose ps
curl http://localhost:8080/health
```

---

## 生产-国内 yzh k8s

### 基本信息

- **URL**: `http://bearer-token.qiniu.io`（Ingress）
- **镜像仓库**: `registry-kubesphere-hd.qiniu.io/miku-stream/bearer-token-service`
- **MongoDB（外部共用）**: `10.70.65.41:27019,10.70.65.34:27019,10.70.65.39:27019` / 库 `bearer_token_main`
- **Redis**: helm 独立部署（非共享）
- **Qconf**: `http://10.34.35.42:8510,http://10.34.35.43:8510`
- **副本数**: 4（HPA 4~20）
- **限流**: 全部开启

### 部署

```bash
# 先推送镜像（修改 IMAGE_TAG 为目标版本）
IMAGE_TAG=v2.x.x make push

# helm upgrade（需要国内 k8s 的 kubeconfig）
helm upgrade --install bearer-token deploy/helm/bearer-token-service \
    -f deploy/helm/bearer-token-service/values-prod.yaml \
    -n bearer-token
```

配置文件: `deploy/helm/bearer-token-service/values-prod.yaml`

---

## 生产-海外 sufy k8s

### 基本信息

- **URL**: `https://bo-api-key.sufy.dev`
- **访问中转**: `ssh nsg4`（所有调试/测试命令都要经过它）
- **MongoDB**: 共享实例（Go template 变量 `{{.mongo_db8}}`，由运维注入）
- **MongoDB（共享）**:
  - 节点: `10.62.31.34:27017,10.62.31.25:27017,10.62.31.23:27017`（Replica Set）
  - 库: `bearer_token_main` / 用户: `bearer_token_main_wr` / authSource: `admin`
  - 凭据变量: `SUFY_MONGO_URI`（见 `_cust/credentials.env`，不入 git）
- **Redis（共享集群）**:
  - 节点: `10.62.31.33:6379,10.62.31.25:6379,10.62.31.23:6379,10.62.31.32:6379,10.62.31.34:6379`（及对应 :6380）
  - 凭据变量: `SUFY_REDIS_HOSTS` / `SUFY_REDIS_PASSWORD`（见 `_cust/credentials.env`，不入 git）
  - 模式: Redis Cluster（用 `redis.RedisCluster` 连接）
- **Qconf**: `http://10.76.25.25:8520,http://10.76.24.24:8520`
  - 有效测试 UID: `1694435075`（`wanghaili@qiniu.com`，该 UID 在 sufy qconf 有数据，可验证 `/validateu` 返回完整 `user_info`）
  - 凭据变量: `SUFY_QCONF_TEST_UID`（见 `_cust/credentials.env`）
- **限流**: 全部开启

### 部署流程（运维驱动）

1. 修改 `deploy/sufy/bo/api-key/sufy-api-key.yml`（Go template 格式）
2. 提交并推送到 GitLab
3. 通知运维：运维拉取配置，渲染 Go template 变量，apply 到 sufy k8s 集群
4. 运维无需手动操作镜像——镜像由开发推送到仓库后在 yml 中指定版本

配置文件: `deploy/sufy/bo/api-key/sufy-api-key.yml`

```yaml
# 关键字段（Go template，运维渲染）
mongo:
  uri: "mongodb://{{.mongo_db8_user}}:{{.mongo_db8_pwd}}@{{.mongo_db8}}/..."
  database: "bearer_token_main"
redis:
  enabled: true
  addr: "{{.redis_host}}"
  password: "{{.redis_password}}"
```

### 调试与测试

所有命令均需通过 `ssh nsg4` 中转：

```bash
# 健康检查
ssh nsg4 "curl https://bo-api-key.sufy.dev/health"

# API 功能测试（全量）
ssh nsg4 "BASE_URL=https://bo-api-key.sufy.dev bash -s" \
    < tests/api/test_qstub_api.sh

# Redis 缓存测试（共享 Redis，安全版，不 FLUSHALL）
# 注：nsg4 无 redis-cli，用 Python redis-py (RedisCluster)
# 测试脚本: /tmp/test_redis_cache.py（本地生成后传入）
ssh nsg4 "python3 -s" < /tmp/test_redis_cache.py

# 手动验证指定 UID
ssh nsg4 "
  RESP=\$(curl -s -X POST https://bo-api-key.sufy.dev/api/v2/tokens \
    -H 'Authorization: QiniuStub uid=1694435075&ut=1' \
    -H 'Content-Type: application/json' \
    -d '{\"description\":\"test\",\"expires_in\":3600}')
  TOKEN=\$(echo \$RESP | python3 -c 'import sys,json; print(json.load(sys.stdin)[\"token\"])')
  curl -s -X POST https://bo-api-key.sufy.dev/api/v2/validateu \
    -H \"Authorization: Bearer \$TOKEN\" | python3 -m json.tool
"
```

### Redis 注意事项

- **禁止** `FLUSHALL` / `FLUSHDB`：共享实例，会清除其他业务数据
- 只操作 `token:val:{token_value}` 格式的 key
- nsg4 无 redis-cli，使用 `redis-py` (`pip3 install redis`)
- 连接用 `RedisCluster` + `ClusterNode`，不能用单节点 `Redis` 客户端

---

## 测试脚本汇总

| 脚本 | 用途 | 使用方式 |
|------|------|----------|
| `tests/api/test_qstub_api.sh` | API 全量功能测试 | `BASE_URL=<url> bash tests/api/test_qstub_api.sh` |
| `tests/api/test_redis_cache.sh` | Redis 缓存测试（本地/docker） | `BASE_URL=<url> bash tests/api/test_redis_cache.sh` |
| `tests/api/test_rate_limit.sh` | 限流测试 | `BASE_URL=<url> bash tests/api/test_rate_limit.sh` |
| `tests/api/start_local.sh` | 启动本地服务 | `bash tests/api/start_local.sh` |

> 海外 sufy 环境的 Redis 测试不能用 `test_redis_cache.sh`（依赖 `docker exec`），
> 使用 `/tmp/test_redis_cache.py`（Python redis-py 版本，安全适配共享 Redis）。

---

## 镜像版本管理

| 环境 | 镜像仓库 | 当前版本 |
|------|----------|----------|
| QA k8s | `aslan-spock-register.qiniu.io/miku-stream/bearer-token-service` | `values-test.yaml` 中的 `tag` |
| 国内 yzh k8s | `registry-kubesphere-hd.qiniu.io/miku-stream/bearer-token-service` | `values-prod.yaml` 中的 `tag` |
| 海外 sufy | 同国内仓库或 sufy 专用仓库 | `sufy-api-key.yml` 中指定 |

升级版本步骤：
1. 修改对应 `values-*.yaml` 或 `sufy-api-key.yml` 中的镜像 tag
2. `make push` 推送新镜像
3. 按各环境部署流程执行

---

## 常见问题排查

### `/api/v2/validateu` 返回 `user_info: null`

Qconf RPC 未连通或该 UID 在 Qconf 无数据。属于优雅降级，服务本身正常。
检查：对应环境的 `qconf.master_hosts` 是否可达，UID 是否在 Qconf 有记录。

### 海外环境 `invalid qiniu uid format`

UID 格式校验失败。注意：`UID` 是 bash 内置只读变量，脚本里要用其他变量名（如 `QUID`）。

### Redis 连接报 `MovedError`

Redis 是 Cluster 模式，不能用单节点 `redis.Redis`，必须用 `redis.RedisCluster`。

### 镜像拉取失败

检查目标 k8s 环境能否访问对应镜像仓库，QA 用 `aslan-spock-register`，国内生产用 `registry-kubesphere-hd`。
