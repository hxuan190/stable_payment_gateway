package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	jwtpkg "github.com/hxuan190/stable_payment_gateway/internal/pkg/jwt"
)

const (
	// AuthorizationHeader is the header name for authorization
	AuthorizationHeader = "Authorization"
	// BearerPrefix is the prefix for bearer tokens
	BearerPrefix = "Bearer "
	// AdminIDKey is the context key for admin ID
	AdminIDKey = "admin_id"
	// AdminEmailKey is the context key for admin email
	AdminEmailKey = "admin_email"
	// AdminRoleKey is the context key for admin role
	AdminRoleKey = "admin_role"
)

var (
	// ErrMissingAuthHeader is returned when authorization header is missing
	ErrMissingAuthHeader = errors.New("authorization header is required")
	// ErrInvalidAuthHeader is returned when authorization header format is invalid
	ErrInvalidAuthHeader = errors.New("invalid authorization header format")
	// ErrUnauthorized is returned when authentication fails
	ErrUnauthorized = errors.New("unauthorized")
)

// JWTAuth creates a middleware that validates JWT tokens
func JWTAuth(jwtManager *jwtpkg.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		token, err := extractToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": err.Error(),
				},
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			var statusCode int
			var errorCode string
			var errorMessage string

			switch {
			case errors.Is(err, jwtpkg.ErrExpiredToken):
				statusCode = http.StatusUnauthorized
				errorCode = "TOKEN_EXPIRED"
				errorMessage = "Token has expired, please login again"
			case errors.Is(err, jwtpkg.ErrInvalidToken):
				statusCode = http.StatusUnauthorized
				errorCode = "INVALID_TOKEN"
				errorMessage = "Invalid authentication token"
			case errors.Is(err, jwtpkg.ErrInvalidClaims):
				statusCode = http.StatusUnauthorized
				errorCode = "INVALID_CLAIMS"
				errorMessage = "Invalid token claims"
			default:
				statusCode = http.StatusUnauthorized
				errorCode = "AUTHENTICATION_FAILED"
				errorMessage = "Authentication failed"
			}

			c.JSON(statusCode, gin.H{
				"error": gin.H{
					"code":    errorCode,
					"message": errorMessage,
				},
			})
			c.Abort()
			return
		}

		// Set claims in context for use in handlers
		c.Set(AdminIDKey, claims.AdminID)
		c.Set(AdminEmailKey, claims.Email)
		c.Set(AdminRoleKey, claims.Role)

		c.Next()
	}
}

// RequireRole creates a middleware that checks if the admin has the required role
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get admin role from context (set by JWTAuth middleware)
		adminRole, exists := c.Get(AdminRoleKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authentication required",
				},
			})
			c.Abort()
			return
		}

		// Check if admin has required role
		roleStr, ok := adminRole.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "Invalid role type",
				},
			})
			c.Abort()
			return
		}

		// Check if role matches any of the required roles
		hasRole := false
		for _, role := range roles {
			if roleStr == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "Insufficient permissions",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetAdminID retrieves the admin ID from the context
func GetAdminID(c *gin.Context) (string, error) {
	adminID, exists := c.Get(AdminIDKey)
	if !exists {
		return "", ErrUnauthorized
	}

	adminIDStr, ok := adminID.(string)
	if !ok {
		return "", errors.New("invalid admin ID type")
	}

	return adminIDStr, nil
}

// GetAdminEmail retrieves the admin email from the context
func GetAdminEmail(c *gin.Context) (string, error) {
	email, exists := c.Get(AdminEmailKey)
	if !exists {
		return "", ErrUnauthorized
	}

	emailStr, ok := email.(string)
	if !ok {
		return "", errors.New("invalid admin email type")
	}

	return emailStr, nil
}

// GetAdminRole retrieves the admin role from the context
func GetAdminRole(c *gin.Context) (string, error) {
	role, exists := c.Get(AdminRoleKey)
	if !exists {
		return "", ErrUnauthorized
	}

	roleStr, ok := role.(string)
	if !ok {
		return "", errors.New("invalid admin role type")
	}

	return roleStr, nil
}

// extractToken extracts the JWT token from the Authorization header
func extractToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader(AuthorizationHeader)
	if authHeader == "" {
		return "", ErrMissingAuthHeader
	}

	// Check if header starts with "Bearer "
	if !strings.HasPrefix(authHeader, BearerPrefix) {
		return "", ErrInvalidAuthHeader
	}

	// Extract token
	token := strings.TrimPrefix(authHeader, BearerPrefix)
	if token == "" {
		return "", ErrInvalidAuthHeader
	}

	return token, nil
}
