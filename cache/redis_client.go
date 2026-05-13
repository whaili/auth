package cache

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient Redis 客户端接口
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Ping(ctx context.Context) error
	Close() error
}

// NewRedisClient 创建 Redis 客户端，自动适配单节点/集群模式。
// addr 支持两种格式:
//   - 单节点: "host:6379"
//   - 集群:   "host1:6379,host2:6379,host3:6379" (逗号分隔)
func NewRedisClient(addr, password string, db, poolSize, minIdleConns, maxRetries int) (RedisClient, error) {
	addrs := splitAddrs(addr)

	if len(addrs) > 1 {
		return newClusterClient(addrs, password, poolSize, minIdleConns, maxRetries)
	}
	return newSingleClient(addr, password, db, poolSize, minIdleConns, maxRetries)
}

func splitAddrs(addr string) []string {
	parts := strings.Split(addr, ",")
	var result []string
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// === 单节点 ===

type singleClient struct {
	client *redis.Client
}

func newSingleClient(addr, password string, db, poolSize, minIdleConns, maxRetries int) (RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        password,
		DB:              db,
		PoolSize:        poolSize,
		MinIdleConns:    minIdleConns,
		MaxRetries:      maxRetries,
		DisableIndentity: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &singleClient{client: client}, nil
}

func (c *singleClient) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *singleClient) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *singleClient) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

func (c *singleClient) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *singleClient) Close() error {
	return c.client.Close()
}

// === 集群 ===

type clusterClient struct {
	client *redis.ClusterClient
}

func newClusterClient(addrs []string, password string, poolSize, minIdleConns, maxRetries int) (RedisClient, error) {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:           addrs,
		Password:        password,
		PoolSize:        poolSize,
		MinIdleConns:    minIdleConns,
		MaxRetries:      maxRetries,
		DisableIndentity: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &clusterClient{client: client}, nil
}

func (c *clusterClient) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *clusterClient) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *clusterClient) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

func (c *clusterClient) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *clusterClient) Close() error {
	return c.client.Close()
}
