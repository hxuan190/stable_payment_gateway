package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// ExchangeRateService handles fetching and caching cryptocurrency exchange rates
type ExchangeRateService struct {
	primaryAPI     string
	secondaryAPI   string
	tertiaryAPI    string
	cacheTTL       time.Duration
	staleCacheTTL  time.Duration // Extended TTL for stale cache fallback
	timeout        time.Duration
	maxRetries     int
	httpClient     *http.Client
	cache          ExchangeRateCache
	logger         *logrus.Logger
	circuitBreaker *CircuitBreaker
}

// ExchangeRateCache defines the interface for caching exchange rates
type ExchangeRateCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
}

// CircuitBreaker implements circuit breaker pattern to prevent cascading failures
type CircuitBreaker struct {
	mu              sync.RWMutex
	failures        map[string]int
	lastFailureTime map[string]time.Time
	threshold       int           // Number of failures before opening circuit
	timeout         time.Duration // Time to wait before attempting to close circuit
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	Threshold int
	Timeout   time.Duration
}

// CoinGeckoResponse represents the response from CoinGecko API
type CoinGeckoResponse struct {
	Tether struct {
		VND float64 `json:"vnd"`
	} `json:"tether"`
}

// BinanceResponse represents the response from Binance API
type BinanceResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

// CryptoCompareResponse represents the response from CryptoCompare API
type CryptoCompareResponse struct {
	VND float64 `json:"VND"`
}

var (
	// ErrExchangeRateNotAvailable is returned when exchange rate cannot be fetched
	ErrExchangeRateNotAvailable = errors.New("exchange rate not available")
	// ErrInvalidResponse is returned when API response is invalid
	ErrInvalidResponse = errors.New("invalid API response")
	// ErrCircuitBreakerOpen is returned when circuit breaker is open
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
	// ErrAllProvidersFailed is returned when all providers fail
	ErrAllProvidersFailed = errors.New("all exchange rate providers failed")
)

const (
	// Cache key for USDT/VND rate
	cacheKeyUSDTVND = "exchange_rate:usdt:vnd"
	// Cache key for stale USDT/VND rate
	cacheKeyStaleUSDTVND = "exchange_rate:usdt:vnd:stale"

	// Default values
	defaultMaxRetries    = 3
	defaultCircuitThreshold = 5
	defaultCircuitTimeout = 60 * time.Second
)

// NewExchangeRateService creates a new exchange rate service
func NewExchangeRateService(
	primaryAPI, secondaryAPI string,
	cacheTTL, timeout time.Duration,
	cache ExchangeRateCache,
	logger *logrus.Logger,
) *ExchangeRateService {
	return NewExchangeRateServiceWithConfig(
		primaryAPI,
		secondaryAPI,
		"",
		cacheTTL,
		cacheTTL*24, // Stale cache TTL is 24x regular TTL
		timeout,
		defaultMaxRetries,
		cache,
		logger,
		&CircuitBreakerConfig{
			Threshold: defaultCircuitThreshold,
			Timeout:   defaultCircuitTimeout,
		},
	)
}

// NewExchangeRateServiceWithConfig creates a new exchange rate service with full configuration
func NewExchangeRateServiceWithConfig(
	primaryAPI, secondaryAPI, tertiaryAPI string,
	cacheTTL, staleCacheTTL, timeout time.Duration,
	maxRetries int,
	cache ExchangeRateCache,
	logger *logrus.Logger,
	cbConfig *CircuitBreakerConfig,
) *ExchangeRateService {
	if cbConfig == nil {
		cbConfig = &CircuitBreakerConfig{
			Threshold: defaultCircuitThreshold,
			Timeout:   defaultCircuitTimeout,
		}
	}

	return &ExchangeRateService{
		primaryAPI:    primaryAPI,
		secondaryAPI:  secondaryAPI,
		tertiaryAPI:   tertiaryAPI,
		cacheTTL:      cacheTTL,
		staleCacheTTL: staleCacheTTL,
		timeout:       timeout,
		maxRetries:    maxRetries,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		cache:  cache,
		logger: logger,
		circuitBreaker: &CircuitBreaker{
			failures:        make(map[string]int),
			lastFailureTime: make(map[string]time.Time),
			threshold:       cbConfig.Threshold,
			timeout:         cbConfig.Timeout,
		},
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failures:        make(map[string]int),
		lastFailureTime: make(map[string]time.Time),
		threshold:       threshold,
		timeout:         timeout,
	}
}

// IsOpen checks if the circuit breaker is open for a given provider
func (cb *CircuitBreaker) IsOpen(provider string) bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	failures, exists := cb.failures[provider]
	if !exists || failures < cb.threshold {
		return false
	}

	lastFailure, exists := cb.lastFailureTime[provider]
	if !exists {
		return false
	}

	// Check if timeout has elapsed since last failure
	if time.Since(lastFailure) > cb.timeout {
		// Circuit should be half-open now, reset failures
		return false
	}

	return true
}

// RecordSuccess records a successful API call
func (cb *CircuitBreaker) RecordSuccess(provider string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	delete(cb.failures, provider)
	delete(cb.lastFailureTime, provider)
}

// RecordFailure records a failed API call
func (cb *CircuitBreaker) RecordFailure(provider string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures[provider]++
	cb.lastFailureTime[provider] = time.Now()
}

// Reset resets the circuit breaker for a given provider
func (cb *CircuitBreaker) Reset(provider string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	delete(cb.failures, provider)
	delete(cb.lastFailureTime, provider)
}

// GetStatus returns the current status of a provider
func (cb *CircuitBreaker) GetStatus(provider string) (failures int, isOpen bool) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	failures = cb.failures[provider]
	isOpen = cb.IsOpen(provider)
	return
}

// GetUSDTToVND returns the current USDT to VND exchange rate
// It first checks the cache, then tries primary API (CoinGecko), and falls back to secondary API (Binance)
// and tertiary API (CryptoCompare). If all fail, it returns stale cached data if available.
func (s *ExchangeRateService) GetUSDTToVND(ctx context.Context) (decimal.Decimal, error) {
	s.logger.Debug("Fetching USDT/VND exchange rate")

	// Try to get from cache first
	if s.cache != nil {
		cachedRate, err := s.getFromCache(ctx)
		if err == nil {
			s.logger.WithField("rate", cachedRate).Debug("Exchange rate retrieved from cache")
			return cachedRate, nil
		}
		s.logger.WithError(err).Debug("Cache miss or error, fetching from API")
	}

	// Track which providers we've tried
	providers := []struct {
		name   string
		fetch  func(context.Context) (decimal.Decimal, error)
		enabled bool
	}{
		{"coingecko", s.fetchFromCoinGeckoWithRetry, true},
		{"binance", s.fetchFromBinanceWithRetry, s.secondaryAPI != ""},
		{"cryptocompare", s.fetchFromCryptoCompareWithRetry, s.tertiaryAPI != ""},
	}

	var lastErr error
	for _, provider := range providers {
		if !provider.enabled {
			continue
		}

		// Check circuit breaker
		if s.circuitBreaker.IsOpen(provider.name) {
			s.logger.WithField("provider", provider.name).Warn("Circuit breaker is open, skipping provider")
			continue
		}

		s.logger.WithField("provider", provider.name).Debug("Trying exchange rate provider")
		rate, err := provider.fetch(ctx)
		if err == nil {
			s.logger.WithFields(logrus.Fields{
				"rate":     rate,
				"provider": provider.name,
			}).Debug("Exchange rate fetched successfully")

			// Record success in circuit breaker
			s.circuitBreaker.RecordSuccess(provider.name)

			// Cache the result
			if s.cache != nil {
				if err := s.saveToCache(ctx, rate); err != nil {
					s.logger.WithError(err).Warn("Failed to cache exchange rate")
				}
				// Also save to stale cache for future fallback
				if err := s.saveToStaleCache(ctx, rate); err != nil {
					s.logger.WithError(err).Warn("Failed to save to stale cache")
				}
			}
			return rate, nil
		}

		s.logger.WithFields(logrus.Fields{
			"provider": provider.name,
			"error":    err,
		}).Warn("Provider failed")

		// Record failure in circuit breaker
		s.circuitBreaker.RecordFailure(provider.name)
		lastErr = err
	}

	// All providers failed, try stale cache as last resort
	s.logger.Warn("All providers failed, trying stale cache")
	if s.cache != nil {
		staleRate, err := s.getFromStaleCache(ctx)
		if err == nil {
			s.logger.WithField("rate", staleRate).Warn("Using stale cached exchange rate")
			return staleRate, nil
		}
		s.logger.WithError(err).Error("Stale cache also unavailable")
	}

	if lastErr != nil {
		return decimal.Zero, fmt.Errorf("%w: %v", ErrAllProvidersFailed, lastErr)
	}
	return decimal.Zero, ErrExchangeRateNotAvailable
}

// GetUSDCToVND returns the current USDC to VND exchange rate
// Note: For MVP, USDC is treated as 1:1 with USDT (both are USD-pegged stablecoins)
func (s *ExchangeRateService) GetUSDCToVND(ctx context.Context) (decimal.Decimal, error) {
	// USDC and USDT are both USD-pegged stablecoins, so we use the same rate
	// In production, you might want to fetch USDC separately
	return s.GetUSDTToVND(ctx)
}

// retryWithExponentialBackoff retries a function with exponential backoff
func (s *ExchangeRateService) retryWithExponentialBackoff(
	ctx context.Context,
	operation func(context.Context) (decimal.Decimal, error),
	providerName string,
) (decimal.Decimal, error) {
	var lastErr error
	baseDelay := 1 * time.Second

	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff: 1s, 2s, 4s, 8s
			delay := baseDelay * (1 << (attempt - 1))
			s.logger.WithFields(logrus.Fields{
				"provider": providerName,
				"attempt":  attempt,
				"delay":    delay,
			}).Debug("Retrying after delay")

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return decimal.Zero, ctx.Err()
			}
		}

		rate, err := operation(ctx)
		if err == nil {
			if attempt > 0 {
				s.logger.WithFields(logrus.Fields{
					"provider": providerName,
					"attempt":  attempt,
				}).Info("Retry succeeded")
			}
			return rate, nil
		}

		lastErr = err
		s.logger.WithFields(logrus.Fields{
			"provider": providerName,
			"attempt":  attempt,
			"error":    err,
		}).Debug("Attempt failed")
	}

	return decimal.Zero, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// fetchFromCoinGeckoWithRetry fetches from CoinGecko with retry logic
func (s *ExchangeRateService) fetchFromCoinGeckoWithRetry(ctx context.Context) (decimal.Decimal, error) {
	return s.retryWithExponentialBackoff(ctx, s.fetchFromCoinGecko, "coingecko")
}

// fetchFromBinanceWithRetry fetches from Binance with retry logic
func (s *ExchangeRateService) fetchFromBinanceWithRetry(ctx context.Context) (decimal.Decimal, error) {
	return s.retryWithExponentialBackoff(ctx, s.fetchFromBinance, "binance")
}

// fetchFromCryptoCompareWithRetry fetches from CryptoCompare with retry logic
func (s *ExchangeRateService) fetchFromCryptoCompareWithRetry(ctx context.Context) (decimal.Decimal, error) {
	return s.retryWithExponentialBackoff(ctx, s.fetchFromCryptoCompare, "cryptocompare")
}

// fetchFromCoinGecko fetches the exchange rate from CoinGecko API
func (s *ExchangeRateService) fetchFromCoinGecko(ctx context.Context) (decimal.Decimal, error) {
	url := fmt.Sprintf("%s/simple/price?ids=tether&vs_currencies=vnd", s.primaryAPI)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to fetch from CoinGecko: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return decimal.Zero, fmt.Errorf("CoinGecko API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result CoinGeckoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return decimal.Zero, fmt.Errorf("failed to decode CoinGecko response: %w", err)
	}

	if result.Tether.VND <= 0 {
		return decimal.Zero, ErrInvalidResponse
	}

	rate := decimal.NewFromFloat(result.Tether.VND)
	return rate, nil
}

// fetchFromBinance fetches the exchange rate from Binance API
// Binance doesn't have direct USDT/VND pair, so we calculate via USD
func (s *ExchangeRateService) fetchFromBinance(ctx context.Context) (decimal.Decimal, error) {
	// Note: Binance doesn't have USDT/VND directly
	// We would need to calculate: USDT → USD → VND
	// For now, we'll use a simplified approach

	// Fetch USDT/USDC price (should be ~1.0)
	url := fmt.Sprintf("%s/ticker/price?symbol=USDTVND", s.secondaryAPI)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to fetch from Binance: %w", err)
	}
	defer resp.Body.Close()

	// If direct pair doesn't exist, we need another approach
	// For MVP, we'll return an error and rely on CoinGecko
	if resp.StatusCode == http.StatusBadRequest {
		// Try alternative: use a known USD/VND rate multiplier
		// This is a fallback approach - in production, you'd want a better solution
		return s.fetchUSDVNDFromBinance(ctx)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return decimal.Zero, fmt.Errorf("Binance API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result BinanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return decimal.Zero, fmt.Errorf("failed to decode Binance response: %w", err)
	}

	rate, err := decimal.NewFromString(result.Price)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid price in Binance response: %w", err)
	}

	return rate, nil
}

// fetchUSDVNDFromBinance is a fallback that calculates USDT/VND via USD
func (s *ExchangeRateService) fetchUSDVNDFromBinance(ctx context.Context) (decimal.Decimal, error) {
	// Since Binance doesn't have USDT/VND directly, we use a fixed multiplier
	// based on the fact that USDT is pegged to USD
	// In production, you would fetch this from a forex API

	// For now, return an error to force using CoinGecko
	return decimal.Zero, errors.New("Binance doesn't support USDT/VND pair directly")
}

// fetchFromCryptoCompare fetches the exchange rate from CryptoCompare API
func (s *ExchangeRateService) fetchFromCryptoCompare(ctx context.Context) (decimal.Decimal, error) {
	if s.tertiaryAPI == "" {
		return decimal.Zero, errors.New("CryptoCompare API not configured")
	}

	url := fmt.Sprintf("%s/price?fsym=USDT&tsyms=VND", s.tertiaryAPI)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to fetch from CryptoCompare: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return decimal.Zero, fmt.Errorf("CryptoCompare API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result CryptoCompareResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return decimal.Zero, fmt.Errorf("failed to decode CryptoCompare response: %w", err)
	}

	if result.VND <= 0 {
		return decimal.Zero, ErrInvalidResponse
	}

	rate := decimal.NewFromFloat(result.VND)
	return rate, nil
}

// getFromCache retrieves the cached exchange rate
func (s *ExchangeRateService) getFromCache(ctx context.Context) (decimal.Decimal, error) {
	value, err := s.cache.Get(ctx, cacheKeyUSDTVND)
	if err != nil {
		return decimal.Zero, err
	}

	rate, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid cached value: %w", err)
	}

	return rate, nil
}

// saveToCache stores the exchange rate in cache
func (s *ExchangeRateService) saveToCache(ctx context.Context, rate decimal.Decimal) error {
	return s.cache.Set(ctx, cacheKeyUSDTVND, rate.String(), s.cacheTTL)
}

// getFromStaleCache retrieves the stale cached exchange rate
func (s *ExchangeRateService) getFromStaleCache(ctx context.Context) (decimal.Decimal, error) {
	value, err := s.cache.Get(ctx, cacheKeyStaleUSDTVND)
	if err != nil {
		return decimal.Zero, err
	}

	rate, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid stale cached value: %w", err)
	}

	return rate, nil
}

// saveToStaleCache stores the exchange rate in stale cache with extended TTL
func (s *ExchangeRateService) saveToStaleCache(ctx context.Context, rate decimal.Decimal) error {
	return s.cache.Set(ctx, cacheKeyStaleUSDTVND, rate.String(), s.staleCacheTTL)
}

// InvalidateCache removes the cached exchange rate
func (s *ExchangeRateService) InvalidateCache(ctx context.Context) error {
	if s.cache != nil {
		// Set with 0 TTL to delete
		if err := s.cache.Set(ctx, cacheKeyUSDTVND, "", 0); err != nil {
			return err
		}
		// Also invalidate stale cache
		return s.cache.Set(ctx, cacheKeyStaleUSDTVND, "", 0)
	}
	return nil
}

// HealthCheck performs a health check on all exchange rate providers
// Returns a map of provider names to their health status
func (s *ExchangeRateService) HealthCheck(ctx context.Context) map[string]HealthStatus {
	status := make(map[string]HealthStatus)

	// Check CoinGecko
	status["coingecko"] = s.checkProviderHealth(ctx, s.fetchFromCoinGecko, "coingecko")

	// Check Binance if configured
	if s.secondaryAPI != "" {
		status["binance"] = s.checkProviderHealth(ctx, s.fetchFromBinance, "binance")
	}

	// Check CryptoCompare if configured
	if s.tertiaryAPI != "" {
		status["cryptocompare"] = s.checkProviderHealth(ctx, s.fetchFromCryptoCompare, "cryptocompare")
	}

	// Check cache
	if s.cache != nil {
		_, err := s.cache.Get(ctx, cacheKeyUSDTVND)
		if err == nil {
			status["cache"] = HealthStatus{Healthy: true, Message: "Cache is accessible"}
		} else {
			status["cache"] = HealthStatus{Healthy: false, Message: fmt.Sprintf("Cache error: %v", err)}
		}
	}

	return status
}

// HealthStatus represents the health status of a provider
type HealthStatus struct {
	Healthy        bool
	Message        string
	CircuitOpen    bool
	FailureCount   int
	LastChecked    time.Time
}

// checkProviderHealth checks the health of a single provider
func (s *ExchangeRateService) checkProviderHealth(
	ctx context.Context,
	fetch func(context.Context) (decimal.Decimal, error),
	providerName string,
) HealthStatus {
	// Create a timeout context for health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	failures, isOpen := s.circuitBreaker.GetStatus(providerName)

	status := HealthStatus{
		CircuitOpen:  isOpen,
		FailureCount: failures,
		LastChecked:  time.Now(),
	}

	if isOpen {
		status.Healthy = false
		status.Message = fmt.Sprintf("Circuit breaker is open (%d failures)", failures)
		return status
	}

	_, err := fetch(checkCtx)
	if err != nil {
		status.Healthy = false
		status.Message = fmt.Sprintf("Health check failed: %v", err)
	} else {
		status.Healthy = true
		status.Message = "Provider is healthy"
	}

	return status
}

// GetCircuitBreakerStatus returns the circuit breaker status for all providers
func (s *ExchangeRateService) GetCircuitBreakerStatus() map[string]CircuitBreakerStatus {
	providers := []string{"coingecko", "binance", "cryptocompare"}
	status := make(map[string]CircuitBreakerStatus)

	for _, provider := range providers {
		failures, isOpen := s.circuitBreaker.GetStatus(provider)
		status[provider] = CircuitBreakerStatus{
			Provider:     provider,
			IsOpen:       isOpen,
			FailureCount: failures,
		}
	}

	return status
}

// CircuitBreakerStatus represents the status of a circuit breaker for a provider
type CircuitBreakerStatus struct {
	Provider     string
	IsOpen       bool
	FailureCount int
}

// ResetCircuitBreaker resets the circuit breaker for a specific provider
func (s *ExchangeRateService) ResetCircuitBreaker(provider string) {
	s.circuitBreaker.Reset(provider)
	s.logger.WithField("provider", provider).Info("Circuit breaker reset")
}
