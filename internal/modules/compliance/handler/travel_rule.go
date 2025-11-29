package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/service"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
)

// TravelRuleHandler handles FATF Travel Rule compliance API endpoints
type TravelRuleHandler struct {
	complianceService service.ComplianceService
	travelRuleRepo    repository.TravelRuleRepository
	logger            *logger.Logger
}

// NewTravelRuleHandler creates a new travel rule handler
func NewTravelRuleHandler(
	complianceService service.ComplianceService,
	travelRuleRepo repository.TravelRuleRepository,
	logger *logger.Logger,
) *TravelRuleHandler {
	return &TravelRuleHandler{
		complianceService: complianceService,
		travelRuleRepo:    travelRuleRepo,
		logger:            logger,
	}
}

// CreateTravelRuleDataRequest represents a request to submit travel rule data
type CreateTravelRuleDataRequest struct {
	PaymentID string `json:"payment_id" binding:"required,uuid"`

	// Payer Information (Originator) - FATF Required
	PayerFullName      string  `json:"payer_full_name" binding:"required,min=2,max=255"`
	PayerWalletAddress string  `json:"payer_wallet_address" binding:"required,min=26,max=255"`
	PayerCountry       string  `json:"payer_country" binding:"required,len=2"`
	PayerIDDocument    *string `json:"payer_id_document,omitempty"`
	PayerDateOfBirth   *string `json:"payer_date_of_birth,omitempty"` // Format: YYYY-MM-DD
	PayerAddress       *string `json:"payer_address,omitempty"`

	// Merchant Information (Beneficiary) - FATF Required
	MerchantFullName             string  `json:"merchant_full_name" binding:"required,min=2,max=255"`
	MerchantCountry              string  `json:"merchant_country" binding:"required,len=2"`
	MerchantWalletAddress        *string `json:"merchant_wallet_address,omitempty"`
	MerchantIDDocument           *string `json:"merchant_id_document,omitempty"`
	MerchantBusinessRegistration *string `json:"merchant_business_registration,omitempty"`
	MerchantAddress              *string `json:"merchant_address,omitempty"`

	// Transaction Information
	TransactionAmount   string  `json:"transaction_amount" binding:"required"` // Decimal as string
	TransactionCurrency string  `json:"transaction_currency" binding:"required,oneof=USD USDT USDC BUSD"`
	TransactionPurpose  *string `json:"transaction_purpose,omitempty"`
}

// TravelRuleDataResponse represents a travel rule data response
type TravelRuleDataResponse struct {
	ID        string `json:"id"`
	PaymentID string `json:"payment_id"`

	// Payer Information
	PayerFullName      string  `json:"payer_full_name"`
	PayerWalletAddress string  `json:"payer_wallet_address"`
	PayerCountry       string  `json:"payer_country"`
	PayerIDDocument    *string `json:"payer_id_document,omitempty"`
	PayerDateOfBirth   *string `json:"payer_date_of_birth,omitempty"`
	PayerAddress       *string `json:"payer_address,omitempty"`

	// Merchant Information
	MerchantFullName             string  `json:"merchant_full_name"`
	MerchantCountry              string  `json:"merchant_country"`
	MerchantWalletAddress        *string `json:"merchant_wallet_address,omitempty"`
	MerchantIDDocument           *string `json:"merchant_id_document,omitempty"`
	MerchantBusinessRegistration *string `json:"merchant_business_registration,omitempty"`
	MerchantAddress              *string `json:"merchant_address,omitempty"`

	// Transaction Information
	TransactionAmount   string  `json:"transaction_amount"`
	TransactionCurrency string  `json:"transaction_currency"`
	TransactionPurpose  *string `json:"transaction_purpose,omitempty"`

	// Risk Assessment
	RiskLevel            *string  `json:"risk_level,omitempty"`
	RiskScore            *float64 `json:"risk_score,omitempty"`
	ScreeningStatus      *string  `json:"screening_status,omitempty"`
	ScreeningCompletedAt *string  `json:"screening_completed_at,omitempty"`

	// Regulatory Reporting
	ReportedToAuthority bool    `json:"reported_to_authority"`
	ReportedAt          *string `json:"reported_at,omitempty"`
	ReportReference     *string `json:"report_reference,omitempty"`

	// Metadata
	IsCrossBorder   bool   `json:"is_cross_border"`
	RetentionPolicy string `json:"retention_policy"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// CreateTravelRuleData handles POST /api/v1/travel-rule
// @Summary Submit Travel Rule compliance data
// @Description Submit FATF Travel Rule data for a transaction > $1000 USD
// @Tags Travel Rule
// @Accept json
// @Produce json
// @Param request body CreateTravelRuleDataRequest true "Travel Rule Data"
// @Success 201 {object} TravelRuleDataResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/travel-rule [post]
func (h *TravelRuleHandler) CreateTravelRuleData(c *gin.Context) {
	var req CreateTravelRuleDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Parse transaction amount
	amount, err := decimal.NewFromString(req.TransactionAmount)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid transaction amount format",
		})
		return
	}

	// Validate amount meets Travel Rule threshold
	threshold := decimal.NewFromInt(1000)
	if amount.LessThan(threshold) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Transaction amount must be >= $1000 USD for Travel Rule compliance",
		})
		return
	}

	// Parse date of birth if provided
	var dobTime sql.NullTime
	if req.PayerDateOfBirth != nil {
		parsedDob, err := time.Parse("2006-01-02", *req.PayerDateOfBirth)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "Invalid date of birth format. Use YYYY-MM-DD",
			})
			return
		}
		dobTime = sql.NullTime{Time: parsedDob, Valid: true}
	}

	// Build Travel Rule data model
	data := &domain.TravelRuleData{
		PaymentID: req.PaymentID,

		// Payer Information
		PayerFullName:      req.PayerFullName,
		PayerWalletAddress: req.PayerWalletAddress,
		PayerCountry:       req.PayerCountry,
		PayerIDDocument:    toNullString(req.PayerIDDocument),
		PayerDateOfBirth:   dobTime,
		PayerAddress:       toNullString(req.PayerAddress),

		// Merchant Information
		MerchantFullName:             req.MerchantFullName,
		MerchantCountry:              req.MerchantCountry,
		MerchantWalletAddress:        toNullString(req.MerchantWalletAddress),
		MerchantIDDocument:           toNullString(req.MerchantIDDocument),
		MerchantBusinessRegistration: toNullString(req.MerchantBusinessRegistration),
		MerchantAddress:              toNullString(req.MerchantAddress),

		// Transaction Information
		TransactionAmount:   amount,
		TransactionCurrency: req.TransactionCurrency,
		TransactionPurpose:  toNullString(req.TransactionPurpose),

		// Set default screening status
		ScreeningStatus: sql.NullString{String: "pending", Valid: true},
		RiskLevel:       sql.NullString{String: "low", Valid: true},
		RetentionPolicy: "standard",
	}

	// Store Travel Rule data via compliance service
	err = h.complianceService.StoreTravelRuleData(c.Request.Context(), data)
	if err != nil {
		h.logger.Error("Failed to store Travel Rule data", err, map[string]interface{}{
			"payment_id": req.PaymentID,
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to store Travel Rule data: " + err.Error(),
		})
		return
	}

	h.logger.Info("Travel Rule data stored successfully", map[string]interface{}{
		"payment_id":       req.PaymentID,
		"payer_country":    req.PayerCountry,
		"merchant_country": req.MerchantCountry,
	})

	// Return created data
	response := h.modelToResponse(data)
	c.JSON(http.StatusCreated, response)
}

// GetTravelRuleDataByPayment handles GET /api/v1/travel-rule/payment/:payment_id
// @Summary Get Travel Rule data by payment ID
// @Description Retrieve Travel Rule compliance data for a specific payment
// @Tags Travel Rule
// @Produce json
// @Param payment_id path string true "Payment ID"
// @Success 200 {object} TravelRuleDataResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/travel-rule/payment/{payment_id} [get]
func (h *TravelRuleHandler) GetTravelRuleDataByPayment(c *gin.Context) {
	paymentID := c.Param("payment_id")

	data, err := h.travelRuleRepo.GetByPaymentID(c.Request.Context(), paymentID)
	if err != nil {
		if err == repository.ErrTravelRuleDataNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "Travel Rule data not found for this payment",
			})
			return
		}

		h.logger.Error("Failed to get Travel Rule data", err, map[string]interface{}{
			"payment_id": paymentID,
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to retrieve Travel Rule data",
		})
		return
	}

	response := h.modelToResponse(data)
	c.JSON(http.StatusOK, response)
}

// GetTravelRuleData handles GET /api/v1/travel-rule/:id
// @Summary Get Travel Rule data by ID
// @Description Retrieve Travel Rule compliance data by record ID
// @Tags Travel Rule
// @Produce json
// @Param id path string true "Travel Rule Data ID"
// @Success 200 {object} TravelRuleDataResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/travel-rule/{id} [get]
func (h *TravelRuleHandler) GetTravelRuleData(c *gin.Context) {
	id := c.Param("id")

	data, err := h.travelRuleRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrTravelRuleDataNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "Travel Rule data not found",
			})
			return
		}

		h.logger.Error("Failed to get Travel Rule data", err, map[string]interface{}{
			"id": id,
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to retrieve Travel Rule data",
		})
		return
	}

	response := h.modelToResponse(data)
	c.JSON(http.StatusOK, response)
}

// ListTravelRuleData handles GET /api/v1/travel-rule
// @Summary List Travel Rule data
// @Description List all Travel Rule compliance records with optional filters
// @Tags Travel Rule
// @Produce json
// @Param payer_country query string false "Filter by payer country code"
// @Param min_amount query number false "Minimum transaction amount"
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Param limit query int false "Page size" default(50)
// @Param offset query int false "Page offset" default(0)
// @Success 200 {array} TravelRuleDataResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/travel-rule [get]
func (h *TravelRuleHandler) ListTravelRuleData(c *gin.Context) {
	// Build filter from query parameters
	filter := repository.TravelRuleFilter{
		PayerCountry: c.Query("payer_country"),
		Limit:        50, // Default
		Offset:       0,  // Default
	}

	// Parse min_amount
	if minAmountStr := c.Query("min_amount"); minAmountStr != "" {
		minAmount, err := decimal.NewFromString(minAmountStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "Invalid min_amount format",
			})
			return
		}
		minAmountFloat, _ := minAmount.Float64()
		filter.MinAmount = &minAmountFloat
	}

	// Parse start_date
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "Invalid start_date format. Use YYYY-MM-DD",
			})
			return
		}
		filter.StartDate = &startDate
	}

	// Parse end_date
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "Invalid end_date format. Use YYYY-MM-DD",
			})
			return
		}
		filter.EndDate = &endDate
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		var limit int
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil && limit > 0 {
			if limit > 100 {
				limit = 100 // Cap at 100
			}
			filter.Limit = limit
		}
	}

	// Parse offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		var offset int
		if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	// Query database
	results, err := h.travelRuleRepo.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list Travel Rule data", err, map[string]interface{}{
			"filter": filter,
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to list Travel Rule data",
		})
		return
	}

	// Convert to response format
	responses := make([]TravelRuleDataResponse, len(results))
	for i, data := range results {
		responses[i] = *h.modelToResponse(data)
	}

	c.JSON(http.StatusOK, responses)
}

// Helper functions

func (h *TravelRuleHandler) modelToResponse(data *domain.TravelRuleData) *TravelRuleDataResponse {
	response := &TravelRuleDataResponse{
		ID:        data.ID,
		PaymentID: data.PaymentID,

		// Payer Information
		PayerFullName:      data.PayerFullName,
		PayerWalletAddress: data.PayerWalletAddress,
		PayerCountry:       data.PayerCountry,
		PayerIDDocument:    fromNullString(data.PayerIDDocument),
		PayerDateOfBirth:   fromNullTime(data.PayerDateOfBirth),
		PayerAddress:       fromNullString(data.PayerAddress),

		// Merchant Information
		MerchantFullName:             data.MerchantFullName,
		MerchantCountry:              data.MerchantCountry,
		MerchantWalletAddress:        fromNullString(data.MerchantWalletAddress),
		MerchantIDDocument:           fromNullString(data.MerchantIDDocument),
		MerchantBusinessRegistration: fromNullString(data.MerchantBusinessRegistration),
		MerchantAddress:              fromNullString(data.MerchantAddress),

		// Transaction Information
		TransactionAmount:   data.TransactionAmount.String(),
		TransactionCurrency: data.TransactionCurrency,
		TransactionPurpose:  fromNullString(data.TransactionPurpose),

		// Risk Assessment
		RiskLevel:            fromNullString(data.RiskLevel),
		RiskScore:            fromNullDecimal(data.RiskScore),
		ScreeningStatus:      fromNullString(data.ScreeningStatus),
		ScreeningCompletedAt: fromNullTimeString(data.ScreeningCompletedAt),

		// Regulatory Reporting
		ReportedToAuthority: data.ReportedToAuthority,
		ReportedAt:          fromNullTimeString(data.ReportedAt),
		ReportReference:     fromNullString(data.ReportReference),

		// Metadata
		IsCrossBorder:   data.IsCrossBorder(),
		RetentionPolicy: data.RetentionPolicy,
		CreatedAt:       data.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       data.UpdatedAt.Format(time.RFC3339),
	}

	return response
}

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

func fromNullString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func fromNullTime(nt sql.NullTime) *string {
	if !nt.Valid {
		return nil
	}
	str := nt.Time.Format("2006-01-02")
	return &str
}

func fromNullTimeString(nt sql.NullTime) *string {
	if !nt.Valid {
		return nil
	}
	str := nt.Time.Format(time.RFC3339)
	return &str
}

func fromNullDecimal(nd decimal.NullDecimal) *float64 {
	if !nd.Valid {
		return nil
	}
	f, _ := nd.Decimal.Float64()
	return &f
}
