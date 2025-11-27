package model

import (
	"database/sql"
	"time"
)

// ActorType represents who performed the action
type ActorType string

const (
	ActorTypeSystem   ActorType = "system"
	ActorTypeMerchant ActorType = "merchant"
	ActorTypeAdmin    ActorType = "admin"
	ActorTypeOps      ActorType = "ops"
	ActorTypeAPI      ActorType = "api"
	ActorTypeWorker   ActorType = "worker"
	ActorTypeListener ActorType = "listener"
)

// ActionCategory represents the category of action
type ActionCategory string

const (
	ActionCategoryPayment        ActionCategory = "payment"
	ActionCategoryPayout         ActionCategory = "payout"
	ActionCategoryKYC            ActionCategory = "kyc"
	ActionCategoryAuthentication ActionCategory = "authentication"
	ActionCategoryAdmin          ActionCategory = "admin"
	ActionCategorySystem         ActionCategory = "system"
	ActionCategoryBlockchain     ActionCategory = "blockchain"
	ActionCategoryWebhook        ActionCategory = "webhook"
	ActionCategorySecurity       ActionCategory = "security"
)

// AuditStatus represents the result of the action
type AuditStatus string

const (
	AuditStatusSuccess AuditStatus = "success"
	AuditStatusFailed  AuditStatus = "failed"
	AuditStatusError   AuditStatus = "error"
)

// AuditLog represents a comprehensive audit trail entry
// This table is append-only and immutable
type AuditLog struct {
	ID string `json:"id" db:"id"`

	// Who performed the action
	ActorType      ActorType      `json:"actor_type" db:"actor_type" validate:"required,oneof=system merchant admin ops api worker listener"`
	ActorID        sql.NullString `json:"actor_id,omitempty" db:"actor_id"`
	ActorEmail     sql.NullString `json:"actor_email,omitempty" db:"actor_email"`
	ActorIPAddress sql.NullString `json:"actor_ip_address,omitempty" db:"actor_ip_address"`

	// What action was performed
	Action         string         `json:"action" db:"action" validate:"required,min=1,max=100"`
	ActionCategory ActionCategory `json:"action_category" db:"action_category" validate:"required,oneof=payment payout kyc authentication admin system blockchain webhook security"`

	// What resource was affected
	ResourceType string `json:"resource_type" db:"resource_type" validate:"required,min=1,max=50"`
	ResourceID   string `json:"resource_id" db:"resource_id" validate:"required,uuid"`

	// Action result
	Status       AuditStatus    `json:"status" db:"status" validate:"required,oneof=success failed error"`
	ErrorMessage sql.NullString `json:"error_message,omitempty" db:"error_message"`

	// Request details
	HTTPMethod     sql.NullString `json:"http_method,omitempty" db:"http_method"`
	HTTPPath       sql.NullString `json:"http_path,omitempty" db:"http_path"`
	HTTPStatusCode sql.NullInt32  `json:"http_status_code,omitempty" db:"http_status_code"`
	UserAgent      sql.NullString `json:"user_agent,omitempty" db:"user_agent"`

	// Before and after state (for updates)
	OldValues JSONBMap `json:"old_values,omitempty" db:"old_values"`
	NewValues JSONBMap `json:"new_values,omitempty" db:"new_values"`

	// Additional context
	Metadata    JSONBMap       `json:"metadata,omitempty" db:"metadata"`
	Description sql.NullString `json:"description,omitempty" db:"description"`

	// Correlation for request tracing
	RequestID     sql.NullString `json:"request_id,omitempty" db:"request_id"`
	CorrelationID sql.NullString `json:"correlation_id,omitempty" db:"correlation_id"`

	// Timestamp - immutable (no updated_at or deleted_at)
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// IsSuccess returns true if the action was successful
func (a *AuditLog) IsSuccess() bool {
	return a.Status == AuditStatusSuccess
}

// IsFailed returns true if the action failed
func (a *AuditLog) IsFailed() bool {
	return a.Status == AuditStatusFailed || a.Status == AuditStatusError
}

// GetActorID returns the actor ID if available
func (a *AuditLog) GetActorID() string {
	if a.ActorID.Valid {
		return a.ActorID.String
	}
	return ""
}

// GetActorEmail returns the actor email if available
func (a *AuditLog) GetActorEmail() string {
	if a.ActorEmail.Valid {
		return a.ActorEmail.String
	}
	return ""
}

// GetActorIPAddress returns the actor IP address if available
func (a *AuditLog) GetActorIPAddress() string {
	if a.ActorIPAddress.Valid {
		return a.ActorIPAddress.String
	}
	return ""
}

// GetErrorMessage returns the error message if available
func (a *AuditLog) GetErrorMessage() string {
	if a.ErrorMessage.Valid {
		return a.ErrorMessage.String
	}
	return ""
}

// GetDescription returns the description if available
func (a *AuditLog) GetDescription() string {
	if a.Description.Valid {
		return a.Description.String
	}
	return ""
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
