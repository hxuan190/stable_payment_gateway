package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/trmlabs"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
)

// MockTRMLabsClient is a mock implementation of TRM Labs client for testing
type MockTRMLabsClient struct {
	mock.Mock
}

func (m *MockTRMLabsClient) ScreenAddress(ctx context.Context, address string, chain string) (*trmlabs.ScreeningResponse, error) {
	args := m.Called(ctx, address, chain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*trmlabs.ScreeningResponse), args.Error(1)
}

func (m *MockTRMLabsClient) BatchScreenAddresses(ctx context.Context, addresses []string, chain string) (map[string]*trmlabs.ScreeningResponse, error) {
	args := m.Called(ctx, addresses, chain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*trmlabs.ScreeningResponse), args.Error(1)
}

// MockAuditLogRepository is a mock implementation of audit log repository
type MockAuditLogRepository struct {
	mock.Mock
}

func (m *MockAuditLogRepository) Create(ctx context.Context, auditLog *model.AuditLog) error {
	args := m.Called(ctx, auditLog)
	return args.Error(0)
}

func (m *MockAuditLogRepository) List(ctx context.Context, filter repository.AuditLogFilter) ([]*model.AuditLog, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AuditLog), args.Error(1)
}

func (m *MockAuditLogRepository) GetByResourceID(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*model.AuditLog, error) {
	args := m.Called(ctx, resourceType, resourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AuditLog), args.Error(1)
}

func (m *MockAuditLogRepository) GetByActorID(ctx context.Context, actorType string, actorID uuid.UUID) ([]*model.AuditLog, error) {
	args := m.Called(ctx, actorType, actorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.AuditLog), args.Error(1)
}

func setupAMLServiceTest() (*amlServiceImpl, *MockTRMLabsClient, *MockAuditLogRepository) {
	trmClient := new(MockTRMLabsClient)
	auditRepo := new(MockAuditLogRepository)
	log := logger.NewLogger()

	service := &amlServiceImpl{
		trmClient:  trmClient,
		auditRepo:  auditRepo,
		paymentRepo: nil, // Not needed for most tests
		logger:     log,
		riskScoreThreshold: 80,
		autoRejectSanctioned: true,
	}

	return service, trmClient, auditRepo
}

func TestScreenWalletAddress_Success(t *testing.T) {
	service, trmClient, _ := setupAMLServiceTest()
	ctx := context.Background()

	expectedResponse := &trmlabs.ScreeningResponse{
		Address:      "SafeWallet123",
		Chain:        "solana",
		RiskScore:    10,
		IsSanctioned: false,
		Flags:        []string{},
		Details:      map[string]string{"reason": "No risk indicators"},
		ScreenedAt:   time.Now(),
	}

	trmClient.On("ScreenAddress", ctx, "SafeWallet123", "solana").Return(expectedResponse, nil)

	result, err := service.ScreenWalletAddress(ctx, "SafeWallet123", "solana")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "SafeWallet123", result.WalletAddress)
	assert.Equal(t, "solana", result.Chain)
	assert.Equal(t, 10, result.RiskScore)
	assert.False(t, result.IsSanctioned)

	trmClient.AssertExpectations(t)
}

func TestScreenWalletAddress_HighRisk(t *testing.T) {
	service, trmClient, _ := setupAMLServiceTest()
	ctx := context.Background()

	expectedResponse := &trmlabs.ScreeningResponse{
		Address:      "HighRiskWallet456",
		Chain:        "solana",
		RiskScore:    85,
		IsSanctioned: false,
		Flags:        []string{"mixer", "high-risk"},
		Details:      map[string]string{"reason": "Associated with mixing service"},
		ScreenedAt:   time.Now(),
	}

	trmClient.On("ScreenAddress", ctx, "HighRiskWallet456", "solana").Return(expectedResponse, nil)

	result, err := service.ScreenWalletAddress(ctx, "HighRiskWallet456", "solana")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 85, result.RiskScore)
	assert.False(t, result.IsSanctioned)
	assert.Contains(t, result.Flags, "mixer")

	trmClient.AssertExpectations(t)
}

func TestScreenWalletAddress_Sanctioned(t *testing.T) {
	service, trmClient, _ := setupAMLServiceTest()
	ctx := context.Background()

	expectedResponse := &trmlabs.ScreeningResponse{
		Address:      "SanctionedWallet789",
		Chain:        "solana",
		RiskScore:    100,
		IsSanctioned: true,
		Flags:        []string{"sanctions", "ofac"},
		Details:      map[string]string{"reason": "OFAC sanctions list"},
		ScreenedAt:   time.Now(),
	}

	trmClient.On("ScreenAddress", ctx, "SanctionedWallet789", "solana").Return(expectedResponse, nil)

	result, err := service.ScreenWalletAddress(ctx, "SanctionedWallet789", "solana")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 100, result.RiskScore)
	assert.True(t, result.IsSanctioned)
	assert.Contains(t, result.Flags, "sanctions")

	trmClient.AssertExpectations(t)
}

func TestRecordScreeningResult_Success(t *testing.T) {
	service, _, auditRepo := setupAMLServiceTest()
	ctx := context.Background()

	paymentID := uuid.New()
	result := &AMLScreeningResult{
		WalletAddress: "TestWallet123",
		Chain:         "solana",
		RiskScore:     10,
		IsSanctioned:  false,
		Flags:         []string{},
		Details:       map[string]string{"reason": "Safe"},
		ScreenedAt:    time.Now(),
	}

	auditRepo.On("Create", ctx, mock.AnythingOfType("*model.AuditLog")).Return(nil)

	err := service.RecordScreeningResult(ctx, paymentID, result)
	require.NoError(t, err)

	auditRepo.AssertExpectations(t)
}

func TestValidateTransaction_SafeAddress(t *testing.T) {
	service, trmClient, auditRepo := setupAMLServiceTest()
	ctx := context.Background()

	paymentID := uuid.New()
	fromAddress := "SafeWallet123"
	chain := "solana"
	amount := decimal.NewFromFloat(100)

	expectedResponse := &trmlabs.ScreeningResponse{
		Address:      fromAddress,
		Chain:        chain,
		RiskScore:    10,
		IsSanctioned: false,
		Flags:        []string{},
		Details:      map[string]string{},
		ScreenedAt:   time.Now(),
	}

	trmClient.On("ScreenAddress", ctx, fromAddress, chain).Return(expectedResponse, nil)
	auditRepo.On("Create", ctx, mock.AnythingOfType("*model.AuditLog")).Return(nil)

	err := service.ValidateTransaction(ctx, paymentID, fromAddress, chain, amount)
	require.NoError(t, err)

	trmClient.AssertExpectations(t)
	auditRepo.AssertExpectations(t)
}

func TestValidateTransaction_SanctionedAddress(t *testing.T) {
	service, trmClient, auditRepo := setupAMLServiceTest()
	ctx := context.Background()

	paymentID := uuid.New()
	fromAddress := "SanctionedWallet789"
	chain := "solana"
	amount := decimal.NewFromFloat(100)

	expectedResponse := &trmlabs.ScreeningResponse{
		Address:      fromAddress,
		Chain:        chain,
		RiskScore:    100,
		IsSanctioned: true,
		Flags:        []string{"sanctions", "ofac"},
		Details:      map[string]string{},
		ScreenedAt:   time.Now(),
	}

	trmClient.On("ScreenAddress", ctx, fromAddress, chain).Return(expectedResponse, nil)
	auditRepo.On("Create", ctx, mock.AnythingOfType("*model.AuditLog")).Return(nil)

	err := service.ValidateTransaction(ctx, paymentID, fromAddress, chain, amount)
	require.Error(t, err)
	assert.Equal(t, ErrSanctionedAddress, err)

	trmClient.AssertExpectations(t)
}

func TestValidateTransaction_HighRiskScore(t *testing.T) {
	service, trmClient, auditRepo := setupAMLServiceTest()
	ctx := context.Background()

	paymentID := uuid.New()
	fromAddress := "HighRiskWallet456"
	chain := "solana"
	amount := decimal.NewFromFloat(100)

	expectedResponse := &trmlabs.ScreeningResponse{
		Address:      fromAddress,
		Chain:        chain,
		RiskScore:    85, // Above threshold of 80
		IsSanctioned: false,
		Flags:        []string{"mixer"},
		Details:      map[string]string{},
		ScreenedAt:   time.Now(),
	}

	trmClient.On("ScreenAddress", ctx, fromAddress, chain).Return(expectedResponse, nil)
	auditRepo.On("Create", ctx, mock.AnythingOfType("*model.AuditLog")).Return(nil)

	err := service.ValidateTransaction(ctx, paymentID, fromAddress, chain, amount)
	require.Error(t, err)
	assert.Equal(t, ErrHighRiskAddress, err)

	trmClient.AssertExpectations(t)
}

func TestValidateTransaction_BlockedFlag(t *testing.T) {
	service, trmClient, auditRepo := setupAMLServiceTest()
	ctx := context.Background()

	paymentID := uuid.New()
	fromAddress := "TerroristWallet"
	chain := "solana"
	amount := decimal.NewFromFloat(100)

	expectedResponse := &trmlabs.ScreeningResponse{
		Address:      fromAddress,
		Chain:        chain,
		RiskScore:    50, // Below threshold
		IsSanctioned: false,
		Flags:        []string{"terrorist-financing"}, // Blocked flag
		Details:      map[string]string{},
		ScreenedAt:   time.Now(),
	}

	trmClient.On("ScreenAddress", ctx, fromAddress, chain).Return(expectedResponse, nil)
	auditRepo.On("Create", ctx, mock.AnythingOfType("*model.AuditLog")).Return(nil)

	err := service.ValidateTransaction(ctx, paymentID, fromAddress, chain, amount)
	require.Error(t, err)
	assert.Equal(t, ErrBlockedFlag, err)

	trmClient.AssertExpectations(t)
}

func TestIsBlockedFlag(t *testing.T) {
	tests := []struct {
		flag     string
		expected bool
	}{
		{"sanctions", true},
		{"ofac", true},
		{"terrorist-financing", true},
		{"child-exploitation", true},
		{"mixer", false},
		{"high-risk", false},
		{"darknet", false},
		{"normal", false},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			result := isBlockedFlag(tt.flag)
			assert.Equal(t, tt.expected, result)
		})
	}
}
