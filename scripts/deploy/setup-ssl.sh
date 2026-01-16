#!/bin/bash

# ========================================
# SSL 证书设置脚本
# ========================================

set -e

echo "========================================="
echo "SSL 证书设置向导"
echo "========================================="
echo ""

# 检查 nginx 目录
if [ ! -d "nginx/ssl" ]; then
    echo "❌ 错误: nginx/ssl 目录不存在"
    echo "请确保在项目根目录下运行此脚本"
    exit 1
fi

# 选择证书类型
echo "请选择证书类型:"
echo "  1) 生成自签名证书（仅用于测试）"
echo "  2) 使用现有证书文件"
echo "  3) 使用 Let's Encrypt（需要域名）"
echo ""
read -p "请输入选项 (1-3): " choice

case $choice in
    1)
        echo ""
        echo "🔧 生成自签名证书..."
        echo ""

        read -p "请输入域名或 IP (默认: localhost): " domain
        domain=${domain:-localhost}

        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout nginx/ssl/server.key \
            -out nginx/ssl/server.crt \
            -subj "/C=CN/ST=Beijing/L=Beijing/O=Test/CN=$domain"

        chmod 600 nginx/ssl/server.key
        chmod 644 nginx/ssl/server.crt

        echo ""
        echo "✅ 自签名证书已生成"
        echo "   证书: nginx/ssl/server.crt"
        echo "   私钥: nginx/ssl/server.key"
        echo "   域名: $domain"
        echo ""
        echo "⚠️  注意: 自签名证书仅用于测试，浏览器会显示不安全警告"
        ;;

    2)
        echo ""
        echo "📁 使用现有证书..."
        echo ""

        read -p "请输入证书文件路径 (.crt): " cert_path
        read -p "请输入私钥文件路径 (.key): " key_path

        if [ ! -f "$cert_path" ]; then
            echo "❌ 证书文件不存在: $cert_path"
            exit 1
        fi

        if [ ! -f "$key_path" ]; then
            echo "❌ 私钥文件不存在: $key_path"
            exit 1
        fi

        cp "$cert_path" nginx/ssl/server.crt
        cp "$key_path" nginx/ssl/server.key
        chmod 600 nginx/ssl/server.key
        chmod 644 nginx/ssl/server.crt

        echo ""
        echo "✅ 证书文件已复制"
        ;;

    3)
        echo ""
        echo "🌐 使用 Let's Encrypt..."
        echo ""

        read -p "请输入域名: " domain

        if [ -z "$domain" ]; then
            echo "❌ 域名不能为空"
            exit 1
        fi

        # 检查 certbot
        if ! command -v certbot &> /dev/null; then
            echo "❌ certbot 未安装"
            echo "请先安装: sudo apt-get install certbot"
            exit 1
        fi

        echo ""
        echo "⚠️  注意事项:"
        echo "  1. 域名必须已解析到此服务器"
        echo "  2. 需要停止占用 80 端口的服务"
        echo "  3. 需要 root 权限"
        echo ""
        read -p "确认继续? (y/N): " confirm

        if [ "$confirm" != "y" ]; then
            echo "已取消"
            exit 0
        fi

        # 停止 nginx
        echo "停止 Nginx..."
        docker compose stop nginx 2>/dev/null || true

        # 获取证书
        echo "获取 Let's Encrypt 证书..."
        sudo certbot certonly --standalone -d "$domain"

        # 复制证书
        sudo cp "/etc/letsencrypt/live/$domain/fullchain.pem" nginx/ssl/server.crt
        sudo cp "/etc/letsencrypt/live/$domain/privkey.pem" nginx/ssl/server.key
        sudo chown $USER:$USER nginx/ssl/*
        chmod 600 nginx/ssl/server.key
        chmod 644 nginx/ssl/server.crt

        echo ""
        echo "✅ Let's Encrypt 证书已配置"
        echo "   域名: $domain"
        echo ""
        echo "📝 证书续期提示:"
        echo "   Let's Encrypt 证书有效期 90 天，需要定期续期"
        echo "   续期命令: sudo certbot renew"
        ;;

    *)
        echo "❌ 无效选项"
        exit 1
        ;;
esac

# 验证证书
echo ""
echo "🔍 验证证书..."
if openssl x509 -in nginx/ssl/server.crt -text -noout > /dev/null 2>&1; then
    echo "✅ 证书文件有效"

    # 显示证书信息
    echo ""
    echo "证书信息:"
    openssl x509 -in nginx/ssl/server.crt -noout -subject -dates
else
    echo "❌ 证书文件无效"
    exit 1
fi

# 询问是否启用 HTTPS
echo ""
read -p "是否现在启用 HTTPS 配置? (y/N): " enable_https

if [ "$enable_https" = "y" ]; then
    echo ""
    echo "📝 请手动编辑以下文件以启用 HTTPS:"
    echo "   1. nginx/conf.d/https.conf - 取消注释并修改 server_name"
    echo "   2. 可选: nginx/conf.d/http.conf - 添加 HTTP→HTTPS 重定向"
    echo ""
    echo "完成后运行: docker compose restart nginx"
fi

echo ""
echo "========================================="
echo "✅ SSL 证书设置完成！"
echo "========================================="
