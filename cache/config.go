package cache

import (
	"os"
	"strconv"
	"time"
)

// RedisConfig Redis 配置
type RedisConfig struct {
	Enabled       bool          // 是否启用 Redis 缓存
	Addr          string        // Redis 地址
	Password      string        // Redis 密码
	DB            int           // Redis 数据库编号
	MaxRetries    int           // 最大重试次数
	PoolSize      int           // 连接池大小
	MinIdleConns  int           // 最小空闲连接数
	TokenCacheTTL time.Duration // Token 缓存过期时间
}

// LoadRedisConfig 加载 Redis 配置
func LoadRedisConfig() *RedisConfig {
	return &RedisConfig{
		Enabled:       getEnvBool("REDIS_ENABLED", false),
		Addr:          getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
		Password:      os.Getenv("REDIS_PASSWORD"),
		DB:            getEnvInt("REDIS_DB", 0),
		MaxRetries:    getEnvInt("REDIS_MAX_RETRIES", 3),
		PoolSize:      getEnvInt("REDIS_POOL_SIZE", 10),
		MinIdleConns:  getEnvInt("REDIS_MIN_IDLE_CONNS", 2),
		TokenCacheTTL: parseDuration("CACHE_TOKEN_TTL", 5*time.Minute),
	}
}

// getEnvOrDefault 获取环境变量，如果不存在则返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool 获取布尔型环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvInt 获取整型环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// parseDuration 解析时长环境变量
func parseDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
