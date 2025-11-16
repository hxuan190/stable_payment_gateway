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

// cleanupBalances removes all balance records from the test database
func cleanupBalances(t *testing.T, db *sql.DB) {
	t.Helper()
	// Delete balances first (before merchants due to FK constraint)
	_, err := db.Exec("DELETE FROM merchant_balances")
	require.NoError(t, err, "Failed to cleanup merchant_balances table")
}

// createTestBalance creates a test merchant balance with default values
func createTestBalance(merchantID string) *model.MerchantBalance {
	return &model.MerchantBalance{
		ID:                 uuid.New().String(),
		MerchantID:         merchantID,
		PendingVND:         decimal.NewFromInt(0),
		AvailableVND:       decimal.NewFromInt(0),
		TotalVND:           decimal.NewFromInt(0),
		ReservedVND:        decimal.NewFromInt(0),
		TotalReceivedVND:   decimal.NewFromInt(0),
		TotalPaidOutVND:    decimal.NewFromInt(0),
		TotalFeesVND:       decimal.NewFromInt(0),
		TotalPaymentsCount: 0,
		TotalPayoutsCount:  0,
		Version:            1,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}

func TestBalanceRepository_GetByMerchantID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	// Setup: Create merchant first
	merchantRepo := NewMerchantRepository(db)
	balanceRepo := NewBalanceRepository(db)

	t.Run("Success", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant (which auto-creates balance via trigger)
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Get the balance
		balance, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)
		assert.NotNil(t, balance)
		assert.Equal(t, merchant.ID, balance.MerchantID)
		assert.True(t, balance.AvailableVND.Equal(decimal.Zero))
		assert.True(t, balance.PendingVND.Equal(decimal.Zero))
		assert.True(t, balance.TotalVND.Equal(decimal.Zero))
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		balance, err := balanceRepo.GetByMerchantID(nonExistentID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrBalanceNotFound)
		assert.Nil(t, balance)
	})

	t.Run("EmptyMerchantID", func(t *testing.T) {
		balance, err := balanceRepo.GetByMerchantID("")
		assert.Error(t, err)
		assert.Nil(t, balance)
	})
}

func TestBalanceRepository_GetByMerchantIDForUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	merchantRepo := NewMerchantRepository(db)
	balanceRepo := NewBalanceRepository(db)

	t.Run("Success", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Begin transaction
		tx, err := db.Begin()
		require.NoError(t, err)
		defer tx.Rollback()

		// Get balance with lock
		balance, err := balanceRepo.GetByMerchantIDForUpdate(tx, merchant.ID)
		require.NoError(t, err)
		assert.NotNil(t, balance)
		assert.Equal(t, merchant.ID, balance.MerchantID)

		// Commit transaction
		err = tx.Commit()
		require.NoError(t, err)
	})

	t.Run("NilTransaction", func(t *testing.T) {
		balance, err := balanceRepo.GetByMerchantIDForUpdate(nil, uuid.New().String())
		assert.Error(t, err)
		assert.Nil(t, balance)
		assert.Contains(t, err.Error(), "transaction cannot be nil")
	})
}

func TestBalanceRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	merchantRepo := NewMerchantRepository(db)
	balanceRepo := NewBalanceRepository(db)

	t.Run("Success", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant without auto-trigger (manually)
		merchant := createTestMerchant()
		merchant.ID = uuid.New().String()

		// Disable trigger temporarily for this test
		_, err := db.Exec("ALTER TABLE merchants DISABLE TRIGGER create_merchant_balance_on_merchant_insert")
		require.NoError(t, err)
		defer db.Exec("ALTER TABLE merchants ENABLE TRIGGER create_merchant_balance_on_merchant_insert")

		err = merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Create balance manually
		balance := createTestBalance(merchant.ID)
		err = balanceRepo.Create(balance)
		require.NoError(t, err)
		assert.NotEmpty(t, balance.ID)
		assert.False(t, balance.CreatedAt.IsZero())
		assert.False(t, balance.UpdatedAt.IsZero())
	})

	t.Run("NilBalance", func(t *testing.T) {
		err := balanceRepo.Create(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "balance cannot be nil")
	})

	t.Run("EmptyMerchantID", func(t *testing.T) {
		balance := createTestBalance("")
		err := balanceRepo.Create(balance)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant ID cannot be empty")
	})
}

func TestBalanceRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	merchantRepo := NewMerchantRepository(db)
	balanceRepo := NewBalanceRepository(db)

	t.Run("Success", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Get balance
		balance, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)

		// Update balance
		balance.AvailableVND = decimal.NewFromInt(1000000)
		balance.TotalVND = decimal.NewFromInt(1000000)
		oldVersion := balance.Version

		err = balanceRepo.Update(balance)
		require.NoError(t, err)
		assert.Equal(t, oldVersion+1, balance.Version)

		// Verify update
		updated, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)
		assert.True(t, updated.AvailableVND.Equal(decimal.NewFromInt(1000000)))
		assert.Equal(t, oldVersion+1, updated.Version)
	})

	t.Run("VersionMismatch", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Get balance
		balance, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)

		// Update balance first time
		balance.AvailableVND = decimal.NewFromInt(1000000)
		balance.TotalVND = decimal.NewFromInt(1000000)
		err = balanceRepo.Update(balance)
		require.NoError(t, err)

		// Try to update with old version (simulate concurrent update)
		oldBalance := *balance
		oldBalance.Version = 1 // Use old version
		oldBalance.AvailableVND = decimal.NewFromInt(2000000)
		oldBalance.TotalVND = decimal.NewFromInt(2000000)

		err = balanceRepo.Update(&oldBalance)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrBalanceVersionMismatch)
	})

	t.Run("InvalidBalanceState", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Get balance
		balance, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)

		// Set invalid state (total != pending + available)
		balance.PendingVND = decimal.NewFromInt(100000)
		balance.AvailableVND = decimal.NewFromInt(200000)
		balance.TotalVND = decimal.NewFromInt(500000) // Invalid: should be 300000

		err = balanceRepo.Update(balance)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidBalanceUpdate)
	})
}

func TestBalanceRepository_IncrementAvailable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	merchantRepo := NewMerchantRepository(db)
	balanceRepo := NewBalanceRepository(db)

	t.Run("Success", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Increment balance
		amount := decimal.NewFromInt(500000)
		err = balanceRepo.IncrementAvailable(merchant.ID, amount)
		require.NoError(t, err)

		// Verify
		balance, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)
		assert.True(t, balance.AvailableVND.Equal(amount))
		assert.True(t, balance.TotalVND.Equal(amount))
	})

	t.Run("NegativeAmount", func(t *testing.T) {
		err := balanceRepo.IncrementAvailable(uuid.New().String(), decimal.NewFromInt(-100))
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrNegativeAmount)
	})

	t.Run("MerchantNotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		err := balanceRepo.IncrementAvailable(nonExistentID, decimal.NewFromInt(1000))
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrBalanceNotFound)
	})
}

func TestBalanceRepository_DecrementAvailable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	merchantRepo := NewMerchantRepository(db)
	balanceRepo := NewBalanceRepository(db)

	t.Run("Success", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Add some balance first
		err = balanceRepo.IncrementAvailable(merchant.ID, decimal.NewFromInt(1000000))
		require.NoError(t, err)

		// Decrement balance
		amount := decimal.NewFromInt(300000)
		err = balanceRepo.DecrementAvailable(merchant.ID, amount)
		require.NoError(t, err)

		// Verify
		balance, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)
		expectedBalance := decimal.NewFromInt(700000)
		assert.True(t, balance.AvailableVND.Equal(expectedBalance))
		assert.True(t, balance.TotalVND.Equal(expectedBalance))
	})

	t.Run("InsufficientBalance", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Try to decrement more than available
		err = balanceRepo.DecrementAvailable(merchant.ID, decimal.NewFromInt(1000000))
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInsufficientBalance)
	})

	t.Run("NegativeAmount", func(t *testing.T) {
		err := balanceRepo.DecrementAvailable(uuid.New().String(), decimal.NewFromInt(-100))
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrNegativeAmount)
	})
}

func TestBalanceRepository_IncrementPending(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	merchantRepo := NewMerchantRepository(db)
	balanceRepo := NewBalanceRepository(db)

	t.Run("Success", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Increment pending balance
		amount := decimal.NewFromInt(250000)
		err = balanceRepo.IncrementPending(merchant.ID, amount)
		require.NoError(t, err)

		// Verify
		balance, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)
		assert.True(t, balance.PendingVND.Equal(amount))
		assert.True(t, balance.TotalVND.Equal(amount))
		assert.True(t, balance.AvailableVND.Equal(decimal.Zero))
	})
}

func TestBalanceRepository_DecrementPending(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	merchantRepo := NewMerchantRepository(db)
	balanceRepo := NewBalanceRepository(db)

	t.Run("Success", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Add pending balance
		err = balanceRepo.IncrementPending(merchant.ID, decimal.NewFromInt(500000))
		require.NoError(t, err)

		// Decrement pending balance
		amount := decimal.NewFromInt(200000)
		err = balanceRepo.DecrementPending(merchant.ID, amount)
		require.NoError(t, err)

		// Verify
		balance, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)
		expectedPending := decimal.NewFromInt(300000)
		assert.True(t, balance.PendingVND.Equal(expectedPending))
		assert.True(t, balance.TotalVND.Equal(expectedPending))
	})

	t.Run("InsufficientPending", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Try to decrement more than pending
		err = balanceRepo.DecrementPending(merchant.ID, decimal.NewFromInt(1000))
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInsufficientBalance)
	})
}

func TestBalanceRepository_IncrementReserved(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	merchantRepo := NewMerchantRepository(db)
	balanceRepo := NewBalanceRepository(db)

	t.Run("Success", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant and add available balance
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		err = balanceRepo.IncrementAvailable(merchant.ID, decimal.NewFromInt(1000000))
		require.NoError(t, err)

		// Reserve some balance
		amount := decimal.NewFromInt(300000)
		err = balanceRepo.IncrementReserved(merchant.ID, amount)
		require.NoError(t, err)

		// Verify
		balance, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)
		assert.True(t, balance.ReservedVND.Equal(amount))
		assert.True(t, balance.AvailableVND.Equal(decimal.NewFromInt(1000000)))
	})

	t.Run("InsufficientAvailable", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Try to reserve more than available
		err = balanceRepo.IncrementReserved(merchant.ID, decimal.NewFromInt(1000000))
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInsufficientBalance)
	})
}

func TestBalanceRepository_DecrementReserved(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	merchantRepo := NewMerchantRepository(db)
	balanceRepo := NewBalanceRepository(db)

	t.Run("Success", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant and set up balances
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		err = balanceRepo.IncrementAvailable(merchant.ID, decimal.NewFromInt(1000000))
		require.NoError(t, err)

		err = balanceRepo.IncrementReserved(merchant.ID, decimal.NewFromInt(500000))
		require.NoError(t, err)

		// Release reserved balance
		amount := decimal.NewFromInt(200000)
		err = balanceRepo.DecrementReserved(merchant.ID, amount)
		require.NoError(t, err)

		// Verify
		balance, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)
		expectedReserved := decimal.NewFromInt(300000)
		assert.True(t, balance.ReservedVND.Equal(expectedReserved))
	})
}

func TestBalanceRepository_RecordPayment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	merchantRepo := NewMerchantRepository(db)
	balanceRepo := NewBalanceRepository(db)

	t.Run("Success", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Begin transaction
		tx, err := db.Begin()
		require.NoError(t, err)
		defer tx.Rollback()

		// Record payment
		amount := decimal.NewFromInt(2300000)
		err = balanceRepo.RecordPayment(tx, merchant.ID, amount)
		require.NoError(t, err)

		// Commit
		err = tx.Commit()
		require.NoError(t, err)

		// Verify
		balance, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)
		assert.True(t, balance.AvailableVND.Equal(amount))
		assert.True(t, balance.TotalReceivedVND.Equal(amount))
		assert.Equal(t, 1, balance.TotalPaymentsCount)
		assert.True(t, balance.LastPaymentAt.Valid)
	})

	t.Run("NilTransaction", func(t *testing.T) {
		err := balanceRepo.RecordPayment(nil, uuid.New().String(), decimal.NewFromInt(1000))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction cannot be nil")
	})

	t.Run("ZeroAmount", func(t *testing.T) {
		tx, _ := db.Begin()
		defer tx.Rollback()

		err := balanceRepo.RecordPayment(tx, uuid.New().String(), decimal.Zero)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "payment amount must be positive")
	})
}

func TestBalanceRepository_RecordPayout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	merchantRepo := NewMerchantRepository(db)
	balanceRepo := NewBalanceRepository(db)

	t.Run("Success", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant and add balance
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Add available and reserved balance
		err = balanceRepo.IncrementAvailable(merchant.ID, decimal.NewFromInt(1000000))
		require.NoError(t, err)

		err = balanceRepo.IncrementReserved(merchant.ID, decimal.NewFromInt(550000))
		require.NoError(t, err)

		// Begin transaction
		tx, err := db.Begin()
		require.NoError(t, err)
		defer tx.Rollback()

		// Record payout
		payoutAmount := decimal.NewFromInt(500000)
		fee := decimal.NewFromInt(50000)
		err = balanceRepo.RecordPayout(tx, merchant.ID, payoutAmount, fee)
		require.NoError(t, err)

		// Commit
		err = tx.Commit()
		require.NoError(t, err)

		// Verify
		balance, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)

		expectedAvailable := decimal.NewFromInt(450000) // 1000000 - 500000 - 50000
		assert.True(t, balance.AvailableVND.Equal(expectedAvailable))
		assert.True(t, balance.TotalPaidOutVND.Equal(payoutAmount))
		assert.True(t, balance.TotalFeesVND.Equal(fee))
		assert.Equal(t, 1, balance.TotalPayoutsCount)
		assert.True(t, balance.LastPayoutAt.Valid)
	})

	t.Run("InsufficientBalance", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant with insufficient balance
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Begin transaction
		tx, err := db.Begin()
		require.NoError(t, err)
		defer tx.Rollback()

		// Try to record payout with insufficient balance
		err = balanceRepo.RecordPayout(tx, merchant.ID, decimal.NewFromInt(1000000), decimal.NewFromInt(10000))
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInsufficientBalance)
	})
}

func TestBalanceRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	merchantRepo := NewMerchantRepository(db)
	balanceRepo := NewBalanceRepository(db)

	t.Run("Success", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create multiple merchants
		merchant1 := createTestMerchant()
		merchant1.Email = "merchant1@example.com"
		err := merchantRepo.Create(merchant1)
		require.NoError(t, err)

		merchant2 := createTestMerchant()
		merchant2.Email = "merchant2@example.com"
		err = merchantRepo.Create(merchant2)
		require.NoError(t, err)

		// List balances
		balances, err := balanceRepo.List(10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(balances), 2)
	})

	t.Run("Pagination", func(t *testing.T) {
		// List with limit
		balances, err := balanceRepo.List(1, 0)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(balances), 1)

		// List with offset
		balances, err = balanceRepo.List(1, 1)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(balances), 1)
	})
}

func TestBalanceRepository_ConcurrentUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	merchantRepo := NewMerchantRepository(db)
	balanceRepo := NewBalanceRepository(db)

	t.Run("OptimisticLockingPreventsConflicts", func(t *testing.T) {
		cleanupMerchants(t, db)
		cleanupBalances(t, db)

		// Create merchant
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		// Get balance twice to simulate concurrent reads
		balance1, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)

		balance2, err := balanceRepo.GetByMerchantID(merchant.ID)
		require.NoError(t, err)

		// Update first balance
		balance1.AvailableVND = decimal.NewFromInt(1000000)
		balance1.TotalVND = decimal.NewFromInt(1000000)
		err = balanceRepo.Update(balance1)
		require.NoError(t, err)

		// Try to update second balance (should fail due to version mismatch)
		balance2.AvailableVND = decimal.NewFromInt(2000000)
		balance2.TotalVND = decimal.NewFromInt(2000000)
		err = balanceRepo.Update(balance2)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrBalanceVersionMismatch)
	})
}
