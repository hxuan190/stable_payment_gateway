package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	payoutDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payout/domain"
)

var (
	ErrPayoutScheduleNotFound      = errors.New("payout schedule not found")
	ErrPayoutScheduleAlreadyExists = errors.New("payout schedule already exists for this merchant")
)

type PayoutScheduleRepository struct {
	db *gorm.DB
}

func NewPayoutScheduleRepository(db *gorm.DB) *PayoutScheduleRepository {
	return &PayoutScheduleRepository{db: db}
}

func (r *PayoutScheduleRepository) Create(schedule *payoutDomain.PayoutSchedule) error {
	if schedule == nil {
		return errors.New("payout schedule cannot be nil")
	}
	if schedule.ID == uuid.Nil {
		schedule.ID = uuid.New()
	}
	if schedule.CreatedAt.IsZero() {
		schedule.CreatedAt = time.Now().UTC()
	}
	if schedule.UpdatedAt.IsZero() {
		schedule.UpdatedAt = time.Now().UTC()
	}
	return r.db.Create(schedule).Error
}

func (r *PayoutScheduleRepository) GetByID(id uuid.UUID) (*payoutDomain.PayoutSchedule, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid payout schedule ID")
	}
	var schedule payoutDomain.PayoutSchedule
	err := r.db.Where("id = ?", id).First(&schedule).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPayoutScheduleNotFound
		}
		return nil, err
	}
	return &schedule, nil
}

func (r *PayoutScheduleRepository) GetByMerchantID(merchantID uuid.UUID) (*payoutDomain.PayoutSchedule, error) {
	if merchantID == uuid.Nil {
		return nil, errors.New("merchant ID cannot be empty")
	}
	var schedule payoutDomain.PayoutSchedule
	err := r.db.Where("merchant_id = ?", merchantID).First(&schedule).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPayoutScheduleNotFound
		}
		return nil, err
	}
	return &schedule, nil
}

func (r *PayoutScheduleRepository) Update(schedule *payoutDomain.PayoutSchedule) error {
	if schedule == nil {
		return errors.New("payout schedule cannot be nil")
	}
	if schedule.ID == uuid.Nil {
		return errors.New("invalid payout schedule ID")
	}
	schedule.UpdatedAt = time.Now().UTC()
	return r.db.Save(schedule).Error
}

func (r *PayoutScheduleRepository) Suspend(id uuid.UUID, reason string) error {
	if id == uuid.Nil {
		return errors.New("invalid payout schedule ID")
	}
	if reason == "" {
		return errors.New("suspend reason cannot be empty")
	}
	now := time.Now().UTC()
	result := r.db.Model(&payoutDomain.PayoutSchedule{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_suspended":     true,
		"suspended_at":     sql.NullTime{Time: now, Valid: true},
		"suspended_reason": sql.NullString{String: reason, Valid: true},
		"updated_at":       now,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPayoutScheduleNotFound
	}
	return nil
}

func (r *PayoutScheduleRepository) Resume(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid payout schedule ID")
	}
	result := r.db.Model(&payoutDomain.PayoutSchedule{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_suspended":     false,
		"suspended_at":     nil,
		"suspended_reason": nil,
		"updated_at":       time.Now().UTC(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPayoutScheduleNotFound
	}
	return nil
}

func (r *PayoutScheduleRepository) UpdateLastScheduledPayout(id uuid.UUID, payoutTime time.Time) error {
	if id == uuid.Nil {
		return errors.New("invalid payout schedule ID")
	}
	result := r.db.Model(&payoutDomain.PayoutSchedule{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_scheduled_payout_at": sql.NullTime{Time: payoutTime, Valid: true},
		"updated_at":               time.Now().UTC(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPayoutScheduleNotFound
	}
	return nil
}

func (r *PayoutScheduleRepository) UpdateLastThresholdPayout(id uuid.UUID, payoutTime time.Time) error {
	if id == uuid.Nil {
		return errors.New("invalid payout schedule ID")
	}
	result := r.db.Model(&payoutDomain.PayoutSchedule{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_threshold_payout_at": sql.NullTime{Time: payoutTime, Valid: true},
		"updated_at":               time.Now().UTC(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPayoutScheduleNotFound
	}
	return nil
}

func (r *PayoutScheduleRepository) ListScheduledPayoutsDue() ([]*payoutDomain.PayoutSchedule, error) {
	var schedules []*payoutDomain.PayoutSchedule
	err := r.db.
		Where("scheduled_enabled = ? AND is_suspended = ? AND next_scheduled_payout_at IS NOT NULL AND next_scheduled_payout_at <= ?", true, false, time.Now()).
		Order("next_scheduled_payout_at ASC").
		Find(&schedules).Error
	return schedules, err
}

func (r *PayoutScheduleRepository) ListThresholdEnabled() ([]*payoutDomain.PayoutSchedule, error) {
	var schedules []*payoutDomain.PayoutSchedule
	err := r.db.
		Where("threshold_enabled = ? AND is_suspended = ?", true, false).
		Order("threshold_usdt ASC").
		Find(&schedules).Error
	return schedules, err
}

func (r *PayoutScheduleRepository) Delete(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid payout schedule ID")
	}
	result := r.db.Where("id = ?", id).Delete(&payoutDomain.PayoutSchedule{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPayoutScheduleNotFound
	}
	return nil
}

func (r *PayoutScheduleRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&payoutDomain.PayoutSchedule{}).Count(&count).Error
	return count, err
}
