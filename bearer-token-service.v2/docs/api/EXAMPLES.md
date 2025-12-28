# Bearer Token Service V2 - 使用示例

## Token 过期时间示例（秒级精度）

### 1. 创建短期 Token（1小时过期）

```bash
curl -X POST "http://localhost:8080/api/v2/tokens" \
  -H "Authorization: QINIU ${ACCESS_KEY}:${SIGNATURE}" \
  -H "X-Qiniu-Date: ${TIMESTAMP}" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Short-lived token for testing",
    "scope": ["storage:read"],
    "expires_in_seconds": 3600
  }'
```

**说明**：`expires_in_seconds: 3600` = 1小时后过期

---

### 2. 创建中期 Token（7天过期）

```json
{
  "description": "Weekly access token",
  "scope": ["storage:*"],
  "expires_in_seconds": 604800
}
```

**说明**：`expires_in_seconds: 604800` = 7天 = 7 × 24 × 3600秒

---

### 3. 创建长期 Token（90天过期）

```json
{
  "description": "Production token",
  "scope": ["storage:read", "cdn:refresh"],
  "expires_in_seconds": 7776000
}
```

**说明**：`expires_in_seconds: 7776000` = 90天 = 90 × 24 × 3600秒

---

### 4. 创建永久 Token（永不过期）

```json
{
  "description": "Permanent API token",
  "scope": ["*"],
  "expires_in_seconds": 0
}
```

**说明**：`expires_in_seconds: 0` 或不提供该字段 = 永不过期

---

## 常用过期时间换算表

| 时长 | 秒数 | 用途 |
|------|------|------|
| 5分钟 | 300 | 临时测试 |
| 15分钟 | 900 | 短期验证 |
| 1小时 | 3,600 | 临时访问 |
| 1天 | 86,400 | 日常使用 |
| 7天 | 604,800 | 周期访问 |
| 30天 | 2,592,000 | 月度访问 |
| 90天 | 7,776,000 | 季度访问 |
| 365天 | 31,536,000 | 年度访问 |
| 永不过期 | 0 | 生产环境 |

---

## Python 计算工具

```python
# 快速计算过期秒数
def days_to_seconds(days):
    return days * 24 * 3600

def hours_to_seconds(hours):
    return hours * 3600

def minutes_to_seconds(minutes):
    return minutes * 60

# 使用示例
print(f"1小时 = {hours_to_seconds(1)} 秒")          # 3600
print(f"1天 = {days_to_seconds(1)} 秒")             # 86400
print(f"7天 = {days_to_seconds(7)} 秒")             # 604800
print(f"90天 = {days_to_seconds(90)} 秒")           # 7776000
```

---

## Bash 计算工具

```bash
# 快速计算过期秒数
days_to_seconds() {
    echo $(($1 * 24 * 3600))
}

hours_to_seconds() {
    echo $(($1 * 3600))
}

# 使用示例
echo "1小时 = $(hours_to_seconds 1) 秒"    # 3600
echo "1天 = $(days_to_seconds 1) 秒"       # 86400
echo "90天 = $(days_to_seconds 90) 秒"     # 7776000
```

---

## 完整创建流程示例

### 步骤 1：准备参数

```bash
ACCESS_KEY="AK_f8e7d6c5b4a39281"
SECRET_KEY="SK_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
METHOD="POST"
URI="/api/v2/tokens"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# 创建1小时有效期的 Token
EXPIRES_IN_SECONDS=3600

BODY=$(cat <<EOF
{
  "description": "1-hour temporary token",
  "scope": ["storage:read"],
  "expires_in_seconds": ${EXPIRES_IN_SECONDS}
}
EOF
)
```

### 步骤 2：生成签名

```bash
STRING_TO_SIGN="${METHOD}\n${URI}\n${TIMESTAMP}\n${BODY}"
SIGNATURE=$(echo -n "$STRING_TO_SIGN" | openssl dgst -sha256 -hmac "$SECRET_KEY" -binary | base64)
```

### 步骤 3：发送请求

```bash
curl -X POST "http://localhost:8080${URI}" \
  -H "Authorization: QINIU ${ACCESS_KEY}:${SIGNATURE}" \
  -H "X-Qiniu-Date: ${TIMESTAMP}" \
  -H "Content-Type: application/json" \
  -d "$BODY"
```

### 预期响应

```json
{
  "token_id": "tk_xyz123",
  "token": "sk-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6...",
  "account_id": "acc_1a2b3c4d5e6f",
  "description": "1-hour temporary token",
  "scope": ["storage:read"],
  "created_at": "2025-12-26T10:00:00Z",
  "expires_at": "2025-12-26T11:00:00Z",  # 1小时后
  "is_active": true
}
```

---

## 精度对比：V1 vs V2

### V1（天级精度）

```json
{
  "description": "Old version token",
  "scope": ["storage:read"],
  "expires_in_days": 1  // 只能设置整数天
}
```

**限制**：无法精确控制小时或分钟级别的过期时间

### V2（秒级精度）

```json
{
  "description": "New version token",
  "scope": ["storage:read"],
  "expires_in_seconds": 3600  // 可精确到秒
}
```

**优势**：
- ✅ 支持小时级：`3600`（1小时）
- ✅ 支持分钟级：`900`（15分钟）
- ✅ 支持秒级：`300`（5分钟）
- ✅ 仍支持天级：`86400`（1天）

---

**文档版本**: 2.0
**更新日期**: 2025-12-26
**更新内容**: 添加秒级精度过期时间支持
