package domain

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// KYCStatus represents the KYC verification status
type KYCStatus string

const (
	KYCStatusPending   KYCStatus = "pending"
	KYCStatusApproved  KYCStatus = "approved"
	KYCStatusRejected  KYCStatus = "rejected"
	KYCStatusSuspended KYCStatus = "suspended"
)

// KYCTier represents the KYC tier level for compliance limits
type KYCTier string

const (
	KYCTier1 KYCTier = "tier1" // $5,000/month - basic verification
	KYCTier2 KYCTier = "tier2" // $50,000/month - enhanced verification
	KYCTier3 KYCTier = "tier3" // $500,000/month - full business verification
)

// MerchantStatus represents the merchant account status
type MerchantStatus string

const (
	MerchantStatusActive    MerchantStatus = "active"
	MerchantStatusSuspended MerchantStatus = "suspended"
	MerchantStatusClosed    MerchantStatus = "closed"
)

// Merchant represents a merchant in the payment gateway system
type Merchant struct {
	ID                    string         `json:"id" db:"id"`
	Email                 string         `json:"email" db:"email" validate:"required,email"`
	BusinessName          string         `json:"business_name" db:"business_name" validate:"required,min=2,max=255"`
	BusinessTaxID         sql.NullString `json:"business_tax_id,omitempty" db:"business_tax_id"`
	BusinessLicenseNumber sql.NullString `json:"business_license_number,omitempty" db:"business_license_number"`
	OwnerFullName         string         `json:"owner_full_name" db:"owner_full_name" validate:"required,min=2,max=255"`
	OwnerIDCard           sql.NullString `json:"owner_id_card,omitempty" db:"owner_id_card"`
	PhoneNumber           sql.NullString `json:"phone_number,omitempty" db:"phone_number" validate:"omitempty,e164"`

	// Address information
	BusinessAddress sql.NullString `json:"business_address,omitempty" db:"business_address"`
	City            sql.NullString `json:"city,omitempty" db:"city"`
	District        sql.NullString `json:"district,omitempty" db:"district"`

	// Bank account information for payouts
	BankAccountName   sql.NullString `json:"bank_account_name,omitempty" db:"bank_account_name"`
	BankAccountNumber sql.NullString `json:"bank_account_number,omitempty" db:"bank_account_number"`
	BankName          sql.NullString `json:"bank_name,omitempty" db:"bank_name"`
	BankBranch        sql.NullString `json:"bank_branch,omitempty" db:"bank_branch"`

	// KYC status
	KYCStatus          KYCStatus      `json:"kyc_status" db:"kyc_status" validate:"required,oneof=pending approved rejected suspended"`
	KYCSubmittedAt     sql.NullTime   `json:"kyc_submitted_at,omitempty" db:"kyc_submitted_at"`
	KYCApprovedAt      sql.NullTime   `json:"kyc_approved_at,omitempty" db:"kyc_approved_at"`
	KYCApprovedBy      sql.NullString `json:"kyc_approved_by,omitempty" db:"kyc_approved_by"`
	KYCRejectionReason sql.NullString `json:"kyc_rejection_reason,omitempty" db:"kyc_rejection_reason"`

	// KYC Tier and monthly limits (MVP v1.1 compliance)
	KYCTier                 KYCTier         `json:"kyc_tier" db:"kyc_tier" validate:"required,oneof=tier1 tier2 tier3"`
	MonthlyLimitUSD         decimal.Decimal `json:"monthly_limit_usd" db:"monthly_limit_usd" validate:"gte=0"`
	TotalVolumeThisMonthUSD decimal.Decimal `json:"total_volume_this_month_usd" db:"total_volume_this_month_usd" validate:"gte=0"`
	VolumeLastResetAt       time.Time       `json:"volume_last_reset_at" db:"volume_last_reset_at"`

	// API credentials
	APIKey          sql.NullString `json:"api_key,omitempty" db:"api_key"`
	APIKeyCreatedAt sql.NullTime   `json:"api_key_created_at,omitempty" db:"api_key_created_at"`

	// Webhook configuration
	WebhookURL    sql.NullString `json:"webhook_url,omitempty" db:"webhook_url" validate:"omitempty,url"`
	WebhookSecret sql.NullString `json:"webhook_secret,omitempty" db:"webhook_secret"`
	WebhookEvents pq.StringArray `json:"webhook_events,omitempty" db:"webhook_events"`

	// Status
	Status MerchantStatus `json:"status" db:"status" validate:"required,oneof=active suspended closed"`

	// Metadata
	Metadata JSONBMap `json:"metadata,omitempty" db:"metadata"`

	// Timestamps
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt sql.NullTime `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsApproved returns true if the merchant's KYC is approved
func (m *Merchant) IsApproved() bool {
	return m.KYCStatus == KYCStatusApproved
}

// IsActive returns true if the merchant is active
func (m *Merchant) IsActive() bool {
	return m.Status == MerchantStatusActive && !m.DeletedAt.Valid
}

// CanAcceptPayments returns true if the merchant can accept payments
func (m *Merchant) CanAcceptPayments() bool {
	return m.IsApproved() && m.IsActive() && m.APIKey.Valid
}

// HasWebhook returns true if the merchant has webhook configured
func (m *Merchant) HasWebhook() bool {
	return m.WebhookURL.Valid && m.WebhookURL.String != ""
}

// GetWebhookURL returns the webhook URL if configured
func (m *Merchant) GetWebhookURL() string {
	if m.WebhookURL.Valid {
		return m.WebhookURL.String
	}
	return ""
}

// GetWebhookSecret returns the webhook secret if configured
func (m *Merchant) GetWebhookSecret() string {
	if m.WebhookSecret.Valid {
		return m.WebhookSecret.String
	}
	return ""
}

// IsWithinMonthlyLimit checks if the merchant can process additional amount without exceeding monthly limit
func (m *Merchant) IsWithinMonthlyLimit(additionalAmountUSD decimal.Decimal) bool {
	totalAfterTransaction := m.TotalVolumeThisMonthUSD.Add(additionalAmountUSD)
	return totalAfterTransaction.LessThanOrEqual(m.MonthlyLimitUSD)
}

// GetRemainingLimit returns the remaining transaction limit for this month
func (m *Merchant) GetRemainingLimit() decimal.Decimal {
	remaining := m.MonthlyLimitUSD.Sub(m.TotalVolumeThisMonthUSD)
	if remaining.IsNegative() {
		return decimal.Zero
	}
	return remaining
}

// NeedsMonthlyReset checks if the monthly volume tracking needs to be reset
func (m *Merchant) NeedsMonthlyReset() bool {
	now := time.Now()
	lastReset := m.VolumeLastResetAt

	// Reset needed if we're in a different month than the last reset
	return now.Year() > lastReset.Year() ||
		(now.Year() == lastReset.Year() && now.Month() > lastReset.Month())
}

// GetTierLimitUSD returns the default monthly limit for a given tier
func GetTierLimitUSD(tier KYCTier) decimal.Decimal {
	switch tier {
	case KYCTier1:
		return decimal.NewFromInt(5000) // $5,000/month
	case KYCTier2:
		return decimal.NewFromInt(50000) // $50,000/month
	case KYCTier3:
		return decimal.NewFromInt(500000) // $500,000/month
	default:
		return decimal.NewFromInt(5000) // Default to Tier 1
	}
}
