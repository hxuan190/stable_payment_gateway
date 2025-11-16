package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/shopspring/decimal"
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
	db *sql.DB
}

// NewLedgerRepository creates a new ledger repository
func NewLedgerRepository(db *sql.DB) *LedgerRepository {
	return &LedgerRepository{
		db: db,
	}
}

// CreateEntry inserts a single ledger entry into the database
// Note: For double-entry accounting, you should use CreateEntries to ensure atomicity
func (r *LedgerRepository) CreateEntry(entry *model.LedgerEntry) error {
	if entry == nil {
		return ErrInvalidLedgerEntry
	}

	// Validate entry
	if err := r.validateEntry(entry); err != nil {
		return err
	}

	query := `
		INSERT INTO ledger_entries (
			id,
			debit_account,
			credit_account,
			amount,
			currency,
			reference_type,
			reference_id,
			merchant_id,
			description,
			transaction_group,
			entry_type,
			metadata,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
		RETURNING id, created_at
	`

	now := time.Now()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}

	// Generate ID if not provided
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	// Generate transaction group if not provided
	if entry.TransactionGroup == "" {
		entry.TransactionGroup = uuid.New().String()
	}

	err := r.db.QueryRow(
		query,
		entry.ID,
		entry.DebitAccount,
		entry.CreditAccount,
		entry.Amount,
		entry.Currency,
		entry.ReferenceType,
		entry.ReferenceID,
		entry.MerchantID,
		entry.Description,
		entry.TransactionGroup,
		entry.EntryType,
		entry.Metadata,
		entry.CreatedAt,
	).Scan(&entry.ID, &entry.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create ledger entry: %w", err)
	}

	return nil
}

// CreateEntries inserts multiple ledger entries as a single atomic transaction
// This ensures that double-entry accounting rules are enforced
func (r *LedgerRepository) CreateEntries(entries []*model.LedgerEntry) error {
	if len(entries) == 0 {
		return errors.New("no entries to create")
	}

	// Validate that all entries have the same transaction group
	transactionGroup := entries[0].TransactionGroup
	if transactionGroup == "" {
		transactionGroup = uuid.New().String()
	}

	// Calculate totals for validation
	totalDebits := decimal.Zero
	totalCredits := decimal.Zero
	currency := entries[0].Currency

	for _, entry := range entries {
		if entry == nil {
			return ErrInvalidLedgerEntry
		}

		// Ensure all entries have the same transaction group
		if entry.TransactionGroup == "" {
			entry.TransactionGroup = transactionGroup
		} else if entry.TransactionGroup != transactionGroup {
			return errors.New("all entries must have the same transaction group")
		}

		// Ensure all entries have the same currency
		if entry.Currency != currency {
			return errors.New("all entries must have the same currency")
		}

		// Validate entry
		if err := r.validateEntry(entry); err != nil {
			return err
		}

		// Calculate totals
		if entry.EntryType == model.EntryTypeDebit {
			totalDebits = totalDebits.Add(entry.Amount)
		} else {
			totalCredits = totalCredits.Add(entry.Amount)
		}
	}

	// Validate that debits equal credits
	if !totalDebits.Equal(totalCredits) {
		return fmt.Errorf("%w: debits=%s, credits=%s", ErrUnbalancedTransaction, totalDebits.String(), totalCredits.String())
	}

	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO ledger_entries (
			id,
			debit_account,
			credit_account,
			amount,
			currency,
			reference_type,
			reference_id,
			merchant_id,
			description,
			transaction_group,
			entry_type,
			metadata,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
		RETURNING id, created_at
	`

	now := time.Now()

	for _, entry := range entries {
		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = now
		}

		// Generate ID if not provided
		if entry.ID == "" {
			entry.ID = uuid.New().String()
		}

		err := tx.QueryRow(
			query,
			entry.ID,
			entry.DebitAccount,
			entry.CreditAccount,
			entry.Amount,
			entry.Currency,
			entry.ReferenceType,
			entry.ReferenceID,
			entry.MerchantID,
			entry.Description,
			entry.TransactionGroup,
			entry.EntryType,
			entry.Metadata,
			entry.CreatedAt,
		).Scan(&entry.ID, &entry.CreatedAt)

		if err != nil {
			return fmt.Errorf("failed to create ledger entry: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a ledger entry by ID
func (r *LedgerRepository) GetByID(id string) (*model.LedgerEntry, error) {
	if id == "" {
		return nil, errors.New("ledger entry ID cannot be empty")
	}

	query := `
		SELECT
			id,
			debit_account,
			credit_account,
			amount,
			currency,
			reference_type,
			reference_id,
			merchant_id,
			description,
			transaction_group,
			entry_type,
			metadata,
			created_at
		FROM ledger_entries
		WHERE id = $1
	`

	entry := &model.LedgerEntry{}
	err := r.db.QueryRow(query, id).Scan(
		&entry.ID,
		&entry.DebitAccount,
		&entry.CreditAccount,
		&entry.Amount,
		&entry.Currency,
		&entry.ReferenceType,
		&entry.ReferenceID,
		&entry.MerchantID,
		&entry.Description,
		&entry.TransactionGroup,
		&entry.EntryType,
		&entry.Metadata,
		&entry.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrLedgerEntryNotFound
		}
		return nil, fmt.Errorf("failed to get ledger entry by ID: %w", err)
	}

	return entry, nil
}

// GetByMerchant retrieves ledger entries for a specific merchant with pagination
func (r *LedgerRepository) GetByMerchant(merchantID string, limit, offset int) ([]*model.LedgerEntry, error) {
	if merchantID == "" {
		return nil, errors.New("merchant ID cannot be empty")
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id,
			debit_account,
			credit_account,
			amount,
			currency,
			reference_type,
			reference_id,
			merchant_id,
			description,
			transaction_group,
			entry_type,
			metadata,
			created_at
		FROM ledger_entries
		WHERE merchant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, merchantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entries by merchant: %w", err)
	}
	defer rows.Close()

	entries := make([]*model.LedgerEntry, 0)
	for rows.Next() {
		entry := &model.LedgerEntry{}
		err := rows.Scan(
			&entry.ID,
			&entry.DebitAccount,
			&entry.CreditAccount,
			&entry.Amount,
			&entry.Currency,
			&entry.ReferenceType,
			&entry.ReferenceID,
			&entry.MerchantID,
			&entry.Description,
			&entry.TransactionGroup,
			&entry.EntryType,
			&entry.Metadata,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ledger entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ledger entries: %w", err)
	}

	return entries, nil
}

// GetByReference retrieves ledger entries by reference type and ID
func (r *LedgerRepository) GetByReference(refType model.ReferenceType, refID string) ([]*model.LedgerEntry, error) {
	if refType == "" {
		return nil, errors.New("reference type cannot be empty")
	}
	if refID == "" {
		return nil, errors.New("reference ID cannot be empty")
	}

	query := `
		SELECT
			id,
			debit_account,
			credit_account,
			amount,
			currency,
			reference_type,
			reference_id,
			merchant_id,
			description,
			transaction_group,
			entry_type,
			metadata,
			created_at
		FROM ledger_entries
		WHERE reference_type = $1 AND reference_id = $2
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, refType, refID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entries by reference: %w", err)
	}
	defer rows.Close()

	entries := make([]*model.LedgerEntry, 0)
	for rows.Next() {
		entry := &model.LedgerEntry{}
		err := rows.Scan(
			&entry.ID,
			&entry.DebitAccount,
			&entry.CreditAccount,
			&entry.Amount,
			&entry.Currency,
			&entry.ReferenceType,
			&entry.ReferenceID,
			&entry.MerchantID,
			&entry.Description,
			&entry.TransactionGroup,
			&entry.EntryType,
			&entry.Metadata,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ledger entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ledger entries: %w", err)
	}

	return entries, nil
}

// GetByTransactionGroup retrieves all ledger entries in a transaction group
func (r *LedgerRepository) GetByTransactionGroup(transactionGroup string) ([]*model.LedgerEntry, error) {
	if transactionGroup == "" {
		return nil, ErrEmptyTransactionGroup
	}

	query := `
		SELECT
			id,
			debit_account,
			credit_account,
			amount,
			currency,
			reference_type,
			reference_id,
			merchant_id,
			description,
			transaction_group,
			entry_type,
			metadata,
			created_at
		FROM ledger_entries
		WHERE transaction_group = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, transactionGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entries by transaction group: %w", err)
	}
	defer rows.Close()

	entries := make([]*model.LedgerEntry, 0)
	for rows.Next() {
		entry := &model.LedgerEntry{}
		err := rows.Scan(
			&entry.ID,
			&entry.DebitAccount,
			&entry.CreditAccount,
			&entry.Amount,
			&entry.Currency,
			&entry.ReferenceType,
			&entry.ReferenceID,
			&entry.MerchantID,
			&entry.Description,
			&entry.TransactionGroup,
			&entry.EntryType,
			&entry.Metadata,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ledger entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ledger entries: %w", err)
	}

	return entries, nil
}

// GetAccountBalance calculates the balance for a specific account
// Debits increase asset/expense accounts, credits increase liability/revenue accounts
func (r *LedgerRepository) GetAccountBalance(accountName string) (decimal.Decimal, error) {
	if accountName == "" {
		return decimal.Zero, errors.New("account name cannot be empty")
	}

	query := `
		SELECT
			COALESCE(SUM(CASE WHEN debit_account = $1 THEN amount ELSE 0 END), 0) as total_debits,
			COALESCE(SUM(CASE WHEN credit_account = $1 THEN amount ELSE 0 END), 0) as total_credits
		FROM ledger_entries
		WHERE debit_account = $1 OR credit_account = $1
	`

	var totalDebits, totalCredits decimal.Decimal
	err := r.db.QueryRow(query, accountName).Scan(&totalDebits, &totalCredits)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get account balance: %w", err)
	}

	// For asset and expense accounts: balance = debits - credits
	// For liability, equity, and revenue accounts: balance = credits - debits
	// We return the net amount (debits - credits) and let the caller interpret based on account type
	balance := totalDebits.Sub(totalCredits)

	return balance, nil
}

// Count returns the total number of ledger entries
func (r *LedgerRepository) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM ledger_entries`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count ledger entries: %w", err)
	}

	return count, nil
}

// CountByMerchant returns the count of ledger entries for a specific merchant
func (r *LedgerRepository) CountByMerchant(merchantID string) (int64, error) {
	if merchantID == "" {
		return 0, errors.New("merchant ID cannot be empty")
	}

	query := `SELECT COUNT(*) FROM ledger_entries WHERE merchant_id = $1`

	var count int64
	err := r.db.QueryRow(query, merchantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count ledger entries by merchant: %w", err)
	}

	return count, nil
}

// ValidateDoubleEntry validates that all transactions in the ledger balance
// This is useful for periodic integrity checks
func (r *LedgerRepository) ValidateDoubleEntry() error {
	query := `
		SELECT
			transaction_group,
			currency,
			SUM(CASE WHEN entry_type = 'debit' THEN amount ELSE 0 END) as total_debits,
			SUM(CASE WHEN entry_type = 'credit' THEN amount ELSE 0 END) as total_credits
		FROM ledger_entries
		GROUP BY transaction_group, currency
		HAVING SUM(CASE WHEN entry_type = 'debit' THEN amount ELSE 0 END) !=
		       SUM(CASE WHEN entry_type = 'credit' THEN amount ELSE 0 END)
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to validate double entry: %w", err)
	}
	defer rows.Close()

	unbalancedGroups := make([]string, 0)
	for rows.Next() {
		var group, currency string
		var debits, credits decimal.Decimal
		err := rows.Scan(&group, &currency, &debits, &credits)
		if err != nil {
			return fmt.Errorf("failed to scan unbalanced transaction: %w", err)
		}
		unbalancedGroups = append(unbalancedGroups, fmt.Sprintf("%s (debits: %s, credits: %s %s)", group, debits, credits, currency))
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating unbalanced transactions: %w", err)
	}

	if len(unbalancedGroups) > 0 {
		return fmt.Errorf("found %d unbalanced transaction groups: %v", len(unbalancedGroups), unbalancedGroups)
	}

	return nil
}

// validateEntry validates a ledger entry before insertion
func (r *LedgerRepository) validateEntry(entry *model.LedgerEntry) error {
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
	if entry.EntryType != model.EntryTypeDebit && entry.EntryType != model.EntryTypeCredit {
		return errors.New("entry type must be either 'debit' or 'credit'")
	}

	return nil
}
