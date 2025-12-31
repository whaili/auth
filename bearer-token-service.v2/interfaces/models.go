package interfaces

import "time"

// ========================================
// 数据模型定义
// ========================================

// Account 租户账户模型
type Account struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	Email     string    `bson:"email" json:"email"`
	Company   string    `bson:"company" json:"company"`
	AccessKey string    `bson:"access_key" json:"access_key"` // AK_xxx
	SecretKey string    `bson:"secret_key" json:"-"`          // bcrypt 加密，不返回客户端
	Status    string    `bson:"status" json:"status"`         // active, suspended
	QiniuUID  uint32    `bson:"qiniu_uid,omitempty" json:"qiniu_uid,omitempty"` // 七牛 UID（可选）
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// Token Bearer Token 模型（带权限控制）
type Token struct {
	ID          string     `bson:"_id,omitempty" json:"token_id"`
	AccountID   string     `bson:"account_id" json:"account_id"`         // 关联到账户
	Token       string     `bson:"token" json:"token"`                   // 实际的 token 值
	Description string     `bson:"description" json:"description"`       // Token 描述
	Scope       []string   `bson:"scope" json:"scope"`                   // 权限范围
	RateLimit   *RateLimit `bson:"rate_limit,omitempty" json:"rate_limit,omitempty"`
	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
	ExpiresAt   *time.Time `bson:"expires_at,omitempty" json:"expires_at,omitempty"` // nil 表示永不过期
	IsActive    bool       `bson:"is_active" json:"is_active"`
	Prefix      string     `bson:"-" json:"-"`                           // 自定义前缀（不存储到数据库）

	// 使用统计
	TotalRequests int64      `bson:"total_requests" json:"total_requests"`
	LastUsedAt    *time.Time `bson:"last_used_at,omitempty" json:"last_used_at,omitempty"` // nil 表示从未使用
}

// RateLimit API 频率限制
type RateLimit struct {
	RequestsPerMinute int `bson:"requests_per_minute" json:"requests_per_minute"`
	RequestsPerHour   int `bson:"requests_per_hour,omitempty" json:"requests_per_hour,omitempty"`
	RequestsPerDay    int `bson:"requests_per_day,omitempty" json:"requests_per_day,omitempty"`
}

// AuditLog 审计日志模型
type AuditLog struct {
	ID          string                 `bson:"_id,omitempty" json:"id"`
	AccountID   string                 `bson:"account_id" json:"account_id"`
	Action      string                 `bson:"action" json:"action"`           // create_token, delete_token, validate_token
	ResourceID  string                 `bson:"resource_id" json:"resource_id"` // token_id
	IP          string                 `bson:"ip" json:"ip"`
	UserAgent   string                 `bson:"user_agent" json:"user_agent"`
	RequestData map[string]interface{} `bson:"request_data,omitempty" json:"request_data,omitempty"`
	Result      string                 `bson:"result" json:"result"`           // success, failure
	ErrorMsg    string                 `bson:"error_msg,omitempty" json:"error_msg,omitempty"`
	Timestamp   time.Time              `bson:"timestamp" json:"timestamp"`
}

// ========================================
// 请求/响应模型
// ========================================

// AccountRegisterRequest 账户注册请求
type AccountRegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Company  string `json:"company" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

// AccountRegisterResponse 账户注册响应
type AccountRegisterResponse struct {
	AccountID string    `json:"account_id"`
	Email     string    `json:"email"`
	Company   string    `json:"company"`
	AccessKey string    `json:"access_key"`
	SecretKey string    `json:"secret_key"` // 仅在注册时返回一次
	CreatedAt time.Time `json:"created_at"`
}

// RegenerateSecretKeyResponse 重新生成 SK 响应
type RegenerateSecretKeyResponse struct {
	AccessKey string    `json:"access_key"`
	SecretKey string    `json:"secret_key"` // 新的 SecretKey，仅此一次显示
	UpdatedAt time.Time `json:"updated_at"`
}

// TokenCreateRequest 创建 Token 请求
type TokenCreateRequest struct {
	Description      string     `json:"description" binding:"required"`
	Scope            []string   `json:"scope" binding:"required,min=1"`            // 至少一个权限
	ExpiresInSeconds int64      `json:"expires_in_seconds,omitempty"`              // 0 表示永不过期，支持秒级精度
	RateLimit        *RateLimit `json:"rate_limit,omitempty"`
	Prefix           string     `json:"prefix,omitempty"`                          // 自定义 Token 前缀，默认 "sk-"
}

// TokenCreateResponse 创建 Token 响应
type TokenCreateResponse struct {
	TokenID     string     `json:"token_id"`
	Token       string     `json:"token"`       // 完整 token，仅在创建时返回
	AccountID   string     `json:"account_id"`
	Description string     `json:"description"`
	Scope       []string   `json:"scope"`
	RateLimit   *RateLimit `json:"rate_limit,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"` // nil 表示永不过期
	IsActive    bool       `json:"is_active"`
}

// TokenListResponse Token 列表响应
type TokenListResponse struct {
	AccountID string       `json:"account_id"`
	Tokens    []TokenBrief `json:"tokens"`
	Total     int          `json:"total"`
}

// TokenBrief Token 摘要信息（隐藏完整 token）
type TokenBrief struct {
	TokenID       string     `json:"token_id"`
	TokenPreview  string     `json:"token_preview"`  // 中间隐藏，如 "sk-abc123...******************************...xyz789"
	Description   string     `json:"description"`
	Scope         []string   `json:"scope"`
	RateLimit     *RateLimit `json:"rate_limit,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`  // nil 表示永不过期
	IsActive      bool       `json:"is_active"`
	Status        string     `json:"status"` // Token 综合状态：normal=正常，expired=已过期，disabled=已停用
	TotalRequests int64      `json:"total_requests"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"` // nil 表示从未使用
}

// TokenUpdateStatusRequest 更新 Token 状态请求
type TokenUpdateStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// TokenValidateRequest Token 验证请求
type TokenValidateRequest struct {
	Token         string `json:"-"`                                  // 从 Authorization header 提取
	RequiredScope string `json:"required_scope,omitempty"`           // 可选：要求的权限
}

// TokenValidateResponse Token 验证响应
type TokenValidateResponse struct {
	Valid            bool       `json:"valid"`
	Message          string     `json:"message"`
	TokenInfo        *TokenInfo `json:"token_info,omitempty"`
	PermissionCheck  *PermissionCheckResult `json:"permission_check,omitempty"`
}

// TokenInfo Token 基本信息（用于验证响应）
type TokenInfo struct {
	TokenID    string     `json:"token_id"`
	AccountID  string     `json:"account_id,omitempty"`  // HMAC 用户使用
	UID        uint32     `json:"uid,omitempty"`         // QiniuStub 用户使用（从 account_id 提取）
	Scope      []string   `json:"scope"`
	IsActive   bool       `json:"is_active"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`  // nil 表示永不过期
	LastUsedAt *time.Time `json:"last_used_at,omitempty"` // nil 表示从未使用
}

// PermissionCheckResult 权限检查结果
type PermissionCheckResult struct {
	Requested string `json:"requested"` // 请求的权限
	Granted   bool   `json:"granted"`   // 是否授权
}

// TokenStatsResponse Token 使用统计响应
type TokenStatsResponse struct {
	TokenID       string     `json:"token_id"`
	TotalRequests int64      `json:"total_requests"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"` // nil 表示从未使用
	CreatedAt     time.Time  `json:"created_at"`
	DailyStats    []DailyStat `json:"daily_stats,omitempty"` // 未来：每日统计
}

// DailyStat 每日统计（未来扩展）
type DailyStat struct {
	Date     string `json:"date"`     // YYYY-MM-DD
	Requests int64  `json:"requests"`
}

// AuditLogQuery 审计日志查询参数
type AuditLogQuery struct {
	Action     string    `form:"action"`      // 过滤操作类型
	ResourceID string    `form:"resource_id"` // 过滤资源 ID
	StartTime  time.Time `form:"start_time"`  // 开始时间
	EndTime    time.Time `form:"end_time"`    // 结束时间
	Limit      int       `form:"limit"`       // 返回数量，默认 50
	Offset     int       `form:"offset"`      // 偏移量
}

// AuditLogResponse 审计日志响应
type AuditLogResponse struct {
	AccountID string      `json:"account_id"`
	Logs      []AuditLog  `json:"logs"`
	Total     int         `json:"total"`
}

// ========================================
// 常量定义
// ========================================

const (
	// Account Status
	AccountStatusActive    = "active"
	AccountStatusSuspended = "suspended"

	// Token Status
	TokenStatusNormal   = "normal"   // 正常（未过期且已激活）
	TokenStatusExpired  = "expired"  // 已过期
	TokenStatusDisabled = "disabled" // 已停用

	// Token Prefix (保持与 V1 兼容)
	TokenPrefix = "sk-"

	// AccessKey Prefix (通用前缀)
	AccessKeyPrefix = "AK_"

	// SecretKey Prefix (通用前缀)
	SecretKeyPrefix = "SK_"

	// Scope Wildcards
	ScopeAll = "*"

	// Audit Actions
	AuditActionCreateToken    = "create_token"
	AuditActionDeleteToken    = "delete_token"
	AuditActionUpdateToken    = "update_token"
	AuditActionValidateToken  = "validate_token"
	AuditActionRegenerateKey  = "regenerate_secret_key"

	// Audit Results
	AuditResultSuccess = "success"
	AuditResultFailure = "failure"
)
