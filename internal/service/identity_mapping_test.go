package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWalletIdentityMappingRepository is a mock for WalletIdentityMappingRepository
type MockWalletIdentityMappingRepository struct {
	mock.Mock
}

func (m *MockWalletIdentityMappingRepository) Create(mapping *model.WalletIdentityMapping) error {
	args := m.Called(mapping)
	return args.Error(0)
}

func (m *MockWalletIdentityMappingRepository) GetByID(id uuid.UUID) (*model.WalletIdentityMapping, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WalletIdentityMapping), args.Error(1)
}

func (m *MockWalletIdentityMappingRepository) GetByWalletAddress(walletAddress string, blockchain model.Chain) (*model.WalletIdentityMapping, error) {
	args := m.Called(walletAddress, blockchain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WalletIdentityMapping), args.Error(1)
}

func (m *MockWalletIdentityMappingRepository) GetByUserID(userID uuid.UUID) ([]*model.WalletIdentityMapping, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.WalletIdentityMapping), args.Error(1)
}

func (m *MockWalletIdentityMappingRepository) Update(mapping *model.WalletIdentityMapping) error {
	args := m.Called(mapping)
	return args.Error(0)
}

func (m *MockWalletIdentityMappingRepository) UpdateActivity(id uuid.UUID, amountUSD decimal.Decimal) error {
	args := m.Called(id, amountUSD)
	return args.Error(0)
}

func (m *MockWalletIdentityMappingRepository) Flag(id uuid.UUID, reason string) error {
	args := m.Called(id, reason)
	return args.Error(0)
}

func (m *MockWalletIdentityMappingRepository) Unflag(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockWalletIdentityMappingRepository) ListActive(limit, offset int) ([]*model.WalletIdentityMapping, error) {
	args := m.Called(limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.WalletIdentityMapping), args.Error(1)
}

func (m *MockWalletIdentityMappingRepository) ListFlagged(limit, offset int) ([]*model.WalletIdentityMapping, error) {
	args := m.Called(limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.WalletIdentityMapping), args.Error(1)
}

func (m *MockWalletIdentityMappingRepository) Count() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// MockUserRepository is a mock for UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(id uuid.UUID) (*model.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*model.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) Update(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateKYCStatus(id uuid.UUID, status model.KYCStatus, rejectionReason *string) error {
	args := m.Called(id, status, rejectionReason)
	return args.Error(0)
}

func (m *MockUserRepository) IncrementPaymentStats(id uuid.UUID, amountUSD decimal.Decimal) error {
	args := m.Called(id, amountUSD)
	return args.Error(0)
}

func (m *MockUserRepository) Count() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// MockRedisCache is a mock for RedisCache
type MockRedisCache struct {
	mock.Mock
}

func (m *MockRedisCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockRedisCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockRedisCache) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRedisCache) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockKYCProvider is a mock for KYCProvider
type MockKYCProvider struct {
	mock.Mock
}

func (m *MockKYCProvider) CreateApplicant(ctx context.Context, fullName, email string) (string, error) {
	args := m.Called(ctx, fullName, email)
	return args.String(0), args.Error(1)
}

func (m *MockKYCProvider) GetApplicantStatus(ctx context.Context, applicantID string) (model.KYCStatus, error) {
	args := m.Called(ctx, applicantID)
	return args.Get(0).(model.KYCStatus), args.Error(1)
}

func (m *MockKYCProvider) VerifyFaceLiveness(ctx context.Context, applicantID string) (bool, float64, error) {
	args := m.Called(ctx, applicantID)
	return args.Bool(0), args.Get(1).(float64), args.Error(2)
}

func (m *MockKYCProvider) GetApplicantData(ctx context.Context, applicantID string) (map[string]interface{}, error) {
	args := m.Called(ctx, applicantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// Test helper functions
func createTestUser() *model.User {
	return &model.User{
		ID:                 uuid.New(),
		FullName:           "John Doe",
		Email:              sql.NullString{String: "john@example.com", Valid: true},
		KYCStatus:          model.KYCStatusApproved,
		FaceLivenessPassed: true,
		IsSanctioned:       false,
		RiskLevel:          model.RiskLevelLow,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}

func createTestWalletMapping(userID uuid.UUID, walletAddress string, blockchain model.Chain) *model.WalletIdentityMapping {
	return &model.WalletIdentityMapping{
		ID:                uuid.New(),
		WalletAddress:     walletAddress,
		Blockchain:        blockchain,
		WalletAddressHash: model.ComputeWalletHash(walletAddress),
		UserID:            userID,
		KYCStatus:         model.KYCStatusApproved,
		FirstSeenAt:       time.Now(),
		LastSeenAt:        time.Now(),
		PaymentCount:      5,
		TotalVolumeUSD:    decimal.NewFromFloat(1000.00),
		IsFlagged:         false,
		Metadata:          make(map[string]interface{}),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// Test RecognizeWallet - Cache Hit
func TestRecognizeWallet_CacheHit(t *testing.T) {
	// Setup mocks
	mockWalletRepo := new(MockWalletIdentityMappingRepository)
	mockUserRepo := new(MockUserRepository)
	mockCache := new(MockRedisCache)
	mockKYC := new(MockKYCProvider)

	service := NewIdentityMappingService(mockWalletRepo, mockUserRepo, mockCache, mockKYC)

	ctx := context.Background()
	walletAddress := "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU"
	blockchain := model.ChainSolana

	// Create test data
	user := createTestUser()
	mapping := createTestWalletMapping(user.ID, walletAddress, blockchain)

	// Mock cache hit
	cachedData, _ := json.Marshal(mapping)
	mockCache.On("Get", ctx, "wallet_identity:solana:"+walletAddress).Return(string(cachedData), nil)
	mockUserRepo.On("GetByID", user.ID).Return(user, nil)

	// Execute
	resultMapping, resultUser, err := service.RecognizeWallet(ctx, walletAddress, blockchain)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resultMapping)
	assert.NotNil(t, resultUser)
	assert.Equal(t, mapping.WalletAddress, resultMapping.WalletAddress)
	assert.Equal(t, user.ID, resultUser.ID)

	mockCache.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockWalletRepo.AssertNotCalled(t, "GetByWalletAddress") // DB should not be called
}

// Test RecognizeWallet - Cache Miss, DB Hit
func TestRecognizeWallet_CacheMiss_DBHit(t *testing.T) {
	// Setup mocks
	mockWalletRepo := new(MockWalletIdentityMappingRepository)
	mockUserRepo := new(MockUserRepository)
	mockCache := new(MockRedisCache)
	mockKYC := new(MockKYCProvider)

	service := NewIdentityMappingService(mockWalletRepo, mockUserRepo, mockCache, mockKYC)

	ctx := context.Background()
	walletAddress := "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU"
	blockchain := model.ChainSolana

	// Create test data
	user := createTestUser()
	mapping := createTestWalletMapping(user.ID, walletAddress, blockchain)

	// Mock cache miss
	mockCache.On("Get", ctx, "wallet_identity:solana:"+walletAddress).Return("", errors.New("key not found"))

	// Mock DB hit
	mockWalletRepo.On("GetByWalletAddress", walletAddress, blockchain).Return(mapping, nil)
	mockUserRepo.On("GetByID", user.ID).Return(user, nil)

	// Mock cache set (will be called after DB lookup)
	mockCache.On("Set", ctx, "wallet_identity:solana:"+walletAddress, mock.Anything, 7*24*time.Hour).Return(nil)

	// Execute
	resultMapping, resultUser, err := service.RecognizeWallet(ctx, walletAddress, blockchain)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, resultMapping)
	assert.NotNil(t, resultUser)
	assert.Equal(t, mapping.WalletAddress, resultMapping.WalletAddress)
	assert.Equal(t, user.ID, resultUser.ID)

	mockCache.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockWalletRepo.AssertExpectations(t)
}

// Test RecognizeWallet - Wallet Not Found
func TestRecognizeWallet_WalletNotFound(t *testing.T) {
	// Setup mocks
	mockWalletRepo := new(MockWalletIdentityMappingRepository)
	mockUserRepo := new(MockUserRepository)
	mockCache := new(MockRedisCache)
	mockKYC := new(MockKYCProvider)

	service := NewIdentityMappingService(mockWalletRepo, mockUserRepo, mockCache, mockKYC)

	ctx := context.Background()
	walletAddress := "NewWalletAddress123"
	blockchain := model.ChainSolana

	// Mock cache miss
	mockCache.On("Get", ctx, "wallet_identity:solana:"+walletAddress).Return("", errors.New("key not found"))

	// Mock DB miss
	mockWalletRepo.On("GetByWalletAddress", walletAddress, blockchain).Return(nil, errors.New("wallet identity mapping not found"))

	// Execute
	resultMapping, resultUser, err := service.RecognizeWallet(ctx, walletAddress, blockchain)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrWalletNotRecognized)
	assert.Nil(t, resultMapping)
	assert.Nil(t, resultUser)

	mockCache.AssertExpectations(t)
	mockWalletRepo.AssertExpectations(t)
	mockUserRepo.AssertNotCalled(t, "GetByID")
}

// Test RecognizeWallet - User Sanctioned
func TestRecognizeWallet_UserSanctioned(t *testing.T) {
	// Setup mocks
	mockWalletRepo := new(MockWalletIdentityMappingRepository)
	mockUserRepo := new(MockUserRepository)
	mockCache := new(MockRedisCache)
	mockKYC := new(MockKYCProvider)

	service := NewIdentityMappingService(mockWalletRepo, mockUserRepo, mockCache, mockKYC)

	ctx := context.Background()
	walletAddress := "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU"
	blockchain := model.ChainSolana

	// Create test data with sanctioned user
	user := createTestUser()
	user.IsSanctioned = true
	mapping := createTestWalletMapping(user.ID, walletAddress, blockchain)

	// Mock cache miss
	mockCache.On("Get", ctx, "wallet_identity:solana:"+walletAddress).Return("", errors.New("key not found"))

	// Mock DB hit
	mockWalletRepo.On("GetByWalletAddress", walletAddress, blockchain).Return(mapping, nil)
	mockUserRepo.On("GetByID", user.ID).Return(user, nil)

	// Execute
	resultMapping, resultUser, err := service.RecognizeWallet(ctx, walletAddress, blockchain)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUserBlocked)
	assert.NotNil(t, resultMapping)
	assert.NotNil(t, resultUser)

	mockCache.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockWalletRepo.AssertExpectations(t)
}

// Test RecognizeWallet - Wallet Flagged
func TestRecognizeWallet_WalletFlagged(t *testing.T) {
	// Setup mocks
	mockWalletRepo := new(MockWalletIdentityMappingRepository)
	mockUserRepo := new(MockUserRepository)
	mockCache := new(MockRedisCache)
	mockKYC := new(MockKYCProvider)

	service := NewIdentityMappingService(mockWalletRepo, mockUserRepo, mockCache, mockKYC)

	ctx := context.Background()
	walletAddress := "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU"
	blockchain := model.ChainSolana

	// Create test data with flagged wallet
	user := createTestUser()
	mapping := createTestWalletMapping(user.ID, walletAddress, blockchain)
	mapping.IsFlagged = true
	mapping.FlagReason = sql.NullString{String: "Suspicious activity detected", Valid: true}

	// Mock cache miss
	mockCache.On("Get", ctx, "wallet_identity:solana:"+walletAddress).Return("", errors.New("key not found"))

	// Mock DB hit
	mockWalletRepo.On("GetByWalletAddress", walletAddress, blockchain).Return(mapping, nil)
	mockUserRepo.On("GetByID", user.ID).Return(user, nil)

	// Execute
	resultMapping, resultUser, err := service.RecognizeWallet(ctx, walletAddress, blockchain)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUserBlocked)
	assert.NotNil(t, resultMapping)
	assert.NotNil(t, resultUser)

	mockCache.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockWalletRepo.AssertExpectations(t)
}

// Test RecognizeWallet - KYC Not Verified
func TestRecognizeWallet_KYCNotVerified(t *testing.T) {
	// Setup mocks
	mockWalletRepo := new(MockWalletIdentityMappingRepository)
	mockUserRepo := new(MockUserRepository)
	mockCache := new(MockRedisCache)
	mockKYC := new(MockKYCProvider)

	service := NewIdentityMappingService(mockWalletRepo, mockUserRepo, mockCache, mockKYC)

	ctx := context.Background()
	walletAddress := "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU"
	blockchain := model.ChainSolana

	// Create test data with pending KYC
	user := createTestUser()
	user.KYCStatus = model.KYCStatusPending
	user.FaceLivenessPassed = false
	mapping := createTestWalletMapping(user.ID, walletAddress, blockchain)

	// Mock cache miss
	mockCache.On("Get", ctx, "wallet_identity:solana:"+walletAddress).Return("", errors.New("key not found"))

	// Mock DB hit
	mockWalletRepo.On("GetByWalletAddress", walletAddress, blockchain).Return(mapping, nil)
	mockUserRepo.On("GetByID", user.ID).Return(user, nil)

	// Execute
	resultMapping, resultUser, err := service.RecognizeWallet(ctx, walletAddress, blockchain)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrKYCNotVerified)
	assert.NotNil(t, resultMapping)
	assert.NotNil(t, resultUser)

	mockCache.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockWalletRepo.AssertExpectations(t)
}

// Test CreateWalletMappingAfterKYC - Success
func TestCreateWalletMappingAfterKYC_Success(t *testing.T) {
	// Setup mocks
	mockWalletRepo := new(MockWalletIdentityMappingRepository)
	mockUserRepo := new(MockUserRepository)
	mockCache := new(MockRedisCache)
	mockKYC := new(MockKYCProvider)

	service := NewIdentityMappingService(mockWalletRepo, mockUserRepo, mockCache, mockKYC)

	ctx := context.Background()
	walletAddress := "NewWallet123"
	blockchain := model.ChainSolana
	user := createTestUser()

	// Mock user lookup
	mockUserRepo.On("GetByID", user.ID).Return(user, nil)

	// Mock wallet mapping doesn't exist
	mockWalletRepo.On("GetByWalletAddress", walletAddress, blockchain).Return(nil, errors.New("wallet identity mapping not found"))

	// Mock create mapping
	mockWalletRepo.On("Create", mock.AnythingOfType("*model.WalletIdentityMapping")).Return(nil)

	// Mock cache set
	mockCache.On("Set", ctx, mock.Anything, mock.Anything, 7*24*time.Hour).Return(nil)

	// Execute
	mapping, err := service.CreateWalletMappingAfterKYC(ctx, walletAddress, blockchain, user.ID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, mapping)
	assert.Equal(t, walletAddress, mapping.WalletAddress)
	assert.Equal(t, blockchain, mapping.Blockchain)
	assert.Equal(t, user.ID, mapping.UserID)

	mockUserRepo.AssertExpectations(t)
	mockWalletRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

// Test UpdateWalletActivity - Success
func TestUpdateWalletActivity_Success(t *testing.T) {
	// Setup mocks
	mockWalletRepo := new(MockWalletIdentityMappingRepository)
	mockUserRepo := new(MockUserRepository)
	mockCache := new(MockRedisCache)
	mockKYC := new(MockKYCProvider)

	service := NewIdentityMappingService(mockWalletRepo, mockUserRepo, mockCache, mockKYC)

	ctx := context.Background()
	walletAddress := "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU"
	blockchain := model.ChainSolana
	paymentID := uuid.New()
	amountUSD := decimal.NewFromFloat(100.50)

	user := createTestUser()
	mapping := createTestWalletMapping(user.ID, walletAddress, blockchain)

	// Mock get wallet mapping (twice - once for update, once for refresh)
	mockWalletRepo.On("GetByWalletAddress", walletAddress, blockchain).Return(mapping, nil).Twice()

	// Mock update activity
	mockWalletRepo.On("UpdateActivity", mapping.ID, amountUSD).Return(nil)

	// Mock increment user payment stats
	mockUserRepo.On("IncrementPaymentStats", user.ID, amountUSD).Return(nil)

	// Mock cache update
	mockCache.On("Set", ctx, mock.Anything, mock.Anything, 7*24*time.Hour).Return(nil)

	// Execute
	err := service.UpdateWalletActivity(ctx, walletAddress, blockchain, paymentID, amountUSD)

	// Assert
	assert.NoError(t, err)

	mockWalletRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

// Test FlagWallet - Success
func TestFlagWallet_Success(t *testing.T) {
	// Setup mocks
	mockWalletRepo := new(MockWalletIdentityMappingRepository)
	mockUserRepo := new(MockUserRepository)
	mockCache := new(MockRedisCache)
	mockKYC := new(MockKYCProvider)

	service := NewIdentityMappingService(mockWalletRepo, mockUserRepo, mockCache, mockKYC)

	ctx := context.Background()
	walletAddress := "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU"
	blockchain := model.ChainSolana
	reason := "Suspicious activity detected"

	user := createTestUser()
	mapping := createTestWalletMapping(user.ID, walletAddress, blockchain)

	// Mock get wallet mapping
	mockWalletRepo.On("GetByWalletAddress", walletAddress, blockchain).Return(mapping, nil)

	// Mock flag wallet
	mockWalletRepo.On("Flag", mapping.ID, reason).Return(nil)

	// Mock cache invalidation
	mockCache.On("Delete", ctx, "wallet_identity:solana:"+walletAddress).Return(nil)

	// Execute
	err := service.FlagWallet(ctx, walletAddress, blockchain, reason)

	// Assert
	assert.NoError(t, err)

	mockWalletRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

// Test GetCacheStats - Success
func TestGetCacheStats_Success(t *testing.T) {
	// Setup mocks
	mockWalletRepo := new(MockWalletIdentityMappingRepository)
	mockUserRepo := new(MockUserRepository)
	mockCache := new(MockRedisCache)
	mockKYC := new(MockKYCProvider)

	service := NewIdentityMappingService(mockWalletRepo, mockUserRepo, mockCache, mockKYC)

	ctx := context.Background()

	// Mock count
	mockWalletRepo.On("Count").Return(int64(1500), nil)

	// Mock active mappings
	activeMappings := []*model.WalletIdentityMapping{
		createTestWalletMapping(uuid.New(), "wallet1", model.ChainSolana),
		createTestWalletMapping(uuid.New(), "wallet2", model.ChainSolana),
		createTestWalletMapping(uuid.New(), "wallet3", model.ChainBSC),
	}
	mockWalletRepo.On("ListActive", 1000, 0).Return(activeMappings, nil)

	// Execute
	stats, err := service.GetCacheStats(ctx)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(1500), stats["total_wallet_mappings"])
	assert.Equal(t, 3, stats["active_mappings_7d"])
	assert.Equal(t, 7, stats["cache_ttl_days"])

	mockWalletRepo.AssertExpectations(t)
}
