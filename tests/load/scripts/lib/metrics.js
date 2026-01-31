// Bearer Token Service V2 - 自定义指标
// 用于 k6 压测脚本

import { Counter, Trend, Rate, Gauge } from 'k6/metrics';

// ========================================
// Token 验证接口指标
// ========================================

// 验证延迟（毫秒）
export const tokenValidationDuration = new Trend('token_validation_duration_ms', true);

// 验证成功率
export const tokenValidationSuccess = new Rate('token_validation_success');

// 验证请求计数
export const tokenValidationTotal = new Counter('token_validation_total');

// ========================================
// Token 创建接口指标
// ========================================

// 创建延迟（毫秒）
export const tokenCreationDuration = new Trend('token_creation_duration_ms', true);

// 创建成功率
export const tokenCreationSuccess = new Rate('token_creation_success');

// ========================================
// Token 列表接口指标
// ========================================

// 列表延迟（毫秒）
export const tokenListDuration = new Trend('token_list_duration_ms', true);

// ========================================
// 缓存相关指标
// ========================================

// 缓存命中率（通过响应时间推断，<10ms 认为命中）
export const cacheHitRate = new Rate('cache_hit_rate');

// ========================================
// 错误统计
// ========================================

// 4xx 错误计数
export const errors4xx = new Counter('errors_4xx');

// 5xx 错误计数
export const errors5xx = new Counter('errors_5xx');

// 限流触发计数（429）
export const rateLimitHits = new Counter('rate_limit_hits');

// ========================================
// 系统指标
// ========================================

// 当前活跃 VU 数
export const activeVUs = new Gauge('active_vus');

// 请求队列深度（通过响应时间推断）
export const queueDepth = new Gauge('estimated_queue_depth');

// ========================================
// 辅助函数
// ========================================

/**
 * 根据响应状态码记录错误指标
 * @param {number} status - HTTP 状态码
 */
export function recordErrorByStatus(status) {
  if (status >= 400 && status < 500) {
    errors4xx.add(1);
    if (status === 429) {
      rateLimitHits.add(1);
    }
  } else if (status >= 500) {
    errors5xx.add(1);
  }
}

/**
 * 根据响应时间推断缓存命中
 * @param {number} durationMs - 响应时间（毫秒）
 * @param {number} threshold - 命中阈值（默认 10ms）
 */
export function recordCacheHit(durationMs, threshold = 10) {
  cacheHitRate.add(durationMs < threshold);
}
