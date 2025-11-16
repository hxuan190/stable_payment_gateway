package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/shopspring/decimal"
)

var (
	// ErrBalanceNotFound is returned when a merchant balance is not found
	ErrBalanceNotFound = errors.New("merchant balance not found")
	// ErrInsufficientBalance is returned when trying to deduct more than available balance
	ErrInsufficientBalance = errors.New("insufficient balance")
	// ErrBalanceVersionMismatch is returned when optimistic locking fails
	ErrBalanceVersionMismatch = errors.New("balance version mismatch - concurrent update detected")
	// ErrInvalidBalanceUpdate is returned when balance update would result in invalid state
	ErrInvalidBalanceUpdate = errors.New("invalid balance update")
	// ErrNegativeAmount is returned when a negative amount is provided
	ErrNegativeAmount = errors.New("amount cannot be negative")
)

// BalanceRepository handles database operations for merchant balances
type BalanceRepository struct {
	db *sql.DB
}

// NewBalanceRepository creates a new balance repository
func NewBalanceRepository(db *sql.DB) *BalanceRepository {
	return &BalanceRepository{
		db: db,
	}
}

// GetByMerchantID retrieves a merchant's balance by merchant ID
func (r *BalanceRepository) GetByMerchantID(merchantID string) (*model.MerchantBalance, error) {
	if merchantID == "" {
		return nil, errors.New("merchant ID cannot be empty")
	}

	query := `
		SELECT
			id, merchant_id,
			pending_vnd, available_vnd, total_vnd, reserved_vnd,
			total_received_vnd, total_paid_out_vnd, total_fees_vnd,
			total_payments_count, total_payouts_count,
			last_payment_at, last_payout_at,
			version, created_at, updated_at
		FROM merchant_balances
		WHERE merchant_id = $1
	`

	balance := &model.MerchantBalance{}
	err := r.db.QueryRow(query, merchantID).Scan(
		&balance.ID,
		&balance.MerchantID,
		&balance.PendingVND,
		&balance.AvailableVND,
		&balance.TotalVND,
		&balance.ReservedVND,
		&balance.TotalReceivedVND,
		&balance.TotalPaidOutVND,
		&balance.TotalFeesVND,
		&balance.TotalPaymentsCount,
		&balance.TotalPayoutsCount,
		&balance.LastPaymentAt,
		&balance.LastPayoutAt,
		&balance.Version,
		&balance.CreatedAt,
		&balance.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrBalanceNotFound
		}
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}

// GetByMerchantIDForUpdate retrieves a merchant's balance with row-level locking (SELECT ... FOR UPDATE)
// This should be used within a transaction when you need to update the balance
func (r *BalanceRepository) GetByMerchantIDForUpdate(tx *sql.Tx, merchantID string) (*model.MerchantBalance, error) {
	if merchantID == "" {
		return nil, errors.New("merchant ID cannot be empty")
	}

	if tx == nil {
		return nil, errors.New("transaction cannot be nil - use GetByMerchantID for non-transactional reads")
	}

	query := `
		SELECT
			id, merchant_id,
			pending_vnd, available_vnd, total_vnd, reserved_vnd,
			total_received_vnd, total_paid_out_vnd, total_fees_vnd,
			total_payments_count, total_payouts_count,
			last_payment_at, last_payout_at,
			version, created_at, updated_at
		FROM merchant_balances
		WHERE merchant_id = $1
		FOR UPDATE
	`

	balance := &model.MerchantBalance{}
	err := tx.QueryRow(query, merchantID).Scan(
		&balance.ID,
		&balance.MerchantID,
		&balance.PendingVND,
		&balance.AvailableVND,
		&balance.TotalVND,
		&balance.ReservedVND,
		&balance.TotalReceivedVND,
		&balance.TotalPaidOutVND,
		&balance.TotalFeesVND,
		&balance.TotalPaymentsCount,
		&balance.TotalPayoutsCount,
		&balance.LastPaymentAt,
		&balance.LastPayoutAt,
		&balance.Version,
		&balance.CreatedAt,
		&balance.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrBalanceNotFound
		}
		return nil, fmt.Errorf("failed to get balance for update: %w", err)
	}

	return balance, nil
}

// Create creates a new balance record for a merchant
// Note: This is typically handled automatically by the database trigger when a merchant is created
func (r *BalanceRepository) Create(balance *model.MerchantBalance) error {
	if balance == nil {
		return errors.New("balance cannot be nil")
	}

	if balance.MerchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	query := `
		INSERT INTO merchant_balances (
			id, merchant_id,
			pending_vnd, available_vnd, total_vnd, reserved_vnd,
			total_received_vnd, total_paid_out_vnd, total_fees_vnd,
			total_payments_count, total_payouts_count,
			last_payment_at, last_payout_at,
			version, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if balance.CreatedAt.IsZero() {
		balance.CreatedAt = now
	}
	if balance.UpdatedAt.IsZero() {
		balance.UpdatedAt = now
	}

	// Set default values if not provided
	if balance.Version == 0 {
		balance.Version = 1
	}

	err := r.db.QueryRow(
		query,
		balance.ID,
		balance.MerchantID,
		balance.PendingVND,
		balance.AvailableVND,
		balance.TotalVND,
		balance.ReservedVND,
		balance.TotalReceivedVND,
		balance.TotalPaidOutVND,
		balance.TotalFeesVND,
		balance.TotalPaymentsCount,
		balance.TotalPayoutsCount,
		balance.LastPaymentAt,
		balance.LastPayoutAt,
		balance.Version,
		balance.CreatedAt,
		balance.UpdatedAt,
	).Scan(&balance.ID, &balance.CreatedAt, &balance.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create balance: %w", err)
	}

	return nil
}

// Update updates a merchant's balance with optimistic locking
// The version field is used to prevent concurrent updates
func (r *BalanceRepository) Update(balance *model.MerchantBalance) error {
	if balance == nil {
		return errors.New("balance cannot be nil")
	}

	if balance.MerchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	// Validate balance state before updating
	if !balance.Validate() {
		return ErrInvalidBalanceUpdate
	}

	query := `
		UPDATE merchant_balances
		SET
			pending_vnd = $1,
			available_vnd = $2,
			total_vnd = $3,
			reserved_vnd = $4,
			total_received_vnd = $5,
			total_paid_out_vnd = $6,
			total_fees_vnd = $7,
			total_payments_count = $8,
			total_payouts_count = $9,
			last_payment_at = $10,
			last_payout_at = $11,
			version = $12,
			updated_at = $13
		WHERE merchant_id = $14 AND version = $15
		RETURNING id, created_at, updated_at
	`

	oldVersion := balance.Version
	balance.Version++ // Increment version for optimistic locking
	balance.UpdatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		balance.PendingVND,
		balance.AvailableVND,
		balance.TotalVND,
		balance.ReservedVND,
		balance.TotalReceivedVND,
		balance.TotalPaidOutVND,
		balance.TotalFeesVND,
		balance.TotalPaymentsCount,
		balance.TotalPayoutsCount,
		balance.LastPaymentAt,
		balance.LastPayoutAt,
		balance.Version,
		balance.UpdatedAt,
		balance.MerchantID,
		oldVersion,
	).Scan(&balance.ID, &balance.CreatedAt, &balance.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrBalanceVersionMismatch
		}
		return fmt.Errorf("failed to update balance: %w", err)
	}

	return nil
}

// UpdateInTransaction updates a merchant's balance within a transaction with optimistic locking
func (r *BalanceRepository) UpdateInTransaction(tx *sql.Tx, balance *model.MerchantBalance) error {
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}

	if balance == nil {
		return errors.New("balance cannot be nil")
	}

	if balance.MerchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	// Validate balance state before updating
	if !balance.Validate() {
		return ErrInvalidBalanceUpdate
	}

	query := `
		UPDATE merchant_balances
		SET
			pending_vnd = $1,
			available_vnd = $2,
			total_vnd = $3,
			reserved_vnd = $4,
			total_received_vnd = $5,
			total_paid_out_vnd = $6,
			total_fees_vnd = $7,
			total_payments_count = $8,
			total_payouts_count = $9,
			last_payment_at = $10,
			last_payout_at = $11,
			version = $12,
			updated_at = $13
		WHERE merchant_id = $14 AND version = $15
		RETURNING id, created_at, updated_at
	`

	oldVersion := balance.Version
	balance.Version++ // Increment version for optimistic locking
	balance.UpdatedAt = time.Now()

	err := tx.QueryRow(
		query,
		balance.PendingVND,
		balance.AvailableVND,
		balance.TotalVND,
		balance.ReservedVND,
		balance.TotalReceivedVND,
		balance.TotalPaidOutVND,
		balance.TotalFeesVND,
		balance.TotalPaymentsCount,
		balance.TotalPayoutsCount,
		balance.LastPaymentAt,
		balance.LastPayoutAt,
		balance.Version,
		balance.UpdatedAt,
		balance.MerchantID,
		oldVersion,
	).Scan(&balance.ID, &balance.CreatedAt, &balance.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrBalanceVersionMismatch
		}
		return fmt.Errorf("failed to update balance in transaction: %w", err)
	}

	return nil
}

// IncrementAvailable atomically increments the available balance
func (r *BalanceRepository) IncrementAvailable(merchantID string, amount decimal.Decimal) error {
	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	if amount.LessThan(decimal.Zero) {
		return ErrNegativeAmount
	}

	query := `
		UPDATE merchant_balances
		SET
			available_vnd = available_vnd + $1,
			total_vnd = total_vnd + $1,
			version = version + 1,
			updated_at = $2
		WHERE merchant_id = $3
		RETURNING id
	`

	var id string
	err := r.db.QueryRow(query, amount, time.Now(), merchantID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrBalanceNotFound
		}
		return fmt.Errorf("failed to increment available balance: %w", err)
	}

	return nil
}

// DecrementAvailable atomically decrements the available balance
// Returns ErrInsufficientBalance if the operation would result in a negative balance
func (r *BalanceRepository) DecrementAvailable(merchantID string, amount decimal.Decimal) error {
	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	if amount.LessThan(decimal.Zero) {
		return ErrNegativeAmount
	}

	// First, check if there's sufficient balance
	query := `
		UPDATE merchant_balances
		SET
			available_vnd = available_vnd - $1,
			total_vnd = total_vnd - $1,
			version = version + 1,
			updated_at = $2
		WHERE merchant_id = $3 AND available_vnd >= $1
		RETURNING id
	`

	var id string
	err := r.db.QueryRow(query, amount, time.Now(), merchantID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			// Check if the balance exists or if it's insufficient
			balance, getErr := r.GetByMerchantID(merchantID)
			if getErr != nil {
				if getErr == ErrBalanceNotFound {
					return ErrBalanceNotFound
				}
				return fmt.Errorf("failed to verify balance: %w", getErr)
			}
			if balance.AvailableVND.LessThan(amount) {
				return ErrInsufficientBalance
			}
			return fmt.Errorf("failed to decrement balance: %w", err)
		}
		return fmt.Errorf("failed to decrement available balance: %w", err)
	}

	return nil
}

// IncrementPending atomically increments the pending balance
func (r *BalanceRepository) IncrementPending(merchantID string, amount decimal.Decimal) error {
	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	if amount.LessThan(decimal.Zero) {
		return ErrNegativeAmount
	}

	query := `
		UPDATE merchant_balances
		SET
			pending_vnd = pending_vnd + $1,
			total_vnd = total_vnd + $1,
			version = version + 1,
			updated_at = $2
		WHERE merchant_id = $3
		RETURNING id
	`

	var id string
	err := r.db.QueryRow(query, amount, time.Now(), merchantID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrBalanceNotFound
		}
		return fmt.Errorf("failed to increment pending balance: %w", err)
	}

	return nil
}

// DecrementPending atomically decrements the pending balance
func (r *BalanceRepository) DecrementPending(merchantID string, amount decimal.Decimal) error {
	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	if amount.LessThan(decimal.Zero) {
		return ErrNegativeAmount
	}

	query := `
		UPDATE merchant_balances
		SET
			pending_vnd = pending_vnd - $1,
			total_vnd = total_vnd - $1,
			version = version + 1,
			updated_at = $2
		WHERE merchant_id = $3 AND pending_vnd >= $1
		RETURNING id
	`

	var id string
	err := r.db.QueryRow(query, amount, time.Now(), merchantID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			// Check if the balance exists or if it's insufficient
			balance, getErr := r.GetByMerchantID(merchantID)
			if getErr != nil {
				if getErr == ErrBalanceNotFound {
					return ErrBalanceNotFound
				}
				return fmt.Errorf("failed to verify balance: %w", getErr)
			}
			if balance.PendingVND.LessThan(amount) {
				return ErrInsufficientBalance
			}
			return fmt.Errorf("failed to decrement balance: %w", err)
		}
		return fmt.Errorf("failed to decrement pending balance: %w", err)
	}

	return nil
}

// IncrementReserved atomically increments the reserved balance
func (r *BalanceRepository) IncrementReserved(merchantID string, amount decimal.Decimal) error {
	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	if amount.LessThan(decimal.Zero) {
		return ErrNegativeAmount
	}

	// Reserved balance is deducted from available but doesn't affect total
	query := `
		UPDATE merchant_balances
		SET
			reserved_vnd = reserved_vnd + $1,
			version = version + 1,
			updated_at = $2
		WHERE merchant_id = $3 AND available_vnd >= reserved_vnd + $1
		RETURNING id
	`

	var id string
	err := r.db.QueryRow(query, amount, time.Now(), merchantID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrInsufficientBalance
		}
		return fmt.Errorf("failed to increment reserved balance: %w", err)
	}

	return nil
}

// DecrementReserved atomically decrements the reserved balance
func (r *BalanceRepository) DecrementReserved(merchantID string, amount decimal.Decimal) error {
	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	if amount.LessThan(decimal.Zero) {
		return ErrNegativeAmount
	}

	query := `
		UPDATE merchant_balances
		SET
			reserved_vnd = reserved_vnd - $1,
			version = version + 1,
			updated_at = $2
		WHERE merchant_id = $3 AND reserved_vnd >= $1
		RETURNING id
	`

	var id string
	err := r.db.QueryRow(query, amount, time.Now(), merchantID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrBalanceNotFound
		}
		return fmt.Errorf("failed to decrement reserved balance: %w", err)
	}

	return nil
}

// RecordPayment updates balance when a payment is received and confirmed
// This includes incrementing statistics and updating timestamps
func (r *BalanceRepository) RecordPayment(tx *sql.Tx, merchantID string, amountVND decimal.Decimal) error {
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}

	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	if amountVND.LessThanOrEqual(decimal.Zero) {
		return errors.New("payment amount must be positive")
	}

	query := `
		UPDATE merchant_balances
		SET
			available_vnd = available_vnd + $1,
			total_vnd = total_vnd + $1,
			total_received_vnd = total_received_vnd + $1,
			total_payments_count = total_payments_count + 1,
			last_payment_at = $2,
			version = version + 1,
			updated_at = $2
		WHERE merchant_id = $3
		RETURNING id
	`

	now := time.Now()
	var id string
	err := tx.QueryRow(query, amountVND, now, merchantID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrBalanceNotFound
		}
		return fmt.Errorf("failed to record payment: %w", err)
	}

	return nil
}

// RecordPayout updates balance when a payout is completed
// This includes decrementing balances and updating statistics
func (r *BalanceRepository) RecordPayout(tx *sql.Tx, merchantID string, amountVND, feeVND decimal.Decimal) error {
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}

	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	if amountVND.LessThanOrEqual(decimal.Zero) {
		return errors.New("payout amount must be positive")
	}

	if feeVND.LessThan(decimal.Zero) {
		return errors.New("fee cannot be negative")
	}

	totalDeduction := amountVND.Add(feeVND)

	query := `
		UPDATE merchant_balances
		SET
			available_vnd = available_vnd - $1,
			total_vnd = total_vnd - $1,
			reserved_vnd = reserved_vnd - $1,
			total_paid_out_vnd = total_paid_out_vnd + $2,
			total_fees_vnd = total_fees_vnd + $3,
			total_payouts_count = total_payouts_count + 1,
			last_payout_at = $4,
			version = version + 1,
			updated_at = $4
		WHERE merchant_id = $5 AND available_vnd >= $1 AND reserved_vnd >= $1
		RETURNING id
	`

	now := time.Now()
	var id string
	err := tx.QueryRow(query, totalDeduction, amountVND, feeVND, now, merchantID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrInsufficientBalance
		}
		return fmt.Errorf("failed to record payout: %w", err)
	}

	return nil
}

// List retrieves all merchant balances (primarily for admin/reporting purposes)
func (r *BalanceRepository) List(limit, offset int) ([]*model.MerchantBalance, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, merchant_id,
			pending_vnd, available_vnd, total_vnd, reserved_vnd,
			total_received_vnd, total_paid_out_vnd, total_fees_vnd,
			total_payments_count, total_payouts_count,
			last_payment_at, last_payout_at,
			version, created_at, updated_at
		FROM merchant_balances
		ORDER BY total_vnd DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list balances: %w", err)
	}
	defer rows.Close()

	var balances []*model.MerchantBalance
	for rows.Next() {
		balance := &model.MerchantBalance{}
		err := rows.Scan(
			&balance.ID,
			&balance.MerchantID,
			&balance.PendingVND,
			&balance.AvailableVND,
			&balance.TotalVND,
			&balance.ReservedVND,
			&balance.TotalReceivedVND,
			&balance.TotalPaidOutVND,
			&balance.TotalFeesVND,
			&balance.TotalPaymentsCount,
			&balance.TotalPayoutsCount,
			&balance.LastPaymentAt,
			&balance.LastPayoutAt,
			&balance.Version,
			&balance.CreatedAt,
			&balance.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan balance: %w", err)
		}
		balances = append(balances, balance)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating balances: %w", err)
	}

	return balances, nil
}

// IncrementPendingTx atomically increments the pending balance within a transaction
func (r *BalanceRepository) IncrementPendingTx(tx *sql.Tx, merchantID string, amount decimal.Decimal) error {
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}

	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	if amount.LessThan(decimal.Zero) {
		return ErrNegativeAmount
	}

	query := `
		UPDATE merchant_balances
		SET
			pending_vnd = pending_vnd + $1,
			total_vnd = total_vnd + $1,
			total_received_vnd = total_received_vnd + $1,
			total_payments_count = total_payments_count + 1,
			last_payment_at = $2,
			version = version + 1,
			updated_at = $2
		WHERE merchant_id = $3
		RETURNING id
	`

	now := time.Now()
	var id string
	err := tx.QueryRow(query, amount, now, merchantID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrBalanceNotFound
		}
		return fmt.Errorf("failed to increment pending balance: %w", err)
	}

	return nil
}

// ConvertPendingToAvailableTx moves balance from pending to available (after OTC conversion)
// This is typically done after confirming a payment
func (r *BalanceRepository) ConvertPendingToAvailableTx(tx *sql.Tx, merchantID string, amount, fee decimal.Decimal) error {
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}

	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("amount must be positive")
	}

	if fee.LessThan(decimal.Zero) {
		return errors.New("fee cannot be negative")
	}

	if fee.GreaterThanOrEqual(amount) {
		return errors.New("fee cannot be greater than or equal to amount")
	}

	netAmount := amount.Sub(fee)

	query := `
		UPDATE merchant_balances
		SET
			pending_vnd = pending_vnd - $1,
			available_vnd = available_vnd + $2,
			total_fees_vnd = total_fees_vnd + $3,
			version = version + 1,
			updated_at = $4
		WHERE merchant_id = $5 AND pending_vnd >= $1
		RETURNING id
	`

	now := time.Now()
	var id string
	err := tx.QueryRow(query, amount, netAmount, fee, now, merchantID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrInsufficientBalance
		}
		return fmt.Errorf("failed to convert pending to available: %w", err)
	}

	return nil
}

// ReserveBalanceTx reserves balance for a pending payout within a transaction
func (r *BalanceRepository) ReserveBalanceTx(tx *sql.Tx, merchantID string, amount decimal.Decimal) error {
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}

	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	if amount.LessThan(decimal.Zero) {
		return ErrNegativeAmount
	}

	// Reserved balance is deducted from available but doesn't affect total
	query := `
		UPDATE merchant_balances
		SET
			reserved_vnd = reserved_vnd + $1,
			version = version + 1,
			updated_at = $2
		WHERE merchant_id = $3 AND available_vnd >= reserved_vnd + $1
		RETURNING id
	`

	now := time.Now()
	var id string
	err := tx.QueryRow(query, amount, now, merchantID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrInsufficientBalance
		}
		return fmt.Errorf("failed to reserve balance: %w", err)
	}

	return nil
}

// ReleaseReservedBalanceTx releases reserved balance within a transaction
func (r *BalanceRepository) ReleaseReservedBalanceTx(tx *sql.Tx, merchantID string, amount decimal.Decimal) error {
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}

	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	if amount.LessThan(decimal.Zero) {
		return ErrNegativeAmount
	}

	query := `
		UPDATE merchant_balances
		SET
			reserved_vnd = reserved_vnd - $1,
			version = version + 1,
			updated_at = $2
		WHERE merchant_id = $3 AND reserved_vnd >= $1
		RETURNING id
	`

	now := time.Now()
	var id string
	err := tx.QueryRow(query, amount, now, merchantID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrBalanceNotFound
		}
		return fmt.Errorf("failed to release reserved balance: %w", err)
	}

	return nil
}

// DeductBalanceTx deducts from available and reserved balance (when payout is completed)
func (r *BalanceRepository) DeductBalanceTx(tx *sql.Tx, merchantID string, amount, fee decimal.Decimal) error {
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}

	if merchantID == "" {
		return errors.New("merchant ID cannot be empty")
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("amount must be positive")
	}

	if fee.LessThan(decimal.Zero) {
		return errors.New("fee cannot be negative")
	}

	totalDeduction := amount.Add(fee)

	query := `
		UPDATE merchant_balances
		SET
			available_vnd = available_vnd - $1,
			total_vnd = total_vnd - $1,
			reserved_vnd = reserved_vnd - $1,
			total_paid_out_vnd = total_paid_out_vnd + $2,
			total_fees_vnd = total_fees_vnd + $3,
			total_payouts_count = total_payouts_count + 1,
			last_payout_at = $4,
			version = version + 1,
			updated_at = $4
		WHERE merchant_id = $5 AND available_vnd >= $1 AND reserved_vnd >= $1
		RETURNING id
	`

	now := time.Now()
	var id string
	err := tx.QueryRow(query, totalDeduction, amount, fee, now, merchantID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrInsufficientBalance
		}
		return fmt.Errorf("failed to deduct balance: %w", err)
	}

	return nil
}
