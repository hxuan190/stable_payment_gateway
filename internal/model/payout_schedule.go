package model

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PayoutFrequency represents how often scheduled payouts occur
type PayoutFrequency string

const (
	PayoutFrequencyWeekly  PayoutFrequency = "weekly"
	PayoutFrequencyMonthly PayoutFrequency = "monthly"
)

// PayoutSchedule configures automatic payout scheduling (PRD v2.2 - Advanced Off-ramp)
type PayoutSchedule struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MerchantID uuid.UUID `gorm:"type:uuid;unique;not null" json:"merchant_id"` // One schedule per merchant

	// Scheduled Withdrawals Configuration
	ScheduledEnabled            bool              `gorm:"default:false;index" json:"scheduled_enabled"`
	ScheduledFrequency          sql.NullString    `gorm:"type:varchar(20);index" json:"scheduled_frequency,omitempty"` // 'weekly', 'monthly'
	ScheduledDayOfWeek          sql.NullInt32     `gorm:"type:integer" json:"scheduled_day_of_week,omitempty"`         // 0=Sunday, 6=Saturday
	ScheduledDayOfMonth         sql.NullInt32     `gorm:"type:integer" json:"scheduled_day_of_month,omitempty"`        // 1-31
	ScheduledTime               sql.NullTime      `gorm:"type:time;default:'10:00:00'" json:"scheduled_time,omitempty"`
	ScheduledTimezone           string            `gorm:"type:varchar(50);default:'Asia/Ho_Chi_Minh'" json:"scheduled_timezone"`
	ScheduledWithdrawPercentage int               `gorm:"type:integer;default:80" json:"scheduled_withdraw_percentage"` // 10-100%

	// Threshold-based Withdrawals Configuration
	ThresholdEnabled            bool            `gorm:"default:false;index" json:"threshold_enabled"`
	ThresholdUSDT               decimal.Decimal `gorm:"type:decimal(20,8)" json:"threshold_usdt,omitempty"`
	ThresholdWithdrawPercentage int             `gorm:"type:integer;default:90" json:"threshold_withdraw_percentage"` // 10-100%

	// Minimum/Maximum Withdrawal Amounts
	MinWithdrawalVND decimal.Decimal `gorm:"type:decimal(20,2);default:1000000" json:"min_withdrawal_vnd"` // 1M VND minimum
	MaxWithdrawalVND decimal.Decimal `gorm:"type:decimal(20,2)" json:"max_withdrawal_vnd,omitempty"`       // NULL = no limit

	// Activity Tracking
	LastTriggeredAt          sql.NullTime `gorm:"type:timestamp" json:"last_triggered_at,omitempty"`
	LastThresholdTriggeredAt sql.NullTime `gorm:"type:timestamp" json:"last_threshold_triggered_at,omitempty"`
	TotalScheduledPayouts    int          `gorm:"default:0" json:"total_scheduled_payouts"`
	TotalThresholdPayouts    int          `gorm:"default:0" json:"total_threshold_payouts"`

	// Suspension
	IsSuspended      bool           `gorm:"default:false;index" json:"is_suspended"`
	SuspendedAt      sql.NullTime   `gorm:"type:timestamp" json:"suspended_at,omitempty"`
	SuspensionReason sql.NullString `gorm:"type:text" json:"suspension_reason,omitempty"`

	// Metadata
	Metadata map[string]interface{} `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`

	// Audit fields
	CreatedAt time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
	CreatedBy sql.NullString `gorm:"type:varchar(100)" json:"created_by,omitempty"`
	UpdatedBy sql.NullString `gorm:"type:varchar(100)" json:"updated_by,omitempty"`
}

// TableName specifies the table name
func (PayoutSchedule) TableName() string {
	return "payout_schedules"
}

// BeforeCreate hook
func (ps *PayoutSchedule) BeforeCreate() error {
	if ps.ID == uuid.Nil {
		ps.ID = uuid.New()
	}
	return nil
}

// IsScheduledPayoutEnabled returns true if scheduled payouts are enabled and configured
func (ps *PayoutSchedule) IsScheduledPayoutEnabled() bool {
	return ps.ScheduledEnabled && ps.ScheduledFrequency.Valid && !ps.IsSuspended
}

// IsThresholdPayoutEnabled returns true if threshold-based payouts are enabled
func (ps *PayoutSchedule) IsThresholdPayoutEnabled() bool {
	return ps.ThresholdEnabled && ps.ThresholdUSDT.GreaterThan(decimal.Zero) && !ps.IsSuspended
}

// GetScheduledFrequency returns the payout frequency as PayoutFrequency type
func (ps *PayoutSchedule) GetScheduledFrequency() PayoutFrequency {
	if ps.ScheduledFrequency.Valid {
		return PayoutFrequency(ps.ScheduledFrequency.String)
	}
	return ""
}

// IsWeeklySchedule returns true if this is a weekly schedule
func (ps *PayoutSchedule) IsWeeklySchedule() bool {
	return ps.GetScheduledFrequency() == PayoutFrequencyWeekly
}

// IsMonthlySchedule returns true if this is a monthly schedule
func (ps *PayoutSchedule) IsMonthlySchedule() bool {
	return ps.GetScheduledFrequency() == PayoutFrequencyMonthly
}

// Suspend temporarily suspends automatic payouts
func (ps *PayoutSchedule) Suspend(reason string) {
	ps.IsSuspended = true
	ps.SuspendedAt = sql.NullTime{Time: time.Now(), Valid: true}
	ps.SuspensionReason = sql.NullString{String: reason, Valid: true}
}

// Resume resumes automatic payouts
func (ps *PayoutSchedule) Resume() {
	ps.IsSuspended = false
	ps.SuspendedAt = sql.NullTime{Valid: false}
	ps.SuspensionReason = sql.NullString{Valid: false}
}

// IncrementScheduledPayoutCount increments the scheduled payout counter
func (ps *PayoutSchedule) IncrementScheduledPayoutCount() {
	ps.TotalScheduledPayouts++
	ps.LastTriggeredAt = sql.NullTime{Time: time.Now(), Valid: true}
}

// IncrementThresholdPayoutCount increments the threshold payout counter
func (ps *PayoutSchedule) IncrementThresholdPayoutCount() {
	ps.TotalThresholdPayouts++
	ps.LastThresholdTriggeredAt = sql.NullTime{Time: time.Now(), Valid: true}
}

// CalculateWithdrawalAmount calculates the withdrawal amount based on available balance
func (ps *PayoutSchedule) CalculateWithdrawalAmount(availableBalance decimal.Decimal, isThreshold bool) decimal.Decimal {
	var percentage int
	if isThreshold {
		percentage = ps.ThresholdWithdrawPercentage
	} else {
		percentage = ps.ScheduledWithdrawPercentage
	}

	amount := availableBalance.Mul(decimal.NewFromInt(int64(percentage))).Div(decimal.NewFromInt(100))

	// Apply minimum withdrawal
	if amount.LessThan(ps.MinWithdrawalVND) {
		return decimal.Zero // Don't withdraw if below minimum
	}

	// Apply maximum withdrawal if set
	if ps.MaxWithdrawalVND.GreaterThan(decimal.Zero) && amount.GreaterThan(ps.MaxWithdrawalVND) {
		amount = ps.MaxWithdrawalVND
	}

	return amount
}

// ShouldTriggerThreshold checks if threshold payout should be triggered
func (ps *PayoutSchedule) ShouldTriggerThreshold(availableBalanceUSD decimal.Decimal) bool {
	return ps.IsThresholdPayoutEnabled() && availableBalanceUSD.GreaterThanOrEqual(ps.ThresholdUSDT)
}
