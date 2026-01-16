// Bearer Token Service V2 - 基准测试：Token 验证
// 目的：测试 /api/v2/validate 接口的极限性能

import http from 'k6/http';
import { sleep } from 'k6';
import { SharedArray } from 'k6/data';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';

import { bearerAuth } from '../lib/auth.js';
import {
  tokenValidationDuration,
  tokenValidationTotal,
  recordCacheHit
} from '../lib/metrics.js';
import { checkValidateResponseLenient } from '../lib/checks.js';

// ========================================
// 配置
// ========================================

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';

// 从文件加载测试 Token（如果存在）
let tokens;
try {
  tokens = new SharedArray('tokens', function () {
    const data = open('../../data/tokens.csv');
    return data.split('\n').filter(t => t.trim());
  });
} catch (e) {
  // 如果没有预生成的 token 文件，使用占位符
  tokens = ['sk-placeholder-token-for-testing'];
}

// ========================================
// 测试配置
// ========================================

export const options = {
  // 场景：逐步增加负载，找到极限
  scenarios: {
    baseline_validate: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 50 },   // 预热
        { duration: '1m', target: 100 },   // 逐步增加
        { duration: '1m', target: 200 },
        { duration: '1m', target: 500 },
        { duration: '2m', target: 500 },   // 稳定压力
        { duration: '1m', target: 1000 },  // 峰值测试
        { duration: '2m', target: 1000 },  // 峰值稳定
        { duration: '30s', target: 0 },    // 冷却
      ],
    },
  },

  // 性能阈值
  thresholds: {
    // 响应时间
    'token_validation_duration_ms': [
      'p(50)<20',    // P50 < 20ms
      'p(95)<50',    // P95 < 50ms
      'p(99)<100',   // P99 < 100ms
    ],
    // 成功率
    'token_validation_success': ['rate>0.99'],  // 99% 成功率
    // HTTP 错误
    'http_req_failed': ['rate<0.01'],           // <1% 失败率
    // 5xx 错误
    'errors_5xx': ['count<10'],                 // 5xx 错误少于 10 个
  },
};

// ========================================
// 测试逻辑
// ========================================

export default function () {
  // 随机选择一个 Token
  const token = tokens[Math.floor(Math.random() * tokens.length)];

  const startTime = Date.now();
  const res = http.post(
    `${BASE_URL}/api/v2/validate`,
    null,
    bearerAuth(token)
  );
  const duration = Date.now() - startTime;

  // 记录指标
  tokenValidationDuration.add(duration);
  tokenValidationTotal.add(1);

  // 根据响应时间推断缓存命中（<10ms 认为是缓存命中）
  recordCacheHit(duration);

  // 响应校验
  checkValidateResponseLenient(res);

  // 模拟真实请求间隔（10-50ms）
  sleep(0.01 + Math.random() * 0.04);
}

// ========================================
// 结果汇总
// ========================================

export function handleSummary(data) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);

  // 输出分析
  console.log('\n========== 基准测试分析 ==========');
  console.log(`测试时间: ${timestamp}`);
  console.log(`目标服务: ${BASE_URL}`);

  if (data.metrics.http_reqs) {
    console.log(`总请求数: ${data.metrics.http_reqs.values.count}`);
    console.log(`平均 RPS: ${data.metrics.http_reqs.values.rate.toFixed(2)}`);
  }

  if (data.metrics.http_req_failed) {
    console.log(`失败率: ${(data.metrics.http_req_failed.values.rate * 100).toFixed(2)}%`);
  }

  if (data.metrics.http_req_duration && data.metrics.http_req_duration.values) {
    const v = data.metrics.http_req_duration.values;
    // k6 v1.5+ 延迟单位是秒，转换为毫秒
    const toMs = (val) => val ? (val * 1000).toFixed(2) : '0.00';
    console.log(`P50 延迟: ${toMs(v['p(50)'])}ms`);
    console.log(`P95 延迟: ${toMs(v['p(95)'])}ms`);
    console.log(`P99 延迟: ${toMs(v['p(99)'])}ms`);
    console.log(`最大延迟: ${toMs(v.max)}ms`);
  }

  if (data.metrics.cache_hit_rate) {
    console.log(`缓存命中率: ${(data.metrics.cache_hit_rate.values.rate * 100).toFixed(2)}%`);
  }

  console.log('====================================\n');

  // 结果写入文件和终端
  const resultFile = `results/baseline-validate-${timestamp}.json`;

  return {
    [resultFile]: JSON.stringify(data, null, 2),
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
  };
}
