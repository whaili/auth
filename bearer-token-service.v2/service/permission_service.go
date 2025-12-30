package service

import (
	"bearer-token-service.v1/v2/permission"
)

// ========================================
// PermissionService 权限管理服务
// ========================================

// PermissionService 权限服务接口
type PermissionService interface {
	// GetAllPermissions 获取所有权限定义
	GetAllPermissions() []permission.PermissionCategory
}

// PermissionServiceImpl 权限服务实现
type PermissionServiceImpl struct{}

// NewPermissionService 创建权限服务实例
func NewPermissionService() *PermissionServiceImpl {
	return &PermissionServiceImpl{}
}

// GetAllPermissions 获取所有权限定义
func (s *PermissionServiceImpl) GetAllPermissions() []permission.PermissionCategory {
	return permission.GetAllPermissions()
}
