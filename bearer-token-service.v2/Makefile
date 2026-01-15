# ========================================
# Bearer Token Service V2 - Makefile
# ========================================
# 简化 Docker Compose 操作

.PHONY: help build compile up down restart logs status clean test deploy backup restore package

# 默认目标
.DEFAULT_GOAL := help

# 颜色定义
COLOR_RESET   = \033[0m
COLOR_INFO    = \033[36m
COLOR_SUCCESS = \033[32m
COLOR_WARNING = \033[33m
COLOR_ERROR   = \033[31m

# 配置
PROJECT_NAME = bearer-token-service
VERSION ?= latest
# 自动检测 docker compose 或 docker-compose
COMPOSE := $(shell docker compose version >/dev/null 2>&1 && echo "docker compose" || echo "docker-compose")
GO = go

# ========================================
# 帮助信息
# ========================================

help: ## 显示帮助信息
	@echo "$(COLOR_INFO)Bearer Token Service V2 - Makefile$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_SUCCESS)可用命令:$(COLOR_RESET)"
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_-]+:.*?##/ { printf "  $(COLOR_INFO)%-15s$(COLOR_RESET) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(COLOR_WARNING)示例:$(COLOR_RESET)"
	@echo "  make compile      # 编译 Go 二进制"
	@echo "  make build        # 构建 Docker 镜像"
	@echo "  make up           # 启动服务"
	@echo "  make logs         # 查看日志"
	@echo "  make restart      # 重启服务"
	@echo ""

# ========================================
# 编译与构建
# ========================================

compile: ## 编译 Go 二进制文件
	@echo "$(COLOR_INFO)编译服务...$(COLOR_RESET)"
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags='-w -s' -o bin/tokenserv cmd/server/main.go
	@echo "$(COLOR_SUCCESS)✅ 编译完成: bin/tokenserv$(COLOR_RESET)"
	@ls -lh bin/tokenserv

build: compile ## 构建 Docker 镜像
	@echo "$(COLOR_INFO)构建 Docker 镜像...$(COLOR_RESET)"
	$(COMPOSE) build --no-cache
	@echo "$(COLOR_SUCCESS)✅ 镜像构建完成$(COLOR_RESET)"

build-quick: ## 快速构建（使用缓存）
	@echo "$(COLOR_INFO)快速构建 Docker 镜像...$(COLOR_RESET)"
	$(COMPOSE) build
	@echo "$(COLOR_SUCCESS)✅ 快速构建完成$(COLOR_RESET)"

# ========================================
# 服务管理
# ========================================

up: ## 启动所有服务
	@echo "$(COLOR_INFO)启动服务...$(COLOR_RESET)"
	$(COMPOSE) up -d
	@echo "$(COLOR_SUCCESS)✅ 服务已启动$(COLOR_RESET)"
	@$(MAKE) status

down: ## 停止所有服务
	@echo "$(COLOR_WARNING)停止服务...$(COLOR_RESET)"
	$(COMPOSE) down
	@echo "$(COLOR_SUCCESS)✅ 服务已停止$(COLOR_RESET)"

restart: ## 重启服务
	@echo "$(COLOR_INFO)重启服务...$(COLOR_RESET)"
	$(COMPOSE) restart
	@echo "$(COLOR_SUCCESS)✅ 服务已重启$(COLOR_RESET)"
	@$(MAKE) status

stop: ## 停止服务（不删除容器）
	@echo "$(COLOR_WARNING)停止服务...$(COLOR_RESET)"
	$(COMPOSE) stop
	@echo "$(COLOR_SUCCESS)✅ 服务已停止$(COLOR_RESET)"

start: ## 启动已有容器
	@echo "$(COLOR_INFO)启动服务...$(COLOR_RESET)"
	$(COMPOSE) start
	@echo "$(COLOR_SUCCESS)✅ 服务已启动$(COLOR_RESET)"

# ========================================
# 监控与日志
# ========================================

logs: ## 查看所有服务日志
	$(COMPOSE) logs -f

logs-service: ## 查看服务日志
	$(COMPOSE) logs -f $(PROJECT_NAME)

logs-mongo: ## 查看 MongoDB 日志
	$(COMPOSE) logs -f mongodb

logs-nginx: ## 查看 Nginx 日志
	$(COMPOSE) logs -f nginx

status: ## 查看服务状态
	@echo "$(COLOR_INFO)服务状态:$(COLOR_RESET)"
	@$(COMPOSE) ps
	@echo ""
	@echo "$(COLOR_INFO)健康检查:$(COLOR_RESET)"
	@curl -s http://localhost:8081/health | jq . || echo "$(COLOR_ERROR)❌ 服务未就绪$(COLOR_RESET)"

stats: ## 查看资源使用
	docker stats --no-stream

ps: ## 查看容器状态
	$(COMPOSE) ps

# ========================================
# 开发与调试
# ========================================

shell: ## 进入服务容器 shell
	$(COMPOSE) exec $(PROJECT_NAME) sh

shell-mongo: ## 进入 MongoDB shell
	$(COMPOSE) exec mongodb mongosh -u admin -p changeme

shell-nginx: ## 进入 Nginx shell
	$(COMPOSE) exec nginx sh

exec: ## 在服务容器中执行命令 (用法: make exec CMD="ls -la")
	$(COMPOSE) exec $(PROJECT_NAME) $(CMD)

config: ## 查看配置（验证环境变量）
	$(COMPOSE) config

# ========================================
# 测试
# ========================================

test: ## 运行 API 测试（自动检查和启动测试环境）
	@echo "$(COLOR_INFO)========================================$(COLOR_RESET)"
	@echo "$(COLOR_INFO)准备测试环境...$(COLOR_RESET)"
	@echo "$(COLOR_INFO)========================================$(COLOR_RESET)"
	@echo ""
	@# 1. 检查测试脚本是否存在
	@if [ ! -f tests/test_qstub_api.sh ]; then \
		echo "$(COLOR_ERROR)❌ 测试脚本不存在: tests/test_qstub_api.sh$(COLOR_RESET)"; \
		exit 1; \
	fi
	@# 2. 检查 MongoDB 容器是否运行
	@echo "$(COLOR_INFO)检查 MongoDB 容器...$(COLOR_RESET)"
	@if ! sudo docker ps | grep -q bearer-token-mongodb; then \
		echo "$(COLOR_WARNING)⚠️  MongoDB 容器未运行，正在启动...$(COLOR_RESET)"; \
		sudo docker start bearer-token-mongodb 2>/dev/null || \
		(echo "$(COLOR_ERROR)❌ 无法启动 MongoDB 容器，请先运行: docker-compose up -d mongodb$(COLOR_RESET)" && exit 1); \
		sleep 3; \
	fi
	@echo "$(COLOR_SUCCESS)✅ MongoDB 容器运行中$(COLOR_RESET)"
	@# 3. 检查二进制文件是否存在
	@echo "$(COLOR_INFO)检查服务二进制文件...$(COLOR_RESET)"
	@if [ ! -f bin/tokenserv ]; then \
		echo "$(COLOR_WARNING)⚠️  二进制文件不存在，正在编译...$(COLOR_RESET)"; \
		$(MAKE) compile; \
	fi
	@echo "$(COLOR_SUCCESS)✅ 二进制文件存在$(COLOR_RESET)"
	@# 4. 检查服务是否运行
	@echo "$(COLOR_INFO)检查服务状态...$(COLOR_RESET)"
	@if ! curl -s http://localhost:8081/health > /dev/null 2>&1; then \
		echo "$(COLOR_WARNING)⚠️  服务未运行，正在启动本地服务...$(COLOR_RESET)"; \
		rm -f $(CURDIR)/tokenserv_test.log; \
		nohup ./tests/start_local.sh > $(CURDIR)/tokenserv_test.log 2>&1 & \
		echo "$(COLOR_INFO)等待服务启动...$(COLOR_RESET)"; \
		for i in 1 2 3 4 5; do \
			sleep 1; \
			if curl -s http://localhost:8081/health > /dev/null 2>&1; then \
				break; \
			fi; \
		done; \
		if ! curl -s http://localhost:8081/health > /dev/null 2>&1; then \
			echo "$(COLOR_ERROR)❌ 服务启动失败，请检查日志: $(CURDIR)/tokenserv_test.log$(COLOR_RESET)"; \
			tail -20 $(CURDIR)/tokenserv_test.log; \
			exit 1; \
		fi; \
	fi
	@echo "$(COLOR_SUCCESS)✅ 服务运行中 (http://localhost:8081)$(COLOR_RESET)"
	@echo ""
	@# 5. 运行测试
	@echo "$(COLOR_INFO)========================================$(COLOR_RESET)"
	@echo "$(COLOR_INFO)开始运行测试...$(COLOR_RESET)"
	@echo "$(COLOR_INFO)========================================$(COLOR_RESET)"
	@echo ""
	@cd tests && bash test_qstub_api.sh
	@echo ""
	@echo "$(COLOR_SUCCESS)========================================$(COLOR_RESET)"
	@echo "$(COLOR_SUCCESS)✅ 测试完成$(COLOR_RESET)"
	@echo "$(COLOR_SUCCESS)========================================$(COLOR_RESET)"

test-stop: ## 停止测试服务
	@echo "$(COLOR_INFO)停止测试服务...$(COLOR_RESET)"
	@pkill -f "bin/tokenserv" 2>/dev/null || true
	@echo "$(COLOR_SUCCESS)✅ 测试服务已停止$(COLOR_RESET)"

test-clean: ## 清理测试环境和数据
	@echo "$(COLOR_WARNING)清理测试环境...$(COLOR_RESET)"
	@# 停止服务
	@pkill -f "bin/tokenserv" 2>/dev/null || true
	@# 清理测试数据库（可选）
	@echo "$(COLOR_INFO)如需清理 MongoDB 测试数据，请手动执行:$(COLOR_RESET)"
	@echo "  sudo docker exec bearer-token-mongodb mongosh -u admin -p 123456 --eval 'use token_service_v2; db.dropDatabase()'"
	@echo "$(COLOR_SUCCESS)✅ 测试环境已清理$(COLOR_RESET)"

health: ## 健康检查
	@echo "$(COLOR_INFO)健康检查...$(COLOR_RESET)"
	@curl -s http://localhost:8081/health | jq . && \
		echo "$(COLOR_SUCCESS)✅ 服务健康$(COLOR_RESET)" || \
		echo "$(COLOR_ERROR)❌ 服务异常$(COLOR_RESET)"

nginx-test: ## 测试 Nginx 配置
	@echo "$(COLOR_INFO)测试 Nginx 配置...$(COLOR_RESET)"
	$(COMPOSE) exec nginx nginx -t

nginx-reload: ## 重新加载 Nginx 配置
	@echo "$(COLOR_INFO)重新加载 Nginx...$(COLOR_RESET)"
	$(COMPOSE) exec nginx nginx -s reload
	@echo "$(COLOR_SUCCESS)✅ Nginx 已重新加载$(COLOR_RESET)"

ssl-setup: ## 设置 SSL 证书
	@bash scripts/setup-ssl.sh

# ========================================
# 部署
# ========================================

deploy: build up ## 完整部署（编译 + 构建 + 启动）
	@echo "$(COLOR_SUCCESS)✅ 部署完成$(COLOR_RESET)"
	@$(MAKE) health

redeploy: down deploy ## 重新部署（停止 + 部署）
	@echo "$(COLOR_SUCCESS)✅ 重新部署完成$(COLOR_RESET)"

update: compile build-quick restart ## 更新服务（快速）
	@echo "$(COLOR_SUCCESS)✅ 服务已更新$(COLOR_RESET)"
	@$(MAKE) health

# ========================================
# 备份与恢复
# ========================================

backup: ## 备份 MongoDB 数据
	@echo "$(COLOR_INFO)备份数据库...$(COLOR_RESET)"
	@mkdir -p backup
	$(COMPOSE) exec -T mongodb mongodump \
		--uri="mongodb://admin:changeme@localhost:27017" \
		--archive=/tmp/backup-$(shell date +%Y%m%d-%H%M%S).archive
	@echo "$(COLOR_SUCCESS)✅ 备份完成$(COLOR_RESET)"

backup-export: ## 导出备份到宿主机
	@echo "$(COLOR_INFO)导出备份...$(COLOR_RESET)"
	@mkdir -p backup
	$(COMPOSE) exec -T mongodb mongodump \
		--uri="mongodb://admin:changeme@localhost:27017" \
		--archive | gzip > backup/mongodb-backup-$(shell date +%Y%m%d-%H%M%S).archive.gz
	@echo "$(COLOR_SUCCESS)✅ 备份已导出到 backup/ 目录$(COLOR_RESET)"

restore: ## 恢复数据库 (用法: make restore FILE=backup/file.archive.gz)
	@if [ -z "$(FILE)" ]; then \
		echo "$(COLOR_ERROR)错误: 请指定备份文件$(COLOR_RESET)"; \
		echo "用法: make restore FILE=backup/mongodb-backup-20251226.archive.gz"; \
		exit 1; \
	fi
	@echo "$(COLOR_WARNING)恢复数据库: $(FILE)$(COLOR_RESET)"
	gunzip -c $(FILE) | $(COMPOSE) exec -T mongodb mongorestore \
		--uri="mongodb://admin:changeme@localhost:27017" \
		--archive
	@echo "$(COLOR_SUCCESS)✅ 恢复完成$(COLOR_RESET)"

# ========================================
# 清理
# ========================================

clean: ## 清理容器（保留数据卷）
	@echo "$(COLOR_WARNING)清理容器...$(COLOR_RESET)"
	$(COMPOSE) down
	@echo "$(COLOR_SUCCESS)✅ 清理完成$(COLOR_RESET)"

clean-all: ## 清理所有资源（包括数据卷，危险！）
	@echo "$(COLOR_ERROR)⚠️  警告: 将删除所有数据！$(COLOR_RESET)"
	@read -p "确认删除? [y/N] " confirm; \
	if [ "$$confirm" = "y" ]; then \
		$(COMPOSE) down -v; \
		echo "$(COLOR_SUCCESS)✅ 所有资源已清理$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_INFO)已取消$(COLOR_RESET)"; \
	fi

clean-images: ## 清理镜像
	@echo "$(COLOR_WARNING)清理镜像...$(COLOR_RESET)"
	docker rmi $(PROJECT_NAME):$(VERSION) || true
	@echo "$(COLOR_SUCCESS)✅ 镜像已清理$(COLOR_RESET)"

prune: ## 清理 Docker 系统缓存
	@echo "$(COLOR_WARNING)清理 Docker 缓存...$(COLOR_RESET)"
	docker system prune -f
	@echo "$(COLOR_SUCCESS)✅ 缓存已清理$(COLOR_RESET)"

# ========================================
# 版本管理
# ========================================

version: ## 显示版本信息
	@echo "$(COLOR_INFO)版本信息:$(COLOR_RESET)"
	@echo "  项目: Bearer Token Service V2"
	@echo "  版本: $(VERSION)"
	@$(COMPOSE) version

tag: ## 打标签 (用法: make tag VERSION=v2.0.1)
	@if [ -z "$(VERSION)" ]; then \
		echo "$(COLOR_ERROR)错误: 请指定版本$(COLOR_RESET)"; \
		echo "用法: make tag VERSION=v2.0.1"; \
		exit 1; \
	fi
	docker tag $(PROJECT_NAME):latest $(PROJECT_NAME):$(VERSION)
	@echo "$(COLOR_SUCCESS)✅ 镜像已打标签: $(VERSION)$(COLOR_RESET)"

# ========================================
# 生产环境专用
# ========================================

prod-check: ## 生产环境检查清单
	@echo "$(COLOR_INFO)生产环境检查清单:$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_WARNING)1. 环境变量检查$(COLOR_RESET)"
	@grep -q "changeme" .env && \
		echo "  $(COLOR_ERROR)❌ .env 中仍使用默认密码！$(COLOR_RESET)" || \
		echo "  $(COLOR_SUCCESS)✅ 密码已修改$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_WARNING)2. 端口暴露检查$(COLOR_RESET)"
	@grep -q "27017:27017" docker-compose.yml && \
		echo "  $(COLOR_ERROR)❌ MongoDB 端口暴露到宿主机（不安全）$(COLOR_RESET)" || \
		echo "  $(COLOR_SUCCESS)✅ MongoDB 端口未暴露$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_WARNING)3. 资源限制检查$(COLOR_RESET)"
	@grep -q "deploy:" docker-compose.yml && \
		echo "  $(COLOR_SUCCESS)✅ 已配置资源限制$(COLOR_RESET)" || \
		echo "  $(COLOR_WARNING)⚠️  未配置资源限制$(COLOR_RESET)"
	@echo ""

prod-deploy: prod-check deploy ## 生产部署（含检查）
	@echo "$(COLOR_SUCCESS)✅ 生产部署完成$(COLOR_RESET)"

package: ## 打包生产部署文件（镜像 + 配置）
	@echo "$(COLOR_INFO)打包生产部署文件...$(COLOR_RESET)"
	@mkdir -p dist/deploy
	@echo ""
	@echo "$(COLOR_INFO)1. 导出 Docker 镜像...$(COLOR_RESET)"
	docker save $(PROJECT_NAME):$(VERSION) -o dist/bearer-token-service.tar
	@ls -lh dist/bearer-token-service.tar
	@echo ""
	@echo "$(COLOR_INFO)2. 复制配置文件...$(COLOR_RESET)"
	cp docker-compose.yml dist/deploy/docker-compose.yml
	@[ -f dist/deploy/docker-compose.legacy.yml ] && echo "  - docker-compose.legacy.yml (已存在)" || echo "  ⚠️  docker-compose.legacy.yml 不存在"
	@[ -f dist/deploy/docker-compose-legacy-deploy.sh ] && echo "  - docker-compose-legacy-deploy.sh (已存在)" || echo "  ⚠️  docker-compose-legacy-deploy.sh 不存在"
	cp .env.example dist/deploy/.env.example
	@echo "  - docker-compose.yml"
	@echo "  - .env.example"
	@echo ""
	@echo "$(COLOR_INFO)3. 复制 Nginx 配置...$(COLOR_RESET)"
	cp -r nginx dist/deploy/
	@echo "  - nginx/"
	@echo ""
	@echo "$(COLOR_INFO)4. 复制部署脚本...$(COLOR_RESET)"
	mkdir -p dist/deploy/scripts
	cp -r scripts/init dist/deploy/scripts/ 2>/dev/null || echo "  ⚠️  init 目录不存在，跳过"
	cp -r scripts/deploy/systemd dist/deploy/scripts/ 2>/dev/null || echo "  ⚠️  systemd 文件不存在，跳过"
	cp scripts/deploy/run-without-docker.sh dist/deploy/scripts/ 2>/dev/null || echo "  ⚠️  run-without-docker.sh 不存在，跳过"
	@echo "  - scripts/init/ (数据库索引初始化脚本)"
	@echo "  - scripts/systemd/ (systemd service 文件)"
	@echo "  - scripts/run-without-docker.sh (无 Docker 运行脚本)"
	@echo ""
	@echo "$(COLOR_INFO)5. 生成测试脚本...$(COLOR_RESET)"
	@rm -rf dist/deploy/tests
	@mkdir -p dist/deploy/tests
	@cp tests/test_qstub_api.sh dist/deploy/tests/
	@cp tests/test_redis_cache.sh dist/deploy/tests/
	@sed -i 's|http://localhost:8081|http://localhost|g' dist/deploy/tests/test_qstub_api.sh 2>/dev/null || true
	@sed -i 's|http://localhost:8081|http://localhost|g' dist/deploy/tests/test_redis_cache.sh 2>/dev/null || true
	@echo "  - tests/test_qstub_api.sh"
	@echo "  - tests/test_redis_cache.sh"
	@echo ""
	@echo "$(COLOR_INFO)6. 检查部署说明...$(COLOR_RESET)"
	@[ -f dist/deploy/DEPLOY.md ] && echo "  ✅ DEPLOY.md (已存在)" || echo "  ⚠️  DEPLOY.md 不存在，请创建"
	@[ -f dist/deploy/VM_TEST_CHECKLIST.md ] && echo "  ✅ VM_TEST_CHECKLIST.md (已存在)" || true
	@echo ""
	@echo "$(COLOR_INFO)7. 复制 Docker 镜像到部署目录...$(COLOR_RESET)"
	@cp dist/bearer-token-service.tar dist/deploy/
	@echo "  - bearer-token-service.tar"
	@echo ""
	@echo "$(COLOR_INFO)8. 打包为 tar.gz...$(COLOR_RESET)"
	cd dist && tar czf bearer-token-service-$(VERSION)-deploy.tar.gz deploy/
	@ls -lh dist/bearer-token-service-$(VERSION)-deploy.tar.gz
	@echo ""
	@echo "$(COLOR_SUCCESS)✅ 部署包已创建: dist/bearer-token-service-$(VERSION)-deploy.tar.gz$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_INFO)部署包内容:$(COLOR_RESET)"
	@tar tzf dist/bearer-token-service-$(VERSION)-deploy.tar.gz | head -20
	@echo ""
	@echo "$(COLOR_WARNING)传输到生产环境后，解压并按照 DEPLOY.md 说明部署$(COLOR_RESET)"

# ========================================
# 调试
# ========================================

debug: ## 启动服务（前台，查看详细日志）
	$(COMPOSE) up

inspect: ## 检查服务容器详细信息
	docker inspect $(PROJECT_NAME)

network: ## 查看网络配置
	docker network inspect bearer-token-service_bearer-token-net

volumes: ## 查看数据卷
	docker volume ls | grep bearer-token

# ========================================
# CI/CD
# ========================================

ci-build: compile build ## CI 构建

ci-test: up test down ## CI 测试

ci-deploy: prod-check deploy ## CI 部署
