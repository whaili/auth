#!/bin/bash
# 迁移测试环境到 Helm 管理
# 安全迁移：保留 MongoDB 数据（PVC）

set -e

KUBECONFIG_FILE="/root/src/auth/_cust/kubeconfig-test"
NAMESPACE="bearer-token-test"
RELEASE_NAME="bearer-token"
CHART_PATH="deploy/helm/bearer-token-service"
VALUES_FILE="deploy/helm/bearer-token-service/values-test.yaml"

# 加载凭据
if [ ! -f "_cust/credentials.env" ]; then
    echo "错误: 凭据文件不存在: _cust/credentials.env"
    echo "请先创建凭据文件"
    exit 1
fi

source _cust/credentials.env

if [ -z "$TEST_QCONF_SECRET_KEY" ] || [ -z "$TEST_MONGO_PASSWORD" ]; then
    echo "错误: 凭据未设置"
    echo "请在 _cust/credentials.env 中设置 TEST_QCONF_SECRET_KEY 和 TEST_MONGO_PASSWORD"
    exit 1
fi

cd /root/src/auth

echo "========================================"
echo "迁移到 Helm 管理"
echo "========================================"
echo ""
echo "此脚本将："
echo "  1. 备份当前所有资源"
echo "  2. 删除旧的 Deployment、Service、Ingress"
echo "  3. 保留 MongoDB PVC（数据不丢失）"
echo "  4. 使用 Helm 重新部署"
echo ""
read -p "确认继续？(yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "已取消"
    exit 0
fi

echo ""
echo "=== 1. 备份当前资源 ==="
BACKUP_DIR="/tmp/k8s-backup-bearer-token-test"
BACKUP_FILE="$BACKUP_DIR/backup-$(date +%Y%m%d-%H%M%S).yaml"
mkdir -p $BACKUP_DIR
kubectl --kubeconfig=$KUBECONFIG_FILE get all,ingress,configmap,secret,pvc -n $NAMESPACE -o yaml > $BACKUP_FILE
echo "✓ 备份已保存: $BACKUP_FILE"

echo ""
echo "=== 2. 获取 MongoDB 凭据 ==="
MONGO_USERNAME=$(kubectl --kubeconfig=$KUBECONFIG_FILE get secret mongodb-secret -n $NAMESPACE -o jsonpath='{.data.username}' | base64 -d)
echo "✓ MongoDB 用户名: $MONGO_USERNAME"

echo ""
echo "=== 3. 删除旧资源（保留 PVC 和 Secret） ==="
echo "删除 Deployments..."
kubectl --kubeconfig=$KUBECONFIG_FILE delete deployment bearer-token-service redis -n $NAMESPACE --ignore-not-found=true

echo "删除 StatefulSet（但保留 PVC）..."
kubectl --kubeconfig=$KUBECONFIG_FILE delete statefulset mongodb -n $NAMESPACE --cascade=orphan --ignore-not-found=true

echo "删除 Services..."
kubectl --kubeconfig=$KUBECONFIG_FILE delete service bearer-token-service redis mongodb -n $NAMESPACE --ignore-not-found=true

echo "删除 Ingress..."
kubectl --kubeconfig=$KUBECONFIG_FILE delete ingress bearer-token-service -n $NAMESPACE --ignore-not-found=true

echo "删除 ConfigMap（将由 Helm 重建）..."
kubectl --kubeconfig=$KUBECONFIG_FILE delete configmap bearer-token-config -n $NAMESPACE --ignore-not-found=true

echo "保留 Secret（mongodb-secret 和 bearer-token-secrets）"

echo ""
echo "等待 Pod 终止..."
sleep 5

echo ""
echo "=== 4. 使用 Helm 部署 ==="
helm --kubeconfig=$KUBECONFIG_FILE upgrade --install $RELEASE_NAME $CHART_PATH \
    -f $VALUES_FILE \
    -n $NAMESPACE \
    --create-namespace \
    --set image.tag=latest \
    --set mongodb.auth.username=$MONGO_USERNAME \
    --set mongodb.auth.password=$TEST_MONGO_PASSWORD \
    --set qconf.enabled=true \
    --set qconf.accessKey="-lnWyW53aRF1AYUj5D2oBBub377cTMYawZdOT25z" \
    --set qconf.secretKey=$TEST_QCONF_SECRET_KEY \
    --set qconf.masterHosts="http://kodo-dev.confg.jfcs-k8s-qa2.qiniu.io"

echo ""
echo "=== 5. 等待部署完成 ==="
kubectl --kubeconfig=$KUBECONFIG_FILE rollout status deployment/bearer-token-service -n $NAMESPACE --timeout=120s

echo ""
echo "=== 6. 验证部署 ==="
echo ""
echo "Helm Release:"
helm --kubeconfig=$KUBECONFIG_FILE list -n $NAMESPACE

echo ""
echo "Pods:"
kubectl --kubeconfig=$KUBECONFIG_FILE get pods -n $NAMESPACE

echo ""
echo "Services:"
kubectl --kubeconfig=$KUBECONFIG_FILE get svc -n $NAMESPACE

echo ""
echo "PVCs (检查 MongoDB 数据是否保留):"
kubectl --kubeconfig=$KUBECONFIG_FILE get pvc -n $NAMESPACE

echo ""
echo "=== 7. 健康检查 ==="
sleep 5
curl -sf http://bearer-token-test.jfcs-k8s-qa1.qiniu.io:32252/health | jq . || echo "警告: 健康检查失败"

echo ""
echo "========================================"
echo "✅ 迁移完成！"
echo "========================================"
echo ""
echo "现在可以使用 Helm 管理："
echo "  make helm-deploy-test       # 完整部署（更新配置）"
echo "  make quick-upgrade-test     # 快速升级（只更新镜像）"
echo "  make helm-status            # 查看状态"
echo ""
echo "备份位置: $BACKUP_FILE"
