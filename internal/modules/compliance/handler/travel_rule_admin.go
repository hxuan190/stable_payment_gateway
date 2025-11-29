package handler

import (
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

// TravelRuleAdminHandler handles admin endpoints for Travel Rule compliance
// These endpoints are for regulatory reporting and compliance officers
type TravelRuleAdminHandler struct {
	complianceService service.ComplianceService
	travelRuleRepo    repository.TravelRuleRepository
	logger            *logger.Logger
}

// NewTravelRuleAdminHandler creates a new travel rule admin handler
func NewTravelRuleAdminHandler(
	complianceService service.ComplianceService,
	travelRuleRepo repository.TravelRuleRepository,
	logger *logger.Logger,
) *TravelRuleAdminHandler {
	return &TravelRuleAdminHandler{
		complianceService: complianceService,
		travelRuleRepo:    travelRuleRepo,
		logger:            logger,
	}
}

// RegulatoryReportRequest represents a request for regulatory report
type RegulatoryReportRequest struct {
	StartDate string  `json:"start_date" binding:"required"` // Format: YYYY-MM-DD
	EndDate   string  `json:"end_date" binding:"required"`   // Format: YYYY-MM-DD
	MinAmount *string `json:"min_amount,omitempty"`          // Minimum transaction amount
	Country   *string `json:"country,omitempty"`             // Filter by country (payer or merchant)
}

// RegulatoryReportResponse represents a regulatory compliance report
type RegulatoryReportResponse struct {
	ReportID     string                      `json:"report_id"`
	GeneratedAt  string                      `json:"generated_at"`
	PeriodStart  string                      `json:"period_start"`
	PeriodEnd    string                      `json:"period_end"`
	TotalRecords int                         `json:"total_records"`
	CrossBorder  int                         `json:"cross_border_count"`
	HighRisk     int                         `json:"high_risk_count"`
	Reported     int                         `json:"reported_count"`
	TotalVolume  string                      `json:"total_volume_usd"`
	Transactions []RegulatoryTransactionItem `json:"transactions"`
}

// RegulatoryTransactionItem represents a transaction in the regulatory report
type RegulatoryTransactionItem struct {
	// Originator (Payer) Information
	OriginatorName    string  `json:"originator_name"`
	OriginatorWallet  string  `json:"originator_wallet"`
	OriginatorCountry string  `json:"originator_country"`
	OriginatorID      *string `json:"originator_id,omitempty"`
	OriginatorAddress *string `json:"originator_address,omitempty"`

	// Beneficiary (Merchant) Information
	BeneficiaryName        string  `json:"beneficiary_name"`
	BeneficiaryWallet      *string `json:"beneficiary_wallet,omitempty"`
	BeneficiaryCountry     string  `json:"beneficiary_country"`
	BeneficiaryID          *string `json:"beneficiary_id,omitempty"`
	BeneficiaryBusinessReg *string `json:"beneficiary_business_reg,omitempty"`
	BeneficiaryAddress     *string `json:"beneficiary_address,omitempty"`

	// Transaction Details
	TransactionDate     string  `json:"transaction_date"`
	TransactionAmount   string  `json:"transaction_amount"`
	TransactionCurrency string  `json:"transaction_currency"`
	TransactionPurpose  *string `json:"transaction_purpose,omitempty"`
	PaymentID           string  `json:"payment_id"`

	// Risk & Compliance
	IsCrossBorder       bool     `json:"is_cross_border"`
	RiskLevel           *string  `json:"risk_level,omitempty"`
	RiskScore           *float64 `json:"risk_score,omitempty"`
	ScreeningStatus     *string  `json:"screening_status,omitempty"`
	ReportedToAuthority bool     `json:"reported_to_authority"`
	ReportReference     *string  `json:"report_reference,omitempty"`
}

// GetRegulatoryReport handles POST /api/v1/admin/travel-rule/report
// @Summary Generate regulatory compliance report
// @Description Generate a FATF Travel Rule compliance report for regulatory authorities
// @Tags Admin - Travel Rule
// @Accept json
// @Produce json
// @Security AdminAuth
// @Param request body RegulatoryReportRequest true "Report Parameters"
// @Success 200 {object} RegulatoryReportResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/travel-rule/report [post]
func (h *TravelRuleAdminHandler) GetRegulatoryReport(c *gin.Context) {
	var req RegulatoryReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid start_date format. Use YYYY-MM-DD",
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid end_date format. Use YYYY-MM-DD",
		})
		return
	}

	// Validate date range
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "End date must be after start date",
		})
		return
	}

	// Build filter
	filter := repository.TravelRuleFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
		Limit:     1000, // Cap at 1000 records per report
	}

	// Parse optional filters
	if req.MinAmount != nil {
		minAmount, err := decimal.NewFromString(*req.MinAmount)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "Invalid min_amount format",
			})
			return
		}
		minAmountFloat, _ := minAmount.Float64()
		filter.MinAmount = &minAmountFloat
	}

	if req.Country != nil {
		filter.PayerCountry = *req.Country
	}

	// Query Travel Rule data
	results, err := h.travelRuleRepo.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to generate regulatory report", err, map[string]interface{}{
			"start_date": req.StartDate,
			"end_date":   req.EndDate,
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to generate regulatory report",
		})
		return
	}

	// Calculate statistics
	totalRecords := len(results)
	crossBorder := 0
	highRisk := 0
	reported := 0
	totalVolume := decimal.Zero

	transactions := make([]RegulatoryTransactionItem, totalRecords)
	for i, data := range results {
		if data.IsCrossBorder() {
			crossBorder++
		}
		if data.IsHighRisk() {
			highRisk++
		}
		if data.IsReportedToAuthority() {
			reported++
		}
		totalVolume = totalVolume.Add(data.TransactionAmount)

		transactions[i] = RegulatoryTransactionItem{
			OriginatorName:    data.PayerFullName,
			OriginatorWallet:  data.PayerWalletAddress,
			OriginatorCountry: data.PayerCountry,
			OriginatorID:      fromNullString(data.PayerIDDocument),
			OriginatorAddress: fromNullString(data.PayerAddress),

			BeneficiaryName:        data.MerchantFullName,
			BeneficiaryWallet:      fromNullString(data.MerchantWalletAddress),
			BeneficiaryCountry:     data.MerchantCountry,
			BeneficiaryID:          fromNullString(data.MerchantIDDocument),
			BeneficiaryBusinessReg: fromNullString(data.MerchantBusinessRegistration),
			BeneficiaryAddress:     fromNullString(data.MerchantAddress),

			TransactionDate:     data.CreatedAt.Format(time.RFC3339),
			TransactionAmount:   data.TransactionAmount.String(),
			TransactionCurrency: data.TransactionCurrency,
			TransactionPurpose:  fromNullString(data.TransactionPurpose),
			PaymentID:           data.PaymentID,

			IsCrossBorder:       data.IsCrossBorder(),
			RiskLevel:           fromNullString(data.RiskLevel),
			RiskScore:           fromNullDecimal(data.RiskScore),
			ScreeningStatus:     fromNullString(data.ScreeningStatus),
			ReportedToAuthority: data.ReportedToAuthority,
			ReportReference:     fromNullString(data.ReportReference),
		}
	}

	// Generate report ID
	reportID := fmt.Sprintf("TR-%s-%s-%d", startDate.Format("20060102"), endDate.Format("20060102"), time.Now().Unix())

	response := RegulatoryReportResponse{
		ReportID:     reportID,
		GeneratedAt:  time.Now().Format(time.RFC3339),
		PeriodStart:  req.StartDate,
		PeriodEnd:    req.EndDate,
		TotalRecords: totalRecords,
		CrossBorder:  crossBorder,
		HighRisk:     highRisk,
		Reported:     reported,
		TotalVolume:  totalVolume.String(),
		Transactions: transactions,
	}

	h.logger.Info("Regulatory report generated", map[string]interface{}{
		"report_id":     reportID,
		"total_records": totalRecords,
		"period":        fmt.Sprintf("%s to %s", req.StartDate, req.EndDate),
	})

	c.JSON(http.StatusOK, response)
}

// UpdateRiskAssessmentRequest represents a request to update risk assessment
type UpdateRiskAssessmentRequest struct {
	RiskLevel string  `json:"risk_level" binding:"required,oneof=low medium high critical"`
	RiskScore float64 `json:"risk_score" binding:"required,min=0,max=100"`
}

// UpdateRiskAssessment handles PUT /api/v1/admin/travel-rule/:id/risk
// @Summary Update risk assessment for a Travel Rule record
// @Description Update the risk level and score after manual review
// @Tags Admin - Travel Rule
// @Accept json
// @Produce json
// @Security AdminAuth
// @Param id path string true "Travel Rule Data ID"
// @Param request body UpdateRiskAssessmentRequest true "Risk Assessment"
// @Success 200 {object} TravelRuleDataResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/travel-rule/{id}/risk [put]
func (h *TravelRuleAdminHandler) UpdateRiskAssessment(c *gin.Context) {
	id := c.Param("id")

	var req UpdateRiskAssessmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate risk score range
	if req.RiskScore < 0 || req.RiskScore > 100 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Risk score must be between 0 and 100",
		})
		return
	}

	// Update risk assessment
	err := h.travelRuleRepo.UpdateRiskAssessment(c.Request.Context(), id, req.RiskLevel, req.RiskScore)
	if err != nil {
		if err == repository.ErrTravelRuleDataNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "Travel Rule data not found",
			})
			return
		}

		h.logger.Error("Failed to update risk assessment", err, map[string]interface{}{
			"id":         id,
			"risk_level": req.RiskLevel,
			"risk_score": req.RiskScore,
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to update risk assessment",
		})
		return
	}

	// Retrieve updated data
	data, err := h.travelRuleRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to retrieve updated data",
		})
		return
	}

	h.logger.Info("Risk assessment updated", map[string]interface{}{
		"id":         id,
		"risk_level": req.RiskLevel,
		"risk_score": req.RiskScore,
	})

	response := h.modelToResponse(data)
	c.JSON(http.StatusOK, response)
}

// MarkAsReportedRequest represents a request to mark as reported
type MarkAsReportedRequest struct {
	ReportReference string `json:"report_reference" binding:"required,min=1,max=255"`
}

// MarkAsReported handles PUT /api/v1/admin/travel-rule/:id/report
// @Summary Mark transaction as reported to authority
// @Description Mark a Travel Rule record as reported to regulatory authority
// @Tags Admin - Travel Rule
// @Accept json
// @Produce json
// @Security AdminAuth
// @Param id path string true "Travel Rule Data ID"
// @Param request body MarkAsReportedRequest true "Report Reference"
// @Success 200 {object} TravelRuleDataResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/travel-rule/{id}/report [put]
func (h *TravelRuleAdminHandler) MarkAsReported(c *gin.Context) {
	id := c.Param("id")

	var req MarkAsReportedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Mark as reported
	err := h.travelRuleRepo.MarkAsReported(c.Request.Context(), id, req.ReportReference)
	if err != nil {
		if err == repository.ErrTravelRuleDataNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "Travel Rule data not found",
			})
			return
		}

		h.logger.Error("Failed to mark as reported", err, map[string]interface{}{
			"id":               id,
			"report_reference": req.ReportReference,
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to mark as reported",
		})
		return
	}

	// Retrieve updated data
	data, err := h.travelRuleRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to retrieve updated data",
		})
		return
	}

	h.logger.Info("Travel Rule record marked as reported", map[string]interface{}{
		"id":               id,
		"report_reference": req.ReportReference,
	})

	response := h.modelToResponse(data)
	c.JSON(http.StatusOK, response)
}

// GetComplianceStatistics handles GET /api/v1/admin/travel-rule/statistics
// @Summary Get Travel Rule compliance statistics
// @Description Get overall statistics for Travel Rule compliance
// @Tags Admin - Travel Rule
// @Produce json
// @Security AdminAuth
// @Param days query int false "Number of days to look back" default(30)
// @Success 200 {object} ComplianceStatisticsResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/travel-rule/statistics [get]
func (h *TravelRuleAdminHandler) GetComplianceStatistics(c *gin.Context) {
	// Parse days parameter
	days := 30 // Default
	if daysStr := c.Query("days"); daysStr != "" {
		fmt.Sscanf(daysStr, "%d", &days)
		if days < 1 {
			days = 1
		}
		if days > 365 {
			days = 365
		}
	}

	// Calculate date range
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Query data
	filter := repository.TravelRuleFilter{
		StartDate: &startDate,
		EndDate:   &endDate,
		Limit:     10000, // High limit for statistics
	}

	results, err := h.travelRuleRepo.List(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to get compliance statistics", err, map[string]interface{}{
			"days": days,
		})
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get compliance statistics",
		})
		return
	}

	// Calculate statistics
	stats := calculateStatistics(results)

	c.JSON(http.StatusOK, stats)
}

// ComplianceStatisticsResponse represents compliance statistics
type ComplianceStatisticsResponse struct {
	Period         string                       `json:"period"`
	TotalRecords   int                          `json:"total_records"`
	CrossBorder    int                          `json:"cross_border_count"`
	CrossBorderPct float64                      `json:"cross_border_percentage"`
	HighRisk       int                          `json:"high_risk_count"`
	HighRiskPct    float64                      `json:"high_risk_percentage"`
	Reported       int                          `json:"reported_count"`
	ReportedPct    float64                      `json:"reported_percentage"`
	TotalVolume    string                       `json:"total_volume_usd"`
	ByCountry      map[string]CountryStatistics `json:"by_country"`
	ByRiskLevel    map[string]int               `json:"by_risk_level"`
	ByPurpose      map[string]int               `json:"by_purpose"`
}

// CountryStatistics represents statistics for a specific country
type CountryStatistics struct {
	Count  int    `json:"count"`
	Volume string `json:"volume_usd"`
}

func calculateStatistics(results []*domain.TravelRuleData) ComplianceStatisticsResponse {
	totalRecords := len(results)
	crossBorder := 0
	highRisk := 0
	reported := 0
	totalVolume := decimal.Zero

	byCountry := make(map[string]*CountryStatistics)
	byRiskLevel := make(map[string]int)
	byPurpose := make(map[string]int)

	for _, data := range results {
		if data.IsCrossBorder() {
			crossBorder++
		}
		if data.IsHighRisk() {
			highRisk++
		}
		if data.IsReportedToAuthority() {
			reported++
		}
		totalVolume = totalVolume.Add(data.TransactionAmount)

		// By country (payer)
		if _, exists := byCountry[data.PayerCountry]; !exists {
			byCountry[data.PayerCountry] = &CountryStatistics{
				Count:  0,
				Volume: "0",
			}
		}
		stats := byCountry[data.PayerCountry]
		stats.Count++
		volume, _ := decimal.NewFromString(stats.Volume)
		volume = volume.Add(data.TransactionAmount)
		stats.Volume = volume.String()

		// By risk level
		riskLevel := data.GetRiskLevel()
		byRiskLevel[riskLevel]++

		// By purpose
		purpose := data.GetTransactionPurpose()
		byPurpose[purpose]++
	}

	// Calculate percentages
	crossBorderPct := 0.0
	highRiskPct := 0.0
	reportedPct := 0.0
	if totalRecords > 0 {
		crossBorderPct = float64(crossBorder) / float64(totalRecords) * 100
		highRiskPct = float64(highRisk) / float64(totalRecords) * 100
		reportedPct = float64(reported) / float64(totalRecords) * 100
	}

	// Convert byCountry map
	byCountryResponse := make(map[string]CountryStatistics)
	for country, stats := range byCountry {
		byCountryResponse[country] = *stats
	}

	return ComplianceStatisticsResponse{
		Period:         fmt.Sprintf("Last %d days", len(results)),
		TotalRecords:   totalRecords,
		CrossBorder:    crossBorder,
		CrossBorderPct: crossBorderPct,
		HighRisk:       highRisk,
		HighRiskPct:    highRiskPct,
		Reported:       reported,
		ReportedPct:    reportedPct,
		TotalVolume:    totalVolume.String(),
		ByCountry:      byCountryResponse,
		ByRiskLevel:    byRiskLevel,
		ByPurpose:      byPurpose,
	}
}

// Helper function to convert model to response (reuse from travel_rule.go)
func (h *TravelRuleAdminHandler) modelToResponse(data *domain.TravelRuleData) *TravelRuleDataResponse {
	return &TravelRuleDataResponse{
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
}
