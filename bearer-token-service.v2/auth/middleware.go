package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

// ========================================
// 认证中间件实现（适配 net/http）
// ========================================

// AccountFetcher 账户查询接口（由外部实现）
// 用于解耦中间件和数据访问层
type AccountFetcher interface {
	// GetAccountByAccessKey 根据 AccessKey 查询账户
	GetAccountByAccessKey(ctx context.Context, accessKey string) (*AccountInfo, error)
}

// AccountInfo 账户信息（中间件需要的最小信息）
type AccountInfo struct {
	ID        string
	Email     string
	AccessKey string
	SecretKey string // 用于验证签名（已加密存储）
	Status    string
}

// HMACMiddleware HMAC 认证中间件
type HMACMiddleware struct {
	authenticator  *HMACAuthenticator
	accountFetcher AccountFetcher
}

// NewHMACMiddleware 创建认证中间件
func NewHMACMiddleware(accountFetcher AccountFetcher, timestampTolerance time.Duration) *HMACMiddleware {
	return &HMACMiddleware{
		authenticator:  NewHMACAuthenticator(timestampTolerance),
		accountFetcher: accountFetcher,
	}
}

// Authenticate 认证中间件
//
// 功能：
// 1. 提取并验证 HMAC 签名
// 2. 查询账户信息
// 3. 验证签名
// 4. 将账户信息注入到 Context
//
// 使用方式：
//   middleware := NewHMACMiddleware(accountFetcher, 15*time.Minute)
//   http.HandleFunc("/api/v2/tokens", middleware.Authenticate(handler))
func (m *HMACMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 提取认证信息
		authHeader := r.Header.Get("Authorization")
		timestamp := r.Header.Get("X-Qiniu-Date")

		if authHeader == "" {
			m.respondError(w, http.StatusUnauthorized, "missing Authorization header")
			return
		}

		if timestamp == "" {
			m.respondError(w, http.StatusUnauthorized, "missing X-Qiniu-Date header")
			return
		}

		// 2. 读取请求体（用于签名验证）
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			m.respondError(w, http.StatusBadRequest, "failed to read request body")
			return
		}
		defer r.Body.Close()

		// 重新设置 Body，供后续 handler 使用
		r.Body = io.NopCloser(io.Reader(newBytesReader(bodyBytes)))

		// 3. 解析 AccessKey
		builder := NewSignatureBuilder()
		accessKey, receivedSignature, err := builder.ParseAuthHeader(authHeader)
		if err != nil {
			m.respondError(w, http.StatusUnauthorized, "invalid Authorization header: "+err.Error())
			return
		}

		// 4. 查询账户信息
		account, err := m.accountFetcher.GetAccountByAccessKey(r.Context(), accessKey)
		if err != nil {
			m.respondError(w, http.StatusUnauthorized, "account not found")
			return
		}

		// 5. 检查账户状态
		if account.Status != "active" {
			m.respondError(w, http.StatusForbidden, "account suspended")
			return
		}

		// 6. 验证时间戳
		if err := m.authenticator.ValidateTimestamp(timestamp); err != nil {
			m.respondError(w, http.StatusUnauthorized, "timestamp expired: "+err.Error())
			return
		}

		// 7. 构建待签名字符串
		stringToSign := builder.BuildStringToSign(r.Method, r.URL.Path, timestamp, string(bodyBytes))

		// 8. 验证签名
		valid, err := m.authenticator.VerifySignature(account.SecretKey, receivedSignature, stringToSign)
		if err != nil {
			m.respondError(w, http.StatusInternalServerError, "signature verification failed")
			return
		}

		if !valid {
			m.respondError(w, http.StatusUnauthorized, "invalid signature")
			return
		}

		// 9. 认证成功，将账户信息注入到 Context
		ctx := context.WithValue(r.Context(), "account", account)
		ctx = context.WithValue(ctx, "account_id", account.ID)

		// 10. 调用下一个 handler
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// respondError 返回错误响应
func (m *HMACMiddleware) respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(`{"error": "` + message + `"}`))
}

// ========================================
// Context 辅助函数
// ========================================

// ExtractAccountFromContext 从 Context 中提取账户信息
func ExtractAccountFromContext(ctx context.Context) (*AccountInfo, error) {
	account, ok := ctx.Value("account").(*AccountInfo)
	if !ok {
		return nil, errors.New("account not found in context")
	}
	return account, nil
}

// ExtractAccountIDFromContext 从 Context 中提取账户 ID
func ExtractAccountIDFromContext(ctx context.Context) (string, error) {
	accountID, ok := ctx.Value("account_id").(string)
	if !ok {
		return "", errors.New("account_id not found in context")
	}
	return accountID, nil
}

// ========================================
// 辅助类型（为了避免导入 bytes 包）
// ========================================

type bytesReader struct {
	s        []byte
	i        int64
}

func newBytesReader(b []byte) *bytesReader {
	return &bytesReader{s: b}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	n = copy(p, r.s[r.i:])
	r.i += int64(n)
	return
}

func (r *bytesReader) Close() error {
	return nil
}
