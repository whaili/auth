#!/bin/bash
# ========================================
# Bearer Token Service V2 - 数据库初始化脚本
# ========================================
# 用途：在多实例负载均衡部署前，统一初始化 MongoDB 数据库和索引
# 使用场景：
#   1. 首次部署时执行一次
#   2. 数据库结构升级时执行
#   3. 新增索引时执行
#
# 执行方式：
#   ./scripts/init-db.sh
#   或
#   bash scripts/init-db.sh
# ========================================

set -e  # 遇到错误立即退出

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Bearer Token Service V2 - 数据库初始化${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# ========================================
# 1. 检查环境变量
# ========================================
MONGO_URI=${MONGO_URI:-"mongodb://localhost:27017"}
MONGO_DATABASE=${MONGO_DATABASE:-"token_service_v2"}

echo -e "${YELLOW}📋 配置信息:${NC}"
echo "   MONGO_URI: $MONGO_URI"
echo "   MONGO_DATABASE: $MONGO_DATABASE"
echo ""

# ========================================
# 2. 检查依赖
# ========================================
echo -e "${YELLOW}🔍 检查依赖...${NC}"

# 检查 mongosh 或 mongo 命令
if command -v mongosh &> /dev/null; then
    MONGO_CMD="mongosh"
    echo -e "${GREEN}✅ 找到 mongosh 命令${NC}"
elif command -v mongo &> /dev/null; then
    MONGO_CMD="mongo"
    echo -e "${GREEN}✅ 找到 mongo 命令${NC}"
else
    echo -e "${RED}❌ 错误: 未找到 mongosh 或 mongo 命令${NC}"
    echo -e "${YELLOW}请安装 MongoDB Shell:${NC}"
    echo "   Ubuntu/Debian: apt install mongodb-mongosh"
    echo "   macOS: brew install mongosh"
    echo "   官方文档: https://www.mongodb.com/docs/mongodb-shell/install/"
    exit 1
fi

echo ""

# ========================================
# 3. 测试 MongoDB 连接
# ========================================
echo -e "${YELLOW}🔌 测试 MongoDB 连接...${NC}"

if $MONGO_CMD "$MONGO_URI/$MONGO_DATABASE" --quiet --eval "db.runCommand({ ping: 1 }).ok" > /dev/null 2>&1; then
    echo -e "${GREEN}✅ MongoDB 连接成功${NC}"
else
    echo -e "${RED}❌ MongoDB 连接失败${NC}"
    echo -e "${YELLOW}请检查:${NC}"
    echo "   1. MongoDB 服务是否运行"
    echo "   2. MONGO_URI 是否正确"
    echo "   3. 网络连接是否正常"
    exit 1
fi

echo ""

# ========================================
# 4. 执行初始化脚本
# ========================================
echo -e "${YELLOW}🚀 开始创建索引...${NC}"
echo ""

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INIT_SCRIPT="$SCRIPT_DIR/init-indexes.js"

# 检查 init-indexes.js 是否存在
if [ ! -f "$INIT_SCRIPT" ]; then
    echo -e "${RED}❌ 错误: 找不到 $INIT_SCRIPT${NC}"
    exit 1
fi

# 执行初始化脚本
if $MONGO_CMD "$MONGO_URI/$MONGO_DATABASE" --quiet "$INIT_SCRIPT"; then
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}✅ 数据库初始化成功！${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "${BLUE}📝 下一步操作:${NC}"
    echo "   1. 启动服务时设置环境变量: SKIP_INDEX_CREATION=true"
    echo "   2. 启动多个服务实例进行负载均衡"
    echo ""
    echo -e "${YELLOW}示例:${NC}"
    echo "   export SKIP_INDEX_CREATION=true"
    echo "   ./bin/server"
    echo ""
    exit 0
else
    echo ""
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}❌ 数据库初始化失败${NC}"
    echo -e "${RED}========================================${NC}"
    echo ""
    echo -e "${YELLOW}请检查错误信息并重试${NC}"
    exit 1
fi
