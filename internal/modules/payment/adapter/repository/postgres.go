package repository

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
)

type PostgresPaymentRepository struct {
	db *gorm.DB
}

func NewPostgresPaymentRepository(db *gorm.DB) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{
		db: db,
	}
}

func (r *PostgresPaymentRepository) Create(payment *domain.Payment) error {
	if payment == nil {
		return errors.New("payment cannot be nil")
	}

	now := time.Now()
	if payment.CreatedAt.IsZero() {
		payment.CreatedAt = now
	}
	if payment.UpdatedAt.IsZero() {
		payment.UpdatedAt = now
	}

	if err := r.db.Create(payment).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrPaymentAlreadyExists
		}
		return err
	}

	return nil
}

func (r *PostgresPaymentRepository) GetByID(id string) (*domain.Payment, error) {
	if id == "" {
		return nil, domain.ErrInvalidPaymentID
	}

	payment := &domain.Payment{}
	if err := r.db.Where("id = ?", id).First(payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}

	return payment, nil
}

func (r *PostgresPaymentRepository) GetByTxHash(txHash string) (*domain.Payment, error) {
	if txHash == "" {
		return nil, errors.New("transaction hash cannot be empty")
	}

	payment := &domain.Payment{}
	if err := r.db.Where("tx_hash = ?", txHash).First(payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}

	return payment, nil
}

func (r *PostgresPaymentRepository) GetByPaymentReference(reference string) (*domain.Payment, error) {
	if reference == "" {
		return nil, errors.New("payment reference cannot be empty")
	}

	payment := &domain.Payment{}
	if err := r.db.Where("payment_reference = ?", reference).First(payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}

	return payment, nil
}

func (r *PostgresPaymentRepository) UpdateStatus(id string, status domain.PaymentStatus) error {
	if id == "" {
		return domain.ErrInvalidPaymentID
	}
	if status == "" {
		return domain.ErrInvalidPaymentStatus
	}

	validStatuses := map[domain.PaymentStatus]bool{
		domain.PaymentStatusCreated:           true,
		domain.PaymentStatusPending:           true,
		domain.PaymentStatusPendingCompliance: true,
		domain.PaymentStatusConfirming:        true,
		domain.PaymentStatusCompleted:         true,
		domain.PaymentStatusExpired:           true,
		domain.PaymentStatusFailed:            true,
	}
	if !validStatuses[status] {
		return domain.ErrInvalidPaymentStatus
	}

	result := r.db.Model(&domain.Payment{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrPaymentNotFound
	}

	return nil
}

func (r *PostgresPaymentRepository) Update(payment *domain.Payment) error {
	if payment == nil {
		return errors.New("payment cannot be nil")
	}
	if payment.ID == "" {
		return domain.ErrInvalidPaymentID
	}

	payment.UpdatedAt = time.Now()

	result := r.db.Model(&domain.Payment{}).Where("id = ?", payment.ID).Updates(payment)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrPaymentNotFound
	}

	return nil
}

func (r *PostgresPaymentRepository) ListByMerchant(merchantID string, limit, offset int) ([]*domain.Payment, error) {
	if merchantID == "" {
		return nil, errors.New("merchant ID cannot be empty")
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var payments []*domain.Payment
	if err := r.db.Where("merchant_id = ?", merchantID).Order("created_at DESC").Limit(limit).Offset(offset).Find(&payments).Error; err != nil {
		return nil, err
	}

	return payments, nil
}

func (r *PostgresPaymentRepository) GetExpiredPayments() ([]*domain.Payment, error) {
	var payments []*domain.Payment
	if err := r.db.Where("status IN ?", []string{"created", "pending"}).Where("expires_at < ?", time.Now()).Order("expires_at ASC").Find(&payments).Error; err != nil {
		return nil, err
	}

	return payments, nil
}

func (r *PostgresPaymentRepository) GetComplianceExpiredPayments() ([]*domain.Payment, error) {
	var payments []*domain.Payment
	if err := r.db.Where("status = ?", "pending_compliance").
		Where("compliance_deadline IS NOT NULL").
		Where("compliance_deadline < ?", time.Now()).
		Where("compliance_submitted_at IS NULL").
		Order("compliance_deadline ASC").
		Limit(1000).
		Find(&payments).Error; err != nil {
		return nil, err
	}

	return payments, nil
}

func (r *PostgresPaymentRepository) ListByStatus(status domain.PaymentStatus, limit, offset int) ([]*domain.Payment, error) {
	if status == "" {
		return nil, domain.ErrInvalidPaymentStatus
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var payments []*domain.Payment
	if err := r.db.Where("status = ?", status).Order("created_at DESC").Limit(limit).Offset(offset).Find(&payments).Error; err != nil {
		return nil, err
	}

	return payments, nil
}

func (r *PostgresPaymentRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&domain.Payment{}).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *PostgresPaymentRepository) CountByMerchant(merchantID string) (int64, error) {
	if merchantID == "" {
		return 0, errors.New("merchant ID cannot be empty")
	}

	var count int64
	if err := r.db.Model(&domain.Payment{}).Where("merchant_id = ?", merchantID).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *PostgresPaymentRepository) CountByStatus(status domain.PaymentStatus) (int64, error) {
	if status == "" {
		return 0, domain.ErrInvalidPaymentStatus
	}

	var count int64
	if err := r.db.Model(&domain.Payment{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *PostgresPaymentRepository) GetTotalVolumeByMerchant(merchantID string) (decimal.Decimal, error) {
	if merchantID == "" {
		return decimal.Zero, errors.New("merchant ID cannot be empty")
	}

	var total decimal.Decimal
	if err := r.db.Model(&domain.Payment{}).Where("merchant_id = ? AND status = ?", merchantID, "completed").Select("COALESCE(SUM(amount_vnd), 0)").Scan(&total).Error; err != nil {
		return decimal.Zero, err
	}

	return total, nil
}

func (r *PostgresPaymentRepository) CountByAddressAndWindow(fromAddress string, window time.Duration) (int64, error) {
	if fromAddress == "" {
		return 0, errors.New("from address cannot be empty")
	}

	windowStart := time.Now().Add(-window)
	var count int64
	if err := r.db.Model(&domain.Payment{}).Where("from_address = ? AND created_at > ? AND status != ?", fromAddress, windowStart, "failed").Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *PostgresPaymentRepository) Delete(id string) error {
	if id == "" {
		return domain.ErrInvalidPaymentID
	}

	result := r.db.Where("id = ?", id).Delete(&domain.Payment{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrPaymentNotFound
	}

	return nil
}
