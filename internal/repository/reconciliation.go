package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

var (
	// ErrReconciliationNotFound is returned when a reconciliation log is not found
	ErrReconciliationNotFound = errors.New("reconciliation log not found")
)

// ReconciliationRepository handles database operations for reconciliation logs
type ReconciliationRepository struct {
	db *sql.DB
}

// NewReconciliationRepository creates a new reconciliation repository
func NewReconciliationRepository(db *sql.DB) *ReconciliationRepository {
	return &ReconciliationRepository{
		db: db,
	}
}

// Create creates a new reconciliation log
func (r *ReconciliationRepository) Create(log *model.ReconciliationLog) error {
	if log == nil {
		return errors.New("reconciliation log cannot be nil")
	}

	// Generate ID if not set
	if log.ID == "" {
		log.ID = uuid.New().String()
	}

	// Set created_at if not set
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	// Serialize asset breakdown to JSON
	var assetBreakdownJSON []byte
	var err error
	if log.AssetBreakdown != nil {
		assetBreakdownJSON, err = json.Marshal(log.AssetBreakdown)
		if err != nil {
			return fmt.Errorf("failed to marshal asset breakdown: %w", err)
		}
	}

	query := `
		INSERT INTO reconciliation_logs (
			id,
			total_assets_vnd,
			total_liabilities_vnd,
			difference_vnd,
			asset_breakdown,
			status,
			alert_triggered,
			error_message,
			reconciled_at,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.db.Exec(query,
		log.ID,
		log.TotalAssetsVND,
		log.TotalLiabilitiesVND,
		log.DifferenceVND,
		assetBreakdownJSON,
		log.Status,
		log.AlertTriggered,
		log.ErrorMessage,
		log.ReconciledAt,
		log.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create reconciliation log: %w", err)
	}

	return nil
}

// GetByID retrieves a reconciliation log by ID
func (r *ReconciliationRepository) GetByID(id string) (*model.ReconciliationLog, error) {
	query := `
		SELECT
			id,
			total_assets_vnd,
			total_liabilities_vnd,
			difference_vnd,
			asset_breakdown,
			status,
			alert_triggered,
			error_message,
			reconciled_at,
			created_at
		FROM reconciliation_logs
		WHERE id = $1
	`

	log := &model.ReconciliationLog{}
	var assetBreakdownJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&log.ID,
		&log.TotalAssetsVND,
		&log.TotalLiabilitiesVND,
		&log.DifferenceVND,
		&assetBreakdownJSON,
		&log.Status,
		&log.AlertTriggered,
		&log.ErrorMessage,
		&log.ReconciledAt,
		&log.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrReconciliationNotFound
		}
		return nil, fmt.Errorf("failed to get reconciliation log: %w", err)
	}

	// Deserialize asset breakdown
	if assetBreakdownJSON != nil {
		if err := json.Unmarshal(assetBreakdownJSON, &log.AssetBreakdown); err != nil {
			return nil, fmt.Errorf("failed to unmarshal asset breakdown: %w", err)
		}
	}

	return log, nil
}

// GetLatest retrieves the most recent reconciliation log
func (r *ReconciliationRepository) GetLatest() (*model.ReconciliationLog, error) {
	query := `
		SELECT
			id,
			total_assets_vnd,
			total_liabilities_vnd,
			difference_vnd,
			asset_breakdown,
			status,
			alert_triggered,
			error_message,
			reconciled_at,
			created_at
		FROM reconciliation_logs
		ORDER BY reconciled_at DESC
		LIMIT 1
	`

	log := &model.ReconciliationLog{}
	var assetBreakdownJSON []byte

	err := r.db.QueryRow(query).Scan(
		&log.ID,
		&log.TotalAssetsVND,
		&log.TotalLiabilitiesVND,
		&log.DifferenceVND,
		&assetBreakdownJSON,
		&log.Status,
		&log.AlertTriggered,
		&log.ErrorMessage,
		&log.ReconciledAt,
		&log.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrReconciliationNotFound
		}
		return nil, fmt.Errorf("failed to get latest reconciliation log: %w", err)
	}

	// Deserialize asset breakdown
	if assetBreakdownJSON != nil {
		if err := json.Unmarshal(assetBreakdownJSON, &log.AssetBreakdown); err != nil {
			return nil, fmt.Errorf("failed to unmarshal asset breakdown: %w", err)
		}
	}

	return log, nil
}

// GetByDateRange retrieves reconciliation logs within a date range
func (r *ReconciliationRepository) GetByDateRange(startDate, endDate time.Time, limit int) ([]*model.ReconciliationLog, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT
			id,
			total_assets_vnd,
			total_liabilities_vnd,
			difference_vnd,
			asset_breakdown,
			status,
			alert_triggered,
			error_message,
			reconciled_at,
			created_at
		FROM reconciliation_logs
		WHERE reconciled_at >= $1 AND reconciled_at <= $2
		ORDER BY reconciled_at DESC
		LIMIT $3
	`

	rows, err := r.db.Query(query, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query reconciliation logs: %w", err)
	}
	defer rows.Close()

	var logs []*model.ReconciliationLog

	for rows.Next() {
		log := &model.ReconciliationLog{}
		var assetBreakdownJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.TotalAssetsVND,
			&log.TotalLiabilitiesVND,
			&log.DifferenceVND,
			&assetBreakdownJSON,
			&log.Status,
			&log.AlertTriggered,
			&log.ErrorMessage,
			&log.ReconciledAt,
			&log.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan reconciliation log: %w", err)
		}

		// Deserialize asset breakdown
		if assetBreakdownJSON != nil {
			if err := json.Unmarshal(assetBreakdownJSON, &log.AssetBreakdown); err != nil {
				return nil, fmt.Errorf("failed to unmarshal asset breakdown: %w", err)
			}
		}

		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reconciliation logs: %w", err)
	}

	return logs, nil
}

// GetByStatus retrieves reconciliation logs by status
func (r *ReconciliationRepository) GetByStatus(status string, limit int) ([]*model.ReconciliationLog, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT
			id,
			total_assets_vnd,
			total_liabilities_vnd,
			difference_vnd,
			asset_breakdown,
			status,
			alert_triggered,
			error_message,
			reconciled_at,
			created_at
		FROM reconciliation_logs
		WHERE status = $1
		ORDER BY reconciled_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query reconciliation logs by status: %w", err)
	}
	defer rows.Close()

	var logs []*model.ReconciliationLog

	for rows.Next() {
		log := &model.ReconciliationLog{}
		var assetBreakdownJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.TotalAssetsVND,
			&log.TotalLiabilitiesVND,
			&log.DifferenceVND,
			&assetBreakdownJSON,
			&log.Status,
			&log.AlertTriggered,
			&log.ErrorMessage,
			&log.ReconciledAt,
			&log.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan reconciliation log: %w", err)
		}

		// Deserialize asset breakdown
		if assetBreakdownJSON != nil {
			if err := json.Unmarshal(assetBreakdownJSON, &log.AssetBreakdown); err != nil {
				return nil, fmt.Errorf("failed to unmarshal asset breakdown: %w", err)
			}
		}

		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reconciliation logs: %w", err)
	}

	return logs, nil
}

// GetDeficitAlerts retrieves all reconciliation logs where alert was triggered
func (r *ReconciliationRepository) GetDeficitAlerts(limit int) ([]*model.ReconciliationLog, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT
			id,
			total_assets_vnd,
			total_liabilities_vnd,
			difference_vnd,
			asset_breakdown,
			status,
			alert_triggered,
			error_message,
			reconciled_at,
			created_at
		FROM reconciliation_logs
		WHERE alert_triggered = true
		ORDER BY reconciled_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query deficit alerts: %w", err)
	}
	defer rows.Close()

	var logs []*model.ReconciliationLog

	for rows.Next() {
		log := &model.ReconciliationLog{}
		var assetBreakdownJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.TotalAssetsVND,
			&log.TotalLiabilitiesVND,
			&log.DifferenceVND,
			&assetBreakdownJSON,
			&log.Status,
			&log.AlertTriggered,
			&log.ErrorMessage,
			&log.ReconciledAt,
			&log.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan reconciliation log: %w", err)
		}

		// Deserialize asset breakdown
		if assetBreakdownJSON != nil {
			if err := json.Unmarshal(assetBreakdownJSON, &log.AssetBreakdown); err != nil {
				return nil, fmt.Errorf("failed to unmarshal asset breakdown: %w", err)
			}
		}

		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating deficit alerts: %w", err)
	}

	return logs, nil
}
