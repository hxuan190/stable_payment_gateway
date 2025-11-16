package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

var (
	// ErrPayoutNotFound is returned when a payout is not found
	ErrPayoutNotFound = errors.New("payout not found")
	// ErrInvalidPayoutID is returned when the payout ID is invalid
	ErrInvalidPayoutID = errors.New("invalid payout ID")
	// ErrInvalidPayoutStatus is returned when the payout status is invalid
	ErrInvalidPayoutStatus = errors.New("invalid payout status")
)

// PayoutRepository handles database operations for payouts
type PayoutRepository struct {
	db *sql.DB
}

// NewPayoutRepository creates a new payout repository
func NewPayoutRepository(db *sql.DB) *PayoutRepository {
	return &PayoutRepository{
		db: db,
	}
}

// Create inserts a new payout into the database
func (r *PayoutRepository) Create(payout *model.Payout) error {
	if payout == nil {
		return errors.New("payout cannot be nil")
	}

	query := `
		INSERT INTO payouts (
			id, merchant_id,
			amount_vnd, fee_vnd, net_amount_vnd,
			bank_account_name, bank_account_number, bank_name, bank_branch,
			status,
			requested_by, approved_by, approved_at, rejection_reason,
			processed_by, processed_at, completion_date,
			bank_reference_number, transaction_receipt_url,
			failure_reason, retry_count,
			notes, metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25
		)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if payout.CreatedAt.IsZero() {
		payout.CreatedAt = now
	}
	if payout.UpdatedAt.IsZero() {
		payout.UpdatedAt = now
	}

	err := r.db.QueryRow(
		query,
		payout.ID,
		payout.MerchantID,
		payout.AmountVND,
		payout.FeeVND,
		payout.NetAmountVND,
		payout.BankAccountName,
		payout.BankAccountNumber,
		payout.BankName,
		payout.BankBranch,
		payout.Status,
		payout.RequestedBy,
		payout.ApprovedBy,
		payout.ApprovedAt,
		payout.RejectionReason,
		payout.ProcessedBy,
		payout.ProcessedAt,
		payout.CompletionDate,
		payout.BankReferenceNumber,
		payout.TransactionReceiptURL,
		payout.FailureReason,
		payout.RetryCount,
		payout.Notes,
		payout.Metadata,
		payout.CreatedAt,
		payout.UpdatedAt,
	).Scan(&payout.ID, &payout.CreatedAt, &payout.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create payout: %w", err)
	}

	return nil
}

// GetByID retrieves a payout by ID
func (r *PayoutRepository) GetByID(id string) (*model.Payout, error) {
	if id == "" {
		return nil, ErrInvalidPayoutID
	}

	query := `
		SELECT
			id, merchant_id,
			amount_vnd, fee_vnd, net_amount_vnd,
			bank_account_name, bank_account_number, bank_name, bank_branch,
			status,
			requested_by, approved_by, approved_at, rejection_reason,
			processed_by, processed_at, completion_date,
			bank_reference_number, transaction_receipt_url,
			failure_reason, retry_count,
			notes, metadata,
			created_at, updated_at, deleted_at
		FROM payouts
		WHERE id = $1 AND deleted_at IS NULL
	`

	payout := &model.Payout{}
	err := r.db.QueryRow(query, id).Scan(
		&payout.ID,
		&payout.MerchantID,
		&payout.AmountVND,
		&payout.FeeVND,
		&payout.NetAmountVND,
		&payout.BankAccountName,
		&payout.BankAccountNumber,
		&payout.BankName,
		&payout.BankBranch,
		&payout.Status,
		&payout.RequestedBy,
		&payout.ApprovedBy,
		&payout.ApprovedAt,
		&payout.RejectionReason,
		&payout.ProcessedBy,
		&payout.ProcessedAt,
		&payout.CompletionDate,
		&payout.BankReferenceNumber,
		&payout.TransactionReceiptURL,
		&payout.FailureReason,
		&payout.RetryCount,
		&payout.Notes,
		&payout.Metadata,
		&payout.CreatedAt,
		&payout.UpdatedAt,
		&payout.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPayoutNotFound
		}
		return nil, fmt.Errorf("failed to get payout by ID: %w", err)
	}

	return payout, nil
}

// ListByMerchant retrieves payouts for a specific merchant with pagination
func (r *PayoutRepository) ListByMerchant(merchantID string, limit, offset int) ([]*model.Payout, error) {
	if merchantID == "" {
		return nil, errors.New("merchant ID cannot be empty")
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, merchant_id,
			amount_vnd, fee_vnd, net_amount_vnd,
			bank_account_name, bank_account_number, bank_name, bank_branch,
			status,
			requested_by, approved_by, approved_at, rejection_reason,
			processed_by, processed_at, completion_date,
			bank_reference_number, transaction_receipt_url,
			failure_reason, retry_count,
			notes, metadata,
			created_at, updated_at, deleted_at
		FROM payouts
		WHERE merchant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, merchantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list payouts by merchant: %w", err)
	}
	defer rows.Close()

	payouts := make([]*model.Payout, 0)
	for rows.Next() {
		payout := &model.Payout{}
		err := rows.Scan(
			&payout.ID,
			&payout.MerchantID,
			&payout.AmountVND,
			&payout.FeeVND,
			&payout.NetAmountVND,
			&payout.BankAccountName,
			&payout.BankAccountNumber,
			&payout.BankName,
			&payout.BankBranch,
			&payout.Status,
			&payout.RequestedBy,
			&payout.ApprovedBy,
			&payout.ApprovedAt,
			&payout.RejectionReason,
			&payout.ProcessedBy,
			&payout.ProcessedAt,
			&payout.CompletionDate,
			&payout.BankReferenceNumber,
			&payout.TransactionReceiptURL,
			&payout.FailureReason,
			&payout.RetryCount,
			&payout.Notes,
			&payout.Metadata,
			&payout.CreatedAt,
			&payout.UpdatedAt,
			&payout.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payout: %w", err)
		}
		payouts = append(payouts, payout)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payouts: %w", err)
	}

	return payouts, nil
}

// ListPending retrieves all pending payouts (requested status)
func (r *PayoutRepository) ListPending() ([]*model.Payout, error) {
	query := `
		SELECT
			id, merchant_id,
			amount_vnd, fee_vnd, net_amount_vnd,
			bank_account_name, bank_account_number, bank_name, bank_branch,
			status,
			requested_by, approved_by, approved_at, rejection_reason,
			processed_by, processed_at, completion_date,
			bank_reference_number, transaction_receipt_url,
			failure_reason, retry_count,
			notes, metadata,
			created_at, updated_at, deleted_at
		FROM payouts
		WHERE status = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, model.PayoutStatusRequested)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending payouts: %w", err)
	}
	defer rows.Close()

	payouts := make([]*model.Payout, 0)
	for rows.Next() {
		payout := &model.Payout{}
		err := rows.Scan(
			&payout.ID,
			&payout.MerchantID,
			&payout.AmountVND,
			&payout.FeeVND,
			&payout.NetAmountVND,
			&payout.BankAccountName,
			&payout.BankAccountNumber,
			&payout.BankName,
			&payout.BankBranch,
			&payout.Status,
			&payout.RequestedBy,
			&payout.ApprovedBy,
			&payout.ApprovedAt,
			&payout.RejectionReason,
			&payout.ProcessedBy,
			&payout.ProcessedAt,
			&payout.CompletionDate,
			&payout.BankReferenceNumber,
			&payout.TransactionReceiptURL,
			&payout.FailureReason,
			&payout.RetryCount,
			&payout.Notes,
			&payout.Metadata,
			&payout.CreatedAt,
			&payout.UpdatedAt,
			&payout.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payout: %w", err)
		}
		payouts = append(payouts, payout)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending payouts: %w", err)
	}

	return payouts, nil
}

// ListByStatus retrieves payouts by status with pagination
func (r *PayoutRepository) ListByStatus(status model.PayoutStatus, limit, offset int) ([]*model.Payout, error) {
	if status == "" {
		return nil, ErrInvalidPayoutStatus
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, merchant_id,
			amount_vnd, fee_vnd, net_amount_vnd,
			bank_account_name, bank_account_number, bank_name, bank_branch,
			status,
			requested_by, approved_by, approved_at, rejection_reason,
			processed_by, processed_at, completion_date,
			bank_reference_number, transaction_receipt_url,
			failure_reason, retry_count,
			notes, metadata,
			created_at, updated_at, deleted_at
		FROM payouts
		WHERE status = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list payouts by status: %w", err)
	}
	defer rows.Close()

	payouts := make([]*model.Payout, 0)
	for rows.Next() {
		payout := &model.Payout{}
		err := rows.Scan(
			&payout.ID,
			&payout.MerchantID,
			&payout.AmountVND,
			&payout.FeeVND,
			&payout.NetAmountVND,
			&payout.BankAccountName,
			&payout.BankAccountNumber,
			&payout.BankName,
			&payout.BankBranch,
			&payout.Status,
			&payout.RequestedBy,
			&payout.ApprovedBy,
			&payout.ApprovedAt,
			&payout.RejectionReason,
			&payout.ProcessedBy,
			&payout.ProcessedAt,
			&payout.CompletionDate,
			&payout.BankReferenceNumber,
			&payout.TransactionReceiptURL,
			&payout.FailureReason,
			&payout.RetryCount,
			&payout.Notes,
			&payout.Metadata,
			&payout.CreatedAt,
			&payout.UpdatedAt,
			&payout.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payout: %w", err)
		}
		payouts = append(payouts, payout)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payouts: %w", err)
	}

	return payouts, nil
}

// UpdateStatus updates only the status of a payout
func (r *PayoutRepository) UpdateStatus(id string, status model.PayoutStatus) error {
	if id == "" {
		return ErrInvalidPayoutID
	}
	if status == "" {
		return ErrInvalidPayoutStatus
	}

	// Validate status
	validStatuses := map[model.PayoutStatus]bool{
		model.PayoutStatusRequested:  true,
		model.PayoutStatusApproved:   true,
		model.PayoutStatusProcessing: true,
		model.PayoutStatusCompleted:  true,
		model.PayoutStatusRejected:   true,
		model.PayoutStatusFailed:     true,
	}
	if !validStatuses[status] {
		return ErrInvalidPayoutStatus
	}

	query := `
		UPDATE payouts
		SET status = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, id, status, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update payout status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPayoutNotFound
	}

	return nil
}

// Update updates an existing payout in the database
func (r *PayoutRepository) Update(payout *model.Payout) error {
	if payout == nil {
		return errors.New("payout cannot be nil")
	}
	if payout.ID == "" {
		return ErrInvalidPayoutID
	}

	query := `
		UPDATE payouts SET
			merchant_id = $2,
			amount_vnd = $3,
			fee_vnd = $4,
			net_amount_vnd = $5,
			bank_account_name = $6,
			bank_account_number = $7,
			bank_name = $8,
			bank_branch = $9,
			status = $10,
			requested_by = $11,
			approved_by = $12,
			approved_at = $13,
			rejection_reason = $14,
			processed_by = $15,
			processed_at = $16,
			completion_date = $17,
			bank_reference_number = $18,
			transaction_receipt_url = $19,
			failure_reason = $20,
			retry_count = $21,
			notes = $22,
			metadata = $23,
			updated_at = $24
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`

	payout.UpdatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		payout.ID,
		payout.MerchantID,
		payout.AmountVND,
		payout.FeeVND,
		payout.NetAmountVND,
		payout.BankAccountName,
		payout.BankAccountNumber,
		payout.BankName,
		payout.BankBranch,
		payout.Status,
		payout.RequestedBy,
		payout.ApprovedBy,
		payout.ApprovedAt,
		payout.RejectionReason,
		payout.ProcessedBy,
		payout.ProcessedAt,
		payout.CompletionDate,
		payout.BankReferenceNumber,
		payout.TransactionReceiptURL,
		payout.FailureReason,
		payout.RetryCount,
		payout.Notes,
		payout.Metadata,
		payout.UpdatedAt,
	).Scan(&payout.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrPayoutNotFound
		}
		return fmt.Errorf("failed to update payout: %w", err)
	}

	return nil
}

// MarkCompleted marks a payout as completed with bank reference number
func (r *PayoutRepository) MarkCompleted(id string, referenceNumber string) error {
	if id == "" {
		return ErrInvalidPayoutID
	}
	if referenceNumber == "" {
		return errors.New("bank reference number cannot be empty")
	}

	now := time.Now()
	query := `
		UPDATE payouts
		SET
			status = $2,
			bank_reference_number = $3,
			completion_date = $4,
			updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, id, model.PayoutStatusCompleted, referenceNumber, now)
	if err != nil {
		return fmt.Errorf("failed to mark payout as completed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPayoutNotFound
	}

	return nil
}

// Approve approves a payout request
func (r *PayoutRepository) Approve(id string, approvedBy string) error {
	if id == "" {
		return ErrInvalidPayoutID
	}
	if approvedBy == "" {
		return errors.New("approver ID cannot be empty")
	}

	now := time.Now()
	query := `
		UPDATE payouts
		SET
			status = $2,
			approved_by = $3,
			approved_at = $4,
			updated_at = $4
		WHERE id = $1 AND status = $5 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(
		query,
		id,
		model.PayoutStatusApproved,
		approvedBy,
		now,
		model.PayoutStatusRequested,
	)
	if err != nil {
		return fmt.Errorf("failed to approve payout: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("payout not found or not in requested status")
	}

	return nil
}

// Reject rejects a payout request
func (r *PayoutRepository) Reject(id string, reason string) error {
	if id == "" {
		return ErrInvalidPayoutID
	}
	if reason == "" {
		return errors.New("rejection reason cannot be empty")
	}

	now := time.Now()
	query := `
		UPDATE payouts
		SET
			status = $2,
			rejection_reason = $3,
			updated_at = $4
		WHERE id = $1 AND status = $5 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(
		query,
		id,
		model.PayoutStatusRejected,
		reason,
		now,
		model.PayoutStatusRequested,
	)
	if err != nil {
		return fmt.Errorf("failed to reject payout: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("payout not found or not in requested status")
	}

	return nil
}

// Count returns the total number of payouts (excluding deleted)
func (r *PayoutRepository) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM payouts WHERE deleted_at IS NULL`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count payouts: %w", err)
	}

	return count, nil
}

// CountByMerchant returns the count of payouts for a specific merchant
func (r *PayoutRepository) CountByMerchant(merchantID string) (int64, error) {
	if merchantID == "" {
		return 0, errors.New("merchant ID cannot be empty")
	}

	query := `SELECT COUNT(*) FROM payouts WHERE merchant_id = $1 AND deleted_at IS NULL`

	var count int64
	err := r.db.QueryRow(query, merchantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count payouts by merchant: %w", err)
	}

	return count, nil
}

// CountByStatus returns the count of payouts by status
func (r *PayoutRepository) CountByStatus(status model.PayoutStatus) (int64, error) {
	if status == "" {
		return 0, ErrInvalidPayoutStatus
	}

	query := `SELECT COUNT(*) FROM payouts WHERE status = $1 AND deleted_at IS NULL`

	var count int64
	err := r.db.QueryRow(query, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count payouts by status: %w", err)
	}

	return count, nil
}

// Delete soft-deletes a payout by setting the deleted_at timestamp
func (r *PayoutRepository) Delete(id string) error {
	if id == "" {
		return ErrInvalidPayoutID
	}

	query := `
		UPDATE payouts
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete payout: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPayoutNotFound
	}

	return nil
}
