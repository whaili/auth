# MySQL 测试数据迁移 - 使用指南

## ✅ 脚本已就绪

测试数据迁移脚本已完成并测试通过！

## 🚀 快速使用

### 基本用法

```bash
cd /root/src/auth

# 使用默认配置（500条记录，数据库名: bearer_token_test_<用户名>）
./scripts/migrate_test_data_v2.sh

# 指定数据库名和记录数
./scripts/migrate_test_data_v2.sh bearer_test_20260204 1000
```

### 参数说明

```bash
./scripts/migrate_test_data_v2.sh [数据库名] [记录数]
```

- **数据库名** (可选): 测试数据库名称，默认 `bearer_token_test_<username>`
- **记录数** (可选): 导出记录数量，默认 `500`

## 📊 示例

### 示例 1: 默认配置

```bash
./scripts/migrate_test_data_v2.sh
```

输出：
```
[✓] 从生产环境导出数据 (500 条记录)...
[✓] 下载导出文件...
[✓] 混淆敏感数据...
[✓] 创建测试数据库: bearer_token_test_gem
[✓] 数据导入完成...
[✓] 成功导入 500 条记录
```

### 示例 2: 自定义配置

```bash
./scripts/migrate_test_data_v2.sh my_test_db 100
```

## 🔧 完成后的配置

脚本执行完成后，会输出测试数据库配置：

```bash
export MYSQL_HOST="10.210.31.54"
export MYSQL_PORT="3306"
export MYSQL_USER="bo"
export MYSQL_PASSWORD="bo"
export MYSQL_DATABASE="bearer_test_demo"  # 你的数据库名
```

### 更新 .env 文件

```bash
# 将上面的配置添加到 .env 文件
cat >> .env << 'EOF'
# MySQL 测试环境配置
export MYSQL_HOST="10.210.31.54"
export MYSQL_PORT="3306"
export MYSQL_USER="bo"
export MYSQL_PASSWORD="bo"
export MYSQL_DATABASE="bearer_test_demo"
EOF

# 加载配置
source .env
```

### 启动服务测试

```bash
# 加载环境变量
source .env

# 启动服务
go run cmd/server/main.go
```

### 测试 API

```bash
# 1. 查询一个测试 UID
mysql -h 10.210.31.54 -u bo -p'bo' bearer_test_demo \
  -e "SELECT id, username, email FROM auth LIMIT 1;"

# 假设查到 id=2, username=1383009373

# 2. 创建 Token
curl -X POST http://localhost:8080/api/v2/tokens \
  -H 'Authorization: QiniuStub uid=1383009373&ut=1' \
  -H 'Content-Type: application/json' \
  -d '{"description": "测试Token", "expires_in_seconds": 86400}'

# 3. 测试 validateu 接口
curl -X POST http://localhost:8080/api/v2/validateu \
  -H 'Authorization: Bearer <返回的token>' | jq .
```

## ✨ 功能特性

### 自动化完成

- ✅ 从生产 MySQL 导出数据（通过 vmyzh165）
- ✅ 自动混淆敏感数据（邮箱、密码哈希）
- ✅ 修复字符集兼容性问题
- ✅ 创建测试数据库并导入
- ✅ 自动清理临时文件

### 数据混淆规则

| 字段 | 混淆规则 | 示例 |
|------|---------|------|
| 邮箱 | 保留域名，用户名用 MD5 哈希 | `test_63a9f0ea@example.com` |
| 密码 | SHA256 固定测试密码哈希 | `9f86d081...` |
| IP | 保留前两段，后两段随机 | `192.168.23.156` |
| 手机号 | 保留前3后4位，中间星号 | `138****5678` |

## 🔍 故障排查

### 问题 1: SSH 连接失败

```bash
# 测试 SSH 连接
ssh vmyzh165 "echo OK"
```

### 问题 2: MySQL 连接失败

```bash
# 测试测试 MySQL 连接
mysql -h 10.210.31.54 -u bo -p'bo' -e "SELECT 1;"
```

### 问题 3: 数据库已存在

```bash
# 删除旧数据库
mysql -h 10.210.31.54 -u bo -p'bo' -e "DROP DATABASE IF EXISTS bearer_test_old;"

# 或使用新数据库名
./scripts/migrate_test_data_v2.sh bearer_test_$(date +%Y%m%d_%H%M) 500
```

## 📝 注意事项

1. **数据库命名**: 建议使用唯一的数据库名，避免与其他开发者冲突
2. **记录数量**: 默认500条足够测试，如需更多可自行指定
3. **数据更新**: 如果生产数据有重大更新，重新运行脚本即可
4. **清理**: 不再使用的测试数据库请及时删除

## 🔗 相关文档

- [CLAUDE.md](../CLAUDE.md) - 项目总体说明
- [_cust/DEPLOYMENT.md](../_cust/DEPLOYMENT.md) - 部署文档
- [scripts/obfuscate_data.py](obfuscate_data.py) - 数据混淆脚本

## 💡 提示

### 查看所有测试数据库

```bash
mysql -h 10.210.31.54 -u bo -p'bo' -e "SHOW DATABASES LIKE 'bearer%';"
```

### 连接测试数据库

```bash
mysql -h 10.210.31.54 -u bo -p'bo' bearer_test_demo
```

### 查询数据统计

```bash
mysql -h 10.210.31.54 -u bo -p'bo' bearer_test_demo \
  -e "SELECT COUNT(*) as total, MIN(id) as min_id, MAX(id) as max_id FROM auth;"
```

---

**最后更新**: 2026-02-04
**状态**: ✅ 已测试通过
**维护者**: Bearer Token Service Team
