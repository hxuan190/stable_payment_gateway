package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/service"
)

// SBVReportHandler handles SBV (State Bank of Vietnam) regulatory report endpoints
type SBVReportHandler struct {
	sbvReportService *service.SBVReportService
	logger           *logrus.Logger
}

// NewSBVReportHandler creates a new SBV report handler
func NewSBVReportHandler(
	sbvReportService *service.SBVReportService,
	logger *logrus.Logger,
) *SBVReportHandler {
	return &SBVReportHandler{
		sbvReportService: sbvReportService,
		logger:           logger,
	}
}

// ExportSBVReportRequest represents the request body for exporting SBV report
type ExportSBVReportRequest struct {
	StartDate string  `json:"start_date" binding:"required" example:"2025-01-01"`
	EndDate   string  `json:"end_date" binding:"required" example:"2025-01-31"`
	MinAmount float64 `json:"min_amount,omitempty" example:"1000"` // Optional: minimum transaction amount in USD
}

// ExportSBVReport exports transactions to Excel format for SBV regulatory reporting
// @Summary Export SBV regulatory report
// @Description Generates an Excel file containing all transactions within the specified date range in SBV-compliant format
// @Tags Admin, Regulatory
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param request body ExportSBVReportRequest true "Report parameters"
// @Success 200 {file} file "Excel file download"
// @Failure 400 {object} ErrorResponse "Invalid request parameters"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Router /admin/reports/sbv/export [post]
func (h *SBVReportHandler) ExportSBVReport(c *gin.Context) {
	var req ExportSBVReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid request body")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request body",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	// Parse start date
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		h.logger.WithError(err).Warn("Invalid start_date format")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid start_date format. Use YYYY-MM-DD",
			Code:  "INVALID_DATE_FORMAT",
		})
		return
	}

	// Parse end date
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		h.logger.WithError(err).Warn("Invalid end_date format")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid end_date format. Use YYYY-MM-DD",
			Code:  "INVALID_DATE_FORMAT",
		})
		return
	}

	// Set end date to end of day (23:59:59)
	endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	// Validate date range
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "end_date must be after start_date",
			Code:  "INVALID_DATE_RANGE",
		})
		return
	}

	// Check date range not too large (max 1 year)
	if endDate.Sub(startDate) > 365*24*time.Hour {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Date range cannot exceed 1 year",
			Code:  "DATE_RANGE_TOO_LARGE",
		})
		return
	}

	// Parse minimum amount (optional)
	minAmount := decimal.Zero
	if req.MinAmount > 0 {
		minAmount = decimal.NewFromFloat(req.MinAmount)
	}

	h.logger.WithFields(logrus.Fields{
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
		"min_amount": minAmount.String(),
	}).Info("Exporting SBV report")

	// Generate Excel report
	excelFile, err := h.sbvReportService.GenerateExcelReport(c.Request.Context(), service.SBVReportRequest{
		StartDate: startDate,
		EndDate:   endDate,
		MinAmount: minAmount,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate SBV report")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to generate report",
			Code:  "REPORT_GENERATION_FAILED",
		})
		return
	}

	// Generate filename
	filename := fmt.Sprintf("SBV_Report_%s_to_%s.xlsx",
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
	)

	// Set response headers for file download
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Transfer-Encoding", "binary")

	// Write Excel file to response
	if err := excelFile.Write(c.Writer); err != nil {
		h.logger.WithError(err).Error("Failed to write Excel file to response")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to write report file",
			Code:  "FILE_WRITE_FAILED",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"filename":   filename,
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	}).Info("SBV report exported successfully")
}

// GetSBVReportPreview generates a preview of the SBV report (JSON format)
// @Summary Preview SBV report data
// @Description Returns a JSON preview of transactions that would be included in the SBV report
// @Tags Admin, Regulatory
// @Accept json
// @Produce json
// @Param start_date query string true "Start date (YYYY-MM-DD)" example:"2025-01-01"
// @Param end_date query string true "End date (YYYY-MM-DD)" example:"2025-01-31"
// @Param min_amount query number false "Minimum amount (USD)" example:"1000"
// @Success 200 {object} SuccessResponse{data=[]service.SBVReportRow}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/reports/sbv/preview [get]
func (h *SBVReportHandler) GetSBVReportPreview(c *gin.Context) {
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	minAmountStr := c.DefaultQuery("min_amount", "0")

	if startDateStr == "" || endDateStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "start_date and end_date are required",
			Code:  "MISSING_PARAMETERS",
		})
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid start_date format. Use YYYY-MM-DD",
			Code:  "INVALID_DATE_FORMAT",
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid end_date format. Use YYYY-MM-DD",
			Code:  "INVALID_DATE_FORMAT",
		})
		return
	}

	endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	// Parse min amount
	minAmount := decimal.Zero
	if minAmountFloat, err := decimal.NewFromString(minAmountStr); err == nil && minAmountFloat.GreaterThan(decimal.Zero) {
		minAmount = minAmountFloat
	}

	// This would need to be implemented to return preview data
	// For now, return a success message
	c.JSON(http.StatusOK, SuccessResponse{
		Data: map[string]interface{}{
			"message":    "Preview functionality to be implemented",
			"start_date": startDate.Format("2006-01-02"),
			"end_date":   endDate.Format("2006-01-02"),
			"min_amount": minAmount.String(),
		},
		Timestamp: time.Now(),
	})
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}
