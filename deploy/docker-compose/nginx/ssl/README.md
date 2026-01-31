# SSL 证书目录

## 说明

将你的 SSL 证书文件放在这个目录:

- `server.crt` - SSL 证书文件
- `server.key` - SSL 私钥文件

## 生成自签名证书（仅用于测试）

```bash
# 生成自签名证书（有效期 365 天）
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout nginx/ssl/server.key \
  -out nginx/ssl/server.crt \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=Test/CN=localhost"

# 设置权限
chmod 600 nginx/ssl/server.key
chmod 644 nginx/ssl/server.crt
```

## 使用 Let's Encrypt 证书（生产环境）

```bash
# 使用 certbot 获取证书
certbot certonly --standalone -d your-domain.com

# 复制证书到此目录
cp /etc/letsencrypt/live/your-domain.com/fullchain.pem nginx/ssl/server.crt
cp /etc/letsencrypt/live/your-domain.com/privkey.pem nginx/ssl/server.key
```

## 启用 HTTPS

1. 将证书文件放到此目录
2. 编辑 `nginx/conf.d/https.conf`，取消注释 HTTPS 配置
3. 修改 `server_name` 为你的域名
4. 重启 Nginx: `docker compose restart nginx`

## 安全提醒

⚠️ **重要**:
- 不要将私钥文件 (`server.key`) 提交到 Git 仓库
- 已在 `.gitignore` 中配置忽略此目录
- 确保私钥文件权限为 600
