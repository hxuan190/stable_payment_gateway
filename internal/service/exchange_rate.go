package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// ExchangeRateService handles fetching and caching cryptocurrency exchange rates
type ExchangeRateService struct {
	primaryAPI   string
	secondaryAPI string
	cacheTTL     time.Duration
	timeout      time.Duration
	httpClient   *http.Client
	cache        ExchangeRateCache
	logger       *logrus.Logger
}

// ExchangeRateCache defines the interface for caching exchange rates
type ExchangeRateCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
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

var (
	// ErrExchangeRateNotAvailable is returned when exchange rate cannot be fetched
	ErrExchangeRateNotAvailable = errors.New("exchange rate not available")
	// ErrInvalidResponse is returned when API response is invalid
	ErrInvalidResponse = errors.New("invalid API response")
)

const (
	// Cache key for USDT/VND rate
	cacheKeyUSDTVND = "exchange_rate:usdt:vnd"
)

// NewExchangeRateService creates a new exchange rate service
func NewExchangeRateService(
	primaryAPI, secondaryAPI string,
	cacheTTL, timeout time.Duration,
	cache ExchangeRateCache,
	logger *logrus.Logger,
) *ExchangeRateService {
	return &ExchangeRateService{
		primaryAPI:   primaryAPI,
		secondaryAPI: secondaryAPI,
		cacheTTL:     cacheTTL,
		timeout:      timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		cache:  cache,
		logger: logger,
	}
}

// GetUSDTToVND returns the current USDT to VND exchange rate
// It first checks the cache, then tries primary API (CoinGecko), and falls back to secondary API (Binance)
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

	// Try primary API (CoinGecko)
	rate, err := s.fetchFromCoinGecko(ctx)
	if err == nil {
		s.logger.WithField("rate", rate).Debug("Exchange rate fetched from CoinGecko")
		// Cache the result
		if s.cache != nil {
			if err := s.saveToCache(ctx, rate); err != nil {
				s.logger.WithError(err).Warn("Failed to cache exchange rate")
			}
		}
		return rate, nil
	}

	s.logger.WithError(err).Warn("Primary API (CoinGecko) failed, trying fallback")

	// Fallback to secondary API (Binance)
	rate, err = s.fetchFromBinance(ctx)
	if err == nil {
		s.logger.WithField("rate", rate).Debug("Exchange rate fetched from Binance (fallback)")
		// Cache the result
		if s.cache != nil {
			if err := s.saveToCache(ctx, rate); err != nil {
				s.logger.WithError(err).Warn("Failed to cache exchange rate")
			}
		}
		return rate, nil
	}

	s.logger.WithError(err).Error("Secondary API (Binance) also failed")
	return decimal.Zero, ErrExchangeRateNotAvailable
}

// GetUSDCToVND returns the current USDC to VND exchange rate
// Note: For MVP, USDC is treated as 1:1 with USDT (both are USD-pegged stablecoins)
func (s *ExchangeRateService) GetUSDCToVND(ctx context.Context) (decimal.Decimal, error) {
	// USDC and USDT are both USD-pegged stablecoins, so we use the same rate
	// In production, you might want to fetch USDC separately
	return s.GetUSDTToVND(ctx)
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

// InvalidateCache removes the cached exchange rate
func (s *ExchangeRateService) InvalidateCache(ctx context.Context) error {
	if s.cache != nil {
		// Set with 0 TTL to delete
		return s.cache.Set(ctx, cacheKeyUSDTVND, "", 0)
	}
	return nil
}
