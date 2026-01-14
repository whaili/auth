package service

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"bearer-token-service.v1/v2/interfaces"
	"bearer-token-service.v1/v2/observability"
)

// ValidationServiceImpl Token 验证服务实现
type ValidationServiceImpl struct {
	tokenRepo interfaces.TokenRepository
}

// NewValidationService 创建验证服务实例
func NewValidationService(tokenRepo interfaces.TokenRepository) *ValidationServiceImpl {
	return &ValidationServiceImpl{
		tokenRepo: tokenRepo,
	}
}

// ValidateToken 验证 Token
func (s *ValidationServiceImpl) ValidateToken(ctx context.Context, req *interfaces.TokenValidateRequest) (*interfaces.TokenValidateResponse, error) {
	start := time.Now()

	// 1. 查询 Token
	token, err := s.tokenRepo.GetByTokenValue(ctx, req.Token)

	// 记录验证耗时
	duration := time.Since(start)
	observability.TokenValidationDuration.Observe(duration.Seconds())

	if err != nil {
		observability.TokenValidationsTotal.WithLabelValues("error").Inc()
		observability.LogError(ctx, "Token validation failed", err)
		return &interfaces.TokenValidateResponse{
			Valid:   false,
			Message: "internal error",
		}, err
	}

	if token == nil {
		observability.TokenValidationsTotal.WithLabelValues("not_found").Inc()
		observability.LogInfo(ctx, "Token not found")
		return &interfaces.TokenValidateResponse{
			Valid:   false,
			Message: "Token not found",
		}, nil
	}

	// 2. 检查 Token 是否激活
	if !token.IsActive {
		observability.TokenValidationsTotal.WithLabelValues("inactive").Inc()
		observability.LogInfo(ctx, "Token is inactive", slog.String("token_id", token.ID))
		return &interfaces.TokenValidateResponse{
			Valid:   false,
			Message: "Token is inactive",
		}, nil
	}

	// 3. 检查 Token 是否过期
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		observability.TokenValidationsTotal.WithLabelValues("expired").Inc()
		observability.LogInfo(ctx, "Token has expired",
			slog.String("token_id", token.ID),
			slog.Time("expired_at", *token.ExpiresAt))
		return &interfaces.TokenValidateResponse{
			Valid:   false,
			Message: "Token has expired",
		}, nil
	}

	// 4. 验证通过，返回 Token 信息
	observability.TokenValidationsTotal.WithLabelValues("valid").Inc()
	observability.LogDebug(ctx, "Token validation succeeded",
		slog.String("token_id", token.ID),
		slog.String("account_id", token.AccountID),
		slog.Float64("duration_ms", float64(duration.Microseconds())/1000))

	tokenInfo := &interfaces.TokenInfo{
		TokenID:  token.ID,
		IsActive: token.IsActive,
	}

	// 处理时间字段（避免零值时间）
	if token.ExpiresAt != nil {
		tokenInfo.ExpiresAt = token.ExpiresAt
	}
	if token.LastUsedAt != nil {
		tokenInfo.LastUsedAt = token.LastUsedAt
	}

	// 根据 account_id 格式判断用户类型
	if uid, isQiniuStub := extractUIDFromAccountID(token.AccountID); isQiniuStub {
		// QiniuStub 用户：返回 UID 和 IUID（如果存在）
		tokenInfo.UID = uid
		tokenInfo.IUID = token.IUID // 从 Token 中读取 IUID
	} else {
		// HMAC 用户：返回 AccountID
		tokenInfo.AccountID = token.AccountID
	}

	return &interfaces.TokenValidateResponse{
		Valid:     true,
		Message:   "Token is valid",
		TokenInfo: tokenInfo,
	}, nil
}

// RecordTokenUsage 记录 Token 使用
func (s *ValidationServiceImpl) RecordTokenUsage(ctx context.Context, tokenValue string) error {
	token, err := s.tokenRepo.GetByTokenValue(ctx, tokenValue)
	if err != nil || token == nil {
		return errors.New("token not found")
	}

	// 增加使用计数（异步，不阻塞主流程）
	go s.tokenRepo.IncrementUsage(context.Background(), token.ID)

	return nil
}

// extractUIDFromAccountID 从 account_id 中提取 UID
// 如果是 QiniuStub 用户（格式: qiniu_{uid}），返回 (uid, true)
// 否则返回 ("", false)
func extractUIDFromAccountID(accountID string) (string, bool) {
	// 检查是否是 qiniu_ 前缀
	if !strings.HasPrefix(accountID, "qiniu_") {
		return "", false
	}

	// 提取 UID 部分
	uidStr := strings.TrimPrefix(accountID, "qiniu_")
	// 验证 UID 是否是有效的数字
	_, err := strconv.ParseUint(uidStr, 10, 32)
	if err != nil {
		return "", false
	}

	return uidStr, true
}

