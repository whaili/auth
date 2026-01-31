// Bearer Token Service V2 - 持续压力测试（Soak Test）
// 目的：测试系统长时间运行的稳定性，检测内存泄漏等问题

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
const DURATION = __ENV.DURATION || '2h';  // 默认 2 小时

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
    soak_test: {
      executor: 'constant-vus',
      vus: 100,                    // 稳定 100 并发用户
      duration: DURATION,          // 持续时间（默认 2 小时）
    },
  },

  thresholds: {
    // 长时间运行时的稳定性指标
    'token_validation_duration_ms': [
      'p(95)<100',   // P95 持续低于 100ms
      'p(99)<200',   // P99 持续低于 200ms
      'max<1000',    // 最大延迟不超过 1 秒
    ],
    'token_validation_success': ['rate>0.999'],  // 99.9% 成功率
    'http_req_failed': ['rate<0.001'],           // <0.1% 失败率
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

  // Soak 测试使用较长的间隔，模拟正常负载
  sleep(0.5 + Math.random() * 0.5);  // 500-1000ms
}

// ========================================
// 结果汇总
// ========================================

export function handleSummary(data) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);

  console.log('\n========== 持续压力测试分析 ==========');
  console.log(`测试时间: ${timestamp}`);
  console.log(`测试时长: ${DURATION}`);
  console.log(`并发用户: 100 VUs`);

  if (data.metrics.http_reqs) {
    console.log(`总请求数: ${data.metrics.http_reqs.values.count}`);
    console.log(`平均 RPS: ${data.metrics.http_reqs.values.rate.toFixed(2)}`);
  }

  if (data.metrics.http_req_failed) {
    console.log(`失败率: ${(data.metrics.http_req_failed.values.rate * 100).toFixed(4)}%`);
  }

  if (data.metrics.http_req_duration) {
    console.log(`P50 延迟: ${data.metrics.http_req_duration.values['p(50)'].toFixed(2)}ms`);
    console.log(`P95 延迟: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms`);
    console.log(`P99 延迟: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms`);
    console.log(`最大延迟: ${data.metrics.http_req_duration.values.max.toFixed(2)}ms`);
  }

  // 稳定性分析
  console.log('\n--- 稳定性分析 ---');
  if (data.metrics.http_req_duration) {
    const maxLatency = data.metrics.http_req_duration.values.max;
    const p99Latency = data.metrics.http_req_duration.values['p(99)'];

    if (maxLatency > 1000) {
      console.log('警告: 出现超过 1 秒的延迟，可能存在性能问题');
    } else if (p99Latency > 200) {
      console.log('注意: P99 延迟偏高，建议检查资源使用情况');
    } else {
      console.log('系统表现稳定，无明显性能退化');
    }
  }

  console.log('========================================\n');

  const resultFile = `results/soak-test-${timestamp}.json`;

  return {
    [resultFile]: JSON.stringify(data, null, 2),
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
  };
}
