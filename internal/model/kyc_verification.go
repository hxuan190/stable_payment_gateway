package model

import (
	"time"

	"github.com/google/uuid"
)

// VerificationStatus represents the status of a verification check
type VerificationStatus string

const (
	VerificationStatusPending VerificationStatus = "pending"
	VerificationStatusSuccess VerificationStatus = "success"
	VerificationStatusFailed  VerificationStatus = "failed"
	VerificationStatusSkipped VerificationStatus = "skipped"
)

// VerificationType represents different types of verification
const (
	VerificationTypeOCR             = "ocr"              // OCR extraction from CCCD
	VerificationTypeFaceMatch       = "face_match"       // Selfie vs CCCD photo
	VerificationTypeMSTLookup       = "mst_lookup"       // Tax ID validation
	VerificationTypeAgeCheck        = "age_check"        // Age >= 18
	VerificationTypeNameMatch       = "name_match"       // Name consistency across documents
	VerificationTypeBankMatch       = "bank_match"       // Bank account name vs declared name
	VerificationTypeSanctionsCheck  = "sanctions_check"  // Check against sanctions lists
	VerificationTypeBusinessLookup  = "business_lookup"  // Company registration validation
)

// KYCVerificationResult represents the result of an automated verification check
type KYCVerificationResult struct {
	ID              uuid.UUID          `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	KYCSubmissionID uuid.UUID          `json:"kyc_submission_id" gorm:"type:uuid;not null"`
	MerchantID      uuid.UUID          `json:"merchant_id" gorm:"type:uuid;not null"`
	VerificationType string             `json:"verification_type" gorm:"type:varchar(50);not null"`
	Status          VerificationStatus `json:"status" gorm:"type:varchar(50);default:'pending'"`

	// Results
	Passed          *bool    `json:"passed,omitempty"`
	ConfidenceScore *float64 `json:"confidence_score,omitempty" gorm:"type:decimal(5,4)"` // 0.0000 to 1.0000
	ResultData      *string  `json:"result_data,omitempty" gorm:"type:jsonb"`             // Detailed results
	ErrorMessage    *string  `json:"error_message,omitempty" gorm:"type:text"`

	// Timing
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	DurationMS  *int       `json:"duration_ms,omitempty"`

	CreatedAt time.Time `json:"created_at" gorm:"default:now()"`

	// Relationships
	KYCSubmission *KYCSubmission `json:"kyc_submission,omitempty" gorm:"foreignKey:KYCSubmissionID"`
	Merchant      *Merchant      `json:"merchant,omitempty" gorm:"foreignKey:MerchantID"`
}

// TableName specifies the table name for GORM
func (KYCVerificationResult) TableName() string {
	return "kyc_verification_results"
}

// IsCompleted checks if the verification has completed
func (v *KYCVerificationResult) IsCompleted() bool {
	return v.Status == VerificationStatusSuccess || v.Status == VerificationStatusFailed
}

// IsPassed checks if the verification passed
func (v *KYCVerificationResult) IsPassed() bool {
	return v.Passed != nil && *v.Passed
}

// IsFailed checks if the verification failed
func (v *KYCVerificationResult) IsFailed() bool {
	return v.Passed != nil && !*v.Passed
}

// GetConfidence returns the confidence score or 0 if not set
func (v *KYCVerificationResult) GetConfidence() float64 {
	if v.ConfidenceScore != nil {
		return *v.ConfidenceScore
	}
	return 0.0
}
