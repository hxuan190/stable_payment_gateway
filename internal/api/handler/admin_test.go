package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hxuan190/stable_payment_gateway/internal/api/dto"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Services
type MockMerchantService struct {
	mock.Mock
}

func (m *MockMerchantService) ApproveKYC(merchantID string, approvedBy string) (*model.Merchant, error) {
	args := m.Called(merchantID, approvedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Merchant), args.Error(1)
}

func (m *MockMerchantService) RejectKYC(merchantID string, reason string) (*model.Merchant, error) {
	args := m.Called(merchantID, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Merchant), args.Error(1)
}

func (m *MockMerchantService) GetMerchantBalance(merchantID string) (*model.MerchantBalance, error) {
	args := m.Called(merchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MerchantBalance), args.Error(1)
}

type MockPayoutService struct {
	mock.Mock
}

func (m *MockPayoutService) ApprovePayout(payoutID, approvedBy string) error {
	args := m.Called(payoutID, approvedBy)
	return args.Error(0)
}

func (m *MockPayoutService) RejectPayout(payoutID, reason string) error {
	args := m.Called(payoutID, reason)
	return args.Error(0)
}

func (m *MockPayoutService) CompletePayout(payoutID, bankReferenceNumber, processedBy string) error {
	args := m.Called(payoutID, bankReferenceNumber, processedBy)
	return args.Error(0)
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

func (m *MockMerchantRepository) List(limit, offset int) ([]*model.Merchant, error) {
	args := m.Called(limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) ListByKYCStatus(status string, limit, offset int) ([]*model.Merchant, error) {
	args := m.Called(status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Merchant), args.Error(1)
}

type MockPayoutRepository struct {
	mock.Mock
}

func (m *MockPayoutRepository) GetByID(id string) (*model.Payout, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payout), args.Error(1)
}

func (m *MockPayoutRepository) ListByMerchant(merchantID string, limit, offset int) ([]*model.Payout, error) {
	args := m.Called(merchantID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Payout), args.Error(1)
}

func (m *MockPayoutRepository) ListByStatus(status model.PayoutStatus, limit, offset int) ([]*model.Payout, error) {
	args := m.Called(status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Payout), args.Error(1)
}

type MockBalanceRepository struct {
	mock.Mock
}

func (m *MockBalanceRepository) GetByMerchantID(merchantID string) (*model.MerchantBalance, error) {
	args := m.Called(merchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MerchantBalance), args.Error(1)
}

type MockPaymentRepository struct {
	mock.Mock
}

type MockWallet struct {
	mock.Mock
}

func (m *MockWallet) GetBalance() (decimal.Decimal, error) {
	args := m.Called()
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// Test ApproveKYC
func TestAdminHandler_ApproveKYC(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockMerchantService := new(MockMerchantService)
		mockPayoutService := new(MockPayoutService)
		mockMerchantRepo := new(MockMerchantRepository)
		mockPayoutRepo := new(MockPayoutRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockBalanceRepo := new(MockBalanceRepository)
		mockWallet := new(MockWallet)

		handler := NewAdminHandler(
			mockMerchantService,
			mockPayoutService,
			mockMerchantRepo,
			mockPayoutRepo,
			mockPaymentRepo,
			mockBalanceRepo,
			mockWallet,
		)

		now := time.Now()
		expectedMerchant := &model.Merchant{
			ID:            "merchant-123",
			KYCStatus:     model.KYCStatusApproved,
			APIKey:        sql.NullString{String: "pk_test_123", Valid: true},
			KYCApprovedAt: sql.NullTime{Time: now, Valid: true},
			KYCApprovedBy: sql.NullString{String: "admin@test.com", Valid: true},
		}

		mockMerchantService.On("ApproveKYC", "merchant-123", "admin@test.com").Return(expectedMerchant, nil)

		router := setupTestRouter()
		router.POST("/merchants/:id/kyc/approve", handler.ApproveKYC)

		reqBody := dto.ApproveKYCRequest{
			ApprovedBy: "admin@test.com",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/merchants/merchant-123/kyc/approve", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockMerchantService.AssertExpectations(t)
	})

	t.Run("Merchant Not Found", func(t *testing.T) {
		mockMerchantService := new(MockMerchantService)
		mockPayoutService := new(MockPayoutService)
		mockMerchantRepo := new(MockMerchantRepository)
		mockPayoutRepo := new(MockPayoutRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockBalanceRepo := new(MockBalanceRepository)
		mockWallet := new(MockWallet)

		handler := NewAdminHandler(
			mockMerchantService,
			mockPayoutService,
			mockMerchantRepo,
			mockPayoutRepo,
			mockPaymentRepo,
			mockBalanceRepo,
			mockWallet,
		)

		mockMerchantService.On("ApproveKYC", "merchant-999", "admin@test.com").
			Return(nil, service.ErrMerchantNotFound)

		router := setupTestRouter()
		router.POST("/merchants/:id/kyc/approve", handler.ApproveKYC)

		reqBody := dto.ApproveKYCRequest{
			ApprovedBy: "admin@test.com",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/merchants/merchant-999/kyc/approve", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockMerchantService.AssertExpectations(t)
	})

	t.Run("Invalid KYC Status", func(t *testing.T) {
		mockMerchantService := new(MockMerchantService)
		mockPayoutService := new(MockPayoutService)
		mockMerchantRepo := new(MockMerchantRepository)
		mockPayoutRepo := new(MockPayoutRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockBalanceRepo := new(MockBalanceRepository)
		mockWallet := new(MockWallet)

		handler := NewAdminHandler(
			mockMerchantService,
			mockPayoutService,
			mockMerchantRepo,
			mockPayoutRepo,
			mockPaymentRepo,
			mockBalanceRepo,
			mockWallet,
		)

		mockMerchantService.On("ApproveKYC", "merchant-123", "admin@test.com").
			Return(nil, service.ErrInvalidKYCStatus)

		router := setupTestRouter()
		router.POST("/merchants/:id/kyc/approve", handler.ApproveKYC)

		reqBody := dto.ApproveKYCRequest{
			ApprovedBy: "admin@test.com",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/merchants/merchant-123/kyc/approve", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockMerchantService.AssertExpectations(t)
	})
}

// Test RejectKYC
func TestAdminHandler_RejectKYC(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockMerchantService := new(MockMerchantService)
		mockPayoutService := new(MockPayoutService)
		mockMerchantRepo := new(MockMerchantRepository)
		mockPayoutRepo := new(MockPayoutRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockBalanceRepo := new(MockBalanceRepository)
		mockWallet := new(MockWallet)

		handler := NewAdminHandler(
			mockMerchantService,
			mockPayoutService,
			mockMerchantRepo,
			mockPayoutRepo,
			mockPaymentRepo,
			mockBalanceRepo,
			mockWallet,
		)

		now := time.Now()
		expectedMerchant := &model.Merchant{
			ID:                 "merchant-123",
			KYCStatus:          model.KYCStatusRejected,
			KYCRejectedAt:      sql.NullTime{Time: now, Valid: true},
			KYCRejectionReason: sql.NullString{String: "Invalid documents", Valid: true},
		}

		mockMerchantService.On("RejectKYC", "merchant-123", "Invalid documents").
			Return(expectedMerchant, nil)

		router := setupTestRouter()
		router.POST("/merchants/:id/kyc/reject", handler.RejectKYC)

		reqBody := dto.RejectKYCRequest{
			Reason: "Invalid documents",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/merchants/merchant-123/kyc/reject", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockMerchantService.AssertExpectations(t)
	})
}

// Test ApprovePayout
func TestAdminHandler_ApprovePayout(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockMerchantService := new(MockMerchantService)
		mockPayoutService := new(MockPayoutService)
		mockMerchantRepo := new(MockMerchantRepository)
		mockPayoutRepo := new(MockPayoutRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockBalanceRepo := new(MockBalanceRepository)
		mockWallet := new(MockWallet)

		handler := NewAdminHandler(
			mockMerchantService,
			mockPayoutService,
			mockMerchantRepo,
			mockPayoutRepo,
			mockPaymentRepo,
			mockBalanceRepo,
			mockWallet,
		)

		now := time.Now()
		payout := &model.Payout{
			ID:         "payout-123",
			MerchantID: "merchant-123",
			Status:     model.PayoutStatusApproved,
			ApprovedAt: sql.NullTime{Time: now, Valid: true},
			ApprovedBy: sql.NullString{String: "admin@test.com", Valid: true},
		}

		mockPayoutService.On("ApprovePayout", "payout-123", "admin@test.com").Return(nil)
		mockPayoutRepo.On("GetByID", "payout-123").Return(payout, nil)

		router := setupTestRouter()
		router.POST("/payouts/:id/approve", handler.ApprovePayout)

		reqBody := dto.ApprovePayoutRequest{
			ApprovedBy: "admin@test.com",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/payouts/payout-123/approve", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockPayoutService.AssertExpectations(t)
	})

	t.Run("Payout Not Found", func(t *testing.T) {
		mockMerchantService := new(MockMerchantService)
		mockPayoutService := new(MockPayoutService)
		mockMerchantRepo := new(MockMerchantRepository)
		mockPayoutRepo := new(MockPayoutRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockBalanceRepo := new(MockBalanceRepository)
		mockWallet := new(MockWallet)

		handler := NewAdminHandler(
			mockMerchantService,
			mockPayoutService,
			mockMerchantRepo,
			mockPayoutRepo,
			mockPaymentRepo,
			mockBalanceRepo,
			mockWallet,
		)

		mockPayoutService.On("ApprovePayout", "payout-999", "admin@test.com").
			Return(service.ErrPayoutNotFound)

		router := setupTestRouter()
		router.POST("/payouts/:id/approve", handler.ApprovePayout)

		reqBody := dto.ApprovePayoutRequest{
			ApprovedBy: "admin@test.com",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/payouts/payout-999/approve", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockPayoutService.AssertExpectations(t)
	})
}

// Test ListMerchants
func TestAdminHandler_ListMerchants(t *testing.T) {
	t.Run("Success - All Merchants", func(t *testing.T) {
		mockMerchantService := new(MockMerchantService)
		mockPayoutService := new(MockPayoutService)
		mockMerchantRepo := new(MockMerchantRepository)
		mockPayoutRepo := new(MockPayoutRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockBalanceRepo := new(MockBalanceRepository)
		mockWallet := new(MockWallet)

		handler := NewAdminHandler(
			mockMerchantService,
			mockPayoutService,
			mockMerchantRepo,
			mockPayoutRepo,
			mockPaymentRepo,
			mockBalanceRepo,
			mockWallet,
		)

		merchants := []*model.Merchant{
			{
				ID:           "merchant-1",
				Email:        "test1@example.com",
				BusinessName: "Business 1",
				KYCStatus:    model.KYCStatusApproved,
				IsActive:     true,
				CreatedAt:    time.Now(),
			},
			{
				ID:           "merchant-2",
				Email:        "test2@example.com",
				BusinessName: "Business 2",
				KYCStatus:    model.KYCStatusPending,
				IsActive:     true,
				CreatedAt:    time.Now(),
			},
		}

		mockMerchantRepo.On("List", 20, 0).Return(merchants, nil)

		router := setupTestRouter()
		router.GET("/merchants", handler.ListMerchants)

		req := httptest.NewRequest(http.MethodGet, "/merchants", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockMerchantRepo.AssertExpectations(t)
	})

	t.Run("Success - Filtered By KYC Status", func(t *testing.T) {
		mockMerchantService := new(MockMerchantService)
		mockPayoutService := new(MockPayoutService)
		mockMerchantRepo := new(MockMerchantRepository)
		mockPayoutRepo := new(MockPayoutRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockBalanceRepo := new(MockBalanceRepository)
		mockWallet := new(MockWallet)

		handler := NewAdminHandler(
			mockMerchantService,
			mockPayoutService,
			mockMerchantRepo,
			mockPayoutRepo,
			mockPaymentRepo,
			mockBalanceRepo,
			mockWallet,
		)

		merchants := []*model.Merchant{
			{
				ID:           "merchant-2",
				Email:        "test2@example.com",
				BusinessName: "Business 2",
				KYCStatus:    model.KYCStatusPending,
				IsActive:     true,
				CreatedAt:    time.Now(),
			},
		}

		mockMerchantRepo.On("ListByKYCStatus", "pending", 20, 0).Return(merchants, nil)

		router := setupTestRouter()
		router.GET("/merchants", handler.ListMerchants)

		req := httptest.NewRequest(http.MethodGet, "/merchants?kyc_status=pending", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockMerchantRepo.AssertExpectations(t)
	})
}

// Test GetMerchant
func TestAdminHandler_GetMerchant(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockMerchantService := new(MockMerchantService)
		mockPayoutService := new(MockPayoutService)
		mockMerchantRepo := new(MockMerchantRepository)
		mockPayoutRepo := new(MockPayoutRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockBalanceRepo := new(MockBalanceRepository)
		mockWallet := new(MockWallet)

		handler := NewAdminHandler(
			mockMerchantService,
			mockPayoutService,
			mockMerchantRepo,
			mockPayoutRepo,
			mockPaymentRepo,
			mockBalanceRepo,
			mockWallet,
		)

		merchant := &model.Merchant{
			ID:           "merchant-123",
			Email:        "test@example.com",
			BusinessName: "Test Business",
			KYCStatus:    model.KYCStatusApproved,
			IsActive:     true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		balance := &model.MerchantBalance{
			MerchantID:       "merchant-123",
			AvailableVND:     decimal.NewFromInt(10000000),
			PendingVND:       decimal.NewFromInt(1000000),
			TotalReceivedVND: decimal.NewFromInt(50000000),
			TotalPaidOutVND:  decimal.NewFromInt(39000000),
		}

		mockMerchantRepo.On("GetByID", "merchant-123").Return(merchant, nil)
		mockBalanceRepo.On("GetByMerchantID", "merchant-123").Return(balance, nil)

		router := setupTestRouter()
		router.GET("/merchants/:id", handler.GetMerchant)

		req := httptest.NewRequest(http.MethodGet, "/merchants/merchant-123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockMerchantRepo.AssertExpectations(t)
		mockBalanceRepo.AssertExpectations(t)
	})

	t.Run("Merchant Not Found", func(t *testing.T) {
		mockMerchantService := new(MockMerchantService)
		mockPayoutService := new(MockPayoutService)
		mockMerchantRepo := new(MockMerchantRepository)
		mockPayoutRepo := new(MockPayoutRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockBalanceRepo := new(MockBalanceRepository)
		mockWallet := new(MockWallet)

		handler := NewAdminHandler(
			mockMerchantService,
			mockPayoutService,
			mockMerchantRepo,
			mockPayoutRepo,
			mockPaymentRepo,
			mockBalanceRepo,
			mockWallet,
		)

		mockMerchantRepo.On("GetByID", "merchant-999").
			Return(nil, errors.New("merchant not found"))

		router := setupTestRouter()
		router.GET("/merchants/:id", handler.GetMerchant)

		req := httptest.NewRequest(http.MethodGet, "/merchants/merchant-999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockMerchantRepo.AssertExpectations(t)
	})
}

// Test GetStats
func TestAdminHandler_GetStats(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockMerchantService := new(MockMerchantService)
		mockPayoutService := new(MockPayoutService)
		mockMerchantRepo := new(MockMerchantRepository)
		mockPayoutRepo := new(MockPayoutRepository)
		mockPaymentRepo := new(MockPaymentRepository)
		mockBalanceRepo := new(MockBalanceRepository)
		mockWallet := new(MockWallet)

		handler := NewAdminHandler(
			mockMerchantService,
			mockPayoutService,
			mockMerchantRepo,
			mockPayoutRepo,
			mockPaymentRepo,
			mockBalanceRepo,
			mockWallet,
		)

		merchants := []*model.Merchant{
			{
				ID:        "merchant-1",
				KYCStatus: model.KYCStatusApproved,
			},
			{
				ID:        "merchant-2",
				KYCStatus: model.KYCStatusPending,
			},
			{
				ID:        "merchant-3",
				KYCStatus: model.KYCStatusApproved,
			},
		}

		payouts := []*model.Payout{
			{
				ID:        "payout-1",
				AmountVND: decimal.NewFromInt(1000000),
				Status:    model.PayoutStatusCompleted,
			},
			{
				ID:        "payout-2",
				AmountVND: decimal.NewFromInt(2000000),
				Status:    model.PayoutStatusRequested,
			},
		}

		mockMerchantRepo.On("List", 1000, 0).Return(merchants, nil)
		mockPayoutRepo.On("ListByStatus", model.PayoutStatus(""), 1000, 0).Return(payouts, nil)
		mockWallet.On("GetBalance").Return(decimal.NewFromInt(5000), nil)

		router := setupTestRouter()
		router.GET("/stats", handler.GetStats)

		req := httptest.NewRequest(http.MethodGet, "/stats", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockMerchantRepo.AssertExpectations(t)
		mockPayoutRepo.AssertExpectations(t)
		mockWallet.AssertExpectations(t)
	})
}
