#!/bin/bash
# Helm 升级测试环境脚本
# 使用方法：./scripts/helm-upgrade-test.sh [镜像标签，默认为 latest]

set -e

KUBECONFIG_FILE="/root/src/auth/_cust/kubeconfig-test"
NAMESPACE="bearer-token-test"
RELEASE_NAME="bearer-token"
CHART_PATH="deploy/helm/bearer-token-service"
VALUES_FILE="deploy/helm/bearer-token-service/values-test.yaml"
IMAGE_TAG="${1:-latest}"

# 加载凭据（如果存在）
if [ -f "_cust/credentials.env" ]; then
    source _cust/credentials.env
fi

# 检查必要的环境变量
if [ -z "$TEST_QCONF_SECRET_KEY" ]; then
    echo "错误: TEST_QCONF_SECRET_KEY 未设置"
    echo "请在 _cust/credentials.env 中设置或通过环境变量传递"
    exit 1
fi

if [ -z "$TEST_MONGO_PASSWORD" ]; then
    echo "错误: TEST_MONGO_PASSWORD 未设置"
    echo "请在 _cust/credentials.env 中设置或通过环境变量传递"
    exit 1
fi

echo "=== Helm 升级测试环境 ==="
echo "Kubeconfig: $KUBECONFIG_FILE"
echo "Namespace: $NAMESPACE"
echo "Release: $RELEASE_NAME"
echo "Image Tag: $IMAGE_TAG"
echo ""

cd /root/src/auth

# 使用 Helm upgrade --install（如果不存在会自动安装）
helm --kubeconfig=$KUBECONFIG_FILE upgrade --install $RELEASE_NAME $CHART_PATH \
    -f $VALUES_FILE \
    -n $NAMESPACE \
    --create-namespace \
    --set image.tag=$IMAGE_TAG \
    --set mongodb.auth.password=$TEST_MONGO_PASSWORD \
    --set qconf.enabled=true \
    --set qconf.accessKey="-lnWyW53aRF1AYUj5D2oBBub377cTMYawZdOT25z" \
    --set qconf.secretKey=$TEST_QCONF_SECRET_KEY \
    --set qconf.masterHosts="http://kodo-dev.confg.jfcs-k8s-qa2.qiniu.io"

echo ""
echo "=== 等待部署完成 ==="
kubectl --kubeconfig=$KUBECONFIG_FILE rollout status deployment/bearer-token-service -n $NAMESPACE --timeout=120s

echo ""
echo "=== 查看 Pod 状态 ==="
kubectl --kubeconfig=$KUBECONFIG_FILE get pods -n $NAMESPACE

echo ""
echo "=== 查看 Helm Release ==="
helm --kubeconfig=$KUBECONFIG_FILE list -n $NAMESPACE

echo ""
echo "✅ 升级完成！"
echo ""
echo "测试访问："
echo "  curl http://bearer-token-test.jfcs-k8s-qa1.qiniu.io:32252/health"
