package interfaces

import "time"

// ========================================
// 数据模型定义
// ========================================

// Account 租户账户模型
type Account struct {
	ID        string     `bson:"_id,omitempty" json:"id"`
	Email     string     `bson:"email" json:"email"`
	Company   string     `bson:"company" json:"company"`
	AccessKey string     `bson:"access_key" json:"access_key"` // AK_xxx
	SecretKey string     `bson:"secret_key" json:"-"`          // bcrypt 加密，不返回客户端
	Status    string     `bson:"status" json:"status"`         // active, suspended
	RateLimit *RateLimit `bson:"rate_limit,omitempty" json:"rate_limit,omitempty"` // 账户级限流配置
	QiniuUID  uint32     `bson:"qiniu_uid,omitempty" json:"qiniu_uid,omitempty"` // 七牛 UID（可选）
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
}

// Token Bearer Token 模型
type Token struct {
	ID          string     `bson:"_id,omitempty" json:"token_id"`
	AccountID   string     `bson:"account_id" json:"account_id"`         // 关联到账户
	Token       string     `bson:"token" json:"token"`                   // 实际的 token 值
	Description string     `bson:"description" json:"description"`       // Token 描述
	RateLimit   *RateLimit `bson:"rate_limit,omitempty" json:"rate_limit,omitempty"`
	IUID        string     `bson:"iuid,omitempty" json:"iuid,omitempty"` // IAM 用户ID（从 QiniuStub 认证中提取）
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
	TokenPreview  string     `json:"token_preview"`  // 中间隐藏，如 "sk-a1b2c3d4****e5f6g7h8"
	Description   string     `json:"description"`
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
	Token string `json:"-"` // 从 Authorization header 提取
}

// TokenValidateResponse Token 验证响应
type TokenValidateResponse struct {
	Valid     bool       `json:"valid"`
	Message   string     `json:"message"`
	TokenInfo *TokenInfo `json:"token_info,omitempty"`
}

// TokenInfo Token 基本信息（用于验证响应）
type TokenInfo struct {
	TokenID    string     `json:"token_id"`
	AccountID  string     `json:"account_id,omitempty"`  // HMAC 用户使用
	UID        string     `json:"uid,omitempty"`         // QiniuStub 用户使用（从 account_id 提取）
	IUID       string     `json:"iuid,omitempty"`        // IAM 用户ID（当请求中包含 iuid 时返回，用于标识IAM用户）
	IsActive   bool       `json:"is_active"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`  // nil 表示永不过期
	LastUsedAt *time.Time `json:"last_used_at,omitempty"` // nil 表示从未使用
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

// ========================================
// /api/v2/validateu 扩展模型
// ========================================

// UserInfo 扩展用户信息（从 MySQL 查询）
type UserInfo struct {
	UID            uint32    `json:"uid"`
	Email          string    `json:"email"`
	Username       string    `json:"username"`                  // 显示名称
	Utype          uint32    `json:"utype"`                     // 用户类型位掩码
	Activated      bool      `json:"activated"`                 // 是否已激活
	DisabledType   int       `json:"disabled_type"`             // 冻结类型
	DisabledReason string    `json:"disabled_reason,omitempty"` // 冻结原因
	DisabledAt     *time.Time `json:"disabled_at,omitempty"`    // 冻结时间
	ParentUID      uint32    `json:"parent_uid,omitempty"`      // 父账户 UID
	CreatedAt      int64     `json:"created_at"`                // Unix 时间戳（秒）
	UpdatedAt      int64     `json:"updated_at"`                // Unix 时间戳（秒）
	LastLoginAt    int64     `json:"last_login_at,omitempty"`   // Unix 时间戳（秒）
}

// IsDisabled 检查用户是否被禁用（bit 28）
func (u *UserInfo) IsDisabled() bool {
	return u.Utype&UserTypeDisabled != 0
}

// IsBuffered 检查用户是否处于缓冲期（bit 16）
func (u *UserInfo) IsBuffered() bool {
	return u.Utype&UserTypeBuffered > 0
}

// IsOverseas 检查是否为海外用户（bit 29）
func (u *UserInfo) IsOverseas() bool {
	return u.Utype&UserTypeOverseas > 0
}

// IsOverseasStd 检查是否为海外标准用户（bit 30）
func (u *UserInfo) IsOverseasStd() bool {
	return u.Utype&UserTypeOverseasStd > 0
}

// IsEnterprise 检查是否为企业用户（bit 2）
func (u *UserInfo) IsEnterprise() bool {
	return u.Utype&UserTypeEnterprise > 0
}

// TokenValidateUResponse /api/v2/validateu 响应（扩展了用户信息）
type TokenValidateUResponse struct {
	Valid     bool      `json:"valid"`
	Message   string    `json:"message"`
	TokenInfo *TokenInfoU `json:"token_info,omitempty"`
}

// TokenInfoU Token 信息（包含扩展用户信息）
type TokenInfoU struct {
	TokenID    string     `json:"token_id"`
	AccountID  string     `json:"account_id,omitempty"`  // HMAC 用户使用
	UID        string     `json:"uid,omitempty"`         // QiniuStub 用户使用（从 account_id 提取）
	IUID       string     `json:"iuid,omitempty"`        // IAM 用户ID
	IsActive   bool       `json:"is_active"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`  // nil 表示永不过期
	LastUsedAt *time.Time `json:"last_used_at,omitempty"` // nil 表示从未使用

	// 扩展用户信息（MySQL 查询结果，查询失败时为 nil）
	UserInfo   *UserInfo  `json:"user_info,omitempty"`
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

	// Audit Actions
	AuditActionCreateToken    = "create_token"
	AuditActionDeleteToken    = "delete_token"
	AuditActionUpdateToken    = "update_token"
	AuditActionValidateToken  = "validate_token"
	AuditActionRegenerateKey  = "regenerate_secret_key"

	// Audit Results
	AuditResultSuccess = "success"
	AuditResultFailure = "failure"

	// ========================================
	// Utype 用户类型位掩码常量
	// ========================================

	// Basic User Types
	UserTypeQBox         = 0       // 普通七牛用户
	UserTypeAdmin        = 1 << 0  // 管理员（bit 0）
	UserTypeVIP          = 1 << 1  // VIP（bit 1）
	UserTypeStdUser      = 1 << 2  // 标准/企业用户（bit 2）
	UserTypeStdUser2     = 1 << 3  // 企业虚拟用户（bit 3）
	UserTypeExpUser      = 1 << 4  // 体验用户（bit 4）
	UserTypeParentUser   = 1 << 5  // 父账户（bit 5）
	UserTypeOp           = 1 << 6  // 运维（bit 6）
	UserTypeSupport      = 1 << 7  // 支持（bit 7）
	UserTypeCC           = 1 << 8  // 呼叫中心（bit 8）
	UserTypeQCOS         = 1 << 9  // QCOS 用户（bit 9）
	UserTypePili         = 1 << 10 // Pili 用户（bit 10）
	UserTypeFusion       = 1 << 11 // Fusion 用户（bit 11）
	UserTypePandora      = 1 << 12 // Pandora 用户（bit 12）
	UserTypeDistribution = 1 << 13 // 分发用户（bit 13）
	UserTypeQVM          = 1 << 14 // QVM 用户（bit 14）
	UserTypeUnregistered = 1 << 15 // 未注册（bit 15）

	// Special Status Bits
	UserTypeBuffered     = 1 << 16 // 缓冲期/宽限期（bit 16）
	UserTypeUsers        = 1 << 17 // 用户标志（bit 17）
	UserTypeSudoers      = 1 << 18 // 超级用户标志（bit 18）

	UserTypeDisabled     = 1 << 28 // 已禁用（bit 28）
	UserTypeOverseas     = 1 << 29 // 海外用户（bit 29）
	UserTypeOverseasStd  = 1 << 30 // 海外标准用户（bit 30）

	// Aliases
	UserTypeEnterprise       = UserTypeStdUser
	UserTypeEnterpriseVUser  = UserTypeStdUser2
)
