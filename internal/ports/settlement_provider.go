package ports

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// SettlementProviderType represents the type of settlement provider
type SettlementProviderType string

const (
	SettlementProviderManual SettlementProviderType = "manual"
	SettlementProviderOneFin SettlementProviderType = "onefin"
	SettlementProviderBinanceP2P SettlementProviderType = "binance_p2p"
)

// SettlementStatus represents the status of a settlement operation
type SettlementStatus string

const (
	SettlementStatusPending    SettlementStatus = "pending"
	SettlementStatusInitiated  SettlementStatus = "initiated"
	SettlementStatusConfirmed  SettlementStatus = "confirmed"
	SettlementStatusCompleted  SettlementStatus = "completed"
	SettlementStatusFailed     SettlementStatus = "failed"
	SettlementStatusCancelled  SettlementStatus = "cancelled"
)

// SettlementRequest contains details for initiating a settlement
type SettlementRequest struct {
	// SettlementID is the unique identifier for this settlement
	SettlementID string

	// MerchantID is the merchant requesting the settlement
	MerchantID string

	// AmountCrypto is the amount in cryptocurrency to settle
	AmountCrypto decimal.Decimal

	// CryptoSymbol is the cryptocurrency symbol (e.g., "USDT", "USDC")
	CryptoSymbol string

	// AmountVND is the expected VND amount to receive
	AmountVND decimal.Decimal

	// ExchangeRate is the agreed VND/USD rate
	ExchangeRate decimal.Decimal

	// BankAccountNumber is the merchant's bank account for VND settlement
	BankAccountNumber string

	// BankAccountName is the merchant's bank account name
	BankAccountName string

	// BankName is the name of the bank
	BankName string

	// RequestedAt is when the settlement was requested
	RequestedAt time.Time

	// Metadata contains additional provider-specific data
	Metadata map[string]interface{}
}

// SettlementResponse contains the result of a settlement operation
type SettlementResponse struct {
	// SettlementID is the unique identifier for this settlement
	SettlementID string

	// ProviderReferenceID is the settlement reference from the provider's system
	ProviderReferenceID string

	// Status is the current status of the settlement
	Status SettlementStatus

	// AmountCrypto is the actual crypto amount settled
	AmountCrypto decimal.Decimal

	// AmountVND is the actual VND amount transferred
	AmountVND decimal.Decimal

	// ExchangeRate is the actual rate used
	ExchangeRate decimal.Decimal

	// Fee is the settlement fee charged
	Fee decimal.Decimal

	// InitiatedAt is when the settlement was initiated
	InitiatedAt time.Time

	// ConfirmedAt is when the settlement was confirmed by the provider
	ConfirmedAt *time.Time

	// CompletedAt is when the VND transfer was completed
	CompletedAt *time.Time

	// ErrorMessage contains error details if settlement failed
	ErrorMessage string

	// Metadata contains additional provider-specific response data
	Metadata map[string]interface{}
}

// ExchangeRateQuote contains current exchange rate information
type ExchangeRateQuote struct {
	// CryptoSymbol is the cryptocurrency symbol
	CryptoSymbol string

	// VNDPerUSD is the VND/USD exchange rate
	VNDPerUSD decimal.Decimal

	// USDPerCrypto is the USD price of the cryptocurrency
	USDPerCrypto decimal.Decimal

	// EffectiveRate is the final VND per crypto rate
	EffectiveRate decimal.Decimal

	// ValidUntil is when this quote expires
	ValidUntil time.Time

	// ProviderName is the settlement provider offering this rate
	ProviderName string

	// Spread is the provider's spread/markup
	Spread decimal.Decimal
}

// SettlementProvider defines the interface for OTC settlement providers
// This is the PORT in the Hexagonal Architecture (Ports & Adapters pattern)
// Implementations (Adapters) include: ManualSettlement, OneFinSettlement, BinanceP2PSettlement
type SettlementProvider interface {
	// GetProviderType returns the type of settlement provider
	GetProviderType() SettlementProviderType

	// GetProviderName returns a human-readable name for the provider
	GetProviderName() string

	// InitiateSettlement starts a new settlement operation
	// For manual providers, this creates a pending settlement for ops team approval
	// For API providers, this calls the provider's API to initiate the settlement
	InitiateSettlement(ctx context.Context, request SettlementRequest) (*SettlementResponse, error)

	// ConfirmSettlement confirms that a settlement has been completed
	// For manual providers, this is called by ops team after bank transfer
	// For API providers, this polls the provider's API for completion status
	ConfirmSettlement(ctx context.Context, settlementID string) (*SettlementResponse, error)

	// CancelSettlement cancels a pending settlement
	CancelSettlement(ctx context.Context, settlementID string) (*SettlementResponse, error)

	// GetSettlementStatus retrieves the current status of a settlement
	GetSettlementStatus(ctx context.Context, settlementID string) (*SettlementResponse, error)

	// GetExchangeRate retrieves the current exchange rate quote
	// This is used to calculate VND amount before initiating settlement
	GetExchangeRate(ctx context.Context, cryptoSymbol string, amountCrypto decimal.Decimal) (*ExchangeRateQuote, error)

	// GetAvailableLiquidity returns the provider's current available liquidity
	// Returns the amount in VND that can be settled immediately
	GetAvailableLiquidity(ctx context.Context) (decimal.Decimal, error)

	// IsAvailable checks if the provider is currently available for settlements
	IsAvailable(ctx context.Context) bool

	// GetProviderHealth returns health status and metrics
	GetProviderHealth(ctx context.Context) ProviderHealth
}

// ProviderHealth contains health and performance metrics for a settlement provider
type ProviderHealth struct {
	// IsHealthy indicates if the provider is functioning correctly
	IsHealthy bool

	// IsAvailable indicates if the provider can accept new settlements
	IsAvailable bool

	// AvailableLiquidityVND is the current available liquidity in VND
	AvailableLiquidityVND decimal.Decimal

	// LastSuccessfulSettlement is when the last settlement completed successfully
	LastSuccessfulSettlement *time.Time

	// PendingSettlements is the count of pending settlements
	PendingSettlements int

	// FailedSettlements24h is the count of failed settlements in the last 24 hours
	FailedSettlements24h int

	// AverageSettlementTimeMinutes is the average time to complete settlements
	AverageSettlementTimeMinutes int

	// ErrorMessage contains error details if provider is unhealthy
	ErrorMessage string
}

// SettlementProviderConfig holds common configuration for settlement providers
type SettlementProviderConfig struct {
	// ProviderType identifies the settlement provider
	ProviderType SettlementProviderType

	// ProviderName is a human-readable name
	ProviderName string

	// APIURL is the provider's API endpoint (for API-based providers)
	APIURL string

	// APIKey is the authentication key
	APIKey string

	// APISecret is the authentication secret
	APISecret string

	// EnableAutoSettlement determines if settlements are automatic or require approval
	EnableAutoSettlement bool

	// MinSettlementAmount is the minimum amount for settlements
	MinSettlementAmount decimal.Decimal

	// MaxSettlementAmount is the maximum amount for settlements
	MaxSettlementAmount decimal.Decimal

	// DefaultSpread is the default spread/markup percentage
	DefaultSpread decimal.Decimal

	// Timeout is the timeout for API calls
	TimeoutSeconds int
}
