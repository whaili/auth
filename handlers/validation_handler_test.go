package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qiniu/bearer-token-service/v2/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ========================================
// Mock ValidationService
// ========================================

type MockValidationService struct {
	mock.Mock
}

func (m *MockValidationService) ValidateToken(ctx context.Context, req *interfaces.TokenValidateRequest) (*interfaces.TokenValidateResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.TokenValidateResponse), args.Error(1)
}

func (m *MockValidationService) ValidateTokenWithUserInfo(ctx context.Context, req *interfaces.TokenValidateRequest) (*interfaces.TokenValidateUResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*interfaces.TokenValidateUResponse), args.Error(1)
}

func (m *MockValidationService) RecordTokenUsage(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

// ========================================
// TestValidateToken - Basic validation endpoint
// ========================================

func TestValidateToken_Success(t *testing.T) {
	// Arrange
	mockService := new(MockValidationService)
	handler := NewValidationHandler(mockService)

	expiresAt := time.Now().Add(24 * time.Hour)
	lastUsedAt := time.Now().Add(-1 * time.Hour)

	mockService.On("ValidateToken", mock.Anything, &interfaces.TokenValidateRequest{
		Token: "sk-valid-token",
	}).Return(&interfaces.TokenValidateResponse{
		Valid:   true,
		Message: "valid",
		TokenInfo: &interfaces.TokenInfo{
			TokenID:    "tk_123",
			AccountID:  "qiniu_1369077332",
			UID:        "1369077332",
			IUID:       "",
			IsActive:   true,
			ExpiresAt:  &expiresAt,
			LastUsedAt: &lastUsedAt,
		},
	}, nil)

	mockService.On("RecordTokenUsage", mock.Anything, "sk-valid-token").Return(nil)

	req := httptest.NewRequest("POST", "/api/v2/validate", nil)
	req.Header.Set("Authorization", "Bearer sk-valid-token")
	w := httptest.NewRecorder()

	// Act
	handler.ValidateToken(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp interfaces.TokenValidateResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Valid)
	assert.Equal(t, "valid", resp.Message)
	assert.NotNil(t, resp.TokenInfo)
	assert.Equal(t, "tk_123", resp.TokenInfo.TokenID)
	assert.Equal(t, "1369077332", resp.TokenInfo.UID)

	mockService.AssertExpectations(t)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	// Arrange
	mockService := new(MockValidationService)
	handler := NewValidationHandler(mockService)

	mockService.On("ValidateToken", mock.Anything, &interfaces.TokenValidateRequest{
		Token: "sk-invalid-token",
	}).Return(&interfaces.TokenValidateResponse{
		Valid:   false,
		Message: "token not found",
	}, nil)

	req := httptest.NewRequest("POST", "/api/v2/validate", nil)
	req.Header.Set("Authorization", "Bearer sk-invalid-token")
	w := httptest.NewRecorder()

	// Act
	handler.ValidateToken(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp interfaces.TokenValidateResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.False(t, resp.Valid)
	assert.Equal(t, "token not found", resp.Message)

	// RecordTokenUsage should NOT be called for invalid tokens
	mockService.AssertNotCalled(t, "RecordTokenUsage", mock.Anything, mock.Anything)
	mockService.AssertExpectations(t)
}

func TestValidateToken_MissingAuthHeader(t *testing.T) {
	// Arrange
	mockService := new(MockValidationService)
	handler := NewValidationHandler(mockService)

	req := httptest.NewRequest("POST", "/api/v2/validate", nil)
	w := httptest.NewRecorder()

	// Act
	handler.ValidateToken(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "invalid authorization header", resp["error"])
	assert.Equal(t, float64(http.StatusUnauthorized), resp["code"])

	mockService.AssertNotCalled(t, "ValidateToken", mock.Anything, mock.Anything)
}

func TestValidateToken_InvalidAuthHeaderFormat(t *testing.T) {
	// Arrange
	mockService := new(MockValidationService)
	handler := NewValidationHandler(mockService)

	req := httptest.NewRequest("POST", "/api/v2/validate", nil)
	req.Header.Set("Authorization", "Basic some-base64")
	w := httptest.NewRecorder()

	// Act
	handler.ValidateToken(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "invalid authorization header", resp["error"])
	mockService.AssertNotCalled(t, "ValidateToken", mock.Anything, mock.Anything)
}

// ========================================
// TestValidateTokenU - Extended validation with user info
// ========================================

func TestValidateTokenU_Success(t *testing.T) {
	// Arrange
	mockService := new(MockValidationService)
	handler := NewValidationHandler(mockService)

	expiresAt := time.Now().Add(24 * time.Hour)
	lastUsedAt := time.Now().Add(-1 * time.Hour)
	createdAt := int64(1700000000)
	updatedAt := int64(1700000100)
	lastLoginAt := int64(1700000200)

	mockService.On("ValidateTokenWithUserInfo", mock.Anything, &interfaces.TokenValidateRequest{
		Token: "sk-valid-token",
	}).Return(&interfaces.TokenValidateUResponse{
		Valid:   true,
		Message: "valid",
		TokenInfo: &interfaces.TokenInfoU{
			TokenID:    "tk_123",
			AccountID:  "qiniu_1369077332",
			UID:        "1369077332",
			IUID:       "",
			IsActive:   true,
			ExpiresAt:  &expiresAt,
			LastUsedAt: &lastUsedAt,
			UserInfo: &interfaces.UserInfo{
				UID:            1369077332,
				Email:          "user@example.com",
				Username:       "testuser",
				Utype:          1,
				Activated:      true,
				DisabledType:   0,
				DisabledReason: "",
				DisabledAt:     nil,
				ParentUID:      0,
				CreatedAt:      createdAt,
				UpdatedAt:      updatedAt,
				LastLoginAt:    lastLoginAt,
			},
		},
	}, nil)

	mockService.On("RecordTokenUsage", mock.Anything, "sk-valid-token").Return(nil)

	req := httptest.NewRequest("POST", "/api/v2/validateu", nil)
	req.Header.Set("Authorization", "Bearer sk-valid-token")
	w := httptest.NewRecorder()

	// Act
	handler.ValidateTokenU(w, req)

	// Wait for goroutine to complete
	time.Sleep(10 * time.Millisecond)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp interfaces.TokenValidateUResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Valid)
	assert.Equal(t, "valid", resp.Message)
	assert.NotNil(t, resp.TokenInfo)
	assert.Equal(t, "tk_123", resp.TokenInfo.TokenID)
	assert.Equal(t, "1369077332", resp.TokenInfo.UID)

	// Verify user info
	assert.NotNil(t, resp.TokenInfo.UserInfo)
	assert.Equal(t, uint32(1369077332), resp.TokenInfo.UserInfo.UID)
	assert.Equal(t, "user@example.com", resp.TokenInfo.UserInfo.Email)
	assert.Equal(t, "testuser", resp.TokenInfo.UserInfo.Username)
	assert.Equal(t, uint32(1), resp.TokenInfo.UserInfo.Utype)

	mockService.AssertExpectations(t)
}

func TestValidateTokenU_MySQLFailure_GracefulDegradation(t *testing.T) {
	// Arrange
	mockService := new(MockValidationService)
	handler := NewValidationHandler(mockService)

	expiresAt := time.Now().Add(24 * time.Hour)
	lastUsedAt := time.Now().Add(-1 * time.Hour)

	// Service returns valid token but user_info is nil due to MySQL failure
	mockService.On("ValidateTokenWithUserInfo", mock.Anything, &interfaces.TokenValidateRequest{
		Token: "sk-valid-token",
	}).Return(&interfaces.TokenValidateUResponse{
		Valid:   true,
		Message: "valid",
		TokenInfo: &interfaces.TokenInfoU{
			TokenID:    "tk_123",
			AccountID:  "qiniu_1369077332",
			UID:        "1369077332",
			IUID:       "",
			IsActive:   true,
			ExpiresAt:  &expiresAt,
			LastUsedAt: &lastUsedAt,
			UserInfo:   nil, // MySQL query failed
		},
	}, nil)

	mockService.On("RecordTokenUsage", mock.Anything, "sk-valid-token").Return(nil)

	req := httptest.NewRequest("POST", "/api/v2/validateu", nil)
	req.Header.Set("Authorization", "Bearer sk-valid-token")
	w := httptest.NewRecorder()

	// Act
	handler.ValidateTokenU(w, req)

	// Wait for goroutine to complete
	time.Sleep(10 * time.Millisecond)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp interfaces.TokenValidateUResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Token validation succeeded
	assert.True(t, resp.Valid)
	assert.Equal(t, "valid", resp.Message)
	assert.NotNil(t, resp.TokenInfo)

	// But user_info is nil (graceful degradation)
	assert.Nil(t, resp.TokenInfo.UserInfo)

	mockService.AssertExpectations(t)
}

func TestValidateTokenU_InvalidToken(t *testing.T) {
	// Arrange
	mockService := new(MockValidationService)
	handler := NewValidationHandler(mockService)

	mockService.On("ValidateTokenWithUserInfo", mock.Anything, &interfaces.TokenValidateRequest{
		Token: "sk-invalid-token",
	}).Return(&interfaces.TokenValidateUResponse{
		Valid:   false,
		Message: "token not found",
	}, nil)

	req := httptest.NewRequest("POST", "/api/v2/validateu", nil)
	req.Header.Set("Authorization", "Bearer sk-invalid-token")
	w := httptest.NewRecorder()

	// Act
	handler.ValidateTokenU(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp interfaces.TokenValidateUResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.False(t, resp.Valid)
	assert.Equal(t, "token not found", resp.Message)

	// RecordTokenUsage should NOT be called for invalid tokens
	mockService.AssertNotCalled(t, "RecordTokenUsage", mock.Anything, mock.Anything)
	mockService.AssertExpectations(t)
}

func TestValidateTokenU_MissingAuthHeader(t *testing.T) {
	// Arrange
	mockService := new(MockValidationService)
	handler := NewValidationHandler(mockService)

	req := httptest.NewRequest("POST", "/api/v2/validateu", nil)
	w := httptest.NewRecorder()

	// Act
	handler.ValidateTokenU(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "invalid authorization header", resp["error"])
	assert.Equal(t, float64(http.StatusUnauthorized), resp["code"])

	mockService.AssertNotCalled(t, "ValidateTokenWithUserInfo", mock.Anything, mock.Anything)
}

func TestValidateTokenU_HMACUser_NoUserInfo(t *testing.T) {
	// Arrange
	mockService := new(MockValidationService)
	handler := NewValidationHandler(mockService)

	expiresAt := time.Now().Add(24 * time.Hour)

	// HMAC user (non-QiniuStub) - user_info is nil
	mockService.On("ValidateTokenWithUserInfo", mock.Anything, &interfaces.TokenValidateRequest{
		Token: "sk-hmac-token",
	}).Return(&interfaces.TokenValidateUResponse{
		Valid:   true,
		Message: "valid",
		TokenInfo: &interfaces.TokenInfoU{
			TokenID:   "tk_456",
			AccountID: "hmac_account_123",
			UID:       "",  // HMAC user has no UID
			IUID:      "",
			IsActive:  true,
			ExpiresAt: &expiresAt,
			UserInfo:  nil, // HMAC users don't have UserInfo
		},
	}, nil)

	mockService.On("RecordTokenUsage", mock.Anything, "sk-hmac-token").Return(nil)

	req := httptest.NewRequest("POST", "/api/v2/validateu", nil)
	req.Header.Set("Authorization", "Bearer sk-hmac-token")
	w := httptest.NewRecorder()

	// Act
	handler.ValidateTokenU(w, req)

	// Wait for goroutine to complete
	time.Sleep(10 * time.Millisecond)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp interfaces.TokenValidateUResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Valid)
	assert.NotNil(t, resp.TokenInfo)
	assert.Equal(t, "hmac_account_123", resp.TokenInfo.AccountID)
	assert.Empty(t, resp.TokenInfo.UID)
	assert.Nil(t, resp.TokenInfo.UserInfo)

	mockService.AssertExpectations(t)
}

// ========================================
// Test Helpers
// ========================================

func TestHelpers_RespondJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"message": "test"}

	respondJSON(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "test", resp["message"])
}

func TestHelpers_RespondError(t *testing.T) {
	w := httptest.NewRecorder()

	respondError(w, http.StatusBadRequest, "bad request")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "bad request", resp["error"])
	assert.Equal(t, float64(http.StatusBadRequest), resp["code"])
}
