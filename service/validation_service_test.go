package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/qiniu/bearer-token-service/v2/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ========================================
// Mock Repositories
// ========================================

// MockTokenRepository 模拟 TokenRepository
type MockTokenRepository struct {
	mock.Mock
}

func (m *MockTokenRepository) GetByTokenValue(ctx context.Context, tokenValue string) (*interfaces.Token, error) {
	args := m.Called(ctx, tokenValue)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.Token), args.Error(1)
}

func (m *MockTokenRepository) IncrementUsage(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockTokenRepository) Create(ctx context.Context, token *interfaces.Token) error {
	return nil
}

func (m *MockTokenRepository) GetByID(ctx context.Context, tokenID string) (*interfaces.Token, error) {
	return nil, nil
}

func (m *MockTokenRepository) ListByAccountID(ctx context.Context, accountID string, activeOnly bool, limit, offset int) ([]interfaces.Token, error) {
	return nil, nil
}

func (m *MockTokenRepository) CountByAccountID(ctx context.Context, accountID string, activeOnly bool) (int64, error) {
	return 0, nil
}

func (m *MockTokenRepository) UpdateStatus(ctx context.Context, tokenID string, isActive bool) error {
	return nil
}

func (m *MockTokenRepository) Delete(ctx context.Context, tokenID string) error {
	return nil
}

func (m *MockTokenRepository) UpdateLastUsed(ctx context.Context, tokenID string, lastUsedAt time.Time) error {
	return nil
}

func (m *MockTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

// MockUserInfoRepository 模拟 UserInfoRepository
type MockUserInfoRepository struct {
	mock.Mock
}

func (m *MockUserInfoRepository) GetUserInfoByUID(ctx context.Context, uid uint32) (*interfaces.UserInfo, error) {
	args := m.Called(ctx, uid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.UserInfo), args.Error(1)
}

// ========================================
// Test ValidateToken (基础验证)
// ========================================

func TestValidateToken_Success(t *testing.T) {
	mockTokenRepo := new(MockTokenRepository)
	service := NewValidationService(mockTokenRepo)

	expiresAt := time.Now().Add(24 * time.Hour)
	lastUsedAt := time.Now().Add(-1 * time.Hour)

	token := &interfaces.Token{
		ID:          "tk_123",
		AccountID:   "qiniu_1369077332",
		Token:       "sk-abc123",
		IUID:        "8901234",
		IsActive:    true,
		ExpiresAt:   &expiresAt,
		LastUsedAt:  &lastUsedAt,
	}

	mockTokenRepo.On("GetByTokenValue", mock.Anything, "sk-abc123").Return(token, nil)

	req := &interfaces.TokenValidateRequest{
		Token: "sk-abc123",
	}

	resp, err := service.ValidateToken(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, "Token is valid", resp.Message)
	assert.NotNil(t, resp.TokenInfo)
	assert.Equal(t, "tk_123", resp.TokenInfo.TokenID)
	assert.Equal(t, "1369077332", resp.TokenInfo.UID)
	assert.Equal(t, "8901234", resp.TokenInfo.IUID)
	assert.True(t, resp.TokenInfo.IsActive)

	mockTokenRepo.AssertExpectations(t)
}

func TestValidateToken_NotFound(t *testing.T) {
	mockTokenRepo := new(MockTokenRepository)
	service := NewValidationService(mockTokenRepo)

	mockTokenRepo.On("GetByTokenValue", mock.Anything, "sk-notfound").Return(nil, nil)

	req := &interfaces.TokenValidateRequest{
		Token: "sk-notfound",
	}

	resp, err := service.ValidateToken(context.Background(), req)

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, "Token not found", resp.Message)
	assert.Nil(t, resp.TokenInfo)

	mockTokenRepo.AssertExpectations(t)
}

func TestValidateToken_Inactive(t *testing.T) {
	mockTokenRepo := new(MockTokenRepository)
	service := NewValidationService(mockTokenRepo)

	token := &interfaces.Token{
		ID:        "tk_123",
		AccountID: "qiniu_1369077332",
		Token:     "sk-abc123",
		IsActive:  false, // Token 已停用
	}

	mockTokenRepo.On("GetByTokenValue", mock.Anything, "sk-abc123").Return(token, nil)

	req := &interfaces.TokenValidateRequest{
		Token: "sk-abc123",
	}

	resp, err := service.ValidateToken(context.Background(), req)

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, "Token is inactive", resp.Message)

	mockTokenRepo.AssertExpectations(t)
}

func TestValidateToken_Expired(t *testing.T) {
	mockTokenRepo := new(MockTokenRepository)
	service := NewValidationService(mockTokenRepo)

	expiresAt := time.Now().Add(-1 * time.Hour) // 已过期

	token := &interfaces.Token{
		ID:        "tk_123",
		AccountID: "qiniu_1369077332",
		Token:     "sk-abc123",
		IsActive:  true,
		ExpiresAt: &expiresAt,
	}

	mockTokenRepo.On("GetByTokenValue", mock.Anything, "sk-abc123").Return(token, nil)

	req := &interfaces.TokenValidateRequest{
		Token: "sk-abc123",
	}

	resp, err := service.ValidateToken(context.Background(), req)

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, "Token has expired", resp.Message)

	mockTokenRepo.AssertExpectations(t)
}

func TestValidateToken_DatabaseError(t *testing.T) {
	mockTokenRepo := new(MockTokenRepository)
	service := NewValidationService(mockTokenRepo)

	mockTokenRepo.On("GetByTokenValue", mock.Anything, "sk-abc123").Return(nil, errors.New("database connection failed"))

	req := &interfaces.TokenValidateRequest{
		Token: "sk-abc123",
	}

	resp, err := service.ValidateToken(context.Background(), req)

	require.Error(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, "internal error", resp.Message)

	mockTokenRepo.AssertExpectations(t)
}

// ========================================
// Test ValidateTokenWithUserInfo (扩展验证)
// ========================================

func TestValidateTokenWithUserInfo_Success(t *testing.T) {
	mockTokenRepo := new(MockTokenRepository)
	mockUserInfoRepo := new(MockUserInfoRepository)
	service := NewValidationServiceWithUserInfo(mockTokenRepo, mockUserInfoRepo)

	expiresAt := time.Now().Add(24 * time.Hour)

	token := &interfaces.Token{
		ID:        "tk_123",
		AccountID: "qiniu_1369077332",
		Token:     "sk-abc123",
		IUID:      "8901234",
		IsActive:  true,
		ExpiresAt: &expiresAt,
	}

	userInfo := &interfaces.UserInfo{
		UID:       1369077332,
		Email:     "test@example.com",
		Username:  "testuser",
		Utype:     interfaces.UserTypeEnterprise | interfaces.UserTypeBuffered,
		Activated: true,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	mockTokenRepo.On("GetByTokenValue", mock.Anything, "sk-abc123").Return(token, nil)
	mockUserInfoRepo.On("GetUserInfoByUID", mock.Anything, uint32(1369077332)).Return(userInfo, nil)

	req := &interfaces.TokenValidateRequest{
		Token: "sk-abc123",
	}

	resp, err := service.ValidateTokenWithUserInfo(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, "Token is valid", resp.Message)
	assert.NotNil(t, resp.TokenInfo)
	assert.Equal(t, "tk_123", resp.TokenInfo.TokenID)
	assert.Equal(t, "1369077332", resp.TokenInfo.UID)
	assert.Equal(t, "8901234", resp.TokenInfo.IUID)

	// 验证扩展用户信息
	assert.NotNil(t, resp.TokenInfo.UserInfo)
	assert.Equal(t, uint32(1369077332), resp.TokenInfo.UserInfo.UID)
	assert.Equal(t, "test@example.com", resp.TokenInfo.UserInfo.Email)
	assert.Equal(t, "testuser", resp.TokenInfo.UserInfo.Username)
	assert.True(t, resp.TokenInfo.UserInfo.Activated)

	// 验证 Utype 位掩码检查
	assert.True(t, resp.TokenInfo.UserInfo.IsEnterprise())
	assert.True(t, resp.TokenInfo.UserInfo.IsBuffered())
	assert.False(t, resp.TokenInfo.UserInfo.IsDisabled())

	mockTokenRepo.AssertExpectations(t)
	mockUserInfoRepo.AssertExpectations(t)
}

func TestValidateTokenWithUserInfo_MySQLFailure_GracefulDegradation(t *testing.T) {
	mockTokenRepo := new(MockTokenRepository)
	mockUserInfoRepo := new(MockUserInfoRepository)
	service := NewValidationServiceWithUserInfo(mockTokenRepo, mockUserInfoRepo)

	expiresAt := time.Now().Add(24 * time.Hour)

	token := &interfaces.Token{
		ID:        "tk_123",
		AccountID: "qiniu_1369077332",
		Token:     "sk-abc123",
		IsActive:  true,
		ExpiresAt: &expiresAt,
	}

	mockTokenRepo.On("GetByTokenValue", mock.Anything, "sk-abc123").Return(token, nil)
	// MySQL 查询失败
	mockUserInfoRepo.On("GetUserInfoByUID", mock.Anything, uint32(1369077332)).Return(nil, errors.New("MySQL connection timeout"))

	req := &interfaces.TokenValidateRequest{
		Token: "sk-abc123",
	}

	resp, err := service.ValidateTokenWithUserInfo(context.Background(), req)

	// 关键：即使 MySQL 失败，验证仍然成功（优雅降级）
	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, "Token is valid", resp.Message)
	assert.NotNil(t, resp.TokenInfo)
	assert.Equal(t, "tk_123", resp.TokenInfo.TokenID)
	assert.Equal(t, "1369077332", resp.TokenInfo.UID)

	// UserInfo 为 nil（因为 MySQL 查询失败）
	assert.Nil(t, resp.TokenInfo.UserInfo)

	mockTokenRepo.AssertExpectations(t)
	mockUserInfoRepo.AssertExpectations(t)
}

func TestValidateTokenWithUserInfo_TokenInvalid(t *testing.T) {
	mockTokenRepo := new(MockTokenRepository)
	mockUserInfoRepo := new(MockUserInfoRepository)
	service := NewValidationServiceWithUserInfo(mockTokenRepo, mockUserInfoRepo)

	// Token 不存在
	mockTokenRepo.On("GetByTokenValue", mock.Anything, "sk-invalid").Return(nil, nil)

	req := &interfaces.TokenValidateRequest{
		Token: "sk-invalid",
	}

	resp, err := service.ValidateTokenWithUserInfo(context.Background(), req)

	require.NoError(t, err)
	assert.False(t, resp.Valid)
	assert.Equal(t, "Token not found", resp.Message)

	// 不应该尝试查询用户信息（因为 token 已经无效）
	mockUserInfoRepo.AssertNotCalled(t, "GetUserInfoByUID")

	mockTokenRepo.AssertExpectations(t)
}

func TestValidateTokenWithUserInfo_HMACUser_NoUserInfoQuery(t *testing.T) {
	mockTokenRepo := new(MockTokenRepository)
	mockUserInfoRepo := new(MockUserInfoRepository)
	service := NewValidationServiceWithUserInfo(mockTokenRepo, mockUserInfoRepo)

	expiresAt := time.Now().Add(24 * time.Hour)

	// HMAC 用户（非 QiniuStub）
	token := &interfaces.Token{
		ID:        "tk_123",
		AccountID: "acc_hmac_user", // 不是 qiniu_ 前缀
		Token:     "sk-abc123",
		IsActive:  true,
		ExpiresAt: &expiresAt,
	}

	mockTokenRepo.On("GetByTokenValue", mock.Anything, "sk-abc123").Return(token, nil)

	req := &interfaces.TokenValidateRequest{
		Token: "sk-abc123",
	}

	resp, err := service.ValidateTokenWithUserInfo(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.NotNil(t, resp.TokenInfo)
	assert.Equal(t, "acc_hmac_user", resp.TokenInfo.AccountID)
	assert.Empty(t, resp.TokenInfo.UID) // 没有 UID

	// 不应该查询用户信息（因为不是 QiniuStub 用户）
	mockUserInfoRepo.AssertNotCalled(t, "GetUserInfoByUID")

	// UserInfo 为 nil（因为不是 QiniuStub 用户）
	assert.Nil(t, resp.TokenInfo.UserInfo)

	mockTokenRepo.AssertExpectations(t)
}

func TestValidateTokenWithUserInfo_NoUserInfoRepo(t *testing.T) {
	mockTokenRepo := new(MockTokenRepository)
	// 没有提供 UserInfoRepository
	service := NewValidationService(mockTokenRepo)

	expiresAt := time.Now().Add(24 * time.Hour)

	token := &interfaces.Token{
		ID:        "tk_123",
		AccountID: "qiniu_1369077332",
		Token:     "sk-abc123",
		IsActive:  true,
		ExpiresAt: &expiresAt,
	}

	mockTokenRepo.On("GetByTokenValue", mock.Anything, "sk-abc123").Return(token, nil)

	req := &interfaces.TokenValidateRequest{
		Token: "sk-abc123",
	}

	resp, err := service.ValidateTokenWithUserInfo(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.NotNil(t, resp.TokenInfo)
	assert.Equal(t, "1369077332", resp.TokenInfo.UID)

	// UserInfo 为 nil（因为没有 UserInfoRepository）
	assert.Nil(t, resp.TokenInfo.UserInfo)

	mockTokenRepo.AssertExpectations(t)
}

// ========================================
// Test RecordTokenUsage
// ========================================

func TestRecordTokenUsage_Success(t *testing.T) {
	mockTokenRepo := new(MockTokenRepository)
	service := NewValidationService(mockTokenRepo)

	token := &interfaces.Token{
		ID:    "tk_123",
		Token: "sk-abc123",
	}

	mockTokenRepo.On("GetByTokenValue", mock.Anything, "sk-abc123").Return(token, nil)
	mockTokenRepo.On("IncrementUsage", mock.Anything, "tk_123").Return(nil)

	err := service.RecordTokenUsage(context.Background(), "sk-abc123")

	// Wait for goroutine to complete
	time.Sleep(10 * time.Millisecond)

	require.NoError(t, err)
	mockTokenRepo.AssertExpectations(t)
}

func TestRecordTokenUsage_TokenNotFound(t *testing.T) {
	mockTokenRepo := new(MockTokenRepository)
	service := NewValidationService(mockTokenRepo)

	mockTokenRepo.On("GetByTokenValue", mock.Anything, "sk-notfound").Return(nil, nil)

	err := service.RecordTokenUsage(context.Background(), "sk-notfound")

	require.Error(t, err)
	assert.Equal(t, "token not found", err.Error())
	mockTokenRepo.AssertExpectations(t)
}

// ========================================
// Test extractUIDFromAccountID
// ========================================

func TestExtractUIDFromAccountID(t *testing.T) {
	tests := []struct {
		name      string
		accountID string
		wantUID   string
		wantOK    bool
	}{
		{
			name:      "Valid QiniuStub UID",
			accountID: "qiniu_1369077332",
			wantUID:   "1369077332",
			wantOK:    true,
		},
		{
			name:      "HMAC User",
			accountID: "acc_123456",
			wantUID:   "",
			wantOK:    false,
		},
		{
			name:      "Invalid UID (non-numeric)",
			accountID: "qiniu_abc",
			wantUID:   "",
			wantOK:    false,
		},
		{
			name:      "Empty AccountID",
			accountID: "",
			wantUID:   "",
			wantOK:    false,
		},
		{
			name:      "Just prefix",
			accountID: "qiniu_",
			wantUID:   "",
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uid, ok := extractUIDFromAccountID(tt.accountID)
			assert.Equal(t, tt.wantUID, uid)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
