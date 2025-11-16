package model

import (
	"time"

	"github.com/google/uuid"
)

// KYCSubmission represents a KYC submission for a merchant
type KYCSubmission struct {
	ID           uuid.UUID    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MerchantID   uuid.UUID    `json:"merchant_id" gorm:"type:uuid;not null"`
	MerchantType MerchantType `json:"merchant_type" gorm:"type:varchar(50);not null"`

	// Status tracking
	Status     KYCStatus  `json:"status" gorm:"type:varchar(50);default:'in_progress'"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	RejectedAt  *time.Time `json:"rejected_at,omitempty"`

	// Individual-specific fields
	IndividualFullName    *string    `json:"individual_full_name,omitempty" gorm:"type:varchar(255)"`
	IndividualCCCDNumber  *string    `json:"individual_cccd_number,omitempty" gorm:"type:varchar(20)"`
	IndividualDateOfBirth *time.Time `json:"individual_date_of_birth,omitempty" gorm:"type:date"`
	IndividualAddress     *string    `json:"individual_address,omitempty" gorm:"type:text"`

	// Household business-specific fields (Hộ Kinh Tế)
	HKTBusinessName    *string `json:"hkt_business_name,omitempty" gorm:"type:varchar(255)"`
	HKTTaxID           *string `json:"hkt_tax_id,omitempty" gorm:"type:varchar(50)"`
	HKTBusinessAddress *string `json:"hkt_business_address,omitempty" gorm:"type:text"`
	HKTOwnerName       *string `json:"hkt_owner_name,omitempty" gorm:"type:varchar(255)"`
	HKTOwnerCCCD       *string `json:"hkt_owner_cccd,omitempty" gorm:"type:varchar(20)"`

	// Company-specific fields
	CompanyLegalName            *string `json:"company_legal_name,omitempty" gorm:"type:varchar(255)"`
	CompanyRegistrationNumber   *string `json:"company_registration_number,omitempty" gorm:"type:varchar(50)"`
	CompanyTaxID                *string `json:"company_tax_id,omitempty" gorm:"type:varchar(50)"`
	CompanyHeadquartersAddress  *string `json:"company_headquarters_address,omitempty" gorm:"type:text"`
	CompanyWebsite              *string `json:"company_website,omitempty" gorm:"type:varchar(500)"`
	CompanyDirectorName         *string `json:"company_director_name,omitempty" gorm:"type:varchar(255)"`
	CompanyDirectorCCCD         *string `json:"company_director_cccd,omitempty" gorm:"type:varchar(20)"`

	// Common fields
	BusinessDescription *string `json:"business_description,omitempty" gorm:"type:text"`
	BusinessWebsite     *string `json:"business_website,omitempty" gorm:"type:varchar(500)"`
	BusinessFacebook    *string `json:"business_facebook,omitempty" gorm:"type:varchar(500)"`
	ProductCategory     *string `json:"product_category,omitempty" gorm:"type:varchar(100)"`

	// Risk and verification
	AutoApproved           bool    `json:"auto_approved" gorm:"default:false"`
	RequiresManualReview   bool    `json:"requires_manual_review" gorm:"default:false"`
	RejectionReason        *string `json:"rejection_reason,omitempty" gorm:"type:text"`
	AdminNotes             *string `json:"admin_notes,omitempty" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at" gorm:"default:now()"`
	UpdatedAt time.Time `json:"updated_at" gorm:"default:now()"`

	// Relationships
	Merchant             *Merchant               `json:"merchant,omitempty" gorm:"foreignKey:MerchantID"`
	Documents            []KYCDocument           `json:"documents,omitempty" gorm:"foreignKey:KYCSubmissionID"`
	VerificationResults  []KYCVerificationResult `json:"verification_results,omitempty" gorm:"foreignKey:KYCSubmissionID"`
	RiskAssessment       *KYCRiskAssessment      `json:"risk_assessment,omitempty" gorm:"foreignKey:KYCSubmissionID"`
	ReviewActions        []KYCReviewAction       `json:"review_actions,omitempty" gorm:"foreignKey:KYCSubmissionID"`
}

// TableName specifies the table name for GORM
func (KYCSubmission) TableName() string {
	return "kyc_submissions"
}

// IsSubmitted checks if the KYC has been submitted
func (k *KYCSubmission) IsSubmitted() bool {
	return k.SubmittedAt != nil
}

// IsApproved checks if the KYC has been approved
func (k *KYCSubmission) IsApproved() bool {
	return k.Status == KYCStatusApproved && k.ApprovedAt != nil
}

// IsRejected checks if the KYC has been rejected
func (k *KYCSubmission) IsRejected() bool {
	return k.Status == KYCStatusRejected && k.RejectedAt != nil
}

// IsPendingReview checks if the KYC is pending manual review
func (k *KYCSubmission) IsPendingReview() bool {
	return k.Status == KYCStatusPendingReview
}

// CanBeAutoApproved checks if submission can be auto-approved
func (k *KYCSubmission) CanBeAutoApproved() bool {
	return !k.RequiresManualReview && k.Status == KYCStatusInProgress
}
