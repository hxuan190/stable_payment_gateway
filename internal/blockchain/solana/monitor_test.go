package solana

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/hxuan190/stable_payment_gateway/internal/config"
)

func TestDefaultWalletMonitorConfig(t *testing.T) {
	cfg := DefaultWalletMonitorConfig()

	assert.Equal(t, 5*time.Minute, cfg.CheckInterval)
	assert.Equal(t, decimal.NewFromInt(1000), cfg.MinThresholdUSD)
	assert.Equal(t, decimal.NewFromInt(10000), cfg.MaxThresholdUSD)
	assert.Equal(t, []string{}, cfg.AlertEmails)
}

func TestNewWalletMonitor(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Load wallet for testing
	cfg, err := config.Load()
	assert.NoError(t, err)

	if cfg.Solana.WalletPrivateKey == "" {
		t.Skip("SOLANA_WALLET_PRIVATE_KEY not set")
	}

	wallet, err := LoadWallet(cfg.Solana.WalletPrivateKey, cfg.Solana.RPCURL)
	assert.NoError(t, err)
	assert.NotNil(t, wallet)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	monitorConfig := DefaultWalletMonitorConfig()
	monitor := NewWalletMonitor(wallet, monitorConfig, logger, nil, nil, cfg.Solana)

	assert.NotNil(t, monitor)
	assert.Equal(t, wallet, monitor.wallet)
	assert.Equal(t, monitorConfig.CheckInterval, monitor.config.CheckInterval)
	assert.Equal(t, logger, monitor.logger)
}

func TestCheckBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Load wallet for testing
	cfg, err := config.Load()
	assert.NoError(t, err)

	if cfg.Solana.WalletPrivateKey == "" {
		t.Skip("SOLANA_WALLET_PRIVATE_KEY not set")
	}

	wallet, err := LoadWallet(cfg.Solana.WalletPrivateKey, cfg.Solana.RPCURL)
	assert.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	monitorConfig := DefaultWalletMonitorConfig()
	monitor := NewWalletMonitor(wallet, monitorConfig, logger, nil, nil, cfg.Solana)

	ctx := context.Background()
	balance, err := monitor.CheckBalance(ctx)
	assert.NoError(t, err)
	assert.True(t, balance.GreaterThanOrEqual(decimal.Zero))

	t.Logf("Wallet balance: %s USD", balance.String())
}

func TestWalletMonitorThresholds(t *testing.T) {
	tests := []struct {
		name            string
		totalUSDValue   decimal.Decimal
		minThreshold    decimal.Decimal
		maxThreshold    decimal.Decimal
		expectBelowMin  bool
		expectAboveMax  bool
	}{
		{
			name:           "Balance within normal range",
			totalUSDValue:  decimal.NewFromInt(5000),
			minThreshold:   decimal.NewFromInt(1000),
			maxThreshold:   decimal.NewFromInt(10000),
			expectBelowMin: false,
			expectAboveMax: false,
		},
		{
			name:           "Balance below minimum",
			totalUSDValue:  decimal.NewFromInt(500),
			minThreshold:   decimal.NewFromInt(1000),
			maxThreshold:   decimal.NewFromInt(10000),
			expectBelowMin: true,
			expectAboveMax: false,
		},
		{
			name:           "Balance above maximum",
			totalUSDValue:  decimal.NewFromInt(15000),
			minThreshold:   decimal.NewFromInt(1000),
			maxThreshold:   decimal.NewFromInt(10000),
			expectBelowMin: false,
			expectAboveMax: true,
		},
		{
			name:           "Balance at minimum threshold",
			totalUSDValue:  decimal.NewFromInt(1000),
			minThreshold:   decimal.NewFromInt(1000),
			maxThreshold:   decimal.NewFromInt(10000),
			expectBelowMin: false,
			expectAboveMax: false,
		},
		{
			name:           "Balance at maximum threshold",
			totalUSDValue:  decimal.NewFromInt(10000),
			minThreshold:   decimal.NewFromInt(1000),
			maxThreshold:   decimal.NewFromInt(10000),
			expectBelowMin: false,
			expectAboveMax: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isBelowMin := tt.totalUSDValue.LessThan(tt.minThreshold)
			isAboveMax := tt.totalUSDValue.GreaterThan(tt.maxThreshold)

			assert.Equal(t, tt.expectBelowMin, isBelowMin, "Below minimum check failed")
			assert.Equal(t, tt.expectAboveMax, isAboveMax, "Above maximum check failed")
		})
	}
}

func TestWalletMonitorStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Load wallet for testing
	cfg, err := config.Load()
	assert.NoError(t, err)

	if cfg.Solana.WalletPrivateKey == "" {
		t.Skip("SOLANA_WALLET_PRIVATE_KEY not set")
	}

	wallet, err := LoadWallet(cfg.Solana.WalletPrivateKey, cfg.Solana.RPCURL)
	assert.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Use a very short check interval for testing
	monitorConfig := DefaultWalletMonitorConfig()
	monitorConfig.CheckInterval = 1 * time.Second

	monitor := NewWalletMonitor(wallet, monitorConfig, logger, nil, nil, cfg.Solana)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start monitor in goroutine
	go monitor.Start(ctx)

	// Let it run for a bit
	time.Sleep(2 * time.Second)

	// Stop monitor
	monitor.Stop()

	// Ensure it stopped
	select {
	case <-monitor.stoppedChan:
		// Successfully stopped
	case <-time.After(2 * time.Second):
		t.Fatal("Monitor did not stop in time")
	}
}
