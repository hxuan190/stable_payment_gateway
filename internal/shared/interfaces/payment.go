package interfaces

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// PaymentInfo contains basic payment information for cross-module access
type PaymentInfo struct {
	ID             string
	MerchantID     string
	AmountVND      decimal.Decimal
	AmountCrypto   decimal.Decimal
	Chain          string
	Token          string
	Status         string
	WalletAddress  string
	TxHash         string
	Memo           string
	ExchangeRate   decimal.Decimal
	ExpiresAt      time.Time
	ConfirmedAt    *time.Time
	CreatedAt      time.Time
}

// PaymentReader provides read-only access to payment data
type PaymentReader interface {
	// GetByID retrieves payment by ID
	GetByID(ctx context.Context, id string) (*PaymentInfo, error)

	// GetByMemo retrieves payment by memo (used by blockchain listener)
	GetByMemo(ctx context.Context, memo string) (*PaymentInfo, error)

	// GetByTxHash retrieves payment by transaction hash
	GetByTxHash(ctx context.Context, txHash string) (*PaymentInfo, error)

	// ListByMerchant lists payments for a merchant
	ListByMerchant(ctx context.Context, merchantID string, limit, offset int) ([]*PaymentInfo, error)

	// IsExpired checks if a payment has expired
	IsExpired(ctx context.Context, paymentID string) (bool, error)
}

// PaymentWriter provides write access to payment data
type PaymentWriter interface {
	// UpdateStatus updates payment status
	UpdateStatus(ctx context.Context, paymentID string, status string) error

	// ConfirmPayment confirms a payment with transaction hash
	ConfirmPayment(ctx context.Context, paymentID string, txHash string) error

	// MarkExpired marks a payment as expired
	MarkExpired(ctx context.Context, paymentID string) error
}
