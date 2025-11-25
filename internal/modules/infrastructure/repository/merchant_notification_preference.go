package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/lib/pq"
)

var (
	// ErrPreferenceNotFound is returned when a merchant notification preference is not found
	ErrPreferenceNotFound = errors.New("merchant notification preference not found")
	// ErrPreferenceAlreadyExists is returned when a preference already exists
	ErrPreferenceAlreadyExists = errors.New("merchant notification preference already exists")
)

// MerchantNotificationPreferenceRepository handles database operations for merchant notification preferences
type MerchantNotificationPreferenceRepository struct {
	db *sql.DB
}

// NewMerchantNotificationPreferenceRepository creates a new merchant notification preference repository
func NewMerchantNotificationPreferenceRepository(db *sql.DB) *MerchantNotificationPreferenceRepository {
	return &MerchantNotificationPreferenceRepository{
		db: db,
	}
}

// Create inserts a new merchant notification preference
func (r *MerchantNotificationPreferenceRepository) Create(pref *model.MerchantNotificationPreference) error {
	if pref == nil {
		return errors.New("merchant notification preference cannot be nil")
	}

	// Generate UUID if not provided
	if pref.ID == uuid.Nil {
		pref.ID = uuid.New()
	}

	query := `
		INSERT INTO merchant_notification_preferences (
			id, merchant_id, channel,
			enabled, subscribed_events,
			config, rate_limit_per_hour, rate_limit_per_day,
			is_test_mode, test_config,
			last_notification_at, total_notifications_sent,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if pref.CreatedAt.IsZero() {
		pref.CreatedAt = now
	}
	if pref.UpdatedAt.IsZero() {
		pref.UpdatedAt = now
	}

	err := r.db.QueryRow(
		query,
		pref.ID,
		pref.MerchantID,
		pref.Channel,
		pref.Enabled,
		pq.Array(pref.SubscribedEvents),
		pref.Config,
		pref.RateLimitPerHour,
		pref.RateLimitPerDay,
		pref.IsTestMode,
		pref.TestConfig,
		pref.LastNotificationAt,
		pref.TotalNotificationsSent,
		pref.CreatedAt,
		pref.UpdatedAt,
	).Scan(&pref.ID, &pref.CreatedAt, &pref.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation
		if err.Error() == `pq: duplicate key value violates unique constraint "idx_merchant_channel"` {
			return ErrPreferenceAlreadyExists
		}
		return fmt.Errorf("failed to create merchant notification preference: %w", err)
	}

	return nil
}

// GetByID retrieves a merchant notification preference by ID
func (r *MerchantNotificationPreferenceRepository) GetByID(id uuid.UUID) (*model.MerchantNotificationPreference, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid preference ID")
	}

	query := `
		SELECT
			id, merchant_id, channel,
			enabled, subscribed_events,
			config, rate_limit_per_hour, rate_limit_per_day,
			is_test_mode, test_config,
			last_notification_at, total_notifications_sent,
			created_at, updated_at
		FROM merchant_notification_preferences
		WHERE id = $1
	`

	pref := &model.MerchantNotificationPreference{}
	err := r.db.QueryRow(query, id).Scan(
		&pref.ID,
		&pref.MerchantID,
		&pref.Channel,
		&pref.Enabled,
		pq.Array(&pref.SubscribedEvents),
		&pref.Config,
		&pref.RateLimitPerHour,
		&pref.RateLimitPerDay,
		&pref.IsTestMode,
		&pref.TestConfig,
		&pref.LastNotificationAt,
		&pref.TotalNotificationsSent,
		&pref.CreatedAt,
		&pref.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPreferenceNotFound
		}
		return nil, fmt.Errorf("failed to get merchant notification preference by ID: %w", err)
	}

	return pref, nil
}

// GetByMerchantAndChannel retrieves a preference by merchant ID and channel
func (r *MerchantNotificationPreferenceRepository) GetByMerchantAndChannel(merchantID uuid.UUID, channel model.NotificationChannel) (*model.MerchantNotificationPreference, error) {
	if merchantID == uuid.Nil {
		return nil, errors.New("merchant ID cannot be empty")
	}

	query := `
		SELECT
			id, merchant_id, channel,
			enabled, subscribed_events,
			config, rate_limit_per_hour, rate_limit_per_day,
			is_test_mode, test_config,
			last_notification_at, total_notifications_sent,
			created_at, updated_at
		FROM merchant_notification_preferences
		WHERE merchant_id = $1 AND channel = $2
	`

	pref := &model.MerchantNotificationPreference{}
	err := r.db.QueryRow(query, merchantID, channel).Scan(
		&pref.ID,
		&pref.MerchantID,
		&pref.Channel,
		&pref.Enabled,
		pq.Array(&pref.SubscribedEvents),
		&pref.Config,
		&pref.RateLimitPerHour,
		&pref.RateLimitPerDay,
		&pref.IsTestMode,
		&pref.TestConfig,
		&pref.LastNotificationAt,
		&pref.TotalNotificationsSent,
		&pref.CreatedAt,
		&pref.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPreferenceNotFound
		}
		return nil, fmt.Errorf("failed to get merchant notification preference: %w", err)
	}

	return pref, nil
}

// GetByMerchantID retrieves all notification preferences for a merchant
func (r *MerchantNotificationPreferenceRepository) GetByMerchantID(merchantID uuid.UUID) ([]*model.MerchantNotificationPreference, error) {
	if merchantID == uuid.Nil {
		return nil, errors.New("merchant ID cannot be empty")
	}

	query := `
		SELECT
			id, merchant_id, channel,
			enabled, subscribed_events,
			config, rate_limit_per_hour, rate_limit_per_day,
			is_test_mode, test_config,
			last_notification_at, total_notifications_sent,
			created_at, updated_at
		FROM merchant_notification_preferences
		WHERE merchant_id = $1
		ORDER BY channel
	`

	rows, err := r.db.Query(query, merchantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merchant notification preferences: %w", err)
	}
	defer rows.Close()

	prefs := make([]*model.MerchantNotificationPreference, 0)
	for rows.Next() {
		pref := &model.MerchantNotificationPreference{}
		err := rows.Scan(
			&pref.ID,
			&pref.MerchantID,
			&pref.Channel,
			&pref.Enabled,
			pq.Array(&pref.SubscribedEvents),
			&pref.Config,
			&pref.RateLimitPerHour,
			&pref.RateLimitPerDay,
			&pref.IsTestMode,
			&pref.TestConfig,
			&pref.LastNotificationAt,
			&pref.TotalNotificationsSent,
			&pref.CreatedAt,
			&pref.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan merchant notification preference: %w", err)
		}
		prefs = append(prefs, pref)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating merchant notification preferences: %w", err)
	}

	return prefs, nil
}

// GetEnabledByMerchantID retrieves enabled notification preferences for a merchant
func (r *MerchantNotificationPreferenceRepository) GetEnabledByMerchantID(merchantID uuid.UUID) ([]*model.MerchantNotificationPreference, error) {
	if merchantID == uuid.Nil {
		return nil, errors.New("merchant ID cannot be empty")
	}

	query := `
		SELECT
			id, merchant_id, channel,
			enabled, subscribed_events,
			config, rate_limit_per_hour, rate_limit_per_day,
			is_test_mode, test_config,
			last_notification_at, total_notifications_sent,
			created_at, updated_at
		FROM merchant_notification_preferences
		WHERE merchant_id = $1 AND enabled = TRUE
		ORDER BY channel
	`

	rows, err := r.db.Query(query, merchantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled merchant notification preferences: %w", err)
	}
	defer rows.Close()

	prefs := make([]*model.MerchantNotificationPreference, 0)
	for rows.Next() {
		pref := &model.MerchantNotificationPreference{}
		err := rows.Scan(
			&pref.ID,
			&pref.MerchantID,
			&pref.Channel,
			&pref.Enabled,
			pq.Array(&pref.SubscribedEvents),
			&pref.Config,
			&pref.RateLimitPerHour,
			&pref.RateLimitPerDay,
			&pref.IsTestMode,
			&pref.TestConfig,
			&pref.LastNotificationAt,
			&pref.TotalNotificationsSent,
			&pref.CreatedAt,
			&pref.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan merchant notification preference: %w", err)
		}
		prefs = append(prefs, pref)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating merchant notification preferences: %w", err)
	}

	return prefs, nil
}

// Update updates an existing merchant notification preference
func (r *MerchantNotificationPreferenceRepository) Update(pref *model.MerchantNotificationPreference) error {
	if pref == nil {
		return errors.New("merchant notification preference cannot be nil")
	}
	if pref.ID == uuid.Nil {
		return errors.New("invalid preference ID")
	}

	query := `
		UPDATE merchant_notification_preferences SET
			enabled = $2,
			subscribed_events = $3,
			config = $4,
			rate_limit_per_hour = $5,
			rate_limit_per_day = $6,
			is_test_mode = $7,
			test_config = $8,
			last_notification_at = $9,
			total_notifications_sent = $10,
			updated_at = $11
		WHERE id = $1
		RETURNING updated_at
	`

	pref.UpdatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		pref.ID,
		pref.Enabled,
		pq.Array(pref.SubscribedEvents),
		pref.Config,
		pref.RateLimitPerHour,
		pref.RateLimitPerDay,
		pref.IsTestMode,
		pref.TestConfig,
		pref.LastNotificationAt,
		pref.TotalNotificationsSent,
		pref.UpdatedAt,
	).Scan(&pref.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrPreferenceNotFound
		}
		return fmt.Errorf("failed to update merchant notification preference: %w", err)
	}

	return nil
}

// Enable enables a notification channel for a merchant
func (r *MerchantNotificationPreferenceRepository) Enable(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid preference ID")
	}

	query := `
		UPDATE merchant_notification_preferences
		SET enabled = TRUE, updated_at = $2
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to enable notification channel: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPreferenceNotFound
	}

	return nil
}

// Disable disables a notification channel for a merchant
func (r *MerchantNotificationPreferenceRepository) Disable(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid preference ID")
	}

	query := `
		UPDATE merchant_notification_preferences
		SET enabled = FALSE, updated_at = $2
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to disable notification channel: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPreferenceNotFound
	}

	return nil
}

// IncrementNotificationCount increments the notification sent counter
func (r *MerchantNotificationPreferenceRepository) IncrementNotificationCount(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid preference ID")
	}

	query := `
		UPDATE merchant_notification_preferences
		SET
			total_notifications_sent = total_notifications_sent + 1,
			last_notification_at = $2,
			updated_at = $2
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(query, id, sql.NullTime{Time: now, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to increment notification count: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPreferenceNotFound
	}

	return nil
}

// Delete deletes a merchant notification preference
func (r *MerchantNotificationPreferenceRepository) Delete(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid preference ID")
	}

	query := `DELETE FROM merchant_notification_preferences WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete merchant notification preference: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPreferenceNotFound
	}

	return nil
}

// Count returns the total number of merchant notification preferences
func (r *MerchantNotificationPreferenceRepository) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM merchant_notification_preferences`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count merchant notification preferences: %w", err)
	}

	return count, nil
}

// CountByChannel returns the count of preferences by channel
func (r *MerchantNotificationPreferenceRepository) CountByChannel(channel model.NotificationChannel) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM merchant_notification_preferences
		WHERE channel = $1 AND enabled = TRUE
	`

	var count int64
	err := r.db.QueryRow(query, channel).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count preferences by channel: %w", err)
	}

	return count, nil
}
