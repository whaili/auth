# Bearer Token Service V2 - 生产部署说明

## 📋 部署方式选择

### 方式 1: Docker 部署（推荐）
适合快速部署、容器化环境、多实例负载均衡

### 方式 2: 非 Docker 部署
适合传统虚拟机、物理服务器

---

## 🐳 方式 1: Docker 部署

### 1. 导入 Docker 镜像

```bash
# 加载镜像
docker load -i bearer-token-service.tar

# 验证镜像
docker images | grep bearer-token-service
```

### 2. 配置环境变量

```bash
# 复制配置模板
cp .env.example .env

# 编辑配置文件
vim .env
```

**⚠️ 重要**: 必须修改以下配置：
- `MONGO_ROOT_USERNAME` - MongoDB 用户名
- `MONGO_ROOT_PASSWORD` - MongoDB 密码（务必使用强密码！）
- `SKIP_INDEX_CREATION=true` - 数据库索引管理（默认已正确配置）

**Redis 缓存配置**（可选但推荐）：
- `REDIS_ENABLED=true` - 启用 Redis 缓存（提升查询性能）
- `REDIS_ADDR=redis:6379` - Redis 地址（使用本地 docker-compose 部署的 Redis）
- `REDIS_PASSWORD` - Redis 密码（可选）
- `CACHE_TOKEN_TTL=5m` - Token 缓存过期时间

### 3. （可选）配置 SSL 证书

```bash
# 将证书放到 nginx/ssl/ 目录
cp your-cert.crt nginx/ssl/certificate.crt
cp your-key.key nginx/ssl/private.key

# 更新 Nginx 配置
vim nginx/conf.d/default.conf
```

### 4. 启动服务

```bash
# 启动所有服务（MongoDB + 初始化 + Bearer Token Service + Nginx）
docker-compose up -d

# 查看启动日志
docker-compose logs -f

# 等待初始化完成（看到 "✅ 数据库初始化完成"）
docker-compose logs mongodb-init

# 查看服务状态
docker-compose ps
```

**启动顺序**：
1. `redis` - Redis 缓存服务启动
2. `mongodb` - MongoDB 数据库启动
3. `mongodb-init` - 自动创建数据库索引（仅运行一次）
4. `bearer-token-service` - 服务启动（SKIP_INDEX_CREATION=true）
5. `nginx` - Nginx 反向代理

### 5. 健康检查

```bash
# 检查 Redis
docker-compose exec redis redis-cli ping

# 检查 MongoDB
docker-compose exec mongodb mongosh --eval "db.adminCommand('ping')"

# 检查服务健康
curl http://localhost/health

# 或直接访问服务（如果不使用 Nginx）
curl http://localhost:8080/health
```

### 6. 验证数据库索引

```bash
# 进入 MongoDB 查看索引
docker-compose exec mongodb mongosh -u admin -p changeme

# 在 mongosh 中执行
use token_service_v2
db.accounts.getIndexes()
db.tokens.getIndexes()
db.audit_logs.getIndexes()
```

### 7. 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f bearer-token-service
docker-compose logs -f mongodb
docker-compose logs -f nginx

# 查看初始化日志
docker-compose logs mongodb-init
```

### 8. 停止服务

```bash
# 停止服务（保留数据）
docker-compose down

# 停止服务并删除数据卷（慎用！）
docker-compose down -v
```

---

## 🖥️ 方式 2: 非 Docker 部署

### 1. 准备环境

```bash
# 安装依赖
# - Go 1.18+
# - MongoDB 5.0+
# - Nginx（可选）

# 解压服务包
tar -xzf bearer-token-service.tar.gz
cd bearer-token-service
```

### 2. 初始化数据库

**⚠️ 重要步骤**：首次部署必须执行！

```bash
# 设置环境变量
export MONGO_URI="mongodb://localhost:27017"
export MONGO_DATABASE="token_service_v2"

# 执行初始化脚本
./scripts/init/init-db.sh
```

**预期输出**：
```
✅ 数据库初始化成功！
```

### 3. 配置环境变量

```bash
# 创建环境变量文件
cat > .env <<EOF
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=token_service_v2
SKIP_INDEX_CREATION=true
PORT=8080
EOF

# 加载环境变量
source .env
```

### 4. 启动服务

#### 方式 A: 直接运行

```bash
# 启动服务
SKIP_INDEX_CREATION=true ./bin/server

# 后台运行
nohup ./bin/server > logs/server.log 2>&1 &
```

#### 方式 B: 使用 Systemd

```bash
# 复制服务文件
sudo cp scripts/systemd/bearer-token-service-v2.service /etc/systemd/system/

# 编辑服务文件（修改路径和环境变量）
sudo vim /etc/systemd/system/bearer-token-service-v2.service

# 重载配置
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start bearer-token-service-v2

# 查看状态
sudo systemctl status bearer-token-service-v2

# 设置开机自启
sudo systemctl enable bearer-token-service-v2
```

### 5. 配置 Nginx（可选）

```bash
# 复制 Nginx 配置
sudo cp nginx/conf.d/default.conf /etc/nginx/sites-available/bearer-token-service

# 创建软链接
sudo ln -s /etc/nginx/sites-available/bearer-token-service /etc/nginx/sites-enabled/

# 测试配置
sudo nginx -t

# 重载 Nginx
sudo systemctl reload nginx
```

---

## 🔍 故障排查

### 问题 1: MongoDB 连接失败

```bash
# 检查 MongoDB 是否运行
docker-compose ps mongodb  # Docker 部署
systemctl status mongod    # 非 Docker 部署

# 检查连接字符串
echo $MONGO_URI

# 测试连接
mongosh $MONGO_URI --eval "db.adminCommand('ping')"
```

### 问题 2: 服务启动失败

```bash
# 查看日志
docker-compose logs bearer-token-service  # Docker
journalctl -u bearer-token-service-v2 -f  # Systemd

# 检查端口占用
ss -tlnp | grep 8080

# 检查环境变量
env | grep -E "MONGO|SKIP|PORT"
```

### 问题 3: 索引未创建

```bash
# 检查初始化日志
docker-compose logs mongodb-init  # Docker

# 手动初始化（非 Docker）
./scripts/init/init-db.sh

# 验证索引
mongosh $MONGO_URI/$MONGO_DATABASE --eval "
  db.accounts.getIndexes();
  db.tokens.getIndexes();
  db.audit_logs.getIndexes();
"
```

### 问题 4: Nginx 502 错误

```bash
# 检查服务是否运行
curl http://localhost:8080/health

# 检查 Nginx 配置
nginx -t

# 查看 Nginx 日志
tail -f /var/log/nginx/error.log
```

---

## 📊 监控和维护

### 健康检查

```bash
# API 健康检查
curl http://localhost/health

# 预期返回
{"status":"ok"}
```

### 查看日志

```bash
# Docker
docker-compose logs -f --tail=100

# Systemd
journalctl -u bearer-token-service-v2 -f

# 日志文件
tail -f logs/server.log
```

### 备份数据库

```bash
# Docker
docker-compose exec mongodb mongodump -u admin -p changeme --out /backup

# 非 Docker
mongodump --uri="$MONGO_URI" --db="$MONGO_DATABASE" --out=/backup
```

---

## 🆙 升级部署

### 升级步骤

1. **备份数据库**
   ```bash
   mongodump --uri="$MONGO_URI" --db="$MONGO_DATABASE"
   ```

2. **停止服务**
   ```bash
   docker-compose down  # Docker
   systemctl stop bearer-token-service-v2  # Systemd
   ```

3. **更新镜像/二进制**
   ```bash
   docker load -i bearer-token-service-new.tar  # Docker
   tar -xzf bearer-token-service-new.tar.gz  # 非 Docker
   ```

4. **执行数据库迁移**（如有新索引）
   ```bash
   ./scripts/init/init-db.sh
   ```

5. **启动服务**
   ```bash
   docker-compose up -d  # Docker
   systemctl start bearer-token-service-v2  # Systemd
   ```

---

## 📚 相关文档

- [API 文档](../../docs/api/API.md)
- [配置指南](../../docs/CONFIG.md)
- [数据库初始化](../../docs/DATABASE_INIT.md)
- [生产部署完整指南](../../PRODUCTION_DEPLOYMENT.md)

---

## ✅ 部署检查清单

- [ ] MongoDB 已配置强密码
- [ ] Redis 缓存已启动（可选但推荐）
- [ ] 数据库索引已创建（`mongodb-init` 容器成功运行）
- [ ] 服务健康检查通过（`/health` 返回 200）
- [ ] Nginx 配置正确（如使用）
- [ ] SSL 证书已配置（生产环境）
- [ ] 环境变量已正确设置
- [ ] 日志轮转已配置
- [ ] 监控告警已设置
- [ ] 备份策略已实施

---

**部署完成！** 🎉

如有问题，请参考故障排查章节或查看详细文档。
