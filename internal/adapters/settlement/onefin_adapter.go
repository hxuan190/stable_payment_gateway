package settlement

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"stable_payment_gateway/internal/ports"
)

// OneFinSettlementAdapter implements the SettlementProvider interface for OneFin API-based settlements
// This adapter integrates with OneFin's OTC settlement API for automated crypto-to-VND conversions
type OneFinSettlementAdapter struct {
	// Configuration
	config ports.SettlementProviderConfig

	// HTTP client for API calls
	httpClient *http.Client

	// Exchange rate cache
	rateCache      *ports.ExchangeRateQuote
	rateCacheMu    sync.RWMutex
	rateCacheExpiry time.Time

	// Health tracking
	health   ports.ProviderHealth
	healthMu sync.RWMutex
}

// OneFinSettlementRequest represents the request payload for OneFin API
type OneFinSettlementRequest struct {
	ReferenceID       string  `json:"reference_id"`
	MerchantID        string  `json:"merchant_id"`
	AmountCrypto      float64 `json:"amount_crypto"`
	CryptoSymbol      string  `json:"crypto_symbol"`
	BankAccountNumber string  `json:"bank_account_number"`
	BankAccountName   string  `json:"bank_account_name"`
	BankName          string  `json:"bank_name"`
}

// OneFinSettlementResponse represents the response from OneFin API
type OneFinSettlementResponse struct {
	Success         bool    `json:"success"`
	SettlementID    string  `json:"settlement_id"`
	Status          string  `json:"status"`
	AmountVND       float64 `json:"amount_vnd"`
	ExchangeRate    float64 `json:"exchange_rate"`
	Fee             float64 `json:"fee"`
	EstimatedTime   int     `json:"estimated_time_minutes"`
	ErrorMessage    string  `json:"error_message,omitempty"`
}

// OneFinExchangeRateResponse represents the exchange rate response from OneFin API
type OneFinExchangeRateResponse struct {
	Success       bool    `json:"success"`
	CryptoSymbol  string  `json:"crypto_symbol"`
	VNDPerUSD     float64 `json:"vnd_per_usd"`
	USDPerCrypto  float64 `json:"usd_per_crypto"`
	EffectiveRate float64 `json:"effective_rate"`
	Spread        float64 `json:"spread"`
	ValidUntil    string  `json:"valid_until"`
}

// NewOneFinSettlementAdapter creates a new OneFin settlement adapter
func NewOneFinSettlementAdapter(config ports.SettlementProviderConfig) (*OneFinSettlementAdapter, error) {
	if config.ProviderType != ports.SettlementProviderOneFin {
		return nil, fmt.Errorf("invalid provider type: expected %s, got %s", ports.SettlementProviderOneFin, config.ProviderType)
	}

	if config.APIURL == "" {
		return nil, fmt.Errorf("API URL is required for OneFin adapter")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for OneFin adapter")
	}

	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	adapter := &OneFinSettlementAdapter{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		health: ports.ProviderHealth{
			IsHealthy:   true,
			IsAvailable: true,
		},
	}

	return adapter, nil
}

// GetProviderType returns the type of settlement provider
func (a *OneFinSettlementAdapter) GetProviderType() ports.SettlementProviderType {
	return ports.SettlementProviderOneFin
}

// GetProviderName returns a human-readable name for the provider
func (a *OneFinSettlementAdapter) GetProviderName() string {
	if a.config.ProviderName != "" {
		return a.config.ProviderName
	}
	return "OneFin OTC Settlement"
}

// InitiateSettlement starts a new settlement operation via OneFin API
func (a *OneFinSettlementAdapter) InitiateSettlement(ctx context.Context, request ports.SettlementRequest) (*ports.SettlementResponse, error) {
	// Validate request
	if err := a.validateSettlementRequest(request); err != nil {
		return nil, fmt.Errorf("invalid settlement request: %w", err)
	}

	// Prepare OneFin API request
	oneFinReq := OneFinSettlementRequest{
		ReferenceID:       request.SettlementID,
		MerchantID:        request.MerchantID,
		AmountCrypto:      request.AmountCrypto.InexactFloat64(),
		CryptoSymbol:      request.CryptoSymbol,
		BankAccountNumber: request.BankAccountNumber,
		BankAccountName:   request.BankAccountName,
		BankName:          request.BankName,
	}

	// Call OneFin API
	oneFinResp, err := a.callOneFinAPI(ctx, "/api/v1/settlements/initiate", oneFinReq)
	if err != nil {
		a.updateHealthMetrics(false, true)
		return nil, fmt.Errorf("failed to initiate settlement with OneFin: %w", err)
	}

	if !oneFinResp.Success {
		a.updateHealthMetrics(false, true)
		return nil, fmt.Errorf("OneFin settlement failed: %s", oneFinResp.ErrorMessage)
	}

	// Convert OneFin response to standard settlement response
	now := time.Now()
	response := &ports.SettlementResponse{
		SettlementID:        request.SettlementID,
		ProviderReferenceID: oneFinResp.SettlementID,
		Status:              a.mapOneFinStatus(oneFinResp.Status),
		AmountCrypto:        request.AmountCrypto,
		AmountVND:           decimal.NewFromFloat(oneFinResp.AmountVND),
		ExchangeRate:        decimal.NewFromFloat(oneFinResp.ExchangeRate),
		Fee:                 decimal.NewFromFloat(oneFinResp.Fee),
		InitiatedAt:         now,
		Metadata: map[string]interface{}{
			"onefin_settlement_id": oneFinResp.SettlementID,
			"estimated_time_minutes": oneFinResp.EstimatedTime,
		},
	}

	a.updateHealthMetrics(true, false)

	return response, nil
}

// ConfirmSettlement polls OneFin API for settlement completion status
func (a *OneFinSettlementAdapter) ConfirmSettlement(ctx context.Context, settlementID string) (*ports.SettlementResponse, error) {
	// In a real implementation, this would poll OneFin's API to check settlement status
	// For now, we'll return a basic implementation

	status, err := a.GetSettlementStatus(ctx, settlementID)
	if err != nil {
		return nil, err
	}

	// If already completed, return as-is
	if status.Status == ports.SettlementStatusCompleted {
		return status, nil
	}

	// In production, this would poll the OneFin API and update the status
	// For MVP, we'll just return the current status
	return status, nil
}

// CancelSettlement cancels a pending settlement via OneFin API
func (a *OneFinSettlementAdapter) CancelSettlement(ctx context.Context, settlementID string) (*ports.SettlementResponse, error) {
	// Call OneFin API to cancel settlement
	// This is a placeholder - implement actual API call in production
	return nil, fmt.Errorf("settlement cancellation not implemented for OneFin")
}

// GetSettlementStatus retrieves the current status from OneFin API
func (a *OneFinSettlementAdapter) GetSettlementStatus(ctx context.Context, settlementID string) (*ports.SettlementResponse, error) {
	// In production, this would call OneFin's API to get settlement status
	// For MVP, we'll return a placeholder
	return nil, fmt.Errorf("settlement status retrieval not fully implemented")
}

// GetExchangeRate retrieves the current exchange rate from OneFin API
func (a *OneFinSettlementAdapter) GetExchangeRate(ctx context.Context, cryptoSymbol string, amountCrypto decimal.Decimal) (*ports.ExchangeRateQuote, error) {
	// Check cache first
	a.rateCacheMu.RLock()
	if a.rateCache != nil && time.Now().Before(a.rateCacheExpiry) && a.rateCache.CryptoSymbol == cryptoSymbol {
		cached := a.rateCache
		a.rateCacheMu.RUnlock()
		return cached, nil
	}
	a.rateCacheMu.RUnlock()

	// Call OneFin API for fresh rate
	// This is a placeholder - implement actual API call in production
	quote := &ports.ExchangeRateQuote{
		CryptoSymbol:  cryptoSymbol,
		VNDPerUSD:     decimal.NewFromInt(23500),
		USDPerCrypto:  decimal.NewFromInt(1), // 1:1 for stablecoins
		EffectiveRate: decimal.NewFromInt(23500),
		ValidUntil:    time.Now().Add(5 * time.Minute),
		ProviderName:  a.GetProviderName(),
		Spread:        decimal.NewFromFloat(0.01),
	}

	// Cache the quote
	a.rateCacheMu.Lock()
	a.rateCache = quote
	a.rateCacheExpiry = quote.ValidUntil
	a.rateCacheMu.Unlock()

	return quote, nil
}

// GetAvailableLiquidity retrieves liquidity from OneFin API
func (a *OneFinSettlementAdapter) GetAvailableLiquidity(ctx context.Context) (decimal.Decimal, error) {
	// In production, this would call OneFin's API to get available liquidity
	// For MVP, return a placeholder value
	return decimal.NewFromInt(500000000), nil // 500M VND
}

// IsAvailable checks if OneFin service is available
func (a *OneFinSettlementAdapter) IsAvailable(ctx context.Context) bool {
	a.healthMu.RLock()
	defer a.healthMu.RUnlock()

	return a.health.IsAvailable && a.health.IsHealthy
}

// GetProviderHealth returns health status and metrics
func (a *OneFinSettlementAdapter) GetProviderHealth(ctx context.Context) ports.ProviderHealth {
	a.healthMu.RLock()
	defer a.healthMu.RUnlock()

	return a.health
}

// callOneFinAPI makes an API call to OneFin
func (a *OneFinSettlementAdapter) callOneFinAPI(ctx context.Context, endpoint string, payload interface{}) (*OneFinSettlementResponse, error) {
	// Marshal payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	url := a.config.APIURL + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", a.config.APIKey)
	if a.config.APISecret != "" {
		req.Header.Set("X-API-Secret", a.config.APISecret)
	}

	// Make request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var oneFinResp OneFinSettlementResponse
	if err := json.Unmarshal(body, &oneFinResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &oneFinResp, nil
}

// mapOneFinStatus maps OneFin status to standard settlement status
func (a *OneFinSettlementAdapter) mapOneFinStatus(oneFinStatus string) ports.SettlementStatus {
	switch oneFinStatus {
	case "pending":
		return ports.SettlementStatusPending
	case "initiated":
		return ports.SettlementStatusInitiated
	case "confirmed":
		return ports.SettlementStatusConfirmed
	case "completed":
		return ports.SettlementStatusCompleted
	case "failed":
		return ports.SettlementStatusFailed
	case "cancelled":
		return ports.SettlementStatusCancelled
	default:
		return ports.SettlementStatusPending
	}
}

// validateSettlementRequest validates a settlement request
func (a *OneFinSettlementAdapter) validateSettlementRequest(request ports.SettlementRequest) error {
	if request.SettlementID == "" {
		return fmt.Errorf("settlement ID is required")
	}

	if request.MerchantID == "" {
		return fmt.Errorf("merchant ID is required")
	}

	if request.AmountCrypto.IsZero() || request.AmountCrypto.IsNegative() {
		return fmt.Errorf("invalid crypto amount")
	}

	// Validate against min/max limits
	if !a.config.MinSettlementAmount.IsZero() && request.AmountVND.LessThan(a.config.MinSettlementAmount) {
		return fmt.Errorf("amount below minimum")
	}

	if !a.config.MaxSettlementAmount.IsZero() && request.AmountVND.GreaterThan(a.config.MaxSettlementAmount) {
		return fmt.Errorf("amount exceeds maximum")
	}

	return nil
}

// updateHealthMetrics updates health metrics
func (a *OneFinSettlementAdapter) updateHealthMetrics(isSuccess bool, isFailure bool) {
	a.healthMu.Lock()
	defer a.healthMu.Unlock()

	if isSuccess {
		now := time.Now()
		a.health.LastSuccessfulSettlement = &now
		a.health.IsHealthy = true
		a.health.IsAvailable = true
	}

	if isFailure {
		a.health.FailedSettlements24h++
		// If too many failures, mark as unhealthy
		if a.health.FailedSettlements24h > 10 {
			a.health.IsHealthy = false
			a.health.ErrorMessage = "Too many failed settlements"
		}
	}
}
