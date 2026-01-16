// Bearer Token Service V2 - 突发流量测试
// 目的：测试系统应对流量突增的能力

import http from 'k6/http';
import { sleep } from 'k6';
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
    spike_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        // 正常负载阶段
        { duration: '2m', target: 50 },    // 预热到正常负载
        { duration: '3m', target: 50 },    // 维持正常负载

        // 第一次突发（10x）
        { duration: '10s', target: 500 },  // 10 秒内突增到 10 倍
        { duration: '1m', target: 500 },   // 维持峰值
        { duration: '10s', target: 50 },   // 快速恢复
        { duration: '2m', target: 50 },    // 恢复期观察

        // 第二次更大突发（20x）
        { duration: '10s', target: 1000 }, // 突增到 20 倍
        { duration: '30s', target: 1000 }, // 短暂维持
        { duration: '10s', target: 50 },   // 快速恢复
        { duration: '2m', target: 50 },    // 恢复观察期

        // 冷却
        { duration: '30s', target: 0 },
      ],
    },
  },

  thresholds: {
    // 突发场景允许更高的延迟
    'token_validation_duration_ms': [
      'p(95)<200',   // P95 < 200ms（放宽）
      'p(99)<500',   // P99 < 500ms
    ],
    // 成功率仍要保持高
    'token_validation_success': ['rate>0.98'],
    // 5xx 错误限制
    'errors_5xx': ['count<50'],
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

  // 突发测试使用最小间隔
  sleep(0.01);
}

// ========================================
// 结果汇总
// ========================================

export function handleSummary(data) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);

  console.log('\n========== 突发流量测试分析 ==========');
  console.log(`测试时间: ${timestamp}`);
  console.log('测试场景: 正常负载 → 10x突发 → 恢复 → 20x突发 → 恢复');

  if (data.metrics.http_reqs) {
    console.log(`总请求数: ${data.metrics.http_reqs.values.count}`);
    console.log(`峰值 RPS: 约 ${data.metrics.http_reqs.values.rate.toFixed(2)} (平均)`);
  }

  if (data.metrics.http_req_duration) {
    console.log(`P50 延迟: ${data.metrics.http_req_duration.values['p(50)'].toFixed(2)}ms`);
    console.log(`P95 延迟: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms`);
    console.log(`P99 延迟: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms`);
    console.log(`最大延迟: ${data.metrics.http_req_duration.values.max.toFixed(2)}ms`);
  }

  console.log('========================================\n');

  const resultFile = `results/spike-test-${timestamp}.json`;

  return {
    [resultFile]: JSON.stringify(data, null, 2),
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
  };
}
