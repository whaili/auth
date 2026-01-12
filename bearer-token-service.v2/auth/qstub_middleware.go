package auth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ========================================
// QiniuStub 认证中间件
// ========================================

// QstubAuthMiddleware QiniuStub 认证中间件
// 只支持 QiniuStub 认证（七牛内部用户系统）
type QstubAuthMiddleware struct {
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
	UID     string `json:"uid"`             // 必需: 用户ID (主账户)
	Utype   uint32 `json:"ut"`              // 可选: 用户类型
	Appid   uint64 `json:"app,omitempty"`   // 可选: 应用ID(未使用)
	IamUid  string `json:"iuid,omitempty"`  // 可选: IAM 用户ID (子账户)
	Access  string `json:"ak,omitempty"`    // 可选: AccessKey(未使用)
	EndUser string `json:"eu,omitempty"`    // 可选: 最终用户(未使用)
	Email   string `json:"email,omitempty"` // 可选: 邮箱
}

// AccountInfo 简化的账户信息
type AccountInfo struct {
	ID    string
	Email string
}

// NewQstubAuthMiddleware 创建 QiniuStub 认证中间件
func NewQstubAuthMiddleware(qiniuUIDMapper QiniuUIDMapper) *QstubAuthMiddleware {
	return &QstubAuthMiddleware{
		qiniuUIDMapper: qiniuUIDMapper,
	}
}

// Authenticate QiniuStub 认证处理器
//
// 认证方式：
// Authorization 头必须以 "QiniuStub " 开头，格式为 URL 参数格式
//
// 支持两种格式：
// 1. 主账户格式: QiniuStub uid=12345&ut=1
// 2. IAM 子账户格式: QiniuStub uid=12345&ut=1&iuid=8901234
func (m *QstubAuthMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// 检查是否为 QiniuStub 认证
		if !strings.HasPrefix(authHeader, "QiniuStub ") {
			m.respondError(w, http.StatusUnauthorized, "missing or invalid Authorization header, expected 'QiniuStub uid=xxx&ut=x'")
			return
		}

		// 处理 QiniuStub 认证
		m.authenticateQstub(w, r, next)
	}
}

// authenticateQstub 处理 Qstub Bearer Token 认证
func (m *QstubAuthMiddleware) authenticateQstub(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
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
	ctx = context.WithValue(ctx, "qstub_user", qstubUser) // 存储完整的 Qstub 用户信息

	// 5. 调用下一个 handler
	next.ServeHTTP(w, r.WithContext(ctx))
}

// parseQstubToken 解析 QiniuStub Token
// 格式: "QiniuStub uid=12345&ut=1" 或 "QiniuStub uid=12345&ut=1&iuid=8901234"
func (m *QstubAuthMiddleware) parseQstubToken(authHeader string) (*QstubUserInfo, error) {
	if !strings.HasPrefix(authHeader, "QiniuStub ") {
		return nil, fmt.Errorf("invalid qstub token format")
	}
	return m.parseQstubURLParams(authHeader)
}

// parseQstubURLParams 解析 URL 参数格式的 QiniuStub Token
// 例如: "QiniuStub uid=12345&ut=1&iuid=8901234"
func (m *QstubAuthMiddleware) parseQstubURLParams(authHeader string) (*QstubUserInfo, error) {
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
		case "iuid":
			userInfo.IamUid = value
		case "ak":
			userInfo.Access = value
		case "eu":
			userInfo.EndUser = value
		case "email":
			userInfo.Email = value
		}
	}

	if userInfo.UID == "" {
		return nil, fmt.Errorf("uid is required")
	}

	return userInfo, nil
}

// respondError 返回错误响应
func (m *QstubAuthMiddleware) respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(`{"error": "` + message + `"}`))
}

// ========================================
// 辅助函数：提取认证信息
// ========================================

// ExtractAuthMethod 从 Context 中提取认证方式
func ExtractAuthMethod(ctx context.Context) string {
	method, ok := ctx.Value("auth_method").(string)
	if !ok {
		return "qstub" // 默认为 qstub
	}
	return method
}

// ExtractQstubUser 从 Context 中提取 Qstub 用户信息
func ExtractQstubUser(ctx context.Context) *QstubUserInfo {
	qstubUser, ok := ctx.Value("qstub_user").(*QstubUserInfo)
	if !ok {
		return nil
	}
	return qstubUser
}
