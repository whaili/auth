#!/bin/bash
# 快速升级测试环境（只更新镜像，不修改配置）
# 使用方法：./scripts/quick-upgrade-test.sh [镜像标签，默认为 latest]

set -e

KUBECONFIG_FILE="/root/src/auth/_cust/kubeconfig-test"
NAMESPACE="bearer-token-test"
IMAGE_TAG="${1:-latest}"
IMAGE="aslan-spock-register.qiniu.io/miku-stream/bearer-token-service:$IMAGE_TAG"

echo "=== 快速升级测试环境 ==="
echo "镜像: $IMAGE"
echo ""

echo "=== 1. 检查当前版本 ==="
kubectl --kubeconfig=$KUBECONFIG_FILE \
    get deployment bearer-token-service -n $NAMESPACE \
    -o jsonpath='{.spec.template.spec.containers[0].image}'
echo ""
echo ""

echo "=== 2. 更新镜像 ==="
kubectl --kubeconfig=$KUBECONFIG_FILE \
    set image deployment/bearer-token-service \
    bearer-token-service=$IMAGE \
    -n $NAMESPACE

echo ""
echo "=== 3. 等待滚动更新完成 ==="
kubectl --kubeconfig=$KUBECONFIG_FILE \
    rollout status deployment/bearer-token-service \
    -n $NAMESPACE --timeout=120s

echo ""
echo "=== 4. 查看新 Pod ==="
kubectl --kubeconfig=$KUBECONFIG_FILE \
    get pods -n $NAMESPACE -l app=bearer-token-service

echo ""
echo "=== 5. 健康检查 ==="
sleep 5
curl -sf http://bearer-token-test.jfcs-k8s-qa1.qiniu.io:32252/health | jq .

echo ""
echo "✅ 升级完成！"
