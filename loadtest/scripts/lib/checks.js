// Bearer Token Service V2 - 响应校验函数
// 用于 k6 压测脚本

import { check } from 'k6';
import {
  tokenValidationSuccess,
  tokenCreationSuccess,
  recordErrorByStatus
} from './metrics.js';

/**
 * 校验 Token 验证响应
 * @param {object} res - k6 响应对象
 * @returns {boolean} - 校验是否通过
 */
export function checkValidateResponse(res) {
  const passed = check(res, {
    'validate: status is 200': (r) => r.status === 200,
    'validate: has valid field': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.hasOwnProperty('valid');
      } catch {
        return false;
      }
    },
    'validate: response time < 100ms': (r) => r.timings.duration < 100,
  });

  // 记录成功率
  tokenValidationSuccess.add(res.status === 200);

  // 记录错误
  recordErrorByStatus(res.status);

  return passed;
}

/**
 * 校验 Token 验证响应（宽松版，允许 401）
 * @param {object} res - k6 响应对象
 * @returns {boolean} - 校验是否通过
 */
export function checkValidateResponseLenient(res) {
  const passed = check(res, {
    'validate: status is 200 or 401': (r) => r.status === 200 || r.status === 401,
    'validate: has valid field': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.hasOwnProperty('valid');
      } catch {
        return false;
      }
    },
  });

  // 200 和 401 都算业务成功（401 是无效 token 的正常响应）
  tokenValidationSuccess.add(res.status === 200 || res.status === 401);

  // 只记录非业务错误
  if (res.status !== 200 && res.status !== 401) {
    recordErrorByStatus(res.status);
  }

  return passed;
}

/**
 * 校验 Token 创建响应
 * @param {object} res - k6 响应对象
 * @returns {boolean} - 校验是否通过
 */
export function checkCreateResponse(res) {
  const passed = check(res, {
    'create: status is 201': (r) => r.status === 201,
    'create: has token_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.token_id && body.token;
      } catch {
        return false;
      }
    },
    'create: response time < 500ms': (r) => r.timings.duration < 500,
  });

  tokenCreationSuccess.add(res.status === 201);
  recordErrorByStatus(res.status);

  return passed;
}

/**
 * 校验 Token 列表响应
 * @param {object} res - k6 响应对象
 * @returns {boolean} - 校验是否通过
 */
export function checkListResponse(res) {
  const passed = check(res, {
    'list: status is 200': (r) => r.status === 200,
    'list: has tokens array': (r) => {
      try {
        const body = JSON.parse(r.body);
        return Array.isArray(body.tokens);
      } catch {
        return false;
      }
    },
    'list: response time < 200ms': (r) => r.timings.duration < 200,
  });

  recordErrorByStatus(res.status);

  return passed;
}

/**
 * 校验健康检查响应
 * @param {object} res - k6 响应对象
 * @returns {boolean} - 校验是否通过
 */
export function checkHealthResponse(res) {
  return check(res, {
    'health: status is 200': (r) => r.status === 200,
    'health: response time < 50ms': (r) => r.timings.duration < 50,
  });
}

/**
 * 从响应中提取 Token 信息
 * @param {object} res - k6 响应对象
 * @returns {object|null} - Token 信息或 null
 */
export function extractTokenFromResponse(res) {
  try {
    const body = JSON.parse(res.body);
    if (body.token_id && body.token) {
      return {
        tokenId: body.token_id,
        tokenValue: body.token,
      };
    }
  } catch {
    // 忽略解析错误
  }
  return null;
}
