package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	merchantDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/domain"
)

var (
	ErrKYCDocumentNotFound        = errors.New("KYC document not found")
	ErrInvalidKYCDocument         = errors.New("invalid KYC document")
	ErrKYCDocumentAlreadyReviewed = errors.New("KYC document has already been reviewed")
)

type KYCDocumentRepository interface {
	Create(ctx context.Context, document *merchantDomain.KYCDocument) error
	GetByID(ctx context.Context, id uuid.UUID) (*merchantDomain.KYCDocument, error)
	GetByMerchantID(ctx context.Context, merchantID string) ([]*merchantDomain.KYCDocument, error)
	List(ctx context.Context, filter KYCDocumentFilter) ([]*merchantDomain.KYCDocument, error)
	ListByStatus(ctx context.Context, status merchantDomain.KYCDocumentStatus) ([]*merchantDomain.KYCDocument, error)
	Update(ctx context.Context, document *merchantDomain.KYCDocument) error
	Delete(ctx context.Context, id uuid.UUID) error
	Approve(ctx context.Context, id string, reviewerID string, notes string) error
	Reject(ctx context.Context, id string, reviewerID string, notes string) error
	GetPendingDocuments(ctx context.Context, limit int) ([]*merchantDomain.KYCDocument, error)
	CountByMerchantAndStatus(ctx context.Context, merchantID string, status merchantDomain.KYCDocumentStatus) (int64, error)
	HasApprovedDocumentOfType(ctx context.Context, merchantID string, docType merchantDomain.KYCDocumentType) (bool, error)
}

type KYCDocumentFilter struct {
	MerchantID   string
	DocumentType string
	Status       merchantDomain.KYCDocumentStatus
	Limit        int
	Offset       int
}

type kycDocumentRepositoryImpl struct {
	db *gorm.DB
}

func NewKYCDocumentRepository(db *gorm.DB) KYCDocumentRepository {
	return &kycDocumentRepositoryImpl{db: db}
}

func (r *kycDocumentRepositoryImpl) HasApprovedDocumentOfType(ctx context.Context, merchantID string, docType merchantDomain.KYCDocumentType) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&merchantDomain.KYCDocument{}).
		Where("merchant_id = ? AND document_type = ? AND status = ?", merchantID, string(docType), merchantDomain.KYCDocumentStatusApproved).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *kycDocumentRepositoryImpl) Create(ctx context.Context, document *merchantDomain.KYCDocument) error {
	if document.ID == "" {
		document.ID = uuid.New().String()
	}
	if document.CreatedAt.IsZero() {
		document.CreatedAt = time.Now().UTC()
	}
	if document.UpdatedAt.IsZero() {
		document.UpdatedAt = time.Now().UTC()
	}
	return r.db.WithContext(ctx).Create(document).Error
}

func (r *kycDocumentRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*merchantDomain.KYCDocument, error) {
	var document merchantDomain.KYCDocument
	err := r.db.WithContext(ctx).Where("id = ?", id.String()).First(&document).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrKYCDocumentNotFound
		}
		return nil, err
	}
	return &document, nil
}

func (r *kycDocumentRepositoryImpl) GetByMerchantID(ctx context.Context, merchantID string) ([]*merchantDomain.KYCDocument, error) {
	var documents []*merchantDomain.KYCDocument
	err := r.db.WithContext(ctx).Where("merchant_id = ?", merchantID).Order("created_at DESC").Find(&documents).Error
	return documents, err
}

func (r *kycDocumentRepositoryImpl) List(ctx context.Context, filter KYCDocumentFilter) ([]*merchantDomain.KYCDocument, error) {
	query := r.db.WithContext(ctx).Model(&merchantDomain.KYCDocument{})

	if filter.MerchantID != "" {
		query = query.Where("merchant_id = ?", filter.MerchantID)
	}
	if filter.DocumentType != "" {
		query = query.Where("document_type = ?", filter.DocumentType)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var documents []*merchantDomain.KYCDocument
	err := query.Order("created_at DESC").Find(&documents).Error
	return documents, err
}

func (r *kycDocumentRepositoryImpl) ListByStatus(ctx context.Context, status merchantDomain.KYCDocumentStatus) ([]*merchantDomain.KYCDocument, error) {
	var documents []*merchantDomain.KYCDocument
	err := r.db.WithContext(ctx).Where("status = ?", status).Order("created_at DESC").Find(&documents).Error
	return documents, err
}

func (r *kycDocumentRepositoryImpl) Update(ctx context.Context, document *merchantDomain.KYCDocument) error {
	document.UpdatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Save(document).Error
}

func (r *kycDocumentRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id.String()).Delete(&merchantDomain.KYCDocument{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrKYCDocumentNotFound
	}
	return nil
}

func (r *kycDocumentRepositoryImpl) Approve(ctx context.Context, id string, reviewerID string, notes string) error {
	var document merchantDomain.KYCDocument
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&document).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrKYCDocumentNotFound
		}
		return err
	}

	if document.HasBeenReviewed() {
		return ErrKYCDocumentAlreadyReviewed
	}

	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":         merchantDomain.KYCDocumentStatusApproved,
		"reviewed_by":    sql.NullString{String: reviewerID, Valid: true},
		"reviewed_at":    sql.NullTime{Time: now, Valid: true},
		"reviewer_notes": sql.NullString{String: notes, Valid: notes != ""},
		"updated_at":     now,
	}

	return r.db.WithContext(ctx).Model(&merchantDomain.KYCDocument{}).Where("id = ?", id).Updates(updates).Error
}

func (r *kycDocumentRepositoryImpl) Reject(ctx context.Context, id string, reviewerID string, notes string) error {
	var document merchantDomain.KYCDocument
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&document).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrKYCDocumentNotFound
		}
		return err
	}

	if document.HasBeenReviewed() {
		return ErrKYCDocumentAlreadyReviewed
	}

	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":         merchantDomain.KYCDocumentStatusRejected,
		"reviewed_by":    sql.NullString{String: reviewerID, Valid: true},
		"reviewed_at":    sql.NullTime{Time: now, Valid: true},
		"reviewer_notes": sql.NullString{String: notes, Valid: notes != ""},
		"updated_at":     now,
	}

	return r.db.WithContext(ctx).Model(&merchantDomain.KYCDocument{}).Where("id = ?", id).Updates(updates).Error
}

func (r *kycDocumentRepositoryImpl) GetPendingDocuments(ctx context.Context, limit int) ([]*merchantDomain.KYCDocument, error) {
	if limit <= 0 {
		limit = 100
	}
	var documents []*merchantDomain.KYCDocument
	err := r.db.WithContext(ctx).
		Where("status = ?", merchantDomain.KYCDocumentStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&documents).Error
	return documents, err
}

func (r *kycDocumentRepositoryImpl) CountByMerchantAndStatus(ctx context.Context, merchantID string, status merchantDomain.KYCDocumentStatus) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&merchantDomain.KYCDocument{}).
		Where("merchant_id = ? AND status = ?", merchantID, status).
		Count(&count).Error
	return count, err
}
