package model

import (
	"time"

	"github.com/google/uuid"
)

// ReviewAction represents the action taken by a reviewer
type ReviewAction string

const (
	ReviewActionApprove        ReviewAction = "approve"
	ReviewActionReject         ReviewAction = "reject"
	ReviewActionRequestMoreInfo ReviewAction = "request_more_info"
	ReviewActionFlag           ReviewAction = "flag"
	ReviewActionComment        ReviewAction = "comment"
)

// ReviewDecision represents the final decision
type ReviewDecision string

const (
	DecisionApproved ReviewDecision = "approved"
	DecisionRejected ReviewDecision = "rejected"
	DecisionPending  ReviewDecision = "pending"
)

// KYCReviewAction represents a manual review action by admin staff
type KYCReviewAction struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	KYCSubmissionID uuid.UUID      `json:"kyc_submission_id" gorm:"type:uuid;not null"`
	MerchantID      uuid.UUID      `json:"merchant_id" gorm:"type:uuid;not null"`

	// Reviewer info (to be linked with admin user table later)
	ReviewerID    *uuid.UUID `json:"reviewer_id,omitempty" gorm:"type:uuid"`
	ReviewerEmail *string    `json:"reviewer_email,omitempty" gorm:"type:varchar(255)"`
	ReviewerName  *string    `json:"reviewer_name,omitempty" gorm:"type:varchar(255)"`

	// Action taken
	Action   ReviewAction    `json:"action" gorm:"type:varchar(50);not null"`
	Decision *ReviewDecision `json:"decision,omitempty" gorm:"type:varchar(50)"`

	// Details
	Notes             *string `json:"notes,omitempty" gorm:"type:text"`
	InternalNotes     *string `json:"internal_notes,omitempty" gorm:"type:text"` // Not visible to merchant
	RequiredDocuments *string `json:"required_documents,omitempty" gorm:"type:jsonb"` // List of additional docs needed

	// Limits set (if approved)
	ApprovedDailyLimitVND   *float64 `json:"approved_daily_limit_vnd,omitempty" gorm:"type:decimal(15,2)"`
	ApprovedMonthlyLimitVND *float64 `json:"approved_monthly_limit_vnd,omitempty" gorm:"type:decimal(15,2)"`

	CreatedAt time.Time `json:"created_at" gorm:"default:now()"`

	// Relationships
	KYCSubmission *KYCSubmission `json:"kyc_submission,omitempty" gorm:"foreignKey:KYCSubmissionID"`
	Merchant      *Merchant      `json:"merchant,omitempty" gorm:"foreignKey:MerchantID"`
}

// TableName specifies the table name for GORM
func (KYCReviewAction) TableName() string {
	return "kyc_review_actions"
}

// IsApproval checks if this is an approval action
func (r *KYCReviewAction) IsApproval() bool {
	return r.Action == ReviewActionApprove
}

// IsRejection checks if this is a rejection action
func (r *KYCReviewAction) IsRejection() bool {
	return r.Action == ReviewActionReject
}

// RequestsMoreInfo checks if this action requests more information
func (r *KYCReviewAction) RequestsMoreInfo() bool {
	return r.Action == ReviewActionRequestMoreInfo
}
