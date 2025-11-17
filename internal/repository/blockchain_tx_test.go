package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/model"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cleanupBlockchainTxs removes all blockchain transactions from the test database
func cleanupBlockchainTxs(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("DELETE FROM blockchain_transactions")
	require.NoError(t, err, "Failed to cleanup blockchain_transactions table")
}

// createTestBlockchainTx creates a test blockchain transaction with default values
func createTestBlockchainTx() *model.BlockchainTransaction {
	return &model.BlockchainTransaction{
		ID:      uuid.New().String(),
		Chain:   model.ChainSolana,
		Network: model.NetworkDevnet,

		TxHash:      "5J8yv7h5qMxP3L8Y9VPwXKN4n2T6e5J8yv7h5qMxP3L8Y9VPwXKN4n2T6e5J8yv7h5qMxP3L8Y9V", // Mock Solana signature
		FromAddress: "SenderWalletAddress123456789ABCDEF",
		ToAddress:   "ReceiverWalletAddress123456789ABCDEF",

		Amount:    decimal.NewFromInt(100),
		Currency:  "USDT",
		TokenMint: sql.NullString{String: "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", Valid: true}, // USDT mint on Solana

		Memo:                   sql.NullString{String: "payment_ref_123456", Valid: true},
		ParsedPaymentReference: sql.NullString{String: "payment_ref_123456", Valid: true},

		Confirmations: 0,
		IsFinalized:   false,
		Status:        model.BlockchainTxStatusPending,

		RawTransaction: model.JSONBMap{"version": "legacy", "type": "transfer"},
		Metadata:       model.JSONBMap{},
	}
}

// createTestBlockchainTxWithTxHash creates a test blockchain transaction with a specific tx hash
func createTestBlockchainTxWithTxHash(txHash string) *model.BlockchainTransaction {
	tx := createTestBlockchainTx()
	tx.ID = uuid.New().String()
	tx.TxHash = txHash
	return tx
}

func TestBlockchainTxRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupBlockchainTxs(t, db)

	repo := NewBlockchainTxRepository(db)

	t.Run("Success - Create blockchain transaction", func(t *testing.T) {
		tx := createTestBlockchainTx()

		err := repo.Create(tx)
		require.NoError(t, err)

		// Verify the transaction was created
		assert.NotEmpty(t, tx.ID)
		assert.False(t, tx.CreatedAt.IsZero())
		assert.False(t, tx.UpdatedAt.IsZero())
	})

	t.Run("Success - Create with all fields", func(t *testing.T) {
		tx := createTestBlockchainTx()
		tx.TxHash = "UniqueHash123456789ABCDEF"
		tx.BlockNumber = sql.NullInt64{Int64: 12345, Valid: true}
		tx.BlockTimestamp = sql.NullTime{Time: time.Now(), Valid: true}
		tx.GasUsed = sql.NullInt64{Int64: 5000, Valid: true}
		tx.GasPrice = decimal.NullDecimal{Decimal: decimal.NewFromFloat(0.000005), Valid: true}
		tx.TxFee = decimal.NullDecimal{Decimal: decimal.NewFromFloat(0.000025), Valid: true}
		tx.FeeCurrency = sql.NullString{String: "SOL", Valid: true}

		err := repo.Create(tx)
		require.NoError(t, err)

		// Retrieve and verify all fields
		retrieved, err := repo.GetByTxHash(tx.TxHash)
		require.NoError(t, err)

		assert.Equal(t, tx.TxHash, retrieved.TxHash)
		assert.Equal(t, tx.BlockNumber.Int64, retrieved.BlockNumber.Int64)
		assert.True(t, tx.GasPrice.Decimal.Equal(retrieved.GasPrice.Decimal))
		assert.True(t, tx.TxFee.Decimal.Equal(retrieved.TxFee.Decimal))
		assert.Equal(t, tx.FeeCurrency.String, retrieved.FeeCurrency.String)
	})

	t.Run("Error - Nil transaction", func(t *testing.T) {
		err := repo.Create(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("Error - Empty tx hash", func(t *testing.T) {
		tx := createTestBlockchainTx()
		tx.TxHash = ""

		err := repo.Create(tx)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTxHash)
	})

	t.Run("Error - Duplicate tx hash", func(t *testing.T) {
		tx1 := createTestBlockchainTx()
		tx1.TxHash = "DUPLICATE-HASH-123"

		err := repo.Create(tx1)
		require.NoError(t, err)

		// Try to create another transaction with the same tx hash
		tx2 := createTestBlockchainTx()
		tx2.ID = uuid.New().String()
		tx2.TxHash = "DUPLICATE-HASH-123"

		err = repo.Create(tx2)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrBlockchainTxAlreadyExists)
	})
}

func TestBlockchainTxRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupBlockchainTxs(t, db)

	repo := NewBlockchainTxRepository(db)

	t.Run("Success - Get existing transaction", func(t *testing.T) {
		// Create a transaction first
		tx := createTestBlockchainTx()
		err := repo.Create(tx)
		require.NoError(t, err)

		// Retrieve the transaction
		retrieved, err := repo.GetByID(tx.ID)
		require.NoError(t, err)

		assert.Equal(t, tx.ID, retrieved.ID)
		assert.Equal(t, tx.TxHash, retrieved.TxHash)
		assert.Equal(t, tx.Chain, retrieved.Chain)
		assert.Equal(t, tx.Network, retrieved.Network)
		assert.True(t, tx.Amount.Equal(retrieved.Amount))
		assert.Equal(t, tx.Currency, retrieved.Currency)
		assert.Equal(t, tx.Status, retrieved.Status)
	})

	t.Run("Error - Transaction not found", func(t *testing.T) {
		_, err := repo.GetByID(uuid.New().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrBlockchainTxNotFound)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		_, err := repo.GetByID("")
		assert.Error(t, err)
	})
}

func TestBlockchainTxRepository_GetByTxHash(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupBlockchainTxs(t, db)

	repo := NewBlockchainTxRepository(db)

	t.Run("Success - Get existing transaction by hash", func(t *testing.T) {
		// Create a transaction first
		tx := createTestBlockchainTxWithTxHash("UniqueHash789XYZ")
		err := repo.Create(tx)
		require.NoError(t, err)

		// Retrieve the transaction by hash
		retrieved, err := repo.GetByTxHash("UniqueHash789XYZ")
		require.NoError(t, err)

		assert.Equal(t, tx.ID, retrieved.ID)
		assert.Equal(t, tx.TxHash, retrieved.TxHash)
		assert.Equal(t, tx.FromAddress, retrieved.FromAddress)
		assert.Equal(t, tx.ToAddress, retrieved.ToAddress)
	})

	t.Run("Error - Transaction not found", func(t *testing.T) {
		_, err := repo.GetByTxHash("NonExistentHash123")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrBlockchainTxNotFound)
	})

	t.Run("Error - Empty tx hash", func(t *testing.T) {
		_, err := repo.GetByTxHash("")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTxHash)
	})
}

func TestBlockchainTxRepository_UpdateConfirmations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupBlockchainTxs(t, db)

	repo := NewBlockchainTxRepository(db)

	t.Run("Success - Update confirmations", func(t *testing.T) {
		// Create a transaction first
		tx := createTestBlockchainTxWithTxHash("HashForConfirmations123")
		err := repo.Create(tx)
		require.NoError(t, err)

		// Update confirmations
		err = repo.UpdateConfirmations("HashForConfirmations123", 15)
		require.NoError(t, err)

		// Verify the confirmations were updated
		retrieved, err := repo.GetByTxHash("HashForConfirmations123")
		require.NoError(t, err)
		assert.Equal(t, 15, retrieved.Confirmations)
	})

	t.Run("Error - Transaction not found", func(t *testing.T) {
		err := repo.UpdateConfirmations("NonExistentHash456", 10)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrBlockchainTxNotFound)
	})

	t.Run("Error - Empty tx hash", func(t *testing.T) {
		err := repo.UpdateConfirmations("", 10)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTxHash)
	})

	t.Run("Error - Negative confirmations", func(t *testing.T) {
		tx := createTestBlockchainTxWithTxHash("HashForNegativeConfirm")
		err := repo.Create(tx)
		require.NoError(t, err)

		err = repo.UpdateConfirmations("HashForNegativeConfirm", -5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})
}

func TestBlockchainTxRepository_UpdateStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupBlockchainTxs(t, db)

	repo := NewBlockchainTxRepository(db)

	t.Run("Success - Update status to confirmed", func(t *testing.T) {
		tx := createTestBlockchainTxWithTxHash("HashForStatusUpdate")
		err := repo.Create(tx)
		require.NoError(t, err)

		err = repo.UpdateStatus("HashForStatusUpdate", model.BlockchainTxStatusConfirmed)
		require.NoError(t, err)

		retrieved, err := repo.GetByTxHash("HashForStatusUpdate")
		require.NoError(t, err)
		assert.Equal(t, model.BlockchainTxStatusConfirmed, retrieved.Status)
	})

	t.Run("Error - Transaction not found", func(t *testing.T) {
		err := repo.UpdateStatus("NonExistentHashStatus", model.BlockchainTxStatusConfirmed)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrBlockchainTxNotFound)
	})
}

func TestBlockchainTxRepository_MarkAsFinalized(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupBlockchainTxs(t, db)

	repo := NewBlockchainTxRepository(db)

	t.Run("Success - Mark as finalized", func(t *testing.T) {
		tx := createTestBlockchainTxWithTxHash("HashForFinalization")
		err := repo.Create(tx)
		require.NoError(t, err)

		err = repo.MarkAsFinalized("HashForFinalization")
		require.NoError(t, err)

		retrieved, err := repo.GetByTxHash("HashForFinalization")
		require.NoError(t, err)
		assert.True(t, retrieved.IsFinalized)
		assert.Equal(t, model.BlockchainTxStatusFinalized, retrieved.Status)
		assert.True(t, retrieved.FinalizedAt.Valid)
		assert.False(t, retrieved.FinalizedAt.Time.IsZero())
	})

	t.Run("Error - Transaction not found", func(t *testing.T) {
		err := repo.MarkAsFinalized("NonExistentHashFinalize")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrBlockchainTxNotFound)
	})

	t.Run("Error - Empty tx hash", func(t *testing.T) {
		err := repo.MarkAsFinalized("")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTxHash)
	})
}

func TestBlockchainTxRepository_MatchToPayment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupBlockchainTxs(t, db)
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewBlockchainTxRepository(db)
	paymentRepo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	t.Run("Success - Match to payment", func(t *testing.T) {
		// Create a merchant and payment
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		payment := createTestPayment(merchant.ID)
		err = paymentRepo.Create(payment)
		require.NoError(t, err)

		// Create a blockchain transaction
		tx := createTestBlockchainTxWithTxHash("HashForMatching")
		err = repo.Create(tx)
		require.NoError(t, err)

		// Match to payment
		err = repo.MatchToPayment("HashForMatching", payment.ID)
		require.NoError(t, err)

		// Verify the match
		retrieved, err := repo.GetByTxHash("HashForMatching")
		require.NoError(t, err)
		assert.True(t, retrieved.IsMatched)
		assert.True(t, retrieved.PaymentID.Valid)
		assert.Equal(t, payment.ID, retrieved.PaymentID.String)
		assert.True(t, retrieved.MatchedAt.Valid)
		assert.False(t, retrieved.MatchedAt.Time.IsZero())
	})

	t.Run("Error - Empty tx hash", func(t *testing.T) {
		err := repo.MatchToPayment("", "some-payment-id")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTxHash)
	})

	t.Run("Error - Empty payment ID", func(t *testing.T) {
		tx := createTestBlockchainTxWithTxHash("HashForEmptyPayment")
		err := repo.Create(tx)
		require.NoError(t, err)

		err = repo.MatchToPayment("HashForEmptyPayment", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "payment ID cannot be empty")
	})
}

func TestBlockchainTxRepository_MarkAsFailed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupBlockchainTxs(t, db)

	repo := NewBlockchainTxRepository(db)

	t.Run("Success - Mark as failed", func(t *testing.T) {
		tx := createTestBlockchainTxWithTxHash("HashForFailure")
		err := repo.Create(tx)
		require.NoError(t, err)

		errorMsg := "Transaction failed due to insufficient funds"
		err = repo.MarkAsFailed("HashForFailure", errorMsg)
		require.NoError(t, err)

		retrieved, err := repo.GetByTxHash("HashForFailure")
		require.NoError(t, err)
		assert.Equal(t, model.BlockchainTxStatusFailed, retrieved.Status)
		assert.True(t, retrieved.ErrorMessage.Valid)
		assert.Equal(t, errorMsg, retrieved.ErrorMessage.String)
	})

	t.Run("Error - Transaction not found", func(t *testing.T) {
		err := repo.MarkAsFailed("NonExistentHashFail", "Some error")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrBlockchainTxNotFound)
	})
}

func TestBlockchainTxRepository_GetPendingTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupBlockchainTxs(t, db)

	repo := NewBlockchainTxRepository(db)

	t.Run("Success - Get pending transactions", func(t *testing.T) {
		// Create multiple transactions with different statuses
		tx1 := createTestBlockchainTxWithTxHash("PendingHash1")
		tx1.Status = model.BlockchainTxStatusPending
		err := repo.Create(tx1)
		require.NoError(t, err)

		tx2 := createTestBlockchainTxWithTxHash("PendingHash2")
		tx2.Status = model.BlockchainTxStatusPending
		err = repo.Create(tx2)
		require.NoError(t, err)

		tx3 := createTestBlockchainTxWithTxHash("ConfirmedHash1")
		tx3.Status = model.BlockchainTxStatusConfirmed
		err = repo.Create(tx3)
		require.NoError(t, err)

		// Get pending transactions
		pending, err := repo.GetPendingTransactions()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(pending), 2)

		// Verify all returned transactions are pending
		for _, tx := range pending {
			assert.Equal(t, model.BlockchainTxStatusPending, tx.Status)
			assert.False(t, tx.IsFinalized)
		}
	})

	t.Run("Success - Empty list when no pending", func(t *testing.T) {
		// Clean up first
		cleanupBlockchainTxs(t, db)

		// Create only confirmed transactions
		tx := createTestBlockchainTxWithTxHash("OnlyConfirmedHash")
		tx.Status = model.BlockchainTxStatusConfirmed
		err := repo.Create(tx)
		require.NoError(t, err)

		pending, err := repo.GetPendingTransactions()
		require.NoError(t, err)
		assert.Equal(t, 0, len(pending))
	})
}

func TestBlockchainTxRepository_GetUnmatchedTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupBlockchainTxs(t, db)

	repo := NewBlockchainTxRepository(db)

	t.Run("Success - Get unmatched transactions", func(t *testing.T) {
		// Create unmatched confirmed transaction with payment reference
		tx1 := createTestBlockchainTxWithTxHash("UnmatchedHash1")
		tx1.Status = model.BlockchainTxStatusConfirmed
		tx1.IsMatched = false
		tx1.ParsedPaymentReference = sql.NullString{String: "payment_ref_123", Valid: true}
		err := repo.Create(tx1)
		require.NoError(t, err)

		// Create matched transaction (should not be returned)
		tx2 := createTestBlockchainTxWithTxHash("MatchedHash1")
		tx2.Status = model.BlockchainTxStatusConfirmed
		tx2.IsMatched = true
		tx2.ParsedPaymentReference = sql.NullString{String: "payment_ref_456", Valid: true}
		err = repo.Create(tx2)
		require.NoError(t, err)

		// Create pending unmatched (should not be returned)
		tx3 := createTestBlockchainTxWithTxHash("UnmatchedPendingHash")
		tx3.Status = model.BlockchainTxStatusPending
		tx3.IsMatched = false
		tx3.ParsedPaymentReference = sql.NullString{String: "payment_ref_789", Valid: true}
		err = repo.Create(tx3)
		require.NoError(t, err)

		// Get unmatched transactions
		unmatched, err := repo.GetUnmatchedTransactions(model.ChainSolana)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(unmatched), 1)

		// Verify all returned transactions are unmatched and confirmed/finalized
		for _, tx := range unmatched {
			assert.False(t, tx.IsMatched)
			assert.True(t, tx.ParsedPaymentReference.Valid)
			assert.True(t, tx.Status == model.BlockchainTxStatusConfirmed || tx.Status == model.BlockchainTxStatusFinalized)
		}
	})
}

func TestBlockchainTxRepository_GetByPaymentID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupBlockchainTxs(t, db)
	defer cleanupPayments(t, db)
	defer cleanupMerchants(t, db)

	repo := NewBlockchainTxRepository(db)
	paymentRepo := NewPaymentRepository(db)
	merchantRepo := NewMerchantRepository(db)

	t.Run("Success - Get transactions by payment ID", func(t *testing.T) {
		// Create a merchant and payment
		merchant := createTestMerchant()
		err := merchantRepo.Create(merchant)
		require.NoError(t, err)

		payment := createTestPayment(merchant.ID)
		err = paymentRepo.Create(payment)
		require.NoError(t, err)

		// Create blockchain transactions associated with the payment
		tx1 := createTestBlockchainTxWithTxHash("PaymentTxHash1")
		err = repo.Create(tx1)
		require.NoError(t, err)

		err = repo.MatchToPayment("PaymentTxHash1", payment.ID)
		require.NoError(t, err)

		tx2 := createTestBlockchainTxWithTxHash("PaymentTxHash2")
		err = repo.Create(tx2)
		require.NoError(t, err)

		err = repo.MatchToPayment("PaymentTxHash2", payment.ID)
		require.NoError(t, err)

		// Get transactions by payment ID
		txs, err := repo.GetByPaymentID(payment.ID)
		require.NoError(t, err)

		assert.Equal(t, 2, len(txs))
		for _, tx := range txs {
			assert.True(t, tx.PaymentID.Valid)
			assert.Equal(t, payment.ID, tx.PaymentID.String)
		}
	})

	t.Run("Error - Empty payment ID", func(t *testing.T) {
		_, err := repo.GetByPaymentID("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "payment ID cannot be empty")
	})

	t.Run("Success - Empty list for non-existent payment", func(t *testing.T) {
		txs, err := repo.GetByPaymentID(uuid.New().String())
		require.NoError(t, err)
		assert.Equal(t, 0, len(txs))
	})
}

func TestBlockchainTxRepository_ListByChainAndStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupBlockchainTxs(t, db)

	repo := NewBlockchainTxRepository(db)

	t.Run("Success - List by chain and status with pagination", func(t *testing.T) {
		// Create multiple transactions
		for i := 0; i < 5; i++ {
			tx := createTestBlockchainTxWithTxHash(uuid.New().String())
			tx.Chain = model.ChainSolana
			tx.Status = model.BlockchainTxStatusConfirmed
			err := repo.Create(tx)
			require.NoError(t, err)
		}

		// Get first page
		txs, err := repo.ListByChainAndStatus(model.ChainSolana, model.BlockchainTxStatusConfirmed, 3, 0)
		require.NoError(t, err)

		assert.LessOrEqual(t, len(txs), 3)
		for _, tx := range txs {
			assert.Equal(t, model.ChainSolana, tx.Chain)
			assert.Equal(t, model.BlockchainTxStatusConfirmed, tx.Status)
		}

		// Get second page
		txs2, err := repo.ListByChainAndStatus(model.ChainSolana, model.BlockchainTxStatusConfirmed, 3, 3)
		require.NoError(t, err)

		assert.LessOrEqual(t, len(txs2), 3)
	})

	t.Run("Success - Empty list for no matches", func(t *testing.T) {
		cleanupBlockchainTxs(t, db)

		txs, err := repo.ListByChainAndStatus(model.ChainBSC, model.BlockchainTxStatusFailed, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 0, len(txs))
	})
}
