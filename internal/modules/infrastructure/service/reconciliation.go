package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	infrastructureRepository "github.com/hxuan190/stable_payment_gateway/internal/modules/infrastructure/repository"
	ledgerRepository "github.com/hxuan190/stable_payment_gateway/internal/modules/ledger/repository"
	ledgerService "github.com/hxuan190/stable_payment_gateway/internal/modules/ledger/service"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/shopspring/decimal"
)

// ReconciliationStatus represents the status of a reconciliation check
type ReconciliationStatus string

const (
	ReconciliationStatusBalanced   ReconciliationStatus = "balanced"    // Assets == Liabilities (within tolerance)
	ReconciliationStatusSurplus    ReconciliationStatus = "surplus"     // Assets > Liabilities (good)
	ReconciliationStatusDeficit    ReconciliationStatus = "deficit"     // Assets < Liabilities (CRITICAL)
	ReconciliationStatusError      ReconciliationStatus = "error"       // Reconciliation failed
	ReconciliationStatusInProgress ReconciliationStatus = "in_progress" // Currently running
)

// Reconciliation tolerance: Allow 1000 VND difference (rounding errors)
const ReconciliationToleranceVND = 1000

// ReconciliationService handles daily solvency checks and audits
type ReconciliationService struct {
	db                  *sql.DB
	ledgerService       *ledgerService.LedgerService
	exchangeRateService *ExchangeRateService
	reconciliationRepo  *infrastructureRepository.ReconciliationRepository
	balanceRepo         *ledgerRepository.BalanceRepository
	logger              *logger.Logger
}

// NewReconciliationService creates a new reconciliation service
func NewReconciliationService(
	db *sql.DB,
	ledgerService *ledgerService.LedgerService,
	exchangeRateService *ExchangeRateService,
	reconciliationRepo *infrastructureRepository.ReconciliationRepository,
	balanceRepo *ledgerRepository.BalanceRepository,
	logger *logger.Logger,
) *ReconciliationService {
	return &ReconciliationService{
		db:                  db,
		ledgerService:       ledgerService,
		exchangeRateService: exchangeRateService,
		reconciliationRepo:  reconciliationRepo,
		balanceRepo:         balanceRepo,
		logger:              logger,
	}
}

// PerformDailyReconciliation runs a complete reconciliation check
// This verifies that the system is solvent: Assets >= Liabilities
//
// Formula:
//
//	Assets = Hot Wallet Crypto (converted to VND) + VND Pool + Fee Revenue
//	Liabilities = Sum of all merchant balances (available + pending + reserved)
//	Difference = Assets - Liabilities
//
// Status:
//   - Deficit (< -tolerance): CRITICAL - System is insolvent
//   - Balanced (within tolerance): OK
//   - Surplus (> tolerance): OK - Accumulated fees/spread
func (s *ReconciliationService) PerformDailyReconciliation(ctx context.Context) error {
	s.logger.Info("Starting daily reconciliation", nil)

	startTime := time.Now()

	// Create initial reconciliation log
	reconLog := &model.ReconciliationLog{
		ReconciledAt: startTime,
		Status:       string(ReconciliationStatusInProgress),
	}

	// Step 1: Calculate total liabilities (what we owe merchants)
	totalLiabilities, err := s.calculateTotalLiabilities(ctx)
	if err != nil {
		s.logger.Error("Failed to calculate liabilities", err, nil)
		reconLog.Status = string(ReconciliationStatusError)
		reconLog.ErrorMessage = sql.NullString{String: err.Error(), Valid: true}
		s.reconciliationRepo.Create(reconLog)
		return fmt.Errorf("failed to calculate liabilities: %w", err)
	}
	reconLog.TotalLiabilitiesVND = totalLiabilities

	s.logger.Info("Calculated total liabilities", map[string]interface{}{
		"total_liabilities_vnd": totalLiabilities.String(),
	})

	// Step 2: Calculate total assets (what we have)
	totalAssets, assetBreakdown, err := s.calculateTotalAssets(ctx)
	if err != nil {
		s.logger.Error("Failed to calculate assets", err, nil)
		reconLog.Status = string(ReconciliationStatusError)
		reconLog.ErrorMessage = sql.NullString{String: err.Error(), Valid: true}
		reconLog.TotalAssetsVND = totalAssets // Save partial result
		s.reconciliationRepo.Create(reconLog)
		return fmt.Errorf("failed to calculate assets: %w", err)
	}
	reconLog.TotalAssetsVND = totalAssets
	reconLog.AssetBreakdown = assetBreakdown

	s.logger.Info("Calculated total assets", map[string]interface{}{
		"total_assets_vnd": totalAssets.String(),
		"breakdown":        assetBreakdown,
	})

	// Step 3: Calculate difference
	difference := totalAssets.Sub(totalLiabilities)
	reconLog.DifferenceVND = difference

	s.logger.Info("Calculated reconciliation difference", map[string]interface{}{
		"assets":      totalAssets.String(),
		"liabilities": totalLiabilities.String(),
		"difference":  difference.String(),
	})

	// Step 4: Determine status and alert if necessary
	tolerance := decimal.NewFromInt(ReconciliationToleranceVND)

	if difference.LessThan(tolerance.Neg()) {
		// CRITICAL: System is insolvent (Assets < Liabilities)
		reconLog.Status = string(ReconciliationStatusDeficit)
		reconLog.AlertTriggered = true

		s.logger.Error("CRITICAL: System insolvency detected!", nil, map[string]interface{}{
			"assets_vnd":      totalAssets.String(),
			"liabilities_vnd": totalLiabilities.String(),
			"deficit_vnd":     difference.String(),
			"breakdown":       assetBreakdown,
		})

		// TODO: Trigger critical alert to ops team
		// - Send email to finance team
		// - Send SMS to on-call engineer
		// - Create high-priority ticket
		// - Halt new payouts until resolved

	} else if difference.Abs().LessThanOrEqual(tolerance) {
		// Balanced: Within tolerance range
		reconLog.Status = string(ReconciliationStatusBalanced)
		s.logger.Info("Reconciliation balanced (within tolerance)", map[string]interface{}{
			"difference_vnd": difference.String(),
			"tolerance_vnd":  tolerance.String(),
		})
	} else {
		// Surplus: Assets > Liabilities (accumulated fees/spread)
		reconLog.Status = string(ReconciliationStatusSurplus)
		s.logger.Info("Reconciliation shows surplus (accumulated revenue)", map[string]interface{}{
			"surplus_vnd": difference.String(),
			"breakdown":   assetBreakdown,
		})
	}

	// Step 5: Save reconciliation log
	if err := s.reconciliationRepo.Create(reconLog); err != nil {
		s.logger.Error("Failed to save reconciliation log", err, nil)
		return fmt.Errorf("failed to save reconciliation log: %w", err)
	}

	duration := time.Since(startTime)
	s.logger.Info("Daily reconciliation completed", map[string]interface{}{
		"status":           reconLog.Status,
		"duration_seconds": duration.Seconds(),
		"alert_triggered":  reconLog.AlertTriggered,
	})

	// Return error if system is insolvent to trigger worker alerts
	if reconLog.Status == string(ReconciliationStatusDeficit) {
		return fmt.Errorf("CRITICAL: system insolvency detected - deficit: %s VND", difference.String())
	}

	return nil
}

// calculateTotalLiabilities sums all merchant balances
// Liabilities = SUM(available_vnd + pending_vnd + reserved_vnd) for all merchants
func (s *ReconciliationService) calculateTotalLiabilities(ctx context.Context) (decimal.Decimal, error) {
	query := `
		SELECT
			COALESCE(SUM(available_vnd + pending_vnd + reserved_vnd), 0) as total_liabilities
		FROM merchant_balances
	`

	var totalLiabilities decimal.Decimal
	err := s.db.QueryRowContext(ctx, query).Scan(&totalLiabilities)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to query merchant balances: %w", err)
	}

	return totalLiabilities, nil
}

// calculateTotalAssets sums all assets the platform holds
// Assets = Crypto Holdings (VND) + VND Pool + Fee Revenue + OTC Spread - Expenses
func (s *ReconciliationService) calculateTotalAssets(ctx context.Context) (decimal.Decimal, map[string]string, error) {
	breakdown := make(map[string]string)
	totalAssets := decimal.Zero

	// 1. Get crypto holdings from ledger (crypto_pool account)
	cryptoPoolBalance, err := s.ledgerService.GetAccountBalance(ledgerService.AccountCryptoPool)
	if err != nil {
		s.logger.Warn("Failed to get crypto pool balance, assuming zero", map[string]interface{}{
			"error": err.Error(),
		})
		cryptoPoolBalance = decimal.Zero
	}

	// Convert crypto to VND (assume USDT for simplicity, in production parse crypto amounts)
	// For now, treat crypto_pool as already VND-converted value
	// In production, you'd query actual wallet balances and convert
	cryptoVND := cryptoPoolBalance
	breakdown["crypto_pool_vnd"] = cryptoVND.String()
	totalAssets = totalAssets.Add(cryptoVND)

	// 2. Get VND pool (cash holdings from OTC conversions)
	vndPoolBalance, err := s.ledgerService.GetAccountBalance(ledgerService.AccountVNDPool)
	if err != nil {
		s.logger.Warn("Failed to get VND pool balance, assuming zero", map[string]interface{}{
			"error": err.Error(),
		})
		vndPoolBalance = decimal.Zero
	}
	breakdown["vnd_pool"] = vndPoolBalance.String()
	totalAssets = totalAssets.Add(vndPoolBalance)

	// 3. Get fee revenue (accumulated fees)
	feeRevenue, err := s.ledgerService.GetAccountBalance(ledgerService.AccountFeeRevenue)
	if err != nil {
		s.logger.Warn("Failed to get fee revenue, assuming zero", map[string]interface{}{
			"error": err.Error(),
		})
		feeRevenue = decimal.Zero
	}
	breakdown["fee_revenue"] = feeRevenue.String()
	totalAssets = totalAssets.Add(feeRevenue)

	// 4. Get OTC spread revenue
	otcSpread, err := s.ledgerService.GetAccountBalance(ledgerService.AccountOTCSpread)
	if err != nil {
		s.logger.Warn("Failed to get OTC spread, assuming zero", map[string]interface{}{
			"error": err.Error(),
		})
		otcSpread = decimal.Zero
	}
	breakdown["otc_spread"] = otcSpread.String()
	totalAssets = totalAssets.Add(otcSpread)

	// 5. Subtract expenses
	otcExpense, err := s.ledgerService.GetAccountBalance(ledgerService.AccountOTCExpense)
	if err != nil {
		s.logger.Warn("Failed to get OTC expense, assuming zero", map[string]interface{}{
			"error": err.Error(),
		})
		otcExpense = decimal.Zero
	}
	breakdown["otc_expense"] = otcExpense.String()
	totalAssets = totalAssets.Sub(otcExpense)

	payoutExpense, err := s.ledgerService.GetAccountBalance(ledgerService.AccountPayoutExpense)
	if err != nil {
		s.logger.Warn("Failed to get payout expense, assuming zero", map[string]interface{}{
			"error": err.Error(),
		})
		payoutExpense = decimal.Zero
	}
	breakdown["payout_expense"] = payoutExpense.String()
	totalAssets = totalAssets.Sub(payoutExpense)

	// 6. Get platform bank account balance (if tracked)
	platformBank, err := s.ledgerService.GetAccountBalance(ledgerService.AccountPlatformBank)
	if err != nil {
		s.logger.Warn("Failed to get platform bank balance, assuming zero", map[string]interface{}{
			"error": err.Error(),
		})
		platformBank = decimal.Zero
	}
	breakdown["platform_bank"] = platformBank.String()
	totalAssets = totalAssets.Add(platformBank)

	breakdown["total_assets"] = totalAssets.String()

	return totalAssets, breakdown, nil
}

// GetLatestReconciliation retrieves the most recent reconciliation result
func (s *ReconciliationService) GetLatestReconciliation(ctx context.Context) (*model.ReconciliationLog, error) {
	return s.reconciliationRepo.GetLatest()
}

// GetReconciliationHistory retrieves reconciliation history for a date range
func (s *ReconciliationService) GetReconciliationHistory(ctx context.Context, startDate, endDate time.Time, limit int) ([]*model.ReconciliationLog, error) {
	return s.reconciliationRepo.GetByDateRange(startDate, endDate, limit)
}

// GetDeficitAlerts retrieves all reconciliation instances where the system was insolvent
func (s *ReconciliationService) GetDeficitAlerts(ctx context.Context, limit int) ([]*model.ReconciliationLog, error) {
	return s.reconciliationRepo.GetByStatus(string(ReconciliationStatusDeficit), limit)
}
