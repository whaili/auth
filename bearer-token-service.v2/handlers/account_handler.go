package handlers

import (
	"encoding/json"
	"net/http"

	"bearer-token-service.v1/v2/auth"
	"bearer-token-service.v1/v2/interfaces"
)

// AccountHandlerImpl 账户管理 Handler 实现
type AccountHandlerImpl struct {
	accountService interfaces.AccountService
}

// NewAccountHandler 创建账户 Handler 实例
func NewAccountHandler(accountService interfaces.AccountService) *AccountHandlerImpl {
	return &AccountHandlerImpl{
		accountService: accountService,
	}
}

// Register 注册新账户
// POST /api/v2/accounts/register
func (h *AccountHandlerImpl) Register(w http.ResponseWriter, r *http.Request) {
	var req interfaces.AccountRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 调用 Service
	resp, err := h.accountService.Register(r.Context(), &req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, resp)
}

// GetAccountInfo 获取当前账户信息
// GET /api/v2/accounts/me
func (h *AccountHandlerImpl) GetAccountInfo(w http.ResponseWriter, r *http.Request) {
	// 从 Context 提取账户 ID（由认证中间件注入）
	accountID, err := auth.ExtractAccountIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	account, err := h.accountService.GetAccountInfo(r.Context(), accountID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, account)
}

// RegenerateSecretKey 重新生成 Secret Key
// POST /api/v2/accounts/regenerate-sk
func (h *AccountHandlerImpl) RegenerateSecretKey(w http.ResponseWriter, r *http.Request) {
	accountID, err := auth.ExtractAccountIDFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.accountService.RegenerateSecretKey(r.Context(), accountID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// ========================================
// 辅助函数
// ========================================

func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
