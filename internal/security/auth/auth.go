package auth

import (
	"context"
	"crypto/x509"
	"net/http"
)

// Credentials represents user authentication credentials
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResult represents the result of an authentication attempt
type AuthResult struct {
	Success   bool   `json:"success"`
	UserID    string `json:"user_id,omitempty"`
	Token     string `json:"token,omitempty"`
	Error     string `json:"error,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

// User represents an authenticated user
type User struct {
	ID          string            `json:"id"`
	Username    string            `json:"username"`
	Roles       []string          `json:"roles"`
	Permissions []string          `json:"permissions"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Authenticator defines the interface for authentication mechanisms
type Authenticator interface {
	Authenticate(ctx context.Context, creds Credentials) (*AuthResult, error)
	ValidateToken(ctx context.Context, token string) (*User, error)
	RefreshToken(ctx context.Context, token string) (*AuthResult, error)
	RevokeToken(ctx context.Context, token string) error
}

// CertificateValidator defines the interface for mTLS certificate validation
type CertificateValidator interface {
	ValidateCertificate(cert *x509.Certificate) (*User, error)
	GetTrustedCerts() []*x509.Certificate
}

// SessionManager defines the interface for session management
type SessionManager interface {
	CreateSession(ctx context.Context, user *User) (string, error)
	GetSession(ctx context.Context, sessionID string) (*User, error)
	DeleteSession(ctx context.Context, sessionID string) error
	ValidateSession(ctx context.Context, sessionID string) (bool, error)
}

// AuthMiddleware provides HTTP middleware for authentication
type AuthMiddleware struct {
	auth          Authenticator
	sessionMgr    SessionManager
	certValidator CertificateValidator
	optionalAuth  bool
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(auth Authenticator, sessionMgr SessionManager, certValidator CertificateValidator, optional bool) *AuthMiddleware {
	return &AuthMiddleware{
		auth:          auth,
		sessionMgr:    sessionMgr,
		certValidator: certValidator,
		optionalAuth:  optional,
	}
}

// Middleware returns an HTTP middleware function
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var user *User
		var err error

		// Try mTLS certificate authentication first
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			user, err = m.certValidator.ValidateCertificate(r.TLS.PeerCertificates[0])
			if err == nil && user != nil {
				ctx = context.WithValue(ctx, UserContextKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try session-based authentication
		sessionCookie, err := r.Cookie("session_id")
		if err == nil {
			user, err = m.sessionMgr.GetSession(ctx, sessionCookie.Value)
			if err == nil && user != nil {
				ctx = context.WithValue(ctx, UserContextKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try token-based authentication
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			token := extractTokenFromHeader(authHeader)
			if token != "" {
				user, err = m.auth.ValidateToken(ctx, token)
				if err == nil && user != nil {
					ctx = context.WithValue(ctx, UserContextKey, user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		// No valid authentication found
		if m.optionalAuth {
			next.ServeHTTP(w, r)
			return
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// UserContextKey is the context key for storing authenticated user
type contextKey string

const UserContextKey contextKey = "user"

// GetUserFromContext retrieves the authenticated user from context
func GetUserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(UserContextKey).(*User)
	return user, ok
}

// extractTokenFromHeader extracts the bearer token from Authorization header
func extractTokenFromHeader(header string) string {
	const prefix = "Bearer "
	if len(header) > len(prefix) && header[:len(prefix)] == prefix {
		return header[len(prefix):]
	}
	return ""
}
