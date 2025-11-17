package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRateLimiter is a mock implementation of RateLimiter for testing
type MockRateLimiter struct {
	mock.Mock
}

func (m *MockRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitState, error) {
	args := m.Called(ctx, key, limit, window)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RateLimitState), args.Error(1)
}

func TestRateLimit_IPRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		ipState        *RateLimitState
		ipError        error
		expectedStatus int
		expectedBody   string
		checkHeaders   bool
	}{
		{
			name: "IP rate limit - allowed",
			ipState: &RateLimitState{
				Allowed:   true,
				Limit:     1000,
				Remaining: 999,
				RetryIn:   0,
			},
			ipError:        nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"ok"}`,
			checkHeaders:   true,
		},
		{
			name: "IP rate limit - exceeded",
			ipState: &RateLimitState{
				Allowed:   false,
				Limit:     1000,
				Remaining: 0,
				RetryIn:   45,
			},
			ipError:        nil,
			expectedStatus: http.StatusTooManyRequests,
			expectedBody:   `"RATE_LIMIT_EXCEEDED"`,
			checkHeaders:   true,
		},
		{
			name:           "IP rate limit - error allows request",
			ipState:        nil,
			ipError:        fmt.Errorf("redis connection error"),
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"ok"}`,
			checkHeaders:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLimiter := new(MockRateLimiter)
			mockLimiter.On("Allow", mock.Anything, mock.MatchedBy(func(key string) bool {
				return key == "ratelimit:ip:192.0.2.1"
			}), 1000, 1*time.Minute).Return(tt.ipState, tt.ipError)

			router := gin.New()
			router.Use(RateLimit(RateLimitConfig{
				Limiter: mockLimiter,
				IPLimit: 1000,
				Window:  1 * time.Minute,
			}))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.0.2.1:12345"
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)

			if tt.checkHeaders && tt.ipState != nil {
				if tt.ipState.Allowed {
					assert.Equal(t, "1000", w.Header().Get("X-RateLimit-Limit"))
					assert.Equal(t, "999", w.Header().Get("X-RateLimit-Remaining"))
					assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
				} else {
					assert.Equal(t, "1000", w.Header().Get("X-RateLimit-Limit"))
					assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
					assert.Equal(t, "45", w.Header().Get("Retry-After"))
					assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
				}
			}

			mockLimiter.AssertExpectations(t)
		})
	}
}

func TestRateLimit_APIKeyRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		authHeader     string
		ipState        *RateLimitState
		apiKeyState    *RateLimitState
		apiKeyError    error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:       "API key rate limit - allowed",
			authHeader: "Bearer test-api-key-12345",
			ipState: &RateLimitState{
				Allowed:   true,
				Limit:     1000,
				Remaining: 999,
				RetryIn:   0,
			},
			apiKeyState: &RateLimitState{
				Allowed:   true,
				Limit:     100,
				Remaining: 95,
				RetryIn:   0,
			},
			apiKeyError:    nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"ok"}`,
		},
		{
			name:       "API key rate limit - exceeded",
			authHeader: "Bearer test-api-key-12345",
			ipState: &RateLimitState{
				Allowed:   true,
				Limit:     1000,
				Remaining: 999,
				RetryIn:   0,
			},
			apiKeyState: &RateLimitState{
				Allowed:   false,
				Limit:     100,
				Remaining: 0,
				RetryIn:   30,
			},
			apiKeyError:    nil,
			expectedStatus: http.StatusTooManyRequests,
			expectedBody:   `"API_KEY_RATE_LIMIT_EXCEEDED"`,
		},
		{
			name:       "API key rate limit - error allows request",
			authHeader: "Bearer test-api-key-12345",
			ipState: &RateLimitState{
				Allowed:   true,
				Limit:     1000,
				Remaining: 999,
				RetryIn:   0,
			},
			apiKeyState:    nil,
			apiKeyError:    fmt.Errorf("redis error"),
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"ok"}`,
		},
		{
			name:       "No API key - only IP limit checked",
			authHeader: "",
			ipState: &RateLimitState{
				Allowed:   true,
				Limit:     1000,
				Remaining: 999,
				RetryIn:   0,
			},
			apiKeyState:    nil,
			apiKeyError:    nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"ok"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLimiter := new(MockRateLimiter)

			// Mock IP rate limit check
			mockLimiter.On("Allow", mock.Anything, "ratelimit:ip:192.0.2.1", 1000, 1*time.Minute).
				Return(tt.ipState, nil)

			// Mock API key rate limit check if auth header is present
			if tt.authHeader != "" {
				mockLimiter.On("Allow", mock.Anything, "ratelimit:apikey:test-api-key-12345", 100, 1*time.Minute).
					Return(tt.apiKeyState, tt.apiKeyError)
			}

			router := gin.New()
			router.Use(RateLimit(RateLimitConfig{
				Limiter:     mockLimiter,
				APIKeyLimit: 100,
				IPLimit:     1000,
				Window:      1 * time.Minute,
			}))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.0.2.1:12345"
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)

			mockLimiter.AssertExpectations(t)
		})
	}
}

func TestRateLimit_Headers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name              string
		skipSuccessHeader bool
		authHeader        string
		apiKeyState       *RateLimitState
		ipState           *RateLimitState
		checkHeaders      map[string]string
	}{
		{
			name:              "Headers included by default - API key",
			skipSuccessHeader: false,
			authHeader:        "Bearer test-key",
			apiKeyState: &RateLimitState{
				Allowed:   true,
				Limit:     100,
				Remaining: 95,
				RetryIn:   0,
			},
			ipState: &RateLimitState{
				Allowed:   true,
				Limit:     1000,
				Remaining: 999,
				RetryIn:   0,
			},
			checkHeaders: map[string]string{
				"X-RateLimit-Limit":     "100",
				"X-RateLimit-Remaining": "95",
			},
		},
		{
			name:              "Headers skipped when configured",
			skipSuccessHeader: true,
			authHeader:        "Bearer test-key",
			apiKeyState: &RateLimitState{
				Allowed:   true,
				Limit:     100,
				Remaining: 95,
				RetryIn:   0,
			},
			ipState: &RateLimitState{
				Allowed:   true,
				Limit:     1000,
				Remaining: 999,
				RetryIn:   0,
			},
			checkHeaders: map[string]string{
				"X-RateLimit-Limit":     "",
				"X-RateLimit-Remaining": "",
			},
		},
		{
			name:              "Headers included - no API key",
			skipSuccessHeader: false,
			authHeader:        "",
			ipState: &RateLimitState{
				Allowed:   true,
				Limit:     1000,
				Remaining: 999,
				RetryIn:   0,
			},
			checkHeaders: map[string]string{
				"X-RateLimit-Limit":     "1000",
				"X-RateLimit-Remaining": "999",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLimiter := new(MockRateLimiter)

			// Mock IP rate limit
			mockLimiter.On("Allow", mock.Anything, "ratelimit:ip:192.0.2.1", 1000, 1*time.Minute).
				Return(tt.ipState, nil)

			// Mock API key rate limit if present
			if tt.authHeader != "" {
				mockLimiter.On("Allow", mock.Anything, "ratelimit:apikey:test-key", 100, 1*time.Minute).
					Return(tt.apiKeyState, nil)
			}

			router := gin.New()
			router.Use(RateLimit(RateLimitConfig{
				Limiter:           mockLimiter,
				APIKeyLimit:       100,
				IPLimit:           1000,
				Window:            1 * time.Minute,
				SkipSuccessHeader: tt.skipSuccessHeader,
			}))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.0.2.1:12345"
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			for header, expectedValue := range tt.checkHeaders {
				actualValue := w.Header().Get(header)
				if expectedValue == "" {
					assert.Empty(t, actualValue, "Header %s should be empty", header)
				} else {
					assert.Equal(t, expectedValue, actualValue, "Header %s mismatch", header)
				}
			}

			mockLimiter.AssertExpectations(t)
		})
	}
}

func TestRateLimit_DefaultConfiguration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockLimiter := new(MockRateLimiter)
	mockLimiter.On("Allow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&RateLimitState{
			Allowed:   true,
			Limit:     100,
			Remaining: 99,
			RetryIn:   0,
		}, nil)

	router := gin.New()
	// Create with empty config to test defaults
	router.Use(RateLimit(RateLimitConfig{
		Limiter: mockLimiter,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify default values were used (100 for API key, 1000 for IP, 1 minute window)
	calls := mockLimiter.Calls
	require.Len(t, calls, 1)

	// Check that IP limit was called with default of 1000
	assert.Equal(t, 1000, calls[0].Arguments.Get(2).(int))
	assert.Equal(t, 1*time.Minute, calls[0].Arguments.Get(3).(time.Duration))
}

func TestRateLimit_MultipleRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockLimiter := new(MockRateLimiter)

	// First 5 requests allowed
	for i := 0; i < 5; i++ {
		mockLimiter.On("Allow", mock.Anything, "ratelimit:ip:192.0.2.1", 1000, 1*time.Minute).
			Return(&RateLimitState{
				Allowed:   true,
				Limit:     1000,
				Remaining: 999 - i,
				RetryIn:   0,
			}, nil).Once()

		mockLimiter.On("Allow", mock.Anything, "ratelimit:apikey:test-key", 100, 1*time.Minute).
			Return(&RateLimitState{
				Allowed:   true,
				Limit:     100,
				Remaining: 99 - i,
				RetryIn:   0,
			}, nil).Once()
	}

	// 6th request blocked
	mockLimiter.On("Allow", mock.Anything, "ratelimit:ip:192.0.2.1", 1000, 1*time.Minute).
		Return(&RateLimitState{
			Allowed:   true,
			Limit:     1000,
			Remaining: 994,
			RetryIn:   0,
		}, nil).Once()

	mockLimiter.On("Allow", mock.Anything, "ratelimit:apikey:test-key", 100, 1*time.Minute).
		Return(&RateLimitState{
			Allowed:   false,
			Limit:     100,
			Remaining: 0,
			RetryIn:   30,
		}, nil).Once()

	router := gin.New()
	router.Use(RateLimit(RateLimitConfig{
		Limiter:     mockLimiter,
		APIKeyLimit: 100,
		IPLimit:     1000,
		Window:      1 * time.Minute,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// Make 5 successful requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.0.2.1:12345"
		req.Header.Set("Authorization", "Bearer test-key")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}

	// 6th request should be blocked
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	req.Header.Set("Authorization", "Bearer test-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "API_KEY_RATE_LIMIT_EXCEEDED")
	assert.Equal(t, "30", w.Header().Get("Retry-After"))

	mockLimiter.AssertExpectations(t)
}

func TestExtractBearerTokenForRateLimit(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		expected   string
	}{
		{
			name:       "Valid Bearer token",
			authHeader: "Bearer abc123",
			expected:   "abc123",
		},
		{
			name:       "Bearer with spaces",
			authHeader: "Bearer   token-with-spaces",
			expected:   "  token-with-spaces",
		},
		{
			name:       "Invalid format - no Bearer",
			authHeader: "abc123",
			expected:   "",
		},
		{
			name:       "Empty string",
			authHeader: "",
			expected:   "",
		},
		{
			name:       "Only Bearer",
			authHeader: "Bearer",
			expected:   "",
		},
		{
			name:       "Case sensitive",
			authHeader: "bearer abc123",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBearerTokenForRateLimit(tt.authHeader)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskAPIKeyForRateLimit(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "Long API key",
			apiKey:   "abcdefgh12345678ijklmnop",
			expected: "abcdefgh...mnop",
		},
		{
			name:     "Short API key",
			apiKey:   "short",
			expected: "***",
		},
		{
			name:     "Exactly 12 characters",
			apiKey:   "abcdefghijkl",
			expected: "***",
		},
		{
			name:     "13 characters",
			apiKey:   "abcdefghijklm",
			expected: "abcdefgh...jklm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskAPIKeyForRateLimit(tt.apiKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}
