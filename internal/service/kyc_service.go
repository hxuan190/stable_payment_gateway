package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
)

var (
	ErrInvalidMerchantType    = errors.New("invalid merchant type")
	ErrKYCAlreadySubmitted    = errors.New("KYC already submitted for this merchant")
	ErrKYCNotFound            = errors.New("KYC submission not found")
	ErrInvalidKYCStatus       = errors.New("invalid KYC status for this operation")
	ErrMissingRequiredFields  = errors.New("missing required fields")
	ErrDocumentUploadFailed   = errors.New("document upload failed")
	ErrVerificationFailed     = errors.New("verification failed")
)

// KYCService handles KYC business logic
type KYCService interface {
	// Submission management
	CreateSubmission(ctx context.Context, req *CreateKYCSubmissionRequest) (*model.KYCSubmission, error)
	GetSubmission(ctx context.Context, submissionID uuid.UUID) (*model.KYCSubmission, error)
	GetSubmissionByMerchant(ctx context.Context, merchantID uuid.UUID) (*model.KYCSubmission, error)
	SubmitForReview(ctx context.Context, submissionID uuid.UUID) error

	// Document management
	UploadDocument(ctx context.Context, req *UploadDocumentRequest) (*model.KYCDocument, error)
	GetDocument(ctx context.Context, documentID uuid.UUID) (*model.KYCDocument, error)
	ListDocuments(ctx context.Context, submissionID uuid.UUID) ([]*model.KYCDocument, error)

	// Verification
	RunVerification(ctx context.Context, submissionID uuid.UUID, verificationType string) (*model.KYCVerificationResult, error)
	RunAllVerifications(ctx context.Context, submissionID uuid.UUID) ([]*model.KYCVerificationResult, error)

	// Risk assessment
	AssessRisk(ctx context.Context, submissionID uuid.UUID) (*model.KYCRiskAssessment, error)

	// Auto-approval
	ProcessAutoApproval(ctx context.Context, submissionID uuid.UUID) (bool, error)

	// Admin review
	ApproveKYC(ctx context.Context, req *ApproveKYCRequest) error
	RejectKYC(ctx context.Context, req *RejectKYCRequest) error
	RequestMoreInfo(ctx context.Context, req *RequestMoreInfoRequest) error

	// Listing
	ListPendingReviews(ctx context.Context, limit, offset int) ([]*model.KYCSubmission, error)
	ListByStatus(ctx context.Context, status model.KYCStatus, limit, offset int) ([]*model.KYCSubmission, error)
}

// kycService is the concrete implementation
type kycService struct {
	repo                repository.KYCRepository
	verificationService VerificationService
	riskService         RiskAssessmentService
	auditService        AuditService
	storageService      StorageService
}

// NewKYCService creates a new KYC service
func NewKYCService(
	repo repository.KYCRepository,
	verificationService VerificationService,
	riskService RiskAssessmentService,
	auditService AuditService,
	storageService StorageService,
) KYCService {
	return &kycService{
		repo:                repo,
		verificationService: verificationService,
		riskService:         riskService,
		auditService:        auditService,
		storageService:      storageService,
	}
}

// ===== Request/Response Structs =====

type CreateKYCSubmissionRequest struct {
	MerchantID   uuid.UUID           `json:"merchant_id"`
	MerchantType model.MerchantType  `json:"merchant_type"`

	// Individual fields
	IndividualFullName    *string    `json:"individual_full_name,omitempty"`
	IndividualCCCDNumber  *string    `json:"individual_cccd_number,omitempty"`
	IndividualDateOfBirth *time.Time `json:"individual_date_of_birth,omitempty"`
	IndividualAddress     *string    `json:"individual_address,omitempty"`

	// Household business fields
	HKTBusinessName    *string `json:"hkt_business_name,omitempty"`
	HKTTaxID           *string `json:"hkt_tax_id,omitempty"`
	HKTBusinessAddress *string `json:"hkt_business_address,omitempty"`
	HKTOwnerName       *string `json:"hkt_owner_name,omitempty"`
	HKTOwnerCCCD       *string `json:"hkt_owner_cccd,omitempty"`

	// Company fields
	CompanyLegalName            *string `json:"company_legal_name,omitempty"`
	CompanyRegistrationNumber   *string `json:"company_registration_number,omitempty"`
	CompanyTaxID                *string `json:"company_tax_id,omitempty"`
	CompanyHeadquartersAddress  *string `json:"company_headquarters_address,omitempty"`
	CompanyWebsite              *string `json:"company_website,omitempty"`
	CompanyDirectorName         *string `json:"company_director_name,omitempty"`
	CompanyDirectorCCCD         *string `json:"company_director_cccd,omitempty"`

	// Common fields
	BusinessDescription *string `json:"business_description,omitempty"`
	BusinessWebsite     *string `json:"business_website,omitempty"`
	BusinessFacebook    *string `json:"business_facebook,omitempty"`
	ProductCategory     *string `json:"product_category,omitempty"`
}

type UploadDocumentRequest struct {
	SubmissionID uuid.UUID              `json:"submission_id"`
	MerchantID   uuid.UUID              `json:"merchant_id"`
	DocumentType model.KYCDocumentType  `json:"document_type"`
	FileName     string                 `json:"file_name"`
	FileData     []byte                 `json:"file_data"`
	MimeType     string                 `json:"mime_type"`
}

type ApproveKYCRequest struct {
	SubmissionID        uuid.UUID  `json:"submission_id"`
	ReviewerID          uuid.UUID  `json:"reviewer_id"`
	ReviewerEmail       string     `json:"reviewer_email"`
	ReviewerName        string     `json:"reviewer_name"`
	Notes               *string    `json:"notes,omitempty"`
	DailyLimitVND       *float64   `json:"daily_limit_vnd,omitempty"`
	MonthlyLimitVND     *float64   `json:"monthly_limit_vnd,omitempty"`
}

type RejectKYCRequest struct {
	SubmissionID  uuid.UUID `json:"submission_id"`
	ReviewerID    uuid.UUID `json:"reviewer_id"`
	ReviewerEmail string    `json:"reviewer_email"`
	ReviewerName  string    `json:"reviewer_name"`
	Reason        string    `json:"reason"`
	Notes         *string   `json:"notes,omitempty"`
}

type RequestMoreInfoRequest struct {
	SubmissionID      uuid.UUID                 `json:"submission_id"`
	ReviewerID        uuid.UUID                 `json:"reviewer_id"`
	ReviewerEmail     string                    `json:"reviewer_email"`
	ReviewerName      string                    `json:"reviewer_name"`
	RequiredDocuments []model.KYCDocumentType   `json:"required_documents"`
	Notes             *string                   `json:"notes,omitempty"`
}

// ===== Implementation =====

func (s *kycService) CreateSubmission(ctx context.Context, req *CreateKYCSubmissionRequest) (*model.KYCSubmission, error) {
	// Validate merchant exists
	merchant, err := s.repo.GetMerchantByID(ctx, req.MerchantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merchant: %w", err)
	}

	// Check if merchant already has a submission
	existing, err := s.repo.GetSubmissionByMerchantID(ctx, req.MerchantID)
	if err == nil && existing != nil {
		// Allow creating a new submission only if previous one was rejected
		if existing.Status != model.KYCStatusRejected {
			return nil, ErrKYCAlreadySubmitted
		}
	}

	// Validate required fields based on merchant type
	if err := s.validateSubmissionFields(req); err != nil {
		return nil, err
	}

	// Create submission
	submission := &model.KYCSubmission{
		MerchantID:   req.MerchantID,
		MerchantType: req.MerchantType,
		Status:       model.KYCStatusInProgress,

		// Individual fields
		IndividualFullName:    req.IndividualFullName,
		IndividualCCCDNumber:  req.IndividualCCCDNumber,
		IndividualDateOfBirth: req.IndividualDateOfBirth,
		IndividualAddress:     req.IndividualAddress,

		// Household business fields
		HKTBusinessName:    req.HKTBusinessName,
		HKTTaxID:           req.HKTTaxID,
		HKTBusinessAddress: req.HKTBusinessAddress,
		HKTOwnerName:       req.HKTOwnerName,
		HKTOwnerCCCD:       req.HKTOwnerCCCD,

		// Company fields
		CompanyLegalName:            req.CompanyLegalName,
		CompanyRegistrationNumber:   req.CompanyRegistrationNumber,
		CompanyTaxID:                req.CompanyTaxID,
		CompanyHeadquartersAddress:  req.CompanyHeadquartersAddress,
		CompanyWebsite:              req.CompanyWebsite,
		CompanyDirectorName:         req.CompanyDirectorName,
		CompanyDirectorCCCD:         req.CompanyDirectorCCCD,

		// Common fields
		BusinessDescription: req.BusinessDescription,
		BusinessWebsite:     req.BusinessWebsite,
		BusinessFacebook:    req.BusinessFacebook,
		ProductCategory:     req.ProductCategory,
	}

	if err := s.repo.CreateSubmission(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	// Update merchant status
	if err := s.repo.UpdateMerchantKYCStatus(ctx, req.MerchantID, model.KYCStatusInProgress); err != nil {
		return nil, fmt.Errorf("failed to update merchant status: %w", err)
	}

	// Audit log
	s.auditService.LogKYCAction(ctx, &AuditLogRequest{
		SubmissionID: &submission.ID,
		MerchantID:   &req.MerchantID,
		ActorType:    model.ActorTypeMerchant,
		ActorID:      merchant.Email,
		Action:       "kyc_submission_created",
		ResourceType: "kyc_submission",
		ResourceID:   &submission.ID,
		NewStatus:    string(model.KYCStatusInProgress),
	})

	return submission, nil
}

func (s *kycService) GetSubmission(ctx context.Context, submissionID uuid.UUID) (*model.KYCSubmission, error) {
	return s.repo.GetSubmissionByID(ctx, submissionID)
}

func (s *kycService) GetSubmissionByMerchant(ctx context.Context, merchantID uuid.UUID) (*model.KYCSubmission, error) {
	return s.repo.GetSubmissionByMerchantID(ctx, merchantID)
}

func (s *kycService) SubmitForReview(ctx context.Context, submissionID uuid.UUID) error {
	submission, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return err
	}

	if submission.Status != model.KYCStatusInProgress {
		return ErrInvalidKYCStatus
	}

	// Verify all required documents are uploaded
	docs, err := s.repo.ListDocumentsBySubmission(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("failed to list documents: %w", err)
	}

	if err := s.validateRequiredDocuments(submission.MerchantType, docs); err != nil {
		return err
	}

	// Run all verifications
	verificationResults, err := s.RunAllVerifications(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	// Assess risk
	riskAssessment, err := s.AssessRisk(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("risk assessment failed: %w", err)
	}

	// Update submission
	now := time.Now()
	submission.SubmittedAt = &now

	// Try auto-approval
	autoApproved, err := s.ProcessAutoApproval(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("auto-approval processing failed: %w", err)
	}

	if autoApproved {
		submission.Status = model.KYCStatusApproved
		submission.AutoApproved = true
		submission.ApprovedAt = &now
	} else if riskAssessment.RequiresManualReview() {
		submission.Status = model.KYCStatusPendingReview
		submission.RequiresManualReview = true
	}

	if err := s.repo.UpdateSubmission(ctx, submission); err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}

	// Update merchant status
	if err := s.repo.UpdateMerchantKYCStatus(ctx, submission.MerchantID, submission.Status); err != nil {
		return fmt.Errorf("failed to update merchant status: %w", err)
	}

	// Audit log
	s.auditService.LogKYCAction(ctx, &AuditLogRequest{
		SubmissionID: &submission.ID,
		MerchantID:   &submission.MerchantID,
		ActorType:    model.ActorTypeSystem,
		Action:       "kyc_submitted_for_review",
		ResourceType: "kyc_submission",
		ResourceID:   &submission.ID,
		OldStatus:    string(model.KYCStatusInProgress),
		NewStatus:    string(submission.Status),
		Changes:      buildChangesJSON(verificationResults, riskAssessment),
	})

	return nil
}

func (s *kycService) UploadDocument(ctx context.Context, req *UploadDocumentRequest) (*model.KYCDocument, error) {
	// Verify submission exists and is in correct status
	submission, err := s.repo.GetSubmissionByID(ctx, req.SubmissionID)
	if err != nil {
		return nil, err
	}

	if submission.Status != model.KYCStatusInProgress {
		return nil, ErrInvalidKYCStatus
	}

	// Upload file to storage (S3/MinIO)
	filePath, err := s.storageService.UploadKYCDocument(ctx, req.MerchantID, req.DocumentType, req.FileName, req.FileData)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to storage: %w", err)
	}

	// Create document record
	fileSize := int64(len(req.FileData))
	doc := &model.KYCDocument{
		KYCSubmissionID: req.SubmissionID,
		MerchantID:      req.MerchantID,
		DocumentType:    req.DocumentType,
		FilePath:        filePath,
		FileName:        req.FileName,
		FileSizeBytes:   &fileSize,
		MimeType:        &req.MimeType,
	}

	if err := s.repo.CreateDocument(ctx, doc); err != nil {
		return nil, fmt.Errorf("failed to create document record: %w", err)
	}

	// Audit log
	s.auditService.LogKYCAction(ctx, &AuditLogRequest{
		SubmissionID: &req.SubmissionID,
		MerchantID:   &req.MerchantID,
		ActorType:    model.ActorTypeMerchant,
		Action:       "document_uploaded",
		ResourceType: "kyc_document",
		ResourceID:   &doc.ID,
	})

	return doc, nil
}

func (s *kycService) GetDocument(ctx context.Context, documentID uuid.UUID) (*model.KYCDocument, error) {
	return s.repo.GetDocumentByID(ctx, documentID)
}

func (s *kycService) ListDocuments(ctx context.Context, submissionID uuid.UUID) ([]*model.KYCDocument, error) {
	return s.repo.ListDocumentsBySubmission(ctx, submissionID)
}

func (s *kycService) RunVerification(ctx context.Context, submissionID uuid.UUID, verificationType string) (*model.KYCVerificationResult, error) {
	submission, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	// Run the specific verification
	result, err := s.verificationService.RunVerification(ctx, submission, verificationType)
	if err != nil {
		return nil, err
	}

	// Save result
	if err := s.repo.CreateVerificationResult(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to save verification result: %w", err)
	}

	return result, nil
}

func (s *kycService) RunAllVerifications(ctx context.Context, submissionID uuid.UUID) ([]*model.KYCVerificationResult, error) {
	submission, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	// Determine which verifications to run based on merchant type
	verificationTypes := s.getRequiredVerifications(submission.MerchantType)

	var results []*model.KYCVerificationResult
	for _, vType := range verificationTypes {
		result, err := s.verificationService.RunVerification(ctx, submission, vType)
		if err != nil {
			// Log error but continue with other verifications
			continue
		}

		if err := s.repo.CreateVerificationResult(ctx, result); err != nil {
			return nil, fmt.Errorf("failed to save verification result: %w", err)
		}

		results = append(results, result)
	}

	return results, nil
}

func (s *kycService) AssessRisk(ctx context.Context, submissionID uuid.UUID) (*model.KYCRiskAssessment, error) {
	submission, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	// Get verification results
	verificationResults, err := s.repo.ListVerificationResultsBySubmission(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get verification results: %w", err)
	}

	// Run risk assessment
	assessment, err := s.riskService.AssessRisk(ctx, submission, verificationResults)
	if err != nil {
		return nil, err
	}

	// Save assessment
	if err := s.repo.CreateRiskAssessment(ctx, assessment); err != nil {
		return nil, fmt.Errorf("failed to save risk assessment: %w", err)
	}

	return assessment, nil
}

func (s *kycService) ProcessAutoApproval(ctx context.Context, submissionID uuid.UUID) (bool, error) {
	// Get risk assessment
	assessment, err := s.repo.GetRiskAssessmentBySubmission(ctx, submissionID)
	if err != nil {
		return false, err
	}

	// Check if auto-approval is recommended
	if !assessment.ShouldAutoApprove() {
		return false, nil
	}

	// Get submission
	submission, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return false, err
	}

	// Set default limits based on merchant type
	limits := getDefaultLimits(submission.MerchantType)

	// Update merchant with approved status and limits
	merchant, err := s.repo.GetMerchantByID(ctx, submission.MerchantID)
	if err != nil {
		return false, err
	}

	merchant.KYCStatus = model.KYCStatusApproved
	merchant.DailyLimitVND = limits.DailyLimitVND
	merchant.MonthlyLimitVND = limits.MonthlyLimitVND

	if err := s.repo.UpdateMerchant(ctx, merchant); err != nil {
		return false, fmt.Errorf("failed to update merchant: %w", err)
	}

	// Audit log
	s.auditService.LogKYCAction(ctx, &AuditLogRequest{
		SubmissionID: &submission.ID,
		MerchantID:   &submission.MerchantID,
		ActorType:    model.ActorTypeSystem,
		Action:       "kyc_auto_approved",
		ResourceType: "kyc_submission",
		ResourceID:   &submission.ID,
		NewStatus:    string(model.KYCStatusApproved),
	})

	return true, nil
}

func (s *kycService) ApproveKYC(ctx context.Context, req *ApproveKYCRequest) error {
	submission, err := s.repo.GetSubmissionByID(ctx, req.SubmissionID)
	if err != nil {
		return err
	}

	if submission.Status != model.KYCStatusPendingReview {
		return ErrInvalidKYCStatus
	}

	// Create review action
	decision := model.DecisionApproved
	action := &model.KYCReviewAction{
		KYCSubmissionID: req.SubmissionID,
		MerchantID:      submission.MerchantID,
		ReviewerID:      &req.ReviewerID,
		ReviewerEmail:   &req.ReviewerEmail,
		ReviewerName:    &req.ReviewerName,
		Action:          model.ReviewActionApprove,
		Decision:        &decision,
		Notes:           req.Notes,
		ApprovedDailyLimitVND:   req.DailyLimitVND,
		ApprovedMonthlyLimitVND: req.MonthlyLimitVND,
	}

	if err := s.repo.CreateReviewAction(ctx, action); err != nil {
		return fmt.Errorf("failed to create review action: %w", err)
	}

	// Update submission
	now := time.Now()
	submission.Status = model.KYCStatusApproved
	submission.ApprovedAt = &now
	submission.ReviewedAt = &now

	if err := s.repo.UpdateSubmission(ctx, submission); err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}

	// Update merchant
	merchant, err := s.repo.GetMerchantByID(ctx, submission.MerchantID)
	if err != nil {
		return err
	}

	merchant.KYCStatus = model.KYCStatusApproved
	if req.DailyLimitVND != nil {
		merchant.DailyLimitVND = *req.DailyLimitVND
	}
	if req.MonthlyLimitVND != nil {
		merchant.MonthlyLimitVND = *req.MonthlyLimitVND
	}

	if err := s.repo.UpdateMerchant(ctx, merchant); err != nil {
		return fmt.Errorf("failed to update merchant: %w", err)
	}

	// Audit log
	s.auditService.LogKYCAction(ctx, &AuditLogRequest{
		SubmissionID: &req.SubmissionID,
		MerchantID:   &submission.MerchantID,
		ActorType:    model.ActorTypeAdmin,
		ActorID:      req.ReviewerEmail,
		Action:       "kyc_approved",
		ResourceType: "kyc_submission",
		ResourceID:   &req.SubmissionID,
		OldStatus:    string(model.KYCStatusPendingReview),
		NewStatus:    string(model.KYCStatusApproved),
	})

	return nil
}

func (s *kycService) RejectKYC(ctx context.Context, req *RejectKYCRequest) error {
	submission, err := s.repo.GetSubmissionByID(ctx, req.SubmissionID)
	if err != nil {
		return err
	}

	if submission.Status != model.KYCStatusPendingReview {
		return ErrInvalidKYCStatus
	}

	// Create review action
	decision := model.DecisionRejected
	action := &model.KYCReviewAction{
		KYCSubmissionID: req.SubmissionID,
		MerchantID:      submission.MerchantID,
		ReviewerID:      &req.ReviewerID,
		ReviewerEmail:   &req.ReviewerEmail,
		ReviewerName:    &req.ReviewerName,
		Action:          model.ReviewActionReject,
		Decision:        &decision,
		Notes:           req.Notes,
	}

	if err := s.repo.CreateReviewAction(ctx, action); err != nil {
		return fmt.Errorf("failed to create review action: %w", err)
	}

	// Update submission
	now := time.Now()
	submission.Status = model.KYCStatusRejected
	submission.RejectedAt = &now
	submission.ReviewedAt = &now
	submission.RejectionReason = &req.Reason

	if err := s.repo.UpdateSubmission(ctx, submission); err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}

	// Update merchant status
	if err := s.repo.UpdateMerchantKYCStatus(ctx, submission.MerchantID, model.KYCStatusRejected); err != nil {
		return fmt.Errorf("failed to update merchant status: %w", err)
	}

	// Audit log
	s.auditService.LogKYCAction(ctx, &AuditLogRequest{
		SubmissionID: &req.SubmissionID,
		MerchantID:   &submission.MerchantID,
		ActorType:    model.ActorTypeAdmin,
		ActorID:      req.ReviewerEmail,
		Action:       "kyc_rejected",
		ResourceType: "kyc_submission",
		ResourceID:   &req.SubmissionID,
		OldStatus:    string(model.KYCStatusPendingReview),
		NewStatus:    string(model.KYCStatusRejected),
	})

	return nil
}

func (s *kycService) RequestMoreInfo(ctx context.Context, req *RequestMoreInfoRequest) error {
	submission, err := s.repo.GetSubmissionByID(ctx, req.SubmissionID)
	if err != nil {
		return err
	}

	// Convert required documents to JSON
	requiredDocsJSON, _ := json.Marshal(req.RequiredDocuments)
	requiredDocsStr := string(requiredDocsJSON)

	// Create review action
	action := &model.KYCReviewAction{
		KYCSubmissionID: req.SubmissionID,
		MerchantID:      submission.MerchantID,
		ReviewerID:      &req.ReviewerID,
		ReviewerEmail:   &req.ReviewerEmail,
		ReviewerName:    &req.ReviewerName,
		Action:          model.ReviewActionRequestMoreInfo,
		Notes:           req.Notes,
		RequiredDocuments: &requiredDocsStr,
	}

	if err := s.repo.CreateReviewAction(ctx, action); err != nil {
		return fmt.Errorf("failed to create review action: %w", err)
	}

	// Update submission status back to in_progress
	submission.Status = model.KYCStatusInProgress
	submission.RequiresManualReview = true

	if err := s.repo.UpdateSubmission(ctx, submission); err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}

	// Audit log
	s.auditService.LogKYCAction(ctx, &AuditLogRequest{
		SubmissionID: &req.SubmissionID,
		MerchantID:   &submission.MerchantID,
		ActorType:    model.ActorTypeAdmin,
		ActorID:      req.ReviewerEmail,
		Action:       "more_info_requested",
		ResourceType: "kyc_submission",
		ResourceID:   &req.SubmissionID,
		NewStatus:    string(model.KYCStatusInProgress),
	})

	return nil
}

func (s *kycService) ListPendingReviews(ctx context.Context, limit, offset int) ([]*model.KYCSubmission, error) {
	return s.repo.ListSubmissionsByStatus(ctx, model.KYCStatusPendingReview, limit, offset)
}

func (s *kycService) ListByStatus(ctx context.Context, status model.KYCStatus, limit, offset int) ([]*model.KYCSubmission, error) {
	return s.repo.ListSubmissionsByStatus(ctx, status, limit, offset)
}

// ===== Helper Functions =====

func (s *kycService) validateSubmissionFields(req *CreateKYCSubmissionRequest) error {
	switch req.MerchantType {
	case model.MerchantTypeIndividual:
		if req.IndividualFullName == nil || req.IndividualCCCDNumber == nil {
			return ErrMissingRequiredFields
		}
	case model.MerchantTypeHouseholdBusiness:
		if req.HKTBusinessName == nil || req.HKTTaxID == nil || req.HKTOwnerName == nil {
			return ErrMissingRequiredFields
		}
	case model.MerchantTypeCompany:
		if req.CompanyLegalName == nil || req.CompanyRegistrationNumber == nil || req.CompanyDirectorName == nil {
			return ErrMissingRequiredFields
		}
	default:
		return ErrInvalidMerchantType
	}
	return nil
}

func (s *kycService) validateRequiredDocuments(merchantType model.MerchantType, docs []*model.KYCDocument) error {
	docTypes := make(map[model.KYCDocumentType]bool)
	for _, doc := range docs {
		docTypes[doc.DocumentType] = true
	}

	switch merchantType {
	case model.MerchantTypeIndividual:
		if !docTypes[model.DocumentTypeCCCDFront] || !docTypes[model.DocumentTypeCCCDBack] || !docTypes[model.DocumentTypeSelfie] {
			return errors.New("missing required documents: CCCD front, CCCD back, and selfie are required for individuals")
		}
	case model.MerchantTypeHouseholdBusiness:
		if !docTypes[model.DocumentTypeCCCDFront] || !docTypes[model.DocumentTypeCCCDBack] ||
			!docTypes[model.DocumentTypeSelfie] || !docTypes[model.DocumentTypeBusinessLicense] {
			return errors.New("missing required documents for household business")
		}
	case model.MerchantTypeCompany:
		if !docTypes[model.DocumentTypeDirectorID] || !docTypes[model.DocumentTypeBusinessRegistration] {
			return errors.New("missing required documents for company")
		}
	}

	return nil
}

func (s *kycService) getRequiredVerifications(merchantType model.MerchantType) []string {
	base := []string{
		model.VerificationTypeOCR,
		model.VerificationTypeFaceMatch,
		model.VerificationTypeAgeCheck,
		model.VerificationTypeNameMatch,
		model.VerificationTypeSanctionsCheck,
	}

	switch merchantType {
	case model.MerchantTypeHouseholdBusiness:
		return append(base, model.VerificationTypeMSTLookup)
	case model.MerchantTypeCompany:
		return append(base, model.VerificationTypeMSTLookup, model.VerificationTypeBusinessLookup)
	default:
		return base
	}
}

func getDefaultLimits(merchantType model.MerchantType) model.RecommendedLimitsMap {
	switch merchantType {
	case model.MerchantTypeIndividual:
		return model.RecommendedLimitsMap{
			DailyLimitVND:   200000000,  // 200M VND
			MonthlyLimitVND: 3000000000, // 3B VND
		}
	case model.MerchantTypeHouseholdBusiness:
		return model.RecommendedLimitsMap{
			DailyLimitVND:   500000000,   // 500M VND
			MonthlyLimitVND: 10000000000, // 10B VND
		}
	case model.MerchantTypeCompany:
		return model.RecommendedLimitsMap{
			DailyLimitVND:   1000000000,  // 1B VND
			MonthlyLimitVND: 30000000000, // 30B VND
		}
	default:
		return model.RecommendedLimitsMap{
			DailyLimitVND:   200000000,
			MonthlyLimitVND: 3000000000,
		}
	}
}

func buildChangesJSON(verificationResults []*model.KYCVerificationResult, riskAssessment *model.KYCRiskAssessment) *string {
	changes := map[string]interface{}{
		"verification_count": len(verificationResults),
		"risk_level":         riskAssessment.RiskLevel,
		"risk_score":         riskAssessment.RiskScore,
	}

	jsonData, _ := json.Marshal(changes)
	result := string(jsonData)
	return &result
}
