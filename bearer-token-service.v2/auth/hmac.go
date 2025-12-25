package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ========================================
// HMAC 签名认证实现
// ========================================

// SignatureBuilder 签名构建器
type SignatureBuilder struct{}

// NewSignatureBuilder 创建签名构建器
func NewSignatureBuilder() *SignatureBuilder {
	return &SignatureBuilder{}
}

// BuildStringToSign 构建待签名字符串
//
// 签名字符串格式：
//   HTTP_METHOD + "\n" +
//   URI_PATH + "\n" +
//   TIMESTAMP + "\n" +
//   REQUEST_BODY
//
// 示例：
//   POST
//   /api/v2/tokens
//   2025-12-25T10:00:00Z
//   {"description":"test","scope":["storage:read"]}
func (b *SignatureBuilder) BuildStringToSign(method, uri, timestamp, body string) string {
	return fmt.Sprintf("%s\n%s\n%s\n%s", method, uri, timestamp, body)
}

// ParseAuthHeader 解析 Authorization Header
//
// 格式: "QINIU {AccessKey}:{Signature}"
//
// 示例:
//   Input:  "QINIU AK_abc123:dGVzdHNpZ25hdHVyZQ=="
//   Output: accessKey="AK_abc123", signature="dGVzdHNpZ25hdHVyZQ=="
func (b *SignatureBuilder) ParseAuthHeader(authHeader string) (accessKey, signature string, err error) {
	// 去除首尾空格
	authHeader = strings.TrimSpace(authHeader)

	// 检查是否以 "QINIU " 开头
	if !strings.HasPrefix(authHeader, "QINIU ") {
		return "", "", errors.New("invalid auth header: must start with 'QINIU '")
	}

	// 去除前缀
	credentials := strings.TrimPrefix(authHeader, "QINIU ")

	// 分割 AccessKey 和 Signature
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid auth header: must be 'QINIU {AccessKey}:{Signature}'")
	}

	accessKey = parts[0]
	signature = parts[1]

	if accessKey == "" || signature == "" {
		return "", "", errors.New("invalid auth header: accessKey and signature cannot be empty")
	}

	return accessKey, signature, nil
}

// ========================================
// HMAC 认证器
// ========================================

// HMACAuthenticator HMAC 签名认证器
type HMACAuthenticator struct {
	builder           *SignatureBuilder
	timestampTolerance time.Duration // 时间戳容忍度（防重放攻击）
}

// NewHMACAuthenticator 创建 HMAC 认证器
func NewHMACAuthenticator(timestampTolerance time.Duration) *HMACAuthenticator {
	return &HMACAuthenticator{
		builder:           NewSignatureBuilder(),
		timestampTolerance: timestampTolerance,
	}
}

// GenerateSignature 生成 HMAC-SHA256 签名
//
// 算法：Base64(HMAC-SHA256(StringToSign, SecretKey))
func (a *HMACAuthenticator) GenerateSignature(secretKey, stringToSign string) (string, error) {
	mac := hmac.New(sha256.New, []byte(secretKey))
	_, err := mac.Write([]byte(stringToSign))
	if err != nil {
		return "", err
	}

	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return signature, nil
}

// VerifySignature 验证 HMAC 签名
//
// 使用 constant-time 比较防止时序攻击
func (a *HMACAuthenticator) VerifySignature(secretKey, receivedSignature, stringToSign string) (bool, error) {
	// 重新计算签名
	expectedSignature, err := a.GenerateSignature(secretKey, stringToSign)
	if err != nil {
		return false, err
	}

	// 使用 constant-time 比较
	return hmac.Equal([]byte(expectedSignature), []byte(receivedSignature)), nil
}

// ValidateTimestamp 验证时间戳（防重放攻击）
//
// 允许客户端时钟有一定偏差（默认 ±15 分钟）
func (a *HMACAuthenticator) ValidateTimestamp(timestampStr string) error {
	// 解析时间戳（ISO 8601 格式）
	requestTime, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %w", err)
	}

	// 计算时间差
	now := time.Now().UTC()
	timeDiff := now.Sub(requestTime)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}

	// 检查是否在容忍范围内
	if timeDiff > a.timestampTolerance {
		return fmt.Errorf("timestamp expired: diff=%v, tolerance=%v", timeDiff, a.timestampTolerance)
	}

	return nil
}

// ========================================
// 认证流程辅助方法
// ========================================

// AuthRequest 认证请求参数
type AuthRequest struct {
	Method         string // HTTP 方法
	URI            string // 请求路径
	Timestamp      string // 时间戳（X-Qiniu-Date header）
	Body           string // 请求体
	Authorization  string // Authorization header
}

// VerifyRequest 验证完整的请求
//
// 步骤：
// 1. 解析 Authorization header，提取 AccessKey 和 Signature
// 2. 验证时间戳
// 3. 构建 StringToSign
// 4. 使用 SecretKey 验证签名
//
// 返回：accessKey, valid, error
func (a *HMACAuthenticator) VerifyRequest(req *AuthRequest, secretKey string) (string, bool, error) {
	// 1. 解析 Authorization header
	accessKey, receivedSignature, err := a.builder.ParseAuthHeader(req.Authorization)
	if err != nil {
		return "", false, err
	}

	// 2. 验证时间戳
	if err := a.ValidateTimestamp(req.Timestamp); err != nil {
		return accessKey, false, err
	}

	// 3. 构建 StringToSign
	stringToSign := a.builder.BuildStringToSign(req.Method, req.URI, req.Timestamp, req.Body)

	// 4. 验证签名
	valid, err := a.VerifySignature(secretKey, receivedSignature, stringToSign)
	if err != nil {
		return accessKey, false, err
	}

	return accessKey, valid, nil
}

// ========================================
// 客户端签名工具（用于测试和文档）
// ========================================

// ClientSignatureGenerator 客户端签名生成器（用于测试）
type ClientSignatureGenerator struct {
	accessKey string
	secretKey string
	builder   *SignatureBuilder
	auth      *HMACAuthenticator
}

// NewClientSignatureGenerator 创建客户端签名生成器
func NewClientSignatureGenerator(accessKey, secretKey string) *ClientSignatureGenerator {
	return &ClientSignatureGenerator{
		accessKey: accessKey,
		secretKey: secretKey,
		builder:   NewSignatureBuilder(),
		auth:      NewHMACAuthenticator(15 * time.Minute),
	}
}

// Sign 签名请求
//
// 返回：Authorization header 值
func (g *ClientSignatureGenerator) Sign(method, uri, timestamp, body string) (string, error) {
	// 构建待签名字符串
	stringToSign := g.builder.BuildStringToSign(method, uri, timestamp, body)

	// 生成签名
	signature, err := g.auth.GenerateSignature(g.secretKey, stringToSign)
	if err != nil {
		return "", err
	}

	// 构造 Authorization header
	authHeader := fmt.Sprintf("QINIU %s:%s", g.accessKey, signature)
	return authHeader, nil
}

// SignRequest 便捷方法：签名请求并返回完整的 headers
func (g *ClientSignatureGenerator) SignRequest(method, uri, body string) (map[string]string, error) {
	// 使用当前时间作为时间戳
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// 签名
	authHeader, err := g.Sign(method, uri, timestamp, body)
	if err != nil {
		return nil, err
	}

	// 返回完整的请求头
	return map[string]string{
		"Authorization":  authHeader,
		"X-Qiniu-Date":   timestamp,
		"Content-Type":   "application/json",
	}, nil
}
