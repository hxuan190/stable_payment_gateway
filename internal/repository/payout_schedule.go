package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

var (
	// ErrPayoutScheduleNotFound is returned when a payout schedule is not found
	ErrPayoutScheduleNotFound = errors.New("payout schedule not found")
	// ErrPayoutScheduleAlreadyExists is returned when a payout schedule already exists for this merchant
	ErrPayoutScheduleAlreadyExists = errors.New("payout schedule already exists for this merchant")
)

// PayoutScheduleRepository handles database operations for payout schedules
type PayoutScheduleRepository struct {
	db *sql.DB
}

// NewPayoutScheduleRepository creates a new payout schedule repository
func NewPayoutScheduleRepository(db *sql.DB) *PayoutScheduleRepository {
	return &PayoutScheduleRepository{
		db: db,
	}
}

// Create inserts a new payout schedule
func (r *PayoutScheduleRepository) Create(schedule *model.PayoutSchedule) error {
	if schedule == nil {
		return errors.New("payout schedule cannot be nil")
	}

	// Generate UUID if not provided
	if schedule.ID == uuid.Nil {
		schedule.ID = uuid.New()
	}

	query := `
		INSERT INTO payout_schedules (
			id, merchant_id,
			scheduled_enabled, scheduled_frequency, scheduled_day_of_week,
			scheduled_day_of_month, scheduled_time, scheduled_timezone,
			scheduled_withdraw_percentage,
			threshold_enabled, threshold_usdt, threshold_withdraw_percentage,
			last_scheduled_payout_at, last_threshold_payout_at,
			next_scheduled_payout_at, is_suspended, suspended_at, suspended_reason,
			metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if schedule.CreatedAt.IsZero() {
		schedule.CreatedAt = now
	}
	if schedule.UpdatedAt.IsZero() {
		schedule.UpdatedAt = now
	}

	err := r.db.QueryRow(
		query,
		schedule.ID,
		schedule.MerchantID,
		schedule.ScheduledEnabled,
		schedule.ScheduledFrequency,
		schedule.ScheduledDayOfWeek,
		schedule.ScheduledDayOfMonth,
		schedule.ScheduledTime,
		schedule.ScheduledTimezone,
		schedule.ScheduledWithdrawPercentage,
		schedule.ThresholdEnabled,
		schedule.ThresholdUSDT,
		schedule.ThresholdWithdrawPercentage,
		schedule.LastScheduledPayoutAt,
		schedule.LastThresholdPayoutAt,
		schedule.NextScheduledPayoutAt,
		schedule.IsSuspended,
		schedule.SuspendedAt,
		schedule.SuspendedReason,
		schedule.Metadata,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	).Scan(&schedule.ID, &schedule.CreatedAt, &schedule.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation
		if err.Error() == `pq: duplicate key value violates unique constraint "payout_schedules_merchant_id_key"` {
			return ErrPayoutScheduleAlreadyExists
		}
		return fmt.Errorf("failed to create payout schedule: %w", err)
	}

	return nil
}

// GetByID retrieves a payout schedule by ID
func (r *PayoutScheduleRepository) GetByID(id uuid.UUID) (*model.PayoutSchedule, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid payout schedule ID")
	}

	query := `
		SELECT
			id, merchant_id,
			scheduled_enabled, scheduled_frequency, scheduled_day_of_week,
			scheduled_day_of_month, scheduled_time, scheduled_timezone,
			scheduled_withdraw_percentage,
			threshold_enabled, threshold_usdt, threshold_withdraw_percentage,
			last_scheduled_payout_at, last_threshold_payout_at,
			next_scheduled_payout_at, is_suspended, suspended_at, suspended_reason,
			metadata, created_at, updated_at
		FROM payout_schedules
		WHERE id = $1
	`

	schedule := &model.PayoutSchedule{}
	err := r.db.QueryRow(query, id).Scan(
		&schedule.ID,
		&schedule.MerchantID,
		&schedule.ScheduledEnabled,
		&schedule.ScheduledFrequency,
		&schedule.ScheduledDayOfWeek,
		&schedule.ScheduledDayOfMonth,
		&schedule.ScheduledTime,
		&schedule.ScheduledTimezone,
		&schedule.ScheduledWithdrawPercentage,
		&schedule.ThresholdEnabled,
		&schedule.ThresholdUSDT,
		&schedule.ThresholdWithdrawPercentage,
		&schedule.LastScheduledPayoutAt,
		&schedule.LastThresholdPayoutAt,
		&schedule.NextScheduledPayoutAt,
		&schedule.IsSuspended,
		&schedule.SuspendedAt,
		&schedule.SuspendedReason,
		&schedule.Metadata,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPayoutScheduleNotFound
		}
		return nil, fmt.Errorf("failed to get payout schedule by ID: %w", err)
	}

	return schedule, nil
}

// GetByMerchantID retrieves a payout schedule by merchant ID
func (r *PayoutScheduleRepository) GetByMerchantID(merchantID uuid.UUID) (*model.PayoutSchedule, error) {
	if merchantID == uuid.Nil {
		return nil, errors.New("merchant ID cannot be empty")
	}

	query := `
		SELECT
			id, merchant_id,
			scheduled_enabled, scheduled_frequency, scheduled_day_of_week,
			scheduled_day_of_month, scheduled_time, scheduled_timezone,
			scheduled_withdraw_percentage,
			threshold_enabled, threshold_usdt, threshold_withdraw_percentage,
			last_scheduled_payout_at, last_threshold_payout_at,
			next_scheduled_payout_at, is_suspended, suspended_at, suspended_reason,
			metadata, created_at, updated_at
		FROM payout_schedules
		WHERE merchant_id = $1
	`

	schedule := &model.PayoutSchedule{}
	err := r.db.QueryRow(query, merchantID).Scan(
		&schedule.ID,
		&schedule.MerchantID,
		&schedule.ScheduledEnabled,
		&schedule.ScheduledFrequency,
		&schedule.ScheduledDayOfWeek,
		&schedule.ScheduledDayOfMonth,
		&schedule.ScheduledTime,
		&schedule.ScheduledTimezone,
		&schedule.ScheduledWithdrawPercentage,
		&schedule.ThresholdEnabled,
		&schedule.ThresholdUSDT,
		&schedule.ThresholdWithdrawPercentage,
		&schedule.LastScheduledPayoutAt,
		&schedule.LastThresholdPayoutAt,
		&schedule.NextScheduledPayoutAt,
		&schedule.IsSuspended,
		&schedule.SuspendedAt,
		&schedule.SuspendedReason,
		&schedule.Metadata,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPayoutScheduleNotFound
		}
		return nil, fmt.Errorf("failed to get payout schedule by merchant ID: %w", err)
	}

	return schedule, nil
}

// Update updates an existing payout schedule
func (r *PayoutScheduleRepository) Update(schedule *model.PayoutSchedule) error {
	if schedule == nil {
		return errors.New("payout schedule cannot be nil")
	}
	if schedule.ID == uuid.Nil {
		return errors.New("invalid payout schedule ID")
	}

	query := `
		UPDATE payout_schedules SET
			scheduled_enabled = $2,
			scheduled_frequency = $3,
			scheduled_day_of_week = $4,
			scheduled_day_of_month = $5,
			scheduled_time = $6,
			scheduled_timezone = $7,
			scheduled_withdraw_percentage = $8,
			threshold_enabled = $9,
			threshold_usdt = $10,
			threshold_withdraw_percentage = $11,
			last_scheduled_payout_at = $12,
			last_threshold_payout_at = $13,
			next_scheduled_payout_at = $14,
			is_suspended = $15,
			suspended_at = $16,
			suspended_reason = $17,
			metadata = $18,
			updated_at = $19
		WHERE id = $1
		RETURNING updated_at
	`

	schedule.UpdatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		schedule.ID,
		schedule.ScheduledEnabled,
		schedule.ScheduledFrequency,
		schedule.ScheduledDayOfWeek,
		schedule.ScheduledDayOfMonth,
		schedule.ScheduledTime,
		schedule.ScheduledTimezone,
		schedule.ScheduledWithdrawPercentage,
		schedule.ThresholdEnabled,
		schedule.ThresholdUSDT,
		schedule.ThresholdWithdrawPercentage,
		schedule.LastScheduledPayoutAt,
		schedule.LastThresholdPayoutAt,
		schedule.NextScheduledPayoutAt,
		schedule.IsSuspended,
		schedule.SuspendedAt,
		schedule.SuspendedReason,
		schedule.Metadata,
		schedule.UpdatedAt,
	).Scan(&schedule.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrPayoutScheduleNotFound
		}
		return fmt.Errorf("failed to update payout schedule: %w", err)
	}

	return nil
}

// Suspend suspends a payout schedule
func (r *PayoutScheduleRepository) Suspend(id uuid.UUID, reason string) error {
	if id == uuid.Nil {
		return errors.New("invalid payout schedule ID")
	}
	if reason == "" {
		return errors.New("suspend reason cannot be empty")
	}

	query := `
		UPDATE payout_schedules
		SET
			is_suspended = TRUE,
			suspended_at = $2,
			suspended_reason = $3,
			updated_at = $2
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(
		query,
		id,
		sql.NullTime{Time: now, Valid: true},
		sql.NullString{String: reason, Valid: true},
	)
	if err != nil {
		return fmt.Errorf("failed to suspend payout schedule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPayoutScheduleNotFound
	}

	return nil
}

// Resume resumes a suspended payout schedule
func (r *PayoutScheduleRepository) Resume(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid payout schedule ID")
	}

	query := `
		UPDATE payout_schedules
		SET
			is_suspended = FALSE,
			suspended_at = NULL,
			suspended_reason = NULL,
			updated_at = $2
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to resume payout schedule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPayoutScheduleNotFound
	}

	return nil
}

// UpdateLastScheduledPayout updates the last scheduled payout timestamp
func (r *PayoutScheduleRepository) UpdateLastScheduledPayout(id uuid.UUID, payoutTime time.Time) error {
	if id == uuid.Nil {
		return errors.New("invalid payout schedule ID")
	}

	query := `
		UPDATE payout_schedules
		SET
			last_scheduled_payout_at = $2,
			updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id, sql.NullTime{Time: payoutTime, Valid: true}, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update last scheduled payout: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPayoutScheduleNotFound
	}

	return nil
}

// UpdateLastThresholdPayout updates the last threshold payout timestamp
func (r *PayoutScheduleRepository) UpdateLastThresholdPayout(id uuid.UUID, payoutTime time.Time) error {
	if id == uuid.Nil {
		return errors.New("invalid payout schedule ID")
	}

	query := `
		UPDATE payout_schedules
		SET
			last_threshold_payout_at = $2,
			updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id, sql.NullTime{Time: payoutTime, Valid: true}, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update last threshold payout: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPayoutScheduleNotFound
	}

	return nil
}

// ListScheduledPayoutsDue retrieves payout schedules that have scheduled payouts due
func (r *PayoutScheduleRepository) ListScheduledPayoutsDue() ([]*model.PayoutSchedule, error) {
	query := `
		SELECT
			id, merchant_id,
			scheduled_enabled, scheduled_frequency, scheduled_day_of_week,
			scheduled_day_of_month, scheduled_time, scheduled_timezone,
			scheduled_withdraw_percentage,
			threshold_enabled, threshold_usdt, threshold_withdraw_percentage,
			last_scheduled_payout_at, last_threshold_payout_at,
			next_scheduled_payout_at, is_suspended, suspended_at, suspended_reason,
			metadata, created_at, updated_at
		FROM payout_schedules
		WHERE scheduled_enabled = TRUE
		  AND is_suspended = FALSE
		  AND next_scheduled_payout_at IS NOT NULL
		  AND next_scheduled_payout_at <= $1
		ORDER BY next_scheduled_payout_at ASC
	`

	rows, err := r.db.Query(query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to list scheduled payouts due: %w", err)
	}
	defer rows.Close()

	schedules := make([]*model.PayoutSchedule, 0)
	for rows.Next() {
		schedule := &model.PayoutSchedule{}
		err := rows.Scan(
			&schedule.ID,
			&schedule.MerchantID,
			&schedule.ScheduledEnabled,
			&schedule.ScheduledFrequency,
			&schedule.ScheduledDayOfWeek,
			&schedule.ScheduledDayOfMonth,
			&schedule.ScheduledTime,
			&schedule.ScheduledTimezone,
			&schedule.ScheduledWithdrawPercentage,
			&schedule.ThresholdEnabled,
			&schedule.ThresholdUSDT,
			&schedule.ThresholdWithdrawPercentage,
			&schedule.LastScheduledPayoutAt,
			&schedule.LastThresholdPayoutAt,
			&schedule.NextScheduledPayoutAt,
			&schedule.IsSuspended,
			&schedule.SuspendedAt,
			&schedule.SuspendedReason,
			&schedule.Metadata,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payout schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payout schedules: %w", err)
	}

	return schedules, nil
}

// ListThresholdEnabled retrieves all payout schedules with threshold enabled
func (r *PayoutScheduleRepository) ListThresholdEnabled() ([]*model.PayoutSchedule, error) {
	query := `
		SELECT
			id, merchant_id,
			scheduled_enabled, scheduled_frequency, scheduled_day_of_week,
			scheduled_day_of_month, scheduled_time, scheduled_timezone,
			scheduled_withdraw_percentage,
			threshold_enabled, threshold_usdt, threshold_withdraw_percentage,
			last_scheduled_payout_at, last_threshold_payout_at,
			next_scheduled_payout_at, is_suspended, suspended_at, suspended_reason,
			metadata, created_at, updated_at
		FROM payout_schedules
		WHERE threshold_enabled = TRUE AND is_suspended = FALSE
		ORDER BY threshold_usdt ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list threshold-enabled payout schedules: %w", err)
	}
	defer rows.Close()

	schedules := make([]*model.PayoutSchedule, 0)
	for rows.Next() {
		schedule := &model.PayoutSchedule{}
		err := rows.Scan(
			&schedule.ID,
			&schedule.MerchantID,
			&schedule.ScheduledEnabled,
			&schedule.ScheduledFrequency,
			&schedule.ScheduledDayOfWeek,
			&schedule.ScheduledDayOfMonth,
			&schedule.ScheduledTime,
			&schedule.ScheduledTimezone,
			&schedule.ScheduledWithdrawPercentage,
			&schedule.ThresholdEnabled,
			&schedule.ThresholdUSDT,
			&schedule.ThresholdWithdrawPercentage,
			&schedule.LastScheduledPayoutAt,
			&schedule.LastThresholdPayoutAt,
			&schedule.NextScheduledPayoutAt,
			&schedule.IsSuspended,
			&schedule.SuspendedAt,
			&schedule.SuspendedReason,
			&schedule.Metadata,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payout schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payout schedules: %w", err)
	}

	return schedules, nil
}

// Delete deletes a payout schedule
func (r *PayoutScheduleRepository) Delete(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid payout schedule ID")
	}

	query := `DELETE FROM payout_schedules WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete payout schedule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPayoutScheduleNotFound
	}

	return nil
}

// Count returns the total number of payout schedules
func (r *PayoutScheduleRepository) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM payout_schedules`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count payout schedules: %w", err)
	}

	return count, nil
}
