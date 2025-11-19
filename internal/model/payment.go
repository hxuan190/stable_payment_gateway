package model

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusCreated           PaymentStatus = "created"
	PaymentStatusPending           PaymentStatus = "pending"
	PaymentStatusPendingCompliance PaymentStatus = "pending_compliance"
	PaymentStatusConfirming        PaymentStatus = "confirming"
	PaymentStatusCompleted         PaymentStatus = "completed"
	PaymentStatusExpired           PaymentStatus = "expired"
	PaymentStatusFailed            PaymentStatus = "failed"
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
	ID         string `json:"id" db:"id"`
	MerchantID string `json:"merchant_id" db:"merchant_id" validate:"required,uuid"`

	// Payment details
	AmountVND     decimal.Decimal `json:"amount_vnd" db:"amount_vnd" validate:"required,gt=0"`
	AmountCrypto  decimal.Decimal `json:"amount_crypto" db:"amount_crypto" validate:"required,gt=0"`
	Currency      string          `json:"currency" db:"currency" validate:"required,oneof=USDT USDC BUSD"`
	Chain         Chain           `json:"chain" db:"chain" validate:"required,oneof=solana bsc ethereum"`
	ExchangeRate  decimal.Decimal `json:"exchange_rate" db:"exchange_rate" validate:"required,gt=0"`

	// Merchant reference
	OrderID     sql.NullString `json:"order_id,omitempty" db:"order_id"`
	Description sql.NullString `json:"description,omitempty" db:"description"`
	CallbackURL sql.NullString `json:"callback_url,omitempty" db:"callback_url" validate:"omitempty,url"`

	// Payment status
	Status PaymentStatus `json:"status" db:"status" validate:"required,oneof=created pending pending_compliance confirming completed expired failed"`

	// Blockchain transaction details
	TxHash          sql.NullString `json:"tx_hash,omitempty" db:"tx_hash"`
	TxConfirmations sql.NullInt32  `json:"tx_confirmations,omitempty" db:"tx_confirmations"`
	FromAddress     sql.NullString `json:"from_address,omitempty" db:"from_address"`

	// Payment reference (used in memo)
	PaymentReference    string `json:"payment_reference" db:"payment_reference" validate:"required,min=8,max=100"`
	DestinationWallet   string `json:"destination_wallet" db:"destination_wallet" validate:"required"`

	// Timing
	ExpiresAt   time.Time    `json:"expires_at" db:"expires_at"`
	PaidAt      sql.NullTime `json:"paid_at,omitempty" db:"paid_at"`
	ConfirmedAt sql.NullTime `json:"confirmed_at,omitempty" db:"confirmed_at"`

	// Compliance timing (FATF Travel Rule)
	ComplianceDeadline    sql.NullTime `json:"compliance_deadline,omitempty" db:"compliance_deadline"`
	ComplianceSubmittedAt sql.NullTime `json:"compliance_submitted_at,omitempty" db:"compliance_submitted_at"`

	// Fee calculation
	FeePercentage decimal.Decimal `json:"fee_percentage" db:"fee_percentage" validate:"gte=0,lte=0.1"`
	FeeVND        decimal.Decimal `json:"fee_vnd" db:"fee_vnd"`
	NetAmountVND  decimal.Decimal `json:"net_amount_vnd" db:"net_amount_vnd"`

	// Status tracking
	FailureReason sql.NullString `json:"failure_reason,omitempty" db:"failure_reason"`

	// Metadata
	Metadata JSONBMap `json:"metadata,omitempty" db:"metadata"`

	// Timestamps
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt sql.NullTime `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsExpired returns true if the payment has expired
func (p *Payment) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}

// IsCompleted returns true if the payment is completed
func (p *Payment) IsCompleted() bool {
	return p.Status == PaymentStatusCompleted
}

// IsPending returns true if the payment is pending confirmation
func (p *Payment) IsPending() bool {
	return p.Status == PaymentStatusPending || p.Status == PaymentStatusPendingCompliance || p.Status == PaymentStatusConfirming
}

// CanBeConfirmed returns true if the payment can be confirmed
func (p *Payment) CanBeConfirmed() bool {
	return (p.Status == PaymentStatusCreated || p.Status == PaymentStatusPending || p.Status == PaymentStatusPendingCompliance) && !p.IsExpired()
}

// IsPendingCompliance returns true if the payment is waiting for Travel Rule data submission
func (p *Payment) IsPendingCompliance() bool {
	return p.Status == PaymentStatusPendingCompliance
}

// IsComplianceExpired returns true if the compliance deadline has passed
func (p *Payment) IsComplianceExpired() bool {
	if !p.ComplianceDeadline.Valid {
		return false
	}
	return time.Now().After(p.ComplianceDeadline.Time)
}

// HasComplianceData returns true if compliance data has been submitted
func (p *Payment) HasComplianceData() bool {
	return p.ComplianceSubmittedAt.Valid
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
