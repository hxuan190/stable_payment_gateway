package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

var (
	ErrKYCSubmissionNotFound = errors.New("KYC submission not found")
	ErrMerchantNotFound      = errors.New("merchant not found")
)

// KYCRepository handles database operations for KYC
type KYCRepository interface {
	// KYC Submissions
	CreateSubmission(ctx context.Context, submission *model.KYCSubmission) error
	GetSubmissionByID(ctx context.Context, id uuid.UUID) (*model.KYCSubmission, error)
	GetSubmissionByMerchantID(ctx context.Context, merchantID uuid.UUID) (*model.KYCSubmission, error)
	UpdateSubmission(ctx context.Context, submission *model.KYCSubmission) error
	ListSubmissionsByStatus(ctx context.Context, status model.KYCStatus, limit, offset int) ([]*model.KYCSubmission, error)
	CountSubmissionsByStatus(ctx context.Context, status model.KYCStatus) (int64, error)

	// Documents
	CreateDocument(ctx context.Context, doc *model.KYCDocument) error
	GetDocumentByID(ctx context.Context, id uuid.UUID) (*model.KYCDocument, error)
	ListDocumentsBySubmission(ctx context.Context, submissionID uuid.UUID) ([]*model.KYCDocument, error)
	UpdateDocument(ctx context.Context, doc *model.KYCDocument) error
	DeleteDocument(ctx context.Context, id uuid.UUID) error

	// Verification Results
	CreateVerificationResult(ctx context.Context, result *model.KYCVerificationResult) error
	GetVerificationResult(ctx context.Context, submissionID uuid.UUID, verificationType string) (*model.KYCVerificationResult, error)
	ListVerificationResultsBySubmission(ctx context.Context, submissionID uuid.UUID) ([]*model.KYCVerificationResult, error)
	UpdateVerificationResult(ctx context.Context, result *model.KYCVerificationResult) error

	// Risk Assessment
	CreateRiskAssessment(ctx context.Context, assessment *model.KYCRiskAssessment) error
	GetRiskAssessmentBySubmission(ctx context.Context, submissionID uuid.UUID) (*model.KYCRiskAssessment, error)
	UpdateRiskAssessment(ctx context.Context, assessment *model.KYCRiskAssessment) error

	// Review Actions
	CreateReviewAction(ctx context.Context, action *model.KYCReviewAction) error
	ListReviewActionsBySubmission(ctx context.Context, submissionID uuid.UUID) ([]*model.KYCReviewAction, error)
	GetLatestReviewAction(ctx context.Context, submissionID uuid.UUID) (*model.KYCReviewAction, error)

	// Audit Logs
	CreateAuditLog(ctx context.Context, log *model.KYCAuditLog) error
	ListAuditLogsBySubmission(ctx context.Context, submissionID uuid.UUID, limit, offset int) ([]*model.KYCAuditLog, error)
	ListAuditLogsByMerchant(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]*model.KYCAuditLog, error)

	// Merchant operations
	GetMerchantByID(ctx context.Context, id uuid.UUID) (*model.Merchant, error)
	UpdateMerchant(ctx context.Context, merchant *model.Merchant) error
	UpdateMerchantKYCStatus(ctx context.Context, merchantID uuid.UUID, status model.KYCStatus) error
}

// kycRepository is the concrete implementation
type kycRepository struct {
	db *gorm.DB
}

// NewKYCRepository creates a new KYC repository
func NewKYCRepository(db *gorm.DB) KYCRepository {
	return &kycRepository{db: db}
}

// ===== KYC Submissions =====

func (r *kycRepository) CreateSubmission(ctx context.Context, submission *model.KYCSubmission) error {
	return r.db.WithContext(ctx).Create(submission).Error
}

func (r *kycRepository) GetSubmissionByID(ctx context.Context, id uuid.UUID) (*model.KYCSubmission, error) {
	var submission model.KYCSubmission
	err := r.db.WithContext(ctx).
		Preload("Documents").
		Preload("VerificationResults").
		Preload("RiskAssessment").
		Preload("ReviewActions").
		First(&submission, "id = ?", id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrKYCSubmissionNotFound
		}
		return nil, err
	}

	return &submission, nil
}

func (r *kycRepository) GetSubmissionByMerchantID(ctx context.Context, merchantID uuid.UUID) (*model.KYCSubmission, error) {
	var submission model.KYCSubmission
	err := r.db.WithContext(ctx).
		Preload("Documents").
		Preload("VerificationResults").
		Preload("RiskAssessment").
		Preload("ReviewActions").
		Where("merchant_id = ?", merchantID).
		Order("created_at DESC").
		First(&submission).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrKYCSubmissionNotFound
		}
		return nil, err
	}

	return &submission, nil
}

func (r *kycRepository) UpdateSubmission(ctx context.Context, submission *model.KYCSubmission) error {
	submission.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(submission).Error
}

func (r *kycRepository) ListSubmissionsByStatus(ctx context.Context, status model.KYCStatus, limit, offset int) ([]*model.KYCSubmission, error) {
	var submissions []*model.KYCSubmission
	err := r.db.WithContext(ctx).
		Preload("Merchant").
		Where("status = ?", status).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&submissions).Error

	return submissions, err
}

func (r *kycRepository) CountSubmissionsByStatus(ctx context.Context, status model.KYCStatus) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.KYCSubmission{}).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

// ===== Documents =====

func (r *kycRepository) CreateDocument(ctx context.Context, doc *model.KYCDocument) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *kycRepository) GetDocumentByID(ctx context.Context, id uuid.UUID) (*model.KYCDocument, error) {
	var doc model.KYCDocument
	err := r.db.WithContext(ctx).First(&doc, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("document not found")
		}
		return nil, err
	}
	return &doc, nil
}

func (r *kycRepository) ListDocumentsBySubmission(ctx context.Context, submissionID uuid.UUID) ([]*model.KYCDocument, error) {
	var docs []*model.KYCDocument
	err := r.db.WithContext(ctx).
		Where("kyc_submission_id = ?", submissionID).
		Order("created_at ASC").
		Find(&docs).Error
	return docs, err
}

func (r *kycRepository) UpdateDocument(ctx context.Context, doc *model.KYCDocument) error {
	return r.db.WithContext(ctx).Save(doc).Error
}

func (r *kycRepository) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.KYCDocument{}, "id = ?", id).Error
}

// ===== Verification Results =====

func (r *kycRepository) CreateVerificationResult(ctx context.Context, result *model.KYCVerificationResult) error {
	return r.db.WithContext(ctx).Create(result).Error
}

func (r *kycRepository) GetVerificationResult(ctx context.Context, submissionID uuid.UUID, verificationType string) (*model.KYCVerificationResult, error) {
	var result model.KYCVerificationResult
	err := r.db.WithContext(ctx).
		Where("kyc_submission_id = ? AND verification_type = ?", submissionID, verificationType).
		First(&result).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("verification result not found")
		}
		return nil, err
	}
	return &result, nil
}

func (r *kycRepository) ListVerificationResultsBySubmission(ctx context.Context, submissionID uuid.UUID) ([]*model.KYCVerificationResult, error) {
	var results []*model.KYCVerificationResult
	err := r.db.WithContext(ctx).
		Where("kyc_submission_id = ?", submissionID).
		Order("created_at ASC").
		Find(&results).Error
	return results, err
}

func (r *kycRepository) UpdateVerificationResult(ctx context.Context, result *model.KYCVerificationResult) error {
	return r.db.WithContext(ctx).Save(result).Error
}

// ===== Risk Assessment =====

func (r *kycRepository) CreateRiskAssessment(ctx context.Context, assessment *model.KYCRiskAssessment) error {
	return r.db.WithContext(ctx).Create(assessment).Error
}

func (r *kycRepository) GetRiskAssessmentBySubmission(ctx context.Context, submissionID uuid.UUID) (*model.KYCRiskAssessment, error) {
	var assessment model.KYCRiskAssessment
	err := r.db.WithContext(ctx).
		Where("kyc_submission_id = ?", submissionID).
		First(&assessment).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("risk assessment not found")
		}
		return nil, err
	}
	return &assessment, nil
}

func (r *kycRepository) UpdateRiskAssessment(ctx context.Context, assessment *model.KYCRiskAssessment) error {
	return r.db.WithContext(ctx).Save(assessment).Error
}

// ===== Review Actions =====

func (r *kycRepository) CreateReviewAction(ctx context.Context, action *model.KYCReviewAction) error {
	return r.db.WithContext(ctx).Create(action).Error
}

func (r *kycRepository) ListReviewActionsBySubmission(ctx context.Context, submissionID uuid.UUID) ([]*model.KYCReviewAction, error) {
	var actions []*model.KYCReviewAction
	err := r.db.WithContext(ctx).
		Where("kyc_submission_id = ?", submissionID).
		Order("created_at DESC").
		Find(&actions).Error
	return actions, err
}

func (r *kycRepository) GetLatestReviewAction(ctx context.Context, submissionID uuid.UUID) (*model.KYCReviewAction, error) {
	var action model.KYCReviewAction
	err := r.db.WithContext(ctx).
		Where("kyc_submission_id = ?", submissionID).
		Order("created_at DESC").
		First(&action).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No review action yet is not an error
		}
		return nil, err
	}
	return &action, nil
}

// ===== Audit Logs =====

func (r *kycRepository) CreateAuditLog(ctx context.Context, log *model.KYCAuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *kycRepository) ListAuditLogsBySubmission(ctx context.Context, submissionID uuid.UUID, limit, offset int) ([]*model.KYCAuditLog, error) {
	var logs []*model.KYCAuditLog
	err := r.db.WithContext(ctx).
		Where("kyc_submission_id = ?", submissionID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

func (r *kycRepository) ListAuditLogsByMerchant(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]*model.KYCAuditLog, error) {
	var logs []*model.KYCAuditLog
	err := r.db.WithContext(ctx).
		Where("merchant_id = ?", merchantID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

// ===== Merchant Operations =====

func (r *kycRepository) GetMerchantByID(ctx context.Context, id uuid.UUID) (*model.Merchant, error) {
	var merchant model.Merchant
	err := r.db.WithContext(ctx).First(&merchant, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMerchantNotFound
		}
		return nil, err
	}
	return &merchant, nil
}

func (r *kycRepository) UpdateMerchant(ctx context.Context, merchant *model.Merchant) error {
	merchant.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(merchant).Error
}

func (r *kycRepository) UpdateMerchantKYCStatus(ctx context.Context, merchantID uuid.UUID, status model.KYCStatus) error {
	return r.db.WithContext(ctx).
		Model(&model.Merchant{}).
		Where("id = ?", merchantID).
		Updates(map[string]interface{}{
			"kyc_status": status,
			"updated_at": time.Now(),
		}).Error
}
