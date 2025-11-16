package model

import (
	"time"

	"github.com/google/uuid"
)

// ActorType represents the type of actor performing an action
type ActorType string

const (
	ActorTypeSystem   ActorType = "system"
	ActorTypeAdmin    ActorType = "admin"
	ActorTypeMerchant ActorType = "merchant"
)

// KYCAuditLog represents an audit log entry for KYC operations
type KYCAuditLog struct {
	ID              uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	KYCSubmissionID *uuid.UUID `json:"kyc_submission_id,omitempty" gorm:"type:uuid"`
	MerchantID      *uuid.UUID `json:"merchant_id,omitempty" gorm:"type:uuid"`

	// Actor information
	ActorType  ActorType `json:"actor_type" gorm:"type:varchar(50)"`
	ActorID    *string   `json:"actor_id,omitempty" gorm:"type:varchar(255)"`
	ActorEmail *string   `json:"actor_email,omitempty" gorm:"type:varchar(255)"`

	// Action details
	Action       string     `json:"action" gorm:"type:varchar(100);not null"`
	ResourceType *string    `json:"resource_type,omitempty" gorm:"type:varchar(50)"`
	ResourceID   *uuid.UUID `json:"resource_id,omitempty" gorm:"type:uuid"`

	// Change details
	OldStatus *string `json:"old_status,omitempty" gorm:"type:varchar(50)"`
	NewStatus *string `json:"new_status,omitempty" gorm:"type:varchar(50)"`
	Changes   *string `json:"changes,omitempty" gorm:"type:jsonb"`

	// Request details
	IPAddress *string `json:"ip_address,omitempty" gorm:"type:inet"`
	UserAgent *string `json:"user_agent,omitempty" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at" gorm:"default:now()"`

	// Relationships
	KYCSubmission *KYCSubmission `json:"kyc_submission,omitempty" gorm:"foreignKey:KYCSubmissionID"`
	Merchant      *Merchant      `json:"merchant,omitempty" gorm:"foreignKey:MerchantID"`
}

// TableName specifies the table name for GORM
func (KYCAuditLog) TableName() string {
	return "kyc_audit_logs"
}

// IsSystemAction checks if this is a system-generated action
func (a *KYCAuditLog) IsSystemAction() bool {
	return a.ActorType == ActorTypeSystem
}

// IsAdminAction checks if this is an admin action
func (a *KYCAuditLog) IsAdminAction() bool {
	return a.ActorType == ActorTypeAdmin
}

// IsMerchantAction checks if this is a merchant action
func (a *KYCAuditLog) IsMerchantAction() bool {
	return a.ActorType == ActorTypeMerchant
}
