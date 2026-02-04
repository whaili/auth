package interfaces

import (
	"context"
	"time"
)

// ========================================
// Repository 接口定义
// ========================================

// AccountRepository 账户数据访问接口
type AccountRepository interface {
	// Create 创建新账户
	Create(ctx context.Context, account *Account) error

	// GetByAccessKey 根据 AccessKey 查询账户
	GetByAccessKey(ctx context.Context, accessKey string) (*Account, error)

	// GetByEmail 根据邮箱查询账户
	GetByEmail(ctx context.Context, email string) (*Account, error)

	// GetByID 根据 ID 查询账户
	GetByID(ctx context.Context, id string) (*Account, error)

	// UpdateSecretKey 更新 SecretKey
	UpdateSecretKey(ctx context.Context, accountID string, newSecretKey string) error

	// UpdateStatus 更新账户状态
	UpdateStatus(ctx context.Context, accountID string, status string) error

	// List 列出所有账户（管理员功能）
	List(ctx context.Context, limit, offset int) ([]Account, error)

	// Count 统计账户数量
	Count(ctx context.Context) (int64, error)
}

// TokenRepository Token 数据访问接口
type TokenRepository interface {
	// Create 创建新 Token
	Create(ctx context.Context, token *Token) error

	// GetByID 根据 ID 查询 Token
	GetByID(ctx context.Context, tokenID string) (*Token, error)

	// GetByTokenValue 根据 token 值查询 Token
	GetByTokenValue(ctx context.Context, tokenValue string) (*Token, error)

	// ListByAccountID 查询账户的所有 Tokens（租户隔离）
	ListByAccountID(ctx context.Context, accountID string, activeOnly bool, limit, offset int) ([]Token, error)

	// CountByAccountID 统计账户的 Token 数量
	CountByAccountID(ctx context.Context, accountID string, activeOnly bool) (int64, error)

	// UpdateStatus 更新 Token 状态
	UpdateStatus(ctx context.Context, tokenID string, isActive bool) error

	// Delete 删除 Token
	Delete(ctx context.Context, tokenID string) error

	// IncrementUsage 增加使用次数
	IncrementUsage(ctx context.Context, tokenID string) error

	// UpdateLastUsed 更新最后使用时间
	UpdateLastUsed(ctx context.Context, tokenID string, lastUsedAt time.Time) error

	// DeleteExpired 删除过期的 Tokens
	DeleteExpired(ctx context.Context) (int64, error)
}

// AuditLogRepository 审计日志数据访问接口
type AuditLogRepository interface {
	// Create 创建审计日志
	Create(ctx context.Context, log *AuditLog) error

	// ListByAccountID 查询账户的审计日志
	ListByAccountID(ctx context.Context, accountID string, query *AuditLogQuery) ([]AuditLog, error)

	// CountByAccountID 统计账户的审计日志数量
	CountByAccountID(ctx context.Context, accountID string, query *AuditLogQuery) (int64, error)

	// DeleteOldLogs 删除旧日志（数据清理）
	DeleteOldLogs(ctx context.Context, olderThan time.Time) (int64, error)
}

// UserInfoRepository 用户信息数据访问接口（MySQL）
type UserInfoRepository interface {
	// GetUserInfoByUID 根据 UID 查询用户信息
	GetUserInfoByUID(ctx context.Context, uid uint32) (*UserInfo, error)
}


// ========================================
// Service 接口定义
// ========================================

// AccountService 账户管理服务接口
type AccountService interface {
	// Register 注册新账户
	Register(ctx context.Context, req *AccountRegisterRequest) (*AccountRegisterResponse, error)

	// GetAccountInfo 获取账户信息
	GetAccountInfo(ctx context.Context, accountID string) (*Account, error)

	// RegenerateSecretKey 重新生成 SecretKey
	RegenerateSecretKey(ctx context.Context, accountID string) (*RegenerateSecretKeyResponse, error)

	// SuspendAccount 暂停账户
	SuspendAccount(ctx context.Context, accountID string) error

	// ActivateAccount 激活账户
	ActivateAccount(ctx context.Context, accountID string) error
}

// TokenService Token 管理服务接口
type TokenService interface {
	// CreateToken 创建新 Token
	CreateToken(ctx context.Context, accountID string, req *TokenCreateRequest) (*TokenCreateResponse, error)

	// ListTokens 列出账户的所有 Tokens
	ListTokens(ctx context.Context, accountID string, activeOnly bool, limit, offset int) (*TokenListResponse, error)

	// GetTokenInfo 获取 Token 详情
	GetTokenInfo(ctx context.Context, accountID string, tokenID string) (*Token, error)

	// UpdateTokenStatus 更新 Token 状态
	UpdateTokenStatus(ctx context.Context, accountID string, tokenID string, isActive bool) error

	// DeleteToken 删除 Token
	DeleteToken(ctx context.Context, accountID string, tokenID string) error

	// GetTokenStats 获取 Token 使用统计
	GetTokenStats(ctx context.Context, accountID string, tokenID string) (*TokenStatsResponse, error)
}

// ValidationService Token 验证服务接口
type ValidationService interface {
	// ValidateToken 验证 Token
	ValidateToken(ctx context.Context, req *TokenValidateRequest) (*TokenValidateResponse, error)

	// ValidateTokenWithUserInfo 验证 Token 并返回扩展用户信息
	ValidateTokenWithUserInfo(ctx context.Context, req *TokenValidateRequest) (*TokenValidateUResponse, error)

	// RecordTokenUsage 记录 Token 使用
	RecordTokenUsage(ctx context.Context, tokenValue string) error
}

// AuditService 审计服务接口
type AuditService interface {
	// Log 记录审计日志
	Log(ctx context.Context, log *AuditLog) error

	// LogAction 便捷方法：记录操作
	LogAction(ctx context.Context, accountID string, action string, resourceID string, result string, errorMsg string, requestData map[string]interface{}) error

	// QueryLogs 查询审计日志
	QueryLogs(ctx context.Context, accountID string, query *AuditLogQuery) (*AuditLogResponse, error)
}

// ========================================
// Authentication 接口定义
// ========================================

// HMACAuthenticator HMAC 签名认证接口
type HMACAuthenticator interface {
	// VerifySignature 验证 HMAC 签名
	VerifySignature(accessKey string, signature string, stringToSign string) (bool, error)

	// GenerateSignature 生成 HMAC 签名（客户端使用）
	GenerateSignature(secretKey string, stringToSign string) (string, error)

	// ValidateTimestamp 验证时间戳（防重放攻击）
	ValidateTimestamp(timestamp string, tolerance time.Duration) error

	// ExtractAccountFromRequest 从请求中提取账户信息
	ExtractAccountFromRequest(ctx context.Context, authHeader string, timestamp string, method string, uri string, body string) (*Account, error)
}

// SignatureBuilder 签名构建器接口
type SignatureBuilder interface {
	// BuildStringToSign 构建待签名字符串
	BuildStringToSign(method string, uri string, timestamp string, body string) string

	// ParseAuthHeader 解析 Authorization Header
	ParseAuthHeader(authHeader string) (accessKey string, signature string, err error)
}
