package service

import (
	"context"
	"time"

	"github.com/qiniu/bearer-token-service/v2/interfaces"
)

// AuditServiceImpl 审计服务实现
type AuditServiceImpl struct {
	auditRepo interfaces.AuditLogRepository
}

// NewAuditService 创建审计服务实例
func NewAuditService(auditRepo interfaces.AuditLogRepository) *AuditServiceImpl {
	return &AuditServiceImpl{
		auditRepo: auditRepo,
	}
}

// Log 记录审计日志
func (s *AuditServiceImpl) Log(ctx context.Context, log *interfaces.AuditLog) error {
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}
	return s.auditRepo.Create(ctx, log)
}

// LogAction 便捷方法：记录操作
func (s *AuditServiceImpl) LogAction(ctx context.Context, accountID string, action string, resourceID string, result string, errorMsg string, requestData map[string]interface{}) error {
	log := &interfaces.AuditLog{
		AccountID:   accountID,
		Action:      action,
		ResourceID:  resourceID,
		Result:      result,
		ErrorMsg:    errorMsg,
		RequestData: requestData,
		Timestamp:   time.Now(),
	}
	return s.auditRepo.Create(ctx, log)
}

// QueryLogs 查询审计日志
func (s *AuditServiceImpl) QueryLogs(ctx context.Context, accountID string, query *interfaces.AuditLogQuery) (*interfaces.AuditLogResponse, error) {
	logs, err := s.auditRepo.ListByAccountID(ctx, accountID, query)
	if err != nil {
		return nil, err
	}

	total, err := s.auditRepo.CountByAccountID(ctx, accountID, query)
	if err != nil {
		return nil, err
	}

	return &interfaces.AuditLogResponse{
		AccountID: accountID,
		Logs:      logs,
		Total:     int(total),
	}, nil
}
