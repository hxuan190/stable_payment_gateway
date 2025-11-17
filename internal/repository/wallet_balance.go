package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

// WalletBalanceRepository handles database operations for wallet balance snapshots
type WalletBalanceRepository struct {
	db *sql.DB
}

// NewWalletBalanceRepository creates a new wallet balance repository
func NewWalletBalanceRepository(db *sql.DB) *WalletBalanceRepository {
	return &WalletBalanceRepository{
		db: db,
	}
}

// Create saves a new wallet balance snapshot
func (r *WalletBalanceRepository) Create(ctx context.Context, snapshot *model.WalletBalanceSnapshot) error {
	if snapshot.ID == "" {
		snapshot.ID = uuid.New().String()
	}

	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = time.Now().UTC()
	}

	if snapshot.SnapshotAt.IsZero() {
		snapshot.SnapshotAt = time.Now().UTC()
	}

	query := `
		INSERT INTO wallet_balance_snapshots (
			id, chain, network, wallet_address,
			native_balance, native_currency,
			usdt_balance, usdt_mint,
			usdc_balance, usdc_mint,
			total_usd_value,
			min_threshold_usd, max_threshold_usd,
			is_below_min_threshold, is_above_max_threshold,
			alert_sent, alert_sent_at,
			metadata, snapshot_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		snapshot.ID,
		snapshot.Chain,
		snapshot.Network,
		snapshot.WalletAddress,
		snapshot.NativeBalance,
		snapshot.NativeCurrency,
		snapshot.USDTBalance,
		snapshot.USDTMint,
		snapshot.USDCBalance,
		snapshot.USDCMint,
		snapshot.TotalUSDValue,
		snapshot.MinThresholdUSD,
		snapshot.MaxThresholdUSD,
		snapshot.IsBelowMinThreshold,
		snapshot.IsAboveMaxThreshold,
		snapshot.AlertSent,
		snapshot.AlertSentAt,
		snapshot.Metadata,
		snapshot.SnapshotAt,
		snapshot.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create wallet balance snapshot: %w", err)
	}

	return nil
}

// GetLatest retrieves the most recent snapshot for a wallet
func (r *WalletBalanceRepository) GetLatest(ctx context.Context, chain model.Chain, walletAddress string) (*model.WalletBalanceSnapshot, error) {
	query := `
		SELECT
			id, chain, network, wallet_address,
			native_balance, native_currency,
			usdt_balance, usdt_mint,
			usdc_balance, usdc_mint,
			total_usd_value,
			min_threshold_usd, max_threshold_usd,
			is_below_min_threshold, is_above_max_threshold,
			alert_sent, alert_sent_at,
			metadata, snapshot_at, created_at
		FROM wallet_balance_snapshots
		WHERE chain = $1 AND wallet_address = $2
		ORDER BY snapshot_at DESC
		LIMIT 1
	`

	snapshot := &model.WalletBalanceSnapshot{}
	err := r.db.QueryRowContext(ctx, query, chain, walletAddress).Scan(
		&snapshot.ID,
		&snapshot.Chain,
		&snapshot.Network,
		&snapshot.WalletAddress,
		&snapshot.NativeBalance,
		&snapshot.NativeCurrency,
		&snapshot.USDTBalance,
		&snapshot.USDTMint,
		&snapshot.USDCBalance,
		&snapshot.USDCMint,
		&snapshot.TotalUSDValue,
		&snapshot.MinThresholdUSD,
		&snapshot.MaxThresholdUSD,
		&snapshot.IsBelowMinThreshold,
		&snapshot.IsAboveMaxThreshold,
		&snapshot.AlertSent,
		&snapshot.AlertSentAt,
		&snapshot.Metadata,
		&snapshot.SnapshotAt,
		&snapshot.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get latest wallet balance snapshot: %w", err)
	}

	return snapshot, nil
}

// GetHistory retrieves balance snapshots for a wallet within a time range
func (r *WalletBalanceRepository) GetHistory(ctx context.Context, chain model.Chain, walletAddress string, startTime, endTime time.Time, limit int) ([]*model.WalletBalanceSnapshot, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT
			id, chain, network, wallet_address,
			native_balance, native_currency,
			usdt_balance, usdt_mint,
			usdc_balance, usdc_mint,
			total_usd_value,
			min_threshold_usd, max_threshold_usd,
			is_below_min_threshold, is_above_max_threshold,
			alert_sent, alert_sent_at,
			metadata, snapshot_at, created_at
		FROM wallet_balance_snapshots
		WHERE chain = $1
			AND wallet_address = $2
			AND snapshot_at >= $3
			AND snapshot_at <= $4
		ORDER BY snapshot_at DESC
		LIMIT $5
	`

	rows, err := r.db.QueryContext(ctx, query, chain, walletAddress, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet balance history: %w", err)
	}
	defer rows.Close()

	snapshots := []*model.WalletBalanceSnapshot{}
	for rows.Next() {
		snapshot := &model.WalletBalanceSnapshot{}
		err := rows.Scan(
			&snapshot.ID,
			&snapshot.Chain,
			&snapshot.Network,
			&snapshot.WalletAddress,
			&snapshot.NativeBalance,
			&snapshot.NativeCurrency,
			&snapshot.USDTBalance,
			&snapshot.USDTMint,
			&snapshot.USDCBalance,
			&snapshot.USDCMint,
			&snapshot.TotalUSDValue,
			&snapshot.MinThresholdUSD,
			&snapshot.MaxThresholdUSD,
			&snapshot.IsBelowMinThreshold,
			&snapshot.IsAboveMaxThreshold,
			&snapshot.AlertSent,
			&snapshot.AlertSentAt,
			&snapshot.Metadata,
			&snapshot.SnapshotAt,
			&snapshot.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wallet balance snapshot: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating wallet balance snapshots: %w", err)
	}

	return snapshots, nil
}

// MarkAlertSent updates a snapshot to indicate an alert was sent
func (r *WalletBalanceRepository) MarkAlertSent(ctx context.Context, snapshotID string) error {
	query := `
		UPDATE wallet_balance_snapshots
		SET alert_sent = TRUE, alert_sent_at = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, time.Now().UTC(), snapshotID)
	if err != nil {
		return fmt.Errorf("failed to mark alert sent: %w", err)
	}

	return nil
}

// GetPendingAlerts retrieves snapshots that require alerts but haven't sent them yet
func (r *WalletBalanceRepository) GetPendingAlerts(ctx context.Context) ([]*model.WalletBalanceSnapshot, error) {
	query := `
		SELECT
			id, chain, network, wallet_address,
			native_balance, native_currency,
			usdt_balance, usdt_mint,
			usdc_balance, usdc_mint,
			total_usd_value,
			min_threshold_usd, max_threshold_usd,
			is_below_min_threshold, is_above_max_threshold,
			alert_sent, alert_sent_at,
			metadata, snapshot_at, created_at
		FROM wallet_balance_snapshots
		WHERE (is_below_min_threshold = TRUE OR is_above_max_threshold = TRUE)
			AND alert_sent = FALSE
		ORDER BY snapshot_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending alerts: %w", err)
	}
	defer rows.Close()

	snapshots := []*model.WalletBalanceSnapshot{}
	for rows.Next() {
		snapshot := &model.WalletBalanceSnapshot{}
		err := rows.Scan(
			&snapshot.ID,
			&snapshot.Chain,
			&snapshot.Network,
			&snapshot.WalletAddress,
			&snapshot.NativeBalance,
			&snapshot.NativeCurrency,
			&snapshot.USDTBalance,
			&snapshot.USDTMint,
			&snapshot.USDCBalance,
			&snapshot.USDCMint,
			&snapshot.TotalUSDValue,
			&snapshot.MinThresholdUSD,
			&snapshot.MaxThresholdUSD,
			&snapshot.IsBelowMinThreshold,
			&snapshot.IsAboveMaxThreshold,
			&snapshot.AlertSent,
			&snapshot.AlertSentAt,
			&snapshot.Metadata,
			&snapshot.SnapshotAt,
			&snapshot.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wallet balance snapshot: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending alerts: %w", err)
	}

	return snapshots, nil
}
