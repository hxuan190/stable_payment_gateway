package model

import (
	"time"

	"github.com/google/uuid"
)

// MerchantType represents the type of merchant
type MerchantType string

const (
	MerchantTypeIndividual        MerchantType = "individual"
	MerchantTypeHouseholdBusiness MerchantType = "household_business"
	MerchantTypeCompany           MerchantType = "company"
)

// KYCStatus represents the status of KYC verification
type KYCStatus string

const (
	KYCStatusNotStarted    KYCStatus = "not_started"
	KYCStatusInProgress    KYCStatus = "in_progress"
	KYCStatusPendingReview KYCStatus = "pending_review"
	KYCStatusApproved      KYCStatus = "approved"
	KYCStatusRejected      KYCStatus = "rejected"
)

// Merchant represents a merchant account
type Merchant struct {
	ID           uuid.UUID    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email        string       `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	BusinessName string       `json:"business_name" gorm:"type:varchar(255);not null"`
	TaxID        *string      `json:"tax_id,omitempty" gorm:"type:varchar(50)"`
	MerchantType MerchantType `json:"merchant_type" gorm:"type:varchar(50);default:'individual'"`

	// Personal info (for individuals)
	FullName *string `json:"full_name,omitempty" gorm:"type:varchar(255)"`
	Phone    *string `json:"phone,omitempty" gorm:"type:varchar(20)"`

	// Banking info
	BankAccountNumber *string `json:"bank_account_number,omitempty" gorm:"type:varchar(50)"`
	BankAccountName   *string `json:"bank_account_name,omitempty" gorm:"type:varchar(255)"`
	BankName          *string `json:"bank_name,omitempty" gorm:"type:varchar(100)"`

	// KYC
	KYCStatus KYCStatus `json:"kyc_status" gorm:"type:varchar(50);default:'not_started'"`
	KYCData   *string   `json:"kyc_data,omitempty" gorm:"type:jsonb"` // Legacy field, use kyc_submissions instead

	// Transaction limits
	DailyLimitVND   float64 `json:"daily_limit_vnd" gorm:"type:decimal(15,2);default:200000000"`
	MonthlyLimitVND float64 `json:"monthly_limit_vnd" gorm:"type:decimal(15,2);default:3000000000"`

	// API credentials
	APIKey        *string `json:"api_key,omitempty" gorm:"type:varchar(255);uniqueIndex"`
	WebhookURL    *string `json:"webhook_url,omitempty" gorm:"type:varchar(500)"`
	WebhookSecret *string `json:"webhook_secret,omitempty" gorm:"type:varchar(255)"`

	CreatedAt time.Time `json:"created_at" gorm:"default:now()"`
	UpdatedAt time.Time `json:"updated_at" gorm:"default:now()"`
}

// TableName specifies the table name for GORM
func (Merchant) TableName() string {
	return "merchants"
}

// IsKYCApproved checks if the merchant's KYC is approved
func (m *Merchant) IsKYCApproved() bool {
	return m.KYCStatus == KYCStatusApproved
}

// CanAcceptPayments checks if merchant can accept payments
func (m *Merchant) CanAcceptPayments() bool {
	return m.IsKYCApproved() && m.APIKey != nil && *m.APIKey != ""
}
