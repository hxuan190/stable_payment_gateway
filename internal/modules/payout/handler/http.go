package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"

	"github.com/hxuan190/stable_payment_gateway/internal/api/middleware"
	payoutDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payout/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/payout/service"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
)

// PayoutService defines the interface for payout business logic
type PayoutService interface {
	RequestPayout(input service.RequestPayoutInput) (*payoutDomain.Payout, error)
	GetPayoutByID(payoutID string) (*payoutDomain.Payout, error)
	ListPayoutsByMerchant(merchantID string, limit, offset int) ([]*payoutDomain.Payout, error)
}

// PayoutHandler handles HTTP requests for payout operations
type PayoutHandler struct {
	payoutService PayoutService
}

// NewPayoutHandler creates a new payout handler
func NewPayoutHandler(payoutService PayoutService) *PayoutHandler {
	return &PayoutHandler{
		payoutService: payoutService,
	}
}

// RequestPayout handles POST /api/v1/merchant/payouts
// @Summary Request a payout
// @Description Create a new payout request for the authenticated merchant
// @Tags payouts
// @Accept json
// @Produce json
// @Param request body RequestPayoutRequest true "Payout request"
// @Success 201 {object} APIResponse{data=RequestPayoutResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/merchant/payouts [post]
// @Security ApiKeyAuth
func (h *PayoutHandler) RequestPayout(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated merchant from context
	merchant, err := middleware.GetMerchantFromContext(c)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to get merchant from context")

		c.JSON(http.StatusUnauthorized, ErrorResponse("UNAUTHORIZED", "Authentication required"))
		return
	}

	// Parse and validate request
	var req RequestPayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
		}).Warn("Invalid payout request")

		c.JSON(http.StatusBadRequest, ErrorResponseWithDetails(
			"INVALID_REQUEST",
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Convert amount from float64 to decimal.Decimal
	amountVND := decimal.NewFromFloat(req.AmountVND)

	// Build service input
	serviceInput := service.RequestPayoutInput{
		MerchantID:        merchant.ID,
		AmountVND:         amountVND,
		BankAccountName:   req.BankAccountName,
		BankAccountNumber: req.BankAccountNumber,
		BankName:          req.BankName,
		BankBranch:        req.BankBranch,
		Notes:             req.Notes,
	}

	// Request payout
	payout, err := h.payoutService.RequestPayout(serviceInput)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
			"amount_vnd":  amountVND,
		}).Error("Failed to request payout")

		// Map service errors to HTTP errors
		statusCode, errCode, errMessage := h.mapServiceError(err)
		c.JSON(statusCode, ErrorResponse(errCode, errMessage))
		return
	}

	// Build response
	response := buildPayoutResponse(payout)

	logger.WithContext(ctx).WithFields(logrus.Fields{
		"payout_id":   payout.ID,
		"merchant_id": merchant.ID,
		"amount_vnd":  payout.AmountVND,
		"fee_vnd":     payout.FeeVND,
	}).Info("Payout requested successfully")

	c.JSON(http.StatusCreated, SuccessResponse(response))
}

// GetPayout handles GET /api/v1/merchant/payouts/:id
// @Summary Get payout details
// @Description Retrieve details of a specific payout
// @Tags payouts
// @Accept json
// @Produce json
// @Param id path string true "Payout ID"
// @Success 200 {object} APIResponse{data=GetPayoutResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/merchant/payouts/{id} [get]
// @Security ApiKeyAuth
func (h *PayoutHandler) GetPayout(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated merchant from context
	merchant, err := middleware.GetMerchantFromContext(c)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to get merchant from context")

		c.JSON(http.StatusUnauthorized, ErrorResponse("UNAUTHORIZED", "Authentication required"))
		return
	}

	// Get payout ID from path parameter
	payoutID := c.Param("id")
	if payoutID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse("INVALID_REQUEST", "Payout ID is required"))
		return
	}

	// Get payout
	payout, err := h.payoutService.GetPayoutByID(payoutID)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
			"payout_id":   payoutID,
		}).Error("Failed to get payout")

		statusCode, errCode, errMessage := h.mapServiceError(err)
		c.JSON(statusCode, ErrorResponse(errCode, errMessage))
		return
	}

	// Verify payout belongs to this merchant
	if payout.MerchantID != merchant.ID {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"merchant_id":        merchant.ID,
			"payout_id":          payoutID,
			"payout_merchant_id": payout.MerchantID,
		}).Warn("Merchant attempted to access payout from different merchant")

		c.JSON(http.StatusForbidden, ErrorResponse("FORBIDDEN", "Access denied to this payout"))
		return
	}

	// Convert to response DTO
	response := PayoutToResponse(payout)

	logger.WithContext(ctx).WithFields(logrus.Fields{
		"payout_id":   payout.ID,
		"merchant_id": merchant.ID,
		"status":      payout.Status,
	}).Debug("Payout retrieved successfully")

	c.JSON(http.StatusOK, SuccessResponse(response))
}

// ListPayouts handles GET /api/v1/merchant/payouts
// @Summary List payouts
// @Description List all payouts for the authenticated merchant with pagination
// @Tags payouts
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Param status query string false "Filter by status" Enums(requested, approved, processing, completed, rejected, failed)
// @Success 200 {object} APIResponse{data=ListPayoutsResponse}
// @Failure 400 {object} APIResponse
// @Failure 401 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/merchant/payouts [get]
// @Security ApiKeyAuth
func (h *PayoutHandler) ListPayouts(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated merchant from context
	merchant, err := middleware.GetMerchantFromContext(c)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to get merchant from context")

		c.JSON(http.StatusUnauthorized, ErrorResponse("UNAUTHORIZED", "Authentication required"))
		return
	}

	// Parse query parameters
	var req ListPayoutsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
		}).Warn("Invalid list payouts request")

		c.JSON(http.StatusBadRequest, ErrorResponseWithDetails(
			"INVALID_REQUEST",
			"Invalid query parameters",
			err.Error(),
		))
		return
	}

	// Set defaults
	page := req.Page
	if page < 1 {
		page = 1
	}
	perPage := req.PerPage
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	// Calculate offset
	offset := (page - 1) * perPage

	// Get payouts
	payouts, err := h.payoutService.ListPayoutsByMerchant(merchant.ID, perPage, offset)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
		}).Error("Failed to list payouts")

		c.JSON(http.StatusInternalServerError, ErrorResponse("INTERNAL_ERROR", "Failed to retrieve payouts"))
		return
	}

	// Convert to list items
	payoutItems := make([]PayoutListItem, len(payouts))
	for i, payout := range payouts {
		payoutItems[i] = PayoutToListItem(payout)
	}

	// Build pagination metadata
	// Note: For a complete implementation, we should get total count from repository
	// For now, we'll provide basic pagination info
	pagination := &PaginationMeta{
		CurrentPage: page,
		PerPage:     perPage,
		TotalPages:  0, // TODO: Calculate from total count
		TotalCount:  0, // TODO: Get from repository
	}

	response := ListPayoutsResponse{
		Payouts:    payoutItems,
		Pagination: pagination,
	}

	logger.WithContext(ctx).WithFields(logrus.Fields{
		"merchant_id": merchant.ID,
		"count":       len(payouts),
		"page":        page,
		"per_page":    perPage,
	}).Debug("Payouts listed successfully")

	c.JSON(http.StatusOK, SuccessResponse(response))
}

// Helper functions

// buildPayoutResponse builds a RequestPayoutResponse from a payoutDomain.Payout
func buildPayoutResponse(payout *payoutDomain.Payout) RequestPayoutResponse {
	response := RequestPayoutResponse{
		PayoutID:          payout.ID,
		MerchantID:        payout.MerchantID,
		AmountVND:         payout.AmountVND,
		FeeVND:            payout.FeeVND,
		NetAmountVND:      payout.NetAmountVND,
		BankAccountName:   payout.BankAccountName,
		BankAccountNumber: payout.BankAccountNumber,
		BankName:          payout.BankName,
		Status:            string(payout.Status),
		CreatedAt:         payout.CreatedAt,
	}

	// Handle optional fields
	if payout.BankBranch.Valid {
		bankBranch := payout.BankBranch.String
		response.BankBranch = &bankBranch
	}
	if payout.Notes.Valid {
		notes := payout.Notes.String
		response.Notes = &notes
	}

	return response
}

// mapServiceError maps service layer errors to HTTP status codes and error messages
func (h *PayoutHandler) mapServiceError(err error) (statusCode int, errorCode string, errorMessage string) {
	// Default to internal server error
	statusCode = http.StatusInternalServerError
	errorCode = "INTERNAL_ERROR"
	errorMessage = "An internal error occurred"

	// Map specific errors
	switch {
	case errors.Is(err, service.ErrPayoutNotFound):
		statusCode = http.StatusNotFound
		errorCode = "PAYOUT_NOT_FOUND"
		errorMessage = "Payout not found"
	case errors.Is(err, service.ErrPayoutInvalidAmount):
		statusCode = http.StatusBadRequest
		errorCode = "INVALID_AMOUNT"
		errorMessage = "Invalid payout amount"
	case errors.Is(err, service.ErrPayoutBelowMinimum):
		statusCode = http.StatusBadRequest
		errorCode = "AMOUNT_BELOW_MINIMUM"
		errorMessage = "Payout amount is below the minimum threshold"
	case errors.Is(err, service.ErrPayoutInsufficientBalance):
		statusCode = http.StatusBadRequest
		errorCode = "INSUFFICIENT_BALANCE"
		errorMessage = "Insufficient balance for payout"
	case errors.Is(err, service.ErrPayoutInvalidBankDetails):
		statusCode = http.StatusBadRequest
		errorCode = "INVALID_BANK_DETAILS"
		errorMessage = "Invalid bank account details"
	case errors.Is(err, service.ErrPayoutInvalidStatus):
		statusCode = http.StatusBadRequest
		errorCode = "INVALID_PAYOUT_STATUS"
		errorMessage = "Payout cannot be processed in current status"
	case errors.Is(err, service.ErrPayoutMerchantNotFound):
		statusCode = http.StatusNotFound
		errorCode = "MERCHANT_NOT_FOUND"
		errorMessage = "Merchant not found"
	case errors.Is(err, service.ErrPayoutMerchantNotApproved):
		statusCode = http.StatusForbidden
		errorCode = "MERCHANT_NOT_APPROVED"
		errorMessage = "Merchant is not approved for payouts"
	case errors.Is(err, service.ErrPayoutAlreadyProcessed):
		statusCode = http.StatusConflict
		errorCode = "PAYOUT_ALREADY_PROCESSED"
		errorMessage = "Payout has already been processed"
	case errors.Is(err, service.ErrPayoutCannotBeApproved):
		statusCode = http.StatusBadRequest
		errorCode = "CANNOT_APPROVE_PAYOUT"
		errorMessage = "Payout cannot be approved in current status"
	case errors.Is(err, service.ErrPayoutCannotBeRejected):
		statusCode = http.StatusBadRequest
		errorCode = "CANNOT_REJECT_PAYOUT"
		errorMessage = "Payout cannot be rejected in current status"
	default:
		// If error contains specific message, include it
		if err != nil {
			errorMessage = err.Error()
		}
	}

	return statusCode, errorCode, errorMessage
}
