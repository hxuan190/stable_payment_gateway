package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

var (
	// ErrBlockchainTxNotFound is returned when a blockchain transaction is not found
	ErrBlockchainTxNotFound = errors.New("blockchain transaction not found")
	// ErrBlockchainTxAlreadyExists is returned when a blockchain transaction with the same tx hash already exists
	ErrBlockchainTxAlreadyExists = errors.New("blockchain transaction with this tx hash already exists")
	// ErrInvalidTxHash is returned when the tx hash is invalid
	ErrInvalidTxHash = errors.New("invalid transaction hash")
	// ErrInvalidBlockchainTxStatus is returned when the blockchain transaction status is invalid
	ErrInvalidBlockchainTxStatus = errors.New("invalid blockchain transaction status")
)

// BlockchainTxRepository handles database operations for blockchain transactions
type BlockchainTxRepository struct {
	db *sql.DB
}

// NewBlockchainTxRepository creates a new blockchain transaction repository
func NewBlockchainTxRepository(db *sql.DB) *BlockchainTxRepository {
	return &BlockchainTxRepository{
		db: db,
	}
}

// Create inserts a new blockchain transaction into the database
func (r *BlockchainTxRepository) Create(tx *model.BlockchainTransaction) error {
	if tx == nil {
		return errors.New("blockchain transaction cannot be nil")
	}

	if tx.TxHash == "" {
		return ErrInvalidTxHash
	}

	query := `
		INSERT INTO blockchain_transactions (
			id, chain, network,
			tx_hash, block_number, block_timestamp,
			from_address, to_address, amount, currency, token_mint,
			memo, parsed_payment_reference,
			confirmations, is_finalized, finalized_at,
			gas_used, gas_price, transaction_fee, fee_currency,
			payment_id, is_matched, matched_at,
			status,
			raw_transaction,
			error_message, error_details,
			metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24,
			$25, $26, $27, $28, $29, $30
		)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = now
	}
	if tx.UpdatedAt.IsZero() {
		tx.UpdatedAt = now
	}

	err := r.db.QueryRow(
		query,
		tx.ID,
		tx.Chain,
		tx.Network,
		tx.TxHash,
		tx.BlockNumber,
		tx.BlockTimestamp,
		tx.FromAddress,
		tx.ToAddress,
		tx.Amount,
		tx.Currency,
		tx.TokenMint,
		tx.Memo,
		tx.ParsedPaymentReference,
		tx.Confirmations,
		tx.IsFinalized,
		tx.FinalizedAt,
		tx.GasUsed,
		tx.GasPrice,
		tx.TxFee,
		tx.FeeCurrency,
		tx.PaymentID,
		tx.IsMatched,
		tx.MatchedAt,
		tx.Status,
		tx.RawTransaction,
		tx.ErrorMessage,
		tx.ErrorDetails,
		tx.Metadata,
		tx.CreatedAt,
		tx.UpdatedAt,
	).Scan(&tx.ID, &tx.CreatedAt, &tx.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation (duplicate tx hash)
		if err.Error() == `pq: duplicate key value violates unique constraint "idx_blockchain_tx_hash"` {
			return ErrBlockchainTxAlreadyExists
		}
		return fmt.Errorf("failed to create blockchain transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a blockchain transaction by its ID
func (r *BlockchainTxRepository) GetByID(id string) (*model.BlockchainTransaction, error) {
	if id == "" {
		return nil, errors.New("id cannot be empty")
	}

	query := `
		SELECT
			id, chain, network,
			tx_hash, block_number, block_timestamp,
			from_address, to_address, amount, currency, token_mint,
			memo, parsed_payment_reference,
			confirmations, is_finalized, finalized_at,
			gas_used, gas_price, transaction_fee, fee_currency,
			payment_id, is_matched, matched_at,
			status,
			raw_transaction,
			error_message, error_details,
			metadata,
			created_at, updated_at
		FROM blockchain_transactions
		WHERE id = $1
	`

	tx := &model.BlockchainTransaction{}
	err := r.db.QueryRow(query, id).Scan(
		&tx.ID,
		&tx.Chain,
		&tx.Network,
		&tx.TxHash,
		&tx.BlockNumber,
		&tx.BlockTimestamp,
		&tx.FromAddress,
		&tx.ToAddress,
		&tx.Amount,
		&tx.Currency,
		&tx.TokenMint,
		&tx.Memo,
		&tx.ParsedPaymentReference,
		&tx.Confirmations,
		&tx.IsFinalized,
		&tx.FinalizedAt,
		&tx.GasUsed,
		&tx.GasPrice,
		&tx.TxFee,
		&tx.FeeCurrency,
		&tx.PaymentID,
		&tx.IsMatched,
		&tx.MatchedAt,
		&tx.Status,
		&tx.RawTransaction,
		&tx.ErrorMessage,
		&tx.ErrorDetails,
		&tx.Metadata,
		&tx.CreatedAt,
		&tx.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBlockchainTxNotFound
		}
		return nil, fmt.Errorf("failed to get blockchain transaction by ID: %w", err)
	}

	return tx, nil
}

// GetByTxHash retrieves a blockchain transaction by its transaction hash
func (r *BlockchainTxRepository) GetByTxHash(txHash string) (*model.BlockchainTransaction, error) {
	if txHash == "" {
		return nil, ErrInvalidTxHash
	}

	query := `
		SELECT
			id, chain, network,
			tx_hash, block_number, block_timestamp,
			from_address, to_address, amount, currency, token_mint,
			memo, parsed_payment_reference,
			confirmations, is_finalized, finalized_at,
			gas_used, gas_price, transaction_fee, fee_currency,
			payment_id, is_matched, matched_at,
			status,
			raw_transaction,
			error_message, error_details,
			metadata,
			created_at, updated_at
		FROM blockchain_transactions
		WHERE tx_hash = $1
	`

	tx := &model.BlockchainTransaction{}
	err := r.db.QueryRow(query, txHash).Scan(
		&tx.ID,
		&tx.Chain,
		&tx.Network,
		&tx.TxHash,
		&tx.BlockNumber,
		&tx.BlockTimestamp,
		&tx.FromAddress,
		&tx.ToAddress,
		&tx.Amount,
		&tx.Currency,
		&tx.TokenMint,
		&tx.Memo,
		&tx.ParsedPaymentReference,
		&tx.Confirmations,
		&tx.IsFinalized,
		&tx.FinalizedAt,
		&tx.GasUsed,
		&tx.GasPrice,
		&tx.TxFee,
		&tx.FeeCurrency,
		&tx.PaymentID,
		&tx.IsMatched,
		&tx.MatchedAt,
		&tx.Status,
		&tx.RawTransaction,
		&tx.ErrorMessage,
		&tx.ErrorDetails,
		&tx.Metadata,
		&tx.CreatedAt,
		&tx.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBlockchainTxNotFound
		}
		return nil, fmt.Errorf("failed to get blockchain transaction by tx hash: %w", err)
	}

	return tx, nil
}

// UpdateConfirmations updates the confirmation count for a blockchain transaction
func (r *BlockchainTxRepository) UpdateConfirmations(txHash string, confirmations int) error {
	if txHash == "" {
		return ErrInvalidTxHash
	}

	if confirmations < 0 {
		return errors.New("confirmations cannot be negative")
	}

	query := `
		UPDATE blockchain_transactions
		SET
			confirmations = $1,
			updated_at = $2
		WHERE tx_hash = $3
	`

	result, err := r.db.Exec(query, confirmations, time.Now(), txHash)
	if err != nil {
		return fmt.Errorf("failed to update confirmations: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrBlockchainTxNotFound
	}

	return nil
}

// UpdateStatus updates the status of a blockchain transaction
func (r *BlockchainTxRepository) UpdateStatus(txHash string, status model.BlockchainTxStatus) error {
	if txHash == "" {
		return ErrInvalidTxHash
	}

	query := `
		UPDATE blockchain_transactions
		SET
			status = $1,
			updated_at = $2
		WHERE tx_hash = $3
	`

	result, err := r.db.Exec(query, status, time.Now(), txHash)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrBlockchainTxNotFound
	}

	return nil
}

// MarkAsFinalized marks a blockchain transaction as finalized
func (r *BlockchainTxRepository) MarkAsFinalized(txHash string) error {
	if txHash == "" {
		return ErrInvalidTxHash
	}

	query := `
		UPDATE blockchain_transactions
		SET
			is_finalized = TRUE,
			finalized_at = $1,
			status = $2,
			updated_at = $3
		WHERE tx_hash = $4
	`

	now := time.Now()
	result, err := r.db.Exec(query, now, model.BlockchainTxStatusFinalized, now, txHash)
	if err != nil {
		return fmt.Errorf("failed to mark as finalized: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrBlockchainTxNotFound
	}

	return nil
}

// MatchToPayment associates a blockchain transaction with a payment
func (r *BlockchainTxRepository) MatchToPayment(txHash string, paymentID string) error {
	if txHash == "" {
		return ErrInvalidTxHash
	}

	if paymentID == "" {
		return errors.New("payment ID cannot be empty")
	}

	query := `
		UPDATE blockchain_transactions
		SET
			payment_id = $1,
			is_matched = TRUE,
			matched_at = $2,
			updated_at = $3
		WHERE tx_hash = $4
	`

	now := time.Now()
	result, err := r.db.Exec(query, paymentID, now, now, txHash)
	if err != nil {
		return fmt.Errorf("failed to match to payment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrBlockchainTxNotFound
	}

	return nil
}

// MarkAsFailed marks a blockchain transaction as failed with an error message
func (r *BlockchainTxRepository) MarkAsFailed(txHash string, errorMessage string) error {
	if txHash == "" {
		return ErrInvalidTxHash
	}

	query := `
		UPDATE blockchain_transactions
		SET
			status = $1,
			error_message = $2,
			updated_at = $3
		WHERE tx_hash = $4
	`

	result, err := r.db.Exec(query, model.BlockchainTxStatusFailed, errorMessage, time.Now(), txHash)
	if err != nil {
		return fmt.Errorf("failed to mark as failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrBlockchainTxNotFound
	}

	return nil
}

// GetPendingTransactions retrieves all pending blockchain transactions
func (r *BlockchainTxRepository) GetPendingTransactions() ([]*model.BlockchainTransaction, error) {
	query := `
		SELECT
			id, chain, network,
			tx_hash, block_number, block_timestamp,
			from_address, to_address, amount, currency, token_mint,
			memo, parsed_payment_reference,
			confirmations, is_finalized, finalized_at,
			gas_used, gas_price, transaction_fee, fee_currency,
			payment_id, is_matched, matched_at,
			status,
			raw_transaction,
			error_message, error_details,
			metadata,
			created_at, updated_at
		FROM blockchain_transactions
		WHERE status = $1 AND is_finalized = FALSE
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, model.BlockchainTxStatusPending)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*model.BlockchainTransaction
	for rows.Next() {
		tx := &model.BlockchainTransaction{}
		err := rows.Scan(
			&tx.ID,
			&tx.Chain,
			&tx.Network,
			&tx.TxHash,
			&tx.BlockNumber,
			&tx.BlockTimestamp,
			&tx.FromAddress,
			&tx.ToAddress,
			&tx.Amount,
			&tx.Currency,
			&tx.TokenMint,
			&tx.Memo,
			&tx.ParsedPaymentReference,
			&tx.Confirmations,
			&tx.IsFinalized,
			&tx.FinalizedAt,
			&tx.GasUsed,
			&tx.GasPrice,
			&tx.TxFee,
			&tx.FeeCurrency,
			&tx.PaymentID,
			&tx.IsMatched,
			&tx.MatchedAt,
			&tx.Status,
			&tx.RawTransaction,
			&tx.ErrorMessage,
			&tx.ErrorDetails,
			&tx.Metadata,
			&tx.CreatedAt,
			&tx.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

// GetUnmatchedTransactions retrieves blockchain transactions that have a payment reference but haven't been matched yet
func (r *BlockchainTxRepository) GetUnmatchedTransactions(chain model.Chain) ([]*model.BlockchainTransaction, error) {
	query := `
		SELECT
			id, chain, network,
			tx_hash, block_number, block_timestamp,
			from_address, to_address, amount, currency, token_mint,
			memo, parsed_payment_reference,
			confirmations, is_finalized, finalized_at,
			gas_used, gas_price, transaction_fee, fee_currency,
			payment_id, is_matched, matched_at,
			status,
			raw_transaction,
			error_message, error_details,
			metadata,
			created_at, updated_at
		FROM blockchain_transactions
		WHERE
			chain = $1
			AND is_matched = FALSE
			AND parsed_payment_reference IS NOT NULL
			AND status IN ($2, $3)
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, chain, model.BlockchainTxStatusConfirmed, model.BlockchainTxStatusFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get unmatched transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*model.BlockchainTransaction
	for rows.Next() {
		tx := &model.BlockchainTransaction{}
		err := rows.Scan(
			&tx.ID,
			&tx.Chain,
			&tx.Network,
			&tx.TxHash,
			&tx.BlockNumber,
			&tx.BlockTimestamp,
			&tx.FromAddress,
			&tx.ToAddress,
			&tx.Amount,
			&tx.Currency,
			&tx.TokenMint,
			&tx.Memo,
			&tx.ParsedPaymentReference,
			&tx.Confirmations,
			&tx.IsFinalized,
			&tx.FinalizedAt,
			&tx.GasUsed,
			&tx.GasPrice,
			&tx.TxFee,
			&tx.FeeCurrency,
			&tx.PaymentID,
			&tx.IsMatched,
			&tx.MatchedAt,
			&tx.Status,
			&tx.RawTransaction,
			&tx.ErrorMessage,
			&tx.ErrorDetails,
			&tx.Metadata,
			&tx.CreatedAt,
			&tx.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

// GetByPaymentID retrieves all blockchain transactions associated with a payment
func (r *BlockchainTxRepository) GetByPaymentID(paymentID string) ([]*model.BlockchainTransaction, error) {
	if paymentID == "" {
		return nil, errors.New("payment ID cannot be empty")
	}

	query := `
		SELECT
			id, chain, network,
			tx_hash, block_number, block_timestamp,
			from_address, to_address, amount, currency, token_mint,
			memo, parsed_payment_reference,
			confirmations, is_finalized, finalized_at,
			gas_used, gas_price, transaction_fee, fee_currency,
			payment_id, is_matched, matched_at,
			status,
			raw_transaction,
			error_message, error_details,
			metadata,
			created_at, updated_at
		FROM blockchain_transactions
		WHERE payment_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by payment ID: %w", err)
	}
	defer rows.Close()

	var transactions []*model.BlockchainTransaction
	for rows.Next() {
		tx := &model.BlockchainTransaction{}
		err := rows.Scan(
			&tx.ID,
			&tx.Chain,
			&tx.Network,
			&tx.TxHash,
			&tx.BlockNumber,
			&tx.BlockTimestamp,
			&tx.FromAddress,
			&tx.ToAddress,
			&tx.Amount,
			&tx.Currency,
			&tx.TokenMint,
			&tx.Memo,
			&tx.ParsedPaymentReference,
			&tx.Confirmations,
			&tx.IsFinalized,
			&tx.FinalizedAt,
			&tx.GasUsed,
			&tx.GasPrice,
			&tx.TxFee,
			&tx.FeeCurrency,
			&tx.PaymentID,
			&tx.IsMatched,
			&tx.MatchedAt,
			&tx.Status,
			&tx.RawTransaction,
			&tx.ErrorMessage,
			&tx.ErrorDetails,
			&tx.Metadata,
			&tx.CreatedAt,
			&tx.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

// ListByChainAndStatus retrieves blockchain transactions filtered by chain and status
func (r *BlockchainTxRepository) ListByChainAndStatus(chain model.Chain, status model.BlockchainTxStatus, limit, offset int) ([]*model.BlockchainTransaction, error) {
	query := `
		SELECT
			id, chain, network,
			tx_hash, block_number, block_timestamp,
			from_address, to_address, amount, currency, token_mint,
			memo, parsed_payment_reference,
			confirmations, is_finalized, finalized_at,
			gas_used, gas_price, transaction_fee, fee_currency,
			payment_id, is_matched, matched_at,
			status,
			raw_transaction,
			error_message, error_details,
			metadata,
			created_at, updated_at
		FROM blockchain_transactions
		WHERE chain = $1 AND status = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.Query(query, chain, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions by chain and status: %w", err)
	}
	defer rows.Close()

	var transactions []*model.BlockchainTransaction
	for rows.Next() {
		tx := &model.BlockchainTransaction{}
		err := rows.Scan(
			&tx.ID,
			&tx.Chain,
			&tx.Network,
			&tx.TxHash,
			&tx.BlockNumber,
			&tx.BlockTimestamp,
			&tx.FromAddress,
			&tx.ToAddress,
			&tx.Amount,
			&tx.Currency,
			&tx.TokenMint,
			&tx.Memo,
			&tx.ParsedPaymentReference,
			&tx.Confirmations,
			&tx.IsFinalized,
			&tx.FinalizedAt,
			&tx.GasUsed,
			&tx.GasPrice,
			&tx.TxFee,
			&tx.FeeCurrency,
			&tx.PaymentID,
			&tx.IsMatched,
			&tx.MatchedAt,
			&tx.Status,
			&tx.RawTransaction,
			&tx.ErrorMessage,
			&tx.ErrorDetails,
			&tx.Metadata,
			&tx.CreatedAt,
			&tx.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}
