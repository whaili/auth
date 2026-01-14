// Bearer Token Service V2 - 压力极限测试
// 目的：找到系统性能拐点和极限

import http from 'k6/http';
import { SharedArray } from 'k6/data';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';

import { bearerAuth } from '../lib/auth.js';
import {
  tokenValidationDuration,
  recordCacheHit
} from '../lib/metrics.js';
import { checkValidateResponseLenient } from '../lib/checks.js';

// ========================================
// 配置
// ========================================

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';

let tokens;
try {
  tokens = new SharedArray('tokens', function () {
    const data = open('../../data/tokens.csv');
    return data.split('\n').filter(t => t.trim());
  });
} catch (e) {
  tokens = ['sk-placeholder-token-for-testing'];
}

// ========================================
// 测试配置
// ========================================

export const options = {
  scenarios: {
    stress_test: {
      executor: 'ramping-arrival-rate',
      startRate: 100,
      timeUnit: '1s',
      preAllocatedVUs: 500,
      maxVUs: 2000,
      stages: [
        { duration: '1m', target: 100 },    // 预热
        { duration: '2m', target: 500 },    // 增加到 500 RPS
        { duration: '2m', target: 1000 },   // 增加到 1000 RPS
        { duration: '2m', target: 2000 },   // 增加到 2000 RPS
        { duration: '2m', target: 3000 },   // 增加到 3000 RPS
        { duration: '2m', target: 5000 },   // 尝试 5000 RPS（预计超过极限）
        { duration: '3m', target: 5000 },   // 维持极限负载
        { duration: '2m', target: 100 },    // 恢复观察
        { duration: '1m', target: 0 },      // 冷却
      ],
    },
  },

  // 压力测试不设置严格阈值，目的是找到极限
  thresholds: {
    // 只关注系统是否崩溃
    'http_req_failed': ['rate<0.5'],  // 失败率不超过 50%
  },
};

// ========================================
// 测试逻辑
// ========================================

export default function () {
  const token = tokens[Math.floor(Math.random() * tokens.length)];
  const start = Date.now();

  const res = http.post(`${BASE_URL}/api/v2/validate`, null, bearerAuth(token));

  const duration = Date.now() - start;
  tokenValidationDuration.add(duration);
  recordCacheHit(duration);
  checkValidateResponseLenient(res);
}

// ========================================
// 结果汇总
// ========================================

export function handleSummary(data) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);

  console.log('\n');
  console.log('╔══════════════════════════════════════════════════════════════╗');
  console.log('║              压力极限测试分析报告                              ║');
  console.log('╠══════════════════════════════════════════════════════════════╣');

  if (data.metrics.http_reqs) {
    console.log(`║  总请求数: ${String(data.metrics.http_reqs.values.count).padEnd(47)}║`);
    console.log(`║  平均 RPS: ${String(data.metrics.http_reqs.values.rate.toFixed(2)).padEnd(47)}║`);
  }

  if (data.metrics.http_req_failed) {
    const failRate = (data.metrics.http_req_failed.values.rate * 100).toFixed(2);
    console.log(`║  失败率:   ${String(failRate + '%').padEnd(47)}║`);
  }

  console.log('╠══════════════════════════════════════════════════════════════╣');

  if (data.metrics.http_req_duration) {
    console.log(`║  P50 延迟: ${String(data.metrics.http_req_duration.values['p(50)'].toFixed(2) + 'ms').padEnd(47)}║`);
    console.log(`║  P95 延迟: ${String(data.metrics.http_req_duration.values['p(95)'].toFixed(2) + 'ms').padEnd(47)}║`);
    console.log(`║  P99 延迟: ${String(data.metrics.http_req_duration.values['p(99)'].toFixed(2) + 'ms').padEnd(47)}║`);
    console.log(`║  最大延迟: ${String(data.metrics.http_req_duration.values.max.toFixed(2) + 'ms').padEnd(47)}║`);
  }

  console.log('╠══════════════════════════════════════════════════════════════╣');

  if (data.metrics.rate_limit_hits) {
    console.log(`║  限流触发: ${String(data.metrics.rate_limit_hits.values.count).padEnd(47)}║`);
  }

  if (data.metrics.cache_hit_rate) {
    const hitRate = (data.metrics.cache_hit_rate.values.rate * 100).toFixed(2);
    console.log(`║  缓存命中: ${String(hitRate + '%').padEnd(47)}║`);
  }

  console.log('╠══════════════════════════════════════════════════════════════╣');

  // 性能拐点分析
  let recommendation = '系统表现良好';
  if (data.metrics.http_req_failed && data.metrics.http_req_failed.values.rate > 0.1) {
    recommendation = '已达到系统极限，建议扩容';
  } else if (data.metrics.http_req_duration && data.metrics.http_req_duration.values['p(95)'] > 100) {
    recommendation = 'P95 延迟偏高，接近容量上限';
  }

  console.log(`║  建议:     ${recommendation.padEnd(47)}║`);
  console.log('╚══════════════════════════════════════════════════════════════╝');
  console.log('\n');

  const resultFile = `results/stress-test-${timestamp}.json`;

  return {
    [resultFile]: JSON.stringify(data, null, 2),
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
  };
}
