package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CredentialValidator defines an interface for validating user credentials.
// Implementations should check credentials against a user store (database, LDAP, etc.).
type CredentialValidator interface {
	ValidateCredentials(ctx context.Context, username, password string) (userID string, roles []string, permissions []string, err error)
}

// defaultCredentialValidator is a placeholder that always returns an error.
type defaultCredentialValidator struct{}

func (d *defaultCredentialValidator) ValidateCredentials(ctx context.Context, username, password string) (string, []string, []string, error) {
	return "", nil, nil, errors.New("no credential validator configured")
}

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey        string        `json:"secret_key"`
	Issuer           string        `json:"issuer"`
	Audience         string        `json:"audience"`
	TokenExpiry      time.Duration `json:"token_expiry"`
	RefreshExpiry    time.Duration `json:"refresh_expiry"`
	SigningAlgorithm string        `json:"signing_algorithm"`
}

// DefaultJWTConfig returns default JWT configuration
func DefaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		SecretKey:        "default-secret-key-for-testing-only-change-in-production",
		Issuer:           "ussycode-auth",
		Audience:         "ussycode-api",
		TokenExpiry:      time.Hour * 24,
		RefreshExpiry:    time.Hour * 24 * 7,
		SigningAlgorithm: "HS256",
	}
}

// JWTAuthenticator implements JWT-based authentication
type JWTAuthenticator struct {
	config              *JWTConfig
	credentialValidator CredentialValidator
	// revokedTokens stores revoked JTIs mapped to their expiration time.
	// Tokens are checked against this blacklist during validation.
	revokedTokens sync.Map // map[string]time.Time (JTI -> expiration)
}

// NewJWTAuthenticator creates a new JWT authenticator
func NewJWTAuthenticator(config *JWTConfig) (*JWTAuthenticator, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}
	if config.SecretKey == "" {
		return nil, errors.New("secret key is required")
	}
	return &JWTAuthenticator{
		config:              config,
		credentialValidator: &defaultCredentialValidator{},
	}, nil
}

// SetCredentialValidator sets a custom credential validator for authentication.
func (j *JWTAuthenticator) SetCredentialValidator(v CredentialValidator) {
	j.credentialValidator = v
}

// jwtClaims represents custom JWT claims
type jwtClaims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	JTI         string   `json:"jti"` // JWT ID for uniqueness
	jwt.RegisteredClaims
}

// Authenticate generates a JWT token for valid credentials
func (j *JWTAuthenticator) Authenticate(ctx context.Context, creds Credentials) (*AuthResult, error) {
	if creds.Username == "" || creds.Password == "" {
		return &AuthResult{
			Success: false,
			Error:   "username and password are required",
		}, nil
	}

	// Delegate credential validation to the pluggable validator
	userID, roles, permissions, err := j.credentialValidator.ValidateCredentials(ctx, creds.Username, creds.Password)
	if err != nil {
		return &AuthResult{
			Success: false,
			Error:   "invalid credentials",
		}, nil
	}

	token, expiresAt, err := j.generateToken(userID, creds.Username, roles, permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResult{
		Success:   true,
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

// ValidateToken validates a JWT token and returns the associated user
func (j *JWTAuthenticator) ValidateToken(ctx context.Context, tokenStr string) (*User, error) {
	expectedAlg := j.config.SigningAlgorithm
	if expectedAlg == "" {
		expectedAlg = "HS256"
	}

	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != expectedAlg {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.config.SecretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check if the token has been revoked (blacklisted by JTI)
	if _, revoked := j.revokedTokens.Load(claims.JTI); revoked {
		return nil, ErrInvalidToken
	}

	return &User{
		ID:          claims.UserID,
		Username:    claims.Username,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
	}, nil
}

// RefreshToken generates a new token from an existing valid token
func (j *JWTAuthenticator) RefreshToken(ctx context.Context, tokenStr string) (*AuthResult, error) {
	user, err := j.ValidateToken(ctx, tokenStr)
	if err != nil {
		return nil, err
	}

	newToken, expiresAt, err := j.generateToken(user.ID, user.Username, user.Roles, user.Permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new token: %w", err)
	}

	return &AuthResult{
		Success:   true,
		UserID:    user.ID,
		Token:     newToken,
		ExpiresAt: expiresAt,
	}, nil
}

// RevokeToken revokes a token by adding its JTI to the in-memory blacklist
func (j *JWTAuthenticator) RevokeToken(ctx context.Context, tokenStr string) error {
	// Parse the token to extract the JTI and expiration
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.config.SecretKey), nil
	})
	if err != nil {
		return ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return ErrInvalidToken
	}

	// Store the JTI with its expiration time so it can be cleaned up later
	expiration := time.Time{}
	if claims.ExpiresAt != nil {
		expiration = claims.ExpiresAt.Time
	}
	j.revokedTokens.Store(claims.JTI, expiration)

	return nil
}

// CleanupRevokedTokens removes expired entries from the revoked tokens blacklist.
// This should be called periodically to prevent unbounded memory growth.
func (j *JWTAuthenticator) CleanupRevokedTokens() {
	now := time.Now()
	j.revokedTokens.Range(func(key, value interface{}) bool {
		if exp, ok := value.(time.Time); ok && !exp.IsZero() && now.After(exp) {
			j.revokedTokens.Delete(key)
		}
		return true
	})
}

// generateToken creates a new JWT token with the given user information
func (j *JWTAuthenticator) generateToken(userID, username string, roles, permissions []string) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(j.config.TokenExpiry)

	// Generate a cryptographically secure random JWT ID
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", 0, fmt.Errorf("failed to generate JWT ID: %w", err)
	}
	jti := hex.EncodeToString(jtiBytes)

	claims := jwtClaims{
		UserID:      userID,
		Username:    username,
		Roles:       roles,
		Permissions: permissions,
		JTI:         jti,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.config.Issuer,
			Subject:   userID,
			Audience:  []string{j.config.Audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(getSigningMethod(j.config.SigningAlgorithm), claims)
	tokenStr, err := token.SignedString([]byte(j.config.SecretKey))
	if err != nil {
		return "", 0, err
	}

	return tokenStr, expiresAt.Unix(), nil
}

// getSigningMethod returns the JWT signing method based on the algorithm name
func getSigningMethod(algorithm string) jwt.SigningMethod {
	switch algorithm {
	case "HS384":
		return jwt.SigningMethodHS384
	case "HS512":
		return jwt.SigningMethodHS512
	default:
		return jwt.SigningMethodHS256
	}
}
