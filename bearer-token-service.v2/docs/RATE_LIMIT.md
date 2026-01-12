# 三层限流功能说明

本项目实现了三层限流架构，用于保护服务免受滥用和过载。

## 限流层级

### 1. 应用层限流（Application Rate Limit）
- **作用域**：全局（整个服务）
- **目的**：保护服务整体不被过载
- **配置方式**：环境变量
- **默认状态**：关闭

### 2. 账户层限流（Account Rate Limit）
- **作用域**：单个租户账户
- **目的**：防止单个租户滥用资源
- **配置方式**：数据库（Account.RateLimit 字段）
- **默认状态**：关闭

### 3. Token 层限流（Token Rate Limit）
- **作用域**：单个 Bearer Token
- **目的**：精细化控制每个 Token 的使用频率
- **配置方式**：数据库（Token.RateLimit 字段）
- **默认状态**：关闭

## 技术特性

- **算法**：滑动窗口（Sliding Window）
- **存储**：内存（高性能，自动清理过期数据）
- **维度**：支持分钟/小时/天三个时间窗口
- **响应**：返回 `429 Too Many Requests` 和 `Retry-After` 头
- **状态**：响应头包含限流状态（剩余请求数、重置时间）

## 快速开始

### 1. 启用应用层限流

```bash
export ENABLE_APP_RATE_LIMIT=true
export APP_RATE_LIMIT_PER_MINUTE=1000    # 每分钟 1000 个请求
export APP_RATE_LIMIT_PER_HOUR=50000     # 每小时 50000 个请求
export APP_RATE_LIMIT_PER_DAY=1000000    # 每天 1000000 个请求
```

### 2. 启用账户层限流

```bash
export ENABLE_ACCOUNT_RATE_LIMIT=true
```

然后在创建账户后，通过数据库设置账户限流：

```javascript
db.accounts.updateOne(
  { _id: "account_id" },
  {
    $set: {
      rate_limit: {
        requests_per_minute: 500,
        requests_per_hour: 30000,
        requests_per_day: 500000
      }
    }
  }
)
```

### 3. 启用 Token 层限流

```bash
export ENABLE_TOKEN_RATE_LIMIT=true
```

创建 Token 时指定限流配置：

```bash
POST /api/v2/tokens
{
  "description": "Limited token",
  "expires_in_seconds": 3600,
  "rate_limit": {
    "requests_per_minute": 100,
    "requests_per_hour": 5000,
    "requests_per_day": 100000
  }
}
```

## 环境变量配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `ENABLE_APP_RATE_LIMIT` | `false` | 启用应用层限流 |
| `ENABLE_ACCOUNT_RATE_LIMIT` | `false` | 启用账户层限流 |
| `ENABLE_TOKEN_RATE_LIMIT` | `false` | 启用 Token 层限流 |
| `APP_RATE_LIMIT_PER_MINUTE` | `1000` | 应用层每分钟限流 |
| `APP_RATE_LIMIT_PER_HOUR` | `50000` | 应用层每小时限流 |
| `APP_RATE_LIMIT_PER_DAY` | `1000000` | 应用层每天限流 |

## 限流触发的响应

```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
X-RateLimit-Limit-Token: 100
X-RateLimit-Remaining-Token: 0
X-RateLimit-Reset-Token: 1735992345
Retry-After: 45

{
  "error": "Token rate limit exceeded",
  "code": 429,
  "timestamp": "2026-01-04T10:30:00Z"
}
```

## 响应头说明

### 应用层限流响应头
- `X-RateLimit-Limit-App`: 应用层限流上限
- `X-RateLimit-Remaining-App`: 剩余请求数
- `X-RateLimit-Reset-App`: 重置时间（Unix 时间戳）

### 账户层限流响应头
- `X-RateLimit-Limit-Account`: 账户层限流上限
- `X-RateLimit-Remaining-Account`: 剩余请求数
- `X-RateLimit-Reset-Account`: 重置时间（Unix 时间戳）

### Token 层限流响应头
- `X-RateLimit-Limit-Token`: Token 层限流上限
- `X-RateLimit-Remaining-Token`: 剩余请求数
- `X-RateLimit-Reset-Token`: 重置时间（Unix 时间戳）

## 测试

运行测试脚本验证限流功能：

```bash
./test_rate_limit.sh
```

测试脚本会：
1. 启动服务（启用三层限流）
2. 创建测试账户和 Token
3. 测试应用层限流
4. 测试 Token 层限流
5. 检查限流响应头

## 限流优先级

请求按以下顺序检查限流：

1. **应用层限流**（最高优先级）
2. **账户层限流**
3. **Token 层限流**（最低优先级）

任何一层触发限流，都会返回 `429 Too Many Requests`。

## 最佳实践

1. **生产环境建议**：
   - 启用应用层限流，设置合理的全局上限
   - 启用账户层限流，防止单租户滥用
   - 根据业务需求选择性启用 Token 层限流

2. **限流配置建议**：
   - 从宽松的限流开始，逐步收紧
   - 监控限流触发情况，调整阈值
   - 不同优先级的客户可以设置不同的账户限流

3. **错误处理**：
   - 客户端收到 `429` 响应时，应遵守 `Retry-After` 头
   - 实现指数退避重试策略
   - 避免在触发限流后立即重试

## 架构设计

```
请求流程:
  ↓
应用层限流中间件
  ↓ (通过)
账户层限流中间件
  ↓ (通过)
Token 层限流中间件
  ↓ (通过)
业务处理
```

## 实现细节

- **文件位置**：
  - 限流核心：`ratelimit/limiter.go`
  - 中间件：`ratelimit/middleware.go`
  - 配置：`config/ratelimit.go`

- **滑动窗口算法**：
  - 记录每个请求的时间戳
  - 定期清理过期的时间戳
  - O(1) 时间复杂度的限流检查

- **内存管理**：
  - 自动清理 5 分钟未使用的限流桶
  - 支持高并发访问（读写锁）

## 故障排查

1. **限流未生效**：
   - 检查环境变量是否正确设置
   - 确认服务已重启
   - 查看服务启动日志

2. **误触发限流**：
   - 检查限流配置是否过于严格
   - 查看限流响应头中的剩余请求数
   - 调整限流阈值

3. **性能问题**：
   - 限流器使用内存存储，性能开销很小
   - 如需更高性能，可考虑使用 Redis 实现

## 未来改进

- [ ] Redis 存储支持（分布式限流）
- [ ] 限流统计和监控
- [ ] 动态调整限流阈值
- [ ] 限流白名单功能
- [ ] 限流指标导出（Prometheus）

---

**注意**：限流功能默认全部关闭，需要通过环境变量显式启用。
