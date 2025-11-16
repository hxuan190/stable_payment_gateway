package model

import (
	"time"

	"github.com/google/uuid"
)

// RiskLevel represents the risk level of a merchant
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// RecommendedAction represents the recommended action based on risk assessment
type RecommendedAction string

const (
	ActionAutoApprove  RecommendedAction = "auto_approve"
	ActionManualReview RecommendedAction = "manual_review"
	ActionReject       RecommendedAction = "reject"
)

// KYCRiskAssessment represents the risk assessment for a KYC submission
type KYCRiskAssessment struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	KYCSubmissionID uuid.UUID `json:"kyc_submission_id" gorm:"type:uuid;not null"`
	MerchantID      uuid.UUID `json:"merchant_id" gorm:"type:uuid;not null"`

	// Overall risk
	RiskLevel RiskLevel `json:"risk_level" gorm:"type:varchar(50);default:'low'"`
	RiskScore int       `json:"risk_score" gorm:"default:0"` // 0-100

	// Risk factors (stored as JSONB)
	RiskFactors *string `json:"risk_factors,omitempty" gorm:"type:jsonb"`
	/*
	Example structure:
	{
		"high_risk_industry": true,
		"sanctioned_entity": false,
		"suspicious_business_pattern": false,
		"age_below_threshold": false,
		"name_mismatch": false,
		"document_quality_issues": false,
		"unusual_transaction_patterns": false
	}
	*/

	// Industry risk
	IndustryCategory    *string `json:"industry_category,omitempty" gorm:"type:varchar(100)"`
	IsHighRiskIndustry  bool    `json:"is_high_risk_industry" gorm:"default:false"`

	// Sanctions and compliance
	SanctionsChecked bool    `json:"sanctions_checked" gorm:"default:false"`
	SanctionsHit     bool    `json:"sanctions_hit" gorm:"default:false"`
	SanctionsDetails *string `json:"sanctions_details,omitempty" gorm:"type:jsonb"`

	// Recommendations
	RecommendedAction RecommendedAction `json:"recommended_action" gorm:"type:varchar(50)"`
	RecommendedLimits *string           `json:"recommended_limits,omitempty" gorm:"type:jsonb"`
	/*
	Example structure:
	{
		"daily_limit_vnd": 200000000,
		"monthly_limit_vnd": 3000000000
	}
	*/

	AssessedAt time.Time `json:"assessed_at" gorm:"default:now()"`
	CreatedAt  time.Time `json:"created_at" gorm:"default:now()"`

	// Relationships
	KYCSubmission *KYCSubmission `json:"kyc_submission,omitempty" gorm:"foreignKey:KYCSubmissionID"`
	Merchant      *Merchant      `json:"merchant,omitempty" gorm:"foreignKey:MerchantID"`
}

// TableName specifies the table name for GORM
func (KYCRiskAssessment) TableName() string {
	return "kyc_risk_assessments"
}

// IsLowRisk checks if the assessment is low risk
func (r *KYCRiskAssessment) IsLowRisk() bool {
	return r.RiskLevel == RiskLevelLow
}

// IsHighRisk checks if the assessment is high or critical risk
func (r *KYCRiskAssessment) IsHighRisk() bool {
	return r.RiskLevel == RiskLevelHigh || r.RiskLevel == RiskLevelCritical
}

// ShouldAutoApprove checks if the assessment recommends auto-approval
func (r *KYCRiskAssessment) ShouldAutoApprove() bool {
	return r.RecommendedAction == ActionAutoApprove
}

// ShouldReject checks if the assessment recommends rejection
func (r *KYCRiskAssessment) ShouldReject() bool {
	return r.RecommendedAction == ActionReject
}

// RequiresManualReview checks if the assessment requires manual review
func (r *KYCRiskAssessment) RequiresManualReview() bool {
	return r.RecommendedAction == ActionManualReview
}

// HasSanctionsHit checks if there's a sanctions match
func (r *KYCRiskAssessment) HasSanctionsHit() bool {
	return r.SanctionsChecked && r.SanctionsHit
}

// RiskFactorsMap represents the structure of risk factors
type RiskFactorsMap struct {
	HighRiskIndustry           bool `json:"high_risk_industry"`
	SanctionedEntity           bool `json:"sanctioned_entity"`
	SuspiciousBusinessPattern  bool `json:"suspicious_business_pattern"`
	AgeBelowThreshold          bool `json:"age_below_threshold"`
	NameMismatch               bool `json:"name_mismatch"`
	DocumentQualityIssues      bool `json:"document_quality_issues"`
	UnusualTransactionPatterns bool `json:"unusual_transaction_patterns"`
}

// RecommendedLimitsMap represents the structure of recommended limits
type RecommendedLimitsMap struct {
	DailyLimitVND   float64 `json:"daily_limit_vnd"`
	MonthlyLimitVND float64 `json:"monthly_limit_vnd"`
}
