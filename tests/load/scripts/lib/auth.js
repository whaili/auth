// Bearer Token Service V2 - 认证辅助函数
// 用于 k6 压测脚本

/**
 * 生成 Bearer Token 认证头
 * @param {string} token - Bearer Token 值
 * @returns {object} - 包含认证头的请求参数
 */
export function bearerAuth(token) {
  return {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
  };
}

/**
 * 生成 QiniuStub 认证头（用于 Token 管理 API）
 * @param {string|number} uid - 用户 UID
 * @param {number} ut - 用户类型，默认 1
 * @param {string|number|null} iuid - IAM 子账户 ID（可选）
 * @returns {object} - 包含认证头的请求参数
 */
export function qiniuStubAuth(uid, ut = 1, iuid = null) {
  let authValue = `QiniuStub uid=${uid}&ut=${ut}`;
  if (iuid) {
    authValue += `&iuid=${iuid}`;
  }
  return {
    headers: {
      'Authorization': authValue,
      'Content-Type': 'application/json',
    },
  };
}

/**
 * 合并请求参数
 * @param {object} auth - 认证参数
 * @param {object} extra - 额外参数
 * @returns {object} - 合并后的参数
 */
export function mergeParams(auth, extra = {}) {
  return {
    ...auth,
    headers: {
      ...auth.headers,
      ...(extra.headers || {}),
    },
    ...extra,
  };
}
