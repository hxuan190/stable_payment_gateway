package model

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// WalletBalanceSnapshot represents a point-in-time snapshot of wallet balances
// Used for monitoring and tracking hot wallet balance over time
type WalletBalanceSnapshot struct {
	ID string `json:"id" db:"id"`

	// Blockchain details
	Chain   Chain   `json:"chain" db:"chain" validate:"required,oneof=solana bsc ethereum"`
	Network Network `json:"network" db:"network" validate:"required,oneof=mainnet testnet devnet"`

	// Wallet information
	WalletAddress string `json:"wallet_address" db:"wallet_address" validate:"required"`

	// Native token balance (SOL, BNB, ETH)
	NativeBalance  decimal.Decimal `json:"native_balance" db:"native_balance" validate:"gte=0"`
	NativeCurrency string          `json:"native_currency" db:"native_currency" validate:"required"`

	// USDT balance
	USDTBalance decimal.Decimal `json:"usdt_balance" db:"usdt_balance" validate:"gte=0"`
	USDTMint    sql.NullString  `json:"usdt_mint,omitempty" db:"usdt_mint"`

	// USDC balance
	USDCBalance decimal.Decimal `json:"usdc_balance" db:"usdc_balance" validate:"gte=0"`
	USDCMint    sql.NullString  `json:"usdc_mint,omitempty" db:"usdc_mint"`

	// Total USD value (approximate)
	TotalUSDValue decimal.Decimal `json:"total_usd_value" db:"total_usd_value" validate:"gte=0"`

	// Alert thresholds
	MinThresholdUSD decimal.Decimal `json:"min_threshold_usd" db:"min_threshold_usd"`
	MaxThresholdUSD decimal.Decimal `json:"max_threshold_usd" db:"max_threshold_usd"`

	// Alert status
	IsBelowMinThreshold bool         `json:"is_below_min_threshold" db:"is_below_min_threshold"`
	IsAboveMaxThreshold bool         `json:"is_above_max_threshold" db:"is_above_max_threshold"`
	AlertSent           bool         `json:"alert_sent" db:"alert_sent"`
	AlertSentAt         sql.NullTime `json:"alert_sent_at,omitempty" db:"alert_sent_at"`

	// Metadata
	Metadata JSONBMap `json:"metadata,omitempty" db:"metadata"`

	// Timestamps
	SnapshotAt time.Time `json:"snapshot_at" db:"snapshot_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// ShouldAlert returns true if an alert should be sent based on thresholds
func (w *WalletBalanceSnapshot) ShouldAlert() bool {
	return (w.IsBelowMinThreshold || w.IsAboveMaxThreshold) && !w.AlertSent
}

// GetAlertMessage returns a human-readable alert message
func (w *WalletBalanceSnapshot) GetAlertMessage() string {
	if w.IsBelowMinThreshold {
		return "ALERT: Wallet balance is below minimum threshold"
	}
	if w.IsAboveMaxThreshold {
		return "ALERT: Wallet balance is above maximum threshold"
	}
	return ""
}

// GetAlertDetails returns detailed alert information
func (w *WalletBalanceSnapshot) GetAlertDetails() map[string]interface{} {
	return map[string]interface{}{
		"chain":            w.Chain,
		"network":          w.Network,
		"wallet_address":   w.WalletAddress,
		"native_balance":   w.NativeBalance.String(),
		"native_currency":  w.NativeCurrency,
		"usdt_balance":     w.USDTBalance.String(),
		"usdc_balance":     w.USDCBalance.String(),
		"total_usd_value":  w.TotalUSDValue.String(),
		"min_threshold":    w.MinThresholdUSD.String(),
		"max_threshold":    w.MaxThresholdUSD.String(),
		"below_threshold":  w.IsBelowMinThreshold,
		"above_threshold":  w.IsAboveMaxThreshold,
		"snapshot_time":    w.SnapshotAt.Format(time.RFC3339),
	}
}
