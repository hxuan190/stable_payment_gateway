package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"stable_payment_gateway/internal/model"
)

var (
	// ErrPaymentNotFound is returned when a payment is not found
	ErrPaymentNotFound = errors.New("payment not found")
	// ErrPaymentAlreadyExists is returned when a payment with the same payment reference already exists
	ErrPaymentAlreadyExists = errors.New("payment with this reference already exists")
	// ErrInvalidPaymentID is returned when the payment ID is invalid
	ErrInvalidPaymentID = errors.New("invalid payment ID")
	// ErrInvalidPaymentStatus is returned when the payment status is invalid
	ErrInvalidPaymentStatus = errors.New("invalid payment status")
)

// PaymentRepository handles database operations for payments
type PaymentRepository struct {
	db *sql.DB
}

// NewPaymentRepository creates a new payment repository
func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{
		db: db,
	}
}

// Create inserts a new payment into the database
func (r *PaymentRepository) Create(payment *model.Payment) error {
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
			fee_percentage, fee_vnd, net_amount_vnd,
			failure_reason,
			metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26
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
			return ErrPaymentAlreadyExists
		}
		return fmt.Errorf("failed to create payment: %w", err)
	}

	return nil
}

// GetByID retrieves a payment by ID
func (r *PaymentRepository) GetByID(id string) (*model.Payment, error) {
	if id == "" {
		return nil, ErrInvalidPaymentID
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
		WHERE id = $1 AND deleted_at IS NULL
	`

	payment := &model.Payment{}
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
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment by ID: %w", err)
	}

	return payment, nil
}

// GetByTxHash retrieves a payment by blockchain transaction hash
func (r *PaymentRepository) GetByTxHash(txHash string) (*model.Payment, error) {
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

	payment := &model.Payment{}
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
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment by tx hash: %w", err)
	}

	return payment, nil
}

// GetByPaymentReference retrieves a payment by payment reference
func (r *PaymentRepository) GetByPaymentReference(reference string) (*model.Payment, error) {
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

	payment := &model.Payment{}
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
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment by reference: %w", err)
	}

	return payment, nil
}

// UpdateStatus updates only the status of a payment
func (r *PaymentRepository) UpdateStatus(id string, status model.PaymentStatus) error {
	if id == "" {
		return ErrInvalidPaymentID
	}
	if status == "" {
		return ErrInvalidPaymentStatus
	}

	// Validate status
	validStatuses := map[model.PaymentStatus]bool{
		model.PaymentStatusCreated:    true,
		model.PaymentStatusPending:    true,
		model.PaymentStatusConfirming: true,
		model.PaymentStatusCompleted:  true,
		model.PaymentStatusExpired:    true,
		model.PaymentStatusFailed:     true,
	}
	if !validStatuses[status] {
		return ErrInvalidPaymentStatus
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
		return ErrPaymentNotFound
	}

	return nil
}

// Update updates an existing payment in the database
func (r *PaymentRepository) Update(payment *model.Payment) error {
	if payment == nil {
		return errors.New("payment cannot be nil")
	}
	if payment.ID == "" {
		return ErrInvalidPaymentID
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
			fee_percentage = $20,
			fee_vnd = $21,
			net_amount_vnd = $22,
			failure_reason = $23,
			metadata = $24,
			updated_at = $25
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
		payment.FeePercentage,
		payment.FeeVND,
		payment.NetAmountVND,
		payment.FailureReason,
		payment.Metadata,
		payment.UpdatedAt,
	).Scan(&payment.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrPaymentNotFound
		}
		return fmt.Errorf("failed to update payment: %w", err)
	}

	return nil
}

// ListByMerchant retrieves payments for a specific merchant with pagination
func (r *PaymentRepository) ListByMerchant(merchantID string, limit, offset int) ([]*model.Payment, error) {
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

	payments := make([]*model.Payment, 0)
	for rows.Next() {
		payment := &model.Payment{}
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
func (r *PaymentRepository) GetExpiredPayments() ([]*model.Payment, error) {
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

	payments := make([]*model.Payment, 0)
	for rows.Next() {
		payment := &model.Payment{}
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
		return nil, fmt.Errorf("error iterating expired payments: %w", err)
	}

	return payments, nil
}

// ListByStatus retrieves payments by status with pagination
func (r *PaymentRepository) ListByStatus(status model.PaymentStatus, limit, offset int) ([]*model.Payment, error) {
	if status == "" {
		return nil, ErrInvalidPaymentStatus
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

	payments := make([]*model.Payment, 0)
	for rows.Next() {
		payment := &model.Payment{}
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
func (r *PaymentRepository) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM payments WHERE deleted_at IS NULL`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count payments: %w", err)
	}

	return count, nil
}

// CountByMerchant returns the count of payments for a specific merchant
func (r *PaymentRepository) CountByMerchant(merchantID string) (int64, error) {
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
func (r *PaymentRepository) CountByStatus(status model.PaymentStatus) (int64, error) {
	if status == "" {
		return 0, ErrInvalidPaymentStatus
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
func (r *PaymentRepository) GetTotalVolumeByMerchant(merchantID string) (decimal.Decimal, error) {
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

// Delete soft-deletes a payment by setting the deleted_at timestamp
func (r *PaymentRepository) Delete(id string) error {
	if id == "" {
		return ErrInvalidPaymentID
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
		return ErrPaymentNotFound
	}

	return nil
}
