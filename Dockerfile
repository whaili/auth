# ========================================
# Bearer Token Service V2 - Production Dockerfile
# ========================================
# 运行预编译的二进制文件（在 vm-test 中编译）

FROM aslan-spock-register.qiniu.io/miku-stream/test-alpine:latest

# 安装运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl

# 设置时区（可通过环境变量覆盖）
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# 创建必要的目录
RUN mkdir -p /app/logs && \
    chown -R appuser:appuser /app

# 切换到非 root 用户
USER appuser

WORKDIR /app

# 复制预编译的二进制文件（从 bin/ 目录）
COPY --chown=appuser:appuser bin/tokenserv /app/tokenserv

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:${PORT:-8080}/health || exit 1

# 暴露端口
EXPOSE 8080

# 运行服务
CMD ["/app/tokenserv"]
