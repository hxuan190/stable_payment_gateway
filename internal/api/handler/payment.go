package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"

	"github.com/hxuan190/stable_payment_gateway/internal/api/dto"
	"github.com/hxuan190/stable_payment_gateway/internal/api/middleware"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/qrcode"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
)

// PaymentService defines the interface for payment business logic
type PaymentService interface {
	CreatePayment(ctx context.Context, req service.CreatePaymentRequest) (*model.Payment, error)
	GetPaymentStatus(ctx context.Context, paymentID string) (*model.Payment, error)
	ListPaymentsByMerchant(ctx context.Context, merchantID string, limit, offset int) ([]*model.Payment, error)
}

// ComplianceService defines the interface for compliance-related operations
type ComplianceService interface {
	ValidatePaymentCompliance(ctx context.Context, payment *model.Payment, travelRuleData *model.TravelRuleData, fromAddress string) error
	StoreTravelRuleData(ctx context.Context, data *model.TravelRuleData) error
}

// ExchangeRateService defines the interface for exchange rate operations
type ExchangeRateService interface {
	GetExchangeRate(ctx context.Context, currency string) (decimal.Decimal, error)
}

// PaymentHandler handles HTTP requests for payment operations
type PaymentHandler struct {
	paymentService      PaymentService
	complianceService   ComplianceService
	exchangeRateService ExchangeRateService
	qrGenerator         *qrcode.Generator
	baseURL             string // Base URL for payment pages (e.g., https://pay.example.com)
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(
	paymentService PaymentService,
	complianceService ComplianceService,
	exchangeRateService ExchangeRateService,
	baseURL string,
) *PaymentHandler {
	return &PaymentHandler{
		paymentService:      paymentService,
		complianceService:   complianceService,
		exchangeRateService: exchangeRateService,
		qrGenerator:         qrcode.NewGenerator(),
		baseURL:             baseURL,
	}
}

// CreatePayment handles POST /api/v1/payments
// @Summary Create a new payment
// @Description Create a new payment request for a merchant
// @Tags payments
// @Accept json
// @Produce json
// @Param request body dto.CreatePaymentRequest true "Payment creation request"
// @Success 201 {object} dto.APIResponse{data=dto.CreatePaymentResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 401 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/payments [post]
// @Security ApiKeyAuth
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated merchant from context
	merchant, err := middleware.GetMerchantFromContext(c)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to get merchant from context")

		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("UNAUTHORIZED", "Authentication required"))
		return
	}

	// Parse and validate request
	var req dto.CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
		}).Warn("Invalid payment creation request")

		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
			"INVALID_REQUEST",
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Convert amount from float64 to decimal.Decimal
	amountVND := decimal.NewFromFloat(req.AmountVND)

	// Build service request
	serviceReq := service.CreatePaymentRequest{
		MerchantID:  merchant.ID,
		AmountVND:   amountVND,
		Currency:    req.Currency,
		OrderID:     req.OrderID,
		Description: req.Description,
		CallbackURL: req.CallbackURL,
	}

	// Set chain if provided
	if req.Chain != "" {
		serviceReq.Chain = model.Chain(req.Chain)
	}

	// Create payment
	payment, err := h.paymentService.CreatePayment(ctx, serviceReq)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
			"amount_vnd":  amountVND,
		}).Error("Failed to create payment")

		// Map service errors to HTTP errors
		statusCode, errCode, errMessage := h.mapServiceError(err)
		c.JSON(statusCode, dto.ErrorResponse(errCode, errMessage))
		return
	}

	// Check if Travel Rule data is required (transactions > $1000 USD)
	// Convert VND to USD using payment's exchange rate
	travelRuleThreshold := decimal.NewFromInt(1000)
	if payment.AmountUSD.GreaterThan(travelRuleThreshold) {
		if req.TravelRule == nil {
			logger.WithContext(ctx).WithFields(logrus.Fields{
				"payment_id": payment.ID,
				"amount_usd": payment.AmountUSD.String(),
			}).Warn("Travel Rule data required but not provided")

			c.JSON(http.StatusBadRequest, dto.ErrorResponse(
				"TRAVEL_RULE_REQUIRED",
				fmt.Sprintf("Travel Rule data is required for transactions exceeding $%.2f USD", travelRuleThreshold),
			))
			return
		}

		// Convert DTO to model and store Travel Rule data
		travelRuleData := &model.TravelRuleData{
			PaymentID:           payment.ID,
			PayerFullName:       req.TravelRule.PayerFullName,
			PayerWalletAddress:  req.TravelRule.PayerWalletAddress,
			PayerCountry:        req.TravelRule.PayerCountry,
			MerchantFullName:    merchant.BusinessName,
			MerchantCountry:     "VN", // Default to Vietnam for now
			TransactionAmount:   payment.AmountUSD,
			TransactionCurrency: "USD",
		}

		// Set optional PayerIDDocument if provided
		if req.TravelRule.PayerIDDocument != "" {
			travelRuleData.PayerIDDocument.String = req.TravelRule.PayerIDDocument
			travelRuleData.PayerIDDocument.Valid = true
		}

		// Store Travel Rule data (if compliance service is available)
		if h.complianceService != nil {
			err = h.complianceService.StoreTravelRuleData(ctx, travelRuleData)
			if err != nil {
				logger.WithContext(ctx).WithFields(logrus.Fields{
					"error":      err.Error(),
					"payment_id": payment.ID,
				}).Error("Failed to store Travel Rule data")

				c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
					"TRAVEL_RULE_ERROR",
					"Failed to store Travel Rule data",
				))
				return
			}

			logger.WithContext(ctx).WithFields(logrus.Fields{
				"payment_id":    payment.ID,
				"payer_country": req.TravelRule.PayerCountry,
			}).Info("Travel Rule data stored successfully")
		} else {
			logger.WithContext(ctx).WithFields(logrus.Fields{
				"payment_id": payment.ID,
			}).Warn("Compliance service not available, Travel Rule data not stored")
		}
	}

	// Generate QR code
	qrCodeBase64, err := h.generateQRCode(payment)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":      err.Error(),
			"payment_id": payment.ID,
		}).Error("Failed to generate QR code")

		// Log error but don't fail the request - QR code can be generated later
		// Continue with empty QR code
		qrCodeBase64 = ""
	}

	// Build response
	response := dto.CreatePaymentResponse{
		PaymentID:         payment.ID,
		AmountVND:         payment.AmountVND,
		AmountCrypto:      payment.AmountCrypto,
		Currency:          payment.Currency,
		Chain:             string(payment.Chain),
		ExchangeRate:      payment.ExchangeRate,
		DestinationWallet: payment.DestinationWallet,
		PaymentReference:  payment.PaymentReference,
		ExpiresAt:         payment.ExpiresAt,
		FeeVND:            payment.FeeVND,
		NetAmountVND:      payment.NetAmountVND,
		Status:            string(payment.Status),
		QRCodeURL:         fmt.Sprintf("data:image/png;base64,%s", qrCodeBase64),
		PaymentURL:        h.getPaymentURL(payment.ID),
		CreatedAt:         payment.CreatedAt,
	}

	logger.WithContext(ctx).WithFields(logrus.Fields{
		"payment_id":  payment.ID,
		"merchant_id": merchant.ID,
		"amount_vnd":  payment.AmountVND,
		"currency":    payment.Currency,
		"chain":       payment.Chain,
	}).Info("Payment created successfully")

	c.JSON(http.StatusCreated, dto.SuccessResponse(response))
}

// GetPayment handles GET /api/v1/payments/:id
// @Summary Get payment details
// @Description Retrieve details of a specific payment
// @Tags payments
// @Accept json
// @Produce json
// @Param id path string true "Payment ID"
// @Success 200 {object} dto.APIResponse{data=dto.GetPaymentResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 401 {object} dto.APIResponse
// @Failure 404 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/payments/{id} [get]
// @Security ApiKeyAuth
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated merchant from context
	merchant, err := middleware.GetMerchantFromContext(c)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to get merchant from context")

		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("UNAUTHORIZED", "Authentication required"))
		return
	}

	// Get payment ID from path parameter
	paymentID := c.Param("id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("INVALID_REQUEST", "Payment ID is required"))
		return
	}

	// Get payment
	payment, err := h.paymentService.GetPaymentStatus(ctx, paymentID)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
			"payment_id":  paymentID,
		}).Error("Failed to get payment")

		statusCode, errCode, errMessage := h.mapServiceError(err)
		c.JSON(statusCode, dto.ErrorResponse(errCode, errMessage))
		return
	}

	// Verify payment belongs to this merchant
	if payment.MerchantID != merchant.ID {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"merchant_id":          merchant.ID,
			"payment_id":           paymentID,
			"payment_merchant_id":  payment.MerchantID,
		}).Warn("Merchant attempted to access payment from different merchant")

		c.JSON(http.StatusForbidden, dto.ErrorResponse("FORBIDDEN", "Access denied to this payment"))
		return
	}

	// Convert to response DTO
	response := dto.PaymentToResponse(payment)

	logger.WithContext(ctx).WithFields(logrus.Fields{
		"payment_id":  payment.ID,
		"merchant_id": merchant.ID,
		"status":      payment.Status,
	}).Debug("Payment retrieved successfully")

	c.JSON(http.StatusOK, dto.SuccessResponse(response))
}

// ListPayments handles GET /api/v1/payments
// @Summary List payments
// @Description List all payments for the authenticated merchant with pagination
// @Tags payments
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Param status query string false "Filter by status" Enums(created, pending, confirming, completed, expired, failed)
// @Success 200 {object} dto.APIResponse{data=dto.ListPaymentsResponse}
// @Failure 400 {object} dto.APIResponse
// @Failure 401 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/payments [get]
// @Security ApiKeyAuth
func (h *PaymentHandler) ListPayments(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated merchant from context
	merchant, err := middleware.GetMerchantFromContext(c)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to get merchant from context")

		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("UNAUTHORIZED", "Authentication required"))
		return
	}

	// Parse query parameters
	var req dto.ListPaymentsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
		}).Warn("Invalid list payments request")

		c.JSON(http.StatusBadRequest, dto.ErrorResponseWithDetails(
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

	// Get payments
	payments, err := h.paymentService.ListPaymentsByMerchant(ctx, merchant.ID, perPage, offset)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":       err.Error(),
			"merchant_id": merchant.ID,
		}).Error("Failed to list payments")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("INTERNAL_ERROR", "Failed to retrieve payments"))
		return
	}

	// Convert to list items
	paymentItems := make([]dto.PaymentListItem, len(payments))
	for i, payment := range payments {
		paymentItems[i] = dto.PaymentToListItem(payment)
	}

	// Build pagination metadata
	// Note: For a complete implementation, we should get total count from repository
	// For now, we'll provide basic pagination info
	pagination := &dto.PaginationMeta{
		CurrentPage: page,
		PerPage:     perPage,
		TotalPages:  0, // TODO: Calculate from total count
		TotalCount:  0, // TODO: Get from repository
	}

	response := dto.ListPaymentsResponse{
		Payments:   paymentItems,
		Pagination: pagination,
	}

	logger.WithContext(ctx).WithFields(logrus.Fields{
		"merchant_id":   merchant.ID,
		"count":         len(payments),
		"page":          page,
		"per_page":      perPage,
	}).Debug("Payments listed successfully")

	c.JSON(http.StatusOK, dto.SuccessResponse(response))
}

// Helper functions

// generateQRCode generates a QR code for the payment
func (h *PaymentHandler) generateQRCode(payment *model.Payment) (string, error) {
	// Determine token mint based on currency
	var tokenMint string
	switch payment.Currency {
	case "USDT":
		tokenMint = qrcode.USDTMintSolana
	case "USDC":
		tokenMint = qrcode.USDCMintSolana
	default:
		return "", fmt.Errorf("unsupported currency for QR code: %s", payment.Currency)
	}

	// Generate QR code
	qrConfig := qrcode.PaymentQRConfig{
		WalletAddress: payment.DestinationWallet,
		Amount:        payment.AmountCrypto,
		TokenMint:     tokenMint,
		Memo:          payment.PaymentReference,
		Label:         "Payment",
		Size:          qrcode.QRCodeSizeMedium,
	}

	return h.qrGenerator.GeneratePaymentQR(qrConfig)
}

// getPaymentURL constructs the payment page URL
func (h *PaymentHandler) getPaymentURL(paymentID string) string {
	if h.baseURL == "" {
		return fmt.Sprintf("/pay/%s", paymentID)
	}
	return fmt.Sprintf("%s/pay/%s", h.baseURL, paymentID)
}

// GetPublicPaymentStatus handles GET /api/v1/public/payments/:id/status
// @Summary Get public payment status
// @Description Retrieve payment status for public access (no authentication required) - Payer Experience Layer
// @Tags payments
// @Accept json
// @Produce json
// @Param id path string true "Payment ID"
// @Success 200 {object} dto.APIResponse{data=dto.PaymentStatusResponse}
// @Failure 404 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /api/v1/public/payments/{id}/status [get]
func (h *PaymentHandler) GetPublicPaymentStatus(c *gin.Context) {
	ctx := c.Request.Context()

	// Get payment ID from path parameter
	paymentID := c.Param("id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("INVALID_REQUEST", "Payment ID is required"))
		return
	}

	// Get payment
	payment, err := h.paymentService.GetPaymentStatus(ctx, paymentID)
	if err != nil {
		logger.WithContext(ctx).WithFields(logrus.Fields{
			"error":      err.Error(),
			"payment_id": paymentID,
		}).Error("Failed to get payment status")

		statusCode, errCode, errMessage := h.mapServiceError(err)
		c.JSON(statusCode, dto.ErrorResponse(errCode, errMessage))
		return
	}

	// Convert to public status response (excludes merchant-sensitive information)
	response := dto.PaymentToPublicStatusResponse(payment)

	logger.WithContext(ctx).WithFields(logrus.Fields{
		"payment_id": payment.ID,
		"status":     payment.Status,
	}).Debug("Public payment status retrieved successfully")

	c.JSON(http.StatusOK, dto.SuccessResponse(response))
}

// mapServiceError maps service layer errors to HTTP status codes and error messages
func (h *PaymentHandler) mapServiceError(err error) (statusCode int, errorCode string, errorMessage string) {
	// Default to internal server error
	statusCode = http.StatusInternalServerError
	errorCode = "INTERNAL_ERROR"
	errorMessage = "An internal error occurred"

	// Map specific errors
	switch {
	case errors.Is(err, service.ErrPaymentNotFound):
		statusCode = http.StatusNotFound
		errorCode = "PAYMENT_NOT_FOUND"
		errorMessage = "Payment not found"
	case errors.Is(err, service.ErrInvalidAmount):
		statusCode = http.StatusBadRequest
		errorCode = "INVALID_AMOUNT"
		errorMessage = "Invalid payment amount"
	case errors.Is(err, service.ErrPaymentExpired):
		statusCode = http.StatusBadRequest
		errorCode = "PAYMENT_EXPIRED"
		errorMessage = "Payment has expired"
	case errors.Is(err, service.ErrPaymentAlreadyCompleted):
		statusCode = http.StatusConflict
		errorCode = "PAYMENT_ALREADY_COMPLETED"
		errorMessage = "Payment has already been completed"
	case errors.Is(err, service.ErrInvalidPaymentState):
		statusCode = http.StatusBadRequest
		errorCode = "INVALID_PAYMENT_STATE"
		errorMessage = "Payment is in invalid state for this operation"
	case errors.Is(err, service.ErrInvalidChain):
		statusCode = http.StatusBadRequest
		errorCode = "INVALID_CHAIN"
		errorMessage = "Invalid or unsupported blockchain chain"
	case errors.Is(err, service.ErrMerchantNotFound):
		statusCode = http.StatusNotFound
		errorCode = "MERCHANT_NOT_FOUND"
		errorMessage = "Merchant not found"
	case errors.Is(err, service.ErrMerchantNotApproved):
		statusCode = http.StatusForbidden
		errorCode = "MERCHANT_NOT_APPROVED"
		errorMessage = "Merchant is not approved to accept payments"
	}

	return statusCode, errorCode, errorMessage
}
