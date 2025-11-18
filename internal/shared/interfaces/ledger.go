package interfaces

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// BalanceInfo contains balance information
type BalanceInfo struct {
	MerchantID     string
	AvailableVND   decimal.Decimal
	PendingVND     decimal.Decimal
	TotalReceived  decimal.Decimal
	TotalPaidOut   decimal.Decimal
	LastUpdatedAt  time.Time
}

// LedgerReader provides read access to ledger/balance data
type LedgerReader interface {
	// GetBalance retrieves merchant balance
	GetBalance(ctx context.Context, merchantID string) (*BalanceInfo, error)

	// HasSufficientBalance checks if merchant has sufficient available balance
	HasSufficientBalance(ctx context.Context, merchantID string, amount decimal.Decimal) (bool, error)
}

// LedgerEntryRequest represents a request to create ledger entries
type LedgerEntryRequest struct {
	ReferenceType string          // "payment", "payout", "fee"
	ReferenceID   string
	MerchantID    string
	Amount        decimal.Decimal
	Currency      string
	Description   string
}

// LedgerWriter provides write access to ledger
type LedgerWriter interface {
	// RecordPaymentConfirmed records ledger entries when payment is confirmed
	RecordPaymentConfirmed(ctx context.Context, req LedgerEntryRequest) error

	// RecordPayoutCompleted records ledger entries when payout is completed
	RecordPayoutCompleted(ctx context.Context, req LedgerEntryRequest) error

	// RecordFee records fee-related ledger entries
	RecordFee(ctx context.Context, req LedgerEntryRequest) error
}
