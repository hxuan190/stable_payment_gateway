package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	payoutDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payout/domain"
	"gorm.io/gorm"
)

var (
	// ErrPayoutNotFound is returned when a payout is not found
	ErrPayoutNotFound = errors.New("payout not found")
	// ErrInvalidPayoutID is returned when the payout ID is invalid
	ErrInvalidPayoutID = errors.New("invalid payout ID")
	// ErrInvalidPayoutStatus is returned when the payout status is invalid
	ErrInvalidPayoutStatus = errors.New("invalid payout status")
)

// PayoutRepository handles database operations for payouts
type PayoutRepository struct {
	gormDB *gorm.DB
}

// NewPayoutRepository creates a new payout repository
func NewPayoutRepository(gormDB *gorm.DB) *PayoutRepository {
	return &PayoutRepository{
		gormDB: gormDB,
	}
}

// Create inserts a new payout into the database
func (r *PayoutRepository) Create(payout *payoutDomain.Payout) error {
	if payout == nil {
		return errors.New("payout cannot be nil")
	}

	if err := r.gormDB.Create(payout).Error; err != nil {
		return fmt.Errorf("failed to create payout: %w", err)
	}

	return nil
}

// GetByID retrieves a payout by ID
func (r *PayoutRepository) GetByID(id string) (*payoutDomain.Payout, error) {
	if id == "" {
		return nil, ErrInvalidPayoutID
	}

	payout := &payoutDomain.Payout{}
	if err := r.gormDB.Where("id = ?", id).First(payout).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPayoutNotFound
		}
		return nil, err
	}

	return payout, nil
}

// ListByMerchant retrieves payouts for a specific merchant with pagination
func (r *PayoutRepository) ListByMerchant(merchantID string, limit, offset int) ([]*payoutDomain.Payout, error) {
	if merchantID == "" {
		return nil, errors.New("merchant ID cannot be empty")
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	payouts := make([]*payoutDomain.Payout, 0)
	if err := r.gormDB.Where("merchant_id = ?", merchantID).Offset(offset).Limit(limit).Find(&payouts).Error; err != nil {
		return nil, fmt.Errorf("failed to list payouts by merchant: %w", err)
	}

	return payouts, nil
}

// ListByStatus retrieves payouts by status with pagination
func (r *PayoutRepository) ListByStatus(status payoutDomain.PayoutStatus, limit, offset int) ([]*payoutDomain.Payout, error) {
	if status == "" {
		return nil, ErrInvalidPayoutStatus
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	payouts := make([]*payoutDomain.Payout, 0)
	if err := r.gormDB.Where("status = ?", status).Offset(offset).Limit(limit).Find(&payouts).Error; err != nil {
		return nil, fmt.Errorf("failed to list payouts by status: %w", err)
	}

	return payouts, nil
}

// UpdateStatus updates only the status of a payout
func (r *PayoutRepository) UpdateStatus(id string, status payoutDomain.PayoutStatus) error {
	if id == "" {
		return ErrInvalidPayoutID
	}
	if status == "" {
		return ErrInvalidPayoutStatus
	}

	// Validate status
	validStatuses := map[payoutDomain.PayoutStatus]bool{
		payoutDomain.PayoutStatusRequested:  true,
		payoutDomain.PayoutStatusApproved:   true,
		payoutDomain.PayoutStatusProcessing: true,
		payoutDomain.PayoutStatusCompleted:  true,
		payoutDomain.PayoutStatusRejected:   true,
		payoutDomain.PayoutStatusFailed:     true,
	}
	if !validStatuses[status] {
		return ErrInvalidPayoutStatus
	}

	if err := r.gormDB.Model(&payoutDomain.Payout{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update payout status: %w", err)
	}

	return nil
}

// Update updates an existing payout in the database
func (r *PayoutRepository) Update(payout *payoutDomain.Payout) error {
	if payout == nil {
		return errors.New("payout cannot be nil")
	}
	if payout.ID == "" {
		return ErrInvalidPayoutID
	}

	if err := r.gormDB.Where("id = ?", payout.ID).Updates(payout).Error; err != nil {
		return fmt.Errorf("failed to update payout: %w", err)
	}

	if err := r.gormDB.Save(payout).Error; err != nil {
		return fmt.Errorf("failed to update payout: %w", err)
	}

	return nil
}

// MarkCompleted marks a payout as completed with bank reference number
func (r *PayoutRepository) MarkCompleted(id string, referenceNumber string) error {
	if id == "" {
		return ErrInvalidPayoutID
	}

	if referenceNumber == "" {
		return errors.New("bank reference number cannot be empty")
	}

	payout, err := r.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get payout: %w", err)
	}

	payout.Status = payoutDomain.PayoutStatusCompleted
	payout.BankReferenceNumber = sql.NullString{String: referenceNumber, Valid: true}
	payout.CompletionDate = sql.NullTime{Time: time.Now(), Valid: true}
	payout.UpdatedAt = time.Now()

	return r.Update(payout)
}

func (r *PayoutRepository) Approve(id, approvedBy string) error {
	if id == "" {
		return ErrInvalidPayoutID
	}
	if approvedBy == "" {
		return errors.New("approver ID cannot be empty")
	}

	now := time.Now()
	payout, err := r.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get payout: %w", err)
	}

	payout.Status = payoutDomain.PayoutStatusApproved
	payout.ApprovedBy = sql.NullString{String: approvedBy, Valid: true}
	payout.ApprovedAt = sql.NullTime{Time: now, Valid: true}
	payout.UpdatedAt = now

	return r.Update(payout)
}

// Reject rejects a payout request
func (r *PayoutRepository) Reject(id string, reason string) error {
	if id == "" {
		return ErrInvalidPayoutID
	}
	if reason == "" {
		return errors.New("rejection reason cannot be empty")
	}

	now := time.Now()
	payout, err := r.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get payout: %w", err)
	}

	payout.Status = payoutDomain.PayoutStatusRejected
	payout.RejectionReason = sql.NullString{String: reason, Valid: true}
	payout.UpdatedAt = now

	return r.Update(payout)
}

// Count returns the total number of payouts (excluding deleted)
func (r *PayoutRepository) Count() (int64, error) {
	var count int64
	if err := r.gormDB.Model(&payoutDomain.Payout{}).Where("deleted_at IS NULL").Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count payouts: %w", err)
	}

	return count, nil
}

// CountByMerchant returns the count of payouts for a specific merchant
func (r *PayoutRepository) CountByMerchant(merchantID string) (int64, error) {
	if merchantID == "" {
		return 0, errors.New("merchant ID cannot be empty")
	}

	var count int64
	if err := r.gormDB.Model(&payoutDomain.Payout{}).Where("merchant_id = ? AND deleted_at IS NULL", merchantID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count payouts by merchant: %w", err)
	}

	return count, nil
}

// CountByStatus returns the count of payouts by status
func (r *PayoutRepository) CountByStatus(status payoutDomain.PayoutStatus) (int64, error) {
	if status == "" {
		return 0, ErrInvalidPayoutStatus
	}

	var count int64
	if err := r.gormDB.Model(&payoutDomain.Payout{}).Where("status = ? AND deleted_at IS NULL", status).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count payouts by status: %w", err)
	}

	return count, nil
}

// Delete soft-deletes a payout by setting the deleted_at timestamp
func (r *PayoutRepository) Delete(id string) error {
	if id == "" {
		return ErrInvalidPayoutID
	}

	payout, err := r.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get payout: %w", err)
	}

	payout.DeletedAt = sql.NullTime{Time: time.Now(), Valid: true}
	payout.UpdatedAt = time.Now()

	return r.Update(payout)
}
