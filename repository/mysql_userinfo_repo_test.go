package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/qiniu/bearer-token-service/v2/config"
	"github.com/qiniu/bearer-token-service/v2/interfaces"
	"github.com/qiniu/bearer-token-service/v2/pkg/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createMockClient 创建带 mock 的 MySQL 客户端
func createMockClient(t *testing.T) (*mysql.Client, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)

	cfg := &config.MySQLConfig{
		Host:            "localhost",
		Port:            3306,
		User:            "test",
		Password:        "test",
		Database:        "test",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		Timeout:         3 * time.Second,
	}

	// 使用 NewTestClient 注入 mock db
	client := mysql.NewTestClient(db, cfg)
	return client, mock
}

// TestGetUserInfoByUID_Success 测试成功查询用户信息
func TestGetUserInfoByUID_Success(t *testing.T) {
	client, mock := createMockClient(t)
	defer client.Close()

	repo := NewMySQLUserInfoRepository(client)

	// 准备测试数据
	now := time.Now().Truncate(time.Second)
	disabledAt := now.Add(-24 * time.Hour)
	lastLoginAt := now.Add(-1 * time.Hour)

	expectedUID := uint32(1369077332)
	expectedEmail := "test@qiniu.com"
	expectedUsername := "testuser"
	expectedUtype := uint32(interfaces.UserTypeEnterprise | interfaces.UserTypeBuffered)

	// 准备查询期望
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

	rows := sqlmock.NewRows([]string{
		"uid", "email", "username", "utype", "activated",
		"disabled_type", "disabled_reason", "disabled_at", "parent_uid",
		"created_at", "updated_at", "last_login_at",
	}).AddRow(
		expectedUID,
		expectedEmail,
		expectedUsername,
		expectedUtype,
		true,                        // activated
		0,                           // disabled_type
		"",                          // disabled_reason
		disabledAt,                  // disabled_at
		uint32(0),                   // parent_uid
		now.Unix(),                  // created_at
		now.Unix(),                  // updated_at
		lastLoginAt.Unix(),          // last_login_at
	)

	mock.ExpectQuery(query).WithArgs(expectedUID).WillReturnRows(rows)

	// 执行查询
	ctx := context.Background()
	userInfo, err := repo.GetUserInfoByUID(ctx, expectedUID)

	// 验证结果
	require.NoError(t, err)
	assert.NotNil(t, userInfo)
	assert.Equal(t, expectedUID, userInfo.UID)
	assert.Equal(t, expectedEmail, userInfo.Email)
	assert.Equal(t, expectedUsername, userInfo.Username)
	assert.Equal(t, expectedUtype, userInfo.Utype)
	assert.True(t, userInfo.Activated)
	assert.NotNil(t, userInfo.DisabledAt)
	assert.Equal(t, now.Unix(), userInfo.CreatedAt)
	assert.Equal(t, lastLoginAt.Unix(), userInfo.LastLoginAt)

	// 验证 Utype 辅助方法
	assert.False(t, userInfo.IsDisabled())
	assert.True(t, userInfo.IsBuffered())
	assert.True(t, userInfo.IsEnterprise())

	// 验证 mock 期望
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestGetUserInfoByUID_NotFound 测试用户不存在
func TestGetUserInfoByUID_NotFound(t *testing.T) {
	client, mock := createMockClient(t)
	defer client.Close()

	repo := NewMySQLUserInfoRepository(client)

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

	mock.ExpectQuery(query).WithArgs(uint32(999999)).WillReturnError(sql.ErrNoRows)

	// 执行查询
	ctx := context.Background()
	userInfo, err := repo.GetUserInfoByUID(ctx, 999999)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "user not found")

	// 验证 mock 期望
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestGetUserInfoByUID_DatabaseError 测试数据库错误
func TestGetUserInfoByUID_DatabaseError(t *testing.T) {
	client, mock := createMockClient(t)
	defer client.Close()

	repo := NewMySQLUserInfoRepository(client)

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

	mock.ExpectQuery(query).WithArgs(uint32(123456)).WillReturnError(sql.ErrConnDone)

	// 执行查询
	ctx := context.Background()
	userInfo, err := repo.GetUserInfoByUID(ctx, 123456)

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "failed to query user info")

	// 验证 mock 期望
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// TestUserInfo_UtypeHelpers 测试 Utype 位掩码辅助方法
func TestUserInfo_UtypeHelpers(t *testing.T) {
	tests := []struct {
		name     string
		utype    uint32
		disabled bool
		buffered bool
		overseas bool
		stdUser  bool
	}{
		{
			name:     "Normal enterprise user",
			utype:    interfaces.UserTypeEnterprise,
			disabled: false,
			buffered: false,
			overseas: false,
			stdUser:  true,
		},
		{
			name:     "Disabled user",
			utype:    interfaces.UserTypeDisabled,
			disabled: true,
			buffered: false,
			overseas: false,
			stdUser:  false,
		},
		{
			name:     "Buffered enterprise user",
			utype:    interfaces.UserTypeEnterprise | interfaces.UserTypeBuffered,
			disabled: false,
			buffered: true,
			overseas: false,
			stdUser:  true,
		},
		{
			name:     "Overseas standard user",
			utype:    interfaces.UserTypeOverseasStd,
			disabled: false,
			buffered: false,
			overseas: false,
			stdUser:  false,
		},
		{
			name:     "Overseas user",
			utype:    interfaces.UserTypeOverseas,
			disabled: false,
			buffered: false,
			overseas: true,
			stdUser:  false,
		},
		{
			name:     "Multiple flags",
			utype:    interfaces.UserTypeEnterprise | interfaces.UserTypeBuffered | interfaces.UserTypeOverseas,
			disabled: false,
			buffered: true,
			overseas: true,
			stdUser:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userInfo := &interfaces.UserInfo{
				UID:       123456,
				Email:     "test@example.com",
				Username:  "testuser",
				Utype:     tt.utype,
				Activated: true,
			}

			assert.Equal(t, tt.disabled, userInfo.IsDisabled(), "IsDisabled mismatch")
			assert.Equal(t, tt.buffered, userInfo.IsBuffered(), "IsBuffered mismatch")
			assert.Equal(t, tt.overseas, userInfo.IsOverseas(), "IsOverseas mismatch")
			assert.Equal(t, tt.stdUser, userInfo.IsEnterprise(), "IsEnterprise mismatch")
		})
	}
}

// TestGetUserInfoByUID_NullFields 测试可空字段处理
func TestGetUserInfoByUID_NullFields(t *testing.T) {
	client, mock := createMockClient(t)
	defer client.Close()

	repo := NewMySQLUserInfoRepository(client)

	now := time.Now().Truncate(time.Second)

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

	// 测试 disabled_at 和 last_login_at 为 NULL 的情况
	rows := sqlmock.NewRows([]string{
		"uid", "email", "username", "utype", "activated",
		"disabled_type", "disabled_reason", "disabled_at", "parent_uid",
		"created_at", "updated_at", "last_login_at",
	}).AddRow(
		uint32(123456),
		"test@qiniu.com",
		"testuser",
		uint32(interfaces.UserTypeEnterprise),
		true,
		0,
		"",
		nil,        // disabled_at is NULL
		uint32(0),
		now.Unix(),
		now.Unix(),
		nil,        // last_login_at is NULL
	)

	mock.ExpectQuery(query).WithArgs(uint32(123456)).WillReturnRows(rows)

	// 执行查询
	ctx := context.Background()
	userInfo, err := repo.GetUserInfoByUID(ctx, 123456)

	// 验证结果
	require.NoError(t, err)
	assert.NotNil(t, userInfo)
	assert.Equal(t, uint32(123456), userInfo.UID)

	// 验证 NULL 字段被正确处理
	assert.Nil(t, userInfo.DisabledAt, "disabled_at should be nil")
	assert.Equal(t, int64(0), userInfo.LastLoginAt, "last_login_at should be 0")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
