package auth

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

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
	config *JWTConfig
}

// NewJWTAuthenticator creates a new JWT authenticator
func NewJWTAuthenticator(config *JWTConfig) (*JWTAuthenticator, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}
	if config.SecretKey == "" {
		return nil, errors.New("secret key is required")
	}
	return &JWTAuthenticator{config: config}, nil
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

	// In a real implementation, validate credentials against a database
	// For now, we'll use a simple validation
	if creds.Password != "valid_password" {
		return &AuthResult{
			Success: false,
			Error:   "invalid credentials",
		}, nil
	}

	userID := fmt.Sprintf("user_%s", creds.Username)
	token, expiresAt, err := j.generateToken(userID, creds.Username, []string{"user"}, []string{"read", "write"})
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

// RevokeToken revokes a token (in JWT, tokens are stateless, so this would typically use a blacklist)
func (j *JWTAuthenticator) RevokeToken(ctx context.Context, tokenStr string) error {
	// In a production system, you would add the token to a blacklist
	// For now, we'll just validate it exists
	_, err := j.ValidateToken(ctx, tokenStr)
	if err != nil {
		return err
	}
	return nil
}

// generateToken creates a new JWT token with the given user information
func (j *JWTAuthenticator) generateToken(userID, username string, roles, permissions []string) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(j.config.TokenExpiry)

	// Generate unique JWT ID using timestamp and nanoseconds
	jti := fmt.Sprintf("%s-%d-%d", userID, now.Unix(), now.UnixNano())

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

// GenerateKeyPair generates an RSA key pair for asymmetric JWT signing
func GenerateKeyPair() (crypto.PrivateKey, crypto.PublicKey, error) {
	// This would typically generate RSA keys
	// For simplicity in this implementation, we return nil
	return nil, nil, errors.New("RSA key generation not implemented")
}
