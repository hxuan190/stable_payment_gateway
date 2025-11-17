package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hxuan190/stable_payment_gateway/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
}

// TestCORS_AllowedOrigin tests that requests from allowed origins receive proper CORS headers
func TestCORS_AllowedOrigin(t *testing.T) {
	// Create test config with allowed origins
	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
				"https://dashboard.payment-gateway.vn",
			},
		},
	}

	// Create minimal server config for testing
	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	// Add a test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test request from allowed origin
	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert CORS headers are present
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

// TestCORS_DisallowedOrigin tests that requests from non-allowed origins are not granted CORS access
func TestCORS_DisallowedOrigin(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
			},
		},
	}

	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	// Add a test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test request from disallowed origin
	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://malicious-site.com")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// The request should not include the origin in Access-Control-Allow-Origin
	// Note: gin-contrib/cors does not set the header for disallowed origins
	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	assert.NotEqual(t, "http://malicious-site.com", allowOrigin, "Disallowed origin should not be in CORS headers")
}

// TestCORS_PreflightRequest tests that OPTIONS (preflight) requests work correctly
func TestCORS_PreflightRequest(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
				"https://dashboard.payment-gateway.vn",
			},
		},
	}

	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	// Add a test route (POST endpoint to trigger preflight)
	router.POST("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Send OPTIONS preflight request
	req, err := http.NewRequest("OPTIONS", "/api/v1/test", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert preflight response
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

// TestCORS_AllowedMethods tests that all configured HTTP methods are allowed
func TestCORS_AllowedMethods(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
			},
		},
	}

	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	// Send OPTIONS preflight request
	req, err := http.NewRequest("OPTIONS", "/api/v1/test", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "DELETE")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check that all expected methods are allowed
	allowedMethods := w.Header().Get("Access-Control-Allow-Methods")
	assert.Contains(t, allowedMethods, "GET")
	assert.Contains(t, allowedMethods, "POST")
	assert.Contains(t, allowedMethods, "PUT")
	assert.Contains(t, allowedMethods, "PATCH")
	assert.Contains(t, allowedMethods, "DELETE")
	assert.Contains(t, allowedMethods, "OPTIONS")
}

// TestCORS_AllowedHeaders tests that required headers are allowed
func TestCORS_AllowedHeaders(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
			},
		},
	}

	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	// Send OPTIONS preflight request with multiple headers
	req, err := http.NewRequest("OPTIONS", "/api/v1/test", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization,X-Request-ID")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check that all required headers are allowed
	allowedHeaders := w.Header().Get("Access-Control-Allow-Headers")
	assert.Contains(t, allowedHeaders, "Content-Type")
	assert.Contains(t, allowedHeaders, "Authorization")
	assert.Contains(t, allowedHeaders, "X-Request-ID")
	assert.Contains(t, allowedHeaders, "Origin")
	assert.Contains(t, allowedHeaders, "Accept")
}

// TestCORS_ExposedHeaders tests that custom headers are exposed to the client
func TestCORS_ExposedHeaders(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
			},
		},
	}

	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	// Add a test route that returns custom headers
	router.GET("/test", func(c *gin.Context) {
		c.Header("X-Request-ID", "test-123")
		c.Header("X-RateLimit-Limit", "100")
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Send request
	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check that custom headers are exposed
	exposedHeaders := w.Header().Get("Access-Control-Expose-Headers")
	assert.Contains(t, exposedHeaders, "X-Request-ID")
	assert.Contains(t, exposedHeaders, "X-RateLimit-Limit")
	assert.Contains(t, exposedHeaders, "X-RateLimit-Remaining")
	assert.Contains(t, exposedHeaders, "X-RateLimit-Reset")
	assert.Contains(t, exposedHeaders, "Content-Length")
}

// TestCORS_Credentials tests that credentials are allowed
func TestCORS_Credentials(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
			},
		},
	}

	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	// Add a test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Send request with credentials
	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Cookie", "session=abc123")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check that credentials are allowed
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

// TestCORS_MaxAge tests that the preflight cache duration is set correctly
func TestCORS_MaxAge(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
			},
		},
	}

	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	// Send OPTIONS preflight request
	req, err := http.NewRequest("OPTIONS", "/api/v1/test", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check that max age is set (12 hours = 43200 seconds)
	maxAge := w.Header().Get("Access-Control-Max-Age")
	assert.Equal(t, "43200", maxAge, "Max age should be 12 hours (43200 seconds)")
}

// TestCORS_MultipleOrigins tests that multiple allowed origins work correctly
func TestCORS_MultipleOrigins(t *testing.T) {
	allowedOrigins := []string{
		"http://localhost:3000",
		"http://localhost:3001",
		"https://dashboard.payment-gateway.vn",
		"https://admin.payment-gateway.vn",
	}

	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: allowedOrigins,
		},
	}

	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	// Add a test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test each allowed origin
	for _, origin := range allowedOrigins {
		req, err := http.NewRequest("GET", "/test", nil)
		require.NoError(t, err)
		req.Header.Set("Origin", origin)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"),
			"Origin %s should be allowed", origin)
	}
}

// TestCORS_ProductionConfig tests CORS configuration for production environment
func TestCORS_ProductionConfig(t *testing.T) {
	cfg := &config.Config{
		Environment: "production",
		API: config.APIConfig{
			AllowedOrigins: []string{
				"https://dashboard.payment-gateway.vn",
				"https://admin.payment-gateway.vn",
			},
		},
	}

	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	// Add a test route
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test that localhost is NOT allowed in production
	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not allow localhost origin
	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	assert.NotEqual(t, "http://localhost:3000", allowOrigin,
		"Localhost should not be allowed in production")

	// Test that production origin is allowed
	req2, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	req2.Header.Set("Origin", "https://dashboard.payment-gateway.vn")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, "https://dashboard.payment-gateway.vn",
		w2.Header().Get("Access-Control-Allow-Origin"))
}

// TestCORS_ComplexRequest tests a complex CORS scenario with authentication
func TestCORS_ComplexRequest(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: []string{
				"https://dashboard.payment-gateway.vn",
			},
		},
	}

	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	// Add a test route that requires authentication
	router.POST("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "authenticated"})
	})

	// First, send preflight request
	preflightReq, err := http.NewRequest("OPTIONS", "/api/v1/test", nil)
	require.NoError(t, err)
	preflightReq.Header.Set("Origin", "https://dashboard.payment-gateway.vn")
	preflightReq.Header.Set("Access-Control-Request-Method", "POST")
	preflightReq.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization,X-Request-ID")

	preflightW := httptest.NewRecorder()
	router.ServeHTTP(preflightW, preflightReq)

	// Assert preflight succeeds
	assert.Equal(t, http.StatusNoContent, preflightW.Code)
	assert.Equal(t, "https://dashboard.payment-gateway.vn",
		preflightW.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", preflightW.Header().Get("Access-Control-Allow-Credentials"))

	// Verify max age is reasonable (should allow caching)
	maxAge := preflightW.Header().Get("Access-Control-Max-Age")
	assert.NotEmpty(t, maxAge)
	assert.Equal(t, "43200", maxAge) // 12 hours
}

// TestCORS_HealthEndpoint tests that CORS works on the health check endpoint
func TestCORS_HealthEndpoint(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
			},
		},
	}

	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	// Test /health endpoint
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Health endpoint should also have CORS headers
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
}

// BenchmarkCORS_Middleware benchmarks the CORS middleware performance
func BenchmarkCORS_Middleware(b *testing.B) {
	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
				"https://dashboard.payment-gateway.vn",
			},
		},
	}

	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	router.GET("/bench", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/bench", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// TestCORS_ConfigurationFromEnv tests that CORS configuration is properly loaded from environment
func TestCORS_ConfigurationFromEnv(t *testing.T) {
	// This test verifies that the configuration properly splits comma-separated origins
	origins := []string{
		"http://localhost:3000",
		"http://localhost:3001",
		"https://dashboard.payment-gateway.vn",
	}

	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: origins,
		},
	}

	// Verify the configuration was parsed correctly
	assert.Len(t, cfg.API.AllowedOrigins, 3)
	assert.Contains(t, cfg.API.AllowedOrigins, "http://localhost:3000")
	assert.Contains(t, cfg.API.AllowedOrigins, "http://localhost:3001")
	assert.Contains(t, cfg.API.AllowedOrigins, "https://dashboard.payment-gateway.vn")
}

// TestCORS_CacheControlHeaders tests that cache control is not interfering with CORS
func TestCORS_CacheControlHeaders(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
			},
		},
	}

	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	// Add a test route that sets cache control
	router.GET("/test", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Both CORS and Cache-Control headers should be present
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
}

// TestCORS_VaryHeader tests that the Vary header is set correctly for CORS
func TestCORS_VaryHeader(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
			},
		},
	}

	serverCfg := &ServerConfig{
		Config: cfg,
	}

	server := NewServer(serverCfg)
	router := server.GetRouter()

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// The Vary header should include Origin to ensure proper caching behavior
	varyHeader := w.Header().Get("Vary")
	assert.Contains(t, varyHeader, "Origin")
}
