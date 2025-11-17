package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMerchantRepository is a mock implementation of MerchantRepository
type MockMerchantRepository struct {
	mock.Mock
}

func (m *MockMerchantRepository) GetByAPIKey(apiKey string) (*model.Merchant, error) {
	args := m.Called(apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Merchant), args.Error(1)
}

// MockCache is a mock implementation of Cache
type MockCache struct {
	mock.Mock
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCache) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCache) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestAPIKeyAuth_Success(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create mock merchant
	merchant := &model.Merchant{
		ID:           "merchant-123",
		Email:        "test@example.com",
		BusinessName: "Test Business",
		KYCStatus:    model.KYCStatusApproved,
		Status:       model.MerchantStatusActive,
		APIKey:       sql.NullString{String: "test-api-key-12345", Valid: true},
	}

	// Setup mocks
	mockRepo := new(MockMerchantRepository)
	mockCache := new(MockCache)

	// Mock cache miss
	mockCache.On("Get", mock.Anything, "merchant:apikey:test-api-key-12345").
		Return("", errors.New("cache miss"))

	// Mock repository call
	mockRepo.On("GetByAPIKey", "test-api-key-12345").Return(merchant, nil)

	// Mock cache set
	mockCache.On("Set", mock.Anything, "merchant:apikey:test-api-key-12345", mock.Anything, 5*time.Minute).
		Return(nil)

	// Create test router
	router := gin.New()
	router.Use(APIKeyAuth(APIKeyAuthConfig{
		MerchantRepo: mockRepo,
		Cache:        mockCache,
		CacheTTL:     5 * time.Minute,
	}))
	router.GET("/test", func(c *gin.Context) {
		merchantFromCtx, err := GetMerchantFromContext(c)
		assert.NoError(t, err)
		assert.Equal(t, merchant.ID, merchantFromCtx.ID)
		c.JSON(200, gin.H{"success": true})
	})

	// Create request
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-api-key-12345")

	// Record response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, 200, w.Code)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestAPIKeyAuth_CacheHit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	merchant := &model.Merchant{
		ID:           "merchant-123",
		Email:        "test@example.com",
		BusinessName: "Test Business",
		KYCStatus:    model.KYCStatusApproved,
		Status:       model.MerchantStatusActive,
		APIKey:       sql.NullString{String: "test-api-key-12345", Valid: true},
	}

	// Setup mocks
	mockRepo := new(MockMerchantRepository)
	mockCache := new(MockCache)

	// Mock cache hit
	merchantJSON, _ := json.Marshal(merchant)
	mockCache.On("Get", mock.Anything, "merchant:apikey:test-api-key-12345").
		Return(string(merchantJSON), nil)

	// Repository should NOT be called
	mockRepo.AssertNotCalled(t, "GetByAPIKey", mock.Anything)

	// Create test router
	router := gin.New()
	router.Use(APIKeyAuth(APIKeyAuthConfig{
		MerchantRepo: mockRepo,
		Cache:        mockCache,
		CacheTTL:     5 * time.Minute,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	// Create request
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-api-key-12345")

	// Record response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, 200, w.Code)
	mockCache.AssertExpectations(t)
}

func TestAPIKeyAuth_MissingAuthHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockMerchantRepository)
	mockCache := new(MockCache)

	router := gin.New()
	router.Use(APIKeyAuth(APIKeyAuthConfig{
		MerchantRepo: mockRepo,
		Cache:        mockCache,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	// No Authorization header

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
	assert.Contains(t, w.Body.String(), "Missing Authorization header")
}

func TestAPIKeyAuth_InvalidAuthHeaderFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockMerchantRepository)
	mockCache := new(MockCache)

	router := gin.New()
	router.Use(APIKeyAuth(APIKeyAuthConfig{
		MerchantRepo: mockRepo,
		Cache:        mockCache,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	testCases := []struct {
		name       string
		authHeader string
	}{
		{"No Bearer prefix", "test-api-key"},
		{"Wrong prefix", "Basic test-api-key"},
		{"Empty token", "Bearer "},
		{"Only Bearer", "Bearer"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tc.authHeader)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, 401, w.Code)
			assert.Contains(t, w.Body.String(), "Invalid Authorization header format")
		})
	}
}

func TestAPIKeyAuth_InvalidAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockMerchantRepository)
	mockCache := new(MockCache)

	// Mock cache miss
	mockCache.On("Get", mock.Anything, "merchant:apikey:invalid-key").
		Return("", errors.New("cache miss"))

	// Mock repository returning error
	mockRepo.On("GetByAPIKey", "invalid-key").
		Return(nil, sql.ErrNoRows)

	router := gin.New()
	router.Use(APIKeyAuth(APIKeyAuthConfig{
		MerchantRepo: mockRepo,
		Cache:        mockCache,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-key")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid API key")
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestAPIKeyAuth_MerchantKYCNotApproved(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name      string
		kycStatus model.KYCStatus
		status    model.MerchantStatus
		hasAPIKey bool
		expected  string
	}{
		{
			name:      "KYC Pending",
			kycStatus: model.KYCStatusPending,
			status:    model.MerchantStatusActive,
			hasAPIKey: true,
			expected:  "Merchant KYC not approved (status: pending)",
		},
		{
			name:      "KYC Rejected",
			kycStatus: model.KYCStatusRejected,
			status:    model.MerchantStatusActive,
			hasAPIKey: true,
			expected:  "Merchant KYC not approved (status: rejected)",
		},
		{
			name:      "Merchant Suspended",
			kycStatus: model.KYCStatusApproved,
			status:    model.MerchantStatusSuspended,
			hasAPIKey: true,
			expected:  "Merchant account not active (status: suspended)",
		},
		{
			name:      "No API Key",
			kycStatus: model.KYCStatusApproved,
			status:    model.MerchantStatusActive,
			hasAPIKey: false,
			expected:  "API key not configured",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			merchant := &model.Merchant{
				ID:           "merchant-123",
				Email:        "test@example.com",
				BusinessName: "Test Business",
				KYCStatus:    tc.kycStatus,
				Status:       tc.status,
			}

			if tc.hasAPIKey {
				merchant.APIKey = sql.NullString{String: "test-api-key", Valid: true}
			}

			mockRepo := new(MockMerchantRepository)
			mockCache := new(MockCache)

			mockCache.On("Get", mock.Anything, "merchant:apikey:test-api-key").
				Return("", errors.New("cache miss"))
			mockRepo.On("GetByAPIKey", "test-api-key").Return(merchant, nil)
			mockCache.On("Set", mock.Anything, "merchant:apikey:test-api-key", mock.Anything, 5*time.Minute).
				Return(nil)

			router := gin.New()
			router.Use(APIKeyAuth(APIKeyAuthConfig{
				MerchantRepo: mockRepo,
				Cache:        mockCache,
			}))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"success": true})
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer test-api-key")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, 403, w.Code)
			assert.Contains(t, w.Body.String(), tc.expected)
			mockRepo.AssertExpectations(t)
			mockCache.AssertExpectations(t)
		})
	}
}

func TestAPIKeyAuth_DatabaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockMerchantRepository)
	mockCache := new(MockCache)

	mockCache.On("Get", mock.Anything, "merchant:apikey:test-api-key").
		Return("", errors.New("cache miss"))
	mockRepo.On("GetByAPIKey", "test-api-key").
		Return(nil, errors.New("database connection failed"))

	router := gin.New()
	router.Use(APIKeyAuth(APIKeyAuthConfig{
		MerchantRepo: mockRepo,
		Cache:        mockCache,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to authenticate request")
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestExtractBearerToken(t *testing.T) {
	testCases := []struct {
		name       string
		authHeader string
		expected   string
	}{
		{"Valid Bearer token", "Bearer abc123xyz", "abc123xyz"},
		{"Bearer with extra spaces", "Bearer   abc123xyz   ", "abc123xyz"},
		{"Lowercase bearer", "bearer abc123xyz", "abc123xyz"},
		{"Mixed case", "BeArEr abc123xyz", "abc123xyz"},
		{"No Bearer prefix", "abc123xyz", ""},
		{"Wrong prefix", "Basic abc123xyz", ""},
		{"Empty token", "Bearer ", ""},
		{"Only Bearer", "Bearer", ""},
		{"Empty string", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractBearerToken(tc.authHeader)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMaskAPIKey(t *testing.T) {
	testCases := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{"Long key", "abcdefgh12345678ijklmnop", "abcdefgh...mnop"},
		{"Short key", "short", "***"},
		{"Exactly 12 chars", "123456789012", "***"},
		{"13 chars", "1234567890123", "12345678...0123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := maskAPIKey(tc.apiKey)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetMerchantFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		expectedMerchant := &model.Merchant{
			ID:           "merchant-123",
			BusinessName: "Test Business",
		}
		c.Set(MerchantContextKey, expectedMerchant)

		merchant, err := GetMerchantFromContext(c)
		assert.NoError(t, err)
		assert.Equal(t, expectedMerchant, merchant)
	})

	t.Run("Not found", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		merchant, err := GetMerchantFromContext(c)
		assert.Error(t, err)
		assert.Nil(t, merchant)
		assert.Contains(t, err.Error(), "merchant not found in context")
	})

	t.Run("Invalid type", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(MerchantContextKey, "not a merchant")

		merchant, err := GetMerchantFromContext(c)
		assert.Error(t, err)
		assert.Nil(t, merchant)
		assert.Contains(t, err.Error(), "invalid merchant type")
	})
}

func TestGetMerchantIDFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(MerchantIDKey, "merchant-123")

		merchantID, err := GetMerchantIDFromContext(c)
		assert.NoError(t, err)
		assert.Equal(t, "merchant-123", merchantID)
	})

	t.Run("Not found", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		merchantID, err := GetMerchantIDFromContext(c)
		assert.Error(t, err)
		assert.Empty(t, merchantID)
		assert.Contains(t, err.Error(), "merchant ID not found in context")
	})

	t.Run("Invalid type", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(MerchantIDKey, 123)

		merchantID, err := GetMerchantIDFromContext(c)
		assert.Error(t, err)
		assert.Empty(t, merchantID)
		assert.Contains(t, err.Error(), "invalid merchant ID type")
	})
}
