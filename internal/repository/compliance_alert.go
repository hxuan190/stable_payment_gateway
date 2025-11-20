package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/jmoiron/sqlx"
)

// ComplianceAlertRepository handles database operations for compliance alerts
type ComplianceAlertRepository struct {
	db *sqlx.DB
}

// NewComplianceAlertRepository creates a new compliance alert repository
func NewComplianceAlertRepository(db *sqlx.DB) *ComplianceAlertRepository {
	return &ComplianceAlertRepository{db: db}
}

// Create creates a new compliance alert
func (r *ComplianceAlertRepository) Create(ctx context.Context, alert *model.ComplianceAlert) error {
	query := `
		INSERT INTO compliance_alerts (
			id, alert_type, severity, status,
			payment_id, merchant_id, payout_id,
			from_address, to_address, transaction_hash, blockchain,
			risk_score, risk_flags, details, evidence,
			actions_taken, recommended_action,
			assigned_to, reviewed_by, reviewed_at, resolution_notes,
			email_sent, email_sent_at, email_recipients,
			created_at, updated_at, resolved_at
		) VALUES (
			:id, :alert_type, :severity, :status,
			:payment_id, :merchant_id, :payout_id,
			:from_address, :to_address, :transaction_hash, :blockchain,
			:risk_score, :risk_flags, :details, :evidence,
			:actions_taken, :recommended_action,
			:assigned_to, :reviewed_by, :reviewed_at, :resolution_notes,
			:email_sent, :email_sent_at, :email_recipients,
			:created_at, :updated_at, :resolved_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, alert)
	if err != nil {
		return fmt.Errorf("failed to create compliance alert: %w", err)
	}

	return nil
}

// GetByID retrieves a compliance alert by ID
func (r *ComplianceAlertRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ComplianceAlert, error) {
	query := `
		SELECT * FROM compliance_alerts WHERE id = $1
	`

	var alert model.ComplianceAlert
	err := r.db.GetContext(ctx, &alert, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("compliance alert not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get compliance alert: %w", err)
	}

	return &alert, nil
}

// GetByPaymentID retrieves all compliance alerts for a payment
func (r *ComplianceAlertRepository) GetByPaymentID(ctx context.Context, paymentID uuid.UUID) ([]*model.ComplianceAlert, error) {
	query := `
		SELECT * FROM compliance_alerts
		WHERE payment_id = $1
		ORDER BY created_at DESC
	`

	var alerts []*model.ComplianceAlert
	err := r.db.SelectContext(ctx, &alerts, query, paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get compliance alerts by payment: %w", err)
	}

	return alerts, nil
}

// GetByMerchantID retrieves all compliance alerts for a merchant
func (r *ComplianceAlertRepository) GetByMerchantID(ctx context.Context, merchantID uuid.UUID) ([]*model.ComplianceAlert, error) {
	query := `
		SELECT * FROM compliance_alerts
		WHERE merchant_id = $1
		ORDER BY created_at DESC
	`

	var alerts []*model.ComplianceAlert
	err := r.db.SelectContext(ctx, &alerts, query, merchantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get compliance alerts by merchant: %w", err)
	}

	return alerts, nil
}

// GetByStatus retrieves compliance alerts by status
func (r *ComplianceAlertRepository) GetByStatus(ctx context.Context, status string, limit, offset int) ([]*model.ComplianceAlert, error) {
	query := `
		SELECT * FROM compliance_alerts
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var alerts []*model.ComplianceAlert
	err := r.db.SelectContext(ctx, &alerts, query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get compliance alerts by status: %w", err)
	}

	return alerts, nil
}

// GetBySeverity retrieves compliance alerts by severity
func (r *ComplianceAlertRepository) GetBySeverity(ctx context.Context, severity string, limit, offset int) ([]*model.ComplianceAlert, error) {
	query := `
		SELECT * FROM compliance_alerts
		WHERE severity = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var alerts []*model.ComplianceAlert
	err := r.db.SelectContext(ctx, &alerts, query, severity, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get compliance alerts by severity: %w", err)
	}

	return alerts, nil
}

// GetOpenAlerts retrieves all open compliance alerts
func (r *ComplianceAlertRepository) GetOpenAlerts(ctx context.Context, limit, offset int) ([]*model.ComplianceAlert, error) {
	return r.GetByStatus(ctx, model.AlertStatusOpen, limit, offset)
}

// GetCriticalAlerts retrieves all critical severity alerts
func (r *ComplianceAlertRepository) GetCriticalAlerts(ctx context.Context, limit, offset int) ([]*model.ComplianceAlert, error) {
	return r.GetBySeverity(ctx, model.SeverityCritical, limit, offset)
}

// Update updates a compliance alert
func (r *ComplianceAlertRepository) Update(ctx context.Context, alert *model.ComplianceAlert) error {
	query := `
		UPDATE compliance_alerts SET
			alert_type = :alert_type,
			severity = :severity,
			status = :status,
			payment_id = :payment_id,
			merchant_id = :merchant_id,
			payout_id = :payout_id,
			from_address = :from_address,
			to_address = :to_address,
			transaction_hash = :transaction_hash,
			blockchain = :blockchain,
			risk_score = :risk_score,
			risk_flags = :risk_flags,
			details = :details,
			evidence = :evidence,
			actions_taken = :actions_taken,
			recommended_action = :recommended_action,
			assigned_to = :assigned_to,
			reviewed_by = :reviewed_by,
			reviewed_at = :reviewed_at,
			resolution_notes = :resolution_notes,
			email_sent = :email_sent,
			email_sent_at = :email_sent_at,
			email_recipients = :email_recipients,
			updated_at = :updated_at,
			resolved_at = :resolved_at
		WHERE id = :id
	`

	result, err := r.db.NamedExecContext(ctx, query, alert)
	if err != nil {
		return fmt.Errorf("failed to update compliance alert: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("compliance alert not found: %s", alert.ID)
	}

	return nil
}

// Delete deletes a compliance alert (soft delete by updating status)
// Note: In production, compliance alerts should NEVER be hard deleted
func (r *ComplianceAlertRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE compliance_alerts
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, "deleted", id)
	if err != nil {
		return fmt.Errorf("failed to delete compliance alert: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("compliance alert not found: %s", id)
	}

	return nil
}

// CountByStatus counts compliance alerts by status
func (r *ComplianceAlertRepository) CountByStatus(ctx context.Context, status string) (int, error) {
	query := `SELECT COUNT(*) FROM compliance_alerts WHERE status = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, status)
	if err != nil {
		return 0, fmt.Errorf("failed to count compliance alerts: %w", err)
	}

	return count, nil
}

// CountBySeverity counts compliance alerts by severity
func (r *ComplianceAlertRepository) CountBySeverity(ctx context.Context, severity string) (int, error) {
	query := `SELECT COUNT(*) FROM compliance_alerts WHERE severity = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, severity)
	if err != nil {
		return 0, fmt.Errorf("failed to count compliance alerts: %w", err)
	}

	return count, nil
}
