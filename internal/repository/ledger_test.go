package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB is defined in merchant_test.go but we need it here for clarity
// In practice, it should be in a common test helper file

// cleanupLedgerEntries removes all ledger entries from the test database
func cleanupLedgerEntries(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("DELETE FROM ledger_entries")
	require.NoError(t, err, "Failed to cleanup ledger_entries table")
}

func TestLedgerRepository_CreateEntry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupLedgerEntries(t, db)

	repo := NewLedgerRepository(db)

	t.Run("successfully create ledger entry", func(t *testing.T) {
		entry := &model.LedgerEntry{
			DebitAccount:     "merchant_balance:merchant-123",
			CreditAccount:    "revenue:fees",
			Amount:           decimal.NewFromFloat(100.50),
			Currency:         "VND",
			ReferenceType:    model.ReferenceTypePayment,
			ReferenceID:      uuid.New().String(),
			MerchantID:       sql.NullString{String: "merchant-123", Valid: true},
			Description:      "Payment received",
			TransactionGroup: uuid.New().String(),
			EntryType:        model.EntryTypeDebit,
		}

		err := repo.CreateEntry(entry)
		require.NoError(t, err)
		assert.NotEmpty(t, entry.ID)
		assert.NotZero(t, entry.CreatedAt)
	})

	t.Run("fail when entry is nil", func(t *testing.T) {
		err := repo.CreateEntry(nil)
		assert.ErrorIs(t, err, ErrInvalidLedgerEntry)
	})

	t.Run("fail when debit account is empty", func(t *testing.T) {
		entry := &model.LedgerEntry{
			CreditAccount:    "revenue:fees",
			Amount:           decimal.NewFromFloat(100.50),
			Currency:         "VND",
			ReferenceType:    model.ReferenceTypePayment,
			ReferenceID:      uuid.New().String(),
			Description:      "Payment received",
			TransactionGroup: uuid.New().String(),
			EntryType:        model.EntryTypeDebit,
		}

		err := repo.CreateEntry(entry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "debit account cannot be empty")
	})

	t.Run("fail when amount is zero or negative", func(t *testing.T) {
		entry := &model.LedgerEntry{
			DebitAccount:     "merchant_balance:merchant-123",
			CreditAccount:    "revenue:fees",
			Amount:           decimal.Zero,
			Currency:         "VND",
			ReferenceType:    model.ReferenceTypePayment,
			ReferenceID:      uuid.New().String(),
			Description:      "Payment received",
			TransactionGroup: uuid.New().String(),
			EntryType:        model.EntryTypeDebit,
		}

		err := repo.CreateEntry(entry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "amount must be greater than zero")
	})
}

func TestLedgerRepository_CreateEntries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupLedgerEntries(t, db)

	repo := NewLedgerRepository(db)

	t.Run("successfully create balanced entries", func(t *testing.T) {
		transactionGroup := uuid.New().String()
		referenceID := uuid.New().String()
		amount := decimal.NewFromFloat(1000000.00)

		entries := []*model.LedgerEntry{
			{
				DebitAccount:     "crypto_pool",
				CreditAccount:    "merchant_pending_balance:merchant-123",
				Amount:           amount,
				Currency:         "VND",
				ReferenceType:    model.ReferenceTypePayment,
				ReferenceID:      referenceID,
				MerchantID:       sql.NullString{String: "merchant-123", Valid: true},
				Description:      "Payment received - pending",
				TransactionGroup: transactionGroup,
				EntryType:        model.EntryTypeDebit,
			},
			{
				DebitAccount:     "crypto_pool",
				CreditAccount:    "merchant_pending_balance:merchant-123",
				Amount:           amount,
				Currency:         "VND",
				ReferenceType:    model.ReferenceTypePayment,
				ReferenceID:      referenceID,
				MerchantID:       sql.NullString{String: "merchant-123", Valid: true},
				Description:      "Payment received - pending",
				TransactionGroup: transactionGroup,
				EntryType:        model.EntryTypeCredit,
			},
		}

		err := repo.CreateEntries(entries)
		require.NoError(t, err)

		// Verify both entries were created
		for _, entry := range entries {
			assert.NotEmpty(t, entry.ID)
			assert.NotZero(t, entry.CreatedAt)
		}

		// Verify entries can be retrieved
		retrieved, err := repo.GetByTransactionGroup(transactionGroup)
		require.NoError(t, err)
		assert.Len(t, retrieved, 2)
	})

	t.Run("fail when entries are unbalanced", func(t *testing.T) {
		transactionGroup := uuid.New().String()
		referenceID := uuid.New().String()

		entries := []*model.LedgerEntry{
			{
				DebitAccount:     "crypto_pool",
				CreditAccount:    "merchant_balance:merchant-123",
				Amount:           decimal.NewFromFloat(1000.00),
				Currency:         "VND",
				ReferenceType:    model.ReferenceTypePayment,
				ReferenceID:      referenceID,
				Description:      "Debit entry",
				TransactionGroup: transactionGroup,
				EntryType:        model.EntryTypeDebit,
			},
			{
				DebitAccount:     "crypto_pool",
				CreditAccount:    "merchant_balance:merchant-123",
				Amount:           decimal.NewFromFloat(900.00), // Unbalanced!
				Currency:         "VND",
				ReferenceType:    model.ReferenceTypePayment,
				ReferenceID:      referenceID,
				Description:      "Credit entry",
				TransactionGroup: transactionGroup,
				EntryType:        model.EntryTypeCredit,
			},
		}

		err := repo.CreateEntries(entries)
		assert.ErrorIs(t, err, ErrUnbalancedTransaction)
	})

	t.Run("fail when currencies don't match", func(t *testing.T) {
		transactionGroup := uuid.New().String()
		referenceID := uuid.New().String()
		amount := decimal.NewFromFloat(1000.00)

		entries := []*model.LedgerEntry{
			{
				DebitAccount:     "crypto_pool",
				CreditAccount:    "merchant_balance:merchant-123",
				Amount:           amount,
				Currency:         "VND",
				ReferenceType:    model.ReferenceTypePayment,
				ReferenceID:      referenceID,
				Description:      "Debit entry",
				TransactionGroup: transactionGroup,
				EntryType:        model.EntryTypeDebit,
			},
			{
				DebitAccount:     "crypto_pool",
				CreditAccount:    "merchant_balance:merchant-123",
				Amount:           amount,
				Currency:         "USD", // Different currency!
				ReferenceType:    model.ReferenceTypePayment,
				ReferenceID:      referenceID,
				Description:      "Credit entry",
				TransactionGroup: transactionGroup,
				EntryType:        model.EntryTypeCredit,
			},
		}

		err := repo.CreateEntries(entries)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "same currency")
	})

	t.Run("rollback on error - atomic transaction", func(t *testing.T) {
		transactionGroup := uuid.New().String()
		referenceID := uuid.New().String()
		amount := decimal.NewFromFloat(1000.00)

		// Second entry has invalid data (empty description)
		entries := []*model.LedgerEntry{
			{
				DebitAccount:     "crypto_pool",
				CreditAccount:    "merchant_balance:merchant-123",
				Amount:           amount,
				Currency:         "VND",
				ReferenceType:    model.ReferenceTypePayment,
				ReferenceID:      referenceID,
				Description:      "Valid entry",
				TransactionGroup: transactionGroup,
				EntryType:        model.EntryTypeDebit,
			},
			{
				DebitAccount:     "crypto_pool",
				CreditAccount:    "merchant_balance:merchant-123",
				Amount:           amount,
				Currency:         "VND",
				ReferenceType:    model.ReferenceTypePayment,
				ReferenceID:      referenceID,
				Description:      "", // Invalid!
				TransactionGroup: transactionGroup,
				EntryType:        model.EntryTypeCredit,
			},
		}

		err := repo.CreateEntries(entries)
		assert.Error(t, err)

		// Verify no entries were created (transaction was rolled back)
		count, err := repo.CountByMerchant("merchant-123")
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestLedgerRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupLedgerEntries(t, db)

	repo := NewLedgerRepository(db)

	t.Run("successfully get entry by ID", func(t *testing.T) {
		// Create entry first
		entry := &model.LedgerEntry{
			DebitAccount:     "merchant_balance:merchant-123",
			CreditAccount:    "revenue:fees",
			Amount:           decimal.NewFromFloat(100.50),
			Currency:         "VND",
			ReferenceType:    model.ReferenceTypePayment,
			ReferenceID:      uuid.New().String(),
			MerchantID:       sql.NullString{String: "merchant-123", Valid: true},
			Description:      "Payment received",
			TransactionGroup: uuid.New().String(),
			EntryType:        model.EntryTypeDebit,
		}

		err := repo.CreateEntry(entry)
		require.NoError(t, err)

		// Retrieve entry
		retrieved, err := repo.GetByID(entry.ID)
		require.NoError(t, err)
		assert.Equal(t, entry.ID, retrieved.ID)
		assert.Equal(t, entry.Amount.String(), retrieved.Amount.String())
		assert.Equal(t, entry.DebitAccount, retrieved.DebitAccount)
		assert.Equal(t, entry.CreditAccount, retrieved.CreditAccount)
	})

	t.Run("fail when entry not found", func(t *testing.T) {
		retrieved, err := repo.GetByID(uuid.New().String())
		assert.ErrorIs(t, err, ErrLedgerEntryNotFound)
		assert.Nil(t, retrieved)
	})

	t.Run("fail when ID is empty", func(t *testing.T) {
		retrieved, err := repo.GetByID("")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})
}

func TestLedgerRepository_GetByMerchant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupLedgerEntries(t, db)

	repo := NewLedgerRepository(db)

	merchantID := "merchant-456"

	// Create multiple entries for the merchant
	for i := 0; i < 5; i++ {
		entry := &model.LedgerEntry{
			DebitAccount:     "merchant_balance:" + merchantID,
			CreditAccount:    "revenue:fees",
			Amount:           decimal.NewFromInt(int64((i + 1) * 100)),
			Currency:         "VND",
			ReferenceType:    model.ReferenceTypePayment,
			ReferenceID:      uuid.New().String(),
			MerchantID:       sql.NullString{String: merchantID, Valid: true},
			Description:      "Payment received",
			TransactionGroup: uuid.New().String(),
			EntryType:        model.EntryTypeDebit,
		}
		err := repo.CreateEntry(entry)
		require.NoError(t, err)

		// Add small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	t.Run("successfully get entries by merchant", func(t *testing.T) {
		entries, err := repo.GetByMerchant(merchantID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, entries, 5)

		// Verify entries are ordered by created_at DESC
		for i := 0; i < len(entries)-1; i++ {
			assert.True(t, entries[i].CreatedAt.After(entries[i+1].CreatedAt) ||
				entries[i].CreatedAt.Equal(entries[i+1].CreatedAt))
		}
	})

	t.Run("pagination works correctly", func(t *testing.T) {
		// Get first page
		page1, err := repo.GetByMerchant(merchantID, 2, 0)
		require.NoError(t, err)
		assert.Len(t, page1, 2)

		// Get second page
		page2, err := repo.GetByMerchant(merchantID, 2, 2)
		require.NoError(t, err)
		assert.Len(t, page2, 2)

		// Ensure no overlap
		assert.NotEqual(t, page1[0].ID, page2[0].ID)
	})

	t.Run("fail when merchant ID is empty", func(t *testing.T) {
		entries, err := repo.GetByMerchant("", 10, 0)
		assert.Error(t, err)
		assert.Nil(t, entries)
	})
}

func TestLedgerRepository_GetByReference(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupLedgerEntries(t, db)

	repo := NewLedgerRepository(db)

	referenceID := uuid.New().String()
	transactionGroup := uuid.New().String()

	// Create entries with same reference
	entries := []*model.LedgerEntry{
		{
			DebitAccount:     "crypto_pool",
			CreditAccount:    "merchant_balance:merchant-789",
			Amount:           decimal.NewFromFloat(1000.00),
			Currency:         "VND",
			ReferenceType:    model.ReferenceTypePayment,
			ReferenceID:      referenceID,
			Description:      "Debit entry",
			TransactionGroup: transactionGroup,
			EntryType:        model.EntryTypeDebit,
		},
		{
			DebitAccount:     "crypto_pool",
			CreditAccount:    "merchant_balance:merchant-789",
			Amount:           decimal.NewFromFloat(1000.00),
			Currency:         "VND",
			ReferenceType:    model.ReferenceTypePayment,
			ReferenceID:      referenceID,
			Description:      "Credit entry",
			TransactionGroup: transactionGroup,
			EntryType:        model.EntryTypeCredit,
		},
	}

	err := repo.CreateEntries(entries)
	require.NoError(t, err)

	t.Run("successfully get entries by reference", func(t *testing.T) {
		retrieved, err := repo.GetByReference(model.ReferenceTypePayment, referenceID)
		require.NoError(t, err)
		assert.Len(t, retrieved, 2)
		assert.Equal(t, referenceID, retrieved[0].ReferenceID)
		assert.Equal(t, referenceID, retrieved[1].ReferenceID)
	})

	t.Run("fail when reference type is empty", func(t *testing.T) {
		retrieved, err := repo.GetByReference("", referenceID)
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("fail when reference ID is empty", func(t *testing.T) {
		retrieved, err := repo.GetByReference(model.ReferenceTypePayment, "")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})
}

func TestLedgerRepository_GetByTransactionGroup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupLedgerEntries(t, db)

	repo := NewLedgerRepository(db)

	transactionGroup := uuid.New().String()
	referenceID := uuid.New().String()
	amount := decimal.NewFromFloat(5000.00)

	entries := []*model.LedgerEntry{
		{
			DebitAccount:     "crypto_pool",
			CreditAccount:    "merchant_balance:merchant-999",
			Amount:           amount,
			Currency:         "VND",
			ReferenceType:    model.ReferenceTypePayment,
			ReferenceID:      referenceID,
			Description:      "Debit entry",
			TransactionGroup: transactionGroup,
			EntryType:        model.EntryTypeDebit,
		},
		{
			DebitAccount:     "crypto_pool",
			CreditAccount:    "merchant_balance:merchant-999",
			Amount:           amount,
			Currency:         "VND",
			ReferenceType:    model.ReferenceTypePayment,
			ReferenceID:      referenceID,
			Description:      "Credit entry",
			TransactionGroup: transactionGroup,
			EntryType:        model.EntryTypeCredit,
		},
	}

	err := repo.CreateEntries(entries)
	require.NoError(t, err)

	t.Run("successfully get entries by transaction group", func(t *testing.T) {
		retrieved, err := repo.GetByTransactionGroup(transactionGroup)
		require.NoError(t, err)
		assert.Len(t, retrieved, 2)
		assert.Equal(t, transactionGroup, retrieved[0].TransactionGroup)
		assert.Equal(t, transactionGroup, retrieved[1].TransactionGroup)
	})

	t.Run("fail when transaction group is empty", func(t *testing.T) {
		retrieved, err := repo.GetByTransactionGroup("")
		assert.ErrorIs(t, err, ErrEmptyTransactionGroup)
		assert.Nil(t, retrieved)
	})
}

func TestLedgerRepository_GetAccountBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupLedgerEntries(t, db)

	repo := NewLedgerRepository(db)

	accountName := "merchant_balance:merchant-111"

	// Create entries affecting the account
	// Debit: +1000, +500 = 1500
	// Credit: -300 = -300
	// Net: 1200
	entries := []*model.LedgerEntry{
		{
			DebitAccount:     accountName,
			CreditAccount:    "revenue:fees",
			Amount:           decimal.NewFromFloat(1000.00),
			Currency:         "VND",
			ReferenceType:    model.ReferenceTypePayment,
			ReferenceID:      uuid.New().String(),
			Description:      "Debit 1",
			TransactionGroup: uuid.New().String(),
			EntryType:        model.EntryTypeDebit,
		},
		{
			DebitAccount:     accountName,
			CreditAccount:    "revenue:fees",
			Amount:           decimal.NewFromFloat(500.00),
			Currency:         "VND",
			ReferenceType:    model.ReferenceTypePayment,
			ReferenceID:      uuid.New().String(),
			Description:      "Debit 2",
			TransactionGroup: uuid.New().String(),
			EntryType:        model.EntryTypeDebit,
		},
		{
			DebitAccount:     "payout_pool",
			CreditAccount:    accountName,
			Amount:           decimal.NewFromFloat(300.00),
			Currency:         "VND",
			ReferenceType:    model.ReferenceTypePayout,
			ReferenceID:      uuid.New().String(),
			Description:      "Credit 1",
			TransactionGroup: uuid.New().String(),
			EntryType:        model.EntryTypeCredit,
		},
	}

	for _, entry := range entries {
		err := repo.CreateEntry(entry)
		require.NoError(t, err)
	}

	t.Run("successfully calculate account balance", func(t *testing.T) {
		balance, err := repo.GetAccountBalance(accountName)
		require.NoError(t, err)
		expected := decimal.NewFromFloat(1200.00) // 1000 + 500 - 300
		assert.True(t, balance.Equal(expected), "expected %s, got %s", expected, balance)
	})

	t.Run("return zero for non-existent account", func(t *testing.T) {
		balance, err := repo.GetAccountBalance("non_existent_account")
		require.NoError(t, err)
		assert.True(t, balance.Equal(decimal.Zero))
	})

	t.Run("fail when account name is empty", func(t *testing.T) {
		balance, err := repo.GetAccountBalance("")
		assert.Error(t, err)
		assert.True(t, balance.Equal(decimal.Zero))
	})
}

func TestLedgerRepository_ValidateDoubleEntry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupLedgerEntries(t, db)

	repo := NewLedgerRepository(db)

	t.Run("validation passes for balanced entries", func(t *testing.T) {
		transactionGroup := uuid.New().String()
		referenceID := uuid.New().String()
		amount := decimal.NewFromFloat(1000.00)

		entries := []*model.LedgerEntry{
			{
				DebitAccount:     "crypto_pool",
				CreditAccount:    "merchant_balance:merchant-222",
				Amount:           amount,
				Currency:         "VND",
				ReferenceType:    model.ReferenceTypePayment,
				ReferenceID:      referenceID,
				Description:      "Debit entry",
				TransactionGroup: transactionGroup,
				EntryType:        model.EntryTypeDebit,
			},
			{
				DebitAccount:     "crypto_pool",
				CreditAccount:    "merchant_balance:merchant-222",
				Amount:           amount,
				Currency:         "VND",
				ReferenceType:    model.ReferenceTypePayment,
				ReferenceID:      referenceID,
				Description:      "Credit entry",
				TransactionGroup: transactionGroup,
				EntryType:        model.EntryTypeCredit,
			},
		}

		err := repo.CreateEntries(entries)
		require.NoError(t, err)

		// Validate
		err = repo.ValidateDoubleEntry()
		assert.NoError(t, err)
	})
}

func TestLedgerRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupLedgerEntries(t, db)

	repo := NewLedgerRepository(db)

	// Initially should be 0
	count, err := repo.Count()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Create some entries
	for i := 0; i < 3; i++ {
		entry := &model.LedgerEntry{
			DebitAccount:     "account_a",
			CreditAccount:    "account_b",
			Amount:           decimal.NewFromInt(100),
			Currency:         "VND",
			ReferenceType:    model.ReferenceTypePayment,
			ReferenceID:      uuid.New().String(),
			Description:      "Test entry",
			TransactionGroup: uuid.New().String(),
			EntryType:        model.EntryTypeDebit,
		}
		err := repo.CreateEntry(entry)
		require.NoError(t, err)
	}

	// Should now be 3
	count, err = repo.Count()
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestLedgerRepository_CountByMerchant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupLedgerEntries(t, db)

	repo := NewLedgerRepository(db)

	merchantID := "merchant-333"

	// Create entries for the merchant
	for i := 0; i < 4; i++ {
		entry := &model.LedgerEntry{
			DebitAccount:     "merchant_balance:" + merchantID,
			CreditAccount:    "revenue:fees",
			Amount:           decimal.NewFromInt(100),
			Currency:         "VND",
			ReferenceType:    model.ReferenceTypePayment,
			ReferenceID:      uuid.New().String(),
			MerchantID:       sql.NullString{String: merchantID, Valid: true},
			Description:      "Test entry",
			TransactionGroup: uuid.New().String(),
			EntryType:        model.EntryTypeDebit,
		}
		err := repo.CreateEntry(entry)
		require.NoError(t, err)
	}

	count, err := repo.CountByMerchant(merchantID)
	require.NoError(t, err)
	assert.Equal(t, int64(4), count)
}
