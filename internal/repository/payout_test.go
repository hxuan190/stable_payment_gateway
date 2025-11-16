package repository

import (
	"database/sql"
	"testing"

	"github.com/hxuan190/stable_payment_gateway/internal/model"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cleanupPayouts removes all payouts from the test database
func cleanupPayouts(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("DELETE FROM payouts")
	require.NoError(t, err, "Failed to cleanup payouts table")
}

// createTestPayout creates a test payout with default values
func createTestPayout(merchantID string) *model.Payout {
	return &model.Payout{
		ID:                uuid.New().String(),
		MerchantID:        merchantID,
		AmountVND:         decimal.NewFromInt(10000000), // 10M VND
		FeeVND:            decimal.NewFromInt(50000),    // 50K VND
		NetAmountVND:      decimal.NewFromInt(9950000),  // 9.95M VND
		BankAccountName:   "Test Merchant",
		BankAccountNumber: "1234567890",
		BankName:          "Vietcombank",
		BankBranch:        sql.NullString{String: "Da Nang Branch", Valid: true},
		Status:            model.PayoutStatusRequested,
		RequestedBy:       merchantID,
		RetryCount:        0,
		Metadata:          model.JSONBMap{},
	}
}

func TestPayoutRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayouts(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPayoutRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Create payout", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)

		err := repo.Create(payout)
		require.NoError(t, err)

		// Verify the payout was created
		assert.NotEmpty(t, payout.ID)
		assert.False(t, payout.CreatedAt.IsZero())
		assert.False(t, payout.UpdatedAt.IsZero())
	})

	t.Run("Error - Nil payout", func(t *testing.T) {
		err := repo.Create(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})
}

func TestPayoutRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayouts(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPayoutRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Get existing payout", func(t *testing.T) {
		// Create a payout first
		payout := createTestPayout(merchant.ID)
		err := repo.Create(payout)
		require.NoError(t, err)

		// Retrieve the payout
		retrieved, err := repo.GetByID(payout.ID)
		require.NoError(t, err)

		assert.Equal(t, payout.ID, retrieved.ID)
		assert.Equal(t, payout.MerchantID, retrieved.MerchantID)
		assert.True(t, payout.AmountVND.Equal(retrieved.AmountVND))
		assert.True(t, payout.FeeVND.Equal(retrieved.FeeVND))
		assert.True(t, payout.NetAmountVND.Equal(retrieved.NetAmountVND))
		assert.Equal(t, payout.BankAccountName, retrieved.BankAccountName)
		assert.Equal(t, payout.BankAccountNumber, retrieved.BankAccountNumber)
		assert.Equal(t, payout.BankName, retrieved.BankName)
		assert.Equal(t, payout.Status, retrieved.Status)
	})

	t.Run("Error - Payout not found", func(t *testing.T) {
		_, err := repo.GetByID(uuid.New().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPayoutNotFound)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		_, err := repo.GetByID("")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPayoutID)
	})
}

func TestPayoutRepository_ListByMerchant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayouts(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPayoutRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create test merchants
	merchant1 := createTestMerchant()
	merchant1.Email = "merchant1@example.com"
	err := merchantRepo.Create(merchant1)
	require.NoError(t, err)

	merchant2 := createTestMerchant()
	merchant2.ID = uuid.New().String()
	merchant2.Email = "merchant2@example.com"
	err = merchantRepo.Create(merchant2)
	require.NoError(t, err)

	t.Run("Success - List payouts by merchant", func(t *testing.T) {
		// Create payouts for merchant1
		payout1 := createTestPayout(merchant1.ID)
		err := repo.Create(payout1)
		require.NoError(t, err)

		payout2 := createTestPayout(merchant1.ID)
		payout2.ID = uuid.New().String()
		err = repo.Create(payout2)
		require.NoError(t, err)

		// Create payout for merchant2
		payout3 := createTestPayout(merchant2.ID)
		err = repo.Create(payout3)
		require.NoError(t, err)

		// Retrieve payouts for merchant1
		payouts, err := repo.ListByMerchant(merchant1.ID, 10, 0)
		require.NoError(t, err)

		assert.Len(t, payouts, 2)
		assert.Equal(t, merchant1.ID, payouts[0].MerchantID)
		assert.Equal(t, merchant1.ID, payouts[1].MerchantID)
	})

	t.Run("Success - List with pagination", func(t *testing.T) {
		// Retrieve first page
		payouts, err := repo.ListByMerchant(merchant1.ID, 1, 0)
		require.NoError(t, err)
		assert.Len(t, payouts, 1)

		// Retrieve second page
		payouts, err = repo.ListByMerchant(merchant1.ID, 1, 1)
		require.NoError(t, err)
		assert.Len(t, payouts, 1)
	})

	t.Run("Error - Empty merchant ID", func(t *testing.T) {
		_, err := repo.ListByMerchant("", 10, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant ID cannot be empty")
	})
}

func TestPayoutRepository_ListPending(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayouts(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPayoutRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - List pending payouts", func(t *testing.T) {
		// Create requested payout
		payout1 := createTestPayout(merchant.ID)
		payout1.Status = model.PayoutStatusRequested
		err := repo.Create(payout1)
		require.NoError(t, err)

		// Create approved payout
		payout2 := createTestPayout(merchant.ID)
		payout2.ID = uuid.New().String()
		payout2.Status = model.PayoutStatusApproved
		err = repo.Create(payout2)
		require.NoError(t, err)

		// Create completed payout
		payout3 := createTestPayout(merchant.ID)
		payout3.ID = uuid.New().String()
		payout3.Status = model.PayoutStatusCompleted
		err = repo.Create(payout3)
		require.NoError(t, err)

		// Retrieve pending payouts (only requested status)
		payouts, err := repo.ListPending()
		require.NoError(t, err)

		assert.Len(t, payouts, 1)
		assert.Equal(t, model.PayoutStatusRequested, payouts[0].Status)
	})
}

func TestPayoutRepository_ListByStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayouts(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPayoutRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - List payouts by status", func(t *testing.T) {
		// Create approved payout
		payout1 := createTestPayout(merchant.ID)
		payout1.Status = model.PayoutStatusApproved
		err := repo.Create(payout1)
		require.NoError(t, err)

		// Create another approved payout
		payout2 := createTestPayout(merchant.ID)
		payout2.ID = uuid.New().String()
		payout2.Status = model.PayoutStatusApproved
		err = repo.Create(payout2)
		require.NoError(t, err)

		// Create completed payout
		payout3 := createTestPayout(merchant.ID)
		payout3.ID = uuid.New().String()
		payout3.Status = model.PayoutStatusCompleted
		err = repo.Create(payout3)
		require.NoError(t, err)

		// Retrieve approved payouts
		payouts, err := repo.ListByStatus(model.PayoutStatusApproved, 10, 0)
		require.NoError(t, err)

		assert.Len(t, payouts, 2)
		for _, p := range payouts {
			assert.Equal(t, model.PayoutStatusApproved, p.Status)
		}
	})

	t.Run("Error - Empty status", func(t *testing.T) {
		_, err := repo.ListByStatus("", 10, 0)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPayoutStatus)
	})
}

func TestPayoutRepository_UpdateStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayouts(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPayoutRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Update status", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)
		payout.Status = model.PayoutStatusRequested
		err := repo.Create(payout)
		require.NoError(t, err)

		// Update status to approved
		err = repo.UpdateStatus(payout.ID, model.PayoutStatusApproved)
		require.NoError(t, err)

		// Verify the status was updated
		retrieved, err := repo.GetByID(payout.ID)
		require.NoError(t, err)
		assert.Equal(t, model.PayoutStatusApproved, retrieved.Status)
	})

	t.Run("Error - Invalid payout ID", func(t *testing.T) {
		err := repo.UpdateStatus("", model.PayoutStatusApproved)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPayoutID)
	})

	t.Run("Error - Invalid status", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)
		err := repo.Create(payout)
		require.NoError(t, err)

		err = repo.UpdateStatus(payout.ID, "invalid_status")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPayoutStatus)
	})

	t.Run("Error - Payout not found", func(t *testing.T) {
		err := repo.UpdateStatus(uuid.New().String(), model.PayoutStatusApproved)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPayoutNotFound)
	})
}

func TestPayoutRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayouts(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPayoutRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Update payout", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)
		err := repo.Create(payout)
		require.NoError(t, err)

		// Update payout
		payout.AmountVND = decimal.NewFromInt(20000000)
		payout.Status = model.PayoutStatusApproved
		payout.BankAccountName = "Updated Name"

		err = repo.Update(payout)
		require.NoError(t, err)

		// Verify the update
		retrieved, err := repo.GetByID(payout.ID)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(20000000).Equal(retrieved.AmountVND))
		assert.Equal(t, model.PayoutStatusApproved, retrieved.Status)
		assert.Equal(t, "Updated Name", retrieved.BankAccountName)
	})

	t.Run("Error - Nil payout", func(t *testing.T) {
		err := repo.Update(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)
		payout.ID = ""
		err := repo.Update(payout)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPayoutID)
	})

	t.Run("Error - Payout not found", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)
		payout.ID = uuid.New().String()
		err := repo.Update(payout)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPayoutNotFound)
	})
}

func TestPayoutRepository_MarkCompleted(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayouts(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPayoutRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Mark payout as completed", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)
		payout.Status = model.PayoutStatusApproved
		err := repo.Create(payout)
		require.NoError(t, err)

		// Mark as completed
		refNumber := "BANK-REF-123456"
		err = repo.MarkCompleted(payout.ID, refNumber)
		require.NoError(t, err)

		// Verify the payout is completed
		retrieved, err := repo.GetByID(payout.ID)
		require.NoError(t, err)
		assert.Equal(t, model.PayoutStatusCompleted, retrieved.Status)
		assert.Equal(t, refNumber, retrieved.BankReferenceNumber.String)
		assert.True(t, retrieved.CompletionDate.Valid)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		err := repo.MarkCompleted("", "BANK-REF-123")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPayoutID)
	})

	t.Run("Error - Empty reference number", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)
		err := repo.Create(payout)
		require.NoError(t, err)

		err = repo.MarkCompleted(payout.ID, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bank reference number cannot be empty")
	})

	t.Run("Error - Payout not found", func(t *testing.T) {
		err := repo.MarkCompleted(uuid.New().String(), "BANK-REF-123")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPayoutNotFound)
	})
}

func TestPayoutRepository_Approve(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayouts(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPayoutRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Approve payout", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)
		payout.Status = model.PayoutStatusRequested
		err := repo.Create(payout)
		require.NoError(t, err)

		// Approve the payout
		approverID := uuid.New().String()
		err = repo.Approve(payout.ID, approverID)
		require.NoError(t, err)

		// Verify the payout is approved
		retrieved, err := repo.GetByID(payout.ID)
		require.NoError(t, err)
		assert.Equal(t, model.PayoutStatusApproved, retrieved.Status)
		assert.Equal(t, approverID, retrieved.ApprovedBy.String)
		assert.True(t, retrieved.ApprovedAt.Valid)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		err := repo.Approve("", uuid.New().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPayoutID)
	})

	t.Run("Error - Empty approver ID", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)
		err := repo.Create(payout)
		require.NoError(t, err)

		err = repo.Approve(payout.ID, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "approver ID cannot be empty")
	})

	t.Run("Error - Payout not in requested status", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)
		payout.Status = model.PayoutStatusCompleted
		err := repo.Create(payout)
		require.NoError(t, err)

		err = repo.Approve(payout.ID, uuid.New().String())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in requested status")
	})
}

func TestPayoutRepository_Reject(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayouts(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPayoutRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Reject payout", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)
		payout.Status = model.PayoutStatusRequested
		err := repo.Create(payout)
		require.NoError(t, err)

		// Reject the payout
		reason := "Insufficient balance"
		err = repo.Reject(payout.ID, reason)
		require.NoError(t, err)

		// Verify the payout is rejected
		retrieved, err := repo.GetByID(payout.ID)
		require.NoError(t, err)
		assert.Equal(t, model.PayoutStatusRejected, retrieved.Status)
		assert.Equal(t, reason, retrieved.RejectionReason.String)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		err := repo.Reject("", "Some reason")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPayoutID)
	})

	t.Run("Error - Empty reason", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)
		err := repo.Create(payout)
		require.NoError(t, err)

		err = repo.Reject(payout.ID, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rejection reason cannot be empty")
	})

	t.Run("Error - Payout not in requested status", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)
		payout.Status = model.PayoutStatusCompleted
		err := repo.Create(payout)
		require.NoError(t, err)

		err = repo.Reject(payout.ID, "Some reason")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in requested status")
	})
}

func TestPayoutRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayouts(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPayoutRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Count payouts", func(t *testing.T) {
		// Create payouts
		payout1 := createTestPayout(merchant.ID)
		err := repo.Create(payout1)
		require.NoError(t, err)

		payout2 := createTestPayout(merchant.ID)
		payout2.ID = uuid.New().String()
		err = repo.Create(payout2)
		require.NoError(t, err)

		// Count payouts
		count, err := repo.Count()
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})
}

func TestPayoutRepository_CountByMerchant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayouts(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPayoutRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create test merchants
	merchant1 := createTestMerchant()
	merchant1.Email = "merchant1@example.com"
	err := merchantRepo.Create(merchant1)
	require.NoError(t, err)

	merchant2 := createTestMerchant()
	merchant2.ID = uuid.New().String()
	merchant2.Email = "merchant2@example.com"
	err = merchantRepo.Create(merchant2)
	require.NoError(t, err)

	t.Run("Success - Count payouts by merchant", func(t *testing.T) {
		// Create payouts for merchant1
		payout1 := createTestPayout(merchant1.ID)
		err := repo.Create(payout1)
		require.NoError(t, err)

		payout2 := createTestPayout(merchant1.ID)
		payout2.ID = uuid.New().String()
		err = repo.Create(payout2)
		require.NoError(t, err)

		// Create payout for merchant2
		payout3 := createTestPayout(merchant2.ID)
		err = repo.Create(payout3)
		require.NoError(t, err)

		// Count payouts for merchant1
		count, err := repo.CountByMerchant(merchant1.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)

		// Count payouts for merchant2
		count, err = repo.CountByMerchant(merchant2.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("Error - Empty merchant ID", func(t *testing.T) {
		_, err := repo.CountByMerchant("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant ID cannot be empty")
	})
}

func TestPayoutRepository_CountByStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayouts(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPayoutRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Count payouts by status", func(t *testing.T) {
		// Create approved payouts
		payout1 := createTestPayout(merchant.ID)
		payout1.Status = model.PayoutStatusApproved
		err := repo.Create(payout1)
		require.NoError(t, err)

		payout2 := createTestPayout(merchant.ID)
		payout2.ID = uuid.New().String()
		payout2.Status = model.PayoutStatusApproved
		err = repo.Create(payout2)
		require.NoError(t, err)

		// Create requested payout
		payout3 := createTestPayout(merchant.ID)
		payout3.ID = uuid.New().String()
		payout3.Status = model.PayoutStatusRequested
		err = repo.Create(payout3)
		require.NoError(t, err)

		// Count approved payouts
		count, err := repo.CountByStatus(model.PayoutStatusApproved)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)

		// Count requested payouts
		count, err = repo.CountByStatus(model.PayoutStatusRequested)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("Error - Empty status", func(t *testing.T) {
		_, err := repo.CountByStatus("")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPayoutStatus)
	})
}

func TestPayoutRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayouts(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPayoutRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Soft delete payout", func(t *testing.T) {
		payout := createTestPayout(merchant.ID)
		err := repo.Create(payout)
		require.NoError(t, err)

		// Delete the payout
		err = repo.Delete(payout.ID)
		require.NoError(t, err)

		// Verify the payout is soft-deleted
		_, err = repo.GetByID(payout.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPayoutNotFound)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		err := repo.Delete("")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPayoutID)
	})

	t.Run("Error - Payout not found", func(t *testing.T) {
		err := repo.Delete(uuid.New().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPayoutNotFound)
	})
}
