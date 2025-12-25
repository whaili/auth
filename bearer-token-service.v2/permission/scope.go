package permission

import (
	"errors"
	"strings"
)

// ========================================
// Scope 权限验证实现
// ========================================

// ScopeValidator Scope 权限验证器
type ScopeValidator struct{}

// NewScopeValidator 创建 Scope 验证器
func NewScopeValidator() *ScopeValidator {
	return &ScopeValidator{}
}

// HasPermission 检查 Token 的 Scope 是否包含所需权限
//
// 匹配规则：
// 1. 精确匹配: "storage:read" == "storage:read"
// 2. 全局通配: "*" 匹配所有权限
// 3. 前缀通配: "storage:*" 匹配 "storage:read", "storage:write" 等
//
// 示例：
//   tokenScopes = ["storage:*", "cdn:refresh"]
//   HasPermission(tokenScopes, "storage:read")  -> true (通配符匹配)
//   HasPermission(tokenScopes, "storage:write") -> true (通配符匹配)
//   HasPermission(tokenScopes, "cdn:refresh")   -> true (精确匹配)
//   HasPermission(tokenScopes, "cdn:purge")     -> false (不匹配)
func (v *ScopeValidator) HasPermission(tokenScopes []string, requiredScope string) bool {
	// 如果没有要求特定权限，直接通过
	if requiredScope == "" {
		return true
	}

	// 遍历 Token 的所有 Scope
	for _, scope := range tokenScopes {
		// 1. 全局通配符：拥有所有权限
		if scope == "*" {
			return true
		}

		// 2. 精确匹配
		if scope == requiredScope {
			return true
		}

		// 3. 前缀通配符匹配
		// 例如：scope = "storage:*" 可以匹配 "storage:read", "storage:write"
		if strings.HasSuffix(scope, ":*") {
			prefix := strings.TrimSuffix(scope, "*") // "storage:"
			if strings.HasPrefix(requiredScope, prefix) {
				return true
			}
		}
	}

	return false
}

// ValidateScopes 验证 Scope 格式是否合法
//
// 合法格式：
// - "*" (全局通配)
// - "resource:action" (资源:操作)
// - "resource:*" (资源通配)
//
// 示例：
//   ✅ "storage:read"
//   ✅ "storage:*"
//   ✅ "*"
//   ❌ "invalid"
//   ❌ ":read"
//   ❌ "storage:"
func (v *ScopeValidator) ValidateScopes(scopes []string) error {
	if len(scopes) == 0 {
		return errors.New("at least one scope is required")
	}

	for _, scope := range scopes {
		if err := v.validateSingleScope(scope); err != nil {
			return err
		}
	}

	return nil
}

func (v *ScopeValidator) validateSingleScope(scope string) error {
	// 全局通配符
	if scope == "*" {
		return nil
	}

	// 必须包含冒号
	if !strings.Contains(scope, ":") {
		return errors.New("invalid scope format: must be 'resource:action' or 'resource:*'")
	}

	parts := strings.Split(scope, ":")
	if len(parts) != 2 {
		return errors.New("invalid scope format: must have exactly one colon")
	}

	resource := parts[0]
	action := parts[1]

	// 资源名不能为空
	if resource == "" {
		return errors.New("invalid scope: resource cannot be empty")
	}

	// 操作不能为空
	if action == "" {
		return errors.New("invalid scope: action cannot be empty")
	}

	return nil
}

// ExpandWildcardScopes 展开通配符权限（未来可能用于显示）
//
// 示例：
//   ["storage:*"] -> ["storage:read", "storage:write", "storage:delete"]
//
// 注意：这是一个示例方法，实际实现需要根据系统定义的权限列表
func (v *ScopeValidator) ExpandWildcardScopes(scopes []string) []string {
	// 预定义的权限映射
	scopeDefinitions := map[string][]string{
		"storage": {"read", "write", "delete", "list"},
		"cdn":     {"refresh", "purge", "prefetch"},
		"user":    {"read", "write", "delete"},
	}

	expanded := []string{}

	for _, scope := range scopes {
		// 全局通配
		if scope == "*" {
			expanded = append(expanded, "*")
			continue
		}

		// 前缀通配
		if strings.HasSuffix(scope, ":*") {
			resource := strings.TrimSuffix(scope, ":*")
			if actions, exists := scopeDefinitions[resource]; exists {
				for _, action := range actions {
					expanded = append(expanded, resource+":"+action)
				}
			} else {
				// 未定义的资源，保持原样
				expanded = append(expanded, scope)
			}
		} else {
			// 普通权限，直接添加
			expanded = append(expanded, scope)
		}
	}

	return expanded
}

// MatchScopes 批量检查权限
//
// 返回：所有权限都匹配返回 true，否则返回 false
func (v *ScopeValidator) MatchScopes(tokenScopes []string, requiredScopes []string) bool {
	for _, required := range requiredScopes {
		if !v.HasPermission(tokenScopes, required) {
			return false
		}
	}
	return true
}

// GetMissingScopes 获取缺失的权限
//
// 返回：Token 缺少的权限列表
func (v *ScopeValidator) GetMissingScopes(tokenScopes []string, requiredScopes []string) []string {
	missing := []string{}

	for _, required := range requiredScopes {
		if !v.HasPermission(tokenScopes, required) {
			missing = append(missing, required)
		}
	}

	return missing
}
