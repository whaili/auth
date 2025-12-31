package service

import (
	"context"
	"errors"
	"time"

	"bearer-token-service.v1/v2/interfaces"
	"bearer-token-service.v1/v2/permission"
)

// TokenServiceImpl Token 管理服务实现
type TokenServiceImpl struct {
	tokenRepo     interfaces.TokenRepository
	auditRepo     interfaces.AuditLogRepository
	scopeValidator *permission.ScopeValidator
}

// NewTokenService 创建 Token 服务实例
func NewTokenService(tokenRepo interfaces.TokenRepository, auditRepo interfaces.AuditLogRepository) *TokenServiceImpl {
	return &TokenServiceImpl{
		tokenRepo:     tokenRepo,
		auditRepo:     auditRepo,
		scopeValidator: permission.NewScopeValidator(),
	}
}

// CreateToken 创建新 Token
func (s *TokenServiceImpl) CreateToken(ctx context.Context, accountID string, req *interfaces.TokenCreateRequest) (*interfaces.TokenCreateResponse, error) {
	// 1. 验证 Scope 格式
	if err := s.scopeValidator.ValidateScopes(req.Scope); err != nil {
		return nil, err
	}

	// 2. 计算过期时间（秒级精度）
	var expiresAt *time.Time
	if req.ExpiresInSeconds > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresInSeconds) * time.Second)
		expiresAt = &t
	}

	// 3. 创建 Token 对象
	token := &interfaces.Token{
		AccountID:   accountID, // 关联到账户（租户隔离）
		Description: req.Description,
		Scope:       req.Scope,
		RateLimit:   req.RateLimit,
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
		"scope":       req.Scope,
	})

	// 5. 返回响应（包含完整 Token，仅此一次）
	return &interfaces.TokenCreateResponse{
		TokenID:     token.ID,
		Token:       token.Token, // 完整 Token，仅在创建时返回
		AccountID:   token.AccountID,
		Description: token.Description,
		Scope:       token.Scope,
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
			TokenPreview:  token.Token, // hideToken(token.Token), // 隐藏部分 Token - 已注释，返回明文
			Description:   token.Description,
			Scope:         token.Scope,
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

	// 隐藏完整 Token - 已注释，返回明文
	// token.Token = hideToken(token.Token)

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

// hideToken 隐藏 Token 的中间部分，保留前后明文
// 示例: sk-abc123...***************************...xyz789
func hideToken(token string) string {
	const (
		hiddenStart  = 15 // 从第 15 个字符开始隐藏
		hiddenLength = 30 // 隐藏 30 个字符
	)

	if len(token) < hiddenStart+hiddenLength {
		return token // Token 太短，直接返回
	}

	// 将中间部分替换为星号
	bytes := []byte(token)
	for i := hiddenStart; i < hiddenStart+hiddenLength && i < len(bytes); i++ {
		bytes[i] = '*'
	}
	return string(bytes)
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
