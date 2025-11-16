package handler

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hxuan190/stable_payment_gateway/internal/api/dto"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
)

// KYCHandler handles KYC-related HTTP requests
type KYCHandler struct {
	kycService service.KYCService
}

// NewKYCHandler creates a new KYC handler
func NewKYCHandler(kycService service.KYCService) *KYCHandler {
	return &KYCHandler{
		kycService: kycService,
	}
}

// CreateSubmission creates a new KYC submission
// POST /api/v1/kyc/submissions
func (h *KYCHandler) CreateSubmission(c *gin.Context) {
	var req dto.CreateKYCSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Get merchant ID from context (set by auth middleware)
	merchantID, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Merchant not authenticated",
		})
		return
	}

	// Parse date of birth if provided
	var dob *time.Time
	if req.IndividualDateOfBirth != nil {
		parsed, err := time.Parse("2006-01-02", *req.IndividualDateOfBirth)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid date format for date_of_birth. Expected YYYY-MM-DD",
			})
			return
		}
		dob = &parsed
	}

	// Create service request
	serviceReq := &service.CreateKYCSubmissionRequest{
		MerchantID:                  merchantID.(uuid.UUID),
		MerchantType:                req.MerchantType,
		IndividualFullName:          req.IndividualFullName,
		IndividualCCCDNumber:        req.IndividualCCCDNumber,
		IndividualDateOfBirth:       dob,
		IndividualAddress:           req.IndividualAddress,
		HKTBusinessName:             req.HKTBusinessName,
		HKTTaxID:                    req.HKTTaxID,
		HKTBusinessAddress:          req.HKTBusinessAddress,
		HKTOwnerName:                req.HKTOwnerName,
		HKTOwnerCCCD:                req.HKTOwnerCCCD,
		CompanyLegalName:            req.CompanyLegalName,
		CompanyRegistrationNumber:   req.CompanyRegistrationNumber,
		CompanyTaxID:                req.CompanyTaxID,
		CompanyHeadquartersAddress:  req.CompanyHeadquartersAddress,
		CompanyWebsite:              req.CompanyWebsite,
		CompanyDirectorName:         req.CompanyDirectorName,
		CompanyDirectorCCCD:         req.CompanyDirectorCCCD,
		BusinessDescription:         req.BusinessDescription,
		BusinessWebsite:             req.BusinessWebsite,
		BusinessFacebook:            req.BusinessFacebook,
		ProductCategory:             req.ProductCategory,
	}

	submission, err := h.kycService.CreateSubmission(c.Request.Context(), serviceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create KYC submission",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": dto.ToKYCSubmissionResponse(submission),
	})
}

// GetSubmission gets the current KYC submission for the merchant
// GET /api/v1/kyc/submissions/current
func (h *KYCHandler) GetSubmission(c *gin.Context) {
	merchantID, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Merchant not authenticated",
		})
		return
	}

	submission, err := h.kycService.GetSubmissionByMerchant(c.Request.Context(), merchantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No KYC submission found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": dto.ToKYCSubmissionResponse(submission),
	})
}

// GetSubmissionStatus gets detailed KYC status including documents
// GET /api/v1/kyc/status
func (h *KYCHandler) GetSubmissionStatus(c *gin.Context) {
	merchantID, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Merchant not authenticated",
		})
		return
	}

	submission, err := h.kycService.GetSubmissionByMerchant(c.Request.Context(), merchantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No KYC submission found",
		})
		return
	}

	// Get documents
	docs, err := h.kycService.ListDocuments(c.Request.Context(), submission.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch documents",
		})
		return
	}

	// Determine required documents
	requiredDocs := getRequiredDocuments(submission.MerchantType)

	// Check if can submit
	canSubmit := submission.Status == model.KYCStatusInProgress && hasAllRequiredDocuments(docs, requiredDocs)

	response := dto.KYCStatusResponse{
		Submission:        dto.ToKYCSubmissionResponse(submission),
		Documents:         dto.ToKYCDocumentResponses(docs),
		RequiredDocuments: requiredDocs,
		CanSubmit:         canSubmit,
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

// UploadDocument uploads a KYC document
// POST /api/v1/kyc/documents
func (h *KYCHandler) UploadDocument(c *gin.Context) {
	merchantID, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Merchant not authenticated",
		})
		return
	}

	// Get submission
	submission, err := h.kycService.GetSubmissionByMerchant(c.Request.Context(), merchantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No active KYC submission found. Please create a submission first.",
		})
		return
	}

	// Parse multipart form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File is required",
		})
		return
	}
	defer file.Close()

	// Get document type
	docTypeStr := c.PostForm("document_type")
	if docTypeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "document_type is required",
		})
		return
	}
	docType := model.KYCDocumentType(docTypeStr)

	// Validate document type
	if !isValidDocumentType(docType) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid document type",
		})
		return
	}

	// Read file data
	fileData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read file",
		})
		return
	}

	// Validate file size (max 10MB)
	if len(fileData) > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File size exceeds 10MB limit",
		})
		return
	}

	// Upload document
	uploadReq := &service.UploadDocumentRequest{
		SubmissionID: submission.ID,
		MerchantID:   merchantID.(uuid.UUID),
		DocumentType: docType,
		FileName:     header.Filename,
		FileData:     fileData,
		MimeType:     header.Header.Get("Content-Type"),
	}

	doc, err := h.kycService.UploadDocument(c.Request.Context(), uploadReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to upload document",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": dto.ToKYCDocumentResponse(doc),
	})
}

// SubmitForReview submits the KYC for review
// POST /api/v1/kyc/submit
func (h *KYCHandler) SubmitForReview(c *gin.Context) {
	merchantID, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Merchant not authenticated",
		})
		return
	}

	// Get submission
	submission, err := h.kycService.GetSubmissionByMerchant(c.Request.Context(), merchantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No active KYC submission found",
		})
		return
	}

	// Submit for review
	if err := h.kycService.SubmitForReview(c.Request.Context(), submission.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to submit KYC for review",
			"details": err.Error(),
		})
		return
	}

	// Fetch updated submission
	updatedSubmission, err := h.kycService.GetSubmission(c.Request.Context(), submission.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "KYC submitted but failed to fetch updated status",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": dto.ToKYCSubmissionResponse(updatedSubmission),
		"message": getSubmissionMessage(updatedSubmission.Status),
	})
}

// GetDetail gets detailed KYC information including verification results
// GET /api/v1/kyc/detail/:submission_id
func (h *KYCHandler) GetDetail(c *gin.Context) {
	submissionIDStr := c.Param("submission_id")
	submissionID, err := uuid.Parse(submissionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid submission ID",
		})
		return
	}

	// TODO: Verify merchant owns this submission
	merchantID, exists := c.Get("merchant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Merchant not authenticated",
		})
		return
	}

	submission, err := h.kycService.GetSubmission(c.Request.Context(), submissionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "KYC submission not found",
		})
		return
	}

	// Verify ownership
	if submission.MerchantID != merchantID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	// Get documents
	docs, _ := h.kycService.ListDocuments(c.Request.Context(), submissionID)

	response := dto.KYCDetailResponse{
		Submission: dto.ToKYCSubmissionResponse(submission),
		Documents:  dto.ToKYCDocumentResponses(docs),
	}

	// Include verification results and risk assessment if submitted
	if submission.IsSubmitted() {
		if submission.VerificationResults != nil {
			response.VerificationResults = dto.ToVerificationResultResponses(submission.VerificationResults)
		}
		if submission.RiskAssessment != nil {
			response.RiskAssessment = dto.ToRiskAssessmentResponse(submission.RiskAssessment)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

// ===== Helper Functions =====

func getRequiredDocuments(merchantType model.MerchantType) []model.KYCDocumentType {
	switch merchantType {
	case model.MerchantTypeIndividual:
		return []model.KYCDocumentType{
			model.DocumentTypeCCCDFront,
			model.DocumentTypeCCCDBack,
			model.DocumentTypeSelfie,
		}
	case model.MerchantTypeHouseholdBusiness:
		return []model.KYCDocumentType{
			model.DocumentTypeCCCDFront,
			model.DocumentTypeCCCDBack,
			model.DocumentTypeSelfie,
			model.DocumentTypeBusinessLicense,
			model.DocumentTypeShopPhoto,
		}
	case model.MerchantTypeCompany:
		return []model.KYCDocumentType{
			model.DocumentTypeDirectorID,
			model.DocumentTypeBusinessRegistration,
			model.DocumentTypeBusinessCharter,
		}
	default:
		return []model.KYCDocumentType{}
	}
}

func hasAllRequiredDocuments(docs []*model.KYCDocument, required []model.KYCDocumentType) bool {
	docTypes := make(map[model.KYCDocumentType]bool)
	for _, doc := range docs {
		docTypes[doc.DocumentType] = true
	}

	for _, reqType := range required {
		if !docTypes[reqType] {
			return false
		}
	}

	return true
}

func isValidDocumentType(docType model.KYCDocumentType) bool {
	validTypes := []model.KYCDocumentType{
		model.DocumentTypeCCCDFront,
		model.DocumentTypeCCCDBack,
		model.DocumentTypeSelfie,
		model.DocumentTypeBusinessLicense,
		model.DocumentTypeBusinessRegistration,
		model.DocumentTypeBusinessCharter,
		model.DocumentTypeDirectorID,
		model.DocumentTypeAppointmentDecision,
		model.DocumentTypeShopPhoto,
		model.DocumentTypeOther,
	}

	for _, vt := range validTypes {
		if docType == vt {
			return true
		}
	}

	return false
}

func getSubmissionMessage(status model.KYCStatus) string {
	switch status {
	case model.KYCStatusApproved:
		return "Congratulations! Your KYC has been automatically approved. You can now start accepting payments."
	case model.KYCStatusPendingReview:
		return "Your KYC submission is under manual review. We'll notify you once it's complete. This typically takes 1-2 business days."
	case model.KYCStatusRejected:
		return "Your KYC submission has been rejected. Please check the rejection reason and resubmit with correct information."
	default:
		return "KYC submission received and is being processed."
	}
}
