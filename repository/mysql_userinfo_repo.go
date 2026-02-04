package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/qiniu/bearer-token-service/v2/interfaces"
	"github.com/qiniu/bearer-token-service/v2/pkg/mysql"
)

// MySQLUserInfoRepository MySQL 实现的用户信息存储库
type MySQLUserInfoRepository struct {
	client *mysql.Client
}

// NewMySQLUserInfoRepository 创建用户信息存储库实例
func NewMySQLUserInfoRepository(client *mysql.Client) *MySQLUserInfoRepository {
	return &MySQLUserInfoRepository{
		client: client,
	}
}

// GetUserInfoByUID 根据 UID 查询用户信息
// 查询失败时返回 error，调用方决定如何处理（优雅降级）
func (r *MySQLUserInfoRepository) GetUserInfoByUID(ctx context.Context, uid uint32) (*interfaces.UserInfo, error) {
	// 创建带超时的上下文
	queryCtx, cancel := r.client.WithTimeout(ctx)
	defer cancel()

	// SQL 查询语句
	// 注意：这里假设表名为 auth，实际部署时可能需要调整
	// 字段映射基于 qconfapi.AccountInfo 结构和分析文档
	query := `
		SELECT
			uid,
			email,
			username,
			utype,
			activated,
			disabled_type,
			disabled_reason,
			disabled_at,
			parent_uid,
			UNIX_TIMESTAMP(created_at) as created_at,
			UNIX_TIMESTAMP(updated_at) as updated_at,
			UNIX_TIMESTAMP(last_login_at) as last_login_at
		FROM auth
		WHERE uid = ?
		LIMIT 1
	`

	var userInfo interfaces.UserInfo
	var disabledAt sql.NullTime
	var lastLoginAt sql.NullInt64

	// 执行查询
	row := r.client.QueryRow(queryCtx, query, uid)
	err := row.Scan(
		&userInfo.UID,
		&userInfo.Email,
		&userInfo.Username,
		&userInfo.Utype,
		&userInfo.Activated,
		&userInfo.DisabledType,
		&userInfo.DisabledReason,
		&disabledAt,
		&userInfo.ParentUID,
		&userInfo.CreatedAt,
		&userInfo.UpdatedAt,
		&lastLoginAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: uid=%d", uid)
		}
		return nil, fmt.Errorf("failed to query user info: %w", err)
	}

	// 处理可空字段
	if disabledAt.Valid {
		t := disabledAt.Time
		userInfo.DisabledAt = &t
	}

	if lastLoginAt.Valid {
		userInfo.LastLoginAt = lastLoginAt.Int64
	}

	return &userInfo, nil
}

// HealthCheck 检查 MySQL 连接健康状态
func (r *MySQLUserInfoRepository) HealthCheck(ctx context.Context) error {
	return r.client.HealthCheck(ctx)
}

// Close 关闭数据库连接（如果需要）
func (r *MySQLUserInfoRepository) Close() error {
	return r.client.Close()
}
