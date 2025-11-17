package solana

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/hxuan190/stable_payment_gateway/internal/config"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
)

// WalletMonitorConfig holds configuration for wallet balance monitoring
type WalletMonitorConfig struct {
	// Check interval (default: 5 minutes)
	CheckInterval time.Duration

	// Alert thresholds in USD
	MinThresholdUSD decimal.Decimal
	MaxThresholdUSD decimal.Decimal

	// Alert email recipients
	AlertEmails []string

	// Network configuration
	Network model.Network
}

// DefaultWalletMonitorConfig returns default monitoring configuration
func DefaultWalletMonitorConfig() WalletMonitorConfig {
	return WalletMonitorConfig{
		CheckInterval:   5 * time.Minute,
		MinThresholdUSD: decimal.NewFromInt(1000),  // $1,000 minimum
		MaxThresholdUSD: decimal.NewFromInt(10000), // $10,000 maximum
		AlertEmails:     []string{},
		Network:         model.NetworkDevnet,
	}
}

// WalletMonitor monitors hot wallet balances and sends alerts when thresholds are breached
type WalletMonitor struct {
	wallet                *Wallet
	config                WalletMonitorConfig
	logger                *logrus.Logger
	notificationService   *service.NotificationService
	balanceRepo           *repository.WalletBalanceRepository
	solanaConfig          config.SolanaConfig
	stopChan              chan struct{}
	stoppedChan           chan struct{}
}

// NewWalletMonitor creates a new wallet balance monitor
func NewWalletMonitor(
	wallet *Wallet,
	config WalletMonitorConfig,
	logger *logrus.Logger,
	notificationService *service.NotificationService,
	balanceRepo *repository.WalletBalanceRepository,
	solanaConfig config.SolanaConfig,
) *WalletMonitor {
	if logger == nil {
		logger = logrus.New()
	}

	return &WalletMonitor{
		wallet:              wallet,
		config:              config,
		logger:              logger,
		notificationService: notificationService,
		balanceRepo:         balanceRepo,
		solanaConfig:        solanaConfig,
		stopChan:            make(chan struct{}),
		stoppedChan:         make(chan struct{}),
	}
}

// Start begins monitoring wallet balance at configured intervals
func (m *WalletMonitor) Start(ctx context.Context) {
	m.logger.WithFields(logrus.Fields{
		"check_interval": m.config.CheckInterval,
		"min_threshold":  m.config.MinThresholdUSD.String(),
		"max_threshold":  m.config.MaxThresholdUSD.String(),
		"wallet_address": m.wallet.GetAddress(),
	}).Info("Starting wallet balance monitor")

	// Perform initial check
	if err := m.checkBalance(ctx); err != nil {
		m.logger.WithError(err).Error("Initial balance check failed")
	}

	// Start monitoring loop
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.checkBalance(ctx); err != nil {
				m.logger.WithError(err).Error("Balance check failed")
			}

		case <-m.stopChan:
			m.logger.Info("Wallet balance monitor stopped")
			close(m.stoppedChan)
			return

		case <-ctx.Done():
			m.logger.Info("Wallet balance monitor context cancelled")
			close(m.stoppedChan)
			return
		}
	}
}

// Stop gracefully stops the wallet monitor
func (m *WalletMonitor) Stop() {
	m.logger.Info("Stopping wallet balance monitor...")
	close(m.stopChan)
	<-m.stoppedChan
}

// checkBalance checks the current wallet balance and sends alerts if needed
func (m *WalletMonitor) checkBalance(ctx context.Context) error {
	startTime := time.Now()

	m.logger.Debug("Checking wallet balance")

	// Get wallet balances
	tokenMints := map[string]string{
		"USDT": m.solanaConfig.USDTMint,
		"USDC": m.solanaConfig.USDCMint,
	}

	walletBalance, err := m.wallet.GetWalletBalance(ctx, tokenMints)
	if err != nil {
		return fmt.Errorf("failed to get wallet balance: %w", err)
	}

	// Calculate total USD value
	totalUSDValue := decimal.Zero

	// Get USDT balance
	usdtBalance := decimal.Zero
	if usdtInfo, ok := walletBalance.Tokens["USDT"]; ok {
		usdtBalance = usdtInfo.Balance
		totalUSDValue = totalUSDValue.Add(usdtBalance)
	}

	// Get USDC balance
	usdcBalance := decimal.Zero
	if usdcInfo, ok := walletBalance.Tokens["USDC"]; ok {
		usdcBalance = usdcInfo.Balance
		totalUSDValue = totalUSDValue.Add(usdcBalance)
	}

	// For SOL, we'd need to fetch the current price, but for MVP we'll skip it
	// In production, integrate with price API to convert SOL to USD
	solBalance := walletBalance.SOL

	// Determine if alerts should be triggered
	isBelowMin := totalUSDValue.LessThan(m.config.MinThresholdUSD)
	isAboveMax := totalUSDValue.GreaterThan(m.config.MaxThresholdUSD)

	// Create snapshot
	snapshot := &model.WalletBalanceSnapshot{
		Chain:               model.ChainSolana,
		Network:             m.config.Network,
		WalletAddress:       m.wallet.GetAddress(),
		NativeBalance:       solBalance,
		NativeCurrency:      "SOL",
		USDTBalance:         usdtBalance,
		USDTMint:            sql.NullString{String: m.solanaConfig.USDTMint, Valid: true},
		USDCBalance:         usdcBalance,
		USDCMint:            sql.NullString{String: m.solanaConfig.USDCMint, Valid: true},
		TotalUSDValue:       totalUSDValue,
		MinThresholdUSD:     m.config.MinThresholdUSD,
		MaxThresholdUSD:     m.config.MaxThresholdUSD,
		IsBelowMinThreshold: isBelowMin,
		IsAboveMaxThreshold: isAboveMax,
		AlertSent:           false,
		SnapshotAt:          time.Now().UTC(),
	}

	// Save snapshot to database
	if err := m.balanceRepo.Create(ctx, snapshot); err != nil {
		m.logger.WithError(err).Error("Failed to save balance snapshot")
		// Continue with alerting even if save fails
	}

	// Log balance information
	m.logger.WithFields(logrus.Fields{
		"wallet_address":  m.wallet.GetAddress(),
		"sol_balance":     solBalance.String(),
		"usdt_balance":    usdtBalance.String(),
		"usdc_balance":    usdcBalance.String(),
		"total_usd_value": totalUSDValue.String(),
		"min_threshold":   m.config.MinThresholdUSD.String(),
		"max_threshold":   m.config.MaxThresholdUSD.String(),
		"below_min":       isBelowMin,
		"above_max":       isAboveMax,
		"duration_ms":     time.Since(startTime).Milliseconds(),
	}).Info("Wallet balance checked")

	// Send alerts if thresholds are breached
	if snapshot.ShouldAlert() {
		if err := m.sendAlert(ctx, snapshot); err != nil {
			m.logger.WithError(err).Error("Failed to send balance alert")
			return err
		}

		// Mark alert as sent
		if err := m.balanceRepo.MarkAlertSent(ctx, snapshot.ID); err != nil {
			m.logger.WithError(err).Error("Failed to mark alert as sent")
		}
	}

	return nil
}

// sendAlert sends alert notifications when balance thresholds are breached
func (m *WalletMonitor) sendAlert(ctx context.Context, snapshot *model.WalletBalanceSnapshot) error {
	alertMessage := snapshot.GetAlertMessage()
	alertDetails := snapshot.GetAlertDetails()

	m.logger.WithFields(logrus.Fields{
		"message": alertMessage,
		"details": alertDetails,
	}).Warn("Wallet balance alert triggered")

	// Send email alerts to configured recipients
	for _, email := range m.config.AlertEmails {
		err := m.notificationService.SendEmail(
			ctx,
			"wallet_balance_alert",
			email,
			map[string]interface{}{
				"message":         alertMessage,
				"details":         alertDetails,
				"wallet_address":  snapshot.WalletAddress,
				"total_usd_value": snapshot.TotalUSDValue.String(),
				"sol_balance":     snapshot.NativeBalance.String(),
				"usdt_balance":    snapshot.USDTBalance.String(),
				"usdc_balance":    snapshot.USDCBalance.String(),
				"min_threshold":   snapshot.MinThresholdUSD.String(),
				"max_threshold":   snapshot.MaxThresholdUSD.String(),
				"snapshot_time":   snapshot.SnapshotAt.Format(time.RFC3339),
			},
		)

		if err != nil {
			m.logger.WithError(err).WithField("email", email).Error("Failed to send email alert")
		}
	}

	return nil
}

// CheckBalance performs a one-time balance check and returns the total USD value
// This is useful for manual checks or API endpoints
func (m *WalletMonitor) CheckBalance(ctx context.Context) (decimal.Decimal, error) {
	tokenMints := map[string]string{
		"USDT": m.solanaConfig.USDTMint,
		"USDC": m.solanaConfig.USDCMint,
	}

	walletBalance, err := m.wallet.GetWalletBalance(ctx, tokenMints)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get wallet balance: %w", err)
	}

	// Calculate total USD value (USDT + USDC)
	totalUSDValue := decimal.Zero

	if usdtInfo, ok := walletBalance.Tokens["USDT"]; ok {
		totalUSDValue = totalUSDValue.Add(usdtInfo.Balance)
	}

	if usdcInfo, ok := walletBalance.Tokens["USDC"]; ok {
		totalUSDValue = totalUSDValue.Add(usdcInfo.Balance)
	}

	return totalUSDValue, nil
}

// GetLatestSnapshot retrieves the most recent balance snapshot from the database
func (m *WalletMonitor) GetLatestSnapshot(ctx context.Context) (*model.WalletBalanceSnapshot, error) {
	return m.balanceRepo.GetLatest(ctx, model.ChainSolana, m.wallet.GetAddress())
}

// GetBalanceHistory retrieves historical balance snapshots
func (m *WalletMonitor) GetBalanceHistory(ctx context.Context, startTime, endTime time.Time, limit int) ([]*model.WalletBalanceSnapshot, error) {
	return m.balanceRepo.GetHistory(ctx, model.ChainSolana, m.wallet.GetAddress(), startTime, endTime, limit)
}
