package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/shopspring/decimal"
)

// handleWebhookDelivery processes webhook delivery jobs
func (s *Server) handleWebhookDelivery(ctx context.Context, task *asynq.Task) error {
	var payload WebhookDeliveryPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal webhook payload: %w", err)
	}

	logger.Info("Processing webhook delivery", logger.Fields{
		"merchant_id": payload.MerchantID,
		"event":       payload.Event,
		"attempt":     payload.Attempt,
	})

	// Get merchant details
	merchant, err := s.merchantRepo.GetByID(ctx, payload.MerchantID)
	if err != nil {
		return fmt.Errorf("failed to get merchant: %w", err)
	}

	// Check if merchant has webhook URL configured
	if merchant.WebhookURL == "" {
		logger.Warn("Merchant has no webhook URL configured", logger.Fields{
			"merchant_id": payload.MerchantID,
		})
		return nil // Not an error, just skip
	}

	// Send webhook using notification service
	err = s.notificationSvc.SendWebhook(ctx, merchant, payload.Event, payload.Payload)
	if err != nil {
		logger.Error("Webhook delivery failed", err, logger.Fields{
			"merchant_id": payload.MerchantID,
			"event":       payload.Event,
			"attempt":     payload.Attempt,
		})
		return err // Will be retried by asynq
	}

	logger.Info("Webhook delivered successfully", logger.Fields{
		"merchant_id": payload.MerchantID,
		"event":       payload.Event,
		"attempt":     payload.Attempt,
	})

	return nil
}

// handlePaymentExpiry processes payment expiry check jobs
func (s *Server) handlePaymentExpiry(ctx context.Context, task *asynq.Task) error {
	var payload PaymentExpiryPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		// If payload is empty (from scheduled task), use current time
		payload.RunAt = time.Now()
	}

	logger.Info("Processing payment expiry check", logger.Fields{
		"run_at": payload.RunAt,
	})

	// Get expired payments (created > 30 minutes ago and still in pending/created state)
	expiredPayments, err := s.paymentRepo.GetExpiredPayments()
	if err != nil {
		return fmt.Errorf("failed to get expired payments: %w", err)
	}

	if len(expiredPayments) == 0 {
		logger.Info("No expired payments found")
		return nil
	}

	logger.Info("Found expired payments", logger.Fields{
		"count": len(expiredPayments),
	})

	// Expire each payment
	for _, payment := range expiredPayments {
		err := s.paymentService.ExpirePayment(ctx, payment.ID)
		if err != nil {
			logger.Error("Failed to expire payment", err, logger.Fields{
				"payment_id": payment.ID,
			})
			// Continue with other payments even if one fails
			continue
		}

		logger.Info("Payment expired", logger.Fields{
			"payment_id":  payment.ID,
			"merchant_id": payment.MerchantID,
		})
	}

	logger.Info("Payment expiry check completed", logger.Fields{
		"total_expired": len(expiredPayments),
	})

	return nil
}

// handleBalanceCheck processes wallet balance check jobs
func (s *Server) handleBalanceCheck(ctx context.Context, task *asynq.Task) error {
	var payload BalanceCheckPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		// If payload is empty (from scheduled task), use current time
		payload.RunAt = time.Now()
	}

	logger.Info("Processing balance check", logger.Fields{
		"run_at": payload.RunAt,
	})

	if s.solanaWallet == nil {
		logger.Warn("Solana wallet not configured, skipping balance check")
		return nil
	}

	// Get wallet address
	walletAddress := s.solanaWallet.GetAddress()

	// Check USDT balance (SPL token)
	// USDT mint address on Solana: Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB
	usdtMint := "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
	usdtBalance, err := s.solanaWallet.GetTokenBalance(ctx, usdtMint)
	if err != nil {
		logger.Error("Failed to get USDT balance", err)
		// Don't fail the job, just log and continue
	}

	// Check USDC balance (SPL token)
	// USDC mint address on Solana: EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v
	usdcMint := "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	usdcBalance, err := s.solanaWallet.GetTokenBalance(ctx, usdcMint)
	if err != nil {
		logger.Error("Failed to get USDC balance", err)
		// Don't fail the job, just log and continue
	}

	// Calculate total balance in USD
	totalBalance := usdtBalance.Add(usdcBalance)

	logger.Info("Wallet balance checked", logger.Fields{
		"wallet_address": walletAddress,
		"usdt_balance":   usdtBalance.String(),
		"usdc_balance":   usdcBalance.String(),
		"total_balance":  totalBalance.String(),
	})

	// Save balance to database
	walletBalance := &model.WalletBalanceSnapshot{
		WalletAddress: walletAddress,
		Chain:         model.ChainSolana,
		Network:       model.NetworkDevnet, // TODO: Get from config
		NativeBalance: decimal.Zero,
		NativeCurrency: "SOL",
		USDTBalance:   usdtBalance,
		USDCBalance:   usdcBalance,
		TotalUSDValue: totalBalance,
		MinThresholdUSD: decimal.NewFromFloat(1000),
		MaxThresholdUSD: decimal.NewFromFloat(10000),
		IsBelowMinThreshold: totalBalance.LessThan(decimal.NewFromFloat(1000)),
		IsAboveMaxThreshold: totalBalance.GreaterThan(decimal.NewFromFloat(10000)),
		SnapshotAt:    time.Now(),
	}

	err = s.walletBalanceRepo.Create(ctx, walletBalance)
	if err != nil {
		logger.Error("Failed to save wallet balance", err)
		// Don't fail the job, balance was checked successfully
	}

	// Check thresholds and send alerts
	minThreshold := decimal.NewFromFloat(1000)  // $1,000 minimum
	maxThreshold := decimal.NewFromFloat(10000) // $10,000 maximum

	if totalBalance.LessThan(minThreshold) {
		logger.Warn("Wallet balance below minimum threshold", logger.Fields{
			"wallet_address": walletAddress,
			"balance":        totalBalance.String(),
			"threshold":      minThreshold.String(),
		})
		// TODO: Send alert notification to ops team
	}

	if totalBalance.GreaterThan(maxThreshold) {
		logger.Warn("Wallet balance above maximum threshold", logger.Fields{
			"wallet_address": walletAddress,
			"balance":        totalBalance.String(),
			"threshold":      maxThreshold.String(),
		})
		// TODO: Send alert notification to ops team
	}

	return nil
}

// handleDailySettlementReport processes daily settlement report generation jobs
func (s *Server) handleDailySettlementReport(ctx context.Context, task *asynq.Task) error {
	var payload DailySettlementReportPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		// If payload is empty (from scheduled task), use yesterday's date
		payload.Date = time.Now().AddDate(0, 0, -1)
	}

	logger.Info("Processing daily settlement report", logger.Fields{
		"date": payload.Date.Format("2006-01-02"),
	})

	// Get report date range (start and end of day)
	startDate := time.Date(payload.Date.Year(), payload.Date.Month(), payload.Date.Day(), 0, 0, 0, 0, payload.Date.Location())
	endDate := startDate.AddDate(0, 0, 1)

	// Get payment statistics for the day
	// TODO: Implement GetByDateRange in payment repository
	// For now, using placeholder data
	var totalPayments int
	var completedPayments int
	var totalAmount decimal.Decimal
	var totalFees decimal.Decimal

	_ = startDate // Suppress unused variable warning
	_ = endDate   // Suppress unused variable warning

	// payments, err := s.paymentRepo.GetByDateRange(ctx, startDate, endDate)
	// if err != nil {
	// 	return fmt.Errorf("failed to get payments: %w", err)
	// }
	// for _, payment := range payments {
	// 	totalPayments++
	// 	if payment.Status == "completed" {
	// 		completedPayments++
	// 		totalAmount = totalAmount.Add(payment.AmountVND)
	// 		// Assuming 1% fee
	// 		fee := payment.AmountVND.Mul(decimal.NewFromFloat(0.01))
	// 		totalFees = totalFees.Add(fee)
	// 	}
	// }

	// Get payout statistics for the day
	// TODO: Implement GetByDateRange in payout repository
	// For now, using placeholder data
	var totalPayouts int
	var completedPayouts int
	var totalPayoutAmount decimal.Decimal

	// payouts, err := s.payoutRepo.GetByDateRange(ctx, startDate, endDate)
	// if err != nil {
	// 	return fmt.Errorf("failed to get payouts: %w", err)
	// }
	// for _, payout := range payouts {
	// 	totalPayouts++
	// 	if payout.Status == "completed" {
	// 		completedPayouts++
	// 		totalPayoutAmount = totalPayoutAmount.Add(payout.Amount)
	// 	}
	// }

	// Get wallet balance
	var walletBalance decimal.Decimal
	if s.solanaWallet != nil {
		walletAddress := s.solanaWallet.GetAddress()
		latestBalance, err := s.walletBalanceRepo.GetLatest(ctx, walletAddress)
		if err != nil {
			logger.Warn("Failed to get latest wallet balance", logger.Fields{
				"error": err.Error(),
			})
		} else {
			walletBalance = latestBalance.TotalUSDValue
		}
	}

	// Prepare report data
	reportData := map[string]interface{}{
		"date":               payload.Date.Format("2006-01-02"),
		"total_payments":     totalPayments,
		"completed_payments": completedPayments,
		"total_amount_vnd":   totalAmount.String(),
		"total_fees_vnd":     totalFees.String(),
		"total_payouts":      totalPayouts,
		"completed_payouts":  completedPayouts,
		"total_payout_vnd":   totalPayoutAmount.String(),
		"wallet_balance_usd": walletBalance.String(),
		"net_revenue_vnd":    totalFees.String(),
	}

	logger.Info("Daily settlement report generated", logger.Fields{
		"date":               payload.Date.Format("2006-01-02"),
		"total_payments":     totalPayments,
		"completed_payments": completedPayments,
		"total_payouts":      totalPayouts,
		"completed_payouts":  completedPayouts,
	})

	// TODO: Send report via email to ops team
	// For now, just log the report
	logger.Info("Daily settlement report", reportData)

	return nil
}
