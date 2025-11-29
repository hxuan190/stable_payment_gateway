package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/database"
)

// ComplianceAlert represents a compliance issue that requires manual review
type ComplianceAlert struct {
	ID uuid.UUID `json:"id" db:"id"`

	// Alert classification
	AlertType string `json:"alert_type" db:"alert_type"` // 'sanctioned_address', 'high_risk_address', 'aml_failure', 'travel_rule_missing', etc.
	Severity  string `json:"severity" db:"severity"`     // 'critical', 'high', 'medium', 'low'
	Status    string `json:"status" db:"status"`         // 'open', 'investigating', 'resolved', 'false_positive'

	// Related entities
	PaymentID  *uuid.UUID `json:"payment_id,omitempty" db:"payment_id"`
	MerchantID *uuid.UUID `json:"merchant_id,omitempty" db:"merchant_id"`
	PayoutID   *uuid.UUID `json:"payout_id,omitempty" db:"payout_id"`

	// Alert details
	FromAddress     *string `json:"from_address,omitempty" db:"from_address"`
	ToAddress       *string `json:"to_address,omitempty" db:"to_address"`
	TransactionHash *string `json:"transaction_hash,omitempty" db:"transaction_hash"`
	Blockchain      *string `json:"blockchain,omitempty" db:"blockchain"` // 'solana', 'bsc', 'ethereum', 'tron', etc.

	// Risk assessment
	RiskScore *int              `json:"risk_score,omitempty" db:"risk_score"` // 0-100
	RiskFlags []string          `json:"risk_flags,omitempty" db:"risk_flags"` // Array of risk indicators
	Details   string            `json:"details" db:"details"`                 // Description of the compliance issue
	Evidence  database.JSONBMap `json:"evidence,omitempty" db:"evidence"`     // Additional evidence (screening results, TRM Labs data, etc.)

	// Actions taken
	ActionsTaken      []string `json:"actions_taken,omitempty" db:"actions_taken"`           // Array of actions: 'payment_flagged', 'email_sent', 'funds_frozen', etc.
	RecommendedAction *string  `json:"recommended_action,omitempty" db:"recommended_action"` // What should be done

	// Review workflow
	AssignedTo      *string    `json:"assigned_to,omitempty" db:"assigned_to"`           // Email of ops team member
	ReviewedBy      *string    `json:"reviewed_by,omitempty" db:"reviewed_by"`           // Email of reviewer
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`           // When reviewed
	ResolutionNotes *string    `json:"resolution_notes,omitempty" db:"resolution_notes"` // Notes on resolution

	// Notification tracking
	EmailSent       bool       `json:"email_sent" db:"email_sent"`
	EmailSentAt     *time.Time `json:"email_sent_at,omitempty" db:"email_sent_at"`
	EmailRecipients []string   `json:"email_recipients,omitempty" db:"email_recipients"` // Array of email addresses notified

	// Timestamps
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
}

// Alert type constants
const (
	AlertTypeSanctionedAddress  = "sanctioned_address"
	AlertTypeHighRiskAddress    = "high_risk_address"
	AlertTypeAMLFailure         = "aml_failure"
	AlertTypeTravelRuleMissing  = "travel_rule_missing"
	AlertTypeUnusualTransaction = "unusual_transaction"
	AlertTypeLargeTransaction   = "large_transaction"
)

// Severity constants
const (
	SeverityCritical = "critical" // Immediate action required
	SeverityHigh     = "high"     // Same day action required
	SeverityMedium   = "medium"   // Review within 24 hours
	SeverityLow      = "low"      // Review when possible
)

// Status constants
const (
	AlertStatusOpen          = "open"           // Needs review
	AlertStatusInvestigating = "investigating"  // Under review
	AlertStatusResolved      = "resolved"       // Issue handled
	AlertStatusFalsePositive = "false_positive" // No action needed
)

// Action constants
const (
	ActionPaymentFlagged         = "payment_flagged"
	ActionEmailSent              = "email_sent"
	ActionFundsFrozen            = "funds_frozen"
	ActionMerchantNotified       = "merchant_notified"
	ActionLawEnforcementNotified = "law_enforcement_contacted"
	ActionPaymentReversed        = "payment_reversed"
	ActionKYCRequested           = "kyc_requested"
)

// NewComplianceAlert creates a new compliance alert with default values
func NewComplianceAlert(alertType, severity, details string) *ComplianceAlert {
	now := time.Now()
	return &ComplianceAlert{
		ID:              uuid.New(),
		AlertType:       alertType,
		Severity:        severity,
		Status:          AlertStatusOpen,
		Details:         details,
		EmailSent:       false,
		CreatedAt:       now,
		UpdatedAt:       now,
		RiskFlags:       []string{},
		ActionsTaken:    []string{},
		EmailRecipients: []string{},
		Evidence:        make(database.JSONBMap),
	}
}

// AddAction adds an action to the actions_taken list
func (a *ComplianceAlert) AddAction(action string) {
	if a.ActionsTaken == nil {
		a.ActionsTaken = []string{}
	}
	a.ActionsTaken = append(a.ActionsTaken, action)
	a.UpdatedAt = time.Now()
}

// MarkEmailSent marks the alert as having an email sent
func (a *ComplianceAlert) MarkEmailSent(recipients []string) {
	now := time.Now()
	a.EmailSent = true
	a.EmailSentAt = &now
	a.EmailRecipients = recipients
	a.UpdatedAt = now
	a.AddAction(ActionEmailSent)
}

// Resolve marks the alert as resolved
func (a *ComplianceAlert) Resolve(reviewedBy, resolutionNotes string) {
	now := time.Now()
	a.Status = AlertStatusResolved
	a.ReviewedBy = &reviewedBy
	a.ReviewedAt = &now
	a.ResolvedAt = &now
	a.ResolutionNotes = &resolutionNotes
	a.UpdatedAt = now
}

// MarkFalsePositive marks the alert as a false positive
func (a *ComplianceAlert) MarkFalsePositive(reviewedBy, notes string) {
	now := time.Now()
	a.Status = AlertStatusFalsePositive
	a.ReviewedBy = &reviewedBy
	a.ReviewedAt = &now
	a.ResolvedAt = &now
	a.ResolutionNotes = &notes
	a.UpdatedAt = now
}
