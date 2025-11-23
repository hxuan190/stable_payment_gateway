package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
)

// PostgresPaymentRepository handles database operations for payments
type PostgresPaymentRepository struct {
	db *sql.DB
}

// NewPostgresPaymentRepository creates a new payment repository
func NewPostgresPaymentRepository(db *sql.DB) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{
		db: db,
	}
}

// Create inserts a new payment into the database
func (r *PostgresPaymentRepository) Create(payment *domain.Payment) error {
	if payment == nil {
		return errors.New("payment cannot be nil")
	}

	query := `
		INSERT INTO payments (
			id, merchant_id,
			amount_vnd, amount_crypto, currency, chain, exchange_rate,
			order_id, description, callback_url,
			status,
			tx_hash, tx_confirmations, from_address,
			payment_reference, destination_wallet,
			expires_at, paid_at, confirmed_at,
			compliance_deadline, compliance_submitted_at,
			fee_percentage, fee_vnd, net_amount_vnd,
			failure_reason,
			metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28
		)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if payment.CreatedAt.IsZero() {
		payment.CreatedAt = now
	}
	if payment.UpdatedAt.IsZero() {
		payment.UpdatedAt = now
	}

	err := r.db.QueryRow(
		query,
		payment.ID,
		payment.MerchantID,
		payment.AmountVND,
		payment.AmountCrypto,
		payment.Currency,
		payment.Chain,
		payment.ExchangeRate,
		payment.OrderID,
		payment.Description,
		payment.CallbackURL,
		payment.Status,
		payment.TxHash,
		payment.TxConfirmations,
		payment.FromAddress,
		payment.PaymentReference,
		payment.DestinationWallet,
		payment.ExpiresAt,
		payment.PaidAt,
		payment.ConfirmedAt,
		payment.ComplianceDeadline,
		payment.ComplianceSubmittedAt,
		payment.FeePercentage,
		payment.FeeVND,
		payment.NetAmountVND,
		payment.FailureReason,
		payment.Metadata,
		payment.CreatedAt,
		payment.UpdatedAt,
	).Scan(&payment.ID, &payment.CreatedAt, &payment.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation (duplicate payment reference)
		if err.Error() == `pq: duplicate key value violates unique constraint "payments_payment_reference_key"` {
			return domain.ErrPaymentAlreadyExists
		}
		return fmt.Errorf("failed to create payment: %w", err)
	}

	return nil
}

// GetByID retrieves a payment by ID
func (r *PostgresPaymentRepository) GetByID(id string) (*domain.Payment, error) {
	if id == "" {
		return nil, domain.ErrInvalidPaymentID
	}

	query := `
		SELECT
			id, merchant_id,
			amount_vnd, amount_crypto, currency, chain, exchange_rate,
			order_id, description, callback_url,
			status,
			tx_hash, tx_confirmations, from_address,
			payment_reference, destination_wallet,
			expires_at, paid_at, confirmed_at,
			compliance_deadline, compliance_submitted_at,
			fee_percentage, fee_vnd, net_amount_vnd,
			failure_reason,
			metadata,
			created_at, updated_at, deleted_at
		FROM payments
		WHERE id = $1 AND deleted_at IS NULL
	`

	payment := &domain.Payment{}
	err := r.db.QueryRow(query, id).Scan(
		&payment.ID,
		&payment.MerchantID,
		&payment.AmountVND,
		&payment.AmountCrypto,
		&payment.Currency,
		&payment.Chain,
		&payment.ExchangeRate,
		&payment.OrderID,
		&payment.Description,
		&payment.CallbackURL,
		&payment.Status,
		&payment.TxHash,
		&payment.TxConfirmations,
		&payment.FromAddress,
		&payment.PaymentReference,
		&payment.DestinationWallet,
		&payment.ExpiresAt,
		&payment.PaidAt,
		&payment.ConfirmedAt,
		&payment.ComplianceDeadline,
		&payment.ComplianceSubmittedAt,
		&payment.FeePercentage,
		&payment.FeeVND,
		&payment.NetAmountVND,
		&payment.FailureReason,
		&payment.Metadata,
		&payment.CreatedAt,
		&payment.UpdatedAt,
		&payment.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment by ID: %w", err)
	}

	return payment, nil
}

// GetByTxHash retrieves a payment by blockchain transaction hash
func (r *PostgresPaymentRepository) GetByTxHash(txHash string) (*domain.Payment, error) {
	if txHash == "" {
		return nil, errors.New("transaction hash cannot be empty")
	}

	query := `
		SELECT
			id, merchant_id,
			amount_vnd, amount_crypto, currency, chain, exchange_rate,
			order_id, description, callback_url,
			status,
			tx_hash, tx_confirmations, from_address,
			payment_reference, destination_wallet,
			expires_at, paid_at, confirmed_at,
			fee_percentage, fee_vnd, net_amount_vnd,
			failure_reason,
			metadata,
			created_at, updated_at, deleted_at
		FROM payments
		WHERE tx_hash = $1 AND deleted_at IS NULL
	`

	payment := &domain.Payment{}
	// Note: The original query in repo didn't select compliance fields for this method, keeping it consistent or adding them?
	// Original didn't have compliance fields in GetByTxHash. I should probably match the struct or original.
	// The original GetByTxHash scan has fewer fields than struct.
	// Wait, if I scan fewer fields than struct has, other fields will be zero. That's fine.
	// BUT I must scan exactly what I SELECT.
	// The original query:
	/*
		SELECT
			id, merchant_id,
			amount_vnd, amount_crypto, currency, chain, exchange_rate,
			order_id, description, callback_url,
			status,
			tx_hash, tx_confirmations, from_address,
			payment_reference, destination_wallet,
			expires_at, paid_at, confirmed_at,
			fee_percentage, fee_vnd, net_amount_vnd,
			failure_reason,
			metadata,
			created_at, updated_at, deleted_at
	*/
	// It skips compliance_deadline, compliance_submitted_at.

	err := r.db.QueryRow(query, txHash).Scan(
		&payment.ID,
		&payment.MerchantID,
		&payment.AmountVND,
		&payment.AmountCrypto,
		&payment.Currency,
		&payment.Chain,
		&payment.ExchangeRate,
		&payment.OrderID,
		&payment.Description,
		&payment.CallbackURL,
		&payment.Status,
		&payment.TxHash,
		&payment.TxConfirmations,
		&payment.FromAddress,
		&payment.PaymentReference,
		&payment.DestinationWallet,
		&payment.ExpiresAt,
		&payment.PaidAt,
		&payment.ConfirmedAt,
		&payment.FeePercentage,
		&payment.FeeVND,
		&payment.NetAmountVND,
		&payment.FailureReason,
		&payment.Metadata,
		&payment.CreatedAt,
		&payment.UpdatedAt,
		&payment.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment by tx hash: %w", err)
	}

	return payment, nil
}

// GetByPaymentReference retrieves a payment by payment reference
func (r *PostgresPaymentRepository) GetByPaymentReference(reference string) (*domain.Payment, error) {
	if reference == "" {
		return nil, errors.New("payment reference cannot be empty")
	}

	query := `
		SELECT
			id, merchant_id,
			amount_vnd, amount_crypto, currency, chain, exchange_rate,
			order_id, description, callback_url,
			status,
			tx_hash, tx_confirmations, from_address,
			payment_reference, destination_wallet,
			expires_at, paid_at, confirmed_at,
			fee_percentage, fee_vnd, net_amount_vnd,
			failure_reason,
			metadata,
			created_at, updated_at, deleted_at
		FROM payments
		WHERE payment_reference = $1 AND deleted_at IS NULL
	`

	payment := &domain.Payment{}
	// Same as GetByTxHash, skipping compliance fields in query
	err := r.db.QueryRow(query, reference).Scan(
		&payment.ID,
		&payment.MerchantID,
		&payment.AmountVND,
		&payment.AmountCrypto,
		&payment.Currency,
		&payment.Chain,
		&payment.ExchangeRate,
		&payment.OrderID,
		&payment.Description,
		&payment.CallbackURL,
		&payment.Status,
		&payment.TxHash,
		&payment.TxConfirmations,
		&payment.FromAddress,
		&payment.PaymentReference,
		&payment.DestinationWallet,
		&payment.ExpiresAt,
		&payment.PaidAt,
		&payment.ConfirmedAt,
		&payment.FeePercentage,
		&payment.FeeVND,
		&payment.NetAmountVND,
		&payment.FailureReason,
		&payment.Metadata,
		&payment.CreatedAt,
		&payment.UpdatedAt,
		&payment.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment by reference: %w", err)
	}

	return payment, nil
}

// UpdateStatus updates only the status of a payment
func (r *PostgresPaymentRepository) UpdateStatus(id string, status domain.PaymentStatus) error {
	if id == "" {
		return domain.ErrInvalidPaymentID
	}
	if status == "" {
		return domain.ErrInvalidPaymentStatus
	}

	// Validate status
	validStatuses := map[domain.PaymentStatus]bool{
		domain.PaymentStatusCreated:           true,
		domain.PaymentStatusPending:           true,
		domain.PaymentStatusPendingCompliance: true,
		domain.PaymentStatusConfirming:        true,
		domain.PaymentStatusCompleted:         true,
		domain.PaymentStatusExpired:           true,
		domain.PaymentStatusFailed:            true,
	}
	if !validStatuses[status] {
		return domain.ErrInvalidPaymentStatus
	}

	query := `
		UPDATE payments
		SET status = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, id, status, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrPaymentNotFound
	}

	return nil
}

// Update updates an existing payment in the database
func (r *PostgresPaymentRepository) Update(payment *domain.Payment) error {
	if payment == nil {
		return errors.New("payment cannot be nil")
	}
	if payment.ID == "" {
		return domain.ErrInvalidPaymentID
	}

	query := `
		UPDATE payments SET
			merchant_id = $2,
			amount_vnd = $3,
			amount_crypto = $4,
			currency = $5,
			chain = $6,
			exchange_rate = $7,
			order_id = $8,
			description = $9,
			callback_url = $10,
			status = $11,
			tx_hash = $12,
			tx_confirmations = $13,
			from_address = $14,
			payment_reference = $15,
			destination_wallet = $16,
			expires_at = $17,
			paid_at = $18,
			confirmed_at = $19,
			compliance_deadline = $20,
			compliance_submitted_at = $21,
			fee_percentage = $22,
			fee_vnd = $23,
			net_amount_vnd = $24,
			failure_reason = $25,
			metadata = $26,
			updated_at = $27
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`

	payment.UpdatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		payment.ID,
		payment.MerchantID,
		payment.AmountVND,
		payment.AmountCrypto,
		payment.Currency,
		payment.Chain,
		payment.ExchangeRate,
		payment.OrderID,
		payment.Description,
		payment.CallbackURL,
		payment.Status,
		payment.TxHash,
		payment.TxConfirmations,
		payment.FromAddress,
		payment.PaymentReference,
		payment.DestinationWallet,
		payment.ExpiresAt,
		payment.PaidAt,
		payment.ConfirmedAt,
		payment.ComplianceDeadline,
		payment.ComplianceSubmittedAt,
		payment.FeePercentage,
		payment.FeeVND,
		payment.NetAmountVND,
		payment.FailureReason,
		payment.Metadata,
		payment.UpdatedAt,
	).Scan(&payment.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return domain.ErrPaymentNotFound
		}
		return fmt.Errorf("failed to update payment: %w", err)
	}

	return nil
}

// ListByMerchant retrieves payments for a specific merchant with pagination
func (r *PostgresPaymentRepository) ListByMerchant(merchantID string, limit, offset int) ([]*domain.Payment, error) {
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
			amount_vnd, amount_crypto, currency, chain, exchange_rate,
			order_id, description, callback_url,
			status,
			tx_hash, tx_confirmations, from_address,
			payment_reference, destination_wallet,
			expires_at, paid_at, confirmed_at,
			fee_percentage, fee_vnd, net_amount_vnd,
			failure_reason,
			metadata,
			created_at, updated_at, deleted_at
		FROM payments
		WHERE merchant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, merchantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments by merchant: %w", err)
	}
	defer rows.Close()

	payments := make([]*domain.Payment, 0)
	for rows.Next() {
		payment := &domain.Payment{}
		err := rows.Scan(
			&payment.ID,
			&payment.MerchantID,
			&payment.AmountVND,
			&payment.AmountCrypto,
			&payment.Currency,
			&payment.Chain,
			&payment.ExchangeRate,
			&payment.OrderID,
			&payment.Description,
			&payment.CallbackURL,
			&payment.Status,
			&payment.TxHash,
			&payment.TxConfirmations,
			&payment.FromAddress,
			&payment.PaymentReference,
			&payment.DestinationWallet,
			&payment.ExpiresAt,
			&payment.PaidAt,
			&payment.ConfirmedAt,
			&payment.FeePercentage,
			&payment.FeeVND,
			&payment.NetAmountVND,
			&payment.FailureReason,
			&payment.Metadata,
			&payment.CreatedAt,
			&payment.UpdatedAt,
			&payment.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment: %w", err)
		}
		payments = append(payments, payment)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payments: %w", err)
	}

	return payments, nil
}

// GetExpiredPayments retrieves payments that have expired but are still in created or pending status
func (r *PostgresPaymentRepository) GetExpiredPayments() ([]*domain.Payment, error) {
	query := `
		SELECT
			id, merchant_id,
			amount_vnd, amount_crypto, currency, chain, exchange_rate,
			order_id, description, callback_url,
			status,
			tx_hash, tx_confirmations, from_address,
			payment_reference, destination_wallet,
			expires_at, paid_at, confirmed_at,
			compliance_deadline, compliance_submitted_at,
			fee_percentage, fee_vnd, net_amount_vnd,
			failure_reason,
			metadata,
			created_at, updated_at, deleted_at
		FROM payments
		WHERE deleted_at IS NULL
		  AND status IN ('created', 'pending')
		  AND expires_at < $1
		ORDER BY expires_at ASC
	`

	rows, err := r.db.Query(query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get expired payments: %w", err)
	}
	defer rows.Close()

	payments := make([]*domain.Payment, 0)
	for rows.Next() {
		payment := &domain.Payment{}
		err := rows.Scan(
			&payment.ID,
			&payment.MerchantID,
			&payment.AmountVND,
			&payment.AmountCrypto,
			&payment.Currency,
			&payment.Chain,
			&payment.ExchangeRate,
			&payment.OrderID,
			&payment.Description,
			&payment.CallbackURL,
			&payment.Status,
			&payment.TxHash,
			&payment.TxConfirmations,
			&payment.FromAddress,
			&payment.PaymentReference,
			&payment.DestinationWallet,
			&payment.ExpiresAt,
			&payment.PaidAt,
			&payment.ConfirmedAt,
			&payment.ComplianceDeadline,
			&payment.ComplianceSubmittedAt,
			&payment.FeePercentage,
			&payment.FeeVND,
			&payment.NetAmountVND,
			&payment.FailureReason,
			&payment.Metadata,
			&payment.CreatedAt,
			&payment.UpdatedAt,
			&payment.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment: %w", err)
		}
		payments = append(payments, payment)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expired payments: %w", err)
	}

	return payments, nil
}

// GetComplianceExpiredPayments retrieves payments in pending_compliance status that have passed their compliance deadline
func (r *PostgresPaymentRepository) GetComplianceExpiredPayments() ([]*domain.Payment, error) {
	query := `
		SELECT
			id, merchant_id,
			amount_vnd, amount_crypto, currency, chain, exchange_rate,
			order_id, description, callback_url,
			status,
			tx_hash, tx_confirmations, from_address,
			payment_reference, destination_wallet,
			expires_at, paid_at, confirmed_at,
			compliance_deadline, compliance_submitted_at,
			fee_percentage, fee_vnd, net_amount_vnd,
			failure_reason,
			metadata,
			created_at, updated_at, deleted_at
		FROM payments
		WHERE deleted_at IS NULL
		  AND status = 'pending_compliance'
		  AND compliance_deadline IS NOT NULL
		  AND compliance_deadline < $1
		  AND compliance_submitted_at IS NULL
		ORDER BY compliance_deadline ASC
		LIMIT 1000
	`

	rows, err := r.db.Query(query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get compliance expired payments: %w", err)
	}
	defer rows.Close()

	payments := make([]*domain.Payment, 0)
	for rows.Next() {
		payment := &domain.Payment{}
		err := rows.Scan(
			&payment.ID,
			&payment.MerchantID,
			&payment.AmountVND,
			&payment.AmountCrypto,
			&payment.Currency,
			&payment.Chain,
			&payment.ExchangeRate,
			&payment.OrderID,
			&payment.Description,
			&payment.CallbackURL,
			&payment.Status,
			&payment.TxHash,
			&payment.TxConfirmations,
			&payment.FromAddress,
			&payment.PaymentReference,
			&payment.DestinationWallet,
			&payment.ExpiresAt,
			&payment.PaidAt,
			&payment.ConfirmedAt,
			&payment.ComplianceDeadline,
			&payment.ComplianceSubmittedAt,
			&payment.FeePercentage,
			&payment.FeeVND,
			&payment.NetAmountVND,
			&payment.FailureReason,
			&payment.Metadata,
			&payment.CreatedAt,
			&payment.UpdatedAt,
			&payment.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment: %w", err)
		}
		payments = append(payments, payment)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating compliance expired payments: %w", err)
	}

	return payments, nil
}

// ListByStatus retrieves payments by status with pagination
func (r *PostgresPaymentRepository) ListByStatus(status domain.PaymentStatus, limit, offset int) ([]*domain.Payment, error) {
	if status == "" {
		return nil, domain.ErrInvalidPaymentStatus
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
			amount_vnd, amount_crypto, currency, chain, exchange_rate,
			order_id, description, callback_url,
			status,
			tx_hash, tx_confirmations, from_address,
			payment_reference, destination_wallet,
			expires_at, paid_at, confirmed_at,
			fee_percentage, fee_vnd, net_amount_vnd,
			failure_reason,
			metadata,
			created_at, updated_at, deleted_at
		FROM payments
		WHERE status = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments by status: %w", err)
	}
	defer rows.Close()

	payments := make([]*domain.Payment, 0)
	for rows.Next() {
		payment := &domain.Payment{}
		err := rows.Scan(
			&payment.ID,
			&payment.MerchantID,
			&payment.AmountVND,
			&payment.AmountCrypto,
			&payment.Currency,
			&payment.Chain,
			&payment.ExchangeRate,
			&payment.OrderID,
			&payment.Description,
			&payment.CallbackURL,
			&payment.Status,
			&payment.TxHash,
			&payment.TxConfirmations,
			&payment.FromAddress,
			&payment.PaymentReference,
			&payment.DestinationWallet,
			&payment.ExpiresAt,
			&payment.PaidAt,
			&payment.ConfirmedAt,
			&payment.FeePercentage,
			&payment.FeeVND,
			&payment.NetAmountVND,
			&payment.FailureReason,
			&payment.Metadata,
			&payment.CreatedAt,
			&payment.UpdatedAt,
			&payment.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment: %w", err)
		}
		payments = append(payments, payment)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating payments: %w", err)
	}

	return payments, nil
}

// Count returns the total number of payments (excluding deleted)
func (r *PostgresPaymentRepository) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM payments WHERE deleted_at IS NULL`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count payments: %w", err)
	}

	return count, nil
}

// CountByMerchant returns the count of payments for a specific merchant
func (r *PostgresPaymentRepository) CountByMerchant(merchantID string) (int64, error) {
	if merchantID == "" {
		return 0, errors.New("merchant ID cannot be empty")
	}

	query := `SELECT COUNT(*) FROM payments WHERE merchant_id = $1 AND deleted_at IS NULL`

	var count int64
	err := r.db.QueryRow(query, merchantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count payments by merchant: %w", err)
	}

	return count, nil
}

// CountByStatus returns the count of payments by status
func (r *PostgresPaymentRepository) CountByStatus(status domain.PaymentStatus) (int64, error) {
	if status == "" {
		return 0, domain.ErrInvalidPaymentStatus
	}

	query := `SELECT COUNT(*) FROM payments WHERE status = $1 AND deleted_at IS NULL`

	var count int64
	err := r.db.QueryRow(query, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count payments by status: %w", err)
	}

	return count, nil
}

// GetTotalVolumeByMerchant returns the total payment volume for a merchant
func (r *PostgresPaymentRepository) GetTotalVolumeByMerchant(merchantID string) (decimal.Decimal, error) {
	if merchantID == "" {
		return decimal.Zero, errors.New("merchant ID cannot be empty")
	}

	query := `
		SELECT COALESCE(SUM(amount_vnd), 0)
		FROM payments
		WHERE merchant_id = $1
		  AND status = 'completed'
		  AND deleted_at IS NULL
	`

	var total decimal.Decimal
	err := r.db.QueryRow(query, merchantID).Scan(&total)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get total volume by merchant: %w", err)
	}

	return total, nil
}

// CountByAddressAndWindow returns the count of payments from a specific wallet address within a time window
func (r *PostgresPaymentRepository) CountByAddressAndWindow(fromAddress string, window time.Duration) (int64, error) {
	if fromAddress == "" {
		return 0, errors.New("from address cannot be empty")
	}

	// Count only non-failed payments (failed payments don't count toward velocity limit)
	query := `
		SELECT COUNT(*)
		FROM payments
		WHERE from_address = $1
		  AND created_at > $2
		  AND status != 'failed'
		  AND deleted_at IS NULL
	`

	windowStart := time.Now().Add(-window)
	var count int64
	err := r.db.QueryRow(query, fromAddress, windowStart).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count payments by address and window: %w", err)
	}

	return count, nil
}

// Delete soft-deletes a payment by setting the deleted_at timestamp
func (r *PostgresPaymentRepository) Delete(id string) error {
	if id == "" {
		return domain.ErrInvalidPaymentID
	}

	query := `
		UPDATE payments
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete payment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrPaymentNotFound
	}

	return nil
}
