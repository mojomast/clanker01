package rbac

import (
	"context"
	"testing"
)

func TestNewPermissionChecker(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)

	if checker == nil {
		t.Fatal("Expected checker to be created")
	}

	if checker.Policy() != policy {
		t.Error("Expected checker to have the provided policy")
	}
}

func TestCheck(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	ctx := context.Background()

	err := checker.Check(ctx, userID, ResourceAgent, ActionCreate, "")
	if err != nil {
		t.Errorf("Expected no error for valid permission, got %v", err)
	}

	err = checker.Check(ctx, userID, ResourceAgent, ActionDelete, "")
	if err != ErrPermissionError {
		t.Errorf("Expected ErrPermissionError, got %v", err)
	}
}

func TestCheckWildcard(t *testing.T) {
	policy := NewPolicy(nil)
	checker := NewPermissionChecker(policy)

	roleDef := &RoleDefinition{
		Name: Role("test"),
		Permissions: []Permission{
			{Resource: ResourceAgent, Action: ActionRead, ResourceID: "*"},
		},
	}

	policy.AddRole(roleDef)
	policy.AssignUserRole("user123", []Role{Role("test")})

	ctx := context.Background()

	err := checker.CheckWildcard(ctx, "user123", ResourceAgent, ActionRead)
	if err != nil {
		t.Errorf("Expected no error with wildcard permission, got %v", err)
	}
}

func TestCheckMultiple(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	ctx := context.Background()

	perms := []Permission{
		{Resource: ResourceAgent, Action: ActionCreate},
		{Resource: ResourceAgent, Action: ActionRead},
		{Resource: ResourceTask, Action: ActionCreate},
	}

	err := checker.CheckMultiple(ctx, userID, perms)
	if err != nil {
		t.Errorf("Expected no error for valid permissions, got %v", err)
	}

	permsInvalid := []Permission{
		{Resource: ResourceAgent, Action: ActionCreate},
		{Resource: ResourceAgent, Action: ActionDelete},
	}

	err = checker.CheckMultiple(ctx, userID, permsInvalid)
	if err == nil {
		t.Error("Expected error for invalid permissions, got nil")
	}
}

func TestCheckAny(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	ctx := context.Background()

	perms := []Permission{
		{Resource: ResourceAgent, Action: ActionDelete},
		{Resource: ResourceAgent, Action: ActionRead},
	}

	err := checker.CheckAny(ctx, userID, perms)
	if err != nil {
		t.Errorf("Expected no error when user has at least one permission, got %v", err)
	}

	permsNone := []Permission{
		{Resource: ResourceAgent, Action: ActionDelete},
		{Resource: ResourceConfig, Action: ActionUpdate},
	}

	err = checker.CheckAny(ctx, userID, permsNone)
	if err == nil {
		t.Error("Expected error when user has none of the permissions, got nil")
	}
}

func TestCheckAll(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	ctx := context.Background()

	perms := []Permission{
		{Resource: ResourceAgent, Action: ActionCreate},
		{Resource: ResourceAgent, Action: ActionRead},
	}

	err := checker.CheckAll(ctx, userID, perms)
	if err != nil {
		t.Errorf("Expected no error when user has all permissions, got %v", err)
	}

	permsMissing := []Permission{
		{Resource: ResourceAgent, Action: ActionCreate},
		{Resource: ResourceAgent, Action: ActionDelete},
	}

	err = checker.CheckAll(ctx, userID, permsMissing)
	if err == nil {
		t.Error("Expected error when user is missing one of the permissions, got nil")
	}
}

func TestHasAccessToResource(t *testing.T) {
	policy := NewPolicy(nil)
	checker := NewPermissionChecker(policy)

	roleDef := &RoleDefinition{
		Name: Role("test"),
		Permissions: []Permission{
			{Resource: ResourceAgent, Action: ActionRead, ResourceID: "agent1"},
		},
	}

	policy.AddRole(roleDef)
	policy.AssignUserRole("user123", []Role{Role("test")})

	ctx := context.Background()

	if !checker.HasAccessToResource(ctx, "user123", ResourceAgent, ActionRead, "agent1") {
		t.Error("Expected user to have access to agent1")
	}

	if checker.HasAccessToResource(ctx, "user123", ResourceAgent, ActionRead, "agent2") {
		t.Error("Expected user not to have access to agent2")
	}
}

func TestHasWildcardAccess(t *testing.T) {
	policy := NewPolicy(nil)
	checker := NewPermissionChecker(policy)

	roleDef := &RoleDefinition{
		Name: Role("test"),
		Permissions: []Permission{
			{Resource: ResourceAgent, Action: ActionRead, ResourceID: "*"},
		},
	}

	policy.AddRole(roleDef)
	policy.AssignUserRole("user123", []Role{Role("test")})

	ctx := context.Background()

	if !checker.HasWildcardAccess(ctx, "user123", ResourceAgent, ActionRead) {
		t.Error("Expected user to have wildcard access to agents")
	}

	if checker.HasWildcardAccess(ctx, "user123", ResourceAgent, ActionDelete) {
		t.Error("Expected user not to have wildcard access to delete agents")
	}
}

func TestGetAccessibleResources(t *testing.T) {
	policy := NewPolicy(nil)
	checker := NewPermissionChecker(policy)

	roleDef := &RoleDefinition{
		Name: Role("test"),
		Permissions: []Permission{
			{Resource: ResourceAgent, Action: ActionRead, ResourceID: "agent1"},
			{Resource: ResourceAgent, Action: ActionRead, ResourceID: "agent2"},
			{Resource: ResourceAgent, Action: ActionRead, ResourceID: "agent3"},
		},
	}

	policy.AddRole(roleDef)
	policy.AssignUserRole("user123", []Role{Role("test")})

	ctx := context.Background()

	accessible := checker.GetAccessibleResources(ctx, "user123", ResourceAgent, ActionRead)

	if len(accessible) != 3 {
		t.Errorf("Expected 3 accessible resources, got %d", len(accessible))
	}

	resourceMap := make(map[string]bool)
	for _, id := range accessible {
		resourceMap[id] = true
	}

	if !resourceMap["agent1"] || !resourceMap["agent2"] || !resourceMap["agent3"] {
		t.Error("Expected to find agent1, agent2, and agent3 in accessible resources")
	}
}

func TestGetAccessibleResourcesWithWildcard(t *testing.T) {
	policy := NewPolicy(nil)
	checker := NewPermissionChecker(policy)

	roleDef := &RoleDefinition{
		Name: Role("test"),
		Permissions: []Permission{
			{Resource: ResourceAgent, Action: ActionRead, ResourceID: "*"},
		},
	}

	policy.AddRole(roleDef)
	policy.AssignUserRole("user123", []Role{Role("test")})

	ctx := context.Background()

	accessible := checker.GetAccessibleResources(ctx, "user123", ResourceAgent, ActionRead)

	if len(accessible) != 1 || accessible[0] != "*" {
		t.Errorf("Expected wildcard '*', got %v", accessible)
	}
}

func TestGetUserPermissionsChecker(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleAdmin})

	ctx := context.Background()

	perms := checker.GetUserPermissions(ctx, userID)

	if len(perms) == 0 {
		t.Fatal("Expected admin to have permissions")
	}

	hasAgentRead := false
	for _, perm := range perms {
		if perm.Resource == ResourceAgent && perm.Action == ActionRead {
			hasAgentRead = true
			break
		}
	}

	if !hasAgentRead {
		t.Error("Expected admin to have agent:read permission")
	}
}

func TestGetUserRoles(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)

	userID := "user123"
	roles := []Role{RoleUser, RoleReadonly}
	policy.AssignUserRole(userID, roles)

	ctx := context.Background()

	userRoles := checker.GetUserRoles(ctx, userID)

	if len(userRoles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(userRoles))
	}

	roleMap := make(map[Role]bool)
	for _, role := range userRoles {
		roleMap[role] = true
	}

	if !roleMap[RoleUser] || !roleMap[RoleReadonly] {
		t.Error("Expected to find user and readonly roles")
	}
}

func TestValidatePermissionString(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)

	tests := []struct {
		perm  string
		valid bool
	}{
		{"agents:read", true},
		{"tasks:create", true},
		{"skills:update", true},
		{"config:delete", true},
		{"agents:read:agent1", true},
		{"tasks:update:task123", true},
		{"invalid:read", false},
		{"agents:invalid", false},
		{"agents", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.perm, func(t *testing.T) {
			err := checker.ValidatePermissionString(tt.perm)
			if tt.valid && err != nil {
				t.Errorf("Expected %s to be valid, got error: %v", tt.perm, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("Expected %s to be invalid, got no error", tt.perm)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	ctx := context.Background()

	req := &PermissionRequest{
		UserID:   userID,
		Resource: ResourceAgent,
		Action:   ActionCreate,
	}

	result := checker.Evaluate(ctx, req)

	if !result.Allowed {
		t.Error("Expected permission to be allowed")
	}

	if result.Reason != "" {
		t.Errorf("Expected no reason for allowed permission, got: %s", result.Reason)
	}

	if len(result.Required) != 1 {
		t.Errorf("Expected 1 required permission, got %d", len(result.Required))
	}
}

func TestEvaluateDenied(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	ctx := context.Background()

	req := &PermissionRequest{
		UserID:   userID,
		Resource: ResourceAgent,
		Action:   ActionDelete,
	}

	result := checker.Evaluate(ctx, req)

	if result.Allowed {
		t.Error("Expected permission to be denied")
	}

	if result.Reason == "" {
		t.Error("Expected reason for denied permission")
	}
}

func TestEvaluateBatch(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	ctx := context.Background()

	reqs := []*PermissionRequest{
		{
			UserID:   userID,
			Resource: ResourceAgent,
			Action:   ActionCreate,
		},
		{
			UserID:   userID,
			Resource: ResourceAgent,
			Action:   ActionRead,
		},
		{
			UserID:   userID,
			Resource: ResourceAgent,
			Action:   ActionDelete,
		},
	}

	results := checker.EvaluateBatch(ctx, reqs)

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	if !results[0].Allowed {
		t.Error("Expected first permission to be allowed")
	}

	if !results[1].Allowed {
		t.Error("Expected second permission to be allowed")
	}

	if results[2].Allowed {
		t.Error("Expected third permission to be denied")
	}
}

func TestFilterPermissions(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)

	perms := []Permission{
		{Resource: ResourceAgent, Action: ActionRead},
		{Resource: ResourceAgent, Action: ActionCreate},
		{Resource: ResourceTask, Action: ActionRead},
		{Resource: ResourceTask, Action: ActionCreate},
		{Resource: ResourceSkill, Action: ActionRead},
	}

	filter := &PermissionFilter{
		Resource: ResourceAgent,
	}

	filtered := checker.FilterPermissions(perms, filter)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered permissions, got %d", len(filtered))
	}

	for _, perm := range filtered {
		if perm.Resource != ResourceAgent {
			t.Errorf("Expected all filtered permissions to be for agents, got %s", perm.Resource)
		}
	}
}

func TestFilterPermissionsWithAction(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)

	perms := []Permission{
		{Resource: ResourceAgent, Action: ActionRead},
		{Resource: ResourceAgent, Action: ActionCreate},
		{Resource: ResourceTask, Action: ActionRead},
	}

	filter := &PermissionFilter{
		Resource: ResourceAgent,
		Action:   ActionRead,
	}

	filtered := checker.FilterPermissions(perms, filter)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 filtered permission, got %d", len(filtered))
	}

	if filtered[0].Action != ActionRead {
		t.Errorf("Expected filtered permission to be read action, got %s", filtered[0].Action)
	}
}

func TestCanAccessAllResources(t *testing.T) {
	policy := NewPolicy(nil)
	checker := NewPermissionChecker(policy)

	roleDef := &RoleDefinition{
		Name: Role("test"),
		Permissions: []Permission{
			{Resource: ResourceAgent, Action: ActionRead, ResourceID: "*"},
		},
	}

	policy.AddRole(roleDef)
	policy.AssignUserRole("user123", []Role{Role("test")})

	ctx := context.Background()

	if !checker.CanAccessAllResources(ctx, "user123", ResourceAgent, ActionRead) {
		t.Error("Expected user to be able to access all agents")
	}

	if checker.CanAccessAllResources(ctx, "user123", ResourceAgent, ActionDelete) {
		t.Error("Expected user not to be able to delete all agents")
	}
}

func TestCanAccessAllResourcesWithGenericPermission(t *testing.T) {
	policy := NewPolicy(nil)
	checker := NewPermissionChecker(policy)

	roleDef := &RoleDefinition{
		Name: Role("test"),
		Permissions: []Permission{
			{Resource: ResourceAgent, Action: ActionRead},
		},
	}

	policy.AddRole(roleDef)
	policy.AssignUserRole("user123", []Role{Role("test")})

	ctx := context.Background()

	if !checker.CanAccessAllResources(ctx, "user123", ResourceAgent, ActionRead) {
		t.Error("Expected user to be able to access all agents with generic permission")
	}
}
