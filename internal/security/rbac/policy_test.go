package rbac

import (
	"context"
	"testing"
)

func TestNewPolicy(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)

	if policy == nil {
		t.Fatal("Expected policy to be created")
	}

	if len(policy.roles) != 3 {
		t.Errorf("Expected 3 roles, got %d", len(policy.roles))
	}
}

func TestNewPolicyWithNilConfig(t *testing.T) {
	policy := NewPolicy(nil)

	if policy == nil {
		t.Fatal("Expected policy to be created with nil config")
	}

	if len(policy.roles) != 0 {
		t.Errorf("Expected 0 roles with nil config, got %d", len(policy.roles))
	}
}

func TestAddRole(t *testing.T) {
	policy := NewPolicy(nil)

	roleDef := &RoleDefinition{
		Name: Role("custom"),
		Permissions: []Permission{
			{Resource: ResourceAgent, Action: ActionRead},
		},
		Description: "Custom role",
	}

	err := policy.AddRole(roleDef)
	if err != nil {
		t.Errorf("Expected no error adding role, got %v", err)
	}

	role, err := policy.GetRole(Role("custom"))
	if err != nil {
		t.Errorf("Expected to find role, got error: %v", err)
	}

	if role.Name != Role("custom") {
		t.Errorf("Expected role name 'custom', got %s", role.Name)
	}
}

func TestGetRole(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)

	role, err := policy.GetRole(RoleAdmin)
	if err != nil {
		t.Errorf("Expected to find admin role, got error: %v", err)
	}

	if role.Name != RoleAdmin {
		t.Errorf("Expected role name '%s', got %s", RoleAdmin, role.Name)
	}

	_, err = policy.GetRole(Role("nonexistent"))
	if err != ErrRoleNotFound {
		t.Errorf("Expected ErrRoleNotFound, got %v", err)
	}
}

func TestAssignUserRole(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)

	userID := "user123"
	roles := []Role{RoleUser, RoleReadonly}

	policy.AssignUserRole(userID, roles)

	userRoles := policy.GetUserRoles(userID)
	if len(userRoles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(userRoles))
	}
}

func TestGetUserPermissions(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleAdmin})

	ctx := context.Background()
	permissions := policy.GetUserPermissions(ctx, userID)

	if len(permissions) == 0 {
		t.Fatal("Expected admin to have permissions")
	}

	expectedPerms := []struct {
		resource ResourceType
		action   Action
	}{
		{ResourceAgent, ActionCreate},
		{ResourceTask, ActionRead},
		{ResourceConfig, ActionUpdate},
	}

	for _, expected := range expectedPerms {
		found := false
		for _, perm := range permissions {
			if perm.Resource == expected.resource && perm.Action == expected.action {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find permission %s:%s", expected.resource, expected.action)
		}
	}
}

func TestHasPermission(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)

	adminID := "admin123"
	userID := "user123"
	readonlyID := "readonly123"

	policy.AssignUserRole(adminID, []Role{RoleAdmin})
	policy.AssignUserRole(userID, []Role{RoleUser})
	policy.AssignUserRole(readonlyID, []Role{RoleReadonly})

	ctx := context.Background()

	tests := []struct {
		name     string
		userID   string
		resource ResourceType
		action   Action
		allowed  bool
	}{
		{
			name:     "admin can delete agents",
			userID:   adminID,
			resource: ResourceAgent,
			action:   ActionDelete,
			allowed:  true,
		},
		{
			name:     "user cannot delete agents",
			userID:   userID,
			resource: ResourceAgent,
			action:   ActionDelete,
			allowed:  false,
		},
		{
			name:     "readonly cannot create agents",
			userID:   readonlyID,
			resource: ResourceAgent,
			action:   ActionCreate,
			allowed:  false,
		},
		{
			name:     "readonly can read agents",
			userID:   readonlyID,
			resource: ResourceAgent,
			action:   ActionRead,
			allowed:  true,
		},
		{
			name:     "user can create tasks",
			userID:   userID,
			resource: ResourceTask,
			action:   ActionCreate,
			allowed:  true,
		},
		{
			name:     "user cannot update config",
			userID:   userID,
			resource: ResourceConfig,
			action:   ActionUpdate,
			allowed:  false,
		},
		{
			name:     "admin can update config",
			userID:   adminID,
			resource: ResourceConfig,
			action:   ActionUpdate,
			allowed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm := Permission{
				Resource: tt.resource,
				Action:   tt.action,
			}

			allowed := policy.HasPermission(ctx, tt.userID, perm)
			if allowed != tt.allowed {
				t.Errorf("Expected allowed=%v, got %v", tt.allowed, allowed)
			}
		})
	}
}

func TestHasPermissionWithResourceID(t *testing.T) {
	policy := NewPolicy(nil)

	roleDef := &RoleDefinition{
		Name: Role("test"),
		Permissions: []Permission{
			{Resource: ResourceAgent, Action: ActionRead, ResourceID: "agent1"},
			{Resource: ResourceAgent, Action: ActionUpdate, ResourceID: "*"},
		},
	}

	policy.AddRole(roleDef)
	policy.AssignUserRole("user123", []Role{Role("test")})

	ctx := context.Background()

	tests := []struct {
		name       string
		resource   ResourceType
		action     Action
		resourceID string
		allowed    bool
	}{
		{
			name:       "can read agent1",
			resource:   ResourceAgent,
			action:     ActionRead,
			resourceID: "agent1",
			allowed:    true,
		},
		{
			name:       "cannot read agent2",
			resource:   ResourceAgent,
			action:     ActionRead,
			resourceID: "agent2",
			allowed:    false,
		},
		{
			name:       "can update any agent (wildcard)",
			resource:   ResourceAgent,
			action:     ActionUpdate,
			resourceID: "agent999",
			allowed:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm := Permission{
				Resource:   tt.resource,
				Action:     tt.action,
				ResourceID: tt.resourceID,
			}

			allowed := policy.HasPermission(ctx, "user123", perm)
			if allowed != tt.allowed {
				t.Errorf("Expected allowed=%v, got %v", tt.allowed, allowed)
			}
		})
	}
}

func TestHasAnyPermission(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	ctx := context.Background()

	perms := []Permission{
		{Resource: ResourceAgent, Action: ActionDelete},
		{Resource: ResourceAgent, Action: ActionRead},
	}

	allowed := policy.HasAnyPermission(ctx, userID, perms)
	if !allowed {
		t.Error("Expected user to have at least one of the permissions")
	}
}

func TestHasAllPermissions(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)

	adminID := "admin123"
	userID := "user123"

	policy.AssignUserRole(adminID, []Role{RoleAdmin})
	policy.AssignUserRole(userID, []Role{RoleUser})

	ctx := context.Background()

	perms := []Permission{
		{Resource: ResourceAgent, Action: ActionCreate},
		{Resource: ResourceAgent, Action: ActionRead},
		{Resource: ResourceAgent, Action: ActionDelete},
	}

	if !policy.HasAllPermissions(ctx, adminID, perms) {
		t.Error("Expected admin to have all permissions")
	}

	if policy.HasAllPermissions(ctx, userID, perms) {
		t.Error("Expected user not to have all permissions")
	}
}

func TestCheckPermission(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	ctx := context.Background()

	perm := Permission{
		Resource: ResourceAgent,
		Action:   ActionCreate,
	}

	err := policy.CheckPermission(ctx, userID, perm)
	if err != nil {
		t.Errorf("Expected no error for valid permission, got %v", err)
	}

	permDelete := Permission{
		Resource: ResourceAgent,
		Action:   ActionDelete,
	}

	err = policy.CheckPermission(ctx, userID, permDelete)
	if err != ErrPermissionError {
		t.Errorf("Expected ErrPermissionError, got %v", err)
	}
}

func TestAddUserPermission(t *testing.T) {
	policy := NewPolicy(nil)

	userID := "user123"
	perm := Permission{
		Resource:   ResourceAgent,
		Action:     ActionRead,
		ResourceID: "agent1",
	}

	policy.AddUserPermission(userID, perm)

	perms := policy.GetUserPermissions(context.Background(), userID)
	if len(perms) != 1 {
		t.Errorf("Expected 1 permission, got %d", len(perms))
	}

	if perms[0].ResourceID != "agent1" {
		t.Errorf("Expected resourceID 'agent1', got %s", perms[0].ResourceID)
	}
}

func TestRemoveUserPermission(t *testing.T) {
	policy := NewPolicy(nil)

	userID := "user123"
	perm := Permission{
		Resource:   ResourceAgent,
		Action:     ActionRead,
		ResourceID: "agent1",
	}

	policy.AddUserPermission(userID, perm)
	policy.AddUserPermission(userID, Permission{
		Resource: ResourceTask,
		Action:   ActionCreate,
	})

	perms := policy.GetUserPermissions(context.Background(), userID)
	if len(perms) != 2 {
		t.Errorf("Expected 2 permissions before removal, got %d", len(perms))
	}

	policy.RemoveUserPermission(userID, perm)

	perms = policy.GetUserPermissions(context.Background(), userID)
	if len(perms) != 1 {
		t.Errorf("Expected 1 permission after removal, got %d", len(perms))
	}

	if perms[0].Resource != ResourceTask {
		t.Errorf("Expected remaining permission to be for tasks, got %s", perms[0].Resource)
	}
}

func TestPermissionString(t *testing.T) {
	tests := []struct {
		perm     Permission
		expected string
	}{
		{
			perm:     Permission{Resource: ResourceAgent, Action: ActionRead},
			expected: "agents:read",
		},
		{
			perm:     Permission{Resource: ResourceTask, Action: ActionUpdate, ResourceID: "task123"},
			expected: "tasks:update:task123",
		},
		{
			perm:     Permission{Resource: ResourceSkill, Action: ActionDelete, ResourceID: "skill456"},
			expected: "skills:delete:skill456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.perm.String() != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, tt.perm.String())
			}
		})
	}
}

func TestParsePermission(t *testing.T) {
	tests := []struct {
		input    string
		expected *Permission
		err      bool
	}{
		{
			input: "agents:read",
			expected: &Permission{
				Resource: ResourceAgent,
				Action:   ActionRead,
			},
			err: false,
		},
		{
			input: "tasks:update:task123",
			expected: &Permission{
				Resource:   ResourceTask,
				Action:     ActionUpdate,
				ResourceID: "task123",
			},
			err: false,
		},
		{
			input:    "invalid",
			expected: nil,
			err:      true,
		},
		{
			input:    "agents",
			expected: nil,
			err:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			perm, err := ParsePermission(tt.input)

			if tt.err && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tt.err && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if tt.expected != nil && perm != nil {
				if perm.Resource != tt.expected.Resource {
					t.Errorf("Expected resource %s, got %s", tt.expected.Resource, perm.Resource)
				}
				if perm.Action != tt.expected.Action {
					t.Errorf("Expected action %s, got %s", tt.expected.Action, perm.Action)
				}
				if perm.ResourceID != tt.expected.ResourceID {
					t.Errorf("Expected resourceID %s, got %s", tt.expected.ResourceID, perm.ResourceID)
				}
			}
		})
	}
}

func TestMatchesPermission(t *testing.T) {
	policy := NewPolicy(nil)

	tests := []struct {
		name      string
		available Permission
		required  Permission
		match     bool
	}{
		{
			name:      "exact match",
			available: Permission{Resource: ResourceAgent, Action: ActionRead},
			required:  Permission{Resource: ResourceAgent, Action: ActionRead},
			match:     true,
		},
		{
			name:      "wildcard action",
			available: Permission{Resource: ResourceAgent, Action: "*", ResourceID: ""},
			required:  Permission{Resource: ResourceAgent, Action: ActionRead},
			match:     true,
		},
		{
			name:      "wildcard resourceID",
			available: Permission{Resource: ResourceAgent, Action: ActionRead, ResourceID: "*"},
			required:  Permission{Resource: ResourceAgent, Action: ActionRead, ResourceID: "agent1"},
			match:     true,
		},
		{
			name:      "specific resourceID match",
			available: Permission{Resource: ResourceAgent, Action: ActionRead, ResourceID: "agent1"},
			required:  Permission{Resource: ResourceAgent, Action: ActionRead, ResourceID: "agent1"},
			match:     true,
		},
		{
			name:      "specific resourceID mismatch",
			available: Permission{Resource: ResourceAgent, Action: ActionRead, ResourceID: "agent1"},
			required:  Permission{Resource: ResourceAgent, Action: ActionRead, ResourceID: "agent2"},
			match:     false,
		},
		{
			name:      "resource mismatch",
			available: Permission{Resource: ResourceAgent, Action: ActionRead},
			required:  Permission{Resource: ResourceTask, Action: ActionRead},
			match:     false,
		},
		{
			name:      "action mismatch",
			available: Permission{Resource: ResourceAgent, Action: ActionRead},
			required:  Permission{Resource: ResourceAgent, Action: ActionUpdate},
			match:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := policy.matchesPermission(tt.available, tt.required)
			if match != tt.match {
				t.Errorf("Expected match=%v, got %v", tt.match, match)
			}
		})
	}
}

func TestDefaultRoles(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)

	adminRole, err := policy.GetRole(RoleAdmin)
	if err != nil {
		t.Errorf("Expected admin role to exist, got error: %v", err)
	}

	if adminRole.Description != "Administrator with full access" {
		t.Errorf("Expected admin description 'Administrator with full access', got %s", adminRole.Description)
	}

	userRole, err := policy.GetRole(RoleUser)
	if err != nil {
		t.Errorf("Expected user role to exist, got error: %v", err)
	}

	if userRole.Description != "Standard user with limited access" {
		t.Errorf("Expected user description 'Standard user with limited access', got %s", userRole.Description)
	}

	readonlyRole, err := policy.GetRole(RoleReadonly)
	if err != nil {
		t.Errorf("Expected readonly role to exist, got error: %v", err)
	}

	if readonlyRole.Description != "Read-only user" {
		t.Errorf("Expected readonly description 'Read-only user', got %s", readonlyRole.Description)
	}
}
