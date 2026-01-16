package auth

import (
	"context"
	"errors"
)

// ExtractAccountIDFromContext 从 Context 中提取 account_id
func ExtractAccountIDFromContext(ctx context.Context) (string, error) {
	accountID, ok := ctx.Value("account_id").(string)
	if !ok || accountID == "" {
		return "", errors.New("account_id not found in context")
	}
	return accountID, nil
}

// ExtractAccountFromContext 从 Context 中提取完整的账户信息
func ExtractAccountFromContext(ctx context.Context) (*AccountInfo, error) {
	account, ok := ctx.Value("account").(*AccountInfo)
	if !ok {
		return nil, errors.New("account not found in context")
	}
	return account, nil
}
