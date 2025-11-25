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
	// ErrNotificationLogNotFound is returned when a notification log is not found
	ErrNotificationLogNotFound = errors.New("notification log not found")
)

// NotificationLogRepository handles database operations for notification logs
type NotificationLogRepository struct {
	db *sql.DB
}

// NewNotificationLogRepository creates a new notification log repository
func NewNotificationLogRepository(db *sql.DB) *NotificationLogRepository {
	return &NotificationLogRepository{
		db: db,
	}
}

// Create inserts a new notification log
func (r *NotificationLogRepository) Create(log *model.NotificationLog) error {
	if log == nil {
		return errors.New("notification log cannot be nil")
	}

	// Generate UUID if not provided
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	query := `
		INSERT INTO notification_logs (
			id, payment_id, merchant_id,
			channel, event_type, recipient,
			status, sent_at, delivered_at, failed_at,
			error_message, error_code,
			retry_count, max_retries, next_retry_at,
			subject, message, template_id,
			provider_message_id, provider_response, metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23
		)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if log.CreatedAt.IsZero() {
		log.CreatedAt = now
	}
	if log.UpdatedAt.IsZero() {
		log.UpdatedAt = now
	}

	err := r.db.QueryRow(
		query,
		log.ID,
		log.PaymentID,
		log.MerchantID,
		log.Channel,
		log.EventType,
		log.Recipient,
		log.Status,
		log.SentAt,
		log.DeliveredAt,
		log.FailedAt,
		log.ErrorMessage,
		log.ErrorCode,
		log.RetryCount,
		log.MaxRetries,
		log.NextRetryAt,
		log.Subject,
		log.Message,
		log.TemplateID,
		log.ProviderMessageID,
		log.ProviderResponse,
		log.Metadata,
		log.CreatedAt,
		log.UpdatedAt,
	).Scan(&log.ID, &log.CreatedAt, &log.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create notification log: %w", err)
	}

	return nil
}

// GetByID retrieves a notification log by ID
func (r *NotificationLogRepository) GetByID(id uuid.UUID) (*model.NotificationLog, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid notification log ID")
	}

	query := `
		SELECT
			id, payment_id, merchant_id,
			channel, event_type, recipient,
			status, sent_at, delivered_at, failed_at,
			error_message, error_code,
			retry_count, max_retries, next_retry_at,
			subject, message, template_id,
			provider_message_id, provider_response, metadata,
			created_at, updated_at
		FROM notification_logs
		WHERE id = $1
	`

	log := &model.NotificationLog{}
	err := r.db.QueryRow(query, id).Scan(
		&log.ID,
		&log.PaymentID,
		&log.MerchantID,
		&log.Channel,
		&log.EventType,
		&log.Recipient,
		&log.Status,
		&log.SentAt,
		&log.DeliveredAt,
		&log.FailedAt,
		&log.ErrorMessage,
		&log.ErrorCode,
		&log.RetryCount,
		&log.MaxRetries,
		&log.NextRetryAt,
		&log.Subject,
		&log.Message,
		&log.TemplateID,
		&log.ProviderMessageID,
		&log.ProviderResponse,
		&log.Metadata,
		&log.CreatedAt,
		&log.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotificationLogNotFound
		}
		return nil, fmt.Errorf("failed to get notification log by ID: %w", err)
	}

	return log, nil
}

// GetByPaymentID retrieves all notification logs for a payment
func (r *NotificationLogRepository) GetByPaymentID(paymentID uuid.UUID) ([]*model.NotificationLog, error) {
	if paymentID == uuid.Nil {
		return nil, errors.New("payment ID cannot be empty")
	}

	query := `
		SELECT
			id, payment_id, merchant_id,
			channel, event_type, recipient,
			status, sent_at, delivered_at, failed_at,
			error_message, error_code,
			retry_count, max_retries, next_retry_at,
			subject, message, template_id,
			provider_message_id, provider_response, metadata,
			created_at, updated_at
		FROM notification_logs
		WHERE payment_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, sql.NullString{String: paymentID.String(), Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get notification logs by payment ID: %w", err)
	}
	defer rows.Close()

	logs := make([]*model.NotificationLog, 0)
	for rows.Next() {
		log := &model.NotificationLog{}
		err := rows.Scan(
			&log.ID,
			&log.PaymentID,
			&log.MerchantID,
			&log.Channel,
			&log.EventType,
			&log.Recipient,
			&log.Status,
			&log.SentAt,
			&log.DeliveredAt,
			&log.FailedAt,
			&log.ErrorMessage,
			&log.ErrorCode,
			&log.RetryCount,
			&log.MaxRetries,
			&log.NextRetryAt,
			&log.Subject,
			&log.Message,
			&log.TemplateID,
			&log.ProviderMessageID,
			&log.ProviderResponse,
			&log.Metadata,
			&log.CreatedAt,
			&log.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification log: %w", err)
		}
		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notification logs: %w", err)
	}

	return logs, nil
}

// GetByMerchantID retrieves notification logs for a merchant with pagination
func (r *NotificationLogRepository) GetByMerchantID(merchantID uuid.UUID, limit, offset int) ([]*model.NotificationLog, error) {
	if merchantID == uuid.Nil {
		return nil, errors.New("merchant ID cannot be empty")
	}

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, payment_id, merchant_id,
			channel, event_type, recipient,
			status, sent_at, delivered_at, failed_at,
			error_message, error_code,
			retry_count, max_retries, next_retry_at,
			subject, message, template_id,
			provider_message_id, provider_response, metadata,
			created_at, updated_at
		FROM notification_logs
		WHERE merchant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, merchantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification logs by merchant ID: %w", err)
	}
	defer rows.Close()

	logs := make([]*model.NotificationLog, 0)
	for rows.Next() {
		log := &model.NotificationLog{}
		err := rows.Scan(
			&log.ID,
			&log.PaymentID,
			&log.MerchantID,
			&log.Channel,
			&log.EventType,
			&log.Recipient,
			&log.Status,
			&log.SentAt,
			&log.DeliveredAt,
			&log.FailedAt,
			&log.ErrorMessage,
			&log.ErrorCode,
			&log.RetryCount,
			&log.MaxRetries,
			&log.NextRetryAt,
			&log.Subject,
			&log.Message,
			&log.TemplateID,
			&log.ProviderMessageID,
			&log.ProviderResponse,
			&log.Metadata,
			&log.CreatedAt,
			&log.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification log: %w", err)
		}
		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notification logs: %w", err)
	}

	return logs, nil
}

// Update updates an existing notification log
func (r *NotificationLogRepository) Update(log *model.NotificationLog) error {
	if log == nil {
		return errors.New("notification log cannot be nil")
	}
	if log.ID == uuid.Nil {
		return errors.New("invalid notification log ID")
	}

	query := `
		UPDATE notification_logs SET
			status = $2,
			sent_at = $3,
			delivered_at = $4,
			failed_at = $5,
			error_message = $6,
			error_code = $7,
			retry_count = $8,
			next_retry_at = $9,
			provider_message_id = $10,
			provider_response = $11,
			metadata = $12,
			updated_at = $13
		WHERE id = $1
		RETURNING updated_at
	`

	log.UpdatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		log.ID,
		log.Status,
		log.SentAt,
		log.DeliveredAt,
		log.FailedAt,
		log.ErrorMessage,
		log.ErrorCode,
		log.RetryCount,
		log.NextRetryAt,
		log.ProviderMessageID,
		log.ProviderResponse,
		log.Metadata,
		log.UpdatedAt,
	).Scan(&log.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotificationLogNotFound
		}
		return fmt.Errorf("failed to update notification log: %w", err)
	}

	return nil
}

// MarkAsSent updates the status to sent
func (r *NotificationLogRepository) MarkAsSent(id uuid.UUID, providerMessageID string) error {
	if id == uuid.Nil {
		return errors.New("invalid notification log ID")
	}

	query := `
		UPDATE notification_logs
		SET
			status = $2,
			sent_at = $3,
			provider_message_id = $4,
			updated_at = $3
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(
		query,
		id,
		model.NotificationStatusSent,
		sql.NullTime{Time: now, Valid: true},
		sql.NullString{String: providerMessageID, Valid: providerMessageID != ""},
	)
	if err != nil {
		return fmt.Errorf("failed to mark notification as sent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotificationLogNotFound
	}

	return nil
}

// MarkAsDelivered updates the status to delivered
func (r *NotificationLogRepository) MarkAsDelivered(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid notification log ID")
	}

	query := `
		UPDATE notification_logs
		SET
			status = $2,
			delivered_at = $3,
			updated_at = $3
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(query, id, model.NotificationStatusDelivered, sql.NullTime{Time: now, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to mark notification as delivered: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotificationLogNotFound
	}

	return nil
}

// MarkAsFailed updates the status to failed
func (r *NotificationLogRepository) MarkAsFailed(id uuid.UUID, errorMessage, errorCode string) error {
	if id == uuid.Nil {
		return errors.New("invalid notification log ID")
	}

	query := `
		UPDATE notification_logs
		SET
			status = $2,
			failed_at = $3,
			error_message = $4,
			error_code = $5,
			updated_at = $3
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(
		query,
		id,
		model.NotificationStatusFailed,
		sql.NullTime{Time: now, Valid: true},
		sql.NullString{String: errorMessage, Valid: true},
		sql.NullString{String: errorCode, Valid: errorCode != ""},
	)
	if err != nil {
		return fmt.Errorf("failed to mark notification as failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotificationLogNotFound
	}

	return nil
}

// ScheduleRetry schedules the next retry attempt
func (r *NotificationLogRepository) ScheduleRetry(id uuid.UUID, delay time.Duration) error {
	if id == uuid.Nil {
		return errors.New("invalid notification log ID")
	}

	query := `
		UPDATE notification_logs
		SET
			status = $2,
			retry_count = retry_count + 1,
			next_retry_at = $3,
			updated_at = $4
		WHERE id = $1
	`

	now := time.Now()
	nextRetry := now.Add(delay)
	result, err := r.db.Exec(
		query,
		id,
		model.NotificationStatusRetrying,
		sql.NullTime{Time: nextRetry, Valid: true},
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to schedule notification retry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotificationLogNotFound
	}

	return nil
}

// ListPendingRetries retrieves notifications pending retry
func (r *NotificationLogRepository) ListPendingRetries(limit int) ([]*model.NotificationLog, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT
			id, payment_id, merchant_id,
			channel, event_type, recipient,
			status, sent_at, delivered_at, failed_at,
			error_message, error_code,
			retry_count, max_retries, next_retry_at,
			subject, message, template_id,
			provider_message_id, provider_response, metadata,
			created_at, updated_at
		FROM notification_logs
		WHERE status = $1
		  AND next_retry_at IS NOT NULL
		  AND next_retry_at <= $2
		  AND retry_count < max_retries
		ORDER BY next_retry_at ASC
		LIMIT $3
	`

	rows, err := r.db.Query(query, model.NotificationStatusRetrying, time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending retries: %w", err)
	}
	defer rows.Close()

	logs := make([]*model.NotificationLog, 0)
	for rows.Next() {
		log := &model.NotificationLog{}
		err := rows.Scan(
			&log.ID,
			&log.PaymentID,
			&log.MerchantID,
			&log.Channel,
			&log.EventType,
			&log.Recipient,
			&log.Status,
			&log.SentAt,
			&log.DeliveredAt,
			&log.FailedAt,
			&log.ErrorMessage,
			&log.ErrorCode,
			&log.RetryCount,
			&log.MaxRetries,
			&log.NextRetryAt,
			&log.Subject,
			&log.Message,
			&log.TemplateID,
			&log.ProviderMessageID,
			&log.ProviderResponse,
			&log.Metadata,
			&log.CreatedAt,
			&log.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification log: %w", err)
		}
		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notification logs: %w", err)
	}

	return logs, nil
}

// GetDeliveryStats retrieves delivery statistics for a merchant and channel
func (r *NotificationLogRepository) GetDeliveryStats(merchantID uuid.UUID, channel model.NotificationChannel, startTime, endTime time.Time) (map[string]int64, error) {
	if merchantID == uuid.Nil {
		return nil, errors.New("merchant ID cannot be empty")
	}

	query := `
		SELECT
			status,
			COUNT(*) as count
		FROM notification_logs
		WHERE merchant_id = $1
		  AND channel = $2
		  AND created_at >= $3
		  AND created_at <= $4
		GROUP BY status
	`

	rows, err := r.db.Query(query, merchantID, channel, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan delivery stats: %w", err)
		}
		stats[status] = count
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating delivery stats: %w", err)
	}

	return stats, nil
}
