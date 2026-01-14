package config

import (
	"os"
	"strconv"

	"bearer-token-service.v1/v2/interfaces"
)

// ========================================
// 限流配置管理
// ========================================

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// 应用层限流开关
	EnableAppLimit bool

	// 账户层限流开关
	EnableAccountLimit bool

	// Token层限流开关
	EnableTokenLimit bool

	// 应用层限流配置
	AppLimitPerMinute int
	AppLimitPerHour   int
	AppLimitPerDay    int
}

// LoadRateLimitConfig 从环境变量加载限流配置
func LoadRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		// 限流功能开关（默认全部关闭）
		EnableAppLimit:     parseBool(os.Getenv("ENABLE_APP_RATE_LIMIT"), false),
		EnableAccountLimit: parseBool(os.Getenv("ENABLE_ACCOUNT_RATE_LIMIT"), false),
		EnableTokenLimit:   parseBool(os.Getenv("ENABLE_TOKEN_RATE_LIMIT"), false),

		// 应用层限流配置（仅在 ENABLE_APP_RATE_LIMIT=true 时生效）
		AppLimitPerMinute: parseInt(os.Getenv("APP_RATE_LIMIT_PER_MINUTE"), 1000),  // 默认 1000 req/min
		AppLimitPerHour:   parseInt(os.Getenv("APP_RATE_LIMIT_PER_HOUR"), 50000),   // 默认 50000 req/hour
		AppLimitPerDay:    parseInt(os.Getenv("APP_RATE_LIMIT_PER_DAY"), 1000000),  // 默认 1000000 req/day
	}
}

// GetAppRateLimit 获取应用层限流配置
func (c RateLimitConfig) GetAppRateLimit() *interfaces.RateLimit {
	if !c.EnableAppLimit {
		return nil
	}

	return &interfaces.RateLimit{
		RequestsPerMinute: c.AppLimitPerMinute,
		RequestsPerHour:   c.AppLimitPerHour,
		RequestsPerDay:    c.AppLimitPerDay,
	}
}

// parseBool 解析布尔值（带默认值）
func parseBool(s string, defaultValue bool) bool {
	if s == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(s)
	if err != nil {
		return defaultValue
	}

	return value
}

// parseInt 解析整数（带默认值）
func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}

	if value < 0 {
		return defaultValue
	}

	return value
}
