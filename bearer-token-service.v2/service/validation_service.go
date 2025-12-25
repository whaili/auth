package service

import (
	"context"
	"errors"
	"time"

	"bearer-token-service.v1/v2/interfaces"
	"bearer-token-service.v1/v2/permission"
)

// ValidationServiceImpl Token 验证服务实现
type ValidationServiceImpl struct {
	tokenRepo      interfaces.TokenRepository
	scopeValidator *permission.ScopeValidator
}

// NewValidationService 创建验证服务实例
func NewValidationService(tokenRepo interfaces.TokenRepository) *ValidationServiceImpl {
	return &ValidationServiceImpl{
		tokenRepo:      tokenRepo,
		scopeValidator: permission.NewScopeValidator(),
	}
}

// ValidateToken 验证 Token（带权限检查）
func (s *ValidationServiceImpl) ValidateToken(ctx context.Context, req *interfaces.TokenValidateRequest) (*interfaces.TokenValidateResponse, error) {
	// 1. 查询 Token
	token, err := s.tokenRepo.GetByTokenValue(ctx, req.Token)
	if err != nil {
		return &interfaces.TokenValidateResponse{
			Valid:   false,
			Message: "internal error",
		}, err
	}

	if token == nil {
		return &interfaces.TokenValidateResponse{
			Valid:   false,
			Message: "Token not found",
		}, nil
	}

	// 2. 检查 Token 是否激活
	if !token.IsActive {
		return &interfaces.TokenValidateResponse{
			Valid:   false,
			Message: "Token is inactive",
		}, nil
	}

	// 3. 检查 Token 是否过期
	if !token.ExpiresAt.IsZero() && token.ExpiresAt.Before(time.Now()) {
		return &interfaces.TokenValidateResponse{
			Valid:   false,
			Message: "Token has expired",
		}, nil
	}

	// 4. 如果指定了 RequiredScope，检查权限
	var permissionCheck *interfaces.PermissionCheckResult
	if req.RequiredScope != "" {
		granted := s.scopeValidator.HasPermission(token.Scope, req.RequiredScope)
		permissionCheck = &interfaces.PermissionCheckResult{
			Requested: req.RequiredScope,
			Granted:   granted,
		}

		if !granted {
			return &interfaces.TokenValidateResponse{
				Valid:            false,
				Message:          "Permission denied",
				PermissionCheck:  permissionCheck,
			}, nil
		}
	}

	// 5. 验证通过，返回 Token 信息
	return &interfaces.TokenValidateResponse{
		Valid:   true,
		Message: "Token is valid",
		TokenInfo: &interfaces.TokenInfo{
			TokenID:   token.ID,
			AccountID: token.AccountID,
			Scope:     token.Scope,
			IsActive:  token.IsActive,
			ExpiresAt: token.ExpiresAt,
		},
		PermissionCheck: permissionCheck,
	}, nil
}

// ValidateTokenWithScope 验证 Token 并检查特定权限
func (s *ValidationServiceImpl) ValidateTokenWithScope(ctx context.Context, tokenValue string, requiredScope string) (*interfaces.TokenValidateResponse, error) {
	return s.ValidateToken(ctx, &interfaces.TokenValidateRequest{
		Token:         tokenValue,
		RequiredScope: requiredScope,
	})
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
