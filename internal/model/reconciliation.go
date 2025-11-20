package model

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// ReconciliationLog represents a daily reconciliation check result
// This verifies that the system is solvent: Assets >= Liabilities
type ReconciliationLog struct {
	ID string `json:"id" db:"id"`

	// Financial totals
	TotalAssetsVND      decimal.Decimal `json:"total_assets_vnd" db:"total_assets_vnd"`
	TotalLiabilitiesVND decimal.Decimal `json:"total_liabilities_vnd" db:"total_liabilities_vnd"`
	DifferenceVND       decimal.Decimal `json:"difference_vnd" db:"difference_vnd"` // Assets - Liabilities

	// Detailed breakdown of assets (stored as JSONB)
	AssetBreakdown map[string]string `json:"asset_breakdown,omitempty" db:"asset_breakdown"`

	// Status: balanced, surplus, deficit, error, in_progress
	Status string `json:"status" db:"status" validate:"required,oneof=balanced surplus deficit error in_progress"`

	// Alert flag - set to true if deficit detected
	AlertTriggered bool `json:"alert_triggered" db:"alert_triggered"`

	// Error message if reconciliation failed
	ErrorMessage sql.NullString `json:"error_message,omitempty" db:"error_message"`

	// Timestamps
	ReconciledAt time.Time `json:"reconciled_at" db:"reconciled_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// IsDeficit returns true if the system has a deficit (insolvent)
func (r *ReconciliationLog) IsDeficit() bool {
	return r.Status == "deficit"
}

// IsSolvent returns true if the system is solvent (balanced or surplus)
func (r *ReconciliationLog) IsSolvent() bool {
	return r.Status == "balanced" || r.Status == "surplus"
}

// GetDeficitAmount returns the absolute deficit amount (positive number)
// Returns zero if not in deficit
func (r *ReconciliationLog) GetDeficitAmount() decimal.Decimal {
	if r.IsDeficit() && r.DifferenceVND.LessThan(decimal.Zero) {
		return r.DifferenceVND.Abs()
	}
	return decimal.Zero
}

// GetSurplusAmount returns the surplus amount (positive number)
// Returns zero if not in surplus
func (r *ReconciliationLog) GetSurplusAmount() decimal.Decimal {
	if r.Status == "surplus" && r.DifferenceVND.GreaterThan(decimal.Zero) {
		return r.DifferenceVND
	}
	return decimal.Zero
}
