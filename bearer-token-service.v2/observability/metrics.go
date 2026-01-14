package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ========================================
	// HTTP 请求指标
	// ========================================

	// HTTPRequestsTotal HTTP 请求总数
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	// HTTPRequestDuration HTTP 请求延迟分布
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"method", "endpoint"},
	)

	// HTTPRequestsInFlight 当前在途请求数
	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
	)

	// ========================================
	// Token 验证指标
	// ========================================

	// TokenValidationsTotal Token 验证总数
	TokenValidationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "token_validations_total",
			Help: "Total number of token validation requests",
		},
		[]string{"result"}, // valid, invalid, expired, inactive, not_found, error
	)

	// TokenValidationDuration Token 验证延迟
	TokenValidationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "token_validation_duration_seconds",
			Help:    "Token validation latency in seconds",
			Buckets: []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1},
		},
	)

	// ========================================
	// 限流指标
	// ========================================

	// RateLimitHitsTotal 限流拒绝次数
	RateLimitHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_hits_total",
			Help: "Total number of rate limit hits (rejected requests)",
		},
		[]string{"level"}, // app, account, token
	)

	// RateLimitCheckDuration 限流检查延迟
	RateLimitCheckDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "rate_limit_check_duration_seconds",
			Help:    "Rate limit check latency in seconds",
			Buckets: []float64{0.00001, 0.00005, 0.0001, 0.0005, 0.001, 0.005},
		},
	)

	// ========================================
	// 缓存指标
	// ========================================

	// CacheOperationsTotal 缓存操作总数
	CacheOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation", "result"}, // get/set/del, hit/miss/error
	)

	// CacheOperationDuration 缓存操作延迟
	CacheOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_operation_duration_seconds",
			Help:    "Cache operation latency in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025},
		},
		[]string{"operation"},
	)

	// ========================================
	// 业务指标
	// ========================================

	// TokensCreatedTotal 创建的 Token 总数
	TokensCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tokens_created_total",
			Help: "Total number of tokens created",
		},
	)

	// TokensDeletedTotal 删除的 Token 总数
	TokensDeletedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tokens_deleted_total",
			Help: "Total number of tokens deleted",
		},
	)
)
