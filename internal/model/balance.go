package model

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// MerchantBalance represents the cached balance state for a merchant
// This is derived from ledger_entries but cached for performance
type MerchantBalance struct {
	ID         string `json:"id" db:"id"`
	MerchantID string `json:"merchant_id" db:"merchant_id" validate:"required,uuid"`

	// VND balances
	PendingVND   decimal.Decimal `json:"pending_vnd" db:"pending_vnd" validate:"gte=0"`
	AvailableVND decimal.Decimal `json:"available_vnd" db:"available_vnd" validate:"gte=0"`
	TotalVND     decimal.Decimal `json:"total_vnd" db:"total_vnd" validate:"gte=0"`

	// Reserved/locked balance (for pending payouts)
	ReservedVND decimal.Decimal `json:"reserved_vnd" db:"reserved_vnd" validate:"gte=0"`

	// Lifetime statistics
	TotalReceivedVND decimal.Decimal `json:"total_received_vnd" db:"total_received_vnd" validate:"gte=0"`
	TotalPaidOutVND  decimal.Decimal `json:"total_paid_out_vnd" db:"total_paid_out_vnd" validate:"gte=0"`
	TotalFeesVND     decimal.Decimal `json:"total_fees_vnd" db:"total_fees_vnd" validate:"gte=0"`

	// Transaction counts
	TotalPaymentsCount int `json:"total_payments_count" db:"total_payments_count" validate:"gte=0"`
	TotalPayoutsCount  int `json:"total_payouts_count" db:"total_payouts_count" validate:"gte=0"`

	// Last transaction timestamps
	LastPaymentAt sql.NullTime `json:"last_payment_at,omitempty" db:"last_payment_at"`
	LastPayoutAt  sql.NullTime `json:"last_payout_at,omitempty" db:"last_payout_at"`

	// Version for optimistic locking
	Version int `json:"version" db:"version"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// HasSufficientBalance returns true if the merchant has enough available balance for a payout
func (b *MerchantBalance) HasSufficientBalance(amount decimal.Decimal) bool {
	return b.AvailableVND.GreaterThanOrEqual(amount)
}

// GetWithdrawableBalance returns the balance that can be withdrawn (available - reserved)
func (b *MerchantBalance) GetWithdrawableBalance() decimal.Decimal {
	withdrawable := b.AvailableVND.Sub(b.ReservedVND)
	if withdrawable.LessThan(decimal.Zero) {
		return decimal.Zero
	}
	return withdrawable
}

// CanWithdraw returns true if the merchant can withdraw the specified amount
func (b *MerchantBalance) CanWithdraw(amount decimal.Decimal) bool {
	return b.GetWithdrawableBalance().GreaterThanOrEqual(amount)
}

// AddPendingBalance adds to the pending balance (when payment is received but not yet converted)
func (b *MerchantBalance) AddPendingBalance(amount decimal.Decimal) {
	b.PendingVND = b.PendingVND.Add(amount)
	b.TotalVND = b.PendingVND.Add(b.AvailableVND)
	b.TotalReceivedVND = b.TotalReceivedVND.Add(amount)
	b.TotalPaymentsCount++
	b.Version++
}

// ConvertPendingToAvailable moves balance from pending to available (after OTC conversion)
func (b *MerchantBalance) ConvertPendingToAvailable(amount decimal.Decimal, fee decimal.Decimal) {
	netAmount := amount.Sub(fee)
	b.PendingVND = b.PendingVND.Sub(amount)
	b.AvailableVND = b.AvailableVND.Add(netAmount)
	b.TotalVND = b.PendingVND.Add(b.AvailableVND)
	b.TotalFeesVND = b.TotalFeesVND.Add(fee)
	b.Version++
}

// ReserveBalance reserves balance for a pending payout
func (b *MerchantBalance) ReserveBalance(amount decimal.Decimal) {
	b.ReservedVND = b.ReservedVND.Add(amount)
	b.Version++
}

// ReleaseReservedBalance releases reserved balance (if payout is cancelled)
func (b *MerchantBalance) ReleaseReservedBalance(amount decimal.Decimal) {
	b.ReservedVND = b.ReservedVND.Sub(amount)
	if b.ReservedVND.LessThan(decimal.Zero) {
		b.ReservedVND = decimal.Zero
	}
	b.Version++
}

// DeductBalance deducts from available balance (when payout is completed)
func (b *MerchantBalance) DeductBalance(amount decimal.Decimal, fee decimal.Decimal) {
	totalDeduction := amount.Add(fee)
	b.AvailableVND = b.AvailableVND.Sub(totalDeduction)
	b.ReservedVND = b.ReservedVND.Sub(totalDeduction)
	b.TotalVND = b.PendingVND.Add(b.AvailableVND)
	b.TotalPaidOutVND = b.TotalPaidOutVND.Add(amount)
	b.TotalFeesVND = b.TotalFeesVND.Add(fee)
	b.TotalPayoutsCount++
	b.Version++
}

// Validate ensures the balance state is consistent
func (b *MerchantBalance) Validate() bool {
	// All balances should be non-negative
	if b.PendingVND.LessThan(decimal.Zero) || b.AvailableVND.LessThan(decimal.Zero) ||
		b.ReservedVND.LessThan(decimal.Zero) {
		return false
	}

	// Total should equal pending + available
	expectedTotal := b.PendingVND.Add(b.AvailableVND)
	if !b.TotalVND.Equal(expectedTotal) {
		return false
	}

	// Reserved cannot exceed available
	if b.ReservedVND.GreaterThan(b.AvailableVND) {
		return false
	}

	return true
}
