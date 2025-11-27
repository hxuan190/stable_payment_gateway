package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

var (
	ErrReconciliationNotFound = errors.New("reconciliation log not found")
)

type ReconciliationRepository struct {
	db *gorm.DB
}

func NewReconciliationRepository(db *gorm.DB) *ReconciliationRepository {
	return &ReconciliationRepository{db: db}
}

func (r *ReconciliationRepository) Create(log *model.ReconciliationLog) error {
	if log == nil {
		return errors.New("reconciliation log cannot be nil")
	}
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	return r.db.Create(log).Error
}

func (r *ReconciliationRepository) GetByID(id string) (*model.ReconciliationLog, error) {
	var log model.ReconciliationLog
	err := r.db.Where("id = ?", id).First(&log).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReconciliationNotFound
		}
		return nil, err
	}
	return &log, nil
}

func (r *ReconciliationRepository) GetLatest() (*model.ReconciliationLog, error) {
	var log model.ReconciliationLog
	err := r.db.Order("reconciled_at DESC").First(&log).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReconciliationNotFound
		}
		return nil, err
	}
	return &log, nil
}

func (r *ReconciliationRepository) GetByDateRange(startDate, endDate time.Time, limit int) ([]*model.ReconciliationLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var logs []*model.ReconciliationLog
	err := r.db.
		Where("reconciled_at >= ? AND reconciled_at <= ?", startDate, endDate).
		Order("reconciled_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

func (r *ReconciliationRepository) GetByStatus(status string, limit int) ([]*model.ReconciliationLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var logs []*model.ReconciliationLog
	err := r.db.
		Where("status = ?", status).
		Order("reconciled_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

func (r *ReconciliationRepository) GetDeficitAlerts(limit int) ([]*model.ReconciliationLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var logs []*model.ReconciliationLog
	err := r.db.
		Where("alert_triggered = ?", true).
		Order("reconciled_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}
