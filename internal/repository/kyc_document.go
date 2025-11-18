package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
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
	GetByID(ctx context.Context, id uuid.UUID) (*model.KYCDocument, error)
	GetByMerchantID(ctx context.Context, merchantID string) ([]*model.KYCDocument, error)
	List(ctx context.Context, filter KYCDocumentFilter) ([]*model.KYCDocument, error)
	ListByStatus(ctx context.Context, status model.KYCDocumentStatus) ([]*model.KYCDocument, error)
	Update(ctx context.Context, document *model.KYCDocument) error
	Delete(ctx context.Context, id uuid.UUID) error
	Approve(ctx context.Context, id string, reviewerID string, notes string) error
	Reject(ctx context.Context, id string, reviewerID string, notes string) error
	GetPendingDocuments(ctx context.Context, limit int) ([]*model.KYCDocument, error)
	CountByMerchantAndStatus(ctx context.Context, merchantID string, status model.KYCDocumentStatus) (int64, error)
	HasApprovedDocumentOfType(ctx context.Context, merchantID string, docType model.KYCDocumentType) (bool, error)
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
	db     *sql.DB
	logger *logger.Logger
}

// NewKYCDocumentRepository creates a new KYC document repository
func NewKYCDocumentRepository(db *sql.DB, log *logger.Logger) KYCDocumentRepository {
	return &kycDocumentRepositoryImpl{
		db:     db,
		logger: log,
	}
}

// HasApprovedDocumentOfType checks if merchant has an approved document of a specific type
// This is the CRITICAL method used by ComplianceService.CheckKYCTierRequirements
func (r *kycDocumentRepositoryImpl) HasApprovedDocumentOfType(ctx context.Context, merchantID string, docType model.KYCDocumentType) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM kyc_documents
			WHERE merchant_id = $1
			  AND document_type = $2
			  AND status = $3
			LIMIT 1
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, merchantID, string(docType), model.KYCDocumentStatusApproved).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check document: %w", err)
	}

	return exists, nil
}

// Stub implementations for other methods (can be implemented later as needed)

func (r *kycDocumentRepositoryImpl) Create(ctx context.Context, document *model.KYCDocument) error {
	// TODO: Implement when KYC upload feature is needed
	r.logger.Warn("KYCDocumentRepository.Create not yet implemented")
	return errors.New("not implemented")
}

func (r *kycDocumentRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*model.KYCDocument, error) {
	// TODO: Implement when needed
	r.logger.Warn("KYCDocumentRepository.GetByID not yet implemented")
	return nil, errors.New("not implemented")
}

func (r *kycDocumentRepositoryImpl) GetByMerchantID(ctx context.Context, merchantID string) ([]*model.KYCDocument, error) {
	// TODO: Implement when needed
	r.logger.Warn("KYCDocumentRepository.GetByMerchantID not yet implemented")
	return nil, errors.New("not implemented")
}

func (r *kycDocumentRepositoryImpl) List(ctx context.Context, filter KYCDocumentFilter) ([]*model.KYCDocument, error) {
	// TODO: Implement when needed
	r.logger.Warn("KYCDocumentRepository.List not yet implemented")
	return nil, errors.New("not implemented")
}

func (r *kycDocumentRepositoryImpl) ListByStatus(ctx context.Context, status model.KYCDocumentStatus) ([]*model.KYCDocument, error) {
	// TODO: Implement when needed
	r.logger.Warn("KYCDocumentRepository.ListByStatus not yet implemented")
	return nil, errors.New("not implemented")
}

func (r *kycDocumentRepositoryImpl) Update(ctx context.Context, document *model.KYCDocument) error {
	// TODO: Implement when needed
	r.logger.Warn("KYCDocumentRepository.Update not yet implemented")
	return errors.New("not implemented")
}

func (r *kycDocumentRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	// TODO: Implement when needed
	r.logger.Warn("KYCDocumentRepository.Delete not yet implemented")
	return errors.New("not implemented")
}

func (r *kycDocumentRepositoryImpl) Approve(ctx context.Context, id string, reviewerID string, notes string) error {
	// TODO: Implement when needed
	r.logger.Warn("KYCDocumentRepository.Approve not yet implemented")
	return errors.New("not implemented")
}

func (r *kycDocumentRepositoryImpl) Reject(ctx context.Context, id string, reviewerID string, notes string) error {
	// TODO: Implement when needed
	r.logger.Warn("KYCDocumentRepository.Reject not yet implemented")
	return errors.New("not implemented")
}

func (r *kycDocumentRepositoryImpl) GetPendingDocuments(ctx context.Context, limit int) ([]*model.KYCDocument, error) {
	// TODO: Implement when needed
	r.logger.Warn("KYCDocumentRepository.GetPendingDocuments not yet implemented")
	return nil, errors.New("not implemented")
}

func (r *kycDocumentRepositoryImpl) CountByMerchantAndStatus(ctx context.Context, merchantID string, status model.KYCDocumentStatus) (int64, error) {
	// TODO: Implement when needed
	r.logger.Warn("KYCDocumentRepository.CountByMerchantAndStatus not yet implemented")
	return 0, errors.New("not implemented")
}
