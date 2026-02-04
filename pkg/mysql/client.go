package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/qiniu/bearer-token-service/v2/config"
)

// Client MySQL 客户端封装
type Client struct {
	db     *sql.DB
	config *config.MySQLConfig
}

// NewClient 创建 MySQL 客户端
func NewClient(cfg *config.MySQLConfig) (*Client, error) {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping mysql: %w", err)
	}

	return &Client{
		db:     db,
		config: cfg,
	}, nil
}

// DB 返回底层的 *sql.DB 实例
func (c *Client) DB() *sql.DB {
	return c.db
}

// Close 关闭数据库连接
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	if err := c.db.PingContext(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}

// Stats 返回连接池统计信息
func (c *Client) Stats() sql.DBStats {
	return c.db.Stats()
}

// QueryRow 执行单行查询（便捷方法）
func (c *Client) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return c.db.QueryRowContext(ctx, query, args...)
}

// Query 执行多行查询（便捷方法）
func (c *Client) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...)
}

// Exec 执行写入操作（便捷方法）
func (c *Client) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return c.db.ExecContext(ctx, query, args...)
}

// Transaction 执行事务
func (c *Client) Transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WithTimeout 创建带超时的上下文
func (c *Client) WithTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, c.config.Timeout)
}

// IsConnected 检查是否已连接
func (c *Client) IsConnected() bool {
	if c.db == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	return c.db.PingContext(ctx) == nil
}

// NewTestClient 创建用于测试的 MySQL 客户端（注入 mock db）
// 这个函数只应该在测试代码中使用
func NewTestClient(db *sql.DB, cfg *config.MySQLConfig) *Client {
	return &Client{
		db:     db,
		config: cfg,
	}
}
