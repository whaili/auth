# ========================================
# Bearer Token Service V2 - Makefile
# ========================================
# 简化的部署操作

.PHONY: help compile build push test clean \
	up-test up-prod down logs status \
	helm-deploy-test helm-deploy-prod helm-delete-test helm-delete-prod helm-status \
	package

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
IMAGE_REPO ?= aslan-spock-register.qiniu.io/miku-stream/bearer-token-service
GO = go
HELM_CHART = deploy/helm/bearer-token-service
COMPOSE_DIR = deploy/docker-compose
CREDENTIALS_FILE ?= $(CURDIR)/_cust/credentials.env

# Kubeconfig 配置（存放在 _cust 目录，不入库）
KUBECONFIG_TEST ?= $(CURDIR)/_cust/kubeconfig-test
KUBECONFIG_PROD ?= $(CURDIR)/_cust/kubeconfig-prod

# ========================================
# 帮助信息
# ========================================

help: ## 显示帮助信息
	@echo "$(COLOR_INFO)Bearer Token Service V2 - Makefile$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_WARNING)注意事项:$(COLOR_RESET)"
	@echo "  1. Helm 生产部署需要先配置 _cust/credentials.env（执行 'make check-credentials' 检查）"
	@echo "  2. 变更数据库密码需要删除 PVC 重新初始化"
	@echo ""
	@echo "$(COLOR_SUCCESS)编译构建:$(COLOR_RESET)"
	@echo "  $(COLOR_INFO)compile$(COLOR_RESET)           编译 Go 二进制文件"
	@echo "  $(COLOR_INFO)build$(COLOR_RESET)             构建 Docker 镜像"
	@echo "  $(COLOR_INFO)push$(COLOR_RESET)              推送镜像到仓库"
	@echo ""
	@echo "$(COLOR_SUCCESS)Docker Compose 部署:$(COLOR_RESET)"
	@echo "  $(COLOR_INFO)up-test$(COLOR_RESET)           启动测试环境（内置 MongoDB）"
	@echo "  $(COLOR_INFO)up-prod$(COLOR_RESET)           启动生产环境（外部 MongoDB）"
	@echo "  $(COLOR_INFO)down$(COLOR_RESET)              停止服务"
	@echo "  $(COLOR_INFO)logs$(COLOR_RESET)              查看日志"
	@echo "  $(COLOR_INFO)status$(COLOR_RESET)            查看状态"
	@echo ""
	@echo "$(COLOR_SUCCESS)Helm 部署 (K8s):$(COLOR_RESET)"
	@echo "  $(COLOR_INFO)helm-deploy-test$(COLOR_RESET)     部署到 K8s 测试环境"
	@echo "  $(COLOR_INFO)helm-deploy-prod$(COLOR_RESET)     部署到 K8s 生产环境"
	@echo "  $(COLOR_INFO)helm-delete-test$(COLOR_RESET)     删除测试环境"
	@echo "  $(COLOR_INFO)helm-delete-prod$(COLOR_RESET)     删除生产环境"
	@echo "  $(COLOR_INFO)helm-status$(COLOR_RESET)          查看 Helm 发布状态"
	@echo ""
	@echo "$(COLOR_SUCCESS)测试:$(COLOR_RESET)"
	@echo "  $(COLOR_INFO)test$(COLOR_RESET)              运行所有测试（默认: 本地环境）"
	@echo "  $(COLOR_INFO)test TEST_ENV=k8s-test$(COLOR_RESET)  测试 K8s 测试集群"
	@echo "  $(COLOR_INFO)test-api$(COLOR_RESET)          运行 API 集成测试"
	@echo "  $(COLOR_INFO)test-qconf$(COLOR_RESET)        测试 qconfapi 集成（使用有效 UID）"
	@echo ""
	@echo "$(COLOR_SUCCESS)配置检查:$(COLOR_RESET)"
	@echo "  $(COLOR_INFO)check-credentials$(COLOR_RESET) 检查凭据配置"
	@echo ""
	@echo "$(COLOR_SUCCESS)清理:$(COLOR_RESET)"
	@echo "  $(COLOR_INFO)clean$(COLOR_RESET)             清理构建产物"
	@echo ""

# ========================================
# 编译与构建
# ========================================

compile: ## 编译 Go 二进制文件
	@echo "$(COLOR_INFO)编译服务...$(COLOR_RESET)"
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags='-w -s' -o bin/tokenserv cmd/server/main.go
	@echo "$(COLOR_SUCCESS)编译完成: bin/tokenserv$(COLOR_RESET)"
	@ls -lh bin/tokenserv

build: compile ## 构建 Docker 镜像
	@echo "$(COLOR_INFO)构建 Docker 镜像...$(COLOR_RESET)"
	docker build -t $(PROJECT_NAME):$(VERSION) -f Dockerfile .
	@echo "$(COLOR_SUCCESS)镜像构建完成: $(PROJECT_NAME):$(VERSION)$(COLOR_RESET)"

push: build ## 推送镜像到仓库
	@echo "$(COLOR_INFO)推送镜像到仓库...$(COLOR_RESET)"
	docker tag $(PROJECT_NAME):$(VERSION) $(IMAGE_REPO):$(VERSION)
	docker push $(IMAGE_REPO):$(VERSION)
	@echo "$(COLOR_SUCCESS)镜像推送完成: $(IMAGE_REPO):$(VERSION)$(COLOR_RESET)"

# ========================================
# Docker Compose 部署
# ========================================

up-test: ## 启动测试环境（内置 MongoDB + Redis）
	@echo "$(COLOR_INFO)启动测试环境...$(COLOR_RESET)"
	cd $(COMPOSE_DIR) && ./docker-compose-legacy-deploy.sh test

up-prod: ## 启动生产环境（外部 MongoDB）
	@echo "$(COLOR_INFO)启动生产环境...$(COLOR_RESET)"
	cd $(COMPOSE_DIR) && ./docker-compose-legacy-deploy.sh prod

down: ## 停止服务
	@echo "$(COLOR_WARNING)停止服务...$(COLOR_RESET)"
	cd $(COMPOSE_DIR) && docker-compose -f docker-compose.legacy.yml down

logs: ## 查看日志
	cd $(COMPOSE_DIR) && docker-compose -f docker-compose.legacy.yml logs -f

status: ## 查看服务状态
	cd $(COMPOSE_DIR) && docker-compose -f docker-compose.legacy.yml ps

health: ## 健康检查
	@curl -s http://localhost:8080/health || curl -s http://localhost/health

# ========================================
# Helm 部署 (K8s)
# ========================================

# 命名空间配置
HELM_NAMESPACE_TEST = bearer-token-test
HELM_NAMESPACE_PROD = bearer-token
HELM_RELEASE = bearer-token

# 确保命名空间存在且有 Helm 标签（解决已存在命名空间的兼容问题）
define ensure-helm-namespace
	@if KUBECONFIG=$(1) kubectl get namespace $(HELM_NAMESPACE) >/dev/null 2>&1; then \
		echo "$(COLOR_INFO)命名空间 $(HELM_NAMESPACE) 已存在，添加 Helm 标签...$(COLOR_RESET)"; \
		KUBECONFIG=$(1) kubectl label namespace $(HELM_NAMESPACE) app.kubernetes.io/managed-by=Helm --overwrite >/dev/null; \
		KUBECONFIG=$(1) kubectl annotate namespace $(HELM_NAMESPACE) meta.helm.sh/release-name=$(HELM_RELEASE) meta.helm.sh/release-namespace=$(HELM_NAMESPACE) --overwrite >/dev/null; \
	else \
		echo "$(COLOR_INFO)创建命名空间 $(HELM_NAMESPACE)...$(COLOR_RESET)"; \
		KUBECONFIG=$(1) kubectl create namespace $(HELM_NAMESPACE); \
		KUBECONFIG=$(1) kubectl label namespace $(HELM_NAMESPACE) app.kubernetes.io/managed-by=Helm; \
		KUBECONFIG=$(1) kubectl annotate namespace $(HELM_NAMESPACE) meta.helm.sh/release-name=$(HELM_RELEASE) meta.helm.sh/release-namespace=$(HELM_NAMESPACE); \
	fi
endef

helm-deploy-test: ## 部署到 K8s 测试环境（内置 MongoDB + Redis）
	@if [ ! -f "$(KUBECONFIG_TEST)" ]; then \
		echo "$(COLOR_ERROR)错误: kubeconfig 不存在: $(KUBECONFIG_TEST)$(COLOR_RESET)"; \
		exit 1; \
	fi
	@echo "$(COLOR_INFO)部署到 K8s 测试环境...$(COLOR_RESET)"
	KUBECONFIG=$(KUBECONFIG_TEST) helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) \
		-f $(HELM_CHART)/values-test.yaml \
		-n $(HELM_NAMESPACE_TEST)
	@echo "$(COLOR_SUCCESS)测试环境部署完成$(COLOR_RESET)"
	@$(MAKE) helm-status-test

helm-deploy-prod: ## 部署到 K8s 生产环境（外部 MongoDB）
	@if [ ! -f "$(KUBECONFIG_PROD)" ]; then \
		echo "$(COLOR_ERROR)错误: kubeconfig 不存在: $(KUBECONFIG_PROD)$(COLOR_RESET)"; \
		exit 1; \
	fi
	@if [ ! -f "$(CREDENTIALS_FILE)" ]; then \
		echo "$(COLOR_ERROR)错误: 凭据文件不存在: $(CREDENTIALS_FILE)$(COLOR_RESET)"; \
		echo "请先创建 _cust/credentials.env（可从 credentials.env.example 复制）"; \
		exit 1; \
	fi
	@echo "$(COLOR_WARNING)即将部署到生产环境！$(COLOR_RESET)"
	@echo "$(COLOR_INFO)从 $(CREDENTIALS_FILE) 加载凭据...$(COLOR_RESET)"
	@bash -c 'source $(CREDENTIALS_FILE) && \
		KUBECONFIG=$(KUBECONFIG_PROD) helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) \
			-f $(HELM_CHART)/values-prod.yaml \
			--set externalMongodb.uri="$${PROD_MONGO_URI}" \
			--set externalMysql.password="$${PROD_MYSQL_PASSWORD}" \
			--set externalRedis.password="$${REDIS_PASSWORD:-}" \
			-n $(HELM_NAMESPACE_PROD)'
	@echo "$(COLOR_SUCCESS)生产环境部署完成$(COLOR_RESET)"
	@$(MAKE) helm-status-prod

helm-delete-test: ## 删除测试环境
	@echo "$(COLOR_WARNING)删除测试环境...$(COLOR_RESET)"
	KUBECONFIG=$(KUBECONFIG_TEST) helm uninstall $(HELM_RELEASE) -n $(HELM_NAMESPACE_TEST) --ignore-not-found
	@echo "$(COLOR_SUCCESS)测试环境已删除$(COLOR_RESET)"

helm-delete-prod: ## 删除生产环境
	@echo "$(COLOR_ERROR)警告: 即将删除生产环境！$(COLOR_RESET)"
	@read -p "确认删除生产环境? [y/N] " confirm; \
	if [ "$$confirm" = "y" ]; then \
		KUBECONFIG=$(KUBECONFIG_PROD) helm uninstall $(HELM_RELEASE) -n $(HELM_NAMESPACE_PROD) --ignore-not-found; \
		echo "$(COLOR_SUCCESS)生产环境已删除$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_INFO)已取消$(COLOR_RESET)"; \
	fi

helm-status: ## 查看 Helm 发布状态
	@echo "$(COLOR_INFO)Helm 发布状态:$(COLOR_RESET)"
	@echo "$(COLOR_INFO)[测试环境]$(COLOR_RESET)"
	@KUBECONFIG=$(KUBECONFIG_TEST) helm list -n $(HELM_NAMESPACE_TEST) 2>/dev/null | grep $(HELM_RELEASE) || echo "  $(COLOR_WARNING)无$(COLOR_RESET)"
	@echo "$(COLOR_INFO)[生产环境]$(COLOR_RESET)"
	@KUBECONFIG=$(KUBECONFIG_PROD) helm list -n $(HELM_NAMESPACE_PROD) 2>/dev/null | grep $(HELM_RELEASE) || echo "  $(COLOR_WARNING)无$(COLOR_RESET)"

helm-status-test: ## 查看测试环境状态
	@echo "$(COLOR_INFO)测试环境状态:$(COLOR_RESET)"
	@KUBECONFIG=$(KUBECONFIG_TEST) kubectl get all -n $(HELM_NAMESPACE_TEST) 2>/dev/null || echo "$(COLOR_WARNING)命名空间不存在$(COLOR_RESET)"

helm-status-prod: ## 查看生产环境状态
	@echo "$(COLOR_INFO)生产环境状态:$(COLOR_RESET)"
	@KUBECONFIG=$(KUBECONFIG_PROD) kubectl get all -n $(HELM_NAMESPACE_PROD) 2>/dev/null || echo "$(COLOR_WARNING)命名空间不存在$(COLOR_RESET)"

helm-port-forward-test: ## 端口转发测试环境
	@echo "$(COLOR_INFO)转发测试环境端口到 localhost:8080...$(COLOR_RESET)"
	KUBECONFIG=$(KUBECONFIG_TEST) kubectl port-forward svc/$(HELM_RELEASE)-bearer-token-service 8080:8080 -n $(HELM_NAMESPACE_TEST)

helm-port-forward-prod: ## 端口转发生产环境
	@echo "$(COLOR_INFO)转发生产环境端口到 localhost:8080...$(COLOR_RESET)"
	KUBECONFIG=$(KUBECONFIG_PROD) kubectl port-forward svc/$(HELM_RELEASE)-bearer-token-service 8080:8080 -n $(HELM_NAMESPACE_PROD)

helm-package: ## 打包 Helm Chart（用于 KubeSphere 上传）
	@echo "$(COLOR_INFO)打包 Helm Chart...$(COLOR_RESET)"
	@mkdir -p dist
	helm package $(HELM_CHART) -d dist/
	@echo "$(COLOR_SUCCESS)打包完成:$(COLOR_RESET)"
	@ls -lh dist/*.tgz | tail -1

helm-package-test: ## 打包测试环境 Helm Chart
	@echo "$(COLOR_INFO)打包测试环境 Helm Chart...$(COLOR_RESET)"
	@mkdir -p dist
	@# 临时替换 values.yaml 为测试配置
	@cp $(HELM_CHART)/values.yaml $(HELM_CHART)/values.yaml.bak
	@cp $(HELM_CHART)/values-test.yaml $(HELM_CHART)/values.yaml
	helm package $(HELM_CHART) -d dist/ --version $$(grep '^version:' $(HELM_CHART)/Chart.yaml | awk '{print $$2}')-test
	@mv $(HELM_CHART)/values.yaml.bak $(HELM_CHART)/values.yaml
	@echo "$(COLOR_SUCCESS)测试环境打包完成:$(COLOR_RESET)"
	@ls -lh dist/*-test.tgz | tail -1

helm-package-prod: ## 打包生产环境 Helm Chart
	@echo "$(COLOR_INFO)打包生产环境 Helm Chart...$(COLOR_RESET)"
	@mkdir -p dist
	@# 临时替换 values.yaml 为生产配置
	@cp $(HELM_CHART)/values.yaml $(HELM_CHART)/values.yaml.bak
	@cp $(HELM_CHART)/values-prod.yaml $(HELM_CHART)/values.yaml
	helm package $(HELM_CHART) -d dist/ --version $$(grep '^version:' $(HELM_CHART)/Chart.yaml | awk '{print $$2}')-prod
	@mv $(HELM_CHART)/values.yaml.bak $(HELM_CHART)/values.yaml
	@echo "$(COLOR_SUCCESS)生产环境打包完成:$(COLOR_RESET)"
	@ls -lh dist/*-prod.tgz | tail -1

# ========================================
# 配置检查
# ========================================

check-credentials: ## 检查凭据配置
	@echo "$(COLOR_INFO)检查凭据配置...$(COLOR_RESET)"
	@if [ -f "$(CREDENTIALS_FILE)" ]; then \
		echo "$(COLOR_SUCCESS)✓ 凭据文件存在: $(CREDENTIALS_FILE)$(COLOR_RESET)"; \
		bash -c 'source $(CREDENTIALS_FILE) && \
			echo "$(COLOR_INFO)配置项检查:$(COLOR_RESET)" && \
			echo "  PROD_MYSQL_HOST: $${PROD_MYSQL_HOST:-未设置}" && \
			echo "  TEST_MYSQL_HOST: $${TEST_MYSQL_HOST:-未设置}" && \
			echo "  PROD_MONGO_HOSTS: $${PROD_MONGO_HOSTS:-未设置}" && \
			echo "  SSH_PROXY_HOST: $${SSH_PROXY_HOST:-未设置}"'; \
	else \
		echo "$(COLOR_ERROR)✗ 凭据文件不存在: $(CREDENTIALS_FILE)$(COLOR_RESET)"; \
		echo "$(COLOR_WARNING)请执行以下命令创建:$(COLOR_RESET)"; \
		echo "  cp _cust/credentials.env.example _cust/credentials.env"; \
		echo "  vim _cust/credentials.env  # 填入真实凭据"; \
		exit 1; \
	fi

# ========================================
# 测试
# ========================================

# 测试环境配置
# TEST_ENV=local (默认): 本地 Docker Compose 环境 (localhost:8081)
# TEST_ENV=k8s-test: K8s 测试集群 (bearer-token-test.jfcs-k8s-qa1.qiniu.io)
TEST_ENV ?= local
ifeq ($(TEST_ENV),k8s-test)
	TEST_BASE_URL = http://bearer-token-test.jfcs-k8s-qa1.qiniu.io
	TEST_ENV_NAME = K8s 测试集群
else ifeq ($(TEST_ENV),local)
	TEST_BASE_URL = http://localhost:8081
	TEST_ENV_NAME = 本地 Docker Compose
else
	TEST_BASE_URL =
	TEST_ENV_NAME = 未知环境
endif

test: ## 运行所有测试（单元测试 + API 测试）。用法: make test [TEST_ENV=local|k8s-test]
	@if [ "$(TEST_ENV)" != "local" ] && [ "$(TEST_ENV)" != "k8s-test" ]; then \
		echo "$(COLOR_ERROR)错误: 无效的 TEST_ENV: $(TEST_ENV)$(COLOR_RESET)"; \
		echo "$(COLOR_INFO)有效值: local, k8s-test$(COLOR_RESET)"; \
		exit 1; \
	fi
	@echo "$(COLOR_INFO)运行测试...$(COLOR_RESET)"
	@echo "$(COLOR_INFO)测试环境: $(TEST_ENV_NAME)$(COLOR_RESET)"
	@echo "$(COLOR_INFO)测试地址: $(TEST_BASE_URL)$(COLOR_RESET)"
	@echo ""
	@$(MAKE) test-unit
	@$(MAKE) test-api TEST_ENV=$(TEST_ENV)

test-unit: ## 运行单元测试
	@echo "$(COLOR_INFO)运行单元测试...$(COLOR_RESET)"
	$(GO) test -v ./handlers/... ./service/... ./repository/... 2>&1 | tee test-output.log
	@if grep -q "FAIL" test-output.log; then \
		echo "$(COLOR_ERROR)✗ 单元测试失败$(COLOR_RESET)"; \
		rm -f test-output.log; \
		exit 1; \
	fi
	@rm -f test-output.log
	@echo "$(COLOR_SUCCESS)✓ 单元测试通过$(COLOR_RESET)"

test-api: ## 运行 API 集成测试。用法: make test-api [TEST_ENV=local|k8s-test]
	@echo "$(COLOR_INFO)运行 API 集成测试...$(COLOR_RESET)"
	@echo "$(COLOR_INFO)测试环境: $(TEST_ENV_NAME)$(COLOR_RESET)"
	@echo "$(COLOR_INFO)测试地址: $(TEST_BASE_URL)$(COLOR_RESET)"
	@BASE_URL=$(TEST_BASE_URL) bash tests/api/test_qstub_api.sh
	@echo "$(COLOR_SUCCESS)✓ API 集成测试通过$(COLOR_RESET)"

test-qconf: ## 测试 qconfapi 集成（使用有效测试 UID）。用法: make test-qconf [TEST_ENV=local|k8s-test]
	@echo "$(COLOR_INFO)测试 qconfapi 集成...$(COLOR_RESET)"
	@echo "$(COLOR_INFO)测试环境: $(TEST_ENV_NAME)$(COLOR_RESET)"
	@echo "$(COLOR_INFO)测试地址: $(TEST_BASE_URL)$(COLOR_RESET)"
	@echo "$(COLOR_INFO)使用测试 UID: 1810810692$(COLOR_RESET)"
	@bash -c '\
		BASE_URL="$(TEST_BASE_URL)"; \
		TEST_UID="1810810692"; \
		echo "创建 token..."; \
		RESPONSE=$$(curl -s -X POST "$$BASE_URL/api/v2/tokens" \
			-H "Content-Type: application/json" \
			-H "Authorization: QiniuStub uid=$$TEST_UID&ut=1" \
			-d "{\"description\":\"qconf test\",\"expires_at\":\"$$(date -u -d \"+1 hour\" +%Y-%m-%dT%H:%M:%SZ)\"}"); \
		TOKEN_ID=$$(echo "$$RESPONSE" | jq -r ".token_id"); \
		BEARER_TOKEN=$$(echo "$$RESPONSE" | jq -r ".token"); \
		if [ "$$TOKEN_ID" = "null" ]; then echo "$(COLOR_ERROR)✗ 创建 token 失败$(COLOR_RESET)"; exit 1; fi; \
		echo "$(COLOR_SUCCESS)✓ Token 创建成功: $$TOKEN_ID$(COLOR_RESET)"; \
		echo "验证 token 并获取用户信息..."; \
		VALIDATE_RESPONSE=$$(curl -s -X POST "$$BASE_URL/api/v2/validateu" \
			-H "Authorization: Bearer $$BEARER_TOKEN"); \
		USER_INFO=$$(echo "$$VALIDATE_RESPONSE" | jq -r ".token_info.user_info"); \
		if [ "$$USER_INFO" != "null" ]; then \
			echo "$(COLOR_SUCCESS)✓ qconfapi 集成测试通过！$(COLOR_RESET)"; \
			echo ""; \
			echo "用户信息:"; \
			echo "$$VALIDATE_RESPONSE" | jq ".token_info.user_info"; \
		else \
			echo "$(COLOR_ERROR)✗ user_info 为 null$(COLOR_RESET)"; \
			exit 1; \
		fi; \
		curl -s -X DELETE "$$BASE_URL/api/v2/tokens/$$TOKEN_ID" \
			-H "Authorization: QiniuStub uid=$$TEST_UID&ut=1" > /dev/null; \
		echo "$(COLOR_SUCCESS)✓ 清理完成$(COLOR_RESET)"; \
	'

# ========================================
# 清理
# ========================================
# ========================================
# 清理
# ========================================

clean: ## 清理构建产物
	@echo "$(COLOR_WARNING)清理构建产物...$(COLOR_RESET)"
	rm -rf bin/
	rm -rf dist/*.tar
	@echo "$(COLOR_SUCCESS)清理完成$(COLOR_RESET)"

# ========================================
# 打包
# ========================================

package: build ## 打包部署文件
	@echo "$(COLOR_INFO)打包部署文件...$(COLOR_RESET)"
	@mkdir -p dist
	@# 导出镜像
	docker save $(PROJECT_NAME):$(VERSION) -o dist/bearer-token-service-$(VERSION).tar
	@# 打包 Helm Chart
	helm package $(HELM_CHART) -d dist/
	@echo "$(COLOR_SUCCESS)打包完成:$(COLOR_RESET)"
	@ls -lh dist/bearer-token-service-*
