package rbac

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

var (
	ErrRoleNotFound    = fmt.Errorf("role not found")
	ErrPermissionError = fmt.Errorf("permission denied")
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleUser     Role = "user"
	RoleReadonly Role = "readonly"
)

type Action string

const (
	ActionCreate  Action = "create"
	ActionRead    Action = "read"
	ActionUpdate  Action = "update"
	ActionDelete  Action = "delete"
	ActionList    Action = "list"
	ActionExecute Action = "execute"
)

type ResourceType string

const (
	ResourceAgent  ResourceType = "agents"
	ResourceTask   ResourceType = "tasks"
	ResourceSkill  ResourceType = "skills"
	ResourceConfig ResourceType = "config"
	ResourceSystem ResourceType = "system"
)

type Permission struct {
	Resource   ResourceType
	Action     Action
	ResourceID string
}

func (p Permission) String() string {
	if p.ResourceID != "" {
		return fmt.Sprintf("%s:%s:%s", p.Resource, p.Action, p.ResourceID)
	}
	return fmt.Sprintf("%s:%s", p.Resource, p.Action)
}

func ParsePermission(perm string) (*Permission, error) {
	parts := strings.Split(perm, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid permission format: %s", perm)
	}

	p := &Permission{
		Resource: ResourceType(parts[0]),
		Action:   Action(parts[1]),
	}

	if len(parts) > 2 {
		p.ResourceID = parts[2]
	}

	return p, nil
}

type Policy struct {
	mu          sync.RWMutex
	roles       map[Role]*RoleDefinition
	permissions map[string][]Permission
	userRoles   map[string][]Role
}

type RoleDefinition struct {
	Name        Role
	Permissions []Permission
	Description string
}

type PolicyConfig struct {
	RoleDefinitions []*RoleDefinition
	UserRoles       map[string][]Role
}

func NewPolicy(config *PolicyConfig) *Policy {
	p := &Policy{
		roles:       make(map[Role]*RoleDefinition),
		permissions: make(map[string][]Permission),
		userRoles:   make(map[string][]Role),
	}

	if config != nil {
		for _, roleDef := range config.RoleDefinitions {
			p.roles[roleDef.Name] = roleDef
		}

		for userID, roles := range config.UserRoles {
			p.userRoles[userID] = roles
		}
	}

	return p
}

func DefaultPolicyConfig() *PolicyConfig {
	return &PolicyConfig{
		RoleDefinitions: []*RoleDefinition{
			{
				Name: RoleAdmin,
				Permissions: []Permission{
					{Resource: ResourceAgent, Action: ActionCreate},
					{Resource: ResourceAgent, Action: ActionRead},
					{Resource: ResourceAgent, Action: ActionUpdate},
					{Resource: ResourceAgent, Action: ActionDelete},
					{Resource: ResourceAgent, Action: ActionList},
					{Resource: ResourceAgent, Action: ActionExecute},
					{Resource: ResourceTask, Action: ActionCreate},
					{Resource: ResourceTask, Action: ActionRead},
					{Resource: ResourceTask, Action: ActionUpdate},
					{Resource: ResourceTask, Action: ActionDelete},
					{Resource: ResourceTask, Action: ActionList},
					{Resource: ResourceTask, Action: ActionExecute},
					{Resource: ResourceSkill, Action: ActionCreate},
					{Resource: ResourceSkill, Action: ActionRead},
					{Resource: ResourceSkill, Action: ActionUpdate},
					{Resource: ResourceSkill, Action: ActionDelete},
					{Resource: ResourceSkill, Action: ActionList},
					{Resource: ResourceConfig, Action: ActionRead},
					{Resource: ResourceConfig, Action: ActionUpdate},
					{Resource: ResourceSystem, Action: ActionRead},
					{Resource: ResourceSystem, Action: ActionUpdate},
				},
				Description: "Administrator with full access",
			},
			{
				Name: RoleUser,
				Permissions: []Permission{
					{Resource: ResourceAgent, Action: ActionCreate},
					{Resource: ResourceAgent, Action: ActionRead},
					{Resource: ResourceAgent, Action: ActionUpdate},
					{Resource: ResourceAgent, Action: ActionList},
					{Resource: ResourceAgent, Action: ActionExecute},
					{Resource: ResourceTask, Action: ActionCreate},
					{Resource: ResourceTask, Action: ActionRead},
					{Resource: ResourceTask, Action: ActionUpdate},
					{Resource: ResourceTask, Action: ActionList},
					{Resource: ResourceTask, Action: ActionExecute},
					{Resource: ResourceSkill, Action: ActionRead},
					{Resource: ResourceSkill, Action: ActionList},
					{Resource: ResourceConfig, Action: ActionRead},
				},
				Description: "Standard user with limited access",
			},
			{
				Name: RoleReadonly,
				Permissions: []Permission{
					{Resource: ResourceAgent, Action: ActionRead},
					{Resource: ResourceAgent, Action: ActionList},
					{Resource: ResourceTask, Action: ActionRead},
					{Resource: ResourceTask, Action: ActionList},
					{Resource: ResourceSkill, Action: ActionRead},
					{Resource: ResourceSkill, Action: ActionList},
					{Resource: ResourceConfig, Action: ActionRead},
				},
				Description: "Read-only user",
			},
		},
		UserRoles: map[string][]Role{},
	}
}

func (p *Policy) AddRole(role *RoleDefinition) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.roles[role.Name] = role
	return nil
}

func (p *Policy) GetRole(name Role) (*RoleDefinition, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	role, ok := p.roles[name]
	if !ok {
		return nil, ErrRoleNotFound
	}
	return role, nil
}

func (p *Policy) AssignUserRole(userID string, roles []Role) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.userRoles[userID] = roles
}

func (p *Policy) GetUserRoles(userID string) []Role {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if roles, ok := p.userRoles[userID]; ok {
		return append([]Role{}, roles...)
	}
	return []Role{}
}

func (p *Policy) GetUserPermissions(ctx context.Context, userID string) []Permission {
	p.mu.RLock()
	defer p.mu.RUnlock()

	roles := p.userRoles[userID]
	permissions := make([]Permission, 0)
	seen := make(map[string]bool)

	for _, roleName := range roles {
		role, ok := p.roles[roleName]
		if !ok {
			continue
		}

		for _, perm := range role.Permissions {
			key := perm.String()
			if !seen[key] {
				seen[key] = true
				permissions = append(permissions, perm)
			}
		}
	}

	if userPerms, ok := p.permissions[userID]; ok {
		for _, perm := range userPerms {
			key := perm.String()
			if !seen[key] {
				seen[key] = true
				permissions = append(permissions, perm)
			}
		}
	}

	return permissions
}

func (p *Policy) HasPermission(ctx context.Context, userID string, perm Permission) bool {
	permissions := p.GetUserPermissions(ctx, userID)

	for _, userPerm := range permissions {
		if p.matchesPermission(userPerm, perm) {
			return true
		}
	}

	return false
}

func (p *Policy) HasAnyPermission(ctx context.Context, userID string, perms []Permission) bool {
	for _, perm := range perms {
		if p.HasPermission(ctx, userID, perm) {
			return true
		}
	}
	return false
}

func (p *Policy) HasAllPermissions(ctx context.Context, userID string, perms []Permission) bool {
	for _, perm := range perms {
		if !p.HasPermission(ctx, userID, perm) {
			return false
		}
	}
	return true
}

func (p *Policy) matchesPermission(available, required Permission) bool {
	if available.Resource != required.Resource {
		return false
	}

	if available.Action != required.Action && available.Action != "*" {
		return false
	}

	if available.ResourceID != "" {
		if available.ResourceID == required.ResourceID {
			return true
		}
		if available.ResourceID == "*" {
			return true
		}
		return false
	}

	return true
}

func (p *Policy) CheckPermission(ctx context.Context, userID string, perm Permission) error {
	if p.HasPermission(ctx, userID, perm) {
		return nil
	}
	return ErrPermissionError
}

func (p *Policy) AddUserPermission(userID string, perm Permission) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.permissions[userID]; !ok {
		p.permissions[userID] = []Permission{}
	}

	p.permissions[userID] = append(p.permissions[userID], perm)
}

func (p *Policy) RemoveUserPermission(userID string, perm Permission) {
	p.mu.Lock()
	defer p.mu.Unlock()

	perms, ok := p.permissions[userID]
	if !ok {
		return
	}

	for i, existingPerm := range perms {
		if existingPerm.String() == perm.String() {
			p.permissions[userID] = append(perms[:i], perms[i+1:]...)
			break
		}
	}
}
