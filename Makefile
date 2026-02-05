# ========================================
# Bearer Token Service V2 - Makefile
# ========================================
# 简化的部署操作

.PHONY: help compile build push test clean package

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

# ========================================
# 帮助信息
# ========================================

help: ## 显示帮助信息
	@echo "$(COLOR_INFO)Bearer Token Service V2 - Makefile$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_WARNING)注意: 所有部署操作已移至 deploy/scripts/deploy.sh$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_SUCCESS)编译构建:$(COLOR_RESET)"
	@echo "  $(COLOR_INFO)compile$(COLOR_RESET)    编译 Go 二进制文件"
	@echo "  $(COLOR_INFO)build$(COLOR_RESET)      构建 Docker 镜像"
	@echo "  $(COLOR_INFO)package$(COLOR_RESET)    打包部署文件"
	@echo "  $(COLOR_INFO)push$(COLOR_RESET)       推送镜像到仓库"
	@echo ""
	@echo "$(COLOR_SUCCESS)测试:$(COLOR_RESET)"
	@echo "  $(COLOR_INFO)test$(COLOR_RESET)       运行所有测试"
	@echo ""
	@echo "$(COLOR_SUCCESS)清理:$(COLOR_RESET)"
	@echo "  $(COLOR_INFO)clean$(COLOR_RESET)      清理构建产物"
	@echo ""
	@echo "$(COLOR_SUCCESS)部署（使用独立脚本）:$(COLOR_RESET)"
	@echo "  $(COLOR_INFO)./deploy/scripts/deploy.sh local start$(COLOR_RESET)       本地测试环境"
	@echo "  $(COLOR_INFO)./deploy/scripts/deploy.sh k8s-test deploy$(COLOR_RESET)   K8s 测试环境"
	@echo "  $(COLOR_INFO)./deploy/scripts/deploy.sh physical vmxs1$(COLOR_RESET)    物理服务器（生产）"
	@echo "  $(COLOR_INFO)./deploy/scripts/manage.sh local status$(COLOR_RESET)      查看状态/日志"
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
# 测试
# ========================================

test: ## 运行所有测试
	@echo "$(COLOR_INFO)运行测试...$(COLOR_RESET)"
	$(GO) test -v ./handlers/... ./service/... ./repository/...
	@echo "$(COLOR_SUCCESS)测试通过$(COLOR_RESET)"

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
	@echo ""
	@echo "$(COLOR_INFO)部署说明: 使用 ./deploy/scripts/deploy.sh$(COLOR_RESET)"

