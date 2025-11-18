package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCache is a mock implementation of ExchangeRateCache for testing
type MockCache struct {
	data map[string]string
	err  error
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]string),
	}
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	val, ok := m.data[key]
	if !ok {
		return "", errors.New("key not found")
	}
	return val, nil
}

func (m *MockCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if m.err != nil {
		return m.err
	}
	if ttl == 0 {
		delete(m.data, key)
		return nil
	}
	m.data[key] = value
	return nil
}

func (m *MockCache) SetError(err error) {
	m.err = err
}

func TestNewExchangeRateService(t *testing.T) {
	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Suppress logs during tests

	service := NewExchangeRateService(
		"https://api.coingecko.com/api/v3",
		"https://api.binance.com/api/v3",
		5*time.Minute,
		10*time.Second,
		cache,
		logger,
	)

	assert.NotNil(t, service)
	assert.Equal(t, "https://api.coingecko.com/api/v3", service.primaryAPI)
	assert.Equal(t, "https://api.binance.com/api/v3", service.secondaryAPI)
	assert.Equal(t, 5*time.Minute, service.cacheTTL)
	assert.Equal(t, 10*time.Second, service.timeout)
}

func TestGetUSDTToVND_FromCache(t *testing.T) {
	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Pre-populate cache
	expectedRate := decimal.NewFromFloat(25000.50)
	cache.Set(context.Background(), cacheKeyUSDTVND, expectedRate.String(), 5*time.Minute)

	service := NewExchangeRateService(
		"https://api.coingecko.com/api/v3",
		"https://api.binance.com/api/v3",
		5*time.Minute,
		10*time.Second,
		cache,
		logger,
	)

	rate, err := service.GetUSDTToVND(context.Background())
	require.NoError(t, err)
	assert.True(t, expectedRate.Equal(rate), "Expected %s, got %s", expectedRate, rate)
}

func TestGetUSDTToVND_FromCoinGecko(t *testing.T) {
	// Create a test server that mimics CoinGecko API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/simple/price")
		assert.Equal(t, "tether", r.URL.Query().Get("ids"))
		assert.Equal(t, "vnd", r.URL.Query().Get("vs_currencies"))

		response := CoinGeckoResponse{}
		response.Tether.VND = 25345.67
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateService(
		server.URL,
		"https://api.binance.com/api/v3",
		5*time.Minute,
		10*time.Second,
		cache,
		logger,
	)

	rate, err := service.GetUSDTToVND(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "25345.67", rate.String())

	// Verify it was cached
	cachedRate, err := cache.Get(context.Background(), cacheKeyUSDTVND)
	require.NoError(t, err)
	assert.Equal(t, "25345.67", cachedRate)
}

func TestGetUSDTToVND_CoinGeckoFallbackToBinance(t *testing.T) {
	// Create a test server for CoinGecko that returns an error
	coinGeckoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer coinGeckoServer.Close()

	// Create a test server for Binance (fallback)
	binanceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Binance doesn't support VND directly, so it should fail too
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}))
	defer binanceServer.Close()

	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateServiceWithConfig(
		coinGeckoServer.URL,
		binanceServer.URL,
		"",
		5*time.Minute,
		24*time.Hour,
		2*time.Second,
		0, // No retries to make test faster
		cache,
		logger,
		&CircuitBreakerConfig{
			Threshold: 10,
			Timeout:   60 * time.Second,
		},
	)

	_, err := service.GetUSDTToVND(context.Background())
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAllProvidersFailed)
}

func TestGetUSDTToVND_InvalidResponse(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateService(
		server.URL,
		"https://api.binance.com/api/v3",
		5*time.Minute,
		10*time.Second,
		cache,
		logger,
	)

	_, err := service.GetUSDTToVND(context.Background())
	assert.Error(t, err)
}

func TestGetUSDTToVND_ZeroRate(t *testing.T) {
	// Create a test server that returns zero rate
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CoinGeckoResponse{}
		response.Tether.VND = 0 // Invalid rate
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateService(
		server.URL,
		"https://api.binance.com/api/v3",
		5*time.Minute,
		10*time.Second,
		cache,
		logger,
	)

	_, err := service.GetUSDTToVND(context.Background())
	assert.Error(t, err)
}

func TestGetUSDTToVND_CacheMiss(t *testing.T) {
	// Create a test server that mimics CoinGecko API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CoinGeckoResponse{}
		response.Tether.VND = 25100.00
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateService(
		server.URL,
		"https://api.binance.com/api/v3",
		5*time.Minute,
		10*time.Second,
		cache,
		logger,
	)

	// First call - should fetch from API
	rate1, err := service.GetUSDTToVND(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "25100", rate1.String())

	// Second call - should fetch from cache
	rate2, err := service.GetUSDTToVND(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "25100", rate2.String())
}

func TestGetUSDTToVND_NilCache(t *testing.T) {
	// Create a test server that mimics CoinGecko API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CoinGeckoResponse{}
		response.Tether.VND = 25200.50
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Service without cache
	service := NewExchangeRateService(
		server.URL,
		"https://api.binance.com/api/v3",
		5*time.Minute,
		10*time.Second,
		nil, // No cache
		logger,
	)

	rate, err := service.GetUSDTToVND(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "25200.5", rate.String())
}

func TestGetUSDCToVND(t *testing.T) {
	// Create a test server that mimics CoinGecko API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CoinGeckoResponse{}
		response.Tether.VND = 25300.00
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateService(
		server.URL,
		"https://api.binance.com/api/v3",
		5*time.Minute,
		10*time.Second,
		cache,
		logger,
	)

	// USDC should return the same rate as USDT (both are USD-pegged)
	rate, err := service.GetUSDCToVND(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "25300", rate.String())
}

func TestInvalidateCache(t *testing.T) {
	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Pre-populate cache
	expectedRate := decimal.NewFromFloat(25000.50)
	cache.Set(context.Background(), cacheKeyUSDTVND, expectedRate.String(), 5*time.Minute)

	service := NewExchangeRateService(
		"https://api.coingecko.com/api/v3",
		"https://api.binance.com/api/v3",
		5*time.Minute,
		10*time.Second,
		cache,
		logger,
	)

	// Verify cache is populated
	_, err := cache.Get(context.Background(), cacheKeyUSDTVND)
	require.NoError(t, err)

	// Invalidate cache
	err = service.InvalidateCache(context.Background())
	require.NoError(t, err)

	// Verify cache is empty
	_, err = cache.Get(context.Background(), cacheKeyUSDTVND)
	assert.Error(t, err)
}

func TestInvalidateCache_NilCache(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateService(
		"https://api.coingecko.com/api/v3",
		"https://api.binance.com/api/v3",
		5*time.Minute,
		10*time.Second,
		nil, // No cache
		logger,
	)

	// Should not error when cache is nil
	err := service.InvalidateCache(context.Background())
	assert.NoError(t, err)
}

func TestGetUSDTToVND_ContextTimeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		response := CoinGeckoResponse{}
		response.Tether.VND = 25000.00
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateServiceWithConfig(
		server.URL,
		"",
		"",
		5*time.Minute,
		24*time.Hour,
		100*time.Millisecond, // Very short timeout
		0, // No retries to make test faster
		cache,
		logger,
		&CircuitBreakerConfig{
			Threshold: 5,
			Timeout:   60 * time.Second,
		},
	)

	ctx := context.Background()
	_, err := service.GetUSDTToVND(ctx)
	assert.Error(t, err)
}

// TestRetryWithExponentialBackoff tests the retry logic
func TestRetryWithExponentialBackoff(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		response := CoinGeckoResponse{}
		response.Tether.VND = 25000.00
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateServiceWithConfig(
		server.URL,
		"",
		"",
		5*time.Minute,
		24*time.Hour,
		10*time.Second,
		3, // Allow 3 retries
		cache,
		logger,
		nil,
	)

	rate, err := service.GetUSDTToVND(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "25000", rate.String())
	assert.Equal(t, 3, attempts, "Should have made 3 attempts")
}

// TestCircuitBreaker tests the circuit breaker functionality
func TestCircuitBreaker(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateServiceWithConfig(
		server.URL,
		"",
		"",
		5*time.Minute,
		24*time.Hour,
		1*time.Second,
		0, // No retries
		cache,
		logger,
		&CircuitBreakerConfig{
			Threshold: 3,
			Timeout:   10 * time.Second,
		},
	)

	// Make requests until circuit breaker opens
	for i := 0; i < 5; i++ {
		_, err := service.GetUSDTToVND(context.Background())
		assert.Error(t, err)
	}

	// Check circuit breaker status
	status := service.GetCircuitBreakerStatus()
	assert.True(t, status["coingecko"].IsOpen, "Circuit breaker should be open")
	assert.GreaterOrEqual(t, status["coingecko"].FailureCount, 3)

	// Reset circuit breaker
	service.ResetCircuitBreaker("coingecko")
	status = service.GetCircuitBreakerStatus()
	assert.False(t, status["coingecko"].IsOpen, "Circuit breaker should be closed after reset")
}

// TestStaleCacheFallback tests the stale cache fallback
func TestStaleCacheFallback(t *testing.T) {
	// Create a server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Pre-populate stale cache
	staleRate := decimal.NewFromFloat(24000.00)
	cache.Set(context.Background(), cacheKeyStaleUSDTVND, staleRate.String(), 24*time.Hour)

	service := NewExchangeRateServiceWithConfig(
		server.URL,
		"",
		"",
		5*time.Minute,
		24*time.Hour,
		1*time.Second,
		0, // No retries
		cache,
		logger,
		&CircuitBreakerConfig{
			Threshold: 10,
			Timeout:   60 * time.Second,
		},
	)

	// Should fall back to stale cache
	rate, err := service.GetUSDTToVND(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "24000", rate.String())
}

// TestHealthCheck tests the health check functionality
func TestHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CoinGeckoResponse{}
		response.Tether.VND = 25000.00
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateServiceWithConfig(
		server.URL,
		"",
		"",
		5*time.Minute,
		24*time.Hour,
		10*time.Second,
		3,
		cache,
		logger,
		nil,
	)

	// Populate cache
	cache.Set(context.Background(), cacheKeyUSDTVND, "25000", 5*time.Minute)

	status := service.HealthCheck(context.Background())

	assert.True(t, status["coingecko"].Healthy, "CoinGecko should be healthy")
	assert.True(t, status["cache"].Healthy, "Cache should be healthy")
	assert.Contains(t, status["coingecko"].Message, "healthy")
}

// TestCryptoCompare tests the CryptoCompare provider
func TestFetchFromCryptoCompare(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/price")
		assert.Equal(t, "USDT", r.URL.Query().Get("fsym"))
		assert.Equal(t, "VND", r.URL.Query().Get("tsyms"))

		response := CryptoCompareResponse{
			VND: 25400.50,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateServiceWithConfig(
		"http://invalid-url.com", // Make CoinGecko fail
		"http://invalid-url.com", // Make Binance fail
		server.URL,               // CryptoCompare should work
		5*time.Minute,
		24*time.Hour,
		2*time.Second,
		0, // No retries to make test faster
		cache,
		logger,
		&CircuitBreakerConfig{
			Threshold: 10,
			Timeout:   60 * time.Second,
		},
	)

	rate, err := service.GetUSDTToVND(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "25400.5", rate.String())
}

// TestMultipleProviderFallback tests fallback between multiple providers
func TestMultipleProviderFallback(t *testing.T) {
	coinGeckoAttempts := 0
	cryptocompareAttempts := 0

	// CoinGecko fails
	coinGeckoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coinGeckoAttempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer coinGeckoServer.Close()

	// CryptoCompare succeeds
	cryptocompareServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cryptocompareAttempts++
		response := CryptoCompareResponse{VND: 25500.00}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer cryptocompareServer.Close()

	cache := NewMockCache()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateServiceWithConfig(
		coinGeckoServer.URL,
		"",
		cryptocompareServer.URL,
		5*time.Minute,
		24*time.Hour,
		2*time.Second,
		0, // No retries
		cache,
		logger,
		&CircuitBreakerConfig{
			Threshold: 10,
			Timeout:   60 * time.Second,
		},
	)

	rate, err := service.GetUSDTToVND(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "25500", rate.String())
	assert.Greater(t, coinGeckoAttempts, 0, "Should have tried CoinGecko")
	assert.Greater(t, cryptocompareAttempts, 0, "Should have tried CryptoCompare")
}

func TestFetchFromCoinGecko_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := CoinGeckoResponse{}
		response.Tether.VND = 25500.75
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateService(
		server.URL,
		"",
		5*time.Minute,
		10*time.Second,
		nil,
		logger,
	)

	rate, err := service.fetchFromCoinGecko(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "25500.75", rate.String())
}

func TestFetchFromCoinGecko_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("Rate limit exceeded"))
	}))
	defer server.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := NewExchangeRateService(
		server.URL,
		"",
		5*time.Minute,
		10*time.Second,
		nil,
		logger,
	)

	_, err := service.fetchFromCoinGecko(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "429")
}
