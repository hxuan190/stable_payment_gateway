package model

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// OTCPartnerLiquidity represents an OTC partner's VND liquidity pool
type OTCPartnerLiquidity struct {
	ID          string `json:"id" db:"id"`
	PartnerName string `json:"partner_name" db:"partner_name"`
	PartnerCode string `json:"partner_code" db:"partner_code"`

	// Balance tracking
	CurrentBalanceVND   decimal.Decimal `json:"current_balance_vnd" db:"current_balance_vnd"`
	AvailableBalanceVND decimal.Decimal `json:"available_balance_vnd" db:"available_balance_vnd"`
	ReservedBalanceVND  decimal.Decimal `json:"reserved_balance_vnd" db:"reserved_balance_vnd"`

	// Balance thresholds
	LowBalanceThresholdVND decimal.Decimal `json:"low_balance_threshold_vnd" db:"low_balance_threshold_vnd"`
	CriticalThresholdVND   decimal.Decimal `json:"critical_threshold_vnd" db:"critical_threshold_vnd"`

	// Status tracking
	Status            string         `json:"status" db:"status"`
	IsBalanceLow      bool           `json:"is_balance_low" db:"is_balance_low"`
	IsBalanceCritical bool           `json:"is_balance_critical" db:"is_balance_critical"`
	LastAlertSentAt   sql.NullTime   `json:"last_alert_sent_at,omitempty" db:"last_alert_sent_at"`
	AlertCountToday   int            `json:"alert_count_today" db:"alert_count_today"`

	// Partner details
	APIEndpoint   sql.NullString `json:"api_endpoint,omitempty" db:"api_endpoint"`
	APIKey        sql.NullString `json:"-" db:"api_key_encrypted" encrypt:"true"` // Encrypted, never exposed in JSON
	ContactPerson sql.NullString `json:"contact_person,omitempty" db:"contact_person"`
	ContactEmail  sql.NullString `json:"contact_email,omitempty" db:"contact_email"`
	ContactPhone  sql.NullString `json:"contact_phone,omitempty" db:"contact_phone"`

	// Operational hours
	OperatingHoursStart sql.NullTime   `json:"operating_hours_start,omitempty" db:"operating_hours_start"`
	OperatingHoursEnd   sql.NullTime   `json:"operating_hours_end,omitempty" db:"operating_hours_end"`
	Timezone            sql.NullString `json:"timezone,omitempty" db:"timezone"`
	Is247               bool           `json:"is_24_7" db:"is_24_7"`

	// Audit fields
	CreatedAt             time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time    `json:"updated_at" db:"updated_at"`
	LastBalanceCheckAt    sql.NullTime `json:"last_balance_check_at,omitempty" db:"last_balance_check_at"`
	LastBalanceUpdateAt   sql.NullTime `json:"last_balance_update_at,omitempty" db:"last_balance_update_at"`

	// Metadata
	Metadata JSONBMap       `json:"metadata,omitempty" db:"metadata"`
	Notes    sql.NullString `json:"notes,omitempty" db:"notes"`
}

// IsLowBalance returns true if current balance is below low threshold
func (o *OTCPartnerLiquidity) IsLowBalance() bool {
	return o.CurrentBalanceVND.LessThan(o.LowBalanceThresholdVND)
}

// IsCriticalBalance returns true if current balance is below critical threshold
func (o *OTCPartnerLiquidity) IsCriticalBalance() bool {
	return o.CurrentBalanceVND.LessThan(o.CriticalThresholdVND)
}

// CanFulfillPayout returns true if available balance can cover the payout amount
func (o *OTCPartnerLiquidity) CanFulfillPayout(amountVND decimal.Decimal) bool {
	return o.AvailableBalanceVND.GreaterThanOrEqual(amountVND)
}

// GetBalancePercentage returns the current balance as a percentage of low threshold
func (o *OTCPartnerLiquidity) GetBalancePercentage() decimal.Decimal {
	if o.LowBalanceThresholdVND.IsZero() {
		return decimal.Zero
	}
	return o.CurrentBalanceVND.Div(o.LowBalanceThresholdVND).Mul(decimal.NewFromInt(100))
}

// RequiresImmediateAlert returns true if alert should be sent immediately
func (o *OTCPartnerLiquidity) RequiresImmediateAlert() bool {
	// Send alert if:
	// 1. Balance is critical
	// 2. Balance is low and no alert sent in last hour
	if o.IsCriticalBalance() {
		return true
	}

	if o.IsLowBalance() {
		if !o.LastAlertSentAt.Valid {
			return true
		}
		// Send alert if last alert was more than 1 hour ago
		return time.Since(o.LastAlertSentAt.Time) > time.Hour
	}

	return false
}

// GetAlertMessage returns a formatted alert message
func (o *OTCPartnerLiquidity) GetAlertMessage() string {
	if o.IsCriticalBalance() {
		return fmt.Sprintf("🚨 CRITICAL: %s balance is critically low! Current: %s VND (Threshold: %s VND)",
			o.PartnerName,
			o.CurrentBalanceVND.String(),
			o.CriticalThresholdVND.String(),
		)
	}

	if o.IsLowBalance() {
		return fmt.Sprintf("⚠️ WARNING: %s balance is low! Current: %s VND (Threshold: %s VND)",
			o.PartnerName,
			o.CurrentBalanceVND.String(),
			o.LowBalanceThresholdVND.String(),
		)
	}

	return ""
}

// OTCLiquidityAlert represents a liquidity alert log
type OTCLiquidityAlert struct {
	ID                string          `json:"id" db:"id"`
	PartnerID         string          `json:"partner_id" db:"partner_id"`
	AlertType         string          `json:"alert_type" db:"alert_type"`
	AlertLevel        string          `json:"alert_level" db:"alert_level"`
	Message           string          `json:"message" db:"message"`
	CurrentBalanceVND decimal.Decimal `json:"current_balance_vnd" db:"current_balance_vnd"`
	ThresholdVND      decimal.Decimal `json:"threshold_vnd" db:"threshold_vnd"`

	// Notification tracking
	SentToTelegram bool `json:"sent_to_telegram" db:"sent_to_telegram"`
	SentToSlack    bool `json:"sent_to_slack" db:"sent_to_slack"`
	SentToEmail    bool `json:"sent_to_email" db:"sent_to_email"`

	// Resolution tracking
	IsResolved      bool           `json:"is_resolved" db:"is_resolved"`
	ResolvedAt      sql.NullTime   `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy      sql.NullString `json:"resolved_by,omitempty" db:"resolved_by"`
	ResolutionNotes sql.NullString `json:"resolution_notes,omitempty" db:"resolution_notes"`

	// Timestamp
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// OTCLiquidityHistory represents a balance change record
type OTCLiquidityHistory struct {
	ID                 string          `json:"id" db:"id"`
	PartnerID          string          `json:"partner_id" db:"partner_id"`
	PreviousBalanceVND decimal.Decimal `json:"previous_balance_vnd" db:"previous_balance_vnd"`
	NewBalanceVND      decimal.Decimal `json:"new_balance_vnd" db:"new_balance_vnd"`
	BalanceChangeVND   decimal.Decimal `json:"balance_change_vnd" db:"balance_change_vnd"`
	ChangeType         string          `json:"change_type" db:"change_type"`
	ChangeReason       sql.NullString  `json:"change_reason,omitempty" db:"change_reason"`
	ReferenceID        sql.NullString  `json:"reference_id,omitempty" db:"reference_id"`
	RecordedAt         time.Time       `json:"recorded_at" db:"recorded_at"`
}

import "fmt"
