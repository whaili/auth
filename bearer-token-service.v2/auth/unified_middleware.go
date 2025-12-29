package auth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ========================================
// 统一认证中间件（支持多种认证方式）
// ========================================

// UnifiedAuthMiddleware 统一认证中间件
// 支持：
// 1. HMAC 签名认证（AccessKey/SecretKey）
// 2. Qstub Bearer Token 认证（七牛内部用户系统）
type UnifiedAuthMiddleware struct {
	hmacAuth       *HMACMiddleware
	accountFetcher AccountFetcher
	qiniuUIDMapper QiniuUIDMapper // 将七牛 UID 映射到 account_id
}

// QiniuUIDMapper 七牛 UID 映射接口
// 用于将七牛 UID 转换为系统的 account_id
type QiniuUIDMapper interface {
	// GetAccountIDByQiniuUID 根据七牛 UID 获取 account_id
	// 如果 UID 不存在，可以选择自动创建账户或返回错误
	GetAccountIDByQiniuUID(ctx context.Context, qiniuUID uint32) (string, error)
}

// QstubUserInfo 七牛 Qstub 用户信息
type QstubUserInfo struct {
	UID     string `json:"uid"`             // 必需: 用户ID
	Utype   uint32 `json:"ut"`              // 可选: 用户类型
	Appid   uint64 `json:"app,omitempty"`   // 可选: 应用ID(未使用)
	Access  string `json:"ak,omitempty"`    // 可选: AccessKey(未使用)
	EndUser string `json:"eu,omitempty"`    // 可选: 最终用户(未使用)
	Email   string `json:"email,omitempty"` // 可选: 邮箱
}

// NewUnifiedAuthMiddleware 创建统一认证中间件
func NewUnifiedAuthMiddleware(
	accountFetcher AccountFetcher,
	qiniuUIDMapper QiniuUIDMapper,
	timestampTolerance time.Duration,
) *UnifiedAuthMiddleware {
	return &UnifiedAuthMiddleware{
		hmacAuth:       NewHMACMiddleware(accountFetcher, timestampTolerance),
		accountFetcher: accountFetcher,
		qiniuUIDMapper: qiniuUIDMapper,
	}
}

// Authenticate 统一认证处理器
//
// 认证优先级：
// 1. 如果存在 X-Qiniu-Date 头，使用 HMAC 签名认证
// 2. 如果 Authorization 以 "QiniuStub " 开头，使用 QiniuStub 认证（URL 参数格式）
// 3. 否则返回 401
func (m *UnifiedAuthMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		timestampHeader := r.Header.Get("X-Qiniu-Date")

		// 策略 1: HMAC 签名认证（优先级最高）
		if timestampHeader != "" {
			// 使用 HMAC 认证
			m.hmacAuth.Authenticate(next).ServeHTTP(w, r)
			return
		}

		// 策略 2: QiniuStub 认证（URL 参数格式）
		if strings.HasPrefix(authHeader, "QiniuStub ") {
			m.authenticateQstub(w, r, next)
			return
		}

		// 策略 3: 如果是 QINIU 格式但没有时间戳，提示错误
		if strings.HasPrefix(authHeader, "QINIU ") {
			m.respondError(w, http.StatusUnauthorized, "missing X-Qiniu-Date header for HMAC authentication")
			return
		}

		// 无法识别的认证方式
		m.respondError(w, http.StatusUnauthorized, "unsupported authentication method")
	}
}

// authenticateQstub 处理 Qstub Bearer Token 认证
func (m *UnifiedAuthMiddleware) authenticateQstub(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	authHeader := r.Header.Get("Authorization")

	// 1. 解析 Qstub Token
	qstubUser, err := m.parseQstubToken(authHeader)
	if err != nil {
		m.respondError(w, http.StatusUnauthorized, "invalid qstub token: "+err.Error())
		return
	}

	// 2. 转换七牛 UID 为 uint32
	qiniuUID, err := strconv.ParseUint(qstubUser.UID, 10, 32)
	if err != nil {
		m.respondError(w, http.StatusUnauthorized, "invalid qiniu uid format")
		return
	}

	// 3. 映射七牛 UID 到 account_id
	accountID, err := m.qiniuUIDMapper.GetAccountIDByQiniuUID(r.Context(), uint32(qiniuUID))
	if err != nil {
		m.respondError(w, http.StatusUnauthorized, "failed to map qiniu uid to account: "+err.Error())
		return
	}

	// 4. 构建简化的账户信息并注入到 Context
	account := &AccountInfo{
		ID:    accountID,
		Email: qstubUser.Email,
	}

	ctx := context.WithValue(r.Context(), "account", account)
	ctx = context.WithValue(ctx, "account_id", accountID)
	ctx = context.WithValue(ctx, "auth_method", "qstub") // 标记认证方式

	// 5. 调用下一个 handler
	next.ServeHTTP(w, r.WithContext(ctx))
}

// parseQstubToken 解析 QiniuStub Token
// 格式: "QiniuStub uid=12345&ut=1"
func (m *UnifiedAuthMiddleware) parseQstubToken(authHeader string) (*QstubUserInfo, error) {
	if !strings.HasPrefix(authHeader, "QiniuStub ") {
		return nil, fmt.Errorf("invalid qstub token format")
	}
	return m.parseQstubURLParams(authHeader)
}

// parseQstubURLParams 解析 URL 参数格式的 QiniuStub Token
// 例如: "QiniuStub uid=12345&ut=1&app=1"
func (m *UnifiedAuthMiddleware) parseQstubURLParams(authHeader string) (*QstubUserInfo, error) {
	// 移除 "QiniuStub " 前缀
	params := strings.TrimPrefix(authHeader, "QiniuStub ")
	params = strings.TrimSpace(params)

	if params == "" {
		return nil, fmt.Errorf("empty qstub params")
	}

	// 解析 URL 参数
	userInfo := &QstubUserInfo{}
	pairs := strings.Split(params, "&")

	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key, value := kv[0], kv[1]
		switch key {
		case "uid":
			userInfo.UID = value
		case "ut":
			if ut, err := strconv.ParseUint(value, 10, 32); err == nil {
				userInfo.Utype = uint32(ut)
			}
		case "app":
			if app, err := strconv.ParseUint(value, 10, 64); err == nil {
				userInfo.Appid = app
			}
		case "ak":
			userInfo.Access = value
		case "eu":
			userInfo.EndUser = value
		case "email":
			userInfo.Email = value
		}
	}

	if userInfo.UID == "" {
		return nil, fmt.Errorf("uid is empty")
	}

	return userInfo, nil
}


// respondError 返回错误响应
func (m *UnifiedAuthMiddleware) respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(`{"error": "` + message + `"}`))
}

// ========================================
// 辅助函数：提取认证方式
// ========================================

// ExtractAuthMethod 从 Context 中提取认证方式
func ExtractAuthMethod(ctx context.Context) string {
	method, ok := ctx.Value("auth_method").(string)
	if !ok {
		return "hmac" // 默认为 HMAC
	}
	return method
}
