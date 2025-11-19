package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

// SBVReportService handles generation of State Bank of Vietnam (SBV) regulatory reports
type SBVReportService struct {
	travelRuleRepo TravelRuleRepository
	paymentRepo    PaymentRepository
	logger         *logrus.Logger
}

// NewSBVReportService creates a new SBV report service
func NewSBVReportService(
	travelRuleRepo TravelRuleRepository,
	paymentRepo PaymentRepository,
	logger *logrus.Logger,
) *SBVReportService {
	return &SBVReportService{
		travelRuleRepo: travelRuleRepo,
		paymentRepo:    paymentRepo,
		logger:         logger,
	}
}

// SBVReportRequest contains parameters for generating an SBV report
type SBVReportRequest struct {
	StartDate time.Time
	EndDate   time.Time
	MinAmount decimal.Decimal // Optional: filter by minimum amount (e.g., 1000 USD)
}

// SBVReportRow represents a single row in the SBV report
type SBVReportRow struct {
	TransactionDate    time.Time       // Column 1: Transaction Date
	OriginatorName     string          // Column 2: Originator Name (Payer)
	OriginatorCountry  string          // Column 3: Originator Country
	BeneficiaryName    string          // Column 4: Beneficiary Name (Merchant)
	BeneficiaryCountry string          // Column 5: Beneficiary Country
	AmountVND          decimal.Decimal // Column 6: Amount (VND)
	Currency           string          // Column 7: Currency (always VND for SBV)
	Purpose            string          // Column 8: Transaction Purpose/Notes
}

// GenerateExcelReport generates an SBV regulatory report in Excel format
func (s *SBVReportService) GenerateExcelReport(ctx context.Context, req SBVReportRequest) (*excelize.File, error) {
	s.logger.WithFields(logrus.Fields{
		"start_date": req.StartDate.Format("2006-01-02"),
		"end_date":   req.EndDate.Format("2006-01-02"),
	}).Info("Generating SBV Excel report")

	// Validate date range
	if req.EndDate.Before(req.StartDate) {
		return nil, fmt.Errorf("end date must be after start date")
	}

	// Get Travel Rule data for the date range
	travelRuleData, err := s.travelRuleRepo.GetByDateRange(req.StartDate, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get travel rule data: %w", err)
	}

	// Filter by minimum amount if specified
	if req.MinAmount.GreaterThan(decimal.Zero) {
		filtered := make([]*model.TravelRuleData, 0)
		for _, tr := range travelRuleData {
			if tr.TransactionAmount.GreaterThanOrEqual(req.MinAmount) {
				filtered = append(filtered, tr)
			}
		}
		travelRuleData = filtered
	}

	s.logger.WithField("record_count", len(travelRuleData)).Info("Retrieved travel rule data for report")

	// Create Excel file
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			s.logger.WithError(err).Warn("Failed to close Excel file")
		}
	}()

	// Create or use default sheet
	sheetName := "SBV_Report"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheet: %w", err)
	}
	f.SetActiveSheet(index)

	// Delete default sheet if it exists
	if f.GetSheetIndex("Sheet1") >= 0 {
		f.DeleteSheet("Sheet1")
	}

	// Define styles
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   12,
			Color:  "FFFFFF",
			Family: "Arial",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"4472C4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create header style: %w", err)
	}

	dataStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size:   11,
			Family: "Arial",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "D3D3D3", Style: 1},
			{Type: "top", Color: "D3D3D3", Style: 1},
			{Type: "bottom", Color: "D3D3D3", Style: 1},
			{Type: "right", Color: "D3D3D3", Style: 1},
		},
		Alignment: &excelize.Alignment{
			Vertical: "center",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create data style: %w", err)
	}

	dateStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size:   11,
			Family: "Arial",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "D3D3D3", Style: 1},
			{Type: "top", Color: "D3D3D3", Style: 1},
			{Type: "bottom", Color: "D3D3D3", Style: 1},
			{Type: "right", Color: "D3D3D3", Style: 1},
		},
		NumFmt: 14, // Date format: dd/mm/yyyy
		Alignment: &excelize.Alignment{
			Vertical: "center",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create date style: %w", err)
	}

	amountStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size:   11,
			Family: "Arial",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "D3D3D3", Style: 1},
			{Type: "top", Color: "D3D3D3", Style: 1},
			{Type: "bottom", Color: "D3D3D3", Style: 1},
			{Type: "right", Color: "D3D3D3", Style: 1},
		},
		NumFmt: 3, // Number format with thousand separator
		Alignment: &excelize.Alignment{
			Horizontal: "right",
			Vertical:   "center",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create amount style: %w", err)
	}

	// Write header row
	headers := []string{
		"Transaction Date",
		"Originator Name",
		"Originator Country",
		"Beneficiary Name",
		"Beneficiary Country",
		"Amount (VND)",
		"Currency",
		"Transaction Purpose",
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// Set column widths
	f.SetColWidth(sheetName, "A", "A", 18)  // Transaction Date
	f.SetColWidth(sheetName, "B", "B", 25)  // Originator Name
	f.SetColWidth(sheetName, "C", "C", 18)  // Originator Country
	f.SetColWidth(sheetName, "D", "D", 30)  // Beneficiary Name
	f.SetColWidth(sheetName, "E", "E", 20)  // Beneficiary Country
	f.SetColWidth(sheetName, "F", "F", 20)  // Amount
	f.SetColWidth(sheetName, "G", "G", 12)  // Currency
	f.SetColWidth(sheetName, "H", "H", 30)  // Purpose

	// Write data rows
	for rowIdx, tr := range travelRuleData {
		row := rowIdx + 2 // Start from row 2 (row 1 is header)

		// Column 1: Transaction Date
		dateCell, _ := excelize.CoordinatesToCellName(1, row)
		f.SetCellValue(sheetName, dateCell, tr.CreatedAt.Format("2006-01-02"))
		f.SetCellStyle(sheetName, dateCell, dateCell, dateStyle)

		// Column 2: Originator Name (Payer)
		nameCell, _ := excelize.CoordinatesToCellName(2, row)
		f.SetCellValue(sheetName, nameCell, tr.PayerFullName)
		f.SetCellStyle(sheetName, nameCell, nameCell, dataStyle)

		// Column 3: Originator Country
		countryCell, _ := excelize.CoordinatesToCellName(3, row)
		f.SetCellValue(sheetName, countryCell, tr.PayerCountry)
		f.SetCellStyle(sheetName, countryCell, countryCell, dataStyle)

		// Column 4: Beneficiary Name (Merchant)
		beneficiaryCell, _ := excelize.CoordinatesToCellName(4, row)
		f.SetCellValue(sheetName, beneficiaryCell, tr.MerchantFullName)
		f.SetCellStyle(sheetName, beneficiaryCell, beneficiaryCell, dataStyle)

		// Column 5: Beneficiary Country
		beneficiaryCountryCell, _ := excelize.CoordinatesToCellName(5, row)
		f.SetCellValue(sheetName, beneficiaryCountryCell, tr.MerchantCountry)
		f.SetCellStyle(sheetName, beneficiaryCountryCell, beneficiaryCountryCell, dataStyle)

		// Column 6: Amount (VND)
		amountCell, _ := excelize.CoordinatesToCellName(6, row)
		amountFloat, _ := tr.TransactionAmount.Float64()
		f.SetCellValue(sheetName, amountCell, amountFloat)
		f.SetCellStyle(sheetName, amountCell, amountCell, amountStyle)

		// Column 7: Currency
		currencyCell, _ := excelize.CoordinatesToCellName(7, row)
		f.SetCellValue(sheetName, currencyCell, tr.TransactionCurrency)
		f.SetCellStyle(sheetName, currencyCell, currencyCell, dataStyle)

		// Column 8: Transaction Purpose
		purposeCell, _ := excelize.CoordinatesToCellName(8, row)
		purpose := tr.GetTransactionPurpose()
		f.SetCellValue(sheetName, purposeCell, purpose)
		f.SetCellStyle(sheetName, purposeCell, purposeCell, dataStyle)
	}

	// Add summary row
	summaryRow := len(travelRuleData) + 3
	summaryLabelCell, _ := excelize.CoordinatesToCellName(5, summaryRow)
	f.SetCellValue(sheetName, summaryLabelCell, "TOTAL:")
	f.SetCellStyle(sheetName, summaryLabelCell, summaryLabelCell, headerStyle)

	// Calculate total amount
	var totalAmount decimal.Decimal
	for _, tr := range travelRuleData {
		totalAmount = totalAmount.Add(tr.TransactionAmount)
	}

	summaryCellTotal, _ := excelize.CoordinatesToCellName(6, summaryRow)
	totalFloat, _ := totalAmount.Float64()
	f.SetCellValue(sheetName, summaryCellTotal, totalFloat)
	f.SetCellStyle(sheetName, summaryCellTotal, summaryCellTotal, amountStyle)

	// Add metadata footer
	metadataRow := summaryRow + 2
	metadataCell, _ := excelize.CoordinatesToCellName(1, metadataRow)
	f.SetCellValue(sheetName, metadataCell, fmt.Sprintf(
		"Report Generated: %s | Period: %s to %s | Total Transactions: %d",
		time.Now().Format("2006-01-02 15:04:05"),
		req.StartDate.Format("2006-01-02"),
		req.EndDate.Format("2006-01-02"),
		len(travelRuleData),
	))

	s.logger.WithField("row_count", len(travelRuleData)).Info("SBV Excel report generated successfully")

	return f, nil
}

// TravelRuleRepository defines the interface for travel rule data access
type TravelRuleRepository interface {
	GetByDateRange(startDate, endDate time.Time) ([]*model.TravelRuleData, error)
}
