// Bearer Token Service V2 - 测试数据生成脚本
// 用于生成压测所需的测试 Token

import http from 'k6/http';
import { check, sleep } from 'k6';
import { SharedArray } from 'k6/data';

import { qiniuStubAuth } from '../lib/auth.js';

// ========================================
// 配置
// ========================================

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';
const TEST_UID = __ENV.TEST_UID || '1369077332';
const TOKEN_COUNT = parseInt(__ENV.TOKEN_COUNT) || 100;  // 默认生成 100 个

// 存储生成的 Token
const generatedTokens = [];

// ========================================
// 测试配置
// ========================================

export const options = {
  scenarios: {
    setup_tokens: {
      executor: 'shared-iterations',
      vus: 10,                    // 10 个并发
      iterations: TOKEN_COUNT,    // 总共生成 TOKEN_COUNT 个
      maxDuration: '5m',          // 最多 5 分钟
    },
  },
};

// ========================================
// Setup: 检查服务健康
// ========================================

export function setup() {
  console.log(`准备生成 ${TOKEN_COUNT} 个测试 Token...`);
  console.log(`目标服务: ${BASE_URL}`);
  console.log(`测试 UID: ${TEST_UID}`);

  // 健康检查
  const healthRes = http.get(`${BASE_URL}/health`);
  if (healthRes.status !== 200) {
    throw new Error(`服务不健康: ${healthRes.status}`);
  }

  console.log('服务健康检查通过\n');

  return { tokens: [] };
}

// ========================================
// 主逻辑: 创建 Token
// ========================================

export default function (data) {
  const payload = JSON.stringify({
    description: `Load test token - ${Date.now()}-${__VU}-${__ITER}`,
    expires_in_seconds: 86400,  // 24 小时后过期
  });

  const res = http.post(
    `${BASE_URL}/api/v2/tokens`,
    payload,
    qiniuStubAuth(TEST_UID)
  );

  const passed = check(res, {
    'create: status is 201': (r) => r.status === 201,
    'create: has token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.token && body.token_id;
      } catch {
        return false;
      }
    },
  });

  if (passed) {
    try {
      const body = JSON.parse(res.body);
      generatedTokens.push(body.token);
      console.log(`[${generatedTokens.length}/${TOKEN_COUNT}] Created: ${body.token_id}`);
    } catch (e) {
      console.log(`解析响应失败: ${e.message}`);
    }
  } else {
    console.log(`创建失败: ${res.status} - ${res.body}`);
  }

  // 避免请求过快
  sleep(0.1);
}

// ========================================
// Teardown: 输出结果
// ========================================

export function teardown(data) {
  console.log('\n========================================');
  console.log(`成功生成 ${generatedTokens.length} 个 Token`);
  console.log('========================================\n');

  if (generatedTokens.length > 0) {
    // 输出到控制台（可重定向到文件）
    console.log('--- 生成的 Token 列表 ---');
    console.log('请将以下内容保存到 loadtest/data/tokens.csv:');
    console.log('');
    generatedTokens.forEach(token => {
      console.log(token);
    });
    console.log('');
    console.log('--- 列表结束 ---');
  }
}
