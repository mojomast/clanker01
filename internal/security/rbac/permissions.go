package rbac

import (
	"context"
	"fmt"
	"strings"
)

type PermissionChecker struct {
	policy *Policy
}

func NewPermissionChecker(policy *Policy) *PermissionChecker {
	return &PermissionChecker{
		policy: policy,
	}
}

func (pc *PermissionChecker) Policy() *Policy {
	return pc.policy
}

func (pc *PermissionChecker) Check(ctx context.Context, userID string, resource ResourceType, action Action, resourceID string) error {
	perm := Permission{
		Resource:   resource,
		Action:     action,
		ResourceID: resourceID,
	}
	return pc.policy.CheckPermission(ctx, userID, perm)
}

func (pc *PermissionChecker) CheckWildcard(ctx context.Context, userID string, resource ResourceType, action Action) error {
	perm := Permission{
		Resource:   resource,
		Action:     action,
		ResourceID: "*",
	}
	return pc.policy.CheckPermission(ctx, userID, perm)
}

func (pc *PermissionChecker) CheckMultiple(ctx context.Context, userID string, perms []Permission) error {
	for _, perm := range perms {
		if err := pc.policy.CheckPermission(ctx, userID, perm); err != nil {
			return fmt.Errorf("permission denied for %s: %w", perm.String(), err)
		}
	}
	return nil
}

func (pc *PermissionChecker) CheckAny(ctx context.Context, userID string, perms []Permission) error {
	if !pc.policy.HasAnyPermission(ctx, userID, perms) {
		return fmt.Errorf("none of the required permissions are granted")
	}
	return nil
}

func (pc *PermissionChecker) CheckAll(ctx context.Context, userID string, perms []Permission) error {
	if !pc.policy.HasAllPermissions(ctx, userID, perms) {
		return fmt.Errorf("not all required permissions are granted")
	}
	return nil
}

func (pc *PermissionChecker) HasAccessToResource(ctx context.Context, userID string, resource ResourceType, action Action, resourceID string) bool {
	perm := Permission{
		Resource:   resource,
		Action:     action,
		ResourceID: resourceID,
	}
	return pc.policy.HasPermission(ctx, userID, perm)
}

func (pc *PermissionChecker) HasWildcardAccess(ctx context.Context, userID string, resource ResourceType, action Action) bool {
	wildcardPerm := Permission{
		Resource:   resource,
		Action:     action,
		ResourceID: "*",
	}

	if pc.policy.HasPermission(ctx, userID, wildcardPerm) {
		return true
	}

	genericPerm := Permission{
		Resource:   resource,
		Action:     action,
		ResourceID: "",
	}

	return pc.policy.HasPermission(ctx, userID, genericPerm)
}

func (pc *PermissionChecker) GetAccessibleResources(ctx context.Context, userID string, resource ResourceType, action Action) []string {
	permissions := pc.policy.GetUserPermissions(ctx, userID)
	accessible := make(map[string]bool)

	for _, perm := range permissions {
		if perm.Resource == resource && (perm.Action == action || perm.Action == "*") {
			if perm.ResourceID == "*" {
				return []string{"*"}
			}
			if perm.ResourceID != "" {
				accessible[perm.ResourceID] = true
			}
		}
	}

	result := make([]string, 0, len(accessible))
	for id := range accessible {
		result = append(result, id)
	}

	return result
}

func (pc *PermissionChecker) GetUserPermissions(ctx context.Context, userID string) []Permission {
	return pc.policy.GetUserPermissions(ctx, userID)
}

func (pc *PermissionChecker) GetUserRoles(ctx context.Context, userID string) []Role {
	return pc.policy.GetUserRoles(userID)
}

func (pc *PermissionChecker) ValidatePermissionString(permStr string) error {
	parts := strings.Split(permStr, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid permission format, expected resource:action[:resourceID]")
	}

	resource := ResourceType(parts[0])
	action := Action(parts[1])

	validResources := map[ResourceType]bool{
		ResourceAgent:  true,
		ResourceTask:   true,
		ResourceSkill:  true,
		ResourceConfig: true,
		ResourceSystem: true,
	}

	validActions := map[Action]bool{
		ActionCreate:  true,
		ActionRead:    true,
		ActionUpdate:  true,
		ActionDelete:  true,
		ActionList:    true,
		ActionExecute: true,
	}

	if !validResources[resource] {
		return fmt.Errorf("invalid resource type: %s", resource)
	}

	if action != "*" && !validActions[action] {
		return fmt.Errorf("invalid action: %s", action)
	}

	return nil
}

type PermissionRequest struct {
	UserID     string
	Resource   ResourceType
	Action     Action
	ResourceID string
}

type PermissionResult struct {
	Allowed  bool
	Reason   string
	Required []Permission
	Granted  []Permission
}

func (pc *PermissionChecker) Evaluate(ctx context.Context, req *PermissionRequest) *PermissionResult {
	result := &PermissionResult{
		Required: []Permission{
			{
				Resource:   req.Resource,
				Action:     req.Action,
				ResourceID: req.ResourceID,
			},
		},
	}

	permissions := pc.policy.GetUserPermissions(ctx, req.UserID)
	result.Granted = permissions

	allowed := false
	for _, perm := range permissions {
		if pc.policy.matchesPermission(perm, result.Required[0]) {
			allowed = true
			break
		}
	}

	result.Allowed = allowed
	if !allowed {
		result.Reason = "User does not have the required permission"
	}

	return result
}

func (pc *PermissionChecker) EvaluateBatch(ctx context.Context, reqs []*PermissionRequest) []*PermissionResult {
	results := make([]*PermissionResult, len(reqs))

	for i, req := range reqs {
		results[i] = pc.Evaluate(ctx, req)
	}

	return results
}

type PermissionFilter struct {
	Resource   ResourceType
	Action     Action
	ResourceID string
}

func (pc *PermissionChecker) FilterPermissions(permissions []Permission, filter *PermissionFilter) []Permission {
	filtered := make([]Permission, 0)

	for _, perm := range permissions {
		if filter.Resource != "" && perm.Resource != filter.Resource {
			continue
		}

		if filter.Action != "" && perm.Action != filter.Action && perm.Action != "*" {
			continue
		}

		if filter.ResourceID != "" && perm.ResourceID != filter.ResourceID && perm.ResourceID != "*" {
			continue
		}

		filtered = append(filtered, perm)
	}

	return filtered
}

func (pc *PermissionChecker) CanAccessAllResources(ctx context.Context, userID string, resource ResourceType, action Action) bool {
	wildcardPerm := Permission{
		Resource:   resource,
		Action:     action,
		ResourceID: "*",
	}

	if pc.policy.HasPermission(ctx, userID, wildcardPerm) {
		return true
	}

	genericPerm := Permission{
		Resource:   resource,
		Action:     action,
		ResourceID: "",
	}

	if pc.policy.HasPermission(ctx, userID, genericPerm) {
		return true
	}

	return false
}
