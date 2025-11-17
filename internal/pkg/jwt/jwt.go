package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when the token is invalid
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when the token has expired
	ErrExpiredToken = errors.New("token has expired")
	// ErrInvalidClaims is returned when the token claims are invalid
	ErrInvalidClaims = errors.New("invalid token claims")
)

// Claims represents the JWT claims
type Claims struct {
	AdminID string `json:"admin_id"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	jwt.RegisteredClaims
}

// Manager handles JWT token operations
type Manager struct {
	secretKey          string
	expirationDuration time.Duration
}

// NewManager creates a new JWT manager
func NewManager(secretKey string, expirationHours int) *Manager {
	return &Manager{
		secretKey:          secretKey,
		expirationDuration: time.Duration(expirationHours) * time.Hour,
	}
}

// GenerateToken creates a new JWT token for an admin user
func (m *Manager) GenerateToken(adminID, email, role string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(m.expirationDuration)

	claims := &Claims{
		AdminID: adminID,
		Email:   email,
		Role:    role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "payment-gateway",
			Subject:   adminID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(m.secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			// Verify signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(m.secretKey), nil
		},
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshToken generates a new token with the same claims but extended expiration
func (m *Manager) RefreshToken(tokenString string) (string, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		// Allow refresh even if token is expired (within a reasonable window)
		if !errors.Is(err, ErrExpiredToken) {
			return "", err
		}

		// Try to parse the expired token to get claims
		token, parseErr := jwt.ParseWithClaims(
			tokenString,
			&Claims{},
			func(token *jwt.Token) (interface{}, error) {
				return []byte(m.secretKey), nil
			},
			jwt.WithoutClaimsValidation(),
		)
		if parseErr != nil {
			return "", fmt.Errorf("failed to parse expired token: %w", parseErr)
		}

		var ok bool
		claims, ok = token.Claims.(*Claims)
		if !ok {
			return "", ErrInvalidClaims
		}

		// Check if token expired too long ago (max 7 days grace period)
		if time.Since(claims.ExpiresAt.Time) > 7*24*time.Hour {
			return "", fmt.Errorf("token expired too long ago, please login again")
		}
	}

	// Generate new token with same claims
	return m.GenerateToken(claims.AdminID, claims.Email, claims.Role)
}

// ExtractClaims extracts claims from a token without validating expiration
// This is useful for logging or audit purposes
func (m *Manager) ExtractClaims(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(m.secretKey), nil
		},
		jwt.WithoutClaimsValidation(),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// GetExpirationTime returns the expiration time for a token
func (m *Manager) GetExpirationTime(tokenString string) (time.Time, error) {
	claims, err := m.ExtractClaims(tokenString)
	if err != nil {
		return time.Time{}, err
	}

	return claims.ExpiresAt.Time, nil
}

// IsTokenExpired checks if a token is expired
func (m *Manager) IsTokenExpired(tokenString string) bool {
	claims, err := m.ExtractClaims(tokenString)
	if err != nil {
		return true
	}

	return time.Now().After(claims.ExpiresAt.Time)
}
