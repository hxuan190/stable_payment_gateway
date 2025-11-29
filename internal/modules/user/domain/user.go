package domain

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/database"
	"github.com/shopspring/decimal"
)

// RiskLevel represents the user's risk assessment level
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// DocumentType represents the type of identity document
type DocumentType string

const (
	DocumentTypePassport       DocumentType = "passport"
	DocumentTypeNationalID     DocumentType = "national_id"
	DocumentTypeDrivingLicense DocumentType = "driving_license"
	DocumentTypeOther          DocumentType = "other"
)

// User represents an end-user identity (PRD v2.2 - Smart Identity Mapping)
type User struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`

	// Basic Information
	FullName    string         `gorm:"type:varchar(255);not null" json:"full_name"`
	DateOfBirth sql.NullTime   `gorm:"type:date" json:"date_of_birth,omitempty"`
	Nationality sql.NullString `gorm:"type:varchar(3)" json:"nationality,omitempty"` // ISO 3166-1 alpha-3
	Email       sql.NullString `gorm:"type:varchar(255);index" json:"email,omitempty"`
	Phone       sql.NullString `gorm:"type:varchar(50);index" json:"phone,omitempty"`

	// KYC Information
	KYCStatus          domain.KYCStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"kyc_status"`
	KYCProvider        sql.NullString   `gorm:"type:varchar(50);default:'sumsub'" json:"kyc_provider,omitempty"`
	KYCApplicantID     sql.NullString   `gorm:"type:varchar(255);index" json:"kyc_applicant_id,omitempty"` // Sumsub applicant ID
	KYCVerifiedAt      sql.NullTime     `gorm:"type:timestamp" json:"kyc_verified_at,omitempty"`
	KYCRejectionReason sql.NullString   `gorm:"type:text" json:"kyc_rejection_reason,omitempty"`
	FaceLivenessPassed bool             `gorm:"default:false" json:"face_liveness_passed"`

	// Identity Documents
	DocumentType       sql.NullString `gorm:"type:varchar(50)" json:"document_type,omitempty"`
	DocumentNumber     sql.NullString `gorm:"type:varchar(100);index" json:"document_number,omitempty"`
	DocumentCountry    sql.NullString `gorm:"type:varchar(3)" json:"document_country,omitempty"` // ISO 3166-1 alpha-3
	DocumentExpiryDate sql.NullTime   `gorm:"type:date" json:"document_expiry_date,omitempty"`

	// Risk Assessment
	RiskLevel    RiskLevel    `gorm:"type:varchar(20);default:'low';index" json:"risk_level"`
	AMLCheckedAt sql.NullTime `gorm:"type:timestamp" json:"aml_checked_at,omitempty"`
	IsPEP        bool         `gorm:"default:false" json:"is_pep"` // Politically Exposed Person
	IsSanctioned bool         `gorm:"default:false" json:"is_sanctioned"`

	// Activity Tracking
	FirstPaymentAt        sql.NullTime    `gorm:"type:timestamp" json:"first_payment_at,omitempty"`
	LastPaymentAt         sql.NullTime    `gorm:"type:timestamp;index" json:"last_payment_at,omitempty"`
	TotalPaymentCount     int             `gorm:"default:0" json:"total_payment_count"`
	TotalPaymentVolumeUSD decimal.Decimal `gorm:"type:decimal(20,2);default:0" json:"total_payment_volume_usd"`

	// Metadata
	Metadata database.JSONBMap `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`
	Notes    sql.NullString    `gorm:"type:text" json:"notes,omitempty"` // Admin notes

	// Audit fields
	CreatedAt time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;index" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
	CreatedBy sql.NullString `gorm:"type:varchar(100)" json:"created_by,omitempty"`
	UpdatedBy sql.NullString `gorm:"type:varchar(100)" json:"updated_by,omitempty"`

	// Associations - removed to break circular dependency, will be handled by service layer or separate mapping
	// WalletMappings []WalletIdentityMapping `gorm:"foreignKey:UserID" json:"wallet_mappings,omitempty"`
}

// TableName specifies the table name for User model
func (User) TableName() string {
	return "users"
}

// IsKYCVerified returns true if user has passed KYC
func (u *User) IsKYCVerified() bool {
	return u.KYCStatus == domain.KYCStatusApproved && u.FaceLivenessPassed
}

// IsHighRisk returns true if user is high risk or critical
func (u *User) IsHighRisk() bool {
	return u.RiskLevel == RiskLevelHigh || u.RiskLevel == RiskLevelCritical
}

// NeedsEnhancedDueDiligence returns true if user requires EDD
func (u *User) NeedsEnhancedDueDiligence() bool {
	return u.IsPEP || u.IsSanctioned || u.IsHighRisk()
}

// CanMakePayment returns true if user is allowed to make payments
func (u *User) CanMakePayment() bool {
	return u.IsKYCVerified() && !u.IsSanctioned
}

// BeforeCreate hook to set default values
func (u *User) BeforeCreate() error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
