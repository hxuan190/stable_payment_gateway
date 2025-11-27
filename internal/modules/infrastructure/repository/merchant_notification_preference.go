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
	ErrPreferenceNotFound      = errors.New("merchant notification preference not found")
	ErrPreferenceAlreadyExists = errors.New("merchant notification preference already exists")
)

type MerchantNotificationPreferenceRepository struct {
	db *gorm.DB
}

func NewMerchantNotificationPreferenceRepository(db *gorm.DB) *MerchantNotificationPreferenceRepository {
	return &MerchantNotificationPreferenceRepository{
		db: db,
	}
}

func (r *MerchantNotificationPreferenceRepository) Create(pref *model.MerchantNotificationPreference) error {
	if pref == nil {
		return errors.New("merchant notification preference cannot be nil")
	}

	if pref.ID == uuid.Nil {
		pref.ID = uuid.New()
	}

	if err := r.db.Create(pref).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrPreferenceAlreadyExists
		}
		return fmt.Errorf("failed to create merchant notification preference: %w", err)
	}

	return nil
}

func (r *MerchantNotificationPreferenceRepository) GetByID(id uuid.UUID) (*model.MerchantNotificationPreference, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid preference ID")
	}

	pref := &model.MerchantNotificationPreference{}
	if err := r.db.Where("id = ?", id).First(pref).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPreferenceNotFound
		}
		return nil, fmt.Errorf("failed to get merchant notification preference by ID: %w", err)
	}

	return pref, nil
}

func (r *MerchantNotificationPreferenceRepository) GetByMerchantAndChannel(merchantID uuid.UUID, channel model.NotificationChannel) (*model.MerchantNotificationPreference, error) {
	if merchantID == uuid.Nil {
		return nil, errors.New("merchant ID cannot be empty")
	}

	pref := &model.MerchantNotificationPreference{}
	if err := r.db.Where("merchant_id = ? AND channel = ?", merchantID, channel).First(pref).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPreferenceNotFound
		}
		return nil, fmt.Errorf("failed to get merchant notification preference: %w", err)
	}

	return pref, nil
}

func (r *MerchantNotificationPreferenceRepository) GetByMerchantID(merchantID uuid.UUID) ([]*model.MerchantNotificationPreference, error) {
	if merchantID == uuid.Nil {
		return nil, errors.New("merchant ID cannot be empty")
	}

	var prefs []*model.MerchantNotificationPreference
	if err := r.db.Where("merchant_id = ?", merchantID).
		Order("channel").
		Find(&prefs).Error; err != nil {
		return nil, fmt.Errorf("failed to get merchant notification preferences: %w", err)
	}

	return prefs, nil
}

func (r *MerchantNotificationPreferenceRepository) GetEnabledByMerchantID(merchantID uuid.UUID) ([]*model.MerchantNotificationPreference, error) {
	if merchantID == uuid.Nil {
		return nil, errors.New("merchant ID cannot be empty")
	}

	var prefs []*model.MerchantNotificationPreference
	if err := r.db.Where("merchant_id = ? AND enabled = ?", merchantID, true).
		Order("channel").
		Find(&prefs).Error; err != nil {
		return nil, fmt.Errorf("failed to get enabled merchant notification preferences: %w", err)
	}

	return prefs, nil
}

func (r *MerchantNotificationPreferenceRepository) Update(pref *model.MerchantNotificationPreference) error {
	if pref == nil {
		return errors.New("merchant notification preference cannot be nil")
	}
	if pref.ID == uuid.Nil {
		return errors.New("invalid preference ID")
	}

	pref.UpdatedAt = time.Now()

	result := r.db.Model(&model.MerchantNotificationPreference{}).Where("id = ?", pref.ID).Updates(map[string]interface{}{
		"enabled":                  pref.Enabled,
		"subscribed_events":        pref.SubscribedEvents,
		"config":                   pref.Config,
		"rate_limit_per_hour":      pref.RateLimitPerHour,
		"rate_limit_per_day":       pref.RateLimitPerDay,
		"is_test_mode":             pref.IsTestMode,
		"test_config":              pref.TestConfig,
		"last_notification_at":     pref.LastNotificationAt,
		"total_notifications_sent": pref.TotalNotificationsSent,
		"updated_at":               pref.UpdatedAt,
	})

	if result.Error != nil {
		return fmt.Errorf("failed to update merchant notification preference: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrPreferenceNotFound
	}

	return nil
}

func (r *MerchantNotificationPreferenceRepository) Enable(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid preference ID")
	}

	result := r.db.Model(&model.MerchantNotificationPreference{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"enabled":    true,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to enable notification channel: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrPreferenceNotFound
	}

	return nil
}

func (r *MerchantNotificationPreferenceRepository) Disable(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid preference ID")
	}

	result := r.db.Model(&model.MerchantNotificationPreference{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"enabled":    false,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to disable notification channel: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrPreferenceNotFound
	}

	return nil
}

func (r *MerchantNotificationPreferenceRepository) IncrementNotificationCount(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid preference ID")
	}

	now := time.Now()
	result := r.db.Model(&model.MerchantNotificationPreference{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"total_notifications_sent": gorm.Expr("total_notifications_sent + 1"),
			"last_notification_at":     sql.NullTime{Time: now, Valid: true},
			"updated_at":               now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to increment notification count: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrPreferenceNotFound
	}

	return nil
}

func (r *MerchantNotificationPreferenceRepository) Delete(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid preference ID")
	}

	result := r.db.Delete(&model.MerchantNotificationPreference{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete merchant notification preference: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrPreferenceNotFound
	}

	return nil
}

func (r *MerchantNotificationPreferenceRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&model.MerchantNotificationPreference{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count merchant notification preferences: %w", err)
	}

	return count, nil
}

func (r *MerchantNotificationPreferenceRepository) CountByChannel(channel model.NotificationChannel) (int64, error) {
	var count int64
	if err := r.db.Model(&model.MerchantNotificationPreference{}).
		Where("channel = ? AND enabled = ?", channel, true).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count preferences by channel: %w", err)
	}

	return count, nil
}
