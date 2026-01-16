package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/qiniu/bearer-token-service/v2/interfaces"
	"github.com/qiniu/bearer-token-service/v2/observability"
)

// TokenCache Token 缓存接口
type TokenCache interface {
	GetByTokenValue(ctx context.Context, tokenValue string) (*interfaces.Token, error)
	GetByID(ctx context.Context, tokenID string) (*interfaces.Token, error)
	InvalidateByTokenValue(ctx context.Context, tokenValue string) error
	InvalidateByID(ctx context.Context, tokenID string) error
}

// DirectTokenFetcher 直接数据库查询接口（绕过缓存，避免循环调用）
type DirectTokenFetcher interface {
	GetByTokenValueDirect(ctx context.Context, tokenValue string) (*interfaces.Token, error)
	GetByIDDirect(ctx context.Context, tokenID string) (*interfaces.Token, error)
}

// TokenCacheImpl Token 缓存实现
type TokenCacheImpl struct {
	redis   RedisClient
	fetcher DirectTokenFetcher
	baseTTL time.Duration
}

// NewTokenCache 创建 Token 缓存
func NewTokenCache(redis RedisClient, fetcher DirectTokenFetcher, baseTTL time.Duration) TokenCache {
	return &TokenCacheImpl{
		redis:   redis,
		fetcher: fetcher,
		baseTTL: baseTTL,
	}
}

// GetByTokenValue 通过 TokenValue 获取 Token（含缓存）
func (c *TokenCacheImpl) GetByTokenValue(ctx context.Context, tokenValue string) (*interfaces.Token, error) {
	cacheKey := fmt.Sprintf("token:val:%s", tokenValue)
	start := time.Now()

	// 1. 尝试从 Redis 读取
	cached, err := c.redis.Get(ctx, cacheKey)
	if err == nil {
		// 缓存命中
		observability.CacheOperationsTotal.WithLabelValues("get", "hit").Inc()
		observability.CacheOperationDuration.WithLabelValues("get").Observe(time.Since(start).Seconds())

		if cached == "null" {
			// 空对象缓存（防穿透）
			return nil, nil
		}

		var token interfaces.Token
		if err := json.Unmarshal([]byte(cached), &token); err == nil {
			return &token, nil
		}
	} else {
		// 缓存未命中
		observability.CacheOperationsTotal.WithLabelValues("get", "miss").Inc()
	}

	// 2. Redis 未命中或出错，降级到 MongoDB（使用 Direct 方法避免循环调用）
	token, err := c.fetcher.GetByTokenValueDirect(ctx, tokenValue)
	if err != nil {
		observability.CacheOperationsTotal.WithLabelValues("get", "error").Inc()
		return nil, err
	}

	// 3. 异步写入缓存（包括空对象）
	go c.cacheToken(context.Background(), cacheKey, token)

	return token, nil
}

// GetByID 通过 ID 获取 Token（含缓存）
func (c *TokenCacheImpl) GetByID(ctx context.Context, tokenID string) (*interfaces.Token, error) {
	cacheKey := fmt.Sprintf("token:id:%s", tokenID)
	start := time.Now()

	// 1. 尝试从 Redis 读取
	cached, err := c.redis.Get(ctx, cacheKey)
	if err == nil {
		// 缓存命中
		observability.CacheOperationsTotal.WithLabelValues("get", "hit").Inc()
		observability.CacheOperationDuration.WithLabelValues("get").Observe(time.Since(start).Seconds())

		if cached == "null" {
			return nil, nil
		}

		var token interfaces.Token
		if err := json.Unmarshal([]byte(cached), &token); err == nil {
			return &token, nil
		}
	} else {
		// 缓存未命中
		observability.CacheOperationsTotal.WithLabelValues("get", "miss").Inc()
	}

	// 2. Redis 未命中或出错，降级到 MongoDB（使用 Direct 方法避免循环调用）
	token, err := c.fetcher.GetByIDDirect(ctx, tokenID)
	if err != nil {
		observability.CacheOperationsTotal.WithLabelValues("get", "error").Inc()
		return nil, err
	}

	// 3. 异步写入缓存
	go c.cacheToken(context.Background(), cacheKey, token)

	return token, nil
}

// InvalidateByTokenValue 失效缓存（通过 TokenValue）
func (c *TokenCacheImpl) InvalidateByTokenValue(ctx context.Context, tokenValue string) error {
	return c.redis.Del(ctx, fmt.Sprintf("token:val:%s", tokenValue))
}

// InvalidateByID 失效缓存（通过 ID）
func (c *TokenCacheImpl) InvalidateByID(ctx context.Context, tokenID string) error {
	return c.redis.Del(ctx, fmt.Sprintf("token:id:%s", tokenID))
}

// cacheToken 写入缓存（带 TTL 抖动 + 空对象缓存）
func (c *TokenCacheImpl) cacheToken(ctx context.Context, cacheKey string, token *interfaces.Token) {
	start := time.Now()

	if token == nil {
		// 缓存空对象（防穿透），TTL: 1分钟
		err := c.redis.Set(ctx, cacheKey, "null", 1*time.Minute)
		if err != nil {
			observability.CacheOperationsTotal.WithLabelValues("set", "error").Inc()
		} else {
			observability.CacheOperationsTotal.WithLabelValues("set", "success").Inc()
			observability.CacheOperationDuration.WithLabelValues("set").Observe(time.Since(start).Seconds())
		}
		return
	}

	// 序列化 Token
	data, err := json.Marshal(token)
	if err != nil {
		return
	}

	// TTL 加随机抖动（± 10%），防缓存雪崩
	jitter := time.Duration(rand.Intn(int(c.baseTTL.Seconds() / 10)))
	ttl := c.baseTTL + jitter*time.Second

	err = c.redis.Set(ctx, cacheKey, data, ttl)
	if err != nil {
		observability.CacheOperationsTotal.WithLabelValues("set", "error").Inc()
	} else {
		observability.CacheOperationsTotal.WithLabelValues("set", "success").Inc()
		observability.CacheOperationDuration.WithLabelValues("set").Observe(time.Since(start).Seconds())
	}
}
