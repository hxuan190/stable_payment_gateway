package domain

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	StatusCreated    PaymentStatus = "created"
	StatusPending    PaymentStatus = "pending"
	StatusConfirming PaymentStatus = "confirming"
	StatusCompleted  PaymentStatus = "completed"
	StatusExpired    PaymentStatus = "expired"
	StatusFailed     PaymentStatus = "failed"
)

// Chain represents a blockchain network
type Chain string

const (
	ChainSolana   Chain = "solana"
	ChainBSC      Chain = "bsc"
	ChainEthereum Chain = "ethereum"
)

// Payment represents a payment request in the system
type Payment struct {
	ID         string
	MerchantID string

	// Payment details
	AmountVND    decimal.Decimal
	AmountCrypto decimal.Decimal
	Currency     string
	Chain        Chain
	ExchangeRate decimal.Decimal

	// Merchant reference
	OrderID     sql.NullString
	Description sql.NullString
	CallbackURL sql.NullString

	// Payment status
	Status PaymentStatus

	// Blockchain transaction details
	TxHash          sql.NullString
	TxConfirmations sql.NullInt32
	FromAddress     sql.NullString

	// Payment reference (used in memo)
	PaymentReference  string
	DestinationWallet string

	// Timing
	ExpiresAt   time.Time
	PaidAt      sql.NullTime
	ConfirmedAt sql.NullTime

	// Fee calculation
	FeePercentage decimal.Decimal
	FeeVND        decimal.Decimal
	NetAmountVND  decimal.Decimal

	// Status tracking
	FailureReason sql.NullString

	// Timestamps
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime
}

// IsExpired returns true if the payment has expired
func (p *Payment) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}

// IsCompleted returns true if the payment is completed
func (p *Payment) IsCompleted() bool {
	return p.Status == StatusCompleted
}

// IsPending returns true if the payment is pending confirmation
func (p *Payment) IsPending() bool {
	return p.Status == StatusPending || p.Status == StatusConfirming
}

// CanBeConfirmed returns true if the payment can be confirmed
func (p *Payment) CanBeConfirmed() bool {
	return (p.Status == StatusCreated || p.Status == StatusPending) && !p.IsExpired()
}

// CalculateFee calculates the fee and net amount based on the amount and fee percentage
func (p *Payment) CalculateFee() {
	p.FeeVND = p.AmountVND.Mul(p.FeePercentage)
	p.NetAmountVND = p.AmountVND.Sub(p.FeeVND)
}

// GetTxHash returns the transaction hash if available
func (p *Payment) GetTxHash() string {
	if p.TxHash.Valid {
		return p.TxHash.String
	}
	return ""
}

// GetOrderID returns the order ID if available
func (p *Payment) GetOrderID() string {
	if p.OrderID.Valid {
		return p.OrderID.String
	}
	return ""
}

// GetCallbackURL returns the callback URL if available
func (p *Payment) GetCallbackURL() string {
	if p.CallbackURL.Valid {
		return p.CallbackURL.String
	}
	return ""
}
