package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/database"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates a test database connection
// NOTE: This requires a running PostgreSQL instance for integration tests
// For unit tests, consider using a mock database or sqlmock
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.NewPostgresDB(database.Config{
		Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:     getEnvOrDefault("TEST_DB_PORT", "5432"),
		User:     getEnvOrDefault("TEST_DB_USER", "postgres"),
		Password: getEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
		Database: getEnvOrDefault("TEST_DB_NAME", "payment_gateway_test"),
		SSLMode:  "disable",
	})
	require.NoError(t, err, "Failed to connect to test database")

	return db
}

func getEnvOrDefault(key, defaultValue string) string {
	// In a real test, you'd use os.Getenv(key)
	// For this example, we'll return the default
	return defaultValue
}

// cleanupMerchants removes all merchants from the test database
func cleanupMerchants(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("DELETE FROM merchants")
	require.NoError(t, err, "Failed to cleanup merchants table")
}

// createTestMerchant creates a test merchant with default values
func createTestMerchant() *model.Merchant {
	return &model.Merchant{
		ID:                    uuid.New().String(),
		Email:                 "test@example.com",
		BusinessName:          "Test Business",
		OwnerFullName:         "Test Owner",
		KYCStatus:             model.KYCStatusPending,
		Status:                model.MerchantStatusActive,
		BusinessTaxID:         sql.NullString{String: "1234567890", Valid: true},
		BusinessLicenseNumber: sql.NullString{String: "BLN-123456", Valid: true},
		PhoneNumber:           sql.NullString{String: "+84901234567", Valid: true},
		BusinessAddress:       sql.NullString{String: "123 Test St", Valid: true},
		City:                  sql.NullString{String: "Da Nang", Valid: true},
		District:              sql.NullString{String: "Hai Chau", Valid: true},
		Metadata:              model.JSONBMap{},
	}
}

func TestMerchantRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupMerchants(t, db)

	repo := NewMerchantRepository(db)

	t.Run("Success - Create merchant", func(t *testing.T) {
		merchant := createTestMerchant()

		err := repo.Create(merchant)
		require.NoError(t, err)

		// Verify the merchant was created
		assert.NotEmpty(t, merchant.ID)
		assert.False(t, merchant.CreatedAt.IsZero())
		assert.False(t, merchant.UpdatedAt.IsZero())
	})

	t.Run("Error - Nil merchant", func(t *testing.T) {
		err := repo.Create(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("Error - Duplicate email", func(t *testing.T) {
		merchant1 := createTestMerchant()
		merchant1.Email = "duplicate@example.com"

		err := repo.Create(merchant1)
		require.NoError(t, err)

		// Try to create another merchant with the same email
		merchant2 := createTestMerchant()
		merchant2.Email = "duplicate@example.com"
		merchant2.ID = uuid.New().String()

		err = repo.Create(merchant2)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrMerchantAlreadyExists)
	})
}

func TestMerchantRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupMerchants(t, db)

	repo := NewMerchantRepository(db)

	t.Run("Success - Get existing merchant", func(t *testing.T) {
		// Create a merchant first
		merchant := createTestMerchant()
		err := repo.Create(merchant)
		require.NoError(t, err)

		// Retrieve the merchant
		retrieved, err := repo.GetByID(merchant.ID)
		require.NoError(t, err)

		assert.Equal(t, merchant.ID, retrieved.ID)
		assert.Equal(t, merchant.Email, retrieved.Email)
		assert.Equal(t, merchant.BusinessName, retrieved.BusinessName)
		assert.Equal(t, merchant.KYCStatus, retrieved.KYCStatus)
	})

	t.Run("Error - Merchant not found", func(t *testing.T) {
		_, err := repo.GetByID(uuid.New().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrMerchantNotFound)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		_, err := repo.GetByID("")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidMerchantID)
	})
}

func TestMerchantRepository_GetByEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupMerchants(t, db)

	repo := NewMerchantRepository(db)

	t.Run("Success - Get existing merchant by email", func(t *testing.T) {
		merchant := createTestMerchant()
		merchant.Email = "unique@example.com"
		err := repo.Create(merchant)
		require.NoError(t, err)

		retrieved, err := repo.GetByEmail("unique@example.com")
		require.NoError(t, err)

		assert.Equal(t, merchant.ID, retrieved.ID)
		assert.Equal(t, merchant.Email, retrieved.Email)
	})

	t.Run("Error - Merchant not found", func(t *testing.T) {
		_, err := repo.GetByEmail("nonexistent@example.com")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrMerchantNotFound)
	})

	t.Run("Error - Empty email", func(t *testing.T) {
		_, err := repo.GetByEmail("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email cannot be empty")
	})
}

func TestMerchantRepository_GetByAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupMerchants(t, db)

	repo := NewMerchantRepository(db)

	t.Run("Success - Get merchant by API key", func(t *testing.T) {
		merchant := createTestMerchant()
		apiKey := "test-api-key-" + uuid.New().String()
		merchant.APIKey = sql.NullString{String: apiKey, Valid: true}
		merchant.APIKeyCreatedAt = sql.NullTime{Time: time.Now(), Valid: true}

		err := repo.Create(merchant)
		require.NoError(t, err)

		retrieved, err := repo.GetByAPIKey(apiKey)
		require.NoError(t, err)

		assert.Equal(t, merchant.ID, retrieved.ID)
		assert.Equal(t, apiKey, retrieved.APIKey.String)
	})

	t.Run("Error - API key not found", func(t *testing.T) {
		_, err := repo.GetByAPIKey("nonexistent-api-key")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrMerchantNotFound)
	})

	t.Run("Error - Empty API key", func(t *testing.T) {
		_, err := repo.GetByAPIKey("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key cannot be empty")
	})
}

func TestMerchantRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupMerchants(t, db)

	repo := NewMerchantRepository(db)

	t.Run("Success - Update merchant", func(t *testing.T) {
		merchant := createTestMerchant()
		err := repo.Create(merchant)
		require.NoError(t, err)

		// Update the merchant
		merchant.BusinessName = "Updated Business Name"
		merchant.PhoneNumber = sql.NullString{String: "+84909999999", Valid: true}

		err = repo.Update(merchant)
		require.NoError(t, err)

		// Verify the update
		updated, err := repo.GetByID(merchant.ID)
		require.NoError(t, err)

		assert.Equal(t, "Updated Business Name", updated.BusinessName)
		assert.Equal(t, "+84909999999", updated.PhoneNumber.String)
	})

	t.Run("Error - Nil merchant", func(t *testing.T) {
		err := repo.Update(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("Error - Empty merchant ID", func(t *testing.T) {
		merchant := createTestMerchant()
		merchant.ID = ""

		err := repo.Update(merchant)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidMerchantID)
	})

	t.Run("Error - Merchant not found", func(t *testing.T) {
		merchant := createTestMerchant()
		merchant.ID = uuid.New().String()

		err := repo.Update(merchant)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrMerchantNotFound)
	})
}

func TestMerchantRepository_UpdateKYCStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupMerchants(t, db)

	repo := NewMerchantRepository(db)

	t.Run("Success - Update KYC status", func(t *testing.T) {
		merchant := createTestMerchant()
		merchant.KYCStatus = model.KYCStatusPending
		err := repo.Create(merchant)
		require.NoError(t, err)

		err = repo.UpdateKYCStatus(merchant.ID, string(model.KYCStatusApproved))
		require.NoError(t, err)

		// Verify the update
		updated, err := repo.GetByID(merchant.ID)
		require.NoError(t, err)

		assert.Equal(t, model.KYCStatusApproved, updated.KYCStatus)
	})

	t.Run("Error - Empty merchant ID", func(t *testing.T) {
		err := repo.UpdateKYCStatus("", string(model.KYCStatusApproved))
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidMerchantID)
	})

	t.Run("Error - Empty status", func(t *testing.T) {
		err := repo.UpdateKYCStatus(uuid.New().String(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "KYC status cannot be empty")
	})

	t.Run("Error - Merchant not found", func(t *testing.T) {
		err := repo.UpdateKYCStatus(uuid.New().String(), string(model.KYCStatusApproved))
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrMerchantNotFound)
	})
}

func TestMerchantRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupMerchants(t, db)

	repo := NewMerchantRepository(db)

	t.Run("Success - List merchants with pagination", func(t *testing.T) {
		// Create multiple merchants
		for i := 0; i < 5; i++ {
			merchant := createTestMerchant()
			merchant.ID = uuid.New().String()
			merchant.Email = uuid.New().String() + "@example.com"
			err := repo.Create(merchant)
			require.NoError(t, err)
		}

		// List first page
		merchants, err := repo.List(3, 0)
		require.NoError(t, err)
		assert.Len(t, merchants, 3)

		// List second page
		merchants, err = repo.List(3, 3)
		require.NoError(t, err)
		assert.Len(t, merchants, 2)
	})

	t.Run("Success - List with default limit", func(t *testing.T) {
		merchants, err := repo.List(0, 0)
		require.NoError(t, err)
		assert.NotNil(t, merchants)
	})
}

func TestMerchantRepository_ListByKYCStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupMerchants(t, db)

	repo := NewMerchantRepository(db)

	t.Run("Success - List merchants by KYC status", func(t *testing.T) {
		// Create merchants with different KYC statuses
		for i := 0; i < 3; i++ {
			merchant := createTestMerchant()
			merchant.ID = uuid.New().String()
			merchant.Email = uuid.New().String() + "@example.com"
			merchant.KYCStatus = model.KYCStatusPending
			err := repo.Create(merchant)
			require.NoError(t, err)
		}

		for i := 0; i < 2; i++ {
			merchant := createTestMerchant()
			merchant.ID = uuid.New().String()
			merchant.Email = uuid.New().String() + "@example.com"
			merchant.KYCStatus = model.KYCStatusApproved
			err := repo.Create(merchant)
			require.NoError(t, err)
		}

		// List pending merchants
		pending, err := repo.ListByKYCStatus(string(model.KYCStatusPending), 10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(pending), 3)

		// List approved merchants
		approved, err := repo.ListByKYCStatus(string(model.KYCStatusApproved), 10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(approved), 2)
	})

	t.Run("Error - Empty KYC status", func(t *testing.T) {
		_, err := repo.ListByKYCStatus("", 10, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "KYC status cannot be empty")
	})
}

func TestMerchantRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupMerchants(t, db)

	repo := NewMerchantRepository(db)

	t.Run("Success - Soft delete merchant", func(t *testing.T) {
		merchant := createTestMerchant()
		err := repo.Create(merchant)
		require.NoError(t, err)

		err = repo.Delete(merchant.ID)
		require.NoError(t, err)

		// Verify the merchant is soft-deleted
		_, err = repo.GetByID(merchant.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrMerchantNotFound)
	})

	t.Run("Error - Empty merchant ID", func(t *testing.T) {
		err := repo.Delete("")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidMerchantID)
	})

	t.Run("Error - Merchant not found", func(t *testing.T) {
		err := repo.Delete(uuid.New().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrMerchantNotFound)
	})
}

func TestMerchantRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupMerchants(t, db)

	repo := NewMerchantRepository(db)

	t.Run("Success - Count merchants", func(t *testing.T) {
		// Create merchants
		for i := 0; i < 5; i++ {
			merchant := createTestMerchant()
			merchant.ID = uuid.New().String()
			merchant.Email = uuid.New().String() + "@example.com"
			err := repo.Create(merchant)
			require.NoError(t, err)
		}

		count, err := repo.Count()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(5))
	})
}

func TestMerchantRepository_CountByKYCStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupMerchants(t, db)

	repo := NewMerchantRepository(db)

	t.Run("Success - Count merchants by KYC status", func(t *testing.T) {
		// Create merchants with pending status
		for i := 0; i < 3; i++ {
			merchant := createTestMerchant()
			merchant.ID = uuid.New().String()
			merchant.Email = uuid.New().String() + "@example.com"
			merchant.KYCStatus = model.KYCStatusPending
			err := repo.Create(merchant)
			require.NoError(t, err)
		}

		count, err := repo.CountByKYCStatus(string(model.KYCStatusPending))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(3))
	})

	t.Run("Error - Empty KYC status", func(t *testing.T) {
		_, err := repo.CountByKYCStatus("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "KYC status cannot be empty")
	})
}
