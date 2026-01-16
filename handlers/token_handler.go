package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

	"github.com/qiniu/bearer-token-service/v2/auth"
	"github.com/qiniu/bearer-token-service/v2/interfaces"
	"github.com/gorilla/mux"
)

// prefixRegex 校验 prefix：只允许小写字母、数字、下划线
var prefixRegex = regexp.MustCompile(`^[a-z0-9_]+$`)

// TokenHandlerImpl Token 管理 Handler 实现
type TokenHandlerImpl struct {
	tokenService interfaces.TokenService
}

// NewTokenHandler 创建 Token Handler 实例
func NewTokenHandler(tokenService interfaces.TokenService) *TokenHandlerImpl {
	return &TokenHandlerImpl{
		tokenService: tokenService,
	}
}

// CreateToken 创建新 Token
// POST /api/v2/tokens
func (h *TokenHandlerImpl) CreateToken(w http.ResponseWriter, r *http.Request) {
	accountID, err := auth.ExtractAccountIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req interfaces.TokenCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 校验 prefix 参数
	if req.Prefix != "" {
		if len(req.Prefix) > 12 {
			respondError(w, http.StatusBadRequest, "prefix length must not exceed 12 characters")
			return
		}
		if !prefixRegex.MatchString(req.Prefix) {
			respondError(w, http.StatusBadRequest, "prefix must contain only lowercase letters, numbers, and underscores")
			return
		}
	}

	resp, err := h.tokenService.CreateToken(r.Context(), accountID, &req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, resp)
}

// ListTokens 列出当前账户的所有 Tokens
// GET /api/v2/tokens?active_only=true&limit=50&offset=0
func (h *TokenHandlerImpl) ListTokens(w http.ResponseWriter, r *http.Request) {
	accountID, err := auth.ExtractAccountIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// 解析查询参数
	activeOnly := r.URL.Query().Get("active_only") == "true"
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit == 0 {
		limit = 50
	}

	resp, err := h.tokenService.ListTokens(r.Context(), accountID, activeOnly, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// GetTokenInfo 获取单个 Token 详情
// GET /api/v2/tokens/{id}
func (h *TokenHandlerImpl) GetTokenInfo(w http.ResponseWriter, r *http.Request) {
	accountID, err := auth.ExtractAccountIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	tokenID := vars["id"]

	token, err := h.tokenService.GetTokenInfo(r.Context(), accountID, tokenID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, token)
}

// UpdateTokenStatus 更新 Token 状态（启用/禁用）
// PUT /api/v2/tokens/{id}/status
func (h *TokenHandlerImpl) UpdateTokenStatus(w http.ResponseWriter, r *http.Request) {
	accountID, err := auth.ExtractAccountIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	tokenID := vars["id"]

	var req interfaces.TokenUpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.tokenService.UpdateTokenStatus(r.Context(), accountID, tokenID, req.IsActive)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Token status updated successfully",
	})
}

// DeleteToken 删除 Token
// DELETE /api/v2/tokens/{id}
func (h *TokenHandlerImpl) DeleteToken(w http.ResponseWriter, r *http.Request) {
	accountID, err := auth.ExtractAccountIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	tokenID := vars["id"]

	err = h.tokenService.DeleteToken(r.Context(), accountID, tokenID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Token deleted successfully",
	})
}

// GetTokenStats 获取 Token 使用统计
// GET /api/v2/tokens/{id}/stats
func (h *TokenHandlerImpl) GetTokenStats(w http.ResponseWriter, r *http.Request) {
	accountID, err := auth.ExtractAccountIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	vars := mux.Vars(r)
	tokenID := vars["id"]

	stats, err := h.tokenService.GetTokenStats(r.Context(), accountID, tokenID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, stats)
}
