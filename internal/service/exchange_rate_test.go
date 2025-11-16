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

	service := NewExchangeRateService(
		coinGeckoServer.URL,
		binanceServer.URL,
		5*time.Minute,
		10*time.Second,
		cache,
		logger,
	)

	_, err := service.GetUSDTToVND(context.Background())
	assert.Error(t, err)
	assert.Equal(t, ErrExchangeRateNotAvailable, err)
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

	service := NewExchangeRateService(
		server.URL,
		"https://api.binance.com/api/v3",
		5*time.Minute,
		100*time.Millisecond, // Very short timeout
		cache,
		logger,
	)

	ctx := context.Background()
	_, err := service.GetUSDTToVND(ctx)
	assert.Error(t, err)
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
