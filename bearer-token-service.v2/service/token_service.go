package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"bearer-token-service.v1/v2/auth"
	"bearer-token-service.v1/v2/interfaces"
)

// TokenServiceImpl Token 管理服务实现
type TokenServiceImpl struct {
	tokenRepo interfaces.TokenRepository
	auditRepo interfaces.AuditLogRepository
}

// NewTokenService 创建 Token 服务实例
func NewTokenService(tokenRepo interfaces.TokenRepository, auditRepo interfaces.AuditLogRepository) *TokenServiceImpl {
	return &TokenServiceImpl{
		tokenRepo: tokenRepo,
		auditRepo: auditRepo,
	}
}

// CreateToken 创建新 Token
func (s *TokenServiceImpl) CreateToken(ctx context.Context, accountID string, req *interfaces.TokenCreateRequest) (*interfaces.TokenCreateResponse, error) {
	// 1. 计算过期时间（秒级精度）
	var expiresAt *time.Time
	if req.ExpiresInSeconds > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresInSeconds) * time.Second)
		expiresAt = &t
	}

	// 2. 从 Context 中提取 IUID（如果是 QiniuStub 认证）
	var iuid string
	if qstubUser, ok := ctx.Value("qstub_user").(*auth.QstubUserInfo); ok {
		iuid = qstubUser.IamUid
	}

	// 3. 创建 Token 对象
	token := &interfaces.Token{
		AccountID:   accountID, // 关联到账户（租户隔离）
		Description: req.Description,
		RateLimit:   req.RateLimit,
		IUID:        iuid,      // 保存 IUID（如果存在）
		ExpiresAt:   expiresAt,
		IsActive:    true,
		Prefix:      req.Prefix, // 自定义前缀
	}

	// Token 值由 Repository 自动生成
	err := s.tokenRepo.Create(ctx, token)
	if err != nil {
		s.logAction(ctx, accountID, interfaces.AuditActionCreateToken, "", interfaces.AuditResultFailure, err.Error(), nil)
		return nil, err
	}

	// 4. 记录审计日志
	s.logAction(ctx, accountID, interfaces.AuditActionCreateToken, token.ID, interfaces.AuditResultSuccess, "", map[string]interface{}{
		"description": req.Description,
	})

	// 5. 返回响应（包含完整 Token，仅此一次）
	return &interfaces.TokenCreateResponse{
		TokenID:     token.ID,
		Token:       token.Token, // 完整 Token，仅在创建时返回
		AccountID:   token.AccountID,
		Description: token.Description,
		RateLimit:   token.RateLimit,
		CreatedAt:   token.CreatedAt,
		ExpiresAt:   token.ExpiresAt,
		IsActive:    token.IsActive,
	}, nil
}

// ListTokens 列出账户的所有 Tokens
func (s *TokenServiceImpl) ListTokens(ctx context.Context, accountID string, activeOnly bool, limit, offset int) (*interfaces.TokenListResponse, error) {
	// 查询 Tokens（自动租户隔离）
	tokens, err := s.tokenRepo.ListByAccountID(ctx, accountID, activeOnly, limit, offset)
	if err != nil {
		return nil, err
	}

	// 统计总数
	total, err := s.tokenRepo.CountByAccountID(ctx, accountID, activeOnly)
	if err != nil {
		return nil, err
	}

	// 转换为摘要格式（隐藏完整 Token）
	var tokenBriefs []interfaces.TokenBrief
	now := time.Now()
	for _, token := range tokens {
		brief := interfaces.TokenBrief{
			TokenID:       token.ID,
			TokenPreview:  hideToken(token.Token),
			Description:   token.Description,
			RateLimit:     token.RateLimit,
			CreatedAt:     token.CreatedAt,
			IsActive:      token.IsActive,
			Status:        calculateTokenStatus(&token, now), // 动态计算状态
			TotalRequests: token.TotalRequests,
		}

		// 处理时间字段（避免零值时间）
		if token.ExpiresAt != nil {
			brief.ExpiresAt = token.ExpiresAt
		}
		if token.LastUsedAt != nil {
			brief.LastUsedAt = token.LastUsedAt
		}

		tokenBriefs = append(tokenBriefs, brief)
	}

	return &interfaces.TokenListResponse{
		AccountID: accountID,
		Tokens:    tokenBriefs,
		Total:     int(total),
	}, nil
}

// GetTokenInfo 获取 Token 详情
func (s *TokenServiceImpl) GetTokenInfo(ctx context.Context, accountID string, tokenID string) (*interfaces.Token, error) {
	token, err := s.tokenRepo.GetByID(ctx, tokenID)
	if err != nil {
		return nil, err
	}

	if token == nil {
		return nil, errors.New("token not found")
	}

	// 验证 Token 归属（租户隔离）
	if token.AccountID != accountID {
		return nil, errors.New("permission denied: token does not belong to this account")
	}

	// 隐藏完整 Token
	token.Token = hideToken(token.Token)

	return token, nil
}

// UpdateTokenStatus 更新 Token 状态
func (s *TokenServiceImpl) UpdateTokenStatus(ctx context.Context, accountID string, tokenID string, isActive bool) error {
	// 验证归属
	token, err := s.tokenRepo.GetByID(ctx, tokenID)
	if err != nil {
		return err
	}
	if token == nil {
		return errors.New("token not found")
	}
	if token.AccountID != accountID {
		return errors.New("permission denied")
	}

	// 更新状态
	err = s.tokenRepo.UpdateStatus(ctx, tokenID, isActive)
	if err != nil {
		s.logAction(ctx, accountID, interfaces.AuditActionUpdateToken, tokenID, interfaces.AuditResultFailure, err.Error(), nil)
		return err
	}

	s.logAction(ctx, accountID, interfaces.AuditActionUpdateToken, tokenID, interfaces.AuditResultSuccess, "", map[string]interface{}{
		"is_active": isActive,
	})

	return nil
}

// DeleteToken 删除 Token
func (s *TokenServiceImpl) DeleteToken(ctx context.Context, accountID string, tokenID string) error {
	// 验证归属
	token, err := s.tokenRepo.GetByID(ctx, tokenID)
	if err != nil {
		return err
	}
	if token == nil {
		return errors.New("token not found")
	}
	if token.AccountID != accountID {
		return errors.New("permission denied")
	}

	// 删除
	err = s.tokenRepo.Delete(ctx, tokenID)
	if err != nil {
		s.logAction(ctx, accountID, interfaces.AuditActionDeleteToken, tokenID, interfaces.AuditResultFailure, err.Error(), nil)
		return err
	}

	s.logAction(ctx, accountID, interfaces.AuditActionDeleteToken, tokenID, interfaces.AuditResultSuccess, "", nil)

	return nil
}

// GetTokenStats 获取 Token 使用统计
func (s *TokenServiceImpl) GetTokenStats(ctx context.Context, accountID string, tokenID string) (*interfaces.TokenStatsResponse, error) {
	token, err := s.tokenRepo.GetByID(ctx, tokenID)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, errors.New("token not found")
	}
	if token.AccountID != accountID {
		return nil, errors.New("permission denied")
	}

	resp := &interfaces.TokenStatsResponse{
		TokenID:       token.ID,
		TotalRequests: token.TotalRequests,
		CreatedAt:     token.CreatedAt,
	}

	// 处理时间字段（避免零值时间）
	if token.LastUsedAt != nil {
		resp.LastUsedAt = token.LastUsedAt
	}

	return resp, nil
}

// ========================================
// 辅助方法
// ========================================

func (s *TokenServiceImpl) logAction(ctx context.Context, accountID, action, resourceID, result, errorMsg string, requestData map[string]interface{}) {
	log := &interfaces.AuditLog{
		AccountID:   accountID,
		Action:      action,
		ResourceID:  resourceID,
		Result:      result,
		ErrorMsg:    errorMsg,
		RequestData: requestData,
		Timestamp:   time.Now(),
	}

	s.auditRepo.Create(ctx, log)
}

// hideToken 隐藏 Token 的中间部分
// 格式: prefix全部显示 + 8字符 + **** + 8字符
// 示例: sk-a1b2c3d4****e5f6g7h8
func hideToken(token string) string {
	// 找到 prefix 分隔符位置
	prefixEnd := strings.Index(token, "-")
	if prefixEnd == -1 {
		prefixEnd = 0
	} else {
		prefixEnd++ // 包含 "-"
	}

	suffix := token[prefixEnd:] // prefix 之后的部分

	const (
		showBefore = 8 // 显示前 8 个字符
		showAfter  = 8 // 显示后 8 个字符
		maskLen    = 4 // 4 个星号
	)

	// 如果 suffix 太短，直接返回原 token
	if len(suffix) < showBefore+showAfter {
		return token
	}

	// prefix + 前8字符 + **** + 后8字符
	return token[:prefixEnd] + suffix[:showBefore] + strings.Repeat("*", maskLen) + suffix[len(suffix)-showAfter:]
}

// calculateTokenStatus 计算 Token 的综合状态
func calculateTokenStatus(token *interfaces.Token, now time.Time) string {
	// 1. 已停用
	if !token.IsActive {
		return interfaces.TokenStatusDisabled
	}

	// 2. 已过期（ExpiresAt 不为 nil 且已过期）
	if token.ExpiresAt != nil && token.ExpiresAt.Before(now) {
		return interfaces.TokenStatusExpired
	}

	// 3. 正常（未过期且已激活）
	return interfaces.TokenStatusNormal
}
