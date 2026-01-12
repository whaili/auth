package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"bearer-token-service.v1/v2/interfaces"
)

// ValidationHandlerImpl Token 验证 Handler 实现
type ValidationHandlerImpl struct {
	validationService interfaces.ValidationService
}

// NewValidationHandler 创建验证 Handler 实例
func NewValidationHandler(validationService interfaces.ValidationService) *ValidationHandlerImpl {
	return &ValidationHandlerImpl{
		validationService: validationService,
	}
}

// ValidateToken 验证 Bearer Token
// POST /api/v2/validate
func (h *ValidationHandlerImpl) ValidateToken(w http.ResponseWriter, r *http.Request) {
	// 1. 提取 Bearer Token
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		respondError(w, http.StatusUnauthorized, "invalid authorization header")
		return
	}

	tokenValue := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. 调用验证服务
	req := &interfaces.TokenValidateRequest{
		Token: tokenValue,
	}

	resp, err := h.validationService.ValidateToken(r.Context(), req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// 3. 如果验证失败，返回 401
	if !resp.Valid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 4. 记录使用（异步，不影响响应）
	go h.validationService.RecordTokenUsage(r.Context(), tokenValue)

	// 5. 返回成功响应
	respondJSON(w, http.StatusOK, resp)
}
