package repository

import (
	"errors"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/domain"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ErrMerchantNotFound      = errors.New("merchant not found")
	ErrMerchantAlreadyExists = errors.New("merchant with this email already exists")
	ErrInvalidMerchantID     = errors.New("invalid merchant ID")
)

type MerchantRepository struct {
	db *gorm.DB
}

func NewMerchantRepository(db *gorm.DB) *MerchantRepository {
	return &MerchantRepository{
		db: db,
	}
}

func (r *MerchantRepository) Create(merchant *domain.Merchant) error {
	if merchant == nil {
		return errors.New("merchant cannot be nil")
	}

	now := time.Now()
	if merchant.CreatedAt.IsZero() {
		merchant.CreatedAt = now
	}
	if merchant.UpdatedAt.IsZero() {
		merchant.UpdatedAt = now
	}

	if err := r.db.Create(merchant).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrMerchantAlreadyExists
		}
		return err
	}

	return nil
}

func (r *MerchantRepository) GetByID(id string) (*domain.Merchant, error) {
	if id == "" {
		return nil, ErrInvalidMerchantID
	}

	merchant := &domain.Merchant{}
	if err := r.db.Where("id = ?", id).First(merchant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMerchantNotFound
		}
		return nil, err
	}

	return merchant, nil
}

func (r *MerchantRepository) GetByEmail(email string) (*domain.Merchant, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	merchant := &domain.Merchant{}
	if err := r.db.Where("email = ?", email).First(merchant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMerchantNotFound
		}
		return nil, err
	}

	return merchant, nil
}

func (r *MerchantRepository) GetByAPIKey(apiKey string) (*domain.Merchant, error) {
	if apiKey == "" {
		return nil, errors.New("API key cannot be empty")
	}

	merchant := &domain.Merchant{}
	if err := r.db.Where("api_key = ?", apiKey).First(merchant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMerchantNotFound
		}
		return nil, err
	}

	return merchant, nil
}

func (r *MerchantRepository) Update(merchant *domain.Merchant) error {
	if merchant == nil {
		return errors.New("merchant cannot be nil")
	}
	if merchant.ID == "" {
		return ErrInvalidMerchantID
	}

	merchant.UpdatedAt = time.Now()

	result := r.db.Model(&domain.Merchant{}).Where("id = ?", merchant.ID).Updates(merchant)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrMerchantNotFound
	}

	return nil
}

func (r *MerchantRepository) UpdateKYCStatus(id string, status string) error {
	if id == "" {
		return ErrInvalidMerchantID
	}
	if status == "" {
		return errors.New("KYC status cannot be empty")
	}

	result := r.db.Model(&domain.Merchant{}).Where("id = ?", id).Updates(map[string]interface{}{
		"kyc_status": status,
		"updated_at": time.Now(),
	})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrMerchantNotFound
	}

	return nil
}

func (r *MerchantRepository) List(limit, offset int) ([]*domain.Merchant, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var merchants []*domain.Merchant
	if err := r.db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&merchants).Error; err != nil {
		return nil, err
	}

	return merchants, nil
}

func (r *MerchantRepository) ListByKYCStatus(status string, limit, offset int) ([]*domain.Merchant, error) {
	if status == "" {
		return nil, errors.New("KYC status cannot be empty")
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var merchants []*domain.Merchant
	if err := r.db.Where("kyc_status = ?", status).Order("kyc_submitted_at ASC").Limit(limit).Offset(offset).Find(&merchants).Error; err != nil {
		return nil, err
	}

	return merchants, nil
}

func (r *MerchantRepository) Delete(id string) error {
	if id == "" {
		return ErrInvalidMerchantID
	}

	result := r.db.Where("id = ?", id).Delete(&domain.Merchant{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrMerchantNotFound
	}

	return nil
}

func (r *MerchantRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&domain.Merchant{}).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *MerchantRepository) CountByKYCStatus(status string) (int64, error) {
	if status == "" {
		return 0, errors.New("KYC status cannot be empty")
	}

	var count int64
	if err := r.db.Model(&domain.Merchant{}).Where("kyc_status = ?", status).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

// GetMerchantKYCStatus retrieves the KYC status of a merchant (for payment module)
func (r *MerchantRepository) GetMerchantKYCStatus(merchantID string) (string, error) {
	if merchantID == "" {
		return "", ErrInvalidMerchantID
	}

	var merchant domain.Merchant
	if err := r.db.Select("kyc_status").Where("id = ?", merchantID).First(&merchant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrMerchantNotFound
		}
		return "", err
	}

	return string(merchant.KYCStatus), nil
}

// UpdateMerchantVolume updates the merchant's monthly volume (for payment module)
func (r *MerchantRepository) UpdateMerchantVolume(merchantID string, amountUSD decimal.Decimal) error {
	if merchantID == "" {
		return ErrInvalidMerchantID
	}

	return r.db.Model(&domain.Merchant{}).
		Where("id = ?", merchantID).
		UpdateColumn("monthly_volume_usd", gorm.Expr("monthly_volume_usd + ?", amountUSD)).
		Error
}
