// Bearer Token Service V2 - 混合负载测试
// 目的：模拟真实流量分布（90% 验证 + 10% 管理操作）

import http from 'k6/http';
import { sleep, group } from 'k6';
import { SharedArray } from 'k6/data';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';

import { bearerAuth, qiniuStubAuth } from '../lib/auth.js';
import {
  tokenValidationDuration,
  tokenCreationDuration,
  tokenListDuration,
  recordCacheHit
} from '../lib/metrics.js';
import {
  checkValidateResponseLenient,
  checkCreateResponse,
  checkListResponse
} from '../lib/checks.js';

// ========================================
// 配置
// ========================================

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';
const TEST_UID = __ENV.TEST_UID || '1369077332';

// 从文件加载测试 Token
let tokens;
try {
  tokens = new SharedArray('tokens', function () {
    const data = open('../../data/tokens.csv');
    return data.split('\n').filter(t => t.trim());
  });
} catch (e) {
  tokens = ['sk-placeholder-token-for-testing'];
}

// 流量比例配置（模拟真实流量分布）
const TRAFFIC_WEIGHTS = {
  validate: 90,    // 90% - Token 验证
  list: 7,         // 7%  - 列出 Token
  create: 2,       // 2%  - 创建 Token
  getInfo: 1,      // 1%  - 获取详情
};

// ========================================
// 测试配置
// ========================================

export const options = {
  scenarios: {
    mixed_load: {
      executor: 'constant-arrival-rate',
      rate: 500,              // 目标 500 RPS
      timeUnit: '1s',
      duration: '10m',
      preAllocatedVUs: 100,
      maxVUs: 500,
    },
  },

  thresholds: {
    // 验证接口（主要负载）
    'token_validation_duration_ms': ['p(95)<50', 'p(99)<100'],
    'token_validation_success': ['rate>0.995'],

    // 列表接口
    'token_list_duration_ms': ['p(95)<200'],

    // 创建接口
    'token_creation_duration_ms': ['p(95)<500'],
    'token_creation_success': ['rate>0.99'],

    // 全局
    'http_req_failed': ['rate<0.01'],
  },
};

// ========================================
// 辅助函数
// ========================================

// 根据权重选择操作类型
function selectOperation() {
  const rand = Math.random() * 100;
  let cumulative = 0;

  for (const [op, weight] of Object.entries(TRAFFIC_WEIGHTS)) {
    cumulative += weight;
    if (rand < cumulative) return op;
  }
  return 'validate';
}

// ========================================
// 测试逻辑
// ========================================

export default function () {
  const operation = selectOperation();

  switch (operation) {
    case 'validate':
      validateToken();
      break;
    case 'list':
      listTokens();
      break;
    case 'create':
      createToken();
      break;
    case 'getInfo':
      getTokenInfo();
      break;
  }

  // 50-150ms 随机间隔
  sleep(0.05 + Math.random() * 0.1);
}

function validateToken() {
  group('Validate Token', function () {
    const token = tokens[Math.floor(Math.random() * tokens.length)];
    const start = Date.now();

    const res = http.post(`${BASE_URL}/api/v2/validate`, null, bearerAuth(token));

    const duration = Date.now() - start;
    tokenValidationDuration.add(duration);
    recordCacheHit(duration);
    checkValidateResponseLenient(res);
  });
}

function listTokens() {
  group('List Tokens', function () {
    const start = Date.now();

    const res = http.get(
      `${BASE_URL}/api/v2/tokens?limit=20`,
      qiniuStubAuth(TEST_UID)
    );

    tokenListDuration.add(Date.now() - start);
    checkListResponse(res);
  });
}

function createToken() {
  group('Create Token', function () {
    const start = Date.now();

    const payload = JSON.stringify({
      description: `Load test token - ${Date.now()}`,
      expires_in_seconds: 3600,  // 1 小时后过期
    });

    const res = http.post(
      `${BASE_URL}/api/v2/tokens`,
      payload,
      qiniuStubAuth(TEST_UID)
    );

    tokenCreationDuration.add(Date.now() - start);
    checkCreateResponse(res);
  });
}

function getTokenInfo() {
  group('Get Token Info', function () {
    // 先获取列表，再获取详情
    const listRes = http.get(
      `${BASE_URL}/api/v2/tokens?limit=1`,
      qiniuStubAuth(TEST_UID)
    );

    try {
      const body = JSON.parse(listRes.body);
      if (body.tokens && body.tokens.length > 0) {
        const tokenId = body.tokens[0].token_id;
        http.get(
          `${BASE_URL}/api/v2/tokens/${tokenId}`,
          qiniuStubAuth(TEST_UID)
        );
      }
    } catch (e) {
      // 忽略解析错误
    }
  });
}

// ========================================
// 结果汇总
// ========================================

export function handleSummary(data) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);

  console.log('\n========== 混合负载测试分析 ==========');
  console.log(`测试时间: ${timestamp}`);
  console.log(`目标服务: ${BASE_URL}`);
  console.log(`流量分布: 验证${TRAFFIC_WEIGHTS.validate}% + 列表${TRAFFIC_WEIGHTS.list}% + 创建${TRAFFIC_WEIGHTS.create}% + 详情${TRAFFIC_WEIGHTS.getInfo}%`);

  if (data.metrics.http_reqs) {
    console.log(`总请求数: ${data.metrics.http_reqs.values.count}`);
    console.log(`平均 RPS: ${data.metrics.http_reqs.values.rate.toFixed(2)}`);
  }

  console.log('========================================\n');

  const resultFile = `results/mixed-load-${timestamp}.json`;

  return {
    [resultFile]: JSON.stringify(data, null, 2),
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
  };
}
