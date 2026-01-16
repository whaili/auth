package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/qiniu/bearer-token-service/v2/interfaces"
	"github.com/qiniu/bearer-token-service/v2/observability"
)

// ========================================
// HTTP 限流中间件
// ========================================

// Middleware 限流中间件
type Middleware struct {
	manager        *RateLimitManager
	accountRepo    interfaces.AccountRepository
	tokenRepo      interfaces.TokenRepository
}

// NewMiddleware 创建限流中间件
func NewMiddleware(manager *RateLimitManager, accountRepo interfaces.AccountRepository, tokenRepo interfaces.TokenRepository) *Middleware {
	return &Middleware{
		manager:     manager,
		accountRepo: accountRepo,
		tokenRepo:   tokenRepo,
	}
}

// AppLimitMiddleware 应用层限流中间件
func (m *Middleware) AppLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.manager.IsAppLimitEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		start := time.Now()
		allowed, remaining, resetTime, err := m.manager.CheckAppLimit(ctx)

		// 记录限流检查耗时
		observability.RateLimitCheckDuration.Observe(time.Since(start).Seconds())

		// 设置限流响应头
		if remaining >= 0 {
			w.Header().Set("X-RateLimit-Limit-App", fmt.Sprintf("%d", m.manager.appLimit.RequestsPerMinute))
			w.Header().Set("X-RateLimit-Remaining-App", fmt.Sprintf("%d", remaining))
			if !resetTime.IsZero() {
				w.Header().Set("X-RateLimit-Reset-App", fmt.Sprintf("%d", resetTime.Unix()))
			}
		}

		if err != nil {
			m.respondError(w, http.StatusInternalServerError, "Rate limit check failed")
			return
		}

		if !allowed {
			// 记录限流命中
			observability.RateLimitHitsTotal.WithLabelValues("app").Inc()

			retryAfter := time.Until(resetTime).Seconds()
			if retryAfter < 0 {
				retryAfter = 0
			}
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter))
			m.respondError(w, http.StatusTooManyRequests, "Application rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AccountLimitMiddleware 账户层限流中间件
func (m *Middleware) AccountLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.manager.IsAccountLimitEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()

		// 从上下文获取 account_id（由认证中间件设置）
		accountID, ok := ctx.Value("account_id").(string)
		if !ok || accountID == "" {
			// 未认证的请求跳过账户限流
			next.ServeHTTP(w, r)
			return
		}

		// 获取账户限流配置
		account, err := m.accountRepo.GetByID(ctx, accountID)
		if err != nil || account == nil {
			// 无法获取账户信息，跳过限流（不阻塞正常流程）
			next.ServeHTTP(w, r)
			return
		}

		// 检查账户限流
		start := time.Now()
		allowed, remaining, resetTime, err := m.manager.CheckAccountLimit(ctx, accountID, account.RateLimit)

		// 记录限流检查耗时
		observability.RateLimitCheckDuration.Observe(time.Since(start).Seconds())

		// 设置限流响应头
		if account.RateLimit != nil && remaining >= 0 {
			w.Header().Set("X-RateLimit-Limit-Account", fmt.Sprintf("%d", getAccountLimitValue(account.RateLimit)))
			w.Header().Set("X-RateLimit-Remaining-Account", fmt.Sprintf("%d", remaining))
			if !resetTime.IsZero() {
				w.Header().Set("X-RateLimit-Reset-Account", fmt.Sprintf("%d", resetTime.Unix()))
			}
		}

		if err != nil {
			m.respondError(w, http.StatusInternalServerError, "Rate limit check failed")
			return
		}

		if !allowed {
			// 记录限流命中
			observability.RateLimitHitsTotal.WithLabelValues("account").Inc()

			retryAfter := time.Until(resetTime).Seconds()
			if retryAfter < 0 {
				retryAfter = 0
			}
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter))
			m.respondError(w, http.StatusTooManyRequests, "Account rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// TokenLimitMiddleware Token层限流中间件（用于 /validate 接口）
func (m *Middleware) TokenLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.manager.IsTokenLimitEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()

		// 从上下文获取 token（由验证逻辑设置）
		tokenValue, ok := ctx.Value("token_value").(string)
		if !ok || tokenValue == "" {
			// 未提供 token，跳过限流
			next.ServeHTTP(w, r)
			return
		}

		// 获取 Token 信息
		token, err := m.tokenRepo.GetByTokenValue(ctx, tokenValue)
		if err != nil || token == nil {
			// 无法获取 Token 信息，跳过限流（让后续验证逻辑处理）
			next.ServeHTTP(w, r)
			return
		}

		// 检查 Token 限流
		start := time.Now()
		allowed, remaining, resetTime, err := m.manager.CheckTokenLimit(ctx, token.ID, token.RateLimit)

		// 记录限流检查耗时
		observability.RateLimitCheckDuration.Observe(time.Since(start).Seconds())

		// 设置限流响应头
		if token.RateLimit != nil && remaining >= 0 {
			w.Header().Set("X-RateLimit-Limit-Token", fmt.Sprintf("%d", getTokenLimitValue(token.RateLimit)))
			w.Header().Set("X-RateLimit-Remaining-Token", fmt.Sprintf("%d", remaining))
			if !resetTime.IsZero() {
				w.Header().Set("X-RateLimit-Reset-Token", fmt.Sprintf("%d", resetTime.Unix()))
			}
		}

		if err != nil {
			m.respondError(w, http.StatusInternalServerError, "Rate limit check failed")
			return
		}

		if !allowed {
			// 记录限流命中
			observability.RateLimitHitsTotal.WithLabelValues("token").Inc()

			retryAfter := time.Until(resetTime).Seconds()
			if retryAfter < 0 {
				retryAfter = 0
			}
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter))
			m.respondError(w, http.StatusTooManyRequests, "Token rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// respondError 返回错误响应
func (m *Middleware) respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   message,
		"code":    statusCode,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// getAccountLimitValue 获取账户限流值（优先返回分钟级）
func getAccountLimitValue(limit *interfaces.RateLimit) int {
	if limit.RequestsPerMinute > 0 {
		return limit.RequestsPerMinute
	}
	if limit.RequestsPerHour > 0 {
		return limit.RequestsPerHour
	}
	if limit.RequestsPerDay > 0 {
		return limit.RequestsPerDay
	}
	return 0
}

// getTokenLimitValue 获取Token限流值（优先返回分钟级）
func getTokenLimitValue(limit *interfaces.RateLimit) int {
	if limit.RequestsPerMinute > 0 {
		return limit.RequestsPerMinute
	}
	if limit.RequestsPerHour > 0 {
		return limit.RequestsPerHour
	}
	if limit.RequestsPerDay > 0 {
		return limit.RequestsPerDay
	}
	return 0
}

// ========================================
// 辅助函数：在验证 Handler 中设置 token_value 到上下文
// ========================================

// SetTokenToContext 设置 Token 到上下文（供中间件使用）
func SetTokenToContext(r *http.Request, tokenValue string) *http.Request {
	ctx := context.WithValue(r.Context(), "token_value", tokenValue)
	return r.WithContext(ctx)
}
