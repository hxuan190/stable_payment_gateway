package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

var (
	// ErrKYCDocumentNotFound is returned when a KYC document is not found
	ErrKYCDocumentNotFound = errors.New("KYC document not found")
	// ErrInvalidKYCDocument is returned when a KYC document is invalid
	ErrInvalidKYCDocument = errors.New("invalid KYC document")
	// ErrKYCDocumentAlreadyReviewed is returned when attempting to review an already reviewed document
	ErrKYCDocumentAlreadyReviewed = errors.New("KYC document has already been reviewed")
)

// KYCDocumentRepository defines the interface for KYC document data access
type KYCDocumentRepository interface {
	Create(ctx context.Context, document *model.KYCDocument) error
	GetByID(ctx context.Context, id string) (*model.KYCDocument, error)
	GetByMerchantID(ctx context.Context, merchantID string) ([]*model.KYCDocument, error)
	List(ctx context.Context, filter KYCDocumentFilter) ([]*model.KYCDocument, error)
	Update(ctx context.Context, document *model.KYCDocument) error
	Delete(ctx context.Context, id string) error
	Approve(ctx context.Context, id string, reviewerID string, notes string) error
	Reject(ctx context.Context, id string, reviewerID string, notes string) error
	GetPendingDocuments(ctx context.Context, limit int) ([]*model.KYCDocument, error)
	CountByMerchantAndStatus(ctx context.Context, merchantID string, status model.KYCDocumentStatus) (int64, error)
}

// KYCDocumentFilter represents filters for querying KYC documents
type KYCDocumentFilter struct {
	MerchantID   string
	DocumentType string
	Status       model.KYCDocumentStatus
	Limit        int
	Offset       int
}

type kycDocumentRepositoryImpl struct {
	db *gorm.DB
}

// NewKYCDocumentRepository creates a new KYC document repository
func NewKYCDocumentRepository(db *gorm.DB) KYCDocumentRepository {
	return &kycDocumentRepositoryImpl{
		db: db,
	}
}

// Create creates a new KYC document record
func (r *kycDocumentRepositoryImpl) Create(ctx context.Context, document *model.KYCDocument) error {
	if document.ID == "" {
		document.ID = uuid.New().String()
	}

	if document.CreatedAt.IsZero() {
		document.CreatedAt = time.Now()
	}

	if document.UpdatedAt.IsZero() {
		document.UpdatedAt = time.Now()
	}

	// Default status is pending
	if document.Status == "" {
		document.Status = model.KYCDocumentStatusPending
	}

	result := r.db.WithContext(ctx).Create(document)
	if result.Error != nil {
		return fmt.Errorf("failed to create KYC document: %w", result.Error)
	}

	return nil
}

// GetByID retrieves a KYC document by ID
func (r *kycDocumentRepositoryImpl) GetByID(ctx context.Context, id string) (*model.KYCDocument, error) {
	var document model.KYCDocument
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&document)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrKYCDocumentNotFound
		}
		return nil, fmt.Errorf("failed to get KYC document: %w", result.Error)
	}

	return &document, nil
}

// GetByMerchantID retrieves all KYC documents for a merchant
func (r *kycDocumentRepositoryImpl) GetByMerchantID(ctx context.Context, merchantID string) ([]*model.KYCDocument, error) {
	var documents []*model.KYCDocument
	result := r.db.WithContext(ctx).
		Where("merchant_id = ?", merchantID).
		Order("created_at DESC").
		Find(&documents)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get KYC documents by merchant ID: %w", result.Error)
	}

	return documents, nil
}

// List retrieves KYC documents with filters
func (r *kycDocumentRepositoryImpl) List(ctx context.Context, filter KYCDocumentFilter) ([]*model.KYCDocument, error) {
	query := r.db.WithContext(ctx).Model(&model.KYCDocument{})

	// Apply filters
	if filter.MerchantID != "" {
		query = query.Where("merchant_id = ?", filter.MerchantID)
	}

	if filter.DocumentType != "" {
		query = query.Where("document_type = ?", filter.DocumentType)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	// Apply sorting
	query = query.Order("created_at DESC")

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var documents []*model.KYCDocument
	result := query.Find(&documents)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to list KYC documents: %w", result.Error)
	}

	return documents, nil
}

// Update updates a KYC document
func (r *kycDocumentRepositoryImpl) Update(ctx context.Context, document *model.KYCDocument) error {
	document.UpdatedAt = time.Now()

	result := r.db.WithContext(ctx).
		Where("id = ?", document.ID).
		Updates(document)

	if result.Error != nil {
		return fmt.Errorf("failed to update KYC document: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrKYCDocumentNotFound
	}

	return nil
}

// Delete deletes a KYC document by ID
func (r *kycDocumentRepositoryImpl) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&model.KYCDocument{}, "id = ?", id)

	if result.Error != nil {
		return fmt.Errorf("failed to delete KYC document: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrKYCDocumentNotFound
	}

	return nil
}

// Approve approves a KYC document
func (r *kycDocumentRepositoryImpl) Approve(ctx context.Context, id string, reviewerID string, notes string) error {
	// First, check if document exists and hasn't been reviewed
	document, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if document.HasBeenReviewed() {
		return ErrKYCDocumentAlreadyReviewed
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":         model.KYCDocumentStatusApproved,
		"reviewed_by":    reviewerID,
		"reviewed_at":    now,
		"reviewer_notes": notes,
		"updated_at":     now,
	}

	result := r.db.WithContext(ctx).
		Model(&model.KYCDocument{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to approve KYC document: %w", result.Error)
	}

	return nil
}

// Reject rejects a KYC document
func (r *kycDocumentRepositoryImpl) Reject(ctx context.Context, id string, reviewerID string, notes string) error {
	// First, check if document exists and hasn't been reviewed
	document, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if document.HasBeenReviewed() {
		return ErrKYCDocumentAlreadyReviewed
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":         model.KYCDocumentStatusRejected,
		"reviewed_by":    reviewerID,
		"reviewed_at":    now,
		"reviewer_notes": notes,
		"updated_at":     now,
	}

	result := r.db.WithContext(ctx).
		Model(&model.KYCDocument{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to reject KYC document: %w", result.Error)
	}

	return nil
}

// GetPendingDocuments retrieves documents that are pending review
func (r *kycDocumentRepositoryImpl) GetPendingDocuments(ctx context.Context, limit int) ([]*model.KYCDocument, error) {
	var documents []*model.KYCDocument
	query := r.db.WithContext(ctx).
		Where("status = ?", model.KYCDocumentStatusPending).
		Order("created_at ASC") // FIFO: oldest first

	if limit > 0 {
		query = query.Limit(limit)
	}

	result := query.Find(&documents)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get pending documents: %w", result.Error)
	}

	return documents, nil
}

// CountByMerchantAndStatus counts documents by merchant and status
func (r *kycDocumentRepositoryImpl) CountByMerchantAndStatus(ctx context.Context, merchantID string, status model.KYCDocumentStatus) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.KYCDocument{}).
		Where("merchant_id = ? AND status = ?", merchantID, status).
		Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to count documents: %w", result.Error)
	}

	return count, nil
}

// GetDocumentsByTypeAndMerchant retrieves documents by type for a specific merchant
func (r *kycDocumentRepositoryImpl) GetDocumentsByTypeAndMerchant(ctx context.Context, merchantID string, docType model.KYCDocumentType) ([]*model.KYCDocument, error) {
	var documents []*model.KYCDocument
	result := r.db.WithContext(ctx).
		Where("merchant_id = ? AND document_type = ?", merchantID, string(docType)).
		Order("created_at DESC").
		Find(&documents)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get documents by type: %w", result.Error)
	}

	return documents, nil
}

// GetApprovedDocumentsByMerchant retrieves all approved documents for a merchant
func (r *kycDocumentRepositoryImpl) GetApprovedDocumentsByMerchant(ctx context.Context, merchantID string) ([]*model.KYCDocument, error) {
	var documents []*model.KYCDocument
	result := r.db.WithContext(ctx).
		Where("merchant_id = ? AND status = ?", merchantID, model.KYCDocumentStatusApproved).
		Order("created_at DESC").
		Find(&documents)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get approved documents: %w", result.Error)
	}

	return documents, nil
}

// HasApprovedDocumentOfType checks if merchant has an approved document of a specific type
func (r *kycDocumentRepositoryImpl) HasApprovedDocumentOfType(ctx context.Context, merchantID string, docType model.KYCDocumentType) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.KYCDocument{}).
		Where("merchant_id = ? AND document_type = ? AND status = ?", merchantID, string(docType), model.KYCDocumentStatusApproved).
		Count(&count)

	if result.Error != nil {
		return false, fmt.Errorf("failed to check document: %w", result.Error)
	}

	return count > 0, nil
}

// Ensure model.KYCDocument implements the correct table name
func init() {
	// This ensures GORM uses the correct table name
	sql.Register("kyc_documents", &model.KYCDocument{})
}
