package service

import (
	"context"
	"errors"
	"time"

	"bearer-token-service.v1/v2/interfaces"
	"bearer-token-service.v1/v2/repository"
)

// AccountServiceImpl 账户管理服务实现
type AccountServiceImpl struct {
	accountRepo interfaces.AccountRepository
	auditRepo   interfaces.AuditLogRepository
}

// NewAccountService 创建账户服务实例
func NewAccountService(accountRepo interfaces.AccountRepository, auditRepo interfaces.AuditLogRepository) *AccountServiceImpl {
	return &AccountServiceImpl{
		accountRepo: accountRepo,
		auditRepo:   auditRepo,
	}
}

// Register 注册新账户
func (s *AccountServiceImpl) Register(ctx context.Context, req *interfaces.AccountRegisterRequest) (*interfaces.AccountRegisterResponse, error) {
	// 1. 验证邮箱是否已存在
	existingAccount, err := s.accountRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existingAccount != nil {
		return nil, errors.New("email already registered")
	}

	// 2. 生成 SecretKey（明文）
	secretKey, err := repository.GenerateSecretKey()
	if err != nil {
		return nil, err
	}

	// 3. 加密 SecretKey
	hashedSecretKey, err := repository.HashSecretKey(secretKey)
	if err != nil {
		return nil, err
	}

	// 4. 创建账户
	account := &interfaces.Account{
		Email:     req.Email,
		Company:   req.Company,
		SecretKey: hashedSecretKey, // 存储加密后的 SecretKey
		Status:    interfaces.AccountStatusActive,
	}

	// 注意：AccessKey 由 Repository 自动生成
	err = s.accountRepo.Create(ctx, account)
	if err != nil {
		return nil, err
	}

	// 5. 记录审计日志
	s.logAction(ctx, account.ID, "register_account", account.ID, "success", "", nil)

	// 6. 返回响应（包含明文 SecretKey，仅此一次）
	return &interfaces.AccountRegisterResponse{
		AccountID: account.ID,
		Email:     account.Email,
		Company:   account.Company,
		AccessKey: account.AccessKey,
		SecretKey: secretKey, // 明文 SecretKey，仅在注册时返回
		CreatedAt: account.CreatedAt,
	}, nil
}

// GetAccountInfo 获取账户信息
func (s *AccountServiceImpl) GetAccountInfo(ctx context.Context, accountID string) (*interfaces.Account, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	if account == nil {
		return nil, errors.New("account not found")
	}

	// 不返回 SecretKey
	account.SecretKey = ""

	return account, nil
}

// RegenerateSecretKey 重新生成 SecretKey
func (s *AccountServiceImpl) RegenerateSecretKey(ctx context.Context, accountID string) (*interfaces.RegenerateSecretKeyResponse, error) {
	// 1. 验证账户存在
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.New("account not found")
	}

	// 2. 生成新的 SecretKey
	newSecretKey, err := repository.GenerateSecretKey()
	if err != nil {
		return nil, err
	}

	// 3. 加密新的 SecretKey
	hashedSecretKey, err := repository.HashSecretKey(newSecretKey)
	if err != nil {
		return nil, err
	}

	// 4. 更新到数据库
	err = s.accountRepo.UpdateSecretKey(ctx, accountID, hashedSecretKey)
	if err != nil {
		return nil, err
	}

	// 5. 记录审计日志
	s.logAction(ctx, accountID, interfaces.AuditActionRegenerateKey, accountID, interfaces.AuditResultSuccess, "", nil)

	// 6. 返回新的 SecretKey（明文，仅此一次）
	return &interfaces.RegenerateSecretKeyResponse{
		AccessKey: account.AccessKey,
		SecretKey: newSecretKey, // 明文，仅此一次
		UpdatedAt: time.Now(),
	}, nil
}

// SuspendAccount 暂停账户
func (s *AccountServiceImpl) SuspendAccount(ctx context.Context, accountID string) error {
	err := s.accountRepo.UpdateStatus(ctx, accountID, interfaces.AccountStatusSuspended)
	if err != nil {
		return err
	}

	s.logAction(ctx, accountID, "suspend_account", accountID, interfaces.AuditResultSuccess, "", nil)
	return nil
}

// ActivateAccount 激活账户
func (s *AccountServiceImpl) ActivateAccount(ctx context.Context, accountID string) error {
	err := s.accountRepo.UpdateStatus(ctx, accountID, interfaces.AccountStatusActive)
	if err != nil {
		return err
	}

	s.logAction(ctx, accountID, "activate_account", accountID, interfaces.AuditResultSuccess, "", nil)
	return nil
}

// ========================================
// 辅助方法
// ========================================

func (s *AccountServiceImpl) logAction(ctx context.Context, accountID, action, resourceID, result, errorMsg string, requestData map[string]interface{}) {
	log := &interfaces.AuditLog{
		AccountID:   accountID,
		Action:      action,
		ResourceID:  resourceID,
		Result:      result,
		ErrorMsg:    errorMsg,
		RequestData: requestData,
		Timestamp:   time.Now(),
	}

	// 忽略日志错误，不影响主流程
	s.auditRepo.Create(ctx, log)
}
