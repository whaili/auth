package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// MySQLConfig MySQL 数据库配置
type MySQLConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	Timeout         time.Duration
}

// LoadMySQLConfig 从环境变量加载 MySQL 配置
func LoadMySQLConfig() *MySQLConfig {
	config := &MySQLConfig{
		Host:            getEnv("MYSQL_HOST", "10.70.67.40"),
		Port:            getEnvAsInt("MYSQL_PORT", 3306),
		User:            getEnv("MYSQL_USER", "chatnio"),
		Password:        getEnv("MYSQL_PASSWORD", ""), // 密码必须通过环境变量提供
		Database:        getEnv("MYSQL_DATABASE", "chatnio"),
		MaxOpenConns:    getEnvAsInt("MYSQL_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvAsInt("MYSQL_MAX_IDLE_CONNS", 10),
		ConnMaxLifetime: getEnvAsDuration("MYSQL_CONN_MAX_LIFETIME", 5*time.Minute),
		Timeout:         getEnvAsDuration("MYSQL_TIMEOUT", 3*time.Second),
	}
	return config
}

// DSN 生成 MySQL 数据源名称
func (c *MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.Timeout,
	)
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt 获取整数型环境变量
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// getEnvAsDuration 获取时间间隔型环境变量
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
