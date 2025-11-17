package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	jwtpkg "github.com/hxuan190/stable_payment_gateway/internal/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestJWTAuth(t *testing.T) {
	jwtManager := jwtpkg.NewManager("test-secret-key-at-least-32-chars-long", 24)

	t.Run("Valid token", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/protected", JWTAuth(jwtManager), func(c *gin.Context) {
			adminID, _ := c.Get(AdminIDKey)
			adminEmail, _ := c.Get(AdminEmailKey)
			adminRole, _ := c.Get(AdminRoleKey)

			c.JSON(http.StatusOK, gin.H{
				"admin_id":    adminID,
				"admin_email": adminEmail,
				"admin_role":  adminRole,
			})
		})

		token, err := jwtManager.GenerateToken("admin123", "admin@example.com", "super_admin")
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set(AuthorizationHeader, BearerPrefix+token)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Missing Authorization header", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/protected", JWTAuth(jwtManager), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.Contains(t, resp.Body.String(), "authorization header is required")
	})

	t.Run("Invalid Authorization header format - missing Bearer", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/protected", JWTAuth(jwtManager), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		token, err := jwtManager.GenerateToken("admin123", "admin@example.com", "admin")
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set(AuthorizationHeader, token) // Missing "Bearer " prefix
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid authorization header format")
	})

	t.Run("Invalid token", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/protected", JWTAuth(jwtManager), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set(AuthorizationHeader, BearerPrefix+"invalid-token")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.Contains(t, resp.Body.String(), "INVALID_TOKEN")
	})

	t.Run("Expired token", func(t *testing.T) {
		shortJwtManager := jwtpkg.NewManager("test-secret-key-at-least-32-chars-long", 0)
		token, err := shortJwtManager.GenerateToken("admin123", "admin@example.com", "admin")
		require.NoError(t, err)

		router := setupTestRouter()
		router.GET("/protected", JWTAuth(shortJwtManager), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set(AuthorizationHeader, BearerPrefix+token)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.Contains(t, resp.Body.String(), "TOKEN_EXPIRED")
	})

	t.Run("Token with wrong secret", func(t *testing.T) {
		wrongJwtManager := jwtpkg.NewManager("wrong-secret-key-different-32-chars", 24)
		token, err := wrongJwtManager.GenerateToken("admin123", "admin@example.com", "admin")
		require.NoError(t, err)

		router := setupTestRouter()
		router.GET("/protected", JWTAuth(jwtManager), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set(AuthorizationHeader, BearerPrefix+token)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
	})
}

func TestRequireRole(t *testing.T) {
	jwtManager := jwtpkg.NewManager("test-secret-key-at-least-32-chars-long", 24)

	t.Run("User has required role", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/admin", JWTAuth(jwtManager), RequireRole("super_admin", "admin"), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		token, err := jwtManager.GenerateToken("admin123", "admin@example.com", "super_admin")
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set(AuthorizationHeader, BearerPrefix+token)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("User does not have required role", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/admin", JWTAuth(jwtManager), RequireRole("super_admin"), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		token, err := jwtManager.GenerateToken("mod123", "mod@example.com", "moderator")
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set(AuthorizationHeader, BearerPrefix+token)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusForbidden, resp.Code)
		assert.Contains(t, resp.Body.String(), "FORBIDDEN")
		assert.Contains(t, resp.Body.String(), "Insufficient permissions")
	})

	t.Run("Multiple allowed roles", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/admin", JWTAuth(jwtManager), RequireRole("super_admin", "admin", "moderator"), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Test with moderator role
		token, err := jwtManager.GenerateToken("mod123", "mod@example.com", "moderator")
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set(AuthorizationHeader, BearerPrefix+token)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Missing auth context", func(t *testing.T) {
		router := setupTestRouter()
		// Don't use JWTAuth middleware
		router.GET("/admin", RequireRole("admin"), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.Contains(t, resp.Body.String(), "Authentication required")
	})
}

func TestGetAdminID(t *testing.T) {
	t.Run("Get admin ID from context", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/test", func(c *gin.Context) {
			c.Set(AdminIDKey, "admin123")
			adminID, err := GetAdminID(c)
			assert.NoError(t, err)
			assert.Equal(t, "admin123", adminID)
			c.JSON(http.StatusOK, gin.H{"admin_id": adminID})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Admin ID not in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		adminID, err := GetAdminID(c)
		assert.Error(t, err)
		assert.Equal(t, "", adminID)
		assert.ErrorIs(t, err, ErrUnauthorized)
	})

	t.Run("Invalid admin ID type", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(AdminIDKey, 12345) // Wrong type
		adminID, err := GetAdminID(c)
		assert.Error(t, err)
		assert.Equal(t, "", adminID)
	})
}

func TestGetAdminEmail(t *testing.T) {
	t.Run("Get admin email from context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(AdminEmailKey, "admin@example.com")
		email, err := GetAdminEmail(c)
		assert.NoError(t, err)
		assert.Equal(t, "admin@example.com", email)
	})

	t.Run("Admin email not in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		email, err := GetAdminEmail(c)
		assert.Error(t, err)
		assert.Equal(t, "", email)
		assert.ErrorIs(t, err, ErrUnauthorized)
	})

	t.Run("Invalid admin email type", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(AdminEmailKey, 12345) // Wrong type
		email, err := GetAdminEmail(c)
		assert.Error(t, err)
		assert.Equal(t, "", email)
	})
}

func TestGetAdminRole(t *testing.T) {
	t.Run("Get admin role from context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(AdminRoleKey, "super_admin")
		role, err := GetAdminRole(c)
		assert.NoError(t, err)
		assert.Equal(t, "super_admin", role)
	})

	t.Run("Admin role not in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		role, err := GetAdminRole(c)
		assert.Error(t, err)
		assert.Equal(t, "", role)
		assert.ErrorIs(t, err, ErrUnauthorized)
	})

	t.Run("Invalid admin role type", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(AdminRoleKey, 12345) // Wrong type
		role, err := GetAdminRole(c)
		assert.Error(t, err)
		assert.Equal(t, "", role)
	})
}

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		wantToken   string
		wantErr     error
	}{
		{
			name:       "Valid Bearer token",
			authHeader: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantErr:    nil,
		},
		{
			name:       "Missing Authorization header",
			authHeader: "",
			wantToken:  "",
			wantErr:    ErrMissingAuthHeader,
		},
		{
			name:       "Missing Bearer prefix",
			authHeader: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken:  "",
			wantErr:    ErrInvalidAuthHeader,
		},
		{
			name:       "Bearer with empty token",
			authHeader: "Bearer ",
			wantToken:  "",
			wantErr:    ErrInvalidAuthHeader,
		},
		{
			name:       "Wrong prefix",
			authHeader: "Basic eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantToken:  "",
			wantErr:    ErrInvalidAuthHeader,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			if tt.authHeader != "" {
				c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
				c.Request.Header.Set(AuthorizationHeader, tt.authHeader)
			} else {
				c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			}

			token, err := extractToken(c)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantToken, token)
		})
	}
}

func TestMiddlewareIntegration(t *testing.T) {
	jwtManager := jwtpkg.NewManager("test-secret-key-at-least-32-chars-long", 24)

	t.Run("Full authentication and authorization flow", func(t *testing.T) {
		router := setupTestRouter()

		// Public endpoint - no auth
		router.GET("/public", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "public"})
		})

		// Protected endpoint - auth required
		router.GET("/protected", JWTAuth(jwtManager), func(c *gin.Context) {
			adminID, _ := GetAdminID(c)
			c.JSON(http.StatusOK, gin.H{"admin_id": adminID})
		})

		// Admin endpoint - auth + role required
		router.GET("/admin", JWTAuth(jwtManager), RequireRole("super_admin"), func(c *gin.Context) {
			adminID, _ := GetAdminID(c)
			c.JSON(http.StatusOK, gin.H{"admin_id": adminID})
		})

		// Test public endpoint
		req := httptest.NewRequest(http.MethodGet, "/public", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)

		// Generate token for super_admin
		adminToken, err := jwtManager.GenerateToken("admin123", "admin@example.com", "super_admin")
		require.NoError(t, err)

		// Test protected endpoint with valid token
		req = httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set(AuthorizationHeader, BearerPrefix+adminToken)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)

		// Test admin endpoint with super_admin token
		req = httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set(AuthorizationHeader, BearerPrefix+adminToken)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)

		// Generate token for moderator
		modToken, err := jwtManager.GenerateToken("mod123", "mod@example.com", "moderator")
		require.NoError(t, err)

		// Test admin endpoint with moderator token (should fail)
		req = httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set(AuthorizationHeader, BearerPrefix+modToken)
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusForbidden, resp.Code)
	})
}
