# Bearer Token Service V2 - 压测指南

基于 k6 的性能测试框架，用于评估系统极限、监控生产性能、指导扩容决策。

## 快速开始

### 1. 安装 k6

```bash
# macOS
brew install k6

# Ubuntu/Debian
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
    --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
    sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install k6

# Docker
docker run --rm -i grafana/k6 run - <script.js
```

### 2. 生成测试数据

```bash
# 确保服务已启动
./run.sh setup
```

### 3. 运行测试

```bash
# 基准测试
./run.sh baseline

# 混合负载
./run.sh mixed

# 快速测试（1分钟）
./run.sh quick
```

---

## 测试场景

| 场景 | 命令 | 目的 | 时长 |
|------|------|------|------|
| **setup** | `./run.sh setup` | 生成测试 Token | ~1分钟 |
| **baseline** | `./run.sh baseline` | 单接口性能基准 | ~10分钟 |
| **mixed** | `./run.sh mixed` | 模拟真实流量分布 | ~10分钟 |
| **spike** | `./run.sh spike` | 突发流量处理能力 | ~15分钟 |
| **stress** | `./run.sh stress` | 找性能拐点和极限 | ~20分钟 |
| **soak** | `./run.sh soak` | 长期稳定性测试 | 2小时 |
| **quick** | `./run.sh quick` | 快速验证 | 1分钟 |

---

## 场景详解

### Baseline - 基准测试

测试 `/api/v2/validate` 接口的极限性能。

```bash
./run.sh baseline
```

**负载模式**：
- 0 → 50 VUs（预热）
- 50 → 1000 VUs（逐步增加）
- 维持峰值 2 分钟

**性能阈值**：
- P50 < 20ms
- P95 < 50ms
- P99 < 100ms
- 成功率 > 99%

### Mixed - 混合负载

模拟真实流量分布：

- 90% Token 验证
- 7% 列出 Token
- 2% 创建 Token
- 1% 获取详情

```bash
./run.sh mixed
```

### Spike - 突发流量

测试系统应对 10x 和 20x 流量突增的能力。

```bash
./run.sh spike
```

**负载模式**：
1. 正常负载（50 VUs）
2. 突增到 500 VUs（10x）
3. 恢复正常
4. 突增到 1000 VUs（20x）
5. 观察恢复

### Stress - 压力极限

找到系统性能拐点。

```bash
./run.sh stress
```

**负载模式**：100 → 5000 RPS 递增

### Soak - 持续压力

测试长时间运行的稳定性，检测内存泄漏。

```bash
# 默认 2 小时
./run.sh soak

# 自定义时长
DURATION=30m ./run.sh soak
```

---

## 配置选项

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `BASE_URL` | 服务地址 | `http://localhost:8081` |
| `TEST_UID` | 测试用户 UID | `1369077332` |
| `TOKEN_COUNT` | 生成 Token 数量 | `100` |
| `DURATION` | Soak 测试时长 | `2h` |

### 使用示例

```bash
# 指定服务地址
BASE_URL=http://staging:8080 ./run.sh baseline

# 生成更多 Token
TOKEN_COUNT=500 ./run.sh setup

# 30 分钟持续测试
DURATION=30m ./run.sh soak
```

---

## 性能指标

### 核心指标阈值

| 指标 | 正常 | 警告 | 严重 |
|------|------|------|------|
| P50 延迟 | <10ms | 10-50ms | >50ms |
| P95 延迟 | <50ms | 50-100ms | >100ms |
| P99 延迟 | <100ms | 100-500ms | >500ms |
| 单实例 QPS | >1000 | 500-1000 | <500 |
| 错误率 (5xx) | <0.1% | 0.1-1% | >1% |
| 缓存命中率 | >90% | 70-90% | <70% |

### 扩容决策规则

**触发扩容**（满足任一）：
- CPU > 70%，持续 60 秒
- 内存 > 80%，持续 60 秒
- P95 延迟 > 100ms，持续 2 分钟
- 错误率 > 1%，持续 1 分钟

**容量计算**：
```
所需实例 = ceil(峰值QPS / (单实例安全QPS × 0.7)) × 1.3
```

---

## 目录结构

```
loadtest/
├── README.md                 # 本文档
├── run.sh                    # 执行脚本
├── scripts/
│   ├── lib/
│   │   ├── auth.js          # 认证辅助函数
│   │   ├── checks.js        # 响应校验
│   │   └── metrics.js       # 自定义指标
│   ├── scenarios/
│   │   ├── baseline-validate.js
│   │   ├── mixed-load.js
│   │   ├── spike-test.js
│   │   ├── stress-test.js
│   │   └── soak-test.js
│   └── setup/
│       └── create-test-tokens.js
├── data/
│   └── tokens.csv           # 测试 Token（生成后）
└── results/                  # 测试结果（JSON）
```

---

## 结果分析

测试结果保存在 `results/` 目录，格式为 JSON。

### 关键分析点

1. **性能拐点**：P95 延迟开始快速上升的 QPS 值
2. **最大吞吐量**：系统能稳定处理的最高 QPS
3. **资源瓶颈**：CPU、内存、连接数哪个先到达上限
4. **恢复能力**：突发流量后恢复正常的时间
5. **稳定性**：Soak 测试中是否有性能退化

### 示例输出

```
========== 压力极限测试分析报告 ==========
总请求数: 1,234,567
平均 RPS: 2,345.67
失败率:   0.12%
P50 延迟: 5.23ms
P95 延迟: 23.45ms
P99 延迟: 67.89ms
最大延迟: 234.56ms
缓存命中: 92.34%
建议:     系统表现良好
==========================================
```

---

## 与监控集成

### Prometheus 输出

```bash
k6 run --out experimental-prometheus-rw scripts/scenarios/baseline-validate.js
```

### InfluxDB 输出

```bash
k6 run --out influxdb=http://localhost:8086/k6 scripts/scenarios/baseline-validate.js
```

---

## 故障排查

### k6 未安装

```
错误: k6 未安装
```

解决：参考上方安装说明。

### 服务不可用

```
错误: 服务不可用 (http://localhost:8081)
```

解决：确保服务已启动，检查端口和地址。

### Token 文件不存在

测试脚本会使用占位符 Token，验证接口将返回 401。建议先运行 `./run.sh setup` 生成真实 Token。

---

## 最佳实践

1. **测试前**：确保服务处于正常状态，无其他负载
2. **数据准备**：先运行 `setup` 生成足够的测试 Token
3. **逐步增压**：从 `baseline` 开始，逐步进行 `stress` 测试
4. **监控观察**：测试时同步观察 Prometheus/Grafana 指标
5. **结果保存**：每次测试结果自动保存，便于对比分析

---

**最后更新**: 2026-01-14
