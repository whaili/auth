package interfaces

// ========================================
// API Handler 接口定义
// ========================================

// AccountHandler 账户管理 API 处理器接口
type AccountHandler interface {
	// Register 注册新账户
	// POST /api/v2/accounts/register
	// Request Body: AccountRegisterRequest
	// Response: AccountRegisterResponse
	Register(w ResponseWriter, r *Request)

	// GetAccountInfo 获取当前账户信息
	// GET /api/v2/accounts/me
	// Auth: HMAC
	// Response: Account
	GetAccountInfo(w ResponseWriter, r *Request)

	// RegenerateSecretKey 重新生成 Secret Key
	// POST /api/v2/accounts/regenerate-sk
	// Auth: HMAC
	// Response: RegenerateSecretKeyResponse
	RegenerateSecretKey(w ResponseWriter, r *Request)
}

// TokenHandler Token 管理 API 处理器接口
type TokenHandler interface {
	// CreateToken 创建新 Token
	// POST /api/v2/tokens
	// Auth: HMAC
	// Request Body: TokenCreateRequest
	// Response: TokenCreateResponse
	CreateToken(w ResponseWriter, r *Request)

	// ListTokens 列出当前账户的所有 Tokens
	// GET /api/v2/tokens?active_only=true&limit=50&offset=0
	// Auth: HMAC
	// Response: TokenListResponse
	ListTokens(w ResponseWriter, r *Request)

	// GetTokenInfo 获取单个 Token 详情
	// GET /api/v2/tokens/{id}
	// Auth: HMAC
	// Response: Token
	GetTokenInfo(w ResponseWriter, r *Request)

	// UpdateTokenStatus 更新 Token 状态（启用/禁用）
	// PUT /api/v2/tokens/{id}/status
	// Auth: HMAC
	// Request Body: TokenUpdateStatusRequest
	// Response: Token
	UpdateTokenStatus(w ResponseWriter, r *Request)

	// DeleteToken 删除 Token
	// DELETE /api/v2/tokens/{id}
	// Auth: HMAC
	// Response: {"message": "Token deleted successfully"}
	DeleteToken(w ResponseWriter, r *Request)

	// GetTokenStats 获取 Token 使用统计
	// GET /api/v2/tokens/{id}/stats
	// Auth: HMAC
	// Response: TokenStatsResponse
	GetTokenStats(w ResponseWriter, r *Request)
}

// ValidationHandler Token 验证 API 处理器接口
type ValidationHandler interface {
	// ValidateToken 验证 Bearer Token
	// POST /api/v2/validate
	// Auth: Bearer Token
	// Request Header: Authorization: Bearer {token}
	// Request Body (optional): {"required_scope": "storage:read"}
	// Response: TokenValidateResponse
	ValidateToken(w ResponseWriter, r *Request)

	// ValidateWithScope 验证 Token 并检查特定权限
	// GET /api/v2/validate?scope=storage:read
	// Auth: Bearer Token
	// Response: TokenValidateResponse
	ValidateWithScope(w ResponseWriter, r *Request)
}

// AuditHandler 审计日志 API 处理器接口
type AuditHandler interface {
	// QueryAuditLogs 查询审计日志
	// GET /api/v2/audit-logs?action=create_token&start_time=2025-01-01T00:00:00Z&limit=50
	// Auth: HMAC
	// Response: AuditLogResponse
	QueryAuditLogs(w ResponseWriter, r *Request)
}

// ========================================
// Middleware 接口定义
// ========================================

// HMACAuthMiddleware HMAC 签名认证中间件接口
type HMACAuthMiddleware interface {
	// Authenticate 认证中间件
	// 验证 HMAC 签名，并将 Account 信息注入到 Context
	Authenticate(next HandlerFunc) HandlerFunc
}

// CORSMiddleware CORS 中间件接口
type CORSMiddleware interface {
	// Handle CORS 处理
	Handle(next HandlerFunc) HandlerFunc
}

// RateLimitMiddleware 限流中间件接口
type RateLimitMiddleware interface {
	// Limit 限流处理
	Limit(next HandlerFunc) HandlerFunc
}

// LoggingMiddleware 日志中间件接口
type LoggingMiddleware interface {
	// Log 记录请求日志
	Log(next HandlerFunc) HandlerFunc
}

// ========================================
// 类型别名（适配不同 HTTP 框架）
// ========================================

// 这些类型别名用于适配不同的 HTTP 框架
// 实际实现时，可以替换为 gin.Context、http.ResponseWriter 等

type (
	// ResponseWriter HTTP 响应写入器（兼容 net/http、gin 等）
	ResponseWriter interface {
		Header() map[string][]string
		Write([]byte) (int, error)
		WriteHeader(statusCode int)
	}

	// Request HTTP 请求（兼容 net/http、gin 等）
	Request interface {
		Method() string
		URL() string
		Header() map[string][]string
		Body() []byte
		Context() interface{}
	}

	// HandlerFunc 通用 HTTP 处理函数
	HandlerFunc func(w ResponseWriter, r *Request)
)

// ========================================
// API 错误定义
// ========================================

// APIError API 错误响应
type APIError struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Details   string `json:"details,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// 标准错误代码
const (
	// 4xx 客户端错误
	ErrCodeBadRequest          = 400
	ErrCodeUnauthorized        = 401
	ErrCodeForbidden           = 403
	ErrCodeNotFound            = 404
	ErrCodeConflict            = 409
	ErrCodeValidationFailed    = 422
	ErrCodeTooManyRequests     = 429

	// 5xx 服务端错误
	ErrCodeInternalServerError = 500
	ErrCodeServiceUnavailable  = 503
)

// 业务错误代码
const (
	// 认证错误 (4001-4099)
	ErrCodeInvalidSignature     = 4001
	ErrCodeTimestampExpired     = 4002
	ErrCodeAccessKeyNotFound    = 4003
	ErrCodeAccountSuspended     = 4004
	ErrCodeInvalidAuthHeader    = 4005

	// 权限错误 (4031-4099)
	ErrCodePermissionDenied     = 4031
	ErrCodeScopeNotGranted      = 4032
	ErrCodeInvalidScope         = 4033

	// Token 错误 (4041-4099)
	ErrCodeTokenNotFound        = 4041
	ErrCodeTokenExpired         = 4042
	ErrCodeTokenInactive        = 4043
	ErrCodeTokenNotBelongToAccount = 4044

	// 业务错误 (5001-5099)
	ErrCodeDuplicateEmail       = 5001
	ErrCodeDuplicateAccessKey   = 5002
	ErrCodeInvalidScopeFormat   = 5003
)
