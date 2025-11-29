package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/treasury/domain"
	"github.com/shopspring/decimal"
)

var (
	// ErrTreasuryOperationNotFound is returned when a treasury operation is not found
	ErrTreasuryOperationNotFound = errors.New("treasury operation not found")
)

// TreasuryOperationRepository handles database operations for treasury operations
type TreasuryOperationRepository struct {
	db *sql.DB
}

// NewTreasuryOperationRepository creates a new treasury operation repository
func NewTreasuryOperationRepository(db *sql.DB) *TreasuryOperationRepository {
	return &TreasuryOperationRepository{
		db: db,
	}
}

// Create inserts a new treasury operation
func (r *TreasuryOperationRepository) Create(op *domain.TreasuryOperation) error {
	if op == nil {
		return errors.New("treasury operation cannot be nil")
	}

	// Generate UUID if not provided
	if op.ID == uuid.Nil {
		op.ID = uuid.New()
	}

	query := `
		INSERT INTO treasury_operations (
			id, operation_type,
			from_wallet_id, to_wallet_id, from_address, to_address,
			blockchain, amount, currency, amount_usd,
			tx_hash, tx_confirmations,
			requires_multisig, signatures_required, signatures_collected, signers,
			status, initiated_at, broadcasted_at, confirmed_at, failed_at, cancelled_at,
			reason, triggered_by, initiated_by,
			error_message, error_code, retry_count,
			gas_used, gas_price, transaction_fee, transaction_fee_usd,
			metadata,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35, $36, $37
		)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if op.CreatedAt.IsZero() {
		op.CreatedAt = now
	}
	if op.UpdatedAt.IsZero() {
		op.UpdatedAt = now
	}
	if op.InitiatedAt.IsZero() {
		op.InitiatedAt = now
	}

	err := r.db.QueryRow(
		query,
		op.ID,
		op.OperationType,
		op.FromWalletID,
		op.ToWalletID,
		op.FromAddress,
		op.ToAddress,
		op.Blockchain,
		op.Amount,
		op.Currency,
		op.AmountUSD,
		op.TxHash,
		op.TxConfirmations,
		op.RequiresMultisig,
		op.SignaturesRequired,
		op.SignaturesCollected,
		op.Signers,
		op.Status,
		op.InitiatedAt,
		op.BroadcastedAt,
		op.ConfirmedAt,
		op.FailedAt,
		op.CancelledAt,
		op.Reason,
		op.TriggeredBy,
		op.InitiatedBy,
		op.ErrorMessage,
		op.ErrorCode,
		op.RetryCount,
		op.GasUsed,
		op.GasPrice,
		op.TransactionFee,
		op.TransactionFeeUSD,
		op.Metadata,
		op.CreatedAt,
		op.UpdatedAt,
		op.CreatedBy,
		op.UpdatedBy,
	).Scan(&op.ID, &op.CreatedAt, &op.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create treasury operation: %w", err)
	}

	return nil
}

// GetByID retrieves a treasury operation by ID
func (r *TreasuryOperationRepository) GetByID(id uuid.UUID) (*domain.TreasuryOperation, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid treasury operation ID")
	}

	query := `
		SELECT
			id, operation_type,
			from_wallet_id, to_wallet_id, from_address, to_address,
			blockchain, amount, currency, amount_usd,
			tx_hash, tx_confirmations,
			requires_multisig, signatures_required, signatures_collected, signers,
			status, initiated_at, broadcasted_at, confirmed_at, failed_at, cancelled_at,
			reason, triggered_by, initiated_by,
			error_message, error_code, retry_count,
			gas_used, gas_price, transaction_fee, transaction_fee_usd,
			metadata,
			created_at, updated_at, created_by, updated_by
		FROM treasury_operations
		WHERE id = $1
	`

	op := &domain.TreasuryOperation{}
	err := r.db.QueryRow(query, id).Scan(
		&op.ID,
		&op.OperationType,
		&op.FromWalletID,
		&op.ToWalletID,
		&op.FromAddress,
		&op.ToAddress,
		&op.Blockchain,
		&op.Amount,
		&op.Currency,
		&op.AmountUSD,
		&op.TxHash,
		&op.TxConfirmations,
		&op.RequiresMultisig,
		&op.SignaturesRequired,
		&op.SignaturesCollected,
		&op.Signers,
		&op.Status,
		&op.InitiatedAt,
		&op.BroadcastedAt,
		&op.ConfirmedAt,
		&op.FailedAt,
		&op.CancelledAt,
		&op.Reason,
		&op.TriggeredBy,
		&op.InitiatedBy,
		&op.ErrorMessage,
		&op.ErrorCode,
		&op.RetryCount,
		&op.GasUsed,
		&op.GasPrice,
		&op.TransactionFee,
		&op.TransactionFeeUSD,
		&op.Metadata,
		&op.CreatedAt,
		&op.UpdatedAt,
		&op.CreatedBy,
		&op.UpdatedBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTreasuryOperationNotFound
		}
		return nil, fmt.Errorf("failed to get treasury operation by ID: %w", err)
	}

	return op, nil
}

// GetByTxHash retrieves a treasury operation by transaction hash
func (r *TreasuryOperationRepository) GetByTxHash(txHash string) (*domain.TreasuryOperation, error) {
	if txHash == "" {
		return nil, errors.New("transaction hash cannot be empty")
	}

	query := `
		SELECT
			id, operation_type,
			from_wallet_id, to_wallet_id, from_address, to_address,
			blockchain, amount, currency, amount_usd,
			tx_hash, tx_confirmations,
			requires_multisig, signatures_required, signatures_collected, signers,
			status, initiated_at, broadcasted_at, confirmed_at, failed_at, cancelled_at,
			reason, triggered_by, initiated_by,
			error_message, error_code, retry_count,
			gas_used, gas_price, transaction_fee, transaction_fee_usd,
			metadata,
			created_at, updated_at, created_by, updated_by
		FROM treasury_operations
		WHERE tx_hash = $1
	`

	op := &domain.TreasuryOperation{}
	err := r.db.QueryRow(query, sql.NullString{String: txHash, Valid: true}).Scan(
		&op.ID,
		&op.OperationType,
		&op.FromWalletID,
		&op.ToWalletID,
		&op.FromAddress,
		&op.ToAddress,
		&op.Blockchain,
		&op.Amount,
		&op.Currency,
		&op.AmountUSD,
		&op.TxHash,
		&op.TxConfirmations,
		&op.RequiresMultisig,
		&op.SignaturesRequired,
		&op.SignaturesCollected,
		&op.Signers,
		&op.Status,
		&op.InitiatedAt,
		&op.BroadcastedAt,
		&op.ConfirmedAt,
		&op.FailedAt,
		&op.CancelledAt,
		&op.Reason,
		&op.TriggeredBy,
		&op.InitiatedBy,
		&op.ErrorMessage,
		&op.ErrorCode,
		&op.RetryCount,
		&op.GasUsed,
		&op.GasPrice,
		&op.TransactionFee,
		&op.TransactionFeeUSD,
		&op.Metadata,
		&op.CreatedAt,
		&op.UpdatedAt,
		&op.CreatedBy,
		&op.UpdatedBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTreasuryOperationNotFound
		}
		return nil, fmt.Errorf("failed to get treasury operation by tx hash: %w", err)
	}

	return op, nil
}

// ListByWallet retrieves treasury operations by wallet ID
func (r *TreasuryOperationRepository) ListByWallet(walletID uuid.UUID, limit, offset int) ([]*domain.TreasuryOperation, error) {
	if walletID == uuid.Nil {
		return nil, errors.New("wallet ID cannot be empty")
	}

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, operation_type,
			from_wallet_id, to_wallet_id, from_address, to_address,
			blockchain, amount, currency, amount_usd,
			tx_hash, tx_confirmations,
			requires_multisig, signatures_required, signatures_collected, signers,
			status, initiated_at, broadcasted_at, confirmed_at, failed_at, cancelled_at,
			reason, triggered_by, initiated_by,
			error_message, error_code, retry_count,
			gas_used, gas_price, transaction_fee, transaction_fee_usd,
			metadata,
			created_at, updated_at, created_by, updated_by
		FROM treasury_operations
		WHERE from_wallet_id = $1 OR to_wallet_id = $1
		ORDER BY initiated_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, sql.NullString{String: walletID.String(), Valid: true}, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list treasury operations by wallet: %w", err)
	}
	defer rows.Close()

	operations := make([]*domain.TreasuryOperation, 0)
	for rows.Next() {
		op := &domain.TreasuryOperation{}
		err := rows.Scan(
			&op.ID,
			&op.OperationType,
			&op.FromWalletID,
			&op.ToWalletID,
			&op.FromAddress,
			&op.ToAddress,
			&op.Blockchain,
			&op.Amount,
			&op.Currency,
			&op.AmountUSD,
			&op.TxHash,
			&op.TxConfirmations,
			&op.RequiresMultisig,
			&op.SignaturesRequired,
			&op.SignaturesCollected,
			&op.Signers,
			&op.Status,
			&op.InitiatedAt,
			&op.BroadcastedAt,
			&op.ConfirmedAt,
			&op.FailedAt,
			&op.CancelledAt,
			&op.Reason,
			&op.TriggeredBy,
			&op.InitiatedBy,
			&op.ErrorMessage,
			&op.ErrorCode,
			&op.RetryCount,
			&op.GasUsed,
			&op.GasPrice,
			&op.TransactionFee,
			&op.TransactionFeeUSD,
			&op.Metadata,
			&op.CreatedAt,
			&op.UpdatedAt,
			&op.CreatedBy,
			&op.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan treasury operation: %w", err)
		}
		operations = append(operations, op)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating treasury operations: %w", err)
	}

	return operations, nil
}

// ListByType retrieves treasury operations by type
func (r *TreasuryOperationRepository) ListByType(opType domain.TreasuryOperationType, limit, offset int) ([]*domain.TreasuryOperation, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, operation_type,
			from_wallet_id, to_wallet_id, from_address, to_address,
			blockchain, amount, currency, amount_usd,
			tx_hash, tx_confirmations,
			requires_multisig, signatures_required, signatures_collected, signers,
			status, initiated_at, broadcasted_at, confirmed_at, failed_at, cancelled_at,
			reason, triggered_by, initiated_by,
			error_message, error_code, retry_count,
			gas_used, gas_price, transaction_fee, transaction_fee_usd,
			metadata,
			created_at, updated_at, created_by, updated_by
		FROM treasury_operations
		WHERE operation_type = $1
		ORDER BY initiated_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, opType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list treasury operations by type: %w", err)
	}
	defer rows.Close()

	operations := make([]*domain.TreasuryOperation, 0)
	for rows.Next() {
		op := &domain.TreasuryOperation{}
		err := rows.Scan(
			&op.ID,
			&op.OperationType,
			&op.FromWalletID,
			&op.ToWalletID,
			&op.FromAddress,
			&op.ToAddress,
			&op.Blockchain,
			&op.Amount,
			&op.Currency,
			&op.AmountUSD,
			&op.TxHash,
			&op.TxConfirmations,
			&op.RequiresMultisig,
			&op.SignaturesRequired,
			&op.SignaturesCollected,
			&op.Signers,
			&op.Status,
			&op.InitiatedAt,
			&op.BroadcastedAt,
			&op.ConfirmedAt,
			&op.FailedAt,
			&op.CancelledAt,
			&op.Reason,
			&op.TriggeredBy,
			&op.InitiatedBy,
			&op.ErrorMessage,
			&op.ErrorCode,
			&op.RetryCount,
			&op.GasUsed,
			&op.GasPrice,
			&op.TransactionFee,
			&op.TransactionFeeUSD,
			&op.Metadata,
			&op.CreatedAt,
			&op.UpdatedAt,
			&op.CreatedBy,
			&op.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan treasury operation: %w", err)
		}
		operations = append(operations, op)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating treasury operations: %w", err)
	}

	return operations, nil
}

// ListPendingSignatures retrieves operations pending multi-sig signatures
func (r *TreasuryOperationRepository) ListPendingSignatures() ([]*domain.TreasuryOperation, error) {
	query := `
		SELECT
			id, operation_type,
			from_wallet_id, to_wallet_id, from_address, to_address,
			blockchain, amount, currency, amount_usd,
			tx_hash, tx_confirmations,
			requires_multisig, signatures_required, signatures_collected, signers,
			status, initiated_at, broadcasted_at, confirmed_at, failed_at, cancelled_at,
			reason, triggered_by, initiated_by,
			error_message, error_code, retry_count,
			gas_used, gas_price, transaction_fee, transaction_fee_usd,
			metadata,
			created_at, updated_at, created_by, updated_by
		FROM treasury_operations
		WHERE requires_multisig = TRUE
		  AND status = 'pending_signatures'
		ORDER BY initiated_at ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending signature operations: %w", err)
	}
	defer rows.Close()

	operations := make([]*domain.TreasuryOperation, 0)
	for rows.Next() {
		op := &domain.TreasuryOperation{}
		err := rows.Scan(
			&op.ID,
			&op.OperationType,
			&op.FromWalletID,
			&op.ToWalletID,
			&op.FromAddress,
			&op.ToAddress,
			&op.Blockchain,
			&op.Amount,
			&op.Currency,
			&op.AmountUSD,
			&op.TxHash,
			&op.TxConfirmations,
			&op.RequiresMultisig,
			&op.SignaturesRequired,
			&op.SignaturesCollected,
			&op.Signers,
			&op.Status,
			&op.InitiatedAt,
			&op.BroadcastedAt,
			&op.ConfirmedAt,
			&op.FailedAt,
			&op.CancelledAt,
			&op.Reason,
			&op.TriggeredBy,
			&op.InitiatedBy,
			&op.ErrorMessage,
			&op.ErrorCode,
			&op.RetryCount,
			&op.GasUsed,
			&op.GasPrice,
			&op.TransactionFee,
			&op.TransactionFeeUSD,
			&op.Metadata,
			&op.CreatedAt,
			&op.UpdatedAt,
			&op.CreatedBy,
			&op.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan treasury operation: %w", err)
		}
		operations = append(operations, op)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating treasury operations: %w", err)
	}

	return operations, nil
}

// Update updates an existing treasury operation
func (r *TreasuryOperationRepository) Update(op *domain.TreasuryOperation) error {
	if op == nil {
		return errors.New("treasury operation cannot be nil")
	}
	if op.ID == uuid.Nil {
		return errors.New("invalid treasury operation ID")
	}

	query := `
		UPDATE treasury_operations SET
			tx_hash = $2,
			tx_confirmations = $3,
			signatures_collected = $4,
			signers = $5,
			status = $6,
			broadcasted_at = $7,
			confirmed_at = $8,
			failed_at = $9,
			cancelled_at = $10,
			error_message = $11,
			error_code = $12,
			retry_count = $13,
			gas_used = $14,
			gas_price = $15,
			transaction_fee = $16,
			transaction_fee_usd = $17,
			metadata = $18,
			updated_at = $19,
			updated_by = $20
		WHERE id = $1
		RETURNING updated_at
	`

	op.UpdatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		op.ID,
		op.TxHash,
		op.TxConfirmations,
		op.SignaturesCollected,
		op.Signers,
		op.Status,
		op.BroadcastedAt,
		op.ConfirmedAt,
		op.FailedAt,
		op.CancelledAt,
		op.ErrorMessage,
		op.ErrorCode,
		op.RetryCount,
		op.GasUsed,
		op.GasPrice,
		op.TransactionFee,
		op.TransactionFeeUSD,
		op.Metadata,
		op.UpdatedAt,
		op.UpdatedBy,
	).Scan(&op.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrTreasuryOperationNotFound
		}
		return fmt.Errorf("failed to update treasury operation: %w", err)
	}

	return nil
}

// MarkAsBroadcasted updates status to broadcasted
func (r *TreasuryOperationRepository) MarkAsBroadcasted(id uuid.UUID, txHash string) error {
	if id == uuid.Nil {
		return errors.New("invalid treasury operation ID")
	}

	query := `
		UPDATE treasury_operations
		SET
			status = $2,
			tx_hash = $3,
			broadcasted_at = $4,
			updated_at = $4
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(
		query,
		id,
		domain.TreasuryOpStatusBroadcasted,
		sql.NullString{String: txHash, Valid: txHash != ""},
		sql.NullTime{Time: now, Valid: true},
	)
	if err != nil {
		return fmt.Errorf("failed to mark operation as broadcasted: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTreasuryOperationNotFound
	}

	return nil
}

// MarkAsConfirmed updates status to confirmed
func (r *TreasuryOperationRepository) MarkAsConfirmed(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid treasury operation ID")
	}

	query := `
		UPDATE treasury_operations
		SET
			status = $2,
			confirmed_at = $3,
			updated_at = $3
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(query, id, domain.TreasuryOpStatusConfirmed, sql.NullTime{Time: now, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to mark operation as confirmed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTreasuryOperationNotFound
	}

	return nil
}

// MarkAsFailed updates status to failed
func (r *TreasuryOperationRepository) MarkAsFailed(id uuid.UUID, errorMessage, errorCode string) error {
	if id == uuid.Nil {
		return errors.New("invalid treasury operation ID")
	}

	query := `
		UPDATE treasury_operations
		SET
			status = $2,
			failed_at = $3,
			error_message = $4,
			error_code = $5,
			updated_at = $3
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(
		query,
		id,
		domain.TreasuryOpStatusFailed,
		sql.NullTime{Time: now, Valid: true},
		sql.NullString{String: errorMessage, Valid: true},
		sql.NullString{String: errorCode, Valid: errorCode != ""},
	)
	if err != nil {
		return fmt.Errorf("failed to mark operation as failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTreasuryOperationNotFound
	}

	return nil
}

// IncrementSignatures increments the signature count
func (r *TreasuryOperationRepository) IncrementSignatures(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid treasury operation ID")
	}

	query := `
		UPDATE treasury_operations
		SET
			signatures_collected = signatures_collected + 1,
			updated_at = $2
		WHERE id = $1
	`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to increment signatures: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTreasuryOperationNotFound
	}

	return nil
}

// GetOperationStats retrieves statistics for treasury operations
func (r *TreasuryOperationRepository) GetOperationStats(startTime, endTime time.Time) (map[string]interface{}, error) {
	query := `
		SELECT
			operation_type,
			COUNT(*) as count,
			SUM(amount_usd) as total_amount_usd,
			COUNT(*) FILTER (WHERE status = 'confirmed') as confirmed_count,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_count
		FROM treasury_operations
		WHERE initiated_at >= $1 AND initiated_at <= $2
		GROUP BY operation_type
	`

	rows, err := r.db.Query(query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]interface{})
	for rows.Next() {
		var opType string
		var count, confirmedCount, failedCount int64
		var totalAmountUSD decimal.Decimal

		if err := rows.Scan(&opType, &count, &totalAmountUSD, &confirmedCount, &failedCount); err != nil {
			return nil, fmt.Errorf("failed to scan operation stats: %w", err)
		}

		stats[opType] = map[string]interface{}{
			"count":            count,
			"total_amount_usd": totalAmountUSD,
			"confirmed_count":  confirmedCount,
			"failed_count":     failedCount,
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating operation stats: %w", err)
	}

	return stats, nil
}

// Count returns the total number of treasury operations
func (r *TreasuryOperationRepository) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM treasury_operations`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count treasury operations: %w", err)
	}

	return count, nil
}
