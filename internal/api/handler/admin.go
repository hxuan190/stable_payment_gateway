package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/api/dto"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
	"github.com/shopspring/decimal"
)

// AdminHandler handles HTTP requests for admin operations
type AdminHandler struct {
	merchantService   *service.MerchantService
	payoutService     *service.PayoutService
	complianceService service.ComplianceService
	merchantRepo      *repository.MerchantRepository
	payoutRepo        *repository.PayoutRepository
	paymentRepo       *repository.PaymentRepository
	balanceRepo       *repository.BalanceRepository
	travelRuleRepo    repository.TravelRuleRepository
	kycDocumentRepo   repository.KYCDocumentRepository
	solanaWallet      WalletBalanceGetter
}

// WalletBalanceGetter is an interface for getting wallet balance
type WalletBalanceGetter interface {
	GetBalance() (decimal.Decimal, error)
}

// NewAdminHandler creates a new admin handler instance
func NewAdminHandler(
	merchantService *service.MerchantService,
	payoutService *service.PayoutService,
	complianceService service.ComplianceService,
	merchantRepo *repository.MerchantRepository,
	payoutRepo *repository.PayoutRepository,
	paymentRepo *repository.PaymentRepository,
	balanceRepo *repository.BalanceRepository,
	travelRuleRepo repository.TravelRuleRepository,
	kycDocumentRepo repository.KYCDocumentRepository,
	solanaWallet WalletBalanceGetter,
) *AdminHandler {
	return &AdminHandler{
		merchantService:   merchantService,
		payoutService:     payoutService,
		complianceService: complianceService,
		merchantRepo:      merchantRepo,
		payoutRepo:        payoutRepo,
		paymentRepo:       paymentRepo,
		balanceRepo:       balanceRepo,
		travelRuleRepo:    travelRuleRepo,
		kycDocumentRepo:   kycDocumentRepo,
		solanaWallet:      solanaWallet,
	}
}

// ==================== Merchant Management ====================

// ListMerchants lists all merchants with optional filtering by KYC status
// GET /api/admin/merchants
func (h *AdminHandler) ListMerchants(c *gin.Context) {
	var query dto.ListMerchantsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
			"INVALID_QUERY",
			"Invalid query parameters",
			fmt.Sprintf("%v", err),
		))
		return
	}

	// Set default values
	if query.Limit == 0 {
		query.Limit = 20
	}

	var merchants []*model.Merchant
	var err error

	// Filter by KYC status if specified
	if query.KYCStatus != "" {
		merchants, err = h.merchantRepo.ListByKYCStatus(query.KYCStatus, query.Limit, query.Offset)
	} else {
		merchants, err = h.merchantRepo.List(query.Limit, query.Offset)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"FAILED_TO_LIST_MERCHANTS",
			"Failed to retrieve merchants",
		))
		return
	}

	// Convert to list items
	items := make([]*dto.MerchantListItem, len(merchants))
	for i, merchant := range merchants {
		items[i] = dto.MerchantToListItem(merchant)
	}

	// Build response
	response := dto.APIResponse{
		Data: dto.ListMerchantsResponse{
			Merchants: items,
			Total:     len(items),
			Limit:     query.Limit,
			Offset:    query.Offset,
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetMerchant retrieves detailed information about a specific merchant
// GET /api/admin/merchants/:id
func (h *AdminHandler) GetMerchant(c *gin.Context) {
	merchantID := c.Param("id")
	if merchantID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_MERCHANT_ID",
			"Merchant ID is required",
		))
		return
	}

	// Get merchant
	merchant, err := h.merchantRepo.GetByID(merchantID)
	if err != nil {
		if err == repository.ErrMerchantNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(
				"MERCHANT_NOT_FOUND",
				"Merchant not found",
			))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"FAILED_TO_GET_MERCHANT",
			"Failed to retrieve merchant details",
		))
		return
	}

	// Get merchant balance
	balance, err := h.balanceRepo.GetByMerchantID(merchantID)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"FAILED_TO_GET_BALANCE",
			"Failed to retrieve merchant balance",
		))
		return
	}

	// Build response
	response := dto.APIResponse{
		Data:      dto.MerchantToDetailResponse(merchant, balance),
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// ApproveKYC approves a merchant's KYC application
// POST /api/admin/merchants/:id/kyc/approve
func (h *AdminHandler) ApproveKYC(c *gin.Context) {
	merchantID := c.Param("id")
	if merchantID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_MERCHANT_ID",
			"Merchant ID is required",
		))
		return
	}

	var req dto.ApproveKYCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
			"INVALID_REQUEST",
			"Invalid request body",
			fmt.Sprintf("%v", err),
		))
		return
	}

	// Approve KYC
	merchant, err := h.merchantService.ApproveKYC(merchantID, req.ApprovedBy)
	if err != nil {
		if errors.Is(err, service.ErrMerchantNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(
				"MERCHANT_NOT_FOUND",
				"Merchant not found",
			))
			return
		}
		if errors.Is(err, service.ErrInvalidKYCStatus) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(
				"INVALID_KYC_STATUS",
				err.Error(),
			))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"KYC_APPROVAL_FAILED",
			"Failed to approve merchant KYC",
		))
		return
	}

	// Build response
	response := dto.APIResponse{
		Data: gin.H{
			"merchant_id": merchant.ID,
			"api_key":     merchant.APIKey,
			"status":      string(merchant.KYCStatus),
			"approved_at": merchant.KYCApprovedAt,
			"approved_by": merchant.KYCApprovedBy,
			"message":     "KYC approved successfully. API key has been generated.",
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// RejectKYC rejects a merchant's KYC application
// POST /api/admin/merchants/:id/kyc/reject
func (h *AdminHandler) RejectKYC(c *gin.Context) {
	merchantID := c.Param("id")
	if merchantID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_MERCHANT_ID",
			"Merchant ID is required",
		))
		return
	}

	var req dto.RejectKYCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
			"INVALID_REQUEST",
			"Invalid request body",
			fmt.Sprintf("%v", err),
		))
		return
	}

	// Reject KYC
	merchant, err := h.merchantService.RejectKYC(merchantID, req.Reason)
	if err != nil {
		if errors.Is(err, service.ErrMerchantNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(
				"MERCHANT_NOT_FOUND",
				"Merchant not found",
			))
			return
		}
		if errors.Is(err, service.ErrInvalidKYCStatus) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(
				"INVALID_KYC_STATUS",
				err.Error(),
			))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"KYC_REJECTION_FAILED",
			"Failed to reject merchant KYC",
		))
		return
	}

	// Build response
	response := dto.APIResponse{
		Data: gin.H{
			"merchant_id":      merchant.ID,
			"status":           string(merchant.KYCStatus),
			"rejection_reason": merchant.KYCRejectionReason,
			"message":          "KYC rejected successfully",
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// ==================== Payout Management ====================

// ListPayouts lists all payouts with optional filtering by status
// GET /api/admin/payouts
func (h *AdminHandler) ListPayouts(c *gin.Context) {
	var query dto.ListPayoutsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
			"INVALID_QUERY",
			"Invalid query parameters",
			fmt.Sprintf("%v", err),
		))
		return
	}

	// Set default values
	if query.Limit == 0 {
		query.Limit = 20
	}

	var payouts []*model.Payout
	var err error

	// Filter by status and merchant
	if query.MerchantID != "" {
		payouts, err = h.payoutRepo.ListByMerchant(query.MerchantID, query.Limit, query.Offset)
	} else if query.Status != "" {
		payouts, err = h.payoutRepo.ListByStatus(model.PayoutStatus(query.Status), query.Limit, query.Offset)
	} else {
		// Get all payouts with pagination
		payouts, err = h.payoutRepo.ListByStatus("", query.Limit, query.Offset)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"FAILED_TO_LIST_PAYOUTS",
			"Failed to retrieve payouts",
		))
		return
	}

	// Build list items with merchant names
	items := make([]*dto.AdminPayoutListItem, len(payouts))
	for i, payout := range payouts {
		// Get merchant name
		merchant, err := h.merchantRepo.GetByID(payout.MerchantID)
		merchantName := "Unknown"
		if err == nil {
			merchantName = merchant.BusinessName
		}
		items[i] = dto.AdminPayoutToListItem(payout, merchantName)
	}

	// Build response
	response := dto.APIResponse{
		Data: dto.AdminListPayoutsResponse{
			Payouts: items,
			Total:   len(items),
			Limit:   query.Limit,
			Offset:  query.Offset,
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetPayout retrieves detailed information about a specific payout
// GET /api/admin/payouts/:id
func (h *AdminHandler) GetPayout(c *gin.Context) {
	payoutID := c.Param("id")
	if payoutID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_PAYOUT_ID",
			"Payout ID is required",
		))
		return
	}

	// Get payout
	payout, err := h.payoutRepo.GetByID(payoutID)
	if err != nil {
		if err == repository.ErrPayoutNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(
				"PAYOUT_NOT_FOUND",
				"Payout not found",
			))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"FAILED_TO_GET_PAYOUT",
			"Failed to retrieve payout details",
		))
		return
	}

	// Get merchant name
	merchant, err := h.merchantRepo.GetByID(payout.MerchantID)
	merchantName := "Unknown"
	if err == nil {
		merchantName = merchant.BusinessName
	}

	// Build response
	response := dto.APIResponse{
		Data:      dto.PayoutToDetailResponse(payout, merchantName),
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// ApprovePayout approves a payout request
// POST /api/admin/payouts/:id/approve
func (h *AdminHandler) ApprovePayout(c *gin.Context) {
	payoutID := c.Param("id")
	if payoutID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_PAYOUT_ID",
			"Payout ID is required",
		))
		return
	}

	var req dto.ApprovePayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
			"INVALID_REQUEST",
			"Invalid request body",
			fmt.Sprintf("%v", err),
		))
		return
	}

	// Approve payout
	if err := h.payoutService.ApprovePayout(payoutID, req.ApprovedBy); err != nil {
		if errors.Is(err, service.ErrPayoutNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(
				"PAYOUT_NOT_FOUND",
				"Payout not found",
			))
			return
		}
		if errors.Is(err, service.ErrPayoutCannotBeApproved) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(
				"PAYOUT_CANNOT_BE_APPROVED",
				err.Error(),
			))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"PAYOUT_APPROVAL_FAILED",
			"Failed to approve payout",
		))
		return
	}

	// Get updated payout
	payout, _ := h.payoutRepo.GetByID(payoutID)

	// Build response
	response := dto.APIResponse{
		Data: gin.H{
			"payout_id":   payoutID,
			"status":      string(payout.Status),
			"approved_at": payout.ApprovedAt,
			"approved_by": payout.ApprovedBy,
			"message":     "Payout approved successfully",
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// RejectPayout rejects a payout request
// POST /api/admin/payouts/:id/reject
func (h *AdminHandler) RejectPayout(c *gin.Context) {
	payoutID := c.Param("id")
	if payoutID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_PAYOUT_ID",
			"Payout ID is required",
		))
		return
	}

	var req dto.RejectPayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
			"INVALID_REQUEST",
			"Invalid request body",
			fmt.Sprintf("%v", err),
		))
		return
	}

	// Reject payout
	if err := h.payoutService.RejectPayout(payoutID, req.Reason); err != nil {
		if errors.Is(err, service.ErrPayoutNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(
				"PAYOUT_NOT_FOUND",
				"Payout not found",
			))
			return
		}
		if errors.Is(err, service.ErrPayoutCannotBeRejected) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(
				"PAYOUT_CANNOT_BE_REJECTED",
				err.Error(),
			))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"PAYOUT_REJECTION_FAILED",
			"Failed to reject payout",
		))
		return
	}

	// Get updated payout
	payout, _ := h.payoutRepo.GetByID(payoutID)

	// Build response
	response := dto.APIResponse{
		Data: gin.H{
			"payout_id":        payoutID,
			"status":           string(payout.Status),
			"rejection_reason": payout.RejectionReason,
			"message":          "Payout rejected successfully",
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// CompletePayout marks a payout as completed after bank transfer
// POST /api/admin/payouts/:id/complete
func (h *AdminHandler) CompletePayout(c *gin.Context) {
	payoutID := c.Param("id")
	if payoutID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_PAYOUT_ID",
			"Payout ID is required",
		))
		return
	}

	var req dto.CompletePayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
			"INVALID_REQUEST",
			"Invalid request body",
			fmt.Sprintf("%v", err),
		))
		return
	}

	// Complete payout
	if err := h.payoutService.CompletePayout(payoutID, req.BankReferenceNumber, req.ProcessedBy); err != nil {
		if errors.Is(err, service.ErrPayoutNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse(
				"PAYOUT_NOT_FOUND",
				"Payout not found",
			))
			return
		}
		if errors.Is(err, service.ErrPayoutInvalidStatus) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(
				"PAYOUT_CANNOT_BE_COMPLETED",
				err.Error(),
			))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"PAYOUT_COMPLETION_FAILED",
			"Failed to complete payout",
		))
		return
	}

	// Get updated payout
	payout, _ := h.payoutRepo.GetByID(payoutID)

	// Build response
	response := dto.APIResponse{
		Data: gin.H{
			"payout_id":             payoutID,
			"status":                string(payout.Status),
			"completed_at":          payout.CompletionDate,
			"bank_reference_number": payout.BankReferenceNumber,
			"processed_by":          payout.ProcessedBy,
			"message":               "Payout completed successfully",
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// ==================== System Statistics ====================

// GetStats retrieves overall system statistics
// GET /api/admin/stats
func (h *AdminHandler) GetStats(c *gin.Context) {
	// Get total merchants
	allMerchants, err := h.merchantRepo.List(1000, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"FAILED_TO_GET_STATS",
			"Failed to retrieve system statistics",
		))
		return
	}

	// Count merchants by status
	totalMerchants := len(allMerchants)
	activeMerchants := 0
	pendingKYCMerchants := 0

	for _, m := range allMerchants {
		if m.KYCStatus == model.KYCStatusApproved {
			activeMerchants++
		} else if m.KYCStatus == model.KYCStatusPending {
			pendingKYCMerchants++
		}
	}

	// Get total payments count and volume
	// Note: This is a simplified implementation. In production, you'd want to
	// add specific repository methods for aggregations
	totalPayments := 0
	totalPaymentsVolume := decimal.Zero

	// Get total payouts
	allPayouts, _ := h.payoutRepo.ListByStatus("", 1000, 0)
	totalPayouts := len(allPayouts)
	totalPayoutsVolume := decimal.Zero
	pendingPayouts := 0

	for _, p := range allPayouts {
		totalPayoutsVolume = totalPayoutsVolume.Add(p.AmountVND)
		if p.Status == model.PayoutStatusRequested || p.Status == model.PayoutStatusApproved {
			pendingPayouts++
		}
	}

	// Get hot wallet balance
	hotWalletBalance := decimal.Zero
	if h.solanaWallet != nil {
		balance, err := h.solanaWallet.GetBalance()
		if err == nil {
			hotWalletBalance = balance
		}
	}

	// Build response
	response := dto.APIResponse{
		Data: dto.SystemStatsResponse{
			TotalMerchants:      totalMerchants,
			ActiveMerchants:     activeMerchants,
			PendingKYCMerchants: pendingKYCMerchants,
			TotalPayments:       totalPayments,
			TotalPaymentsVolume: totalPaymentsVolume,
			TotalPayouts:        totalPayouts,
			TotalPayoutsVolume:  totalPayoutsVolume,
			PendingPayouts:      pendingPayouts,
			HotWalletBalance:    hotWalletBalance,
			Platform:            "solana",
			Timestamp:           time.Now(),
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetDailyStats retrieves daily statistics for a date range
// GET /api/admin/stats/daily
func (h *AdminHandler) GetDailyStats(c *gin.Context) {
	var query dto.DailyStatsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
			"INVALID_QUERY",
			"Invalid query parameters",
			fmt.Sprintf("%v", err),
		))
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", query.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_START_DATE",
			"Start date must be in format YYYY-MM-DD",
		))
		return
	}

	endDate, err := time.Parse("2006-01-02", query.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_END_DATE",
			"End date must be in format YYYY-MM-DD",
		))
		return
	}

	// Note: This is a simplified implementation
	// In production, you would add repository methods that use SQL aggregations
	// for better performance
	stats := make([]*dto.DailyStatsItem, 0)

	// Calculate total days
	totalDays := int(endDate.Sub(startDate).Hours()/24) + 1

	// Build response
	response := dto.APIResponse{
		Data: dto.DailyStatsResponse{
			Stats:     stats,
			TotalDays: totalDays,
			StartDate: query.StartDate,
			EndDate:   query.EndDate,
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// ==================== Compliance Management ====================

// GetTravelRuleData retrieves Travel Rule data with filtering
// GET /admin/v1/compliance/travel-rule
func (h *AdminHandler) GetTravelRuleData(c *gin.Context) {
	var query dto.GetTravelRuleDataQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
			"INVALID_QUERY",
			"Invalid query parameters",
			fmt.Sprintf("%v", err),
		))
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", query.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_START_DATE",
			"Start date must be in format YYYY-MM-DD",
		))
		return
	}

	endDate, err := time.Parse("2006-01-02", query.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_END_DATE",
			"End date must be in format YYYY-MM-DD",
		))
		return
	}

	// Set defaults
	if query.Limit == 0 {
		query.Limit = 20
	}

	// Build filter
	filter := repository.TravelRuleFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	if query.Country != "" {
		filter.PayerCountry = query.Country
	}

	// Get Travel Rule data
	ctx := c.Request.Context()
	travelRuleData, err := h.travelRuleRepo.List(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"FAILED_TO_GET_TRAVEL_RULE_DATA",
			"Failed to retrieve Travel Rule data",
		))
		return
	}

	// Apply pagination (simple in-memory for now)
	total := len(travelRuleData)
	start := query.Offset
	end := query.Offset + query.Limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	paginatedData := travelRuleData[start:end]

	// Convert to DTOs
	items := make([]dto.TravelRuleDataItem, len(paginatedData))
	for i, data := range paginatedData {
		items[i] = dto.TravelRuleDataToListItem(data)
	}

	// Build response
	response := dto.APIResponse{
		Data: dto.GetTravelRuleDataResponse{
			Data:   items,
			Total:  total,
			Limit:  query.Limit,
			Offset: query.Offset,
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetPendingKYCDocuments retrieves all pending KYC documents across all merchants
// GET /admin/v1/compliance/kyc-documents/pending
func (h *AdminHandler) GetPendingKYCDocuments(c *gin.Context) {
	ctx := c.Request.Context()

	// Get pending documents
	documents, err := h.kycDocumentRepo.ListByStatus(ctx, model.KYCDocumentStatusPending)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"FAILED_TO_GET_DOCUMENTS",
			"Failed to retrieve pending KYC documents",
		))
		return
	}

	// Convert to DTOs
	items := make([]dto.KYCDocumentItem, len(documents))
	for i, doc := range documents {
		items[i] = dto.KYCDocumentToListItem(doc)
	}

	response := dto.APIResponse{
		Data: dto.ListKYCDocumentsResponse{
			Documents: items,
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// ApproveKYCDocument approves a KYC document
// POST /admin/v1/compliance/kyc-documents/:id/approve
func (h *AdminHandler) ApproveKYCDocument(c *gin.Context) {
	ctx := c.Request.Context()

	documentIDStr := c.Param("id")
	documentID, err := parseUUID(documentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_DOCUMENT_ID",
			"Invalid document ID format",
		))
		return
	}

	var req dto.ApproveKYCDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
			"INVALID_REQUEST",
			"Invalid request body",
			fmt.Sprintf("%v", err),
		))
		return
	}

	// Get document
	doc, err := h.kycDocumentRepo.GetByID(ctx, documentID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse(
			"DOCUMENT_NOT_FOUND",
			"KYC document not found",
		))
		return
	}

	// Check if already approved
	if doc.Status == model.KYCDocumentStatusApproved {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"DOCUMENT_ALREADY_APPROVED",
			"Document is already approved",
		))
		return
	}

	// Update document status
	doc.Status = model.KYCDocumentStatusApproved
	doc.ReviewedBy.String = req.ApprovedBy
	doc.ReviewedBy.Valid = true
	doc.ReviewedAt.Time = time.Now()
	doc.ReviewedAt.Valid = true

	if req.Notes != "" {
		doc.ReviewerNotes.String = req.Notes
		doc.ReviewerNotes.Valid = true
	}

	err = h.kycDocumentRepo.Update(ctx, doc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"APPROVAL_FAILED",
			"Failed to approve document",
		))
		return
	}

	response := dto.APIResponse{
		Data: gin.H{
			"document_id": documentIDStr,
			"status":      string(doc.Status),
			"reviewed_by": req.ApprovedBy,
			"reviewed_at": doc.ReviewedAt.Time,
			"message":     "KYC document approved successfully",
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// RejectKYCDocument rejects a KYC document
// POST /admin/v1/compliance/kyc-documents/:id/reject
func (h *AdminHandler) RejectKYCDocument(c *gin.Context) {
	ctx := c.Request.Context()

	documentIDStr := c.Param("id")
	documentID, err := parseUUID(documentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_DOCUMENT_ID",
			"Invalid document ID format",
		))
		return
	}

	var req dto.RejectKYCDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
			"INVALID_REQUEST",
			"Invalid request body",
			fmt.Sprintf("%v", err),
		))
		return
	}

	// Get document
	doc, err := h.kycDocumentRepo.GetByID(ctx, documentID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse(
			"DOCUMENT_NOT_FOUND",
			"KYC document not found",
		))
		return
	}

	// Update document status
	doc.Status = model.KYCDocumentStatusRejected
	doc.ReviewedBy.String = req.RejectedBy
	doc.ReviewedBy.Valid = true
	doc.ReviewedAt.Time = time.Now()
	doc.ReviewedAt.Valid = true
	doc.ReviewerNotes.String = req.Reason
	doc.ReviewerNotes.Valid = true

	err = h.kycDocumentRepo.Update(ctx, doc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"REJECTION_FAILED",
			"Failed to reject document",
		))
		return
	}

	response := dto.APIResponse{
		Data: gin.H{
			"document_id":      documentIDStr,
			"status":           string(doc.Status),
			"reviewed_by":      req.RejectedBy,
			"reviewed_at":      doc.ReviewedAt.Time,
			"rejection_reason": req.Reason,
			"message":          "KYC document rejected successfully",
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// UpgradeMerchantTier upgrades a merchant's KYC tier
// POST /admin/v1/compliance/merchants/:id/upgrade-tier
func (h *AdminHandler) UpgradeMerchantTier(c *gin.Context) {
	ctx := c.Request.Context()

	merchantID := c.Param("id")
	if merchantID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_MERCHANT_ID",
			"Merchant ID is required",
		))
		return
	}

	var req dto.UpgradeMerchantTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
			"INVALID_REQUEST",
			"Invalid request body",
			fmt.Sprintf("%v", err),
		))
		return
	}

	// Upgrade tier using compliance service
	err := h.complianceService.UpgradeKYCTier(ctx, merchantID, req.Tier)
	if err != nil {
		if errors.Is(err, service.ErrInsufficientKYCDocuments) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(
				"INSUFFICIENT_KYC_DOCUMENTS",
				err.Error(),
			))
			return
		}
		if errors.Is(err, service.ErrKYCTierUpgradeNotAllowed) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse(
				"TIER_UPGRADE_NOT_ALLOWED",
				err.Error(),
			))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"TIER_UPGRADE_FAILED",
			"Failed to upgrade merchant tier",
		))
		return
	}

	// Get updated merchant
	merchant, _ := h.merchantRepo.GetByID(merchantID)

	response := dto.APIResponse{
		Data: gin.H{
			"merchant_id":       merchantID,
			"kyc_tier":          merchant.KYCTier,
			"monthly_limit_usd": merchant.MonthlyLimitUSD,
			"message":           "Merchant tier upgraded successfully",
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetComplianceMetrics retrieves compliance metrics for a merchant
// GET /admin/v1/compliance/merchants/:id/metrics
func (h *AdminHandler) GetComplianceMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	merchantID := c.Param("id")
	if merchantID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_MERCHANT_ID",
			"Merchant ID is required",
		))
		return
	}

	// Get compliance metrics from service
	metrics, err := h.complianceService.GetComplianceMetrics(ctx, merchantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"FAILED_TO_GET_METRICS",
			"Failed to retrieve compliance metrics",
		))
		return
	}

	// Convert to DTO
	response := dto.APIResponse{
		Data: dto.GetComplianceMetricsResponse{
			MerchantID:             metrics.MerchantID,
			KYCTier:                metrics.KYCTier,
			MonthlyLimitUSD:        metrics.MonthlyLimitUSD,
			MonthlyVolumeUSD:       metrics.MonthlyVolumeUSD,
			RemainingLimitUSD:      metrics.RemainingLimitUSD,
			UtilizationPercent:     metrics.UtilizationPercent,
			TravelRuleTransactions: metrics.TravelRuleTransactions,
			TotalTransactions:      metrics.TotalTransactions,
			LastScreeningDate:      metrics.LastScreeningDate,
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// Helper functions

// parseUUID parses a UUID string
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// parseUUIDOrNil parses a UUID string or returns a nil UUID
func parseUUIDOrNil(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
