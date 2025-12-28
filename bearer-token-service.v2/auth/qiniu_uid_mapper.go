package auth

import (
	"context"
	"fmt"
)

// ========================================
// 七牛 UID 映射器实现示例
// ========================================

// SimpleQiniuUIDMapper 简单的 UID 映射器
// 策略：直接将七牛 UID 转换为 account_id（适合简单场景）
type SimpleQiniuUIDMapper struct{}

func NewSimpleQiniuUIDMapper() *SimpleQiniuUIDMapper {
	return &SimpleQiniuUIDMapper{}
}

// GetAccountIDByQiniuUID 直接将七牛 UID 转换为 account_id
// 格式: qiniu_{uid}
func (m *SimpleQiniuUIDMapper) GetAccountIDByQiniuUID(ctx context.Context, qiniuUID uint32) (string, error) {
	if qiniuUID == 0 {
		return "", fmt.Errorf("invalid qiniu uid: 0")
	}
	return fmt.Sprintf("qiniu_%d", qiniuUID), nil
}

// ========================================
// 数据库支持的 UID 映射器（高级场景）
// ========================================

// DatabaseQiniuUIDMapper 基于数据库的 UID 映射器
// 支持：
// 1. 查询已存在的映射关系
// 2. 自动创建新映射（可选）
type DatabaseQiniuUIDMapper struct {
	accountRepo AccountRepository
	autoCreate  bool // 是否自动创建不存在的账户
}

// AccountRepository 账户存储接口（简化版）
type AccountRepository interface {
	// GetAccountByQiniuUID 根据七牛 UID 查询账户
	GetAccountByQiniuUID(ctx context.Context, qiniuUID uint32) (string, error)

	// CreateAccountForQiniuUID 为七牛 UID 创建新账户
	CreateAccountForQiniuUID(ctx context.Context, qiniuUID uint32, email string) (string, error)
}

func NewDatabaseQiniuUIDMapper(repo AccountRepository, autoCreate bool) *DatabaseQiniuUIDMapper {
	return &DatabaseQiniuUIDMapper{
		accountRepo: repo,
		autoCreate:  autoCreate,
	}
}

// GetAccountIDByQiniuUID 从数据库查询或创建账户
func (m *DatabaseQiniuUIDMapper) GetAccountIDByQiniuUID(ctx context.Context, qiniuUID uint32) (string, error) {
	if qiniuUID == 0 {
		return "", fmt.Errorf("invalid qiniu uid: 0")
	}

	// 1. 先尝试查询已存在的映射
	accountID, err := m.accountRepo.GetAccountByQiniuUID(ctx, qiniuUID)
	if err == nil && accountID != "" {
		return accountID, nil
	}

	// 2. 如果不存在且允许自动创建
	if m.autoCreate {
		accountID, err := m.accountRepo.CreateAccountForQiniuUID(ctx, qiniuUID, "")
		if err != nil {
			return "", fmt.Errorf("failed to create account for qiniu uid %d: %w", qiniuUID, err)
		}
		return accountID, nil
	}

	// 3. 不允许自动创建，返回错误
	return "", fmt.Errorf("account not found for qiniu uid %d", qiniuUID)
}
