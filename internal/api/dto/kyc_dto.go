package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

// ===== Request DTOs =====

type CreateKYCSubmissionRequest struct {
	MerchantType model.MerchantType `json:"merchant_type" binding:"required"`

	// Individual fields
	IndividualFullName    *string `json:"individual_full_name,omitempty"`
	IndividualCCCDNumber  *string `json:"individual_cccd_number,omitempty"`
	IndividualDateOfBirth *string `json:"individual_date_of_birth,omitempty"` // Format: YYYY-MM-DD
	IndividualAddress     *string `json:"individual_address,omitempty"`

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

type SubmitKYCRequest struct {
	SubmissionID uuid.UUID `json:"submission_id" binding:"required"`
}

type ApproveKYCRequest struct {
	Notes           *string  `json:"notes,omitempty"`
	DailyLimitVND   *float64 `json:"daily_limit_vnd,omitempty"`
	MonthlyLimitVND *float64 `json:"monthly_limit_vnd,omitempty"`
}

type RejectKYCRequest struct {
	Reason string  `json:"reason" binding:"required"`
	Notes  *string `json:"notes,omitempty"`
}

type RequestMoreInfoRequest struct {
	RequiredDocuments []model.KYCDocumentType `json:"required_documents" binding:"required"`
	Notes             *string                 `json:"notes,omitempty"`
}

// ===== Response DTOs =====

type KYCSubmissionResponse struct {
	ID           uuid.UUID          `json:"id"`
	MerchantID   uuid.UUID          `json:"merchant_id"`
	MerchantType model.MerchantType `json:"merchant_type"`
	Status       model.KYCStatus    `json:"status"`

	// Timestamps
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	RejectedAt  *time.Time `json:"rejected_at,omitempty"`

	// Individual fields (masked for privacy)
	IndividualFullName *string `json:"individual_full_name,omitempty"`

	// HKT fields
	HKTBusinessName *string `json:"hkt_business_name,omitempty"`
	HKTTaxID        *string `json:"hkt_tax_id,omitempty"`

	// Company fields
	CompanyLegalName          *string `json:"company_legal_name,omitempty"`
	CompanyRegistrationNumber *string `json:"company_registration_number,omitempty"`

	// Common fields
	ProductCategory *string `json:"product_category,omitempty"`

	// Meta
	AutoApproved         bool    `json:"auto_approved"`
	RequiresManualReview bool    `json:"requires_manual_review"`
	RejectionReason      *string `json:"rejection_reason,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type KYCDocumentResponse struct {
	ID           uuid.UUID              `json:"id"`
	DocumentType model.KYCDocumentType  `json:"document_type"`
	FileName     string                 `json:"file_name"`
	FileSize     *int64                 `json:"file_size,omitempty"`
	Verified     bool                   `json:"verified"`
	UploadedAt   time.Time              `json:"uploaded_at"`
}

type KYCStatusResponse struct {
	Submission        *KYCSubmissionResponse  `json:"submission"`
	Documents         []KYCDocumentResponse   `json:"documents"`
	RequiredDocuments []model.KYCDocumentType `json:"required_documents"`
	CanSubmit         bool                    `json:"can_submit"`
}

type VerificationResultResponse struct {
	VerificationType string                      `json:"verification_type"`
	Status           model.VerificationStatus    `json:"status"`
	Passed           *bool                       `json:"passed,omitempty"`
	ConfidenceScore  *float64                    `json:"confidence_score,omitempty"`
	CompletedAt      *time.Time                  `json:"completed_at,omitempty"`
}

type RiskAssessmentResponse struct {
	RiskLevel          model.RiskLevel        `json:"risk_level"`
	RiskScore          int                    `json:"risk_score"`
	RecommendedAction  model.RecommendedAction `json:"recommended_action"`
	IsHighRiskIndustry bool                   `json:"is_high_risk_industry"`
	SanctionsHit       bool                   `json:"sanctions_hit"`
	AssessedAt         time.Time              `json:"assessed_at"`
}

type KYCDetailResponse struct {
	Submission          *KYCSubmissionResponse       `json:"submission"`
	Documents           []KYCDocumentResponse        `json:"documents"`
	VerificationResults []VerificationResultResponse `json:"verification_results,omitempty"`
	RiskAssessment      *RiskAssessmentResponse      `json:"risk_assessment,omitempty"`
}

// ===== Helper Functions =====

func ToKYCSubmissionResponse(submission *model.KYCSubmission) *KYCSubmissionResponse {
	return &KYCSubmissionResponse{
		ID:                   submission.ID,
		MerchantID:           submission.MerchantID,
		MerchantType:         submission.MerchantType,
		Status:               submission.Status,
		SubmittedAt:          submission.SubmittedAt,
		ReviewedAt:           submission.ReviewedAt,
		ApprovedAt:           submission.ApprovedAt,
		RejectedAt:           submission.RejectedAt,
		IndividualFullName:   submission.IndividualFullName,
		HKTBusinessName:      submission.HKTBusinessName,
		HKTTaxID:             submission.HKTTaxID,
		CompanyLegalName:     submission.CompanyLegalName,
		CompanyRegistrationNumber: submission.CompanyRegistrationNumber,
		ProductCategory:      submission.ProductCategory,
		AutoApproved:         submission.AutoApproved,
		RequiresManualReview: submission.RequiresManualReview,
		RejectionReason:      submission.RejectionReason,
		CreatedAt:            submission.CreatedAt,
		UpdatedAt:            submission.UpdatedAt,
	}
}

func ToKYCDocumentResponse(doc *model.KYCDocument) KYCDocumentResponse {
	return KYCDocumentResponse{
		ID:           doc.ID,
		DocumentType: doc.DocumentType,
		FileName:     doc.FileName,
		FileSize:     doc.FileSizeBytes,
		Verified:     doc.Verified,
		UploadedAt:   doc.UploadedAt,
	}
}

func ToKYCDocumentResponses(docs []*model.KYCDocument) []KYCDocumentResponse {
	responses := make([]KYCDocumentResponse, len(docs))
	for i, doc := range docs {
		responses[i] = ToKYCDocumentResponse(doc)
	}
	return responses
}

func ToVerificationResultResponse(vr *model.KYCVerificationResult) VerificationResultResponse {
	return VerificationResultResponse{
		VerificationType: vr.VerificationType,
		Status:           vr.Status,
		Passed:           vr.Passed,
		ConfidenceScore:  vr.ConfidenceScore,
		CompletedAt:      vr.CompletedAt,
	}
}

func ToVerificationResultResponses(results []*model.KYCVerificationResult) []VerificationResultResponse {
	responses := make([]VerificationResultResponse, len(results))
	for i, vr := range results {
		responses[i] = ToVerificationResultResponse(vr)
	}
	return responses
}

func ToRiskAssessmentResponse(ra *model.KYCRiskAssessment) *RiskAssessmentResponse {
	if ra == nil {
		return nil
	}
	return &RiskAssessmentResponse{
		RiskLevel:          ra.RiskLevel,
		RiskScore:          ra.RiskScore,
		RecommendedAction:  ra.RecommendedAction,
		IsHighRiskIndustry: ra.IsHighRiskIndustry,
		SanctionsHit:       ra.SanctionsHit,
		AssessedAt:         ra.AssessedAt,
	}
}
