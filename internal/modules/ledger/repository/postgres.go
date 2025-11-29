package repository

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	ledgerDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/ledger/domain"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	// ErrLedgerEntryNotFound is returned when a ledger entry is not found
	ErrLedgerEntryNotFound = errors.New("ledger entry not found")
	// ErrInvalidLedgerEntry is returned when a ledger entry is invalid
	ErrInvalidLedgerEntry = errors.New("invalid ledger entry")
	// ErrUnbalancedTransaction is returned when debits don't equal credits
	ErrUnbalancedTransaction = errors.New("transaction is not balanced: total debits must equal total credits")
	// ErrEmptyTransactionGroup is returned when transaction group is empty
	ErrEmptyTransactionGroup = errors.New("transaction group cannot be empty")
)

// LedgerRepository handles database operations for ledger entries
// This is the core of the double-entry accounting system
type LedgerRepository struct {
	db *gorm.DB
}

// NewLedgerRepository creates a new ledger repository
func NewLedgerRepository(db *gorm.DB) *LedgerRepository {
	return &LedgerRepository{
		db: db,
	}
}

func (r *LedgerRepository) CreateEntry(entry *ledgerDomain.LedgerEntry) error {
	if entry == nil {
		return ErrInvalidLedgerEntry
	}

	if err := r.validateEntry(entry); err != nil {
		return err
	}

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	if entry.TransactionGroup == "" {
		entry.TransactionGroup = uuid.New().String()
	}

	if err := r.db.Create(entry).Error; err != nil {
		return fmt.Errorf("failed to create ledger entry: %w", err)
	}

	return nil
}

func (r *LedgerRepository) CreateEntries(entries []*ledgerDomain.LedgerEntry) error {
	if len(entries) == 0 {
		return errors.New("no entries to create")
	}

	transactionGroup := entries[0].TransactionGroup
	if transactionGroup == "" {
		transactionGroup = uuid.New().String()
	}

	totalDebits := decimal.Zero
	totalCredits := decimal.Zero
	currency := entries[0].Currency

	for _, entry := range entries {
		if entry == nil {
			return ErrInvalidLedgerEntry
		}

		if entry.TransactionGroup == "" {
			entry.TransactionGroup = transactionGroup
		} else if entry.TransactionGroup != transactionGroup {
			return errors.New("all entries must have the same transaction group")
		}

		if entry.Currency != currency {
			return errors.New("all entries must have the same currency")
		}

		if err := r.validateEntry(entry); err != nil {
			return err
		}

		if entry.EntryType == ledgerDomain.EntryTypeDebit {
			totalDebits = totalDebits.Add(entry.Amount)
		} else {
			totalCredits = totalCredits.Add(entry.Amount)
		}
	}

	if !totalDebits.Equal(totalCredits) {
		return fmt.Errorf("%w: debits=%s, credits=%s", ErrUnbalancedTransaction, totalDebits.String(), totalCredits.String())
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, entry := range entries {
			if entry.ID == "" {
				entry.ID = uuid.New().String()
			}

			if err := tx.Create(entry).Error; err != nil {
				return fmt.Errorf("failed to create ledger entry: %w", err)
			}
		}
		return nil
	})
}

func (r *LedgerRepository) GetByID(id string) (*ledgerDomain.LedgerEntry, error) {
	if id == "" {
		return nil, errors.New("ledger entry ID cannot be empty")
	}

	entry := &ledgerDomain.LedgerEntry{}
	err := r.db.Where("id = ?", id).First(entry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLedgerEntryNotFound
		}
		return nil, fmt.Errorf("failed to get ledger entry by ID: %w", err)
	}

	return entry, nil
}

func (r *LedgerRepository) GetByMerchant(merchantID string, limit, offset int) ([]*ledgerDomain.LedgerEntry, error) {
	if merchantID == "" {
		return nil, errors.New("merchant ID cannot be empty")
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var entries []*ledgerDomain.LedgerEntry
	err := r.db.Where("merchant_id = ?", merchantID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&entries).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entries by merchant: %w", err)
	}

	return entries, nil
}

func (r *LedgerRepository) GetByReference(refType ledgerDomain.ReferenceType, refID string) ([]*ledgerDomain.LedgerEntry, error) {
	if refType == "" {
		return nil, errors.New("reference type cannot be empty")
	}
	if refID == "" {
		return nil, errors.New("reference ID cannot be empty")
	}

	var entries []*ledgerDomain.LedgerEntry
	err := r.db.Where("reference_type = ? AND reference_id = ?", refType, refID).
		Order("created_at ASC").
		Find(&entries).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entries by reference: %w", err)
	}

	return entries, nil
}

func (r *LedgerRepository) GetByTransactionGroup(transactionGroup string) ([]*ledgerDomain.LedgerEntry, error) {
	if transactionGroup == "" {
		return nil, ErrEmptyTransactionGroup
	}

	var entries []*ledgerDomain.LedgerEntry
	err := r.db.Where("transaction_group = ?", transactionGroup).
		Order("created_at ASC").
		Find(&entries).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entries by transaction group: %w", err)
	}

	return entries, nil
}

func (r *LedgerRepository) GetAccountBalance(accountName string) (decimal.Decimal, error) {
	if accountName == "" {
		return decimal.Zero, errors.New("account name cannot be empty")
	}

	var result struct {
		TotalDebits  decimal.Decimal
		TotalCredits decimal.Decimal
	}

	err := r.db.Model(&ledgerDomain.LedgerEntry{}).
		Select("COALESCE(SUM(CASE WHEN debit_account = ? THEN amount ELSE 0 END), 0) as total_debits, "+
			"COALESCE(SUM(CASE WHEN credit_account = ? THEN amount ELSE 0 END), 0) as total_credits", accountName, accountName).
		Where("debit_account = ? OR credit_account = ?", accountName, accountName).
		Scan(&result).Error
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get account balance: %w", err)
	}

	return result.TotalDebits.Sub(result.TotalCredits), nil
}

func (r *LedgerRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&ledgerDomain.LedgerEntry{}).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count ledger entries: %w", err)
	}

	return count, nil
}

func (r *LedgerRepository) CountByMerchant(merchantID string) (int64, error) {
	if merchantID == "" {
		return 0, errors.New("merchant ID cannot be empty")
	}

	var count int64
	err := r.db.Model(&ledgerDomain.LedgerEntry{}).Where("merchant_id = ?", merchantID).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count ledger entries by merchant: %w", err)
	}

	return count, nil
}

func (r *LedgerRepository) ValidateDoubleEntry() error {
	var results []struct {
		TransactionGroup string
		Currency         string
		TotalDebits      decimal.Decimal
		TotalCredits     decimal.Decimal
	}

	err := r.db.Model(&ledgerDomain.LedgerEntry{}).
		Select("transaction_group, currency, " +
			"SUM(CASE WHEN entry_type = 'debit' THEN amount ELSE 0 END) as total_debits, " +
			"SUM(CASE WHEN entry_type = 'credit' THEN amount ELSE 0 END) as total_credits").
		Group("transaction_group, currency").
		Having("SUM(CASE WHEN entry_type = 'debit' THEN amount ELSE 0 END) != SUM(CASE WHEN entry_type = 'credit' THEN amount ELSE 0 END)").
		Scan(&results).Error
	if err != nil {
		return fmt.Errorf("failed to validate double entry: %w", err)
	}

	if len(results) > 0 {
		unbalancedGroups := make([]string, 0, len(results))
		for _, r := range results {
			unbalancedGroups = append(unbalancedGroups, fmt.Sprintf("%s (debits: %s, credits: %s %s)", r.TransactionGroup, r.TotalDebits, r.TotalCredits, r.Currency))
		}
		return fmt.Errorf("found %d unbalanced transaction groups: %v", len(unbalancedGroups), unbalancedGroups)
	}

	return nil
}

// validateEntry validates a ledger entry before insertion
func (r *LedgerRepository) validateEntry(entry *ledgerDomain.LedgerEntry) error {
	if entry.DebitAccount == "" {
		return errors.New("debit account cannot be empty")
	}
	if entry.CreditAccount == "" {
		return errors.New("credit account cannot be empty")
	}
	if entry.DebitAccount == entry.CreditAccount {
		return errors.New("debit and credit accounts cannot be the same")
	}
	if entry.Amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("amount must be greater than zero")
	}
	if entry.Currency == "" {
		return errors.New("currency cannot be empty")
	}
	if entry.ReferenceType == "" {
		return errors.New("reference type cannot be empty")
	}
	if entry.ReferenceID == "" {
		return errors.New("reference ID cannot be empty")
	}
	if entry.Description == "" {
		return errors.New("description cannot be empty")
	}
	if entry.EntryType == "" {
		return errors.New("entry type cannot be empty")
	}
	if entry.EntryType != ledgerDomain.EntryTypeDebit && entry.EntryType != ledgerDomain.EntryTypeCredit {
		return errors.New("entry type must be either 'debit' or 'credit'")
	}

	return nil
}
