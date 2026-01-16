#!/bin/bash

# Bearer Token Service V2 - 压测执行脚本
# 用法: ./run.sh <scenario> [options]

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 默认配置
BASE_URL="${BASE_URL:-http://localhost:8081}"
TEST_UID="${TEST_UID:-1369077332}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# 帮助信息
show_help() {
    echo -e "${BLUE}Bearer Token Service V2 - 压测工具${NC}"
    echo ""
    echo "用法: $0 <scenario> [options]"
    echo ""
    echo "场景:"
    echo "  setup         生成测试 Token 数据"
    echo "  baseline      基准测试 - Token 验证接口"
    echo "  mixed         混合负载测试 - 模拟真实流量"
    echo "  spike         突发流量测试 - 10x/20x 峰值"
    echo "  stress        压力极限测试 - 找性能拐点"
    echo "  soak          持续压力测试 - 2 小时稳定性"
    echo "  quick         快速测试 - 1 分钟基准"
    echo ""
    echo "选项:"
    echo "  -u, --url     服务地址 (默认: $BASE_URL)"
    echo "  -h, --help    显示帮助信息"
    echo ""
    echo "环境变量:"
    echo "  BASE_URL      服务地址"
    echo "  TEST_UID      测试用户 UID"
    echo "  TOKEN_COUNT   生成 Token 数量 (setup 场景)"
    echo "  DURATION      测试时长 (soak 场景)"
    echo ""
    echo "示例:"
    echo "  $0 setup                           # 生成 100 个测试 Token"
    echo "  $0 baseline                        # 运行基准测试"
    echo "  $0 mixed -u http://staging:8080    # 指定服务地址"
    echo "  TOKEN_COUNT=500 $0 setup           # 生成 500 个 Token"
    echo "  DURATION=30m $0 soak               # 运行 30 分钟持续测试"
}

# 检查 k6 是否安装
check_k6() {
    if ! command -v k6 &> /dev/null; then
        echo -e "${RED}错误: k6 未安装${NC}"
        echo ""
        echo "安装方法:"
        echo "  macOS:  brew install k6"
        echo "  Linux:  sudo apt-get install k6"
        echo "  Docker: docker run --rm -i grafana/k6 run - <script.js"
        echo ""
        echo "详见: https://k6.io/docs/getting-started/installation/"
        exit 1
    fi
}

# 检查服务健康
check_health() {
    echo -e "${BLUE}检查服务健康状态...${NC}"
    local health=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health" 2>/dev/null || echo "000")

    if [ "$health" != "200" ]; then
        echo -e "${RED}错误: 服务不可用 ($BASE_URL)${NC}"
        echo "请确保服务已启动"
        exit 1
    fi

    echo -e "${GREEN}服务健康 ✓${NC}"
    echo ""
}

# 运行测试
run_test() {
    local script=$1
    local name=$2

    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}运行测试: ${name}${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo -e "服务地址: ${BASE_URL}"
    echo -e "测试脚本: ${script}"
    echo -e "结果目录: ${SCRIPT_DIR}/results/"
    echo ""

    # 切换到 loadtest 目录运行，确保结果文件路径正确
    cd "$SCRIPT_DIR"
    k6 run \
        --env BASE_URL="$BASE_URL" \
        --env TEST_UID="$TEST_UID" \
        "$script"
}

# 解析参数
SCENARIO=""
while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--url)
            BASE_URL="$2"
            shift 2
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            SCENARIO="$1"
            shift
            ;;
    esac
done

# 检查场景
if [ -z "$SCENARIO" ]; then
    show_help
    exit 1
fi

# 检查依赖
check_k6

# 根据场景运行测试
case $SCENARIO in
    setup)
        check_health
        echo -e "${YELLOW}提示: Token 将输出到控制台，请手动保存到 data/tokens.csv${NC}"
        echo ""
        k6 run \
            --env BASE_URL="$BASE_URL" \
            --env TEST_UID="$TEST_UID" \
            --env TOKEN_COUNT="${TOKEN_COUNT:-100}" \
            "$SCRIPT_DIR/scripts/setup/create-test-tokens.js" 2>&1 | tee /tmp/k6-setup.log

        # 提取 Token 并保存
        echo ""
        echo -e "${BLUE}提取 Token 到 data/tokens.csv...${NC}"
        grep "^sk-" /tmp/k6-setup.log > "$SCRIPT_DIR/data/tokens.csv" 2>/dev/null || true

        TOKEN_COUNT_SAVED=$(wc -l < "$SCRIPT_DIR/data/tokens.csv" 2>/dev/null || echo "0")
        echo -e "${GREEN}已保存 ${TOKEN_COUNT_SAVED} 个 Token 到 data/tokens.csv${NC}"
        ;;
    baseline)
        check_health
        run_test "scripts/scenarios/baseline-validate.js" "基准测试 - Token 验证"
        ;;
    mixed)
        check_health
        run_test "scripts/scenarios/mixed-load.js" "混合负载测试"
        ;;
    spike)
        check_health
        run_test "scripts/scenarios/spike-test.js" "突发流量测试"
        ;;
    stress)
        check_health
        echo -e "${YELLOW}警告: 压力测试可能导致服务过载${NC}"
        read -p "确认继续? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            run_test "scripts/scenarios/stress-test.js" "压力极限测试"
        else
            echo "已取消"
            exit 0
        fi
        ;;
    soak)
        check_health
        echo -e "${YELLOW}持续压力测试将运行 ${DURATION:-2h}${NC}"
        read -p "确认继续? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            k6 run \
                --env BASE_URL="$BASE_URL" \
                --env TEST_UID="$TEST_UID" \
                --env DURATION="${DURATION:-2h}" \
                "$SCRIPT_DIR/scripts/scenarios/soak-test.js"
        else
            echo "已取消"
            exit 0
        fi
        ;;
    quick)
        # 快速测试：1 分钟基准
        check_health
        echo -e "${BLUE}快速测试 (1 分钟)${NC}"
        k6 run \
            --env BASE_URL="$BASE_URL" \
            --env TEST_UID="$TEST_UID" \
            --duration 1m \
            --vus 50 \
            "$SCRIPT_DIR/scripts/scenarios/baseline-validate.js"
        ;;
    *)
        echo -e "${RED}未知场景: $SCENARIO${NC}"
        echo ""
        show_help
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}测试完成！${NC}"
echo -e "结果保存在: ${SCRIPT_DIR}/results/"
