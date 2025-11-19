package model

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// NotificationChannel represents the delivery channel for notifications
type NotificationChannel string

const (
	NotificationChannelEmail    NotificationChannel = "email"
	NotificationChannelTelegram NotificationChannel = "telegram"
	NotificationChannelZalo     NotificationChannel = "zalo"
	NotificationChannelWebhook  NotificationChannel = "webhook"
	NotificationChannelSpeaker  NotificationChannel = "speaker"
)

// NotificationStatus represents the delivery status
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusDelivered NotificationStatus = "delivered"
	NotificationStatusFailed    NotificationStatus = "failed"
	NotificationStatusRetrying  NotificationStatus = "retrying"
)

// NotificationLog tracks all notification attempts (PRD v2.2 - Notification Center)
type NotificationLog struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`

	// Target Information
	PaymentID  sql.NullString `gorm:"type:uuid;index" json:"payment_id,omitempty"`
	MerchantID uuid.UUID      `gorm:"type:uuid;not null;index" json:"merchant_id"`

	// Notification Details
	Channel   NotificationChannel `gorm:"type:varchar(50);not null;index" json:"channel"`
	EventType string              `gorm:"type:varchar(50);not null;index" json:"event_type"`
	Recipient sql.NullString      `gorm:"type:varchar(255)" json:"recipient,omitempty"`

	// Delivery Status
	Status      NotificationStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	SentAt      sql.NullTime       `gorm:"type:timestamp" json:"sent_at,omitempty"`
	DeliveredAt sql.NullTime       `gorm:"type:timestamp" json:"delivered_at,omitempty"`
	FailedAt    sql.NullTime       `gorm:"type:timestamp" json:"failed_at,omitempty"`
	ErrorMessage sql.NullString    `gorm:"type:text" json:"error_message,omitempty"`
	ErrorCode   sql.NullString     `gorm:"type:varchar(50)" json:"error_code,omitempty"`

	// Retry Logic
	RetryCount  int          `gorm:"default:0" json:"retry_count"`
	MaxRetries  int          `gorm:"default:3" json:"max_retries"`
	NextRetryAt sql.NullTime `gorm:"type:timestamp;index" json:"next_retry_at,omitempty"`

	// Message Content
	Subject    sql.NullString `gorm:"type:varchar(500)" json:"subject,omitempty"` // For email
	Message    sql.NullString `gorm:"type:text" json:"message,omitempty"`
	TemplateID sql.NullString `gorm:"type:varchar(100)" json:"template_id,omitempty"`

	// Metadata
	ProviderMessageID sql.NullString         `gorm:"type:varchar(255)" json:"provider_message_id,omitempty"`
	ProviderResponse  map[string]interface{} `gorm:"type:jsonb" json:"provider_response,omitempty"`
	Metadata          map[string]interface{} `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`

	// Audit fields
	CreatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;index" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name
func (NotificationLog) TableName() string {
	return "notification_logs"
}

// BeforeCreate hook
func (nl *NotificationLog) BeforeCreate() error {
	if nl.ID == uuid.Nil {
		nl.ID = uuid.New()
	}
	return nil
}

// MarkAsSent updates status to sent
func (nl *NotificationLog) MarkAsSent(providerMessageID string) {
	nl.Status = NotificationStatusSent
	nl.SentAt = sql.NullTime{Time: time.Now(), Valid: true}
	if providerMessageID != "" {
		nl.ProviderMessageID = sql.NullString{String: providerMessageID, Valid: true}
	}
}

// MarkAsDelivered updates status to delivered
func (nl *NotificationLog) MarkAsDelivered() {
	nl.Status = NotificationStatusDelivered
	nl.DeliveredAt = sql.NullTime{Time: time.Now(), Valid: true}
}

// MarkAsFailed updates status to failed
func (nl *NotificationLog) MarkAsFailed(errorMessage, errorCode string) {
	nl.Status = NotificationStatusFailed
	nl.FailedAt = sql.NullTime{Time: time.Now(), Valid: true}
	nl.ErrorMessage = sql.NullString{String: errorMessage, Valid: true}
	if errorCode != "" {
		nl.ErrorCode = sql.NullString{String: errorCode, Valid: true}
	}
}

// ScheduleRetry schedules the next retry attempt
func (nl *NotificationLog) ScheduleRetry(delay time.Duration) {
	nl.Status = NotificationStatusRetrying
	nl.RetryCount++
	nl.NextRetryAt = sql.NullTime{Time: time.Now().Add(delay), Valid: true}
}

// CanRetry returns true if retry is allowed
func (nl *NotificationLog) CanRetry() bool {
	return nl.RetryCount < nl.MaxRetries
}

// MerchantNotificationPreference stores merchant's notification channel configuration
type MerchantNotificationPreference struct {
	ID         uuid.UUID           `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MerchantID uuid.UUID           `gorm:"type:uuid;not null;uniqueIndex:idx_merchant_channel" json:"merchant_id"`
	Channel    NotificationChannel `gorm:"type:varchar(50);not null;uniqueIndex:idx_merchant_channel;index" json:"channel"`

	// Channel Enable/Disable
	Enabled bool `gorm:"default:true;index" json:"enabled"`

	// Event Subscriptions
	SubscribedEvents pq.StringArray `gorm:"type:text[];default:'{payment_confirmed,payment_expired,payment_failed}'" json:"subscribed_events"`

	// Channel-Specific Configuration
	Config map[string]interface{} `gorm:"type:jsonb;default:'{}'" json:"config"`

	// Rate Limiting
	RateLimitPerHour sql.NullInt32 `gorm:"type:integer" json:"rate_limit_per_hour,omitempty"`
	RateLimitPerDay  sql.NullInt32 `gorm:"type:integer" json:"rate_limit_per_day,omitempty"`

	// Testing
	IsTestMode bool                   `gorm:"default:false" json:"is_test_mode"`
	TestConfig map[string]interface{} `gorm:"type:jsonb" json:"test_config,omitempty"`

	// Activity Tracking
	LastNotificationAt       sql.NullTime `gorm:"type:timestamp" json:"last_notification_at,omitempty"`
	TotalNotificationsSent   int          `gorm:"default:0" json:"total_notifications_sent"`

	// Audit fields
	CreatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name
func (MerchantNotificationPreference) TableName() string {
	return "merchant_notification_preferences"
}

// BeforeCreate hook
func (mnp *MerchantNotificationPreference) BeforeCreate() error {
	if mnp.ID == uuid.Nil {
		mnp.ID = uuid.New()
	}
	return nil
}

// IsSubscribedToEvent returns true if merchant is subscribed to this event
func (mnp *MerchantNotificationPreference) IsSubscribedToEvent(eventType string) bool {
	for _, event := range mnp.SubscribedEvents {
		if event == eventType {
			return true
		}
	}
	return false
}

// GetConfigValue retrieves a configuration value
func (mnp *MerchantNotificationPreference) GetConfigValue(key string) (interface{}, bool) {
	if mnp.Config == nil {
		return nil, false
	}
	val, ok := mnp.Config[key]
	return val, ok
}

// SetConfigValue sets a configuration value
func (mnp *MerchantNotificationPreference) SetConfigValue(key string, value interface{}) {
	if mnp.Config == nil {
		mnp.Config = make(map[string]interface{})
	}
	mnp.Config[key] = value
}

// IncrementNotificationCount increments the sent notification counter
func (mnp *MerchantNotificationPreference) IncrementNotificationCount() {
	mnp.TotalNotificationsSent++
	mnp.LastNotificationAt = sql.NullTime{Time: time.Now(), Valid: true}
}
