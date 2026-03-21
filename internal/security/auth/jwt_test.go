package auth

import (
	"context"
	"testing"
	"time"
)

func TestDefaultJWTConfig(t *testing.T) {
	config := DefaultJWTConfig()

	if config.Issuer != "ussycode-auth" {
		t.Errorf("Expected issuer ussycode-auth, got %s", config.Issuer)
	}

	if config.Audience != "ussycode-api" {
		t.Errorf("Expected audience ussycode-api, got %s", config.Audience)
	}

	if config.TokenExpiry != time.Hour*24 {
		t.Errorf("Expected token expiry 24h, got %v", config.TokenExpiry)
	}

	if config.SigningAlgorithm != "HS256" {
		t.Errorf("Expected algorithm HS256, got %s", config.SigningAlgorithm)
	}
}

func TestNewJWTAuthenticator(t *testing.T) {
	tests := []struct {
		name    string
		config  *JWTConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &JWTConfig{
				SecretKey: "test-secret-key",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty secret key",
			config: &JWTConfig{
				SecretKey: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewJWTAuthenticator(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewJWTAuthenticator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJWTAuthenticator_Authenticate(t *testing.T) {
	config := &JWTConfig{
		SecretKey: "test-secret-key-12345678901234567890",
		Issuer:    "test-issuer",
		Audience:  "test-audience",
	}

	auth, err := NewJWTAuthenticator(config)
	if err != nil {
		t.Fatalf("Failed to create authenticator: %v", err)
	}

	tests := []struct {
		name      string
		creds     Credentials
		wantErr   bool
		wantToken bool
	}{
		{
			name: "valid credentials",
			creds: Credentials{
				Username: "testuser",
				Password: "valid_password",
			},
			wantErr:   false,
			wantToken: true,
		},
		{
			name: "invalid password",
			creds: Credentials{
				Username: "testuser",
				Password: "wrong_password",
			},
			wantErr:   false,
			wantToken: false,
		},
		{
			name: "empty username",
			creds: Credentials{
				Username: "",
				Password: "password",
			},
			wantErr:   false,
			wantToken: false,
		},
		{
			name: "empty password",
			creds: Credentials{
				Username: "testuser",
				Password: "",
			},
			wantErr:   false,
			wantToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := auth.Authenticate(context.Background(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("Authenticate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result.Success != tt.wantToken {
				t.Errorf("Authenticate() success = %v, want %v", result.Success, tt.wantToken)
			}

			if tt.wantToken && result.Token == "" {
				t.Error("Expected token to be generated")
			}

			if tt.wantToken && result.UserID == "" {
				t.Error("Expected UserID to be set")
			}
		})
	}
}

func TestJWTAuthenticator_ValidateToken(t *testing.T) {
	config := DefaultJWTConfig()
	config.SecretKey = "test-secret-key-12345678901234567890"

	auth, err := NewJWTAuthenticator(config)
	if err != nil {
		t.Fatalf("Failed to create authenticator: %v", err)
	}

	// First authenticate to get a token
	creds := Credentials{
		Username: "testuser",
		Password: "valid_password",
	}

	result, err := auth.Authenticate(context.Background(), creds)
	if err != nil {
		t.Fatalf("Failed to authenticate: %v", err)
	}

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:    "valid token",
			token:   result.Token,
			wantErr: nil,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "invalid token",
			token:   "invalid.token.here",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "malformed token",
			token:   "not-a-jwt",
			wantErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := auth.ValidateToken(context.Background(), tt.token)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ValidateToken() expected error %v, got nil", tt.wantErr)
					return
				}
				// Check if error type matches
				if err.Error() != tt.wantErr.Error() && !isExpiredTokenError(err, tt.wantErr) {
					t.Errorf("ValidateToken() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateToken() unexpected error = %v", err)
				return
			}

			if user.ID != result.UserID {
				t.Errorf("ValidateToken() user ID = %v, want %v", user.ID, result.UserID)
			}

			if user.Username != creds.Username {
				t.Errorf("ValidateToken() username = %v, want %v", user.Username, creds.Username)
			}
		})
	}
}

func TestJWTAuthenticator_RefreshToken(t *testing.T) {
	config := DefaultJWTConfig()
	config.SecretKey = "test-secret-key-12345678901234567890"

	auth, err := NewJWTAuthenticator(config)
	if err != nil {
		t.Fatalf("Failed to create authenticator: %v", err)
	}

	// Authenticate to get a token
	creds := Credentials{
		Username: "testuser",
		Password: "valid_password",
	}

	result, err := auth.Authenticate(context.Background(), creds)
	if err != nil {
		t.Fatalf("Failed to authenticate: %v", err)
	}

	// Refresh the token
	newResult, err := auth.RefreshToken(context.Background(), result.Token)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	if !newResult.Success {
		t.Error("Expected refresh to be successful")
	}

	if newResult.Token == "" {
		t.Error("Expected new token to be generated")
	}

	if newResult.Token == result.Token {
		t.Error("Expected new token to be different from old token")
	}

	// Verify the new token is valid
	user, err := auth.ValidateToken(context.Background(), newResult.Token)
	if err != nil {
		t.Errorf("Failed to validate refreshed token: %v", err)
	}

	if user.Username != creds.Username {
		t.Errorf("Expected username %s, got %s", creds.Username, user.Username)
	}
}

func TestJWTAuthenticator_RevokeToken(t *testing.T) {
	config := DefaultJWTConfig()
	config.SecretKey = "test-secret-key-12345678901234567890"

	auth, err := NewJWTAuthenticator(config)
	if err != nil {
		t.Fatalf("Failed to create authenticator: %v", err)
	}

	// Authenticate to get a token
	creds := Credentials{
		Username: "testuser",
		Password: "valid_password",
	}

	result, err := auth.Authenticate(context.Background(), creds)
	if err != nil {
		t.Fatalf("Failed to authenticate: %v", err)
	}

	// Revoke the token
	err = auth.RevokeToken(context.Background(), result.Token)
	if err != nil {
		t.Errorf("Failed to revoke token: %v", err)
	}

	// Try to revoke invalid token
	err = auth.RevokeToken(context.Background(), "invalid_token")
	if err == nil {
		t.Error("Expected error when revoking invalid token")
	}
}

func TestGetSigningMethod(t *testing.T) {
	tests := []struct {
		algorithm string
		want      string
	}{
		{"HS256", "HS256"},
		{"HS384", "HS384"},
		{"HS512", "HS512"},
		{"unknown", "HS256"}, // defaults to HS256
	}

	for _, tt := range tests {
		t.Run(tt.algorithm, func(t *testing.T) {
			method := getSigningMethod(tt.algorithm)
			if method.Alg() != tt.want {
				t.Errorf("getSigningMethod() = %v, want %v", method.Alg(), tt.want)
			}
		})
	}
}

func isExpiredTokenError(err, target error) bool {
	return err.Error() == target.Error()
}
