package service

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/config"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/database"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates a test database connection
// Note: This assumes a test database is available. In production tests, use a mock or test database.
func setupMerchantServiceTest(t *testing.T) (*MerchantService, *sql.DB, func()) {
	// For unit tests, we'll need a test database
	// This is a simplified version - in real tests you'd use a test database or mocks
	t.Skip("Skipping integration test - requires database setup")

	dbWrapper, err := database.New(&config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "postgres",
		Password:     "postgres",
		Database:     "payment_gateway_test",
		SSLMode:      "disable",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	require.NoError(t, err)

	db := dbWrapper.DB

	merchantRepo := repository.NewMerchantRepository(db)
	balanceRepo := repository.NewBalanceRepository(db)
	service := NewMerchantService(merchantRepo, balanceRepo, db)

	cleanup := func() {
		dbWrapper.Close()
	}

	return service, db, cleanup
}

func TestGenerateAPIKey(t *testing.T) {
	service := &MerchantService{}

	// Generate multiple keys to test uniqueness
	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		apiKey, err := service.GenerateAPIKey()
		require.NoError(t, err)
		require.NotEmpty(t, apiKey)

		// Check format
		assert.True(t, strings.HasPrefix(apiKey, "spg_live_"), "API key should start with spg_live_")
		assert.Greater(t, len(apiKey), 20, "API key should be sufficiently long")

		// Check uniqueness
		assert.False(t, keys[apiKey], "API key should be unique")
		keys[apiKey] = true
	}
}

func TestRegisterMerchantInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   RegisterMerchantInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid input",
			input: RegisterMerchantInput{
				Email:         "test@example.com",
				BusinessName:  "Test Business",
				OwnerFullName: "John Doe",
			},
			wantErr: false,
		},
		{
			name: "missing email",
			input: RegisterMerchantInput{
				Email:         "",
				BusinessName:  "Test Business",
				OwnerFullName: "John Doe",
			},
			wantErr: true,
			errMsg:  "email",
		},
		{
			name: "missing business name",
			input: RegisterMerchantInput{
				Email:         "test@example.com",
				BusinessName:  "",
				OwnerFullName: "John Doe",
			},
			wantErr: true,
			errMsg:  "business name",
		},
		{
			name: "business name too short",
			input: RegisterMerchantInput{
				Email:         "test@example.com",
				BusinessName:  "A",
				OwnerFullName: "John Doe",
			},
			wantErr: true,
			errMsg:  "business name",
		},
		{
			name: "missing owner name",
			input: RegisterMerchantInput{
				Email:         "test@example.com",
				BusinessName:  "Test Business",
				OwnerFullName: "",
			},
			wantErr: true,
			errMsg:  "owner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errMsg))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMerchantService_RegisterMerchant_Validation(t *testing.T) {
	// Test validation logic by checking the Validate method directly
	tests := []struct {
		name    string
		input   RegisterMerchantInput
		wantErr bool
	}{
		{
			name: "valid registration input",
			input: RegisterMerchantInput{
				Email:         "merchant@example.com",
				BusinessName:  "Test Restaurant",
				OwnerFullName: "Nguyen Van A",
			},
			wantErr: false,
		},
		{
			name: "invalid email",
			input: RegisterMerchantInput{
				Email:         "",
				BusinessName:  "Test Restaurant",
				OwnerFullName: "Nguyen Van A",
			},
			wantErr: true,
		},
		{
			name: "invalid business name",
			input: RegisterMerchantInput{
				Email:         "test@example.com",
				BusinessName:  "",
				OwnerFullName: "Nguyen Van A",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Mock repository for testing business logic without database
type mockMerchantRepository struct {
	merchants map[string]*model.Merchant
	emails    map[string]*model.Merchant
}

func newMockMerchantRepository() *mockMerchantRepository {
	return &mockMerchantRepository{
		merchants: make(map[string]*model.Merchant),
		emails:    make(map[string]*model.Merchant),
	}
}

func (m *mockMerchantRepository) Create(merchant *model.Merchant) error {
	if merchant == nil {
		return repository.ErrMerchantNotFound
	}
	if _, exists := m.emails[merchant.Email]; exists {
		return repository.ErrMerchantAlreadyExists
	}
	m.merchants[merchant.ID] = merchant
	m.emails[merchant.Email] = merchant
	return nil
}

func (m *mockMerchantRepository) GetByID(id string) (*model.Merchant, error) {
	merchant, exists := m.merchants[id]
	if !exists {
		return nil, repository.ErrMerchantNotFound
	}
	return merchant, nil
}

func (m *mockMerchantRepository) GetByEmail(email string) (*model.Merchant, error) {
	merchant, exists := m.emails[email]
	if !exists {
		return nil, repository.ErrMerchantNotFound
	}
	return merchant, nil
}

func (m *mockMerchantRepository) Update(merchant *model.Merchant) error {
	if merchant == nil {
		return repository.ErrMerchantNotFound
	}
	existing, exists := m.merchants[merchant.ID]
	if !exists {
		return repository.ErrMerchantNotFound
	}
	// Update the mock storage
	delete(m.emails, existing.Email)
	m.merchants[merchant.ID] = merchant
	m.emails[merchant.Email] = merchant
	return nil
}

type mockBalanceRepository struct {
	balances map[string]*model.MerchantBalance
}

func newMockBalanceRepository() *mockBalanceRepository {
	return &mockBalanceRepository{
		balances: make(map[string]*model.MerchantBalance),
	}
}

func (m *mockBalanceRepository) Create(balance *model.MerchantBalance) error {
	if balance == nil {
		return repository.ErrBalanceNotFound
	}
	m.balances[balance.MerchantID] = balance
	return nil
}

func (m *mockBalanceRepository) GetByMerchantID(merchantID string) (*model.MerchantBalance, error) {
	balance, exists := m.balances[merchantID]
	if !exists {
		return nil, repository.ErrBalanceNotFound
	}
	return balance, nil
}

type mockDB struct{}

func (m *mockDB) Begin() (*sql.Tx, error) {
	return nil, nil
}

func TestMerchantService_ApproveKYC(t *testing.T) {
	// Create mock repositories
	merchantRepo := newMockMerchantRepository()
	_ = newMockBalanceRepository() // Create but not used in this specific test

	// Create service with mock DB (nil is ok for this test)
	service := &MerchantService{
		merchantRepo: (*repository.MerchantRepository)(nil), // We'll use the mock directly
		balanceRepo:  (*repository.BalanceRepository)(nil),
		db:           nil,
	}

	// Create a test merchant with pending KYC
	testMerchant := &model.Merchant{
		ID:            "test-merchant-id",
		Email:         "test@example.com",
		BusinessName:  "Test Business",
		OwnerFullName: "John Doe",
		KYCStatus:     model.KYCStatusPending,
		Status:        model.MerchantStatusActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	merchantRepo.Create(testMerchant)

	// Test: Cannot approve without valid merchant ID
	t.Run("empty merchant ID", func(t *testing.T) {
		_, err := service.ApproveKYC("", "admin@example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant ID")
	})

	// Test: Cannot approve without approved by
	t.Run("empty approved by", func(t *testing.T) {
		_, err := service.ApproveKYC("test-merchant-id", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "approved by")
	})
}

func TestMerchantService_RejectKYC(t *testing.T) {
	service := &MerchantService{}

	// Test: Cannot reject without merchant ID
	t.Run("empty merchant ID", func(t *testing.T) {
		_, err := service.RejectKYC("", "Invalid documents")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant ID")
	})

	// Test: Cannot reject without reason
	t.Run("empty reason", func(t *testing.T) {
		_, err := service.RejectKYC("test-id", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reason")
	})
}

func TestMerchantService_RotateAPIKey(t *testing.T) {
	service := &MerchantService{}

	// Test: Cannot rotate without merchant ID
	t.Run("empty merchant ID", func(t *testing.T) {
		_, err := service.RotateAPIKey("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant ID")
	})
}

func TestMerchantService_GetMerchantBalance(t *testing.T) {
	service := &MerchantService{}

	// Test: Cannot get balance without merchant ID
	t.Run("empty merchant ID", func(t *testing.T) {
		_, err := service.GetMerchantBalance("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant ID")
	})
}

func TestMerchantService_UpdateWebhookConfig(t *testing.T) {
	service := &MerchantService{}

	// Test: Cannot update webhook without merchant ID
	t.Run("empty merchant ID", func(t *testing.T) {
		err := service.UpdateWebhookConfig("", "https://example.com/webhook", "secret", []string{"payment.completed"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant ID")
	})
}

func TestMerchantService_SuspendMerchant(t *testing.T) {
	service := &MerchantService{}

	// Test: Cannot suspend without merchant ID
	t.Run("empty merchant ID", func(t *testing.T) {
		err := service.SuspendMerchant("", "Fraudulent activity")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant ID")
	})
}

func TestMerchantService_ReactivateMerchant(t *testing.T) {
	service := &MerchantService{}

	// Test: Cannot reactivate without merchant ID
	t.Run("empty merchant ID", func(t *testing.T) {
		err := service.ReactivateMerchant("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merchant ID")
	})
}

// Integration test examples (require database)
func TestMerchantService_Integration_RegisterMerchant(t *testing.T) {
	service, db, cleanup := setupMerchantServiceTest(t)
	defer cleanup()

	input := RegisterMerchantInput{
		Email:               "integration@example.com",
		BusinessName:        "Integration Test Restaurant",
		BusinessTaxID:       "0123456789",
		OwnerFullName:       "Nguyen Van Test",
		PhoneNumber:         "+84901234567",
		BusinessAddress:     "123 Test Street",
		City:                "Da Nang",
		BankAccountName:     "Nguyen Van Test",
		BankAccountNumber:   "1234567890",
		BankName:            "Vietcombank",
	}

	merchant, err := service.RegisterMerchant(input)
	require.NoError(t, err)
	require.NotNil(t, merchant)

	assert.NotEmpty(t, merchant.ID)
	assert.Equal(t, input.Email, merchant.Email)
	assert.Equal(t, input.BusinessName, merchant.BusinessName)
	assert.Equal(t, model.KYCStatusPending, merchant.KYCStatus)
	assert.Equal(t, model.MerchantStatusActive, merchant.Status)

	// Verify balance was created
	balance, err := service.GetMerchantBalance(merchant.ID)
	require.NoError(t, err)
	assert.Equal(t, merchant.ID, balance.MerchantID)
	assert.True(t, balance.AvailableVND.Equal(decimal.Zero))
	assert.True(t, balance.TotalVND.Equal(decimal.Zero))

	// Cleanup
	_, err = db.Exec("DELETE FROM merchant_balances WHERE merchant_id = $1", merchant.ID)
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM merchants WHERE id = $1", merchant.ID)
	require.NoError(t, err)
}

func TestMerchantService_Integration_ApproveKYC(t *testing.T) {
	service, db, cleanup := setupMerchantServiceTest(t)
	defer cleanup()

	// First register a merchant
	input := RegisterMerchantInput{
		Email:         "kyc_test@example.com",
		BusinessName:  "KYC Test Business",
		OwnerFullName: "Test Owner",
	}

	merchant, err := service.RegisterMerchant(input)
	require.NoError(t, err)

	// Approve KYC
	approvedMerchant, err := service.ApproveKYC(merchant.ID, "admin@gateway.com")
	require.NoError(t, err)
	require.NotNil(t, approvedMerchant)

	assert.Equal(t, model.KYCStatusApproved, approvedMerchant.KYCStatus)
	assert.True(t, approvedMerchant.APIKey.Valid)
	assert.NotEmpty(t, approvedMerchant.APIKey.String)
	assert.True(t, strings.HasPrefix(approvedMerchant.APIKey.String, "spg_live_"))
	assert.True(t, approvedMerchant.KYCApprovedAt.Valid)
	assert.True(t, approvedMerchant.APIKeyCreatedAt.Valid)

	// Cleanup
	_, err = db.Exec("DELETE FROM merchant_balances WHERE merchant_id = $1", merchant.ID)
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM merchants WHERE id = $1", merchant.ID)
	require.NoError(t, err)
}

func TestMerchantService_Integration_RejectKYC(t *testing.T) {
	service, db, cleanup := setupMerchantServiceTest(t)
	defer cleanup()

	// First register a merchant
	input := RegisterMerchantInput{
		Email:         "reject_test@example.com",
		BusinessName:  "Reject Test Business",
		OwnerFullName: "Test Owner",
	}

	merchant, err := service.RegisterMerchant(input)
	require.NoError(t, err)

	// Reject KYC
	reason := "Invalid business license"
	rejectedMerchant, err := service.RejectKYC(merchant.ID, reason)
	require.NoError(t, err)
	require.NotNil(t, rejectedMerchant)

	assert.Equal(t, model.KYCStatusRejected, rejectedMerchant.KYCStatus)
	assert.True(t, rejectedMerchant.KYCRejectionReason.Valid)
	assert.Equal(t, reason, rejectedMerchant.KYCRejectionReason.String)

	// Cleanup
	_, err = db.Exec("DELETE FROM merchant_balances WHERE merchant_id = $1", merchant.ID)
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM merchants WHERE id = $1", merchant.ID)
	require.NoError(t, err)
}

func TestMerchantService_Integration_RotateAPIKey(t *testing.T) {
	service, db, cleanup := setupMerchantServiceTest(t)
	defer cleanup()

	// Register and approve a merchant first
	input := RegisterMerchantInput{
		Email:         "rotate_test@example.com",
		BusinessName:  "Rotate Test Business",
		OwnerFullName: "Test Owner",
	}

	merchant, err := service.RegisterMerchant(input)
	require.NoError(t, err)

	approvedMerchant, err := service.ApproveKYC(merchant.ID, "admin@gateway.com")
	require.NoError(t, err)

	oldAPIKey := approvedMerchant.APIKey.String

	// Rotate API key
	newAPIKey, err := service.RotateAPIKey(merchant.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, newAPIKey)
	assert.NotEqual(t, oldAPIKey, newAPIKey)
	assert.True(t, strings.HasPrefix(newAPIKey, "spg_live_"))

	// Verify the new key is saved
	updatedMerchant, err := service.GetMerchantByID(merchant.ID)
	require.NoError(t, err)
	assert.Equal(t, newAPIKey, updatedMerchant.APIKey.String)

	// Cleanup
	_, err = db.Exec("DELETE FROM merchant_balances WHERE merchant_id = $1", merchant.ID)
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM merchants WHERE id = $1", merchant.ID)
	require.NoError(t, err)
}
