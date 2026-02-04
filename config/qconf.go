package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/qiniu/bearer-token-service/v2/pkg/qconfapi"
)

// QconfConfig qconfapi 配置
type QconfConfig struct {
	Enabled     bool     // 是否启用 qconfapi
	AccessKey   string   // qconf AccessKey
	SecretKey   string   // qconf SecretKey
	MasterHosts []string // qconf master 主机列表
	McHosts     []string // Memcache 主机列表（可选）

	// 缓存配置
	LcacheExpiresMS  int   // 本地缓存过期时间（毫秒）
	LcacheDurationMS int   // 本地缓存刷新间隔（毫秒）
	LcacheChanBufSize int  // 异步消息队列缓冲区大小
	McExpiresS       int32 // Memcache 过期时间（秒）
	McRWTimeoutMS    int64 // Memcache 读写超时（毫秒）
}

// LoadQconfConfig 从环境变量加载 qconfapi 配置
func LoadQconfConfig() *QconfConfig {
	return &QconfConfig{
		Enabled:     getEnvBoolForQconf("QCONF_ENABLED", false),
		AccessKey:   getEnvForQconf("QCONF_ACCESS_KEY", ""),
		SecretKey:   getEnvForQconf("QCONF_SECRET_KEY", ""),
		MasterHosts: parseCommaSeparatedForQconf(getEnvForQconf("QCONF_MASTER_HOSTS", "")),
		McHosts:     parseCommaSeparatedForQconf(getEnvForQconf("QCONF_MC_HOSTS", "")),

		LcacheExpiresMS:  getEnvIntForQconf("QCONF_LC_EXPIRES_MS", 300000),  // 默认 5 分钟
		LcacheDurationMS: getEnvIntForQconf("QCONF_LC_DURATION_MS", 60000),  // 默认 1 分钟
		LcacheChanBufSize: getEnvIntForQconf("QCONF_LC_CHAN_BUFSIZE", 1000),
		McExpiresS:       int32(getEnvIntForQconf("QCONF_MC_EXPIRES_S", 300)),      // 默认 5 分钟
		McRWTimeoutMS:    int64(getEnvIntForQconf("QCONF_MC_RW_TIMEOUT_MS", 1000)), // 默认 1 秒
	}
}

// ToQconfapiConfig 转换为 qconfapi.Config
func (c *QconfConfig) ToQconfapiConfig() *qconfapi.Config {
	return &qconfapi.Config{
		AccessKey:         c.AccessKey,
		SecretKey:         c.SecretKey,
		MasterHosts:       c.MasterHosts,
		McHosts:           c.McHosts,
		LcacheExpires:     c.LcacheExpiresMS,
		LcacheDuration:    c.LcacheDurationMS,
		LcacheChanBufSize: c.LcacheChanBufSize,
		McExpires:         c.McExpiresS,
		McRWTimeout:       c.McRWTimeoutMS,
	}
}

// IsValid 检查配置是否有效
func (c *QconfConfig) IsValid() bool {
	return c.Enabled && c.AccessKey != "" && c.SecretKey != "" && len(c.MasterHosts) > 0
}

// getEnvForQconf 获取环境变量（qconf 专用，避免与其他 config 文件冲突）
func getEnvForQconf(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func getEnvBoolForQconf(key string, defaultValue bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return b
}

func getEnvIntForQconf(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return i
}

func parseCommaSeparatedForQconf(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
