package config

import (
	"os"
	"testing"
)

func TestLoadFromYAML(t *testing.T) {
	envKeys := []string{
		"PORT", "LOG_LEVEL", "SKIP_INDEX_CREATION",
		"MONGO_URI", "MONGO_DATABASE",
		"REDIS_ENABLED", "REDIS_ADDR", "REDIS_PASSWORD",
		"QCONF_ENABLED", "QCONF_ACCESS_KEY", "QCONF_SECRET_KEY", "QCONF_MASTER_HOSTS",
		"ENABLE_APP_RATE_LIMIT", "APP_RATE_LIMIT_PER_MINUTE",
		"ENABLE_ACCOUNT_RATE_LIMIT", "ENABLE_TOKEN_RATE_LIMIT",
	}
	for _, k := range envKeys {
		os.Unsetenv(k)
		t.Cleanup(func(k string) func() { return func() { os.Unsetenv(k) } }(k))
	}

	tmp := t.TempDir()
	path := tmp + "/config.yml"

	data := `
server:
  port: "9090"
  log_level: "debug"
  skip_index_creation: "true"
mongo:
  uri: "mongodb://test:1234@host/db"
  database: "test_db"
redis:
  enabled: true
  addr: "redis:6379"
  password: "secret"
qconf:
  enabled: true
  access_key: "ak"
  secret_key: "sk"
  master_hosts:
    - "http://h1:8510"
    - "http://h2:8510"
rate_limit:
  app:
    enabled: true
    per_minute: 500
  account:
    enabled: true
  token:
    enabled: true
`
	os.WriteFile(path, []byte(data), 0644)

	LoadFromYAML(path)

	tests := []struct{ key, want string }{
		{"PORT", "9090"},
		{"LOG_LEVEL", "debug"},
		{"SKIP_INDEX_CREATION", "true"},
		{"MONGO_URI", "mongodb://test:1234@host/db"},
		{"MONGO_DATABASE", "test_db"},
		{"REDIS_ENABLED", "true"},
		{"REDIS_ADDR", "redis:6379"},
		{"REDIS_PASSWORD", "secret"},
		{"QCONF_ENABLED", "true"},
		{"QCONF_ACCESS_KEY", "ak"},
		{"QCONF_SECRET_KEY", "sk"},
		{"QCONF_MASTER_HOSTS", "http://h1:8510,http://h2:8510"},
		{"ENABLE_APP_RATE_LIMIT", "true"},
		{"APP_RATE_LIMIT_PER_MINUTE", "500"},
		{"ENABLE_ACCOUNT_RATE_LIMIT", "true"},
		{"ENABLE_TOKEN_RATE_LIMIT", "true"},
	}
	for _, tt := range tests {
		if got := os.Getenv(tt.key); got != tt.want {
			t.Errorf("%s = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestEnvOverridesYAML(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/config.yml"
	os.WriteFile(path, []byte(`server: {port: "9090"}`), 0644)

	os.Setenv("PORT", "8080")
	defer os.Unsetenv("PORT")

	LoadFromYAML(path)

	if got := os.Getenv("PORT"); got != "8080" {
		t.Errorf("PORT = %q, want %q (env should override YAML)", got, "8080")
	}
}

func TestMissingFileNoError(t *testing.T) {
	LoadFromYAML("/nonexistent/config.yml")
	// 不应 panic 或报错
}
