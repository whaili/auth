package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bearer-token-service.v1/v2/interfaces"
)

// ========================================
// 限流器接口
// ========================================

// Limiter 限流器接口
type Limiter interface {
	// Allow 检查是否允许请求
	// key: 限流键（如 account_id, token_id）
	// Returns: (allowed bool, remaining int, resetTime time.Time)
	Allow(ctx context.Context, key string, limit *interfaces.RateLimit) (bool, int, time.Time, error)
}

// ========================================
// 内存限流器实现（基于滑动窗口）
// ========================================

// MemoryLimiter 内存限流器（使用滑动窗口算法）
type MemoryLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*slidingWindow
	// 自动清理过期数据
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// slidingWindow 滑动窗口
type slidingWindow struct {
	mu            sync.Mutex
	minuteWindow  *timeWindow
	hourWindow    *timeWindow
	dayWindow     *timeWindow
}

// timeWindow 时间窗口
type timeWindow struct {
	timestamps []time.Time
	limit      int
	duration   time.Duration
}

// NewMemoryLimiter 创建内存限流器
func NewMemoryLimiter() *MemoryLimiter {
	limiter := &MemoryLimiter{
		buckets:         make(map[string]*slidingWindow),
		cleanupInterval: 5 * time.Minute,
		stopCleanup:     make(chan struct{}),
	}

	// 启动自动清理协程
	go limiter.cleanupExpiredBuckets()

	return limiter
}

// Allow 检查是否允许请求
func (l *MemoryLimiter) Allow(ctx context.Context, key string, limit *interfaces.RateLimit) (bool, int, time.Time, error) {
	if limit == nil {
		// 没有限流配置，直接允许
		return true, -1, time.Time{}, nil
	}

	now := time.Now()

	// 获取或创建滑动窗口
	window := l.getOrCreateWindow(key, limit)
	window.mu.Lock()
	defer window.mu.Unlock()

	// 按优先级检查限流（分钟 > 小时 > 天）
	// 1. 检查分钟级限流
	if limit.RequestsPerMinute > 0 {
		if !window.minuteWindow.allow(now) {
			remaining := window.minuteWindow.remaining(now)
			resetTime := window.minuteWindow.resetTime(now)
			return false, remaining, resetTime, nil
		}
	}

	// 2. 检查小时级限流
	if limit.RequestsPerHour > 0 {
		if !window.hourWindow.allow(now) {
			remaining := window.hourWindow.remaining(now)
			resetTime := window.hourWindow.resetTime(now)
			return false, remaining, resetTime, nil
		}
	}

	// 3. 检查天级限流
	if limit.RequestsPerDay > 0 {
		if !window.dayWindow.allow(now) {
			remaining := window.dayWindow.remaining(now)
			resetTime := window.dayWindow.resetTime(now)
			return false, remaining, resetTime, nil
		}
	}

	// 所有检查通过，记录请求
	window.minuteWindow.record(now)
	window.hourWindow.record(now)
	window.dayWindow.record(now)

	// 计算剩余请求数（取最小值）
	remaining := l.calculateRemaining(window, limit, now)
	resetTime := l.calculateResetTime(window, limit, now)

	return true, remaining, resetTime, nil
}

// getOrCreateWindow 获取或创建滑动窗口
func (l *MemoryLimiter) getOrCreateWindow(key string, limit *interfaces.RateLimit) *slidingWindow {
	l.mu.RLock()
	window, exists := l.buckets[key]
	l.mu.RUnlock()

	if exists {
		return window
	}

	// 创建新窗口
	l.mu.Lock()
	defer l.mu.Unlock()

	// 双重检查
	if window, exists := l.buckets[key]; exists {
		return window
	}

	window = &slidingWindow{
		minuteWindow: newTimeWindow(limit.RequestsPerMinute, 1*time.Minute),
		hourWindow:   newTimeWindow(limit.RequestsPerHour, 1*time.Hour),
		dayWindow:    newTimeWindow(limit.RequestsPerDay, 24*time.Hour),
	}

	l.buckets[key] = window
	return window
}

// calculateRemaining 计算剩余请求数
func (l *MemoryLimiter) calculateRemaining(window *slidingWindow, limit *interfaces.RateLimit, now time.Time) int {
	remaining := -1 // -1 表示无限制

	if limit.RequestsPerMinute > 0 {
		r := window.minuteWindow.remaining(now)
		if remaining == -1 || r < remaining {
			remaining = r
		}
	}

	if limit.RequestsPerHour > 0 {
		r := window.hourWindow.remaining(now)
		if remaining == -1 || r < remaining {
			remaining = r
		}
	}

	if limit.RequestsPerDay > 0 {
		r := window.dayWindow.remaining(now)
		if remaining == -1 || r < remaining {
			remaining = r
		}
	}

	return remaining
}

// calculateResetTime 计算重置时间
func (l *MemoryLimiter) calculateResetTime(window *slidingWindow, limit *interfaces.RateLimit, now time.Time) time.Time {
	var resetTime time.Time

	if limit.RequestsPerMinute > 0 {
		t := window.minuteWindow.resetTime(now)
		if resetTime.IsZero() || t.Before(resetTime) {
			resetTime = t
		}
	}

	if limit.RequestsPerHour > 0 {
		t := window.hourWindow.resetTime(now)
		if resetTime.IsZero() || t.Before(resetTime) {
			resetTime = t
		}
	}

	if limit.RequestsPerDay > 0 {
		t := window.dayWindow.resetTime(now)
		if resetTime.IsZero() || t.Before(resetTime) {
			resetTime = t
		}
	}

	return resetTime
}

// cleanupExpiredBuckets 清理过期的桶
func (l *MemoryLimiter) cleanupExpiredBuckets() {
	ticker := time.NewTicker(l.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.cleanup()
		case <-l.stopCleanup:
			return
		}
	}
}

// cleanup 执行清理
func (l *MemoryLimiter) cleanup() {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	for key, window := range l.buckets {
		window.mu.Lock()
		isEmpty := window.minuteWindow.isEmpty(now) &&
			window.hourWindow.isEmpty(now) &&
			window.dayWindow.isEmpty(now)
		window.mu.Unlock()

		if isEmpty {
			delete(l.buckets, key)
		}
	}
}

// Stop 停止限流器
func (l *MemoryLimiter) Stop() {
	close(l.stopCleanup)
}

// ========================================
// 时间窗口实现
// ========================================

// newTimeWindow 创建时间窗口
func newTimeWindow(limit int, duration time.Duration) *timeWindow {
	return &timeWindow{
		timestamps: make([]time.Time, 0, limit),
		limit:      limit,
		duration:   duration,
	}
}

// allow 检查是否允许请求
func (w *timeWindow) allow(now time.Time) bool {
	if w.limit <= 0 {
		return true // 未配置限流
	}

	w.removeExpired(now)
	return len(w.timestamps) < w.limit
}

// record 记录请求
func (w *timeWindow) record(now time.Time) {
	if w.limit <= 0 {
		return
	}
	w.timestamps = append(w.timestamps, now)
}

// remaining 计算剩余请求数
func (w *timeWindow) remaining(now time.Time) int {
	if w.limit <= 0 {
		return -1
	}

	w.removeExpired(now)
	remaining := w.limit - len(w.timestamps)
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// resetTime 计算重置时间
func (w *timeWindow) resetTime(now time.Time) time.Time {
	if w.limit <= 0 || len(w.timestamps) == 0 {
		return now
	}

	// 最早的时间戳 + 窗口时长
	w.removeExpired(now)
	if len(w.timestamps) == 0 {
		return now
	}

	return w.timestamps[0].Add(w.duration)
}

// removeExpired 移除过期的时间戳
func (w *timeWindow) removeExpired(now time.Time) {
	if w.limit <= 0 {
		return
	}

	cutoff := now.Add(-w.duration)
	validIdx := 0

	for i, ts := range w.timestamps {
		if ts.After(cutoff) {
			validIdx = i
			break
		}
	}

	// 移除过期的时间戳
	if validIdx > 0 {
		w.timestamps = w.timestamps[validIdx:]
	} else if len(w.timestamps) > 0 && w.timestamps[len(w.timestamps)-1].Before(cutoff) {
		// 所有时间戳都过期了
		w.timestamps = w.timestamps[:0]
	}
}

// isEmpty 检查窗口是否为空
func (w *timeWindow) isEmpty(now time.Time) bool {
	if w.limit <= 0 {
		return true
	}

	w.removeExpired(now)
	return len(w.timestamps) == 0
}

// ========================================
// 限流管理器（统一管理三层限流）
// ========================================

// RateLimitManager 限流管理器
type RateLimitManager struct {
	limiter Limiter

	// 应用层限流配置
	appLimit *interfaces.RateLimit

	// 功能开关
	enableAppLimit     bool
	enableAccountLimit bool
	enableTokenLimit   bool
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// 应用层限流配置
	AppLimit *interfaces.RateLimit

	// 功能开关
	EnableAppLimit     bool
	EnableAccountLimit bool
	EnableTokenLimit   bool
}

// NewRateLimitManager 创建限流管理器
func NewRateLimitManager(limiter Limiter, config RateLimitConfig) *RateLimitManager {
	return &RateLimitManager{
		limiter:            limiter,
		appLimit:           config.AppLimit,
		enableAppLimit:     config.EnableAppLimit,
		enableAccountLimit: config.EnableAccountLimit,
		enableTokenLimit:   config.EnableTokenLimit,
	}
}

// CheckAppLimit 检查应用层限流
func (m *RateLimitManager) CheckAppLimit(ctx context.Context) (bool, int, time.Time, error) {
	if !m.enableAppLimit || m.appLimit == nil {
		return true, -1, time.Time{}, nil
	}

	return m.limiter.Allow(ctx, "app", m.appLimit)
}

// CheckAccountLimit 检查账户层限流
func (m *RateLimitManager) CheckAccountLimit(ctx context.Context, accountID string, limit *interfaces.RateLimit) (bool, int, time.Time, error) {
	if !m.enableAccountLimit || limit == nil {
		return true, -1, time.Time{}, nil
	}

	key := fmt.Sprintf("account:%s", accountID)
	return m.limiter.Allow(ctx, key, limit)
}

// CheckTokenLimit 检查Token层限流
func (m *RateLimitManager) CheckTokenLimit(ctx context.Context, tokenID string, limit *interfaces.RateLimit) (bool, int, time.Time, error) {
	if !m.enableTokenLimit || limit == nil {
		return true, -1, time.Time{}, nil
	}

	key := fmt.Sprintf("token:%s", tokenID)
	return m.limiter.Allow(ctx, key, limit)
}

// IsAppLimitEnabled 应用层限流是否启用
func (m *RateLimitManager) IsAppLimitEnabled() bool {
	return m.enableAppLimit
}

// IsAccountLimitEnabled 账户层限流是否启用
func (m *RateLimitManager) IsAccountLimitEnabled() bool {
	return m.enableAccountLimit
}

// IsTokenLimitEnabled Token层限流是否启用
func (m *RateLimitManager) IsTokenLimitEnabled() bool {
	return m.enableTokenLimit
}
