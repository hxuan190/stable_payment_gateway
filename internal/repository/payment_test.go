package repository

import (
	"database/sql"
	"testing"
	"time"

	"stable_payment_gateway/internal/model"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cleanupPayments removes all payments from the test database
func cleanupPayments(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("DELETE FROM payments")
	require.NoError(t, err, "Failed to cleanup payments table")
}

// createTestPayment creates a test payment with default values
func createTestPayment(merchantID string) *model.Payment {
	now := time.Now()
	expiresAt := now.Add(30 * time.Minute)

	return &model.Payment{
		ID:                uuid.New().String(),
		MerchantID:        merchantID,
		AmountVND:         decimal.NewFromInt(2300000),
		AmountCrypto:      decimal.NewFromInt(100),
		Currency:          "USDT",
		Chain:             model.ChainSolana,
		ExchangeRate:      decimal.NewFromInt(23000),
		OrderID:           sql.NullString{String: "ORDER-123", Valid: true},
		Description:       sql.NullString{String: "Test payment", Valid: true},
		CallbackURL:       sql.NullString{String: "https://example.com/webhook", Valid: true},
		Status:            model.PaymentStatusCreated,
		PaymentReference:  uuid.New().String(),
		DestinationWallet: "SolanaWalletAddress123456789",
		ExpiresAt:         expiresAt,
		FeePercentage:     decimal.NewFromFloat(0.01),
		FeeVND:            decimal.NewFromInt(23000),
		NetAmountVND:      decimal.NewFromInt(2277000),
		Metadata:          model.JSONBMap{},
	}
}

func TestPaymentRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Create payment", func(t *testing.T) {
		payment := createTestPayment(merchant.ID)

		err := repo.Create(payment)
		require.NoError(t, err)

		// Verify the payment was created
		assert.NotEmpty(t, payment.ID)
		assert.False(t, payment.CreatedAt.IsZero())
		assert.False(t, payment.UpdatedAt.IsZero())
	})

	t.Run("Error - Nil payment", func(t *testing.T) {
		err := repo.Create(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("Error - Duplicate payment reference", func(t *testing.T) {
		payment1 := createTestPayment(merchant.ID)
		payment1.PaymentReference = "UNIQUE-REF-123"

		err := repo.Create(payment1)
		require.NoError(t, err)

		// Try to create another payment with the same reference
		payment2 := createTestPayment(merchant.ID)
		payment2.ID = uuid.New().String()
		payment2.PaymentReference = "UNIQUE-REF-123"

		err = repo.Create(payment2)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPaymentAlreadyExists)
	})
}

func TestPaymentRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Get existing payment", func(t *testing.T) {
		// Create a payment first
		payment := createTestPayment(merchant.ID)
		err := repo.Create(payment)
		require.NoError(t, err)

		// Retrieve the payment
		retrieved, err := repo.GetByID(payment.ID)
		require.NoError(t, err)

		assert.Equal(t, payment.ID, retrieved.ID)
		assert.Equal(t, payment.MerchantID, retrieved.MerchantID)
		assert.True(t, payment.AmountVND.Equal(retrieved.AmountVND))
		assert.True(t, payment.AmountCrypto.Equal(retrieved.AmountCrypto))
		assert.Equal(t, payment.Currency, retrieved.Currency)
		assert.Equal(t, payment.Chain, retrieved.Chain)
		assert.Equal(t, payment.Status, retrieved.Status)
		assert.Equal(t, payment.PaymentReference, retrieved.PaymentReference)
	})

	t.Run("Error - Payment not found", func(t *testing.T) {
		_, err := repo.GetByID(uuid.New().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPaymentNotFound)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		_, err := repo.GetByID("")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPaymentID)
	})
}

func TestPaymentRepository_GetByTxHash(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Get payment by tx hash", func(t *testing.T) {
		payment := createTestPayment(merchant.ID)
		payment.TxHash = sql.NullString{String: "TX_HASH_123456789", Valid: true}
		err := repo.Create(payment)
		require.NoError(t, err)

		retrieved, err := repo.GetByTxHash("TX_HASH_123456789")
		require.NoError(t, err)

		assert.Equal(t, payment.ID, retrieved.ID)
		assert.Equal(t, "TX_HASH_123456789", retrieved.TxHash.String)
	})

	t.Run("Error - Payment not found", func(t *testing.T) {
		_, err := repo.GetByTxHash("NONEXISTENT_TX_HASH")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPaymentNotFound)
	})

	t.Run("Error - Empty tx hash", func(t *testing.T) {
		_, err := repo.GetByTxHash("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction hash cannot be empty")
	})
}

func TestPaymentRepository_GetByPaymentReference(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Get payment by reference", func(t *testing.T) {
		payment := createTestPayment(merchant.ID)
		payment.PaymentReference = "UNIQUE-REF-456"
		err := repo.Create(payment)
		require.NoError(t, err)

		retrieved, err := repo.GetByPaymentReference("UNIQUE-REF-456")
		require.NoError(t, err)

		assert.Equal(t, payment.ID, retrieved.ID)
		assert.Equal(t, "UNIQUE-REF-456", retrieved.PaymentReference)
	})

	t.Run("Error - Payment not found", func(t *testing.T) {
		_, err := repo.GetByPaymentReference("NONEXISTENT-REF")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPaymentNotFound)
	})

	t.Run("Error - Empty reference", func(t *testing.T) {
		_, err := repo.GetByPaymentReference("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "payment reference cannot be empty")
	})
}

func TestPaymentRepository_UpdateStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Update payment status", func(t *testing.T) {
		payment := createTestPayment(merchant.ID)
		err := repo.Create(payment)
		require.NoError(t, err)

		// Update status
		err = repo.UpdateStatus(payment.ID, model.PaymentStatusCompleted)
		require.NoError(t, err)

		// Verify the status was updated
		retrieved, err := repo.GetByID(payment.ID)
		require.NoError(t, err)
		assert.Equal(t, model.PaymentStatusCompleted, retrieved.Status)
	})

	t.Run("Error - Invalid payment ID", func(t *testing.T) {
		err := repo.UpdateStatus("", model.PaymentStatusCompleted)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPaymentID)
	})

	t.Run("Error - Invalid status", func(t *testing.T) {
		payment := createTestPayment(merchant.ID)
		err := repo.Create(payment)
		require.NoError(t, err)

		err = repo.UpdateStatus(payment.ID, "invalid_status")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPaymentStatus)
	})

	t.Run("Error - Payment not found", func(t *testing.T) {
		err := repo.UpdateStatus(uuid.New().String(), model.PaymentStatusCompleted)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPaymentNotFound)
	})
}

func TestPaymentRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Update payment", func(t *testing.T) {
		payment := createTestPayment(merchant.ID)
		err := repo.Create(payment)
		require.NoError(t, err)

		// Update payment
		payment.Status = model.PaymentStatusCompleted
		payment.TxHash = sql.NullString{String: "NEW_TX_HASH", Valid: true}
		payment.ConfirmedAt = sql.NullTime{Time: time.Now(), Valid: true}

		err = repo.Update(payment)
		require.NoError(t, err)

		// Verify the updates
		retrieved, err := repo.GetByID(payment.ID)
		require.NoError(t, err)
		assert.Equal(t, model.PaymentStatusCompleted, retrieved.Status)
		assert.Equal(t, "NEW_TX_HASH", retrieved.TxHash.String)
		assert.True(t, retrieved.ConfirmedAt.Valid)
	})

	t.Run("Error - Nil payment", func(t *testing.T) {
		err := repo.Update(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		payment := createTestPayment(merchant.ID)
		payment.ID = ""
		err := repo.Update(payment)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPaymentID)
	})

	t.Run("Error - Payment not found", func(t *testing.T) {
		payment := createTestPayment(merchant.ID)
		payment.ID = uuid.New().String()
		err := repo.Update(payment)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPaymentNotFound)
	})
}

func TestPaymentRepository_ListByMerchant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - List payments by merchant", func(t *testing.T) {
		// Create multiple payments for the merchant
		for i := 0; i < 5; i++ {
			payment := createTestPayment(merchant.ID)
			payment.PaymentReference = uuid.New().String()
			err := repo.Create(payment)
			require.NoError(t, err)
		}

		// List payments
		payments, err := repo.ListByMerchant(merchant.ID, 10, 0)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(payments), 5)
		for _, payment := range payments {
			assert.Equal(t, merchant.ID, payment.MerchantID)
		}
	})

	t.Run("Success - Pagination", func(t *testing.T) {
		// List with limit 2
		payments, err := repo.ListByMerchant(merchant.ID, 2, 0)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(payments), 2)

		// List second page
		payments2, err := repo.ListByMerchant(merchant.ID, 2, 2)
		require.NoError(t, err)

		// Ensure different payments are returned
		if len(payments) > 0 && len(payments2) > 0 {
			assert.NotEqual(t, payments[0].ID, payments2[0].ID)
		}
	})

	t.Run("Error - Empty merchant ID", func(t *testing.T) {
		_, err := repo.ListByMerchant("", 10, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant ID cannot be empty")
	})
}

func TestPaymentRepository_GetExpiredPayments(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Get expired payments", func(t *testing.T) {
		// Create an expired payment
		expiredPayment := createTestPayment(merchant.ID)
		expiredPayment.ExpiresAt = time.Now().Add(-10 * time.Minute) // Expired 10 minutes ago
		expiredPayment.Status = model.PaymentStatusCreated
		err := repo.Create(expiredPayment)
		require.NoError(t, err)

		// Create a non-expired payment
		validPayment := createTestPayment(merchant.ID)
		validPayment.PaymentReference = uuid.New().String()
		validPayment.ExpiresAt = time.Now().Add(10 * time.Minute) // Expires in 10 minutes
		validPayment.Status = model.PaymentStatusCreated
		err = repo.Create(validPayment)
		require.NoError(t, err)

		// Create a completed payment (should not be returned even if expired)
		completedPayment := createTestPayment(merchant.ID)
		completedPayment.PaymentReference = uuid.New().String()
		completedPayment.ExpiresAt = time.Now().Add(-10 * time.Minute)
		completedPayment.Status = model.PaymentStatusCompleted
		err = repo.Create(completedPayment)
		require.NoError(t, err)

		// Get expired payments
		expiredPayments, err := repo.GetExpiredPayments()
		require.NoError(t, err)

		// Verify that only the expired payment in created/pending status is returned
		found := false
		for _, payment := range expiredPayments {
			if payment.ID == expiredPayment.ID {
				found = true
				assert.True(t, payment.ExpiresAt.Before(time.Now()))
				assert.Contains(t, []model.PaymentStatus{model.PaymentStatusCreated, model.PaymentStatusPending}, payment.Status)
			}
			// Ensure valid payment and completed payment are not in the list
			assert.NotEqual(t, validPayment.ID, payment.ID)
			assert.NotEqual(t, completedPayment.ID, payment.ID)
		}
		assert.True(t, found, "Expired payment should be in the list")
	})
}

func TestPaymentRepository_ListByStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - List payments by status", func(t *testing.T) {
		// Create payments with different statuses
		payment1 := createTestPayment(merchant.ID)
		payment1.Status = model.PaymentStatusCompleted
		err := repo.Create(payment1)
		require.NoError(t, err)

		payment2 := createTestPayment(merchant.ID)
		payment2.PaymentReference = uuid.New().String()
		payment2.Status = model.PaymentStatusCompleted
		err = repo.Create(payment2)
		require.NoError(t, err)

		payment3 := createTestPayment(merchant.ID)
		payment3.PaymentReference = uuid.New().String()
		payment3.Status = model.PaymentStatusPending
		err = repo.Create(payment3)
		require.NoError(t, err)

		// List completed payments
		completedPayments, err := repo.ListByStatus(model.PaymentStatusCompleted, 10, 0)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(completedPayments), 2)
		for _, payment := range completedPayments {
			assert.Equal(t, model.PaymentStatusCompleted, payment.Status)
		}
	})

	t.Run("Error - Empty status", func(t *testing.T) {
		_, err := repo.ListByStatus("", 10, 0)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPaymentStatus)
	})
}

func TestPaymentRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Count payments", func(t *testing.T) {
		initialCount, err := repo.Count()
		require.NoError(t, err)

		// Create payments
		for i := 0; i < 3; i++ {
			payment := createTestPayment(merchant.ID)
			payment.PaymentReference = uuid.New().String()
			err := repo.Create(payment)
			require.NoError(t, err)
		}

		newCount, err := repo.Count()
		require.NoError(t, err)

		assert.Equal(t, initialCount+3, newCount)
	})
}

func TestPaymentRepository_CountByMerchant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create two test merchants
	merchant1 := createTestMerchant()
	err := merchantRepo.Create(merchant1)
	require.NoError(t, err)

	merchant2 := createTestMerchant()
	merchant2.Email = "merchant2@example.com"
	merchant2.ID = uuid.New().String()
	err = merchantRepo.Create(merchant2)
	require.NoError(t, err)

	t.Run("Success - Count payments by merchant", func(t *testing.T) {
		// Create payments for merchant 1
		for i := 0; i < 3; i++ {
			payment := createTestPayment(merchant1.ID)
			payment.PaymentReference = uuid.New().String()
			err := repo.Create(payment)
			require.NoError(t, err)
		}

		// Create payments for merchant 2
		for i := 0; i < 2; i++ {
			payment := createTestPayment(merchant2.ID)
			payment.PaymentReference = uuid.New().String()
			err := repo.Create(payment)
			require.NoError(t, err)
		}

		count1, err := repo.CountByMerchant(merchant1.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count1)

		count2, err := repo.CountByMerchant(merchant2.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count2)
	})

	t.Run("Error - Empty merchant ID", func(t *testing.T) {
		_, err := repo.CountByMerchant("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant ID cannot be empty")
	})
}

func TestPaymentRepository_CountByStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Count payments by status", func(t *testing.T) {
		// Create payments with different statuses
		for i := 0; i < 3; i++ {
			payment := createTestPayment(merchant.ID)
			payment.PaymentReference = uuid.New().String()
			payment.Status = model.PaymentStatusCompleted
			err := repo.Create(payment)
			require.NoError(t, err)
		}

		for i := 0; i < 2; i++ {
			payment := createTestPayment(merchant.ID)
			payment.PaymentReference = uuid.New().String()
			payment.Status = model.PaymentStatusPending
			err := repo.Create(payment)
			require.NoError(t, err)
		}

		completedCount, err := repo.CountByStatus(model.PaymentStatusCompleted)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, completedCount, int64(3))

		pendingCount, err := repo.CountByStatus(model.PaymentStatusPending)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, pendingCount, int64(2))
	})

	t.Run("Error - Empty status", func(t *testing.T) {
		_, err := repo.CountByStatus("")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPaymentStatus)
	})
}

func TestPaymentRepository_GetTotalVolumeByMerchant(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Get total volume", func(t *testing.T) {
		// Create completed payments
		payment1 := createTestPayment(merchant.ID)
		payment1.Status = model.PaymentStatusCompleted
		payment1.AmountVND = decimal.NewFromInt(1000000)
		err := repo.Create(payment1)
		require.NoError(t, err)

		payment2 := createTestPayment(merchant.ID)
		payment2.PaymentReference = uuid.New().String()
		payment2.Status = model.PaymentStatusCompleted
		payment2.AmountVND = decimal.NewFromInt(2000000)
		err = repo.Create(payment2)
		require.NoError(t, err)

		// Create a pending payment (should not be counted)
		payment3 := createTestPayment(merchant.ID)
		payment3.PaymentReference = uuid.New().String()
		payment3.Status = model.PaymentStatusPending
		payment3.AmountVND = decimal.NewFromInt(500000)
		err = repo.Create(payment3)
		require.NoError(t, err)

		totalVolume, err := repo.GetTotalVolumeByMerchant(merchant.ID)
		require.NoError(t, err)

		expectedVolume := decimal.NewFromInt(3000000)
		assert.True(t, totalVolume.Equal(expectedVolume), "Expected %s but got %s", expectedVolume, totalVolume)
	})

	t.Run("Error - Empty merchant ID", func(t *testing.T) {
		_, err := repo.GetTotalVolumeByMerchant("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant ID cannot be empty")
	})
}

func TestPaymentRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	// Create a test merchant first
	merchant := createTestMerchant()
	err := merchantRepo.Create(merchant)
	require.NoError(t, err)

	t.Run("Success - Delete payment", func(t *testing.T) {
		payment := createTestPayment(merchant.ID)
		err := repo.Create(payment)
		require.NoError(t, err)

		// Delete the payment
		err = repo.Delete(payment.ID)
		require.NoError(t, err)

		// Verify the payment is soft-deleted
		_, err = repo.GetByID(payment.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPaymentNotFound)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		err := repo.Delete("")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidPaymentID)
	})

	t.Run("Error - Payment not found", func(t *testing.T) {
		err := repo.Delete(uuid.New().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrPaymentNotFound)
	})
}
