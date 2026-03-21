package rbac

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/swarm-ai/swarm/internal/security/auth"
)

func TestNewRBACMiddleware(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)

	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	if middleware == nil {
		t.Fatal("Expected middleware to be created")
	}

	if middleware.Checker() != checker {
		t.Error("Expected middleware to have the provided checker")
	}
}

func TestGetPermissionFromPath(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)
	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	tests := []struct {
		method     string
		path       string
		resource   ResourceType
		action     Action
		resourceID string
	}{
		{
			method:     "GET",
			path:       "/agents",
			resource:   ResourceAgent,
			action:     ActionRead,
			resourceID: "",
		},
		{
			method:     "POST",
			path:       "/tasks/task123",
			resource:   ResourceTask,
			action:     ActionCreate,
			resourceID: "task123",
		},
		{
			method:     "PUT",
			path:       "/agents/agent456",
			resource:   ResourceAgent,
			action:     ActionUpdate,
			resourceID: "agent456",
		},
		{
			method:     "DELETE",
			path:       "/skills/skill789",
			resource:   ResourceSkill,
			action:     ActionDelete,
			resourceID: "skill789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			perm := middleware.getPermissionFromPath(tt.method, tt.path)

			if perm == nil {
				t.Fatal("Expected permission to be parsed")
			}

			if perm.Resource != tt.resource {
				t.Errorf("Expected resource %s, got %s", tt.resource, perm.Resource)
			}

			if perm.Action != tt.action {
				t.Errorf("Expected action %s, got %s", tt.action, perm.Action)
			}

			if perm.ResourceID != tt.resourceID {
				t.Errorf("Expected resourceID %s, got %s", tt.resourceID, perm.ResourceID)
			}
		})
	}
}

func TestHTTPMiddleware(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)
	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	requiredPerms := []Permission{
		{Resource: ResourceAgent, Action: ActionCreate},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.HTTPMiddleware(requiredPerms, false)(handler)

	req := httptest.NewRequest("POST", "/agents", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, &auth.User{
		ID: userID,
	}))
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHTTPMiddlewareDenied(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)
	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleReadonly})

	requiredPerms := []Permission{
		{Resource: ResourceAgent, Action: ActionDelete},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.HTTPMiddleware(requiredPerms, false)(handler)

	req := httptest.NewRequest("DELETE", "/agents/agent1", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, &auth.User{
		ID: userID,
	}))
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestHTTPMiddlewareUnauthorized(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)
	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	requiredPerms := []Permission{
		{Resource: ResourceAgent, Action: ActionCreate},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.HTTPMiddleware(requiredPerms, false)(handler)

	req := httptest.NewRequest("POST", "/agents", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestHTTPMiddlewareWithResource(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)
	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.HTTPMiddlewareWithResource()(handler)

	req := httptest.NewRequest("GET", "/agents", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, &auth.User{
		ID: userID,
	}))
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHTTPMiddlewareWithResourceDenied(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)
	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.HTTPMiddlewareWithResource()(handler)

	req := httptest.NewRequest("DELETE", "/agents/agent1", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, &auth.User{
		ID: userID,
	}))
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestHTTPMiddlewareForResource(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)
	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.HTTPMiddlewareForResource(ResourceAgent, []Action{ActionCreate, ActionRead, ActionUpdate})(handler)

	req := httptest.NewRequest("POST", "/agents", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, &auth.User{
		ID: userID,
	}))
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHTTPMiddlewareForResourceMethodNotAllowed(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)
	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.HTTPMiddlewareForResource(ResourceAgent, []Action{ActionCreate, ActionRead})(handler)

	req := httptest.NewRequest("DELETE", "/agents/agent1", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, &auth.User{
		ID: userID,
	}))
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestRequirementsFromHTTPRequest(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)
	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	tests := []struct {
		name          string
		method        string
		path          string
		expectedCount int
		resource      ResourceType
		action        Action
		resourceID    string
	}{
		{
			name:          "simple path",
			method:        "GET",
			path:          "/agents",
			expectedCount: 1,
			resource:      ResourceAgent,
			action:        ActionRead,
			resourceID:    "",
		},
		{
			name:          "path with ID",
			method:        "POST",
			path:          "/tasks/task123",
			expectedCount: 1,
			resource:      ResourceTask,
			action:        ActionCreate,
			resourceID:    "task123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			requirements := middleware.RequirementsFromHTTPRequest(req)

			if len(requirements) != tt.expectedCount {
				t.Errorf("Expected %d requirements, got %d", tt.expectedCount, len(requirements))
			}

			if len(requirements) > 0 {
				if requirements[0].Resource != tt.resource {
					t.Errorf("Expected resource %s, got %s", tt.resource, requirements[0].Resource)
				}
				if requirements[0].Action != tt.action {
					t.Errorf("Expected action %s, got %s", tt.action, requirements[0].Action)
				}
				if requirements[0].ResourceID != tt.resourceID {
					t.Errorf("Expected resourceID %s, got %s", tt.resourceID, requirements[0].ResourceID)
				}
			}
		})
	}
}

func TestCheckHTTPRequestPermissions(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)
	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	user := &auth.User{
		ID: userID,
	}

	tests := []struct {
		name    string
		method  string
		path    string
		allowed bool
	}{
		{
			name:    "user can create agents",
			method:  "POST",
			path:    "/agents",
			allowed: true,
		},
		{
			name:    "user cannot delete agents",
			method:  "DELETE",
			path:    "/agents/agent1",
			allowed: false,
		},
		{
			name:    "user can read tasks",
			method:  "GET",
			path:    "/tasks",
			allowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			allowed := middleware.CheckHTTPRequestPermissions(req, user)

			if allowed != tt.allowed {
				t.Errorf("Expected allowed=%v, got %v", tt.allowed, allowed)
			}
		})
	}
}

func TestWebSocketMiddleware(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)
	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	requiredPerms := []Permission{
		{Resource: ResourceAgent, Action: ActionRead},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.WebSocketMiddleware(requiredPerms)(handler)

	req := httptest.NewRequest("GET", "/ws", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, &auth.User{
		ID: userID,
	}))
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestWebSocketMiddlewareDenied(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)
	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleReadonly})

	requiredPerms := []Permission{
		{Resource: ResourceAgent, Action: ActionDelete},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.WebSocketMiddleware(requiredPerms)(handler)

	req := httptest.NewRequest("GET", "/ws", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, &auth.User{
		ID: userID,
	}))
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestWebSocketConnectInterceptor(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)
	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	checkerFunc := func(ctx context.Context, uid string) bool {
		perm := Permission{Resource: ResourceAgent, Action: ActionRead}
		return checker.Policy().HasPermission(ctx, uid, perm)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.WebSocketConnectInterceptor(checkerFunc)(handler)

	req := httptest.NewRequest("GET", "/ws", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, &auth.User{
		ID: userID,
	}))
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHTTPMiddlewareRequireAll(t *testing.T) {
	config := DefaultPolicyConfig()
	policy := NewPolicy(config)
	checker := NewPermissionChecker(policy)
	authMW := &auth.AuthMiddleware{}
	middleware := NewRBACMiddleware(checker, authMW)

	userID := "user123"
	policy.AssignUserRole(userID, []Role{RoleUser})

	requiredPerms := []Permission{
		{Resource: ResourceAgent, Action: ActionCreate},
		{Resource: ResourceAgent, Action: ActionRead},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.HTTPMiddleware(requiredPerms, true)(handler)

	req := httptest.NewRequest("POST", "/agents", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, &auth.User{
		ID: userID,
	}))
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	requiredPermsMissing := []Permission{
		{Resource: ResourceAgent, Action: ActionCreate},
		{Resource: ResourceAgent, Action: ActionDelete},
	}

	wrapped = middleware.HTTPMiddleware(requiredPermsMissing, true)(handler)

	req = httptest.NewRequest("POST", "/agents", nil)
	req = req.WithContext(context.WithValue(req.Context(), auth.UserContextKey, &auth.User{
		ID: userID,
	}))
	rr = httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
}
