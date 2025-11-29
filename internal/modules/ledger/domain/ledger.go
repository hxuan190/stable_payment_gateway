package domain

import (
	"database/sql"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/pkg/database"
	"github.com/shopspring/decimal"
)

// EntryType represents the type of ledger entry (debit or credit)
type EntryType string

const (
	EntryTypeDebit  EntryType = "debit"
	EntryTypeCredit EntryType = "credit"
)

// ReferenceType represents the type of transaction that created the ledger entry
type ReferenceType string

const (
	ReferenceTypePayment       ReferenceType = "payment"
	ReferenceTypePayout        ReferenceType = "payout"
	ReferenceTypeOTCConversion ReferenceType = "otc_conversion"
	ReferenceTypeFee           ReferenceType = "fee"
	ReferenceTypeRefund        ReferenceType = "refund"
	ReferenceTypeAdjustment    ReferenceType = "adjustment"
)

// LedgerEntry represents a single entry in the double-entry accounting ledger
// This table is append-only and immutable
type LedgerEntry struct {
	ID string `json:"id" db:"id"`

	// Double-entry accounting: every transaction has debit and credit
	DebitAccount  string `json:"debit_account" db:"debit_account" validate:"required,min=1,max=255"`
	CreditAccount string `json:"credit_account" db:"credit_account" validate:"required,min=1,max=255"`

	// Amount and currency
	Amount   decimal.Decimal `json:"amount" db:"amount" validate:"required,gt=0"`
	Currency string          `json:"currency" db:"currency" validate:"required,min=2,max=10"`

	// Reference to the source transaction
	ReferenceType ReferenceType `json:"reference_type" db:"reference_type" validate:"required,oneof=payment payout otc_conversion fee refund adjustment"`
	ReferenceID   string        `json:"reference_id" db:"reference_id" validate:"required,uuid"`

	// Associated merchant (if applicable)
	MerchantID sql.NullString `json:"merchant_id,omitempty" db:"merchant_id"`

	// Transaction description
	Description string `json:"description" db:"description" validate:"required,min=1"`

	// Entry pair ID (links debit and credit entries that are part of same transaction)
	TransactionGroup string `json:"transaction_group" db:"transaction_group" validate:"required,uuid"`

	// Entry type: debit or credit
	EntryType EntryType `json:"entry_type" db:"entry_type" validate:"required,oneof=debit credit"`

	// Metadata for additional context
	Metadata database.JSONBMap `json:"metadata,omitempty" db:"metadata"`

	// Timestamp - ledger entries are immutable (no updated_at or deleted_at)
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// IsDebit returns true if this is a debit entry
func (l *LedgerEntry) IsDebit() bool {
	return l.EntryType == EntryTypeDebit
}

// IsCredit returns true if this is a credit entry
func (l *LedgerEntry) IsCredit() bool {
	return l.EntryType == EntryTypeCredit
}

// GetMerchantID returns the merchant ID if available
func (l *LedgerEntry) GetMerchantID() string {
	if l.MerchantID.Valid {
		return l.MerchantID.String
	}
	return ""
}

// LedgerEntryPair represents a pair of ledger entries (debit and credit) for a transaction
type LedgerEntryPair struct {
	DebitEntry  *LedgerEntry
	CreditEntry *LedgerEntry
}

// Validate validates that the ledger entry pair is balanced
func (p *LedgerEntryPair) Validate() bool {
	if p.DebitEntry == nil || p.CreditEntry == nil {
		return false
	}

	// Check that amounts match
	if !p.DebitEntry.Amount.Equal(p.CreditEntry.Amount) {
		return false
	}

	// Check that currencies match
	if p.DebitEntry.Currency != p.CreditEntry.Currency {
		return false
	}

	// Check that transaction groups match
	if p.DebitEntry.TransactionGroup != p.CreditEntry.TransactionGroup {
		return false
	}

	// Check that one is debit and one is credit
	if p.DebitEntry.EntryType == p.CreditEntry.EntryType {
		return false
	}

	// Check that accounts are different
	if p.DebitEntry.DebitAccount == p.CreditEntry.CreditAccount {
		return false
	}

	return true
}
