package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAuthMiddleware(t *testing.T) {
	jwtAuth, _ := NewJWTAuthenticator(DefaultJWTConfig())
	sessionMgr := NewMemorySessionManager(DefaultSessionConfig())
	mtlsValidator, _ := NewmTLSValidator(&mTLSConfig{})

	middleware := NewAuthMiddleware(jwtAuth, sessionMgr, mtlsValidator, false)
	if middleware == nil {
		t.Fatal("Expected middleware to be created")
	}
}

func TestAuthMiddleware_Middleware(t *testing.T) {
	jwtAuth, _ := NewJWTAuthenticator(DefaultJWTConfig())
	sessionMgr := NewMemorySessionManager(DefaultSessionConfig())
	mtlsValidator, _ := NewmTLSValidator(&mTLSConfig{})

	middleware := NewAuthMiddleware(jwtAuth, sessionMgr, mtlsValidator, false)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		setupAuth  func(*http.Request)
		wantStatus int
	}{
		{
			name:       "no authentication",
			setupAuth:  func(r *http.Request) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "valid JWT token",
			setupAuth: func(r *http.Request) {
				result, _ := jwtAuth.Authenticate(context.Background(), Credentials{
					Username: "testuser",
					Password: "valid_password",
				})
				r.Header.Set("Authorization", "Bearer "+result.Token)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid JWT token",
			setupAuth: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer invalid_token")
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "malformed auth header",
			setupAuth: func(r *http.Request) {
				r.Header.Set("Authorization", "InvalidFormat")
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			tt.setupAuth(req)

			rr := httptest.NewRecorder()
			middleware.Middleware(handler).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}

func TestAuthMiddleware_OptionalAuth(t *testing.T) {
	jwtAuth, _ := NewJWTAuthenticator(DefaultJWTConfig())
	sessionMgr := NewMemorySessionManager(DefaultSessionConfig())
	mtlsValidator, _ := NewmTLSValidator(&mTLSConfig{})

	middleware := NewAuthMiddleware(jwtAuth, sessionMgr, mtlsValidator, true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	middleware.Middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 with optional auth, got %d", rr.Code)
	}
}

func TestAuthMiddleware_SessionAuth(t *testing.T) {
	jwtAuth, _ := NewJWTAuthenticator(DefaultJWTConfig())
	sessionMgr := NewMemorySessionManager(DefaultSessionConfig())
	mtlsValidator, _ := NewmTLSValidator(&mTLSConfig{})

	middleware := NewAuthMiddleware(jwtAuth, sessionMgr, mtlsValidator, false)

	user := &User{
		ID:       "user123",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	sessionID, _ := sessionMgr.CreateSession(context.Background(), user)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})

	rr := httptest.NewRecorder()
	middleware.Middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 with valid session, got %d", rr.Code)
	}
}

func TestGetUserFromContext(t *testing.T) {
	user := &User{
		ID:       "user123",
		Username: "testuser",
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, UserContextKey, user)

	retrievedUser, ok := GetUserFromContext(ctx)
	if !ok {
		t.Fatal("Expected to find user in context")
	}

	if retrievedUser.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, retrievedUser.ID)
	}

	// Test with no user in context
	_, ok = GetUserFromContext(context.Background())
	if ok {
		t.Error("Expected not to find user in empty context")
	}
}

func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"Bearer token123", "token123"},
		{"Bearer token with spaces", "token with spaces"},
		{"NoBearer token", ""},
		{"", ""},
		{"Bearer", ""},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			result := extractTokenFromHeader(tt.header)
			if result != tt.want {
				t.Errorf("Expected %q, got %q", tt.want, result)
			}
		})
	}
}

func TestAuthResult(t *testing.T) {
	result := &AuthResult{
		Success:   true,
		UserID:    "user123",
		Token:     "token123",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}

	if result.UserID != "user123" {
		t.Errorf("Expected user ID user123, got %s", result.UserID)
	}
}

func TestUser(t *testing.T) {
	user := &User{
		ID:          "user123",
		Username:    "testuser",
		Roles:       []string{"user", "admin"},
		Permissions: []string{"read", "write"},
		Metadata: map[string]string{
			"email": "test@example.com",
		},
	}

	if user.ID != "user123" {
		t.Errorf("Expected ID user123, got %s", user.ID)
	}

	if len(user.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(user.Roles))
	}

	if user.Metadata["email"] != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", user.Metadata["email"])
	}
}

func TestCredentials(t *testing.T) {
	creds := Credentials{
		Username: "testuser",
		Password: "password123",
	}

	if creds.Username != "testuser" {
		t.Errorf("Expected username testuser, got %s", creds.Username)
	}

	if creds.Password != "password123" {
		t.Errorf("Expected password password123, got %s", creds.Password)
	}
}
