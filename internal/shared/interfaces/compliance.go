package interfaces

import (
	"context"

	"github.com/shopspring/decimal"
)

// TransactionScreeningRequest represents a request to screen a transaction
type TransactionScreeningRequest struct {
	TransactionID   string
	MerchantID      string
	Amount          decimal.Decimal
	Currency        string
	WalletAddress   string
	Chain           string
	TransactionType string // "payment" or "payout"
}

// TransactionScreeningResult represents the result of transaction screening
type TransactionScreeningResult struct {
	Approved      bool
	RiskScore     float64
	RiskLevel     string // "low", "medium", "high"
	Reasons       []string
	RequiresReview bool
}

// ComplianceChecker provides compliance checking capabilities
type ComplianceChecker interface {
	// ScreenTransaction screens a transaction for AML/compliance
	ScreenTransaction(ctx context.Context, req TransactionScreeningRequest) (*TransactionScreeningResult, error)

	// CheckTransactionLimit checks if transaction is within merchant's limits
	CheckTransactionLimit(ctx context.Context, merchantID string, amount decimal.Decimal, currency string) error

	// ValidateTravelRule validates FATF Travel Rule requirements
	ValidateTravelRule(ctx context.Context, req TransactionScreeningRequest) error
}
