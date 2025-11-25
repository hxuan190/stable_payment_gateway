package repository

import (
	"database/sql"
	"fmt"

	"github.com/shopspring/decimal"
)

type BalanceRepository struct {
	db *sql.DB
}

func NewBalanceRepository(db *sql.DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

func (r *BalanceRepository) IncrementPendingTx(tx *sql.Tx, merchantID string, amount decimal.Decimal) error {
	query := `
		INSERT INTO merchant_balances (merchant_id, pending_balance, available_balance, reserved_balance, currency)
		VALUES ($1, $2, 0, 0, 'VND')
		ON CONFLICT (merchant_id, currency)
		DO UPDATE SET pending_balance = merchant_balances.pending_balance + $2, updated_at = NOW()
	`
	_, err := tx.Exec(query, merchantID, amount)
	if err != nil {
		return fmt.Errorf("failed to increment pending balance: %w", err)
	}
	return nil
}

func (r *BalanceRepository) ConvertPendingToAvailableTx(tx *sql.Tx, merchantID string, grossAmount, feeAmount decimal.Decimal) error {
	netAmount := grossAmount.Sub(feeAmount)
	query := `
		UPDATE merchant_balances
		SET pending_balance = pending_balance - $2,
		    available_balance = available_balance + $3,
		    updated_at = NOW()
		WHERE merchant_id = $1 AND currency = 'VND'
	`
	result, err := tx.Exec(query, merchantID, grossAmount, netAmount)
	if err != nil {
		return fmt.Errorf("failed to convert pending to available: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("merchant balance not found")
	}
	return nil
}

func (r *BalanceRepository) ReserveBalanceTx(tx *sql.Tx, merchantID string, amount decimal.Decimal) error {
	query := `
		UPDATE merchant_balances
		SET available_balance = available_balance - $2,
		    reserved_balance = reserved_balance + $2,
		    updated_at = NOW()
		WHERE merchant_id = $1 AND currency = 'VND' AND available_balance >= $2
	`
	result, err := tx.Exec(query, merchantID, amount)
	if err != nil {
		return fmt.Errorf("failed to reserve balance: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("insufficient available balance")
	}
	return nil
}

func (r *BalanceRepository) DeductBalanceTx(tx *sql.Tx, merchantID string, amount, feeAmount decimal.Decimal) error {
	totalDeduction := amount.Add(feeAmount)
	query := `
		UPDATE merchant_balances
		SET reserved_balance = reserved_balance - amount,
		    updated_at = NOW()
		WHERE merchant_id = $1 AND currency = 'VND' AND reserved_balance >= $2
	`
	result, err := tx.Exec(query, merchantID, totalDeduction)
	if err != nil {
		return fmt.Errorf("failed to deduct balance: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("insufficient reserved balance")
	}
	return nil
}

func (r *BalanceRepository) ReleaseReservedBalanceTx(tx *sql.Tx, merchantID string, amount decimal.Decimal) error {
	query := `
		UPDATE merchant_balances
		SET reserved_balance = reserved_balance - $2,
		    available_balance = available_balance + $2,
		    updated_at = NOW()
		WHERE merchant_id = $1 AND currency = 'VND' AND reserved_balance >= $2
	`
	result, err := tx.Exec(query, merchantID, amount)
	if err != nil {
		return fmt.Errorf("failed to release reserved balance: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("insufficient reserved balance")
	}
	return nil
}
