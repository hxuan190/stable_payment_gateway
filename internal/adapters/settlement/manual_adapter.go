package settlement

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/hxuan190/stable_payment_gateway/internal/ports"
)

// ManualSettlementAdapter implements the SettlementProvider interface for manual OTC settlements
// This adapter manages the workflow where the ops team manually executes bank transfers
// and updates the system when settlements are complete
type ManualSettlementAdapter struct {
	// In-memory storage for pending settlements (in production, this would use database)
	settlements   map[string]*ports.SettlementResponse
	settlementsMu sync.RWMutex

	// Configuration
	config ports.SettlementProviderConfig

	// Exchange rate provider (for MVP, this could be a simple API client)
	exchangeRateProvider ExchangeRateProvider

	// Health tracking
	health   ports.ProviderHealth
	healthMu sync.RWMutex
}

// ExchangeRateProvider interface for fetching exchange rates
type ExchangeRateProvider interface {
	GetVNDPerUSD(ctx context.Context) (decimal.Decimal, error)
	GetUSDPerCrypto(ctx context.Context, cryptoSymbol string) (decimal.Decimal, error)
}

// NewManualSettlementAdapter creates a new manual settlement adapter
func NewManualSettlementAdapter(config ports.SettlementProviderConfig, rateProvider ExchangeRateProvider) (*ManualSettlementAdapter, error) {
	if config.ProviderType != ports.SettlementProviderManual {
		return nil, fmt.Errorf("invalid provider type: expected %s, got %s", ports.SettlementProviderManual, config.ProviderType)
	}

	adapter := &ManualSettlementAdapter{
		config:               config,
		settlements:          make(map[string]*ports.SettlementResponse),
		exchangeRateProvider: rateProvider,
		health: ports.ProviderHealth{
			IsHealthy:   true,
			IsAvailable: true,
			// For manual settlements, liquidity is theoretically unlimited
			AvailableLiquidityVND: decimal.NewFromInt(1000000000), // 1 billion VND placeholder
		},
	}

	return adapter, nil
}

// GetProviderType returns the type of settlement provider
func (a *ManualSettlementAdapter) GetProviderType() ports.SettlementProviderType {
	return ports.SettlementProviderManual
}

// GetProviderName returns a human-readable name for the provider
func (a *ManualSettlementAdapter) GetProviderName() string {
	if a.config.ProviderName != "" {
		return a.config.ProviderName
	}
	return "Manual OTC Settlement"
}

// InitiateSettlement starts a new settlement operation
// For manual settlements, this creates a pending settlement for ops team approval
func (a *ManualSettlementAdapter) InitiateSettlement(ctx context.Context, request ports.SettlementRequest) (*ports.SettlementResponse, error) {
	// Validate request
	if err := a.validateSettlementRequest(request); err != nil {
		return nil, fmt.Errorf("invalid settlement request: %w", err)
	}

	// Create settlement response
	now := time.Now()
	response := &ports.SettlementResponse{
		SettlementID:        request.SettlementID,
		ProviderReferenceID: fmt.Sprintf("MANUAL-%s", uuid.New().String()[:8]),
		Status:              ports.SettlementStatusPending,
		AmountCrypto:        request.AmountCrypto,
		AmountVND:           request.AmountVND,
		ExchangeRate:        request.ExchangeRate,
		Fee:                 a.calculateFee(request.AmountVND),
		InitiatedAt:         now,
		Metadata: map[string]interface{}{
			"merchant_id":         request.MerchantID,
			"bank_account_number": request.BankAccountNumber,
			"bank_account_name":   request.BankAccountName,
			"bank_name":           request.BankName,
			"crypto_symbol":       request.CryptoSymbol,
			"requested_at":        request.RequestedAt,
		},
	}

	// Store settlement (in production, save to database)
	a.settlementsMu.Lock()
	a.settlements[request.SettlementID] = response
	a.settlementsMu.Unlock()

	a.updateHealthMetrics(true, false)

	return response, nil
}

// ConfirmSettlement confirms that a settlement has been completed
// For manual settlements, this is called by ops team after bank transfer
func (a *ManualSettlementAdapter) ConfirmSettlement(ctx context.Context, settlementID string) (*ports.SettlementResponse, error) {
	a.settlementsMu.Lock()
	defer a.settlementsMu.Unlock()

	settlement, exists := a.settlements[settlementID]
	if !exists {
		return nil, fmt.Errorf("settlement not found: %s", settlementID)
	}

	// Check if already confirmed/completed
	if settlement.Status == ports.SettlementStatusCompleted {
		return settlement, nil
	}

	// Update settlement status
	now := time.Now()
	settlement.Status = ports.SettlementStatusCompleted
	settlement.CompletedAt = &now
	settlement.ConfirmedAt = &now

	a.updateHealthMetrics(false, true)

	return settlement, nil
}

// CancelSettlement cancels a pending settlement
func (a *ManualSettlementAdapter) CancelSettlement(ctx context.Context, settlementID string) (*ports.SettlementResponse, error) {
	a.settlementsMu.Lock()
	defer a.settlementsMu.Unlock()

	settlement, exists := a.settlements[settlementID]
	if !exists {
		return nil, fmt.Errorf("settlement not found: %s", settlementID)
	}

	// Can only cancel pending settlements
	if settlement.Status != ports.SettlementStatusPending {
		return nil, fmt.Errorf("cannot cancel settlement with status: %s", settlement.Status)
	}

	settlement.Status = ports.SettlementStatusCancelled
	settlement.ErrorMessage = "Cancelled by user"

	return settlement, nil
}

// GetSettlementStatus retrieves the current status of a settlement
func (a *ManualSettlementAdapter) GetSettlementStatus(ctx context.Context, settlementID string) (*ports.SettlementResponse, error) {
	a.settlementsMu.RLock()
	defer a.settlementsMu.RUnlock()

	settlement, exists := a.settlements[settlementID]
	if !exists {
		return nil, fmt.Errorf("settlement not found: %s", settlementID)
	}

	return settlement, nil
}

// GetExchangeRate retrieves the current exchange rate quote
func (a *ManualSettlementAdapter) GetExchangeRate(ctx context.Context, cryptoSymbol string, amountCrypto decimal.Decimal) (*ports.ExchangeRateQuote, error) {
	if a.exchangeRateProvider == nil {
		return nil, fmt.Errorf("exchange rate provider not configured")
	}

	// Get VND/USD rate
	vndPerUSD, err := a.exchangeRateProvider.GetVNDPerUSD(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get VND/USD rate: %w", err)
	}

	// Get USD price of crypto
	usdPerCrypto, err := a.exchangeRateProvider.GetUSDPerCrypto(ctx, cryptoSymbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get USD/%s rate: %w", cryptoSymbol, err)
	}

	// Calculate effective rate (VND per crypto)
	effectiveRate := vndPerUSD.Mul(usdPerCrypto)

	// Apply spread
	spread := a.config.DefaultSpread
	if spread.IsZero() {
		spread = decimal.NewFromFloat(0.01) // Default 1% spread
	}
	effectiveRate = effectiveRate.Mul(decimal.NewFromInt(1).Sub(spread))

	quote := &ports.ExchangeRateQuote{
		CryptoSymbol:  cryptoSymbol,
		VNDPerUSD:     vndPerUSD,
		USDPerCrypto:  usdPerCrypto,
		EffectiveRate: effectiveRate,
		ValidUntil:    time.Now().Add(5 * time.Minute), // Quote valid for 5 minutes
		ProviderName:  a.GetProviderName(),
		Spread:        spread,
	}

	return quote, nil
}

// GetAvailableLiquidity returns the provider's current available liquidity
func (a *ManualSettlementAdapter) GetAvailableLiquidity(ctx context.Context) (decimal.Decimal, error) {
	a.healthMu.RLock()
	defer a.healthMu.RUnlock()

	// For manual settlements, liquidity is managed by ops team
	// Return a large amount to indicate availability
	return a.health.AvailableLiquidityVND, nil
}

// IsAvailable checks if the provider is currently available for settlements
func (a *ManualSettlementAdapter) IsAvailable(ctx context.Context) bool {
	a.healthMu.RLock()
	defer a.healthMu.RUnlock()

	return a.health.IsAvailable
}

// GetProviderHealth returns health status and metrics
func (a *ManualSettlementAdapter) GetProviderHealth(ctx context.Context) ports.ProviderHealth {
	a.healthMu.RLock()
	defer a.healthMu.RUnlock()

	// Calculate pending settlements
	a.settlementsMu.RLock()
	pendingCount := 0
	for _, settlement := range a.settlements {
		if settlement.Status == ports.SettlementStatusPending {
			pendingCount++
		}
	}
	a.settlementsMu.RUnlock()

	health := a.health
	health.PendingSettlements = pendingCount

	return health
}

// validateSettlementRequest validates a settlement request
func (a *ManualSettlementAdapter) validateSettlementRequest(request ports.SettlementRequest) error {
	if request.SettlementID == "" {
		return fmt.Errorf("settlement ID is required")
	}

	if request.MerchantID == "" {
		return fmt.Errorf("merchant ID is required")
	}

	if request.AmountCrypto.IsZero() || request.AmountCrypto.IsNegative() {
		return fmt.Errorf("invalid crypto amount")
	}

	if request.AmountVND.IsZero() || request.AmountVND.IsNegative() {
		return fmt.Errorf("invalid VND amount")
	}

	// Validate against min/max limits
	if !a.config.MinSettlementAmount.IsZero() && request.AmountVND.LessThan(a.config.MinSettlementAmount) {
		return fmt.Errorf("amount below minimum: %s < %s", request.AmountVND, a.config.MinSettlementAmount)
	}

	if !a.config.MaxSettlementAmount.IsZero() && request.AmountVND.GreaterThan(a.config.MaxSettlementAmount) {
		return fmt.Errorf("amount exceeds maximum: %s > %s", request.AmountVND, a.config.MaxSettlementAmount)
	}

	if request.BankAccountNumber == "" {
		return fmt.Errorf("bank account number is required")
	}

	return nil
}

// calculateFee calculates the settlement fee
func (a *ManualSettlementAdapter) calculateFee(amountVND decimal.Decimal) decimal.Decimal {
	// For manual settlements, fee is typically 0 or a small fixed amount
	// In production, this would be configurable
	return decimal.Zero
}

// updateHealthMetrics updates health metrics
func (a *ManualSettlementAdapter) updateHealthMetrics(isPending bool, isCompleted bool) {
	a.healthMu.Lock()
	defer a.healthMu.Unlock()

	if isPending {
		a.health.PendingSettlements++
	}

	if isCompleted {
		now := time.Now()
		a.health.LastSuccessfulSettlement = &now
		if a.health.PendingSettlements > 0 {
			a.health.PendingSettlements--
		}
	}
}
