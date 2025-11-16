package model

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// PayoutStatus represents the status of a payout request
type PayoutStatus string

const (
	PayoutStatusRequested  PayoutStatus = "requested"
	PayoutStatusApproved   PayoutStatus = "approved"
	PayoutStatusProcessing PayoutStatus = "processing"
	PayoutStatusCompleted  PayoutStatus = "completed"
	PayoutStatusRejected   PayoutStatus = "rejected"
	PayoutStatusFailed     PayoutStatus = "failed"
)

// Payout represents a merchant withdrawal request
type Payout struct {
	ID         string `json:"id" db:"id"`
	MerchantID string `json:"merchant_id" db:"merchant_id" validate:"required,uuid"`

	// Payout amount
	AmountVND    decimal.Decimal `json:"amount_vnd" db:"amount_vnd" validate:"required,gt=0"`
	FeeVND       decimal.Decimal `json:"fee_vnd" db:"fee_vnd" validate:"gte=0"`
	NetAmountVND decimal.Decimal `json:"net_amount_vnd" db:"net_amount_vnd" validate:"required,gt=0"`

	// Bank transfer details
	BankAccountName   string         `json:"bank_account_name" db:"bank_account_name" validate:"required,min=2,max=255"`
	BankAccountNumber string         `json:"bank_account_number" db:"bank_account_number" validate:"required,min=8,max=50"`
	BankName          string         `json:"bank_name" db:"bank_name" validate:"required,min=2,max=100"`
	BankBranch        sql.NullString `json:"bank_branch,omitempty" db:"bank_branch"`

	// Status
	Status PayoutStatus `json:"status" db:"status" validate:"required,oneof=requested approved processing completed rejected failed"`

	// Approval workflow
	RequestedBy      string         `json:"requested_by" db:"requested_by" validate:"required,uuid"`
	ApprovedBy       sql.NullString `json:"approved_by,omitempty" db:"approved_by"`
	ApprovedAt       sql.NullTime   `json:"approved_at,omitempty" db:"approved_at"`
	RejectionReason  sql.NullString `json:"rejection_reason,omitempty" db:"rejection_reason"`

	// Processing details
	ProcessedBy    sql.NullString `json:"processed_by,omitempty" db:"processed_by"`
	ProcessedAt    sql.NullTime   `json:"processed_at,omitempty" db:"processed_at"`
	CompletionDate sql.NullTime   `json:"completion_date,omitempty" db:"completion_date"`

	// Bank transfer reference
	BankReferenceNumber   sql.NullString `json:"bank_reference_number,omitempty" db:"bank_reference_number"`
	TransactionReceiptURL sql.NullString `json:"transaction_receipt_url,omitempty" db:"transaction_receipt_url" validate:"omitempty,url"`

	// Failure tracking
	FailureReason sql.NullString `json:"failure_reason,omitempty" db:"failure_reason"`
	RetryCount    int            `json:"retry_count" db:"retry_count"`

	// Metadata
	Notes    sql.NullString `json:"notes,omitempty" db:"notes"`
	Metadata JSONBMap       `json:"metadata,omitempty" db:"metadata"`

	// Timestamps
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt sql.NullTime `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsApproved returns true if the payout is approved
func (p *Payout) IsApproved() bool {
	return p.Status == PayoutStatusApproved || p.Status == PayoutStatusProcessing || p.Status == PayoutStatusCompleted
}

// IsCompleted returns true if the payout is completed
func (p *Payout) IsCompleted() bool {
	return p.Status == PayoutStatusCompleted
}

// IsPending returns true if the payout is pending approval or processing
func (p *Payout) IsPending() bool {
	return p.Status == PayoutStatusRequested || p.Status == PayoutStatusApproved || p.Status == PayoutStatusProcessing
}

// CanBeApproved returns true if the payout can be approved
func (p *Payout) CanBeApproved() bool {
	return p.Status == PayoutStatusRequested
}

// CanBeRejected returns true if the payout can be rejected
func (p *Payout) CanBeRejected() bool {
	return p.Status == PayoutStatusRequested
}

// CanBeProcessed returns true if the payout can be processed
func (p *Payout) CanBeProcessed() bool {
	return p.Status == PayoutStatusApproved
}

// CalculateNetAmount calculates the net amount (amount - fee)
func (p *Payout) CalculateNetAmount() {
	p.NetAmountVND = p.AmountVND.Sub(p.FeeVND)
}

// GetBankReferenceNumber returns the bank reference number if available
func (p *Payout) GetBankReferenceNumber() string {
	if p.BankReferenceNumber.Valid {
		return p.BankReferenceNumber.String
	}
	return ""
}

// GetNotes returns the notes if available
func (p *Payout) GetNotes() string {
	if p.Notes.Valid {
		return p.Notes.String
	}
	return ""
}
