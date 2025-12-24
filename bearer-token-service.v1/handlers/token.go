package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"bearer-token-service.v1/models"
	"bearer-token-service.v1/repo"
	"github.com/gorilla/mux"
)

// TokenHandler 处理 token 相关的请求
type TokenHandler struct {
	tokenRepo repo.MongoTokenRepository
	adminRepo repo.MongoAdminRepository
}

func NewTokenHandler(tokenRepo repo.MongoTokenRepository, adminRepo repo.MongoAdminRepository) *TokenHandler {
	return &TokenHandler{
		tokenRepo: tokenRepo,
		adminRepo: adminRepo,
	}
}

func hiddeToken(token string, from int, len int) string {
	bytes := []byte(token)
	copy(bytes[from:(from+len)], []byte(strings.Repeat(string('*'), len)))
	return string(bytes)
}

// CreateToken 创建新的 bearer token
func (h *TokenHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	// 验证管理员权限
	if !h.isAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 生成随机 token
	tokenValue, err := generateToken(models.TokenLength)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// 如果有前缀，则添加到 token value 前面
	if req.Prefix != "" {
		tokenValue = req.Prefix + tokenValue
	} else {
		tokenValue = models.DefaultTokenPrefix + tokenValue
	}

	// 计算过期时间
	var expiresAt time.Time
	if req.ExpiresInDays > 0 {
		expiresAt = time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
	}

	token := models.Token{
		Token:       tokenValue,
		Description: req.Description,
		CreatedAt:   time.Now(),
		IsActive:    true,
	}

	if !expiresAt.IsZero() {
		token.ExpiresAt = expiresAt
	}

	// 存储到数据库
	if err := h.tokenRepo.CreateToken(r.Context(), &token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Token created successfully",
		"token":      tokenValue,
		"token_info": token,
	})
}

// ListTokens 列出所有 tokens
func (h *TokenHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	// 验证管理员权限
	if !h.isAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tokens, err := h.tokenRepo.ListTokens(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 避免泄露敏感信息，隐藏 token 的一部分
	var outputTokens []models.Token
	for _, token := range tokens {
		token.Token = hiddeToken(token.Token, models.HiddenTokenIndex, models.HiddenTokenLength)
		outputTokens = append(outputTokens, token)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tokens": outputTokens,
	})
}

// ToggleTokenStatus 启用/禁用 token
func (h *TokenHandler) ToggleTokenStatus(w http.ResponseWriter, r *http.Request) {
	// 验证管理员权限
	if !h.isAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	var req models.TokenStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updatedToken, err := h.tokenRepo.UpdateTokenStatus(r.Context(), id, req.IsActive)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if updatedToken.Token != "" {
		updatedToken.Token = hiddeToken(updatedToken.Token, models.HiddenTokenIndex, models.HiddenTokenLength)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Token status updated successfully",
		"token_info": updatedToken,
	})
}

// DeleteToken 删除 token
func (h *TokenHandler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	// 验证管理员权限
	if !h.isAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.tokenRepo.DeleteToken(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Token deleted successfully",
	})
}

// ValidateToken 验证 bearer token
func (h *TokenHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		h.respondValidationError(w, "Invalid authorization header")
		return
	}

	tokenValue := strings.TrimPrefix(auth, "Bearer ")
	token, err := h.tokenRepo.GetToken(r.Context(), tokenValue)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if token == nil {
		h.respondValidationError(w, "Token not found")
		return
	}

	if !token.IsActive {
		h.respondValidationError(w, "Token is inactive")
		return
	}

	if !token.ExpiresAt.IsZero() && token.ExpiresAt.Before(time.Now()) {
		h.respondValidationError(w, "Token has expired")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.ValidateResponse{
		Valid:     true,
		Message:   "Token is valid",
		TokenInfo: token,
	})
}

func (h *TokenHandler) respondValidationError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(models.ValidateResponse{
		Valid:   false,
		Message: message,
	})
}

// isAdmin 检查请求是否有管理员权限
func (h *TokenHandler) isAdmin(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false
	}

	// Basic Auth 格式: "Basic base64(username:password)"
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || parts[0] != "Basic" {
		return false
	}

	payload, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	pair := strings.SplitN(string(payload), ":", 2)
	if len(pair) != 2 {
		return false
	}

	valid, err := h.adminRepo.VerifyAdmin(r.Context(), pair[0], pair[1])
	if err != nil {
		return false
	}

	return valid
}

// generateToken 生成随机 token
func generateToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
