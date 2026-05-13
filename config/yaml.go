package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// CommaSep 逗号分隔字符串，YAML 解析时兼容数组和字符串两种格式
type CommaSep string

func (c *CommaSep) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		*c = CommaSep(value.Value)
	case yaml.SequenceNode:
		var parts []string
		for _, n := range value.Content {
			parts = append(parts, n.Value)
		}
		*c = CommaSep(strings.Join(parts, ","))
	}
	return nil
}

func (c CommaSep) String() string { return string(c) }

// AppYAML YAML 配置文件结构
type AppYAML struct {
	Mongo  MongoYAML  `yaml:"mongo"`
	Redis  RedisYAML  `yaml:"redis"`
	Qconf  QconfYAML  `yaml:"qconf"`
	Server ServerYAML `yaml:"server"`
	Rate   RateYAML   `yaml:"rate_limit"`
}

type MongoYAML struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
}

type RedisYAML struct {
	Enabled         bool   `yaml:"enabled"`
	Addr            string `yaml:"addr"`
	Password        string `yaml:"password"`
	DB              int    `yaml:"db"`
	PoolSize        int    `yaml:"pool_size"`
	MinIdleConns    int    `yaml:"min_idle_conns"`
	MaxRetries      int    `yaml:"max_retries"`
	TokenCacheTTL   string `yaml:"token_cache_ttl"`
}

type QconfYAML struct {
	Enabled          bool     `yaml:"enabled"`
	AccessKey        string   `yaml:"access_key"`
	SecretKey        string   `yaml:"secret_key"`
	MasterHosts      CommaSep `yaml:"master_hosts"`
	LcacheExpiresMS  int      `yaml:"lcache_expires_ms"`
	LcacheDurationMS int      `yaml:"lcache_duration_ms"`
	LcacheChanBufSize int     `yaml:"lcache_chan_bufsize"`
	McExpiresS       int      `yaml:"mc_expires_s"`
	McRWTimeoutMS    int      `yaml:"mc_rw_timeout_ms"`
}

type ServerYAML struct {
	Port                 string `yaml:"port"`
	LogLevel             string `yaml:"log_level"`
	LogFormat            string `yaml:"log_format"`
	GinMode              string `yaml:"gin_mode"`
	QiniuUIDMapperMode   string `yaml:"qiniu_uid_mapper_mode"`
	QiniuUIDAutoCreate   string `yaml:"qiniu_uid_auto_create"`
	SkipIndexCreation    string `yaml:"skip_index_creation"`
}

type RateYAML struct {
	App     RateAppYAML `yaml:"app"`
	Account EnabledYAML `yaml:"account"`
	Token   EnabledYAML `yaml:"token"`
}

type RateAppYAML struct {
	Enabled   bool `yaml:"enabled"`
	PerMinute int  `yaml:"per_minute"`
	PerHour   int  `yaml:"per_hour"`
	PerDay    int  `yaml:"per_day"`
}

type EnabledYAML struct {
	Enabled bool `yaml:"enabled"`
}

// LoadFromYAML 从 YAML 文件加载配置作为环境变量的默认值。
// 如果环境变量已设置，则保留环境变量的值（环境变量优先级高于 YAML）。
// 如果环境变量未设置，则从 YAML 中取值设置到环境变量。
func LoadFromYAML(path string) {
	if path == "" {
		return
	}

	// 如果路径是目录，自动查找目录下的 config.yml
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		path = path + "/config.yml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "WARNING: failed to read config file %s: %v\n", path, err)
		}
		return
	}

	var cfg AppYAML
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: failed to parse config file %s: %v\n", path, err)
		return
	}

	// Server
	setDefaultEnv("PORT", cfg.Server.Port)
	setDefaultEnv("GIN_MODE", cfg.Server.GinMode)
	setDefaultEnv("LOG_LEVEL", cfg.Server.LogLevel)
	setDefaultEnv("LOG_FORMAT", cfg.Server.LogFormat)
	setDefaultEnv("QINIU_UID_MAPPER_MODE", cfg.Server.QiniuUIDMapperMode)
	setDefaultEnv("QINIU_UID_AUTO_CREATE", cfg.Server.QiniuUIDAutoCreate)
	setDefaultEnv("SKIP_INDEX_CREATION", cfg.Server.SkipIndexCreation)

	// MongoDB
	setDefaultEnv("MONGO_URI", cfg.Mongo.URI)
	setDefaultEnv("MONGO_DATABASE", cfg.Mongo.Database)

	// Redis
	if cfg.Redis.Enabled {
		setDefaultEnv("REDIS_ENABLED", "true")
	}
	setDefaultEnv("REDIS_ADDR", cfg.Redis.Addr)
	setDefaultEnv("REDIS_PASSWORD", cfg.Redis.Password)
	if cfg.Redis.DB != 0 {
		setDefaultEnv("REDIS_DB", strconv.Itoa(cfg.Redis.DB))
	}
	if cfg.Redis.PoolSize != 0 {
		setDefaultEnv("REDIS_POOL_SIZE", strconv.Itoa(cfg.Redis.PoolSize))
	}
	if cfg.Redis.MinIdleConns != 0 {
		setDefaultEnv("REDIS_MIN_IDLE_CONNS", strconv.Itoa(cfg.Redis.MinIdleConns))
	}
	if cfg.Redis.MaxRetries != 0 {
		setDefaultEnv("REDIS_MAX_RETRIES", strconv.Itoa(cfg.Redis.MaxRetries))
	}
	setDefaultEnv("CACHE_TOKEN_TTL", cfg.Redis.TokenCacheTTL)

	// Qconf
	if cfg.Qconf.Enabled {
		setDefaultEnv("QCONF_ENABLED", "true")
	}
	setDefaultEnv("QCONF_ACCESS_KEY", cfg.Qconf.AccessKey)
	setDefaultEnv("QCONF_SECRET_KEY", cfg.Qconf.SecretKey)
	setDefaultEnv("QCONF_MASTER_HOSTS", cfg.Qconf.MasterHosts.String())
	if cfg.Qconf.LcacheExpiresMS != 0 {
		setDefaultEnv("QCONF_LC_EXPIRES_MS", strconv.Itoa(cfg.Qconf.LcacheExpiresMS))
	}
	if cfg.Qconf.LcacheDurationMS != 0 {
		setDefaultEnv("QCONF_LC_DURATION_MS", strconv.Itoa(cfg.Qconf.LcacheDurationMS))
	}
	if cfg.Qconf.LcacheChanBufSize != 0 {
		setDefaultEnv("QCONF_LC_CHAN_BUFSIZE", strconv.Itoa(cfg.Qconf.LcacheChanBufSize))
	}
	if cfg.Qconf.McExpiresS != 0 {
		setDefaultEnv("QCONF_MC_EXPIRES_S", strconv.Itoa(cfg.Qconf.McExpiresS))
	}
	if cfg.Qconf.McRWTimeoutMS != 0 {
		setDefaultEnv("QCONF_MC_RW_TIMEOUT_MS", strconv.Itoa(cfg.Qconf.McRWTimeoutMS))
	}

	// Rate limit
	if cfg.Rate.App.Enabled {
		setDefaultEnv("ENABLE_APP_RATE_LIMIT", "true")
	}
	if cfg.Rate.App.PerMinute != 0 {
		setDefaultEnv("APP_RATE_LIMIT_PER_MINUTE", strconv.Itoa(cfg.Rate.App.PerMinute))
	}
	if cfg.Rate.App.PerHour != 0 {
		setDefaultEnv("APP_RATE_LIMIT_PER_HOUR", strconv.Itoa(cfg.Rate.App.PerHour))
	}
	if cfg.Rate.App.PerDay != 0 {
		setDefaultEnv("APP_RATE_LIMIT_PER_DAY", strconv.Itoa(cfg.Rate.App.PerDay))
	}
	if cfg.Rate.Account.Enabled {
		setDefaultEnv("ENABLE_ACCOUNT_RATE_LIMIT", "true")
	}
	if cfg.Rate.Token.Enabled {
		setDefaultEnv("ENABLE_TOKEN_RATE_LIMIT", "true")
	}
}

// setDefaultEnv 如果 key 的环境变量未设置，则设为 value
func setDefaultEnv(key, value string) {
	if value == "" {
		return
	}
	if _, exists := os.LookupEnv(key); !exists {
		os.Setenv(key, value)
	}
}
