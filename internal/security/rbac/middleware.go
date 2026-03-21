package rbac

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	grpcAuth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/swarm-ai/swarm/internal/security/auth"
)

func mapHTTPMethodToAction(method string) Action {
	switch method {
	case "GET":
		return ActionRead
	case "POST":
		return ActionCreate
	case "PUT", "PATCH":
		return ActionUpdate
	case "DELETE":
		return ActionDelete
	default:
		return Action(method)
	}
}

type RBACMiddleware struct {
	checker *PermissionChecker
	auth    *auth.AuthMiddleware
}

func NewRBACMiddleware(checker *PermissionChecker, auth *auth.AuthMiddleware) *RBACMiddleware {
	return &RBACMiddleware{
		checker: checker,
		auth:    auth,
	}
}

func (m *RBACMiddleware) Checker() *PermissionChecker {
	return m.checker
}

func (m *RBACMiddleware) getPermissionFromPath(method, path string) *Permission {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) < 1 {
		return nil
	}

	resource := ResourceType(parts[0])
	action := mapHTTPMethodToAction(method)

	if len(parts) > 1 {
		resourceID := parts[1]
		return &Permission{
			Resource:   resource,
			Action:     action,
			ResourceID: resourceID,
		}
	}

	return &Permission{
		Resource: resource,
		Action:   action,
	}
}

func (m *RBACMiddleware) HTTPMiddleware(requiredPerms []Permission, requireAll bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			user, ok := auth.GetUserFromContext(ctx)
			if !ok || user == nil {
				m.writeError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			var allowed bool
			if requireAll {
				allowed = m.checker.Policy().HasAllPermissions(ctx, user.ID, requiredPerms)
			} else {
				allowed = m.checker.Policy().HasAnyPermission(ctx, user.ID, requiredPerms)
			}

			if !allowed {
				m.writeError(w, http.StatusForbidden, "Permission denied")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *RBACMiddleware) HTTPMiddlewareWithResource() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			user, ok := auth.GetUserFromContext(ctx)
			if !ok || user == nil {
				m.writeError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			perm := m.getPermissionFromPath(r.Method, r.URL.Path)
			if perm == nil {
				m.writeError(w, http.StatusBadRequest, "Invalid request path")
				return
			}

			if !m.checker.Policy().HasPermission(ctx, user.ID, *perm) {
				m.writeError(w, http.StatusForbidden, fmt.Sprintf("Permission denied for %s", perm.String()))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *RBACMiddleware) HTTPMiddlewareForResource(resource ResourceType, actions []Action) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			user, ok := auth.GetUserFromContext(ctx)
			if !ok || user == nil {
				m.writeError(w, http.StatusUnauthorized, "Authentication required")
				return
			}

			reqAction := mapHTTPMethodToAction(r.Method)
			actionAllowed := false
			for _, action := range actions {
				if action == reqAction || action == "*" {
					actionAllowed = true
					break
				}
			}

			if !actionAllowed {
				m.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
				return
			}

			perm := Permission{
				Resource: resource,
				Action:   reqAction,
			}

			if !m.checker.Policy().HasPermission(ctx, user.ID, perm) {
				m.writeError(w, http.StatusForbidden, fmt.Sprintf("Permission denied for %s", perm.String()))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *RBACMiddleware) gRPCAuthFunc(ctx context.Context) (context.Context, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "authentication required")
	}

	grpcUserID := userID{}
	newCtx := context.WithValue(ctx, grpcUserID, user.ID)

	return newCtx, nil
}

type userID struct{}

func (m *RBACMiddleware) GRPCUnaryInterceptor(requiredPerms []Permission, requireAll bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		uid, ok := ctx.Value(userID{}).(string)
		if !ok || uid == "" {
			return nil, status.Errorf(codes.Unauthenticated, "authentication required")
		}

		var allowed bool
		if requireAll {
			allowed = m.checker.Policy().HasAllPermissions(ctx, uid, requiredPerms)
		} else {
			allowed = m.checker.Policy().HasAnyPermission(ctx, uid, requiredPerms)
		}

		if !allowed {
			return nil, status.Errorf(codes.PermissionDenied, "permission denied")
		}

		return handler(ctx, req)
	}
}

func (m *RBACMiddleware) GRPCStreamInterceptor(requiredPerms []Permission, requireAll bool) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		uid, ok := ctx.Value(userID{}).(string)
		if !ok || uid == "" {
			return status.Errorf(codes.Unauthenticated, "authentication required")
		}

		var allowed bool
		if requireAll {
			allowed = m.checker.Policy().HasAllPermissions(ctx, uid, requiredPerms)
		} else {
			allowed = m.checker.Policy().HasAnyPermission(ctx, uid, requiredPerms)
		}

		if !allowed {
			return status.Errorf(codes.PermissionDenied, "permission denied")
		}

		return handler(srv, ss)
	}
}

func (m *RBACMiddleware) GRPCAuthUnaryInterceptor() grpc.UnaryServerInterceptor {
	return grpcAuth.UnaryServerInterceptor(m.gRPCAuthFunc)
}

func (m *RBACMiddleware) GRPCAuthStreamInterceptor() grpc.StreamServerInterceptor {
	return grpcAuth.StreamServerInterceptor(m.gRPCAuthFunc)
}

func (m *RBACMiddleware) WebSocketMiddleware(requiredPerms []Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			user, ok := auth.GetUserFromContext(ctx)
			if !ok || user == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			if !m.checker.Policy().HasAnyPermission(ctx, user.ID, requiredPerms) {
				http.Error(w, "Permission denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *RBACMiddleware) WebSocketConnectInterceptor(checker func(ctx context.Context, userID string) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			user, ok := auth.GetUserFromContext(ctx)
			if !ok || user == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			if !checker(ctx, user.ID) {
				http.Error(w, "Permission denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *RBACMiddleware) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errResp := struct {
		Error string `json:"error"`
	}{
		Error: message,
	}

	json.NewEncoder(w).Encode(errResp)
}

type PermissionRequirement struct {
	Resource   ResourceType
	Action     Action
	ResourceID string
}

func (m *RBACMiddleware) RequirementsFromHTTPRequest(r *http.Request) []PermissionRequirement {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	requirements := make([]PermissionRequirement, 0)

	if len(parts) < 1 {
		return requirements
	}

	resource := ResourceType(parts[0])
	action := mapHTTPMethodToAction(r.Method)

	requirements = append(requirements, PermissionRequirement{
		Resource: resource,
		Action:   action,
	})

	if len(parts) > 1 {
		requirements[0].ResourceID = parts[1]
	}

	return requirements
}

func (m *RBACMiddleware) CheckHTTPRequestPermissions(r *http.Request, user *auth.User) bool {
	requirements := m.RequirementsFromHTTPRequest(r)
	ctx := r.Context()

	for _, req := range requirements {
		perm := Permission{
			Resource:   req.Resource,
			Action:     req.Action,
			ResourceID: req.ResourceID,
		}

		if !m.checker.Policy().HasPermission(ctx, user.ID, perm) {
			return false
		}
	}

	return true
}
