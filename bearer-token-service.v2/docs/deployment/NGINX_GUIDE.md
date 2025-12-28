# Nginx 反向代理配置指南

> Bearer Token Service V2 - Nginx 集成完全手册

---

## 📋 目录

- [架构说明](#架构说明)
- [配置文件说明](#配置文件说明)
- [快速开始](#快速开始)
- [HTTPS 配置](#https-配置)
- [性能优化](#性能优化)
- [监控与日志](#监控与日志)
- [故障排查](#故障排查)

---

## 🏗️ 架构说明

### 请求流程

```
客户端
  ↓
Nginx (80/443)
  ↓
Bearer Token Service (8080)
  ↓
MongoDB (27017)
```

### 端口配置

| 服务 | 端口 | 说明 |
|------|------|------|
| Nginx HTTP | 80 | 公网访问（HTTP） |
| Nginx HTTPS | 443 | 公网访问（HTTPS） |
| Bearer Token Service | 8080 | 内部端口（不暴露） |
| MongoDB | 27017 | 内部端口（不暴露） |

---

## 📁 配置文件说明

### 目录结构

```
nginx/
├── nginx.conf              # Nginx 主配置
├── conf.d/
│   ├── http.conf          # HTTP (80) 配置
│   └── https.conf         # HTTPS (443) 配置（默认禁用）
├── ssl/
│   ├── README.md          # SSL 证书说明
│   ├── server.crt         # SSL 证书（用户自备）
│   └── server.key         # SSL 私钥（用户自备）
└── logs/
    ├── access.log         # 访问日志
    └── error.log          # 错误日志
```

### nginx.conf

主配置文件，包含:
- Worker 进程配置
- 日志格式定义
- Gzip 压缩配置
- 安全头配置
- 后端服务器定义 (upstream)

### conf.d/http.conf

HTTP 服务配置 (80 端口):
- ✅ 默认启用
- 反向代理到后端服务
- 健康检查端点
- 访问日志配置

### conf.d/https.conf

HTTPS 服务配置 (443 端口):
- ⚠️ 默认禁用（需要 SSL 证书）
- SSL/TLS 安全配置
- HTTP/2 支持
- 自动重定向 HTTP → HTTPS（可选）

---

## 🚀 快速开始

### 1. 启动服务（HTTP）

```bash
# 启动所有服务（包括 Nginx）
docker compose up -d

# 查看服务状态
docker compose ps

# 验证 Nginx
curl http://localhost/health
```

预期输出:
```json
{"status":"ok"}
```

### 2. 测试 API（通过 Nginx）

```bash
# 注册账户（通过 Nginx）
curl -X POST http://localhost/api/v2/accounts/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "company": "Test Inc",
    "password": "test123456"
  }'
```

### 3. 查看 Nginx 日志

```bash
# 实时查看访问日志
docker compose logs -f nginx

# 或者查看日志文件
tail -f nginx/logs/access.log
```

---

## 🔐 HTTPS 配置

### 方式 1: 使用自签名证书（仅测试）

```bash
# 1. 生成自签名证书
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout nginx/ssl/server.key \
  -out nginx/ssl/server.crt \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=Test/CN=localhost"

# 2. 设置权限
chmod 600 nginx/ssl/server.key
chmod 644 nginx/ssl/server.crt

# 3. 启用 HTTPS 配置
vim nginx/conf.d/https.conf
# 取消所有注释，修改 server_name

# 4. 重启 Nginx
docker compose restart nginx

# 5. 测试 HTTPS（-k 跳过证书验证）
curl -k https://localhost/health
```

### 方式 2: 使用 Let's Encrypt（生产环境）

```bash
# 1. 安装 certbot
sudo apt-get install certbot

# 2. 停止 Nginx（certbot 需要占用 80 端口）
docker compose stop nginx

# 3. 获取证书
sudo certbot certonly --standalone -d your-domain.com

# 4. 复制证书到项目目录
sudo cp /etc/letsencrypt/live/your-domain.com/fullchain.pem nginx/ssl/server.crt
sudo cp /etc/letsencrypt/live/your-domain.com/privkey.pem nginx/ssl/server.key
sudo chown $USER:$USER nginx/ssl/*
chmod 600 nginx/ssl/server.key

# 5. 启用 HTTPS 配置
vim nginx/conf.d/https.conf
# 取消所有注释，修改 server_name 为 your-domain.com

# 6. 启用 HTTP → HTTPS 重定向
vim nginx/conf.d/http.conf
# 在 server 块开头添加:
# if ($host = your-domain.com) {
#     return 301 https://$host$request_uri;
# }

# 7. 启动 Nginx
docker compose start nginx

# 8. 测试 HTTPS
curl https://your-domain.com/health
```

### 方式 3: 使用现有证书

```bash
# 1. 复制证书文件
cp /path/to/your.crt nginx/ssl/server.crt
cp /path/to/your.key nginx/ssl/server.key

# 2. 设置权限
chmod 600 nginx/ssl/server.key
chmod 644 nginx/ssl/server.crt

# 3. 按照"方式 1"的步骤 3-5 操作
```

---

## ⚡ 性能优化

### 1. Worker 进程优化

编辑 `nginx/nginx.conf`:

```nginx
# 自动检测 CPU 核心数
worker_processes auto;

# 增加连接数
events {
    worker_connections 2048;  # 从 1024 增加到 2048
}
```

### 2. 缓存优化

编辑 `nginx/conf.d/http.conf`，在 `location /` 块中添加:

```nginx
# 静态资源缓存（如果有）
location ~* \.(jpg|jpeg|png|gif|ico|css|js)$ {
    expires 7d;
    add_header Cache-Control "public, immutable";
}
```

### 3. 连接保持优化

```nginx
# 在 upstream 块中
upstream bearer_token_backend {
    server bearer-token-service:8080;
    keepalive 64;  # 从 32 增加到 64
    keepalive_requests 1000;
}
```

### 4. 限流配置

编辑 `nginx/nginx.conf`，在 `http` 块中添加:

```nginx
# 限制请求频率（每秒 10 个请求）
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
```

在 `nginx/conf.d/http.conf` 中应用:

```nginx
location /api/ {
    limit_req zone=api_limit burst=20 nodelay;
    proxy_pass http://bearer_token_backend;
    # ... 其他配置
}
```

---

## 📊 监控与日志

### 1. 查看实时日志

```bash
# Nginx 容器日志
docker compose logs -f nginx

# 访问日志
tail -f nginx/logs/access.log

# 错误日志
tail -f nginx/logs/error.log

# HTTP 日志
tail -f nginx/logs/http-access.log

# HTTPS 日志（启用后）
tail -f nginx/logs/https-access.log
```

### 2. 日志分析

```bash
# 统计访问量前 10 的 IP
awk '{print $1}' nginx/logs/access.log | sort | uniq -c | sort -rn | head -10

# 统计状态码分布
awk '{print $9}' nginx/logs/access.log | sort | uniq -c | sort -rn

# 统计平均响应时间
awk '{sum+=$NF; count++} END {print sum/count}' nginx/logs/access.log
```

### 3. Nginx 状态监控

启用 Nginx stub_status 模块，编辑 `nginx/conf.d/http.conf`:

```nginx
# 添加 status 端点
location /nginx_status {
    stub_status on;
    access_log off;
    allow 127.0.0.1;
    deny all;
}
```

查看状态:
```bash
docker compose exec nginx curl http://localhost/nginx_status
```

---

## 🔍 故障排查

### 问题 1: Nginx 无法启动

```bash
# 1. 查看日志
docker compose logs nginx

# 2. 检查配置语法
docker compose exec nginx nginx -t

# 3. 常见错误:
# - 端口被占用: 修改 .env 中的 NGINX_HTTP_PORT/NGINX_HTTPS_PORT
# - 配置文件错误: 检查 nginx.conf 和 conf.d/*.conf
# - SSL 证书缺失: 检查 nginx/ssl/ 目录
```

### 问题 2: 502 Bad Gateway

```bash
# 1. 检查后端服务是否运行
docker compose ps bearer-token-service

# 2. 检查后端服务健康状态
curl http://localhost:8080/health

# 3. 检查网络连接
docker compose exec nginx ping bearer-token-service

# 4. 查看 Nginx 错误日志
docker compose logs nginx | grep error
```

### 问题 3: HTTPS 证书错误

```bash
# 1. 检查证书文件
ls -la nginx/ssl/

# 2. 验证证书
openssl x509 -in nginx/ssl/server.crt -text -noout

# 3. 验证私钥
openssl rsa -in nginx/ssl/server.key -check

# 4. 检查证书和私钥是否匹配
openssl x509 -noout -modulus -in nginx/ssl/server.crt | openssl md5
openssl rsa -noout -modulus -in nginx/ssl/server.key | openssl md5
# 两个输出应该相同
```

### 问题 4: 高延迟

```bash
# 1. 查看上游服务器响应时间
tail -f nginx/logs/access.log | grep -oP 'urt="\K[^"]*'

# 2. 检查后端服务性能
docker stats bearer-token-service

# 3. 优化 Nginx 配置
# - 增加 worker_connections
# - 启用 keepalive
# - 调整缓冲区大小
```

---

## 🛠️ 常用运维命令

```bash
# 重新加载配置（不停机）
docker compose exec nginx nginx -s reload

# 检查配置语法
docker compose exec nginx nginx -t

# 重启 Nginx
docker compose restart nginx

# 查看 Nginx 版本
docker compose exec nginx nginx -v

# 进入 Nginx 容器
docker compose exec nginx sh

# 查看 Nginx 进程
docker compose exec nginx ps aux | grep nginx
```

---

## 📚 配置示例

### 启用 IP 白名单

编辑 `nginx/conf.d/http.conf`:

```nginx
location /api/ {
    # IP 白名单
    allow 192.168.1.0/24;
    allow 10.0.0.0/8;
    deny all;

    proxy_pass http://bearer_token_backend;
    # ... 其他配置
}
```

### 启用 Basic Auth

```bash
# 1. 安装 htpasswd
apt-get install apache2-utils

# 2. 创建密码文件
htpasswd -c nginx/htpasswd admin

# 3. 在配置中启用
location /admin/ {
    auth_basic "Restricted Area";
    auth_basic_user_file /etc/nginx/htpasswd;
    proxy_pass http://bearer_token_backend;
}

# 4. 挂载密码文件到容器
# 在 docker-compose.yml 的 nginx volumes 中添加:
# - ./nginx/htpasswd:/etc/nginx/htpasswd:ro
```

### 启用 CORS

编辑 `nginx/conf.d/http.conf`，在 `location /` 块中添加:

```nginx
# CORS 配置
add_header 'Access-Control-Allow-Origin' '*' always;
add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS' always;
add_header 'Access-Control-Allow-Headers' 'Authorization, Content-Type, X-Qiniu-Date' always;

if ($request_method = 'OPTIONS') {
    return 204;
}
```

---

## 🔒 安全最佳实践

### 1. 隐藏 Nginx 版本

编辑 `nginx/nginx.conf`，在 `http` 块中添加:

```nginx
server_tokens off;
```

### 2. 限制请求大小

```nginx
client_max_body_size 10M;  # 限制上传文件大小
```

### 3. 防止缓冲区溢出

```nginx
client_body_buffer_size 1K;
client_header_buffer_size 1k;
large_client_header_buffers 2 1k;
```

### 4. 设置超时

```nginx
client_body_timeout 12;
client_header_timeout 12;
keepalive_timeout 15;
send_timeout 10;
```

---

## 📖 相关文档

- [Docker Compose 部署指南](./DOCKER_DEPLOY.md)
- [快速开始指南](./DOCKER_QUICKSTART.md)
- [配置说明](./CONFIG.md)
- [Nginx 官方文档](https://nginx.org/en/docs/)

---

**版本**: v1.0
**更新日期**: 2025-12-26
**适用版本**: Bearer Token Service V2 + Nginx 1.25
