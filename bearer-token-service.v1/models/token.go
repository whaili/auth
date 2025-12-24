package models

import "time"

const (
	TokenLength = 32 // Token 的长度
	// DefaultTokenPrefix 默认的 token 前缀
	DefaultTokenPrefix = "sk-"
	HiddenTokenIndex   = 15 // 隐藏 token 的起始位置
	HiddenTokenLength  = 30 // 隐藏 token 的长度
)

// Token 表示一个 bearer token
type Token struct {
	ID          string    `bson:"_id,omitempty" json:"id"`
	Token       string    `bson:"token" json:"token"`
	Prefix      string    `bson:"prefix,omitempty" json:"prefix,omitempty"` // 可选的前缀
	Description string    `bson:"description" json:"description"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	ExpiresAt   time.Time `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	IsActive    bool      `bson:"is_active" json:"is_active"`
}

// TokenRequest 创建 token 的请求体
type TokenRequest struct {
	Description   string `json:"description"`
	ExpiresInDays int    `json:"expires_in_days"`
	Prefix        string `json:"prefix,omitempty"` // 可选的前缀
}

// TokenStatusRequest 更新 token 状态的请求体
type TokenStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// ValidateResponse token 验证响应
type ValidateResponse struct {
	Valid     bool   `json:"valid"`
	Message   string `json:"message"`
	TokenInfo *Token `json:"token_info,omitempty"`
}

// AdminUser 管理员用户
type AdminUser struct {
	ID       string `bson:"_id,omitempty"`
	Username string `bson:"username"`
	Password string `bson:"password"` // bcrypt 哈希
}
