package handlers

import (
	"encoding/json"
	"net/http"

	"bearer-token-service.v1/v2/service"
)

// ========================================
// PermissionHandler 权限管理 Handler
// ========================================

type PermissionHandler struct {
	permissionService *service.PermissionServiceImpl
}

// NewPermissionHandler 创建权限 Handler
func NewPermissionHandler(permissionService *service.PermissionServiceImpl) *PermissionHandler {
	return &PermissionHandler{
		permissionService: permissionService,
	}
}

// GetAllPermissions 获取所有权限列表
// @Summary 获取所有可用权限列表
// @Description 返回系统定义的所有权限分类和具体权限
// @Tags Permissions
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "权限列表"
// @Router /api/v2/permissions [get]
func (h *PermissionHandler) GetAllPermissions(w http.ResponseWriter, r *http.Request) {
	permissions := h.permissionService.GetAllPermissions()

	response := map[string]interface{}{
		"categories": permissions,
		"total":      len(permissions),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
