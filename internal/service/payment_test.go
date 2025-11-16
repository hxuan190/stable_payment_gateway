package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
)

// Mock repositories
type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) Create(payment *model.Payment) error {
	args := m.Called(payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) GetByID(id string) (*model.Payment, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payment), args.Error(1)
}

func (m *MockPaymentRepository) GetByTxHash(txHash string) (*model.Payment, error) {
	args := m.Called(txHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payment), args.Error(1)
}

func (m *MockPaymentRepository) GetByPaymentReference(reference string) (*model.Payment, error) {
	args := m.Called(reference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payment), args.Error(1)
}

func (m *MockPaymentRepository) UpdateStatus(id string, status model.PaymentStatus) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockPaymentRepository) Update(payment *model.Payment) error {
	args := m.Called(payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) ListByMerchant(merchantID string, limit, offset int) ([]*model.Payment, error) {
	args := m.Called(merchantID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Payment), args.Error(1)
}

func (m *MockPaymentRepository) GetExpiredPayments() ([]*model.Payment, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Payment), args.Error(1)
}

type MockMerchantRepository struct {
	mock.Mock
}

func (m *MockMerchantRepository) GetByID(id string) (*model.Merchant, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) GetByEmail(email string) (*model.Merchant, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) GetByAPIKey(apiKey string) (*model.Merchant, error) {
	args := m.Called(apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) Create(merchant *model.Merchant) error {
	args := m.Called(merchant)
	return args.Error(0)
}

func (m *MockMerchantRepository) Update(merchant *model.Merchant) error {
	args := m.Called(merchant)
	return args.Error(0)
}

func (m *MockMerchantRepository) UpdateKYCStatus(id string, status model.KYCStatus) error {
	args := m.Called(id, status)
	return args.Error(0)
}

type MockExchangeRateService struct {
	mock.Mock
}

func (m *MockExchangeRateService) GetUSDTToVND(ctx context.Context) (decimal.Decimal, error) {
	args := m.Called(ctx)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func (m *MockExchangeRateService) GetUSDCToVND(ctx context.Context) (decimal.Decimal, error) {
	args := m.Called(ctx)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

// Test helpers

func setupPaymentServiceTest() (*PaymentService, *MockPaymentRepository, *MockMerchantRepository, *MockExchangeRateService) {
	paymentRepo := new(MockPaymentRepository)
	merchantRepo := new(MockMerchantRepository)
	exchangeRateService := new(MockExchangeRateService)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Suppress logs in tests

	service := NewPaymentService(
		paymentRepo,
		merchantRepo,
		exchangeRateService,
		PaymentServiceConfig{
			DefaultChain:    model.ChainSolana,
			DefaultCurrency: "USDT",
			WalletAddress:   "TestWalletAddress123",
			FeePercentage:   0.01,
			ExpiryMinutes:   30,
		},
		logger,
	)

	return service, paymentRepo, merchantRepo, exchangeRateService
}

func createTestMerchant() *model.Merchant {
	return &model.Merchant{
		ID:            "merchant-123",
		Email:         "test@example.com",
		BusinessName:  "Test Business",
		OwnerFullName: "John Doe",
		KYCStatus:     model.KYCStatusApproved,
		Status:        model.MerchantStatusActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// Tests for CreatePayment

func TestCreatePayment_Success(t *testing.T) {
	service, paymentRepo, merchantRepo, exchangeRateService := setupPaymentServiceTest()
	ctx := context.Background()

	merchant := createTestMerchant()
	exchangeRate := decimal.NewFromInt(25000) // 1 USDT = 25,000 VND

	merchantRepo.On("GetByID", merchant.ID).Return(merchant, nil)
	exchangeRateService.On("GetUSDTToVND", ctx).Return(exchangeRate, nil)
	paymentRepo.On("Create", mock.AnythingOfType("*model.Payment")).Return(nil)

	req := CreatePaymentRequest{
		MerchantID:  merchant.ID,
		AmountVND:   decimal.NewFromInt(2500000), // 2.5M VND
		Currency:    "USDT",
		Chain:       model.ChainSolana,
		OrderID:     "order-123",
		Description: "Test payment",
		CallbackURL: "https://example.com/webhook",
	}

	payment, err := service.CreatePayment(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, payment)
	assert.Equal(t, merchant.ID, payment.MerchantID)
	assert.Equal(t, req.AmountVND, payment.AmountVND)
	assert.Equal(t, "USDT", payment.Currency)
	assert.Equal(t, model.ChainSolana, payment.Chain)
	assert.Equal(t, exchangeRate, payment.ExchangeRate)
	assert.Equal(t, model.PaymentStatusCreated, payment.Status)
	assert.NotEmpty(t, payment.ID)
	assert.NotEmpty(t, payment.PaymentReference)
	assert.Equal(t, "TestWalletAddress123", payment.DestinationWallet)

	// Check crypto amount calculation: 2,500,000 / 25,000 = 100 USDT
	expectedCrypto := decimal.NewFromInt(100)
	assert.True(t, payment.AmountCrypto.Equal(expectedCrypto), "Expected %s, got %s", expectedCrypto, payment.AmountCrypto)

	// Check fee calculation: 2,500,000 * 0.01 = 25,000 VND
	expectedFee := decimal.NewFromInt(25000)
	assert.True(t, payment.FeeVND.Equal(expectedFee), "Expected fee %s, got %s", expectedFee, payment.FeeVND)

	// Check net amount: 2,500,000 - 25,000 = 2,475,000 VND
	expectedNet := decimal.NewFromInt(2475000)
	assert.True(t, payment.NetAmountVND.Equal(expectedNet), "Expected net %s, got %s", expectedNet, payment.NetAmountVND)

	merchantRepo.AssertExpectations(t)
	exchangeRateService.AssertExpectations(t)
	paymentRepo.AssertExpectations(t)
}

func TestCreatePayment_MerchantNotFound(t *testing.T) {
	service, _, merchantRepo, _ := setupPaymentServiceTest()
	ctx := context.Background()

	merchantRepo.On("GetByID", "invalid-merchant").Return(nil, repository.ErrMerchantNotFound)

	req := CreatePaymentRequest{
		MerchantID: "invalid-merchant",
		AmountVND:  decimal.NewFromInt(1000000),
	}

	payment, err := service.CreatePayment(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.True(t, errors.Is(err, ErrMerchantNotFound))

	merchantRepo.AssertExpectations(t)
}

func TestCreatePayment_MerchantNotApproved(t *testing.T) {
	service, _, merchantRepo, _ := setupPaymentServiceTest()
	ctx := context.Background()

	merchant := createTestMerchant()
	merchant.KYCStatus = model.KYCStatusPending // Not approved

	merchantRepo.On("GetByID", merchant.ID).Return(merchant, nil)

	req := CreatePaymentRequest{
		MerchantID: merchant.ID,
		AmountVND:  decimal.NewFromInt(1000000),
	}

	payment, err := service.CreatePayment(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.True(t, errors.Is(err, ErrMerchantNotApproved))

	merchantRepo.AssertExpectations(t)
}

func TestCreatePayment_InvalidAmount(t *testing.T) {
	service, _, merchantRepo, _ := setupPaymentServiceTest()
	ctx := context.Background()

	merchant := createTestMerchant()
	merchantRepo.On("GetByID", merchant.ID).Return(merchant, nil)

	tests := []struct {
		name      string
		amountVND decimal.Decimal
	}{
		{"zero amount", decimal.Zero},
		{"negative amount", decimal.NewFromInt(-1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := CreatePaymentRequest{
				MerchantID: merchant.ID,
				AmountVND:  tt.amountVND,
			}

			payment, err := service.CreatePayment(ctx, req)

			assert.Error(t, err)
			assert.Nil(t, payment)
			assert.True(t, errors.Is(err, ErrInvalidAmount))
		})
	}

	merchantRepo.AssertExpectations(t)
}

func TestCreatePayment_InvalidChainCurrency(t *testing.T) {
	service, _, merchantRepo, _ := setupPaymentServiceTest()
	ctx := context.Background()

	merchant := createTestMerchant()
	merchantRepo.On("GetByID", merchant.ID).Return(merchant, nil)

	req := CreatePaymentRequest{
		MerchantID: merchant.ID,
		AmountVND:  decimal.NewFromInt(1000000),
		Currency:   "BTC", // Invalid currency
		Chain:      model.ChainSolana,
	}

	payment, err := service.CreatePayment(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Contains(t, err.Error(), "invalid or unsupported")

	merchantRepo.AssertExpectations(t)
}

// Tests for ConfirmPayment

func TestConfirmPayment_Success(t *testing.T) {
	service, paymentRepo, _, _ := setupPaymentServiceTest()
	ctx := context.Background()

	payment := &model.Payment{
		ID:           "payment-123",
		MerchantID:   "merchant-123",
		AmountVND:    decimal.NewFromInt(2500000),
		AmountCrypto: decimal.NewFromInt(100),
		Currency:     "USDT",
		Chain:        model.ChainSolana,
		Status:       model.PaymentStatusCreated,
		ExpiresAt:    time.Now().Add(30 * time.Minute),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	paymentRepo.On("GetByID", payment.ID).Return(payment, nil)
	paymentRepo.On("Update", mock.AnythingOfType("*model.Payment")).Return(nil)

	req := ConfirmPaymentRequest{
		PaymentID:     payment.ID,
		TxHash:        "tx-hash-123",
		FromAddress:   "sender-address",
		ActualAmount:  decimal.NewFromInt(100), // Exact match
		Confirmations: 1,
	}

	confirmedPayment, err := service.ConfirmPayment(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, confirmedPayment)
	assert.Equal(t, model.PaymentStatusCompleted, confirmedPayment.Status)
	assert.Equal(t, "tx-hash-123", confirmedPayment.TxHash.String)
	assert.True(t, confirmedPayment.PaidAt.Valid)
	assert.True(t, confirmedPayment.ConfirmedAt.Valid)

	paymentRepo.AssertExpectations(t)
}

func TestConfirmPayment_AmountMismatch(t *testing.T) {
	service, paymentRepo, _, _ := setupPaymentServiceTest()
	ctx := context.Background()

	payment := &model.Payment{
		ID:           "payment-123",
		MerchantID:   "merchant-123",
		AmountVND:    decimal.NewFromInt(2500000),
		AmountCrypto: decimal.NewFromInt(100),
		Currency:     "USDT",
		Status:       model.PaymentStatusCreated,
		ExpiresAt:    time.Now().Add(30 * time.Minute),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	paymentRepo.On("GetByID", payment.ID).Return(payment, nil)

	req := ConfirmPaymentRequest{
		PaymentID:     payment.ID,
		TxHash:        "tx-hash-123",
		FromAddress:   "sender-address",
		ActualAmount:  decimal.NewFromInt(50), // Wrong amount
		Confirmations: 1,
	}

	confirmedPayment, err := service.ConfirmPayment(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, confirmedPayment)
	assert.True(t, errors.Is(err, ErrAmountMismatch))

	paymentRepo.AssertExpectations(t)
}

func TestConfirmPayment_PaymentExpired(t *testing.T) {
	service, paymentRepo, _, _ := setupPaymentServiceTest()
	ctx := context.Background()

	payment := &model.Payment{
		ID:           "payment-123",
		MerchantID:   "merchant-123",
		AmountVND:    decimal.NewFromInt(2500000),
		AmountCrypto: decimal.NewFromInt(100),
		Status:       model.PaymentStatusCreated,
		ExpiresAt:    time.Now().Add(-1 * time.Minute), // Expired
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	paymentRepo.On("GetByID", payment.ID).Return(payment, nil)

	req := ConfirmPaymentRequest{
		PaymentID:     payment.ID,
		TxHash:        "tx-hash-123",
		ActualAmount:  decimal.NewFromInt(100),
		Confirmations: 1,
	}

	confirmedPayment, err := service.ConfirmPayment(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, confirmedPayment)
	assert.True(t, errors.Is(err, ErrPaymentExpired))

	paymentRepo.AssertExpectations(t)
}

func TestConfirmPayment_AlreadyCompleted(t *testing.T) {
	service, paymentRepo, _, _ := setupPaymentServiceTest()
	ctx := context.Background()

	payment := &model.Payment{
		ID:           "payment-123",
		Status:       model.PaymentStatusCompleted, // Already completed
		AmountCrypto: decimal.NewFromInt(100),
		ExpiresAt:    time.Now().Add(30 * time.Minute),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	paymentRepo.On("GetByID", payment.ID).Return(payment, nil)

	req := ConfirmPaymentRequest{
		PaymentID:     payment.ID,
		TxHash:        "tx-hash-123",
		ActualAmount:  decimal.NewFromInt(100),
		Confirmations: 1,
	}

	confirmedPayment, err := service.ConfirmPayment(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, confirmedPayment)
	assert.True(t, errors.Is(err, ErrPaymentAlreadyCompleted))

	paymentRepo.AssertExpectations(t)
}

// Tests for ExpirePayment

func TestExpirePayment_Success(t *testing.T) {
	service, paymentRepo, _, _ := setupPaymentServiceTest()
	ctx := context.Background()

	payment := &model.Payment{
		ID:        "payment-123",
		Status:    model.PaymentStatusCreated,
		ExpiresAt: time.Now().Add(-1 * time.Minute),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	paymentRepo.On("GetByID", payment.ID).Return(payment, nil)
	paymentRepo.On("UpdateStatus", payment.ID, model.PaymentStatusExpired).Return(nil)

	err := service.ExpirePayment(ctx, payment.ID)

	assert.NoError(t, err)
	paymentRepo.AssertExpectations(t)
}

func TestExpirePayment_InvalidState(t *testing.T) {
	service, paymentRepo, _, _ := setupPaymentServiceTest()
	ctx := context.Background()

	payment := &model.Payment{
		ID:        "payment-123",
		Status:    model.PaymentStatusCompleted, // Cannot expire completed payment
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	paymentRepo.On("GetByID", payment.ID).Return(payment, nil)

	err := service.ExpirePayment(ctx, payment.ID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidPaymentState))
	paymentRepo.AssertExpectations(t)
}

// Tests for FailPayment

func TestFailPayment_Success(t *testing.T) {
	service, paymentRepo, _, _ := setupPaymentServiceTest()
	ctx := context.Background()

	payment := &model.Payment{
		ID:        "payment-123",
		Status:    model.PaymentStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	paymentRepo.On("GetByID", payment.ID).Return(payment, nil)
	paymentRepo.On("Update", mock.AnythingOfType("*model.Payment")).Return(nil)

	err := service.FailPayment(ctx, payment.ID, "insufficient confirmations")

	assert.NoError(t, err)
	paymentRepo.AssertExpectations(t)
}

func TestFailPayment_AlreadyCompleted(t *testing.T) {
	service, paymentRepo, _, _ := setupPaymentServiceTest()
	ctx := context.Background()

	payment := &model.Payment{
		ID:        "payment-123",
		Status:    model.PaymentStatusCompleted,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	paymentRepo.On("GetByID", payment.ID).Return(payment, nil)

	err := service.FailPayment(ctx, payment.ID, "test reason")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrPaymentAlreadyCompleted))
	paymentRepo.AssertExpectations(t)
}

// Tests for ValidatePayment

func TestValidatePayment_Success(t *testing.T) {
	service, paymentRepo, _, _ := setupPaymentServiceTest()
	ctx := context.Background()

	payment := &model.Payment{
		ID:        "payment-123",
		Status:    model.PaymentStatusCreated,
		ExpiresAt: time.Now().Add(30 * time.Minute),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	paymentRepo.On("GetByID", payment.ID).Return(payment, nil)

	err := service.ValidatePayment(ctx, payment.ID)

	assert.NoError(t, err)
	paymentRepo.AssertExpectations(t)
}

func TestValidatePayment_Expired(t *testing.T) {
	service, paymentRepo, _, _ := setupPaymentServiceTest()
	ctx := context.Background()

	payment := &model.Payment{
		ID:        "payment-123",
		Status:    model.PaymentStatusCreated,
		ExpiresAt: time.Now().Add(-1 * time.Minute), // Expired
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	paymentRepo.On("GetByID", payment.ID).Return(payment, nil)

	err := service.ValidatePayment(ctx, payment.ID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrPaymentExpired))
	paymentRepo.AssertExpectations(t)
}

func TestValidatePayment_AlreadyCompleted(t *testing.T) {
	service, paymentRepo, _, _ := setupPaymentServiceTest()
	ctx := context.Background()

	payment := &model.Payment{
		ID:        "payment-123",
		Status:    model.PaymentStatusCompleted,
		ExpiresAt: time.Now().Add(30 * time.Minute),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	paymentRepo.On("GetByID", payment.ID).Return(payment, nil)

	err := service.ValidatePayment(ctx, payment.ID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrPaymentAlreadyCompleted))
	paymentRepo.AssertExpectations(t)
}

// Test helper functions

func TestGeneratePaymentReference(t *testing.T) {
	service, _, _, _ := setupPaymentServiceTest()

	// Generate multiple references to test uniqueness
	references := make(map[string]bool)
	for i := 0; i < 100; i++ {
		ref, err := service.generatePaymentReference()
		require.NoError(t, err)
		require.NotEmpty(t, ref)

		// Check length (16 hex characters)
		assert.Equal(t, PaymentReferenceLength, len(ref), "Reference should be 16 characters")

		// Check uniqueness
		assert.False(t, references[ref], "Reference should be unique")
		references[ref] = true
	}
}

func TestValidateChainCurrency(t *testing.T) {
	service, _, _, _ := setupPaymentServiceTest()

	tests := []struct {
		name     string
		chain    model.Chain
		currency string
		wantErr  bool
	}{
		{"valid solana usdt", model.ChainSolana, "USDT", false},
		{"valid solana usdc", model.ChainSolana, "USDC", false},
		{"valid bsc usdt", model.ChainBSC, "USDT", false},
		{"valid bsc busd", model.ChainBSC, "BUSD", false},
		{"invalid solana busd", model.ChainSolana, "BUSD", true},
		{"invalid bsc usdc", model.ChainBSC, "USDC", true},
		{"invalid currency", model.ChainSolana, "BTC", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateChainCurrency(tt.chain, tt.currency)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
