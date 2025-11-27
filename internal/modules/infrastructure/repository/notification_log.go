package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"gorm.io/gorm"
)

var (
	// ErrNotificationLogNotFound is returned when a notification log is not found
	ErrNotificationLogNotFound = errors.New("notification log not found")
)

// NotificationLogRepository handles database operations for notification logs
type NotificationLogRepository struct {
	db *gorm.DB
}

// NewNotificationLogRepository creates a new notification log repository
func NewNotificationLogRepository(db *gorm.DB) *NotificationLogRepository {
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

	if err := r.db.Create(log).Error; err != nil {
		return fmt.Errorf("failed to create notification log: %w", err)
	}

	return nil
}

// GetByID retrieves a notification log by ID
func (r *NotificationLogRepository) GetByID(id uuid.UUID) (*model.NotificationLog, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid notification log ID")
	}

	log := &model.NotificationLog{}
	if err := r.db.Where("id = ?", id).First(log).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotificationLogNotFound
		}
		return nil, err
	}

	return log, nil

}

// GetByPaymentID retrieves all notification logs for a payment
func (r *NotificationLogRepository) GetByPaymentID(paymentID uuid.UUID) ([]*model.NotificationLog, error) {
	if paymentID == uuid.Nil {
		return nil, errors.New("payment ID cannot be empty")
	}

	var logs []*model.NotificationLog
	if err := r.db.Where("payment_id = ?", paymentID).Order("created_at ASC").Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to get notification logs by payment ID: %w", err)
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

	var logs []*model.NotificationLog
	if err := r.db.Where("merchant_id = ?", merchantID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to get notification logs by merchant ID: %w", err)
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

	log.UpdatedAt = time.Now()

	result := r.db.Model(&model.NotificationLog{}).Where("id = ?", log.ID).Updates(map[string]interface{}{
		"status":              log.Status,
		"sent_at":             log.SentAt,
		"delivered_at":        log.DeliveredAt,
		"failed_at":           log.FailedAt,
		"error_message":       log.ErrorMessage,
		"error_code":          log.ErrorCode,
		"retry_count":         log.RetryCount,
		"next_retry_at":       log.NextRetryAt,
		"provider_message_id": log.ProviderMessageID,
		"provider_response":   log.ProviderResponse,
		"metadata":            log.Metadata,
		"updated_at":          log.UpdatedAt,
	})

	if result.Error != nil {
		return fmt.Errorf("failed to update notification log: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotificationLogNotFound
	}

	return nil
}

// MarkAsSent updates the status to sent
func (r *NotificationLogRepository) MarkAsSent(id uuid.UUID, providerMessageID string) error {
	if id == uuid.Nil {
		return errors.New("invalid notification log ID")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":     model.NotificationStatusSent,
		"sent_at":    sql.NullTime{Time: now, Valid: true},
		"updated_at": now,
	}

	if providerMessageID != "" {
		updates["provider_message_id"] = sql.NullString{String: providerMessageID, Valid: true}
	}

	result := r.db.Model(&model.NotificationLog{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to mark notification as sent: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotificationLogNotFound
	}

	return nil
}

// MarkAsDelivered updates the status to delivered
func (r *NotificationLogRepository) MarkAsDelivered(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid notification log ID")
	}

	now := time.Now()
	result := r.db.Model(&model.NotificationLog{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       model.NotificationStatusDelivered,
		"delivered_at": sql.NullTime{Time: now, Valid: true},
		"updated_at":   now,
	})

	if result.Error != nil {
		return fmt.Errorf("failed to mark notification as delivered: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotificationLogNotFound
	}

	return nil
}

// MarkAsFailed updates the status to failed
func (r *NotificationLogRepository) MarkAsFailed(id uuid.UUID, errorMessage, errorCode string) error {
	if id == uuid.Nil {
		return errors.New("invalid notification log ID")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":        model.NotificationStatusFailed,
		"failed_at":     sql.NullTime{Time: now, Valid: true},
		"error_message": sql.NullString{String: errorMessage, Valid: true},
		"updated_at":    now,
	}

	if errorCode != "" {
		updates["error_code"] = sql.NullString{String: errorCode, Valid: true}
	}

	result := r.db.Model(&model.NotificationLog{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to mark notification as failed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotificationLogNotFound
	}

	return nil
}

// ScheduleRetry schedules the next retry attempt
func (r *NotificationLogRepository) ScheduleRetry(id uuid.UUID, delay time.Duration) error {
	if id == uuid.Nil {
		return errors.New("invalid notification log ID")
	}

	now := time.Now()
	nextRetry := now.Add(delay)

	result := r.db.Model(&model.NotificationLog{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        model.NotificationStatusRetrying,
		"retry_count":   gorm.Expr("retry_count + 1"),
		"next_retry_at": sql.NullTime{Time: nextRetry, Valid: true},
		"updated_at":    now,
	})

	if result.Error != nil {
		return fmt.Errorf("failed to schedule notification retry: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotificationLogNotFound
	}

	return nil
}

// ListPendingRetries retrieves notifications pending retry
func (r *NotificationLogRepository) ListPendingRetries(limit int) ([]*model.NotificationLog, error) {
	if limit <= 0 {
		limit = 100
	}

	var logs []*model.NotificationLog
	if err := r.db.Where("status = ? AND next_retry_at IS NOT NULL AND next_retry_at <= ? AND retry_count < max_retries",
		model.NotificationStatusRetrying, time.Now()).
		Order("next_retry_at ASC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to list pending retries: %w", err)
	}

	return logs, nil
}

// GetDeliveryStats retrieves delivery statistics for a merchant and channel
func (r *NotificationLogRepository) GetDeliveryStats(merchantID uuid.UUID, channel model.NotificationChannel, startTime, endTime time.Time) (map[string]int64, error) {
	if merchantID == uuid.Nil {
		return nil, errors.New("merchant ID cannot be empty")
	}

	type result struct {
		Status string
		Count  int64
	}

	var results []result
	if err := r.db.Model(&model.NotificationLog{}).
		Select("status, COUNT(*) as count").
		Where("merchant_id = ? AND channel = ? AND created_at >= ? AND created_at <= ?",
			merchantID, channel, startTime, endTime).
		Group("status").
		Find(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get delivery stats: %w", err)
	}

	stats := make(map[string]int64)
	for _, r := range results {
		stats[r.Status] = r.Count
	}

	return stats, nil
}
