package repository

import (
	"fmt"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BalanceRepository struct {
	db *gorm.DB
}

func NewBalanceRepository(db *gorm.DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

func (r *BalanceRepository) IncrementPendingTx(tx *gorm.DB, merchantID string, amount decimal.Decimal) error {
	err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "merchant_id"}, {Name: "currency"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"pending_balance": gorm.Expr("merchant_balances.pending_balance + ?", amount)}),
	}).Create(&map[string]interface{}{
		"merchant_id":       merchantID,
		"pending_balance":   amount,
		"available_balance": decimal.Zero,
		"reserved_balance":  decimal.Zero,
		"currency":          "VND",
	}).Error
	if err != nil {
		return fmt.Errorf("failed to increment pending balance: %w", err)
	}
	return nil
}

func (r *BalanceRepository) ConvertPendingToAvailableTx(tx *gorm.DB, merchantID string, grossAmount, feeAmount decimal.Decimal) error {
	netAmount := grossAmount.Sub(feeAmount)
	result := tx.Table("merchant_balances").
		Where("merchant_id = ? AND currency = ?", merchantID, "VND").
		Updates(map[string]interface{}{
			"pending_balance":   gorm.Expr("pending_balance - ?", grossAmount),
			"available_balance": gorm.Expr("available_balance + ?", netAmount),
		})
	if result.Error != nil {
		return fmt.Errorf("failed to convert pending to available: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("merchant balance not found")
	}
	return nil
}

func (r *BalanceRepository) ReserveBalanceTx(tx *gorm.DB, merchantID string, amount decimal.Decimal) error {
	result := tx.Table("merchant_balances").
		Where("merchant_id = ? AND currency = ? AND available_balance >= ?", merchantID, "VND", amount).
		Updates(map[string]interface{}{
			"available_balance": gorm.Expr("available_balance - ?", amount),
			"reserved_balance":  gorm.Expr("reserved_balance + ?", amount),
		})
	if result.Error != nil {
		return fmt.Errorf("failed to reserve balance: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("insufficient available balance")
	}
	return nil
}

func (r *BalanceRepository) DeductBalanceTx(tx *gorm.DB, merchantID string, amount, feeAmount decimal.Decimal) error {
	totalDeduction := amount.Add(feeAmount)
	result := tx.Table("merchant_balances").
		Where("merchant_id = ? AND currency = ? AND reserved_balance >= ?", merchantID, "VND", totalDeduction).
		Update("reserved_balance", gorm.Expr("reserved_balance - ?", totalDeduction))
	if result.Error != nil {
		return fmt.Errorf("failed to deduct balance: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("insufficient reserved balance")
	}
	return nil
}

func (r *BalanceRepository) ReleaseReservedBalanceTx(tx *gorm.DB, merchantID string, amount decimal.Decimal) error {
	result := tx.Table("merchant_balances").
		Where("merchant_id = ? AND currency = ? AND reserved_balance >= ?", merchantID, "VND", amount).
		Updates(map[string]interface{}{
			"reserved_balance":  gorm.Expr("reserved_balance - ?", amount),
			"available_balance": gorm.Expr("available_balance + ?", amount),
		})
	if result.Error != nil {
		return fmt.Errorf("failed to release reserved balance: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("insufficient reserved balance")
	}
	return nil
}
