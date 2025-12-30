package permission

// ========================================
// 权限定义
// ========================================

// PermissionCategory 权限分类
type PermissionCategory struct {
	Name        string       `json:"name"`        // 分类名称（如 "storage"）
	Description string       `json:"description"` // 分类描述
	Permissions []Permission `json:"permissions"` // 该分类下的权限列表
}

// Permission 单个权限定义
type Permission struct {
	Scope       string `json:"scope"`       // 权限标识（如 "storage:read"）
	Description string `json:"description"` // 权限描述
	Example     string `json:"example"`     // 使用示例
}

// GetAllPermissions 获取所有预定义的权限
func GetAllPermissions() []PermissionCategory {
	return []PermissionCategory{
		{
			Name:        "storage",
			Description: "对象存储相关权限",
			Permissions: []Permission{
				{
					Scope:       "storage:read",
					Description: "读取存储对象",
					Example:     "下载文件、获取文件元数据",
				},
				{
					Scope:       "storage:write",
					Description: "写入存储对象",
					Example:     "上传文件、更新文件",
				},
				{
					Scope:       "storage:delete",
					Description: "删除存储对象",
					Example:     "删除文件",
				},
				{
					Scope:       "storage:list",
					Description: "列举存储对象",
					Example:     "列举bucket内容",
				},
				{
					Scope:       "storage:*",
					Description: "所有存储权限（通配符）",
					Example:     "拥有 storage 下所有操作权限",
				},
			},
		},
		{
			Name:        "cdn",
			Description: "CDN 相关权限",
			Permissions: []Permission{
				{
					Scope:       "cdn:refresh",
					Description: "刷新 CDN 缓存",
					Example:     "刷新指定URL的缓存",
				},
				{
					Scope:       "cdn:purge",
					Description: "清除 CDN 缓存",
					Example:     "清除目录缓存",
				},
				{
					Scope:       "cdn:prefetch",
					Description: "CDN 预取",
					Example:     "预取资源到CDN节点",
				},
				{
					Scope:       "cdn:*",
					Description: "所有 CDN 权限（通配符）",
					Example:     "拥有 cdn 下所有操作权限",
				},
			},
		},
		{
			Name:        "user",
			Description: "用户管理相关权限",
			Permissions: []Permission{
				{
					Scope:       "user:read",
					Description: "读取用户信息",
					Example:     "查看用户详情",
				},
				{
					Scope:       "user:write",
					Description: "修改用户信息",
					Example:     "更新用户资料",
				},
				{
					Scope:       "user:delete",
					Description: "删除用户",
					Example:     "删除用户账户",
				},
				{
					Scope:       "user:*",
					Description: "所有用户权限（通配符）",
					Example:     "拥有 user 下所有操作权限",
				},
			},
		},
		{
			Name:        "token",
			Description: "Token 管理相关权限",
			Permissions: []Permission{
				{
					Scope:       "token:read",
					Description: "读取 Token 信息",
					Example:     "查看 Token 列表、详情",
				},
				{
					Scope:       "token:write",
					Description: "创建和修改 Token",
					Example:     "创建新 Token、更新 Token 状态",
				},
				{
					Scope:       "token:delete",
					Description: "删除 Token",
					Example:     "撤销 Token",
				},
				{
					Scope:       "token:*",
					Description: "所有 Token 权限（通配符）",
					Example:     "拥有 token 下所有操作权限",
				},
			},
		},
		{
			Name:        "global",
			Description: "全局权限",
			Permissions: []Permission{
				{
					Scope:       "*",
					Description: "所有权限（全局通配符）",
					Example:     "拥有系统所有操作权限",
				},
			},
		},
	}
}

// GetPermissionScopes 获取所有权限标识列表（用于验证）
func GetPermissionScopes() []string {
	scopes := []string{}
	categories := GetAllPermissions()

	for _, category := range categories {
		for _, perm := range category.Permissions {
			scopes = append(scopes, perm.Scope)
		}
	}

	return scopes
}
