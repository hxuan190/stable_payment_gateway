package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hxuan190/stable_payment_gateway/internal/api/dto"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
)

// AdminKYCHandler handles admin KYC operations
type AdminKYCHandler struct {
	kycService service.KYCService
}

// NewAdminKYCHandler creates a new admin KYC handler
func NewAdminKYCHandler(kycService service.KYCService) *AdminKYCHandler {
	return &AdminKYCHandler{
		kycService: kycService,
	}
}

// ListPendingReviews lists all KYC submissions pending manual review
// GET /api/v1/admin/kyc/pending
func (h *AdminKYCHandler) ListPendingReviews(c *gin.Context) {
	// Get pagination params
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	submissions, err := h.kycService.ListPendingReviews(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch pending reviews",
		})
		return
	}

	responses := make([]*dto.KYCSubmissionResponse, len(submissions))
	for i, sub := range submissions {
		responses[i] = dto.ToKYCSubmissionResponse(sub)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"total":  len(responses),
		},
	})
}

// ListByStatus lists KYC submissions by status
// GET /api/v1/admin/kyc/list?status=pending_review
func (h *AdminKYCHandler) ListByStatus(c *gin.Context) {
	statusStr := c.Query("status")
	if statusStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "status query parameter is required",
		})
		return
	}

	status := model.KYCStatus(statusStr)

	// Get pagination params
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	submissions, err := h.kycService.ListByStatus(c.Request.Context(), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch submissions",
		})
		return
	}

	responses := make([]*dto.KYCSubmissionResponse, len(submissions))
	for i, sub := range submissions {
		responses[i] = dto.ToKYCSubmissionResponse(sub)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"total":  len(responses),
		},
	})
}

// GetSubmissionDetail gets detailed KYC information for admin review
// GET /api/v1/admin/kyc/submissions/:id
func (h *AdminKYCHandler) GetSubmissionDetail(c *gin.Context) {
	submissionIDStr := c.Param("id")
	submissionID, err := uuid.Parse(submissionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid submission ID",
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

	// Get documents
	docs, _ := h.kycService.ListDocuments(c.Request.Context(), submissionID)

	response := dto.KYCDetailResponse{
		Submission: dto.ToKYCSubmissionResponse(submission),
		Documents:  dto.ToKYCDocumentResponses(docs),
	}

	// Include verification results and risk assessment
	if submission.VerificationResults != nil {
		response.VerificationResults = dto.ToVerificationResultResponses(submission.VerificationResults)
	}
	if submission.RiskAssessment != nil {
		response.RiskAssessment = dto.ToRiskAssessmentResponse(submission.RiskAssessment)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

// ApproveKYC approves a KYC submission
// POST /api/v1/admin/kyc/submissions/:id/approve
func (h *AdminKYCHandler) ApproveKYC(c *gin.Context) {
	submissionIDStr := c.Param("id")
	submissionID, err := uuid.Parse(submissionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid submission ID",
		})
		return
	}

	var req dto.ApproveKYCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Get admin info from context (set by auth middleware)
	adminID, _ := c.Get("admin_id")
	adminEmail, _ := c.Get("admin_email")
	adminName, _ := c.Get("admin_name")

	serviceReq := &service.ApproveKYCRequest{
		SubmissionID:    submissionID,
		ReviewerID:      adminID.(uuid.UUID),
		ReviewerEmail:   adminEmail.(string),
		ReviewerName:    adminName.(string),
		Notes:           req.Notes,
		DailyLimitVND:   req.DailyLimitVND,
		MonthlyLimitVND: req.MonthlyLimitVND,
	}

	if err := h.kycService.ApproveKYC(c.Request.Context(), serviceReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to approve KYC",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "KYC approved successfully",
	})
}

// RejectKYC rejects a KYC submission
// POST /api/v1/admin/kyc/submissions/:id/reject
func (h *AdminKYCHandler) RejectKYC(c *gin.Context) {
	submissionIDStr := c.Param("id")
	submissionID, err := uuid.Parse(submissionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid submission ID",
		})
		return
	}

	var req dto.RejectKYCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Get admin info from context
	adminID, _ := c.Get("admin_id")
	adminEmail, _ := c.Get("admin_email")
	adminName, _ := c.Get("admin_name")

	serviceReq := &service.RejectKYCRequest{
		SubmissionID:  submissionID,
		ReviewerID:    adminID.(uuid.UUID),
		ReviewerEmail: adminEmail.(string),
		ReviewerName:  adminName.(string),
		Reason:        req.Reason,
		Notes:         req.Notes,
	}

	if err := h.kycService.RejectKYC(c.Request.Context(), serviceReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to reject KYC",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "KYC rejected successfully",
	})
}

// RequestMoreInfo requests additional information/documents from merchant
// POST /api/v1/admin/kyc/submissions/:id/request-info
func (h *AdminKYCHandler) RequestMoreInfo(c *gin.Context) {
	submissionIDStr := c.Param("id")
	submissionID, err := uuid.Parse(submissionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid submission ID",
		})
		return
	}

	var req dto.RequestMoreInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Get admin info from context
	adminID, _ := c.Get("admin_id")
	adminEmail, _ := c.Get("admin_email")
	adminName, _ := c.Get("admin_name")

	serviceReq := &service.RequestMoreInfoRequest{
		SubmissionID:      submissionID,
		ReviewerID:        adminID.(uuid.UUID),
		ReviewerEmail:     adminEmail.(string),
		ReviewerName:      adminName.(string),
		RequiredDocuments: req.RequiredDocuments,
		Notes:             req.Notes,
	}

	if err := h.kycService.RequestMoreInfo(c.Request.Context(), serviceReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to request more information",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Additional information requested successfully",
	})
}

// GetDocumentURL gets a signed URL for viewing a document
// GET /api/v1/admin/kyc/documents/:id/url
func (h *AdminKYCHandler) GetDocumentURL(c *gin.Context) {
	documentIDStr := c.Param("id")
	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid document ID",
		})
		return
	}

	doc, err := h.kycService.GetDocument(c.Request.Context(), documentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Document not found",
		})
		return
	}

	// TODO: Generate signed URL from storage service
	// For now, return placeholder
	url := doc.FilePath

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"url":        url,
			"expires_in": 3600, // 1 hour
		},
	})
}

// GetKYCStats gets statistics for KYC submissions
// GET /api/v1/admin/kyc/stats
func (h *AdminKYCHandler) GetKYCStats(c *gin.Context) {
	ctx := c.Request.Context()

	// Get counts by status
	// TODO: Implement count methods in repository
	// For now, return placeholder data
	stats := gin.H{
		"pending_review": 0,
		"approved":       0,
		"rejected":       0,
		"in_progress":    0,
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}
