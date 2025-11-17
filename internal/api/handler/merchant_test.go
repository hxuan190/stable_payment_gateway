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
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMerchantService is a mock implementation of MerchantService
type MockMerchantService struct {
	mock.Mock
}

func (m *MockMerchantService) RegisterMerchant(input service.RegisterMerchantInput) (*model.Merchant, error) {
	args := m.Called(input)
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

func (m *MockMerchantService) GetMerchantByID(merchantID string) (*model.Merchant, error) {
	args := m.Called(merchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Merchant), args.Error(1)
}

// MockPaymentRepository is a mock implementation of PaymentRepository
type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) ListByMerchant(merchantID string, limit, offset int) ([]*model.Payment, error) {
	args := m.Called(merchantID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Payment), args.Error(1)
}

func (m *MockPaymentRepository) CountByMerchant(merchantID string) (int64, error) {
	args := m.Called(merchantID)
	return args.Get(0).(int64), args.Error(1)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.Default()
}

func TestRegisterMerchant_Success(t *testing.T) {
	// Setup
	mockService := new(MockMerchantService)
	mockPaymentRepo := new(MockPaymentRepository)
	handler := NewMerchantHandler(mockService, mockPaymentRepo)

	router := setupTestRouter()
	router.POST("/api/v1/merchant/register", handler.RegisterMerchant)

	// Test data
	now := time.Now()
	expectedMerchant := &model.Merchant{
		ID:            "merchant-123",
		Email:         "test@example.com",
		BusinessName:  "Test Business",
		OwnerFullName: "John Doe",
		KYCStatus:     model.KYCStatusPending,
		Status:        model.MerchantStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	reqBody := dto.RegisterMerchantRequest{
		Email:             "test@example.com",
		BusinessName:      "Test Business",
		TaxID:             "1234567890",
		OwnerName:         "John Doe",
		PhoneNumber:       "0901234567",
		BusinessAddress:   "123 Test Street, Da Nang",
		BankName:          "Vietcombank",
		BankAccountNumber: "1234567890",
		BankAccountName:   "Test Business",
		BusinessDescription: "A test business for testing purposes",
	}

	// Mock expectations
	mockService.On("RegisterMerchant", mock.MatchedBy(func(input service.RegisterMerchantInput) bool {
		return input.Email == reqBody.Email &&
			input.BusinessName == reqBody.BusinessName &&
			input.OwnerFullName == reqBody.OwnerName
	})).Return(expectedMerchant, nil)

	// Execute
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/merchant/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	data := response.Data.(map[string]interface{})
	assert.Equal(t, expectedMerchant.ID, data["merchant_id"])
	assert.Equal(t, expectedMerchant.Email, data["email"])
	assert.Equal(t, string(model.KYCStatusPending), data["status"])

	mockService.AssertExpectations(t)
}

func TestRegisterMerchant_DuplicateEmail(t *testing.T) {
	// Setup
	mockService := new(MockMerchantService)
	mockPaymentRepo := new(MockPaymentRepository)
	handler := NewMerchantHandler(mockService, mockPaymentRepo)

	router := setupTestRouter()
	router.POST("/api/v1/merchant/register", handler.RegisterMerchant)

	reqBody := dto.RegisterMerchantRequest{
		Email:             "existing@example.com",
		BusinessName:      "Test Business",
		TaxID:             "1234567890",
		OwnerName:         "John Doe",
		PhoneNumber:       "0901234567",
		BusinessAddress:   "123 Test Street",
		BankName:          "Vietcombank",
		BankAccountNumber: "1234567890",
		BankAccountName:   "Test Business",
		BusinessDescription: "A test business",
	}

	// Mock expectations
	mockService.On("RegisterMerchant", mock.Anything).Return(nil, service.ErrMerchantAlreadyExists)

	// Execute
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/merchant/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusConflict, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "MERCHANT_ALREADY_EXISTS", response.Error.Code)

	mockService.AssertExpectations(t)
}

func TestRegisterMerchant_InvalidRequest(t *testing.T) {
	// Setup
	mockService := new(MockMerchantService)
	mockPaymentRepo := new(MockPaymentRepository)
	handler := NewMerchantHandler(mockService, mockPaymentRepo)

	router := setupTestRouter()
	router.POST("/api/v1/merchant/register", handler.RegisterMerchant)

	// Invalid request - missing required fields
	reqBody := map[string]interface{}{
		"email": "invalid-email", // Invalid email format
	}

	// Execute
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/merchant/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "INVALID_REQUEST", response.Error.Code)
}

func TestGetBalance_Success(t *testing.T) {
	// Setup
	mockService := new(MockMerchantService)
	mockPaymentRepo := new(MockPaymentRepository)
	handler := NewMerchantHandler(mockService, mockPaymentRepo)

	router := setupTestRouter()
	router.Use(func(c *gin.Context) {
		// Simulate auth middleware
		merchant := &model.Merchant{
			ID:    "merchant-123",
			Email: "test@example.com",
		}
		c.Set("merchant", merchant)
		c.Next()
	})
	router.GET("/api/v1/merchant/balance", handler.GetBalance)

	// Test data
	expectedBalance := &model.MerchantBalance{
		MerchantID:       "merchant-123",
		AvailableVND:     decimal.NewFromInt(1000000),
		PendingVND:       decimal.NewFromInt(500000),
		TotalReceivedVND: decimal.NewFromInt(5000000),
		TotalPaidOutVND:  decimal.NewFromInt(2000000),
		UpdatedAt:        time.Now(),
	}

	// Mock expectations
	mockService.On("GetMerchantBalance", "merchant-123").Return(expectedBalance, nil)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/merchant/balance", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	data := response.Data.(map[string]interface{})
	assert.Equal(t, expectedBalance.MerchantID, data["merchant_id"])
	assert.Equal(t, "VND", data["currency"])

	mockService.AssertExpectations(t)
}

func TestGetBalance_Unauthorized(t *testing.T) {
	// Setup
	mockService := new(MockMerchantService)
	mockPaymentRepo := new(MockPaymentRepository)
	handler := NewMerchantHandler(mockService, mockPaymentRepo)

	router := setupTestRouter()
	// No auth middleware - merchant not in context
	router.GET("/api/v1/merchant/balance", handler.GetBalance)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/merchant/balance", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "UNAUTHORIZED", response.Error.Code)
}

func TestGetTransactions_Success(t *testing.T) {
	// Setup
	mockService := new(MockMerchantService)
	mockPaymentRepo := new(MockPaymentRepository)
	handler := NewMerchantHandler(mockService, mockPaymentRepo)

	router := setupTestRouter()
	router.Use(func(c *gin.Context) {
		// Simulate auth middleware
		merchant := &model.Merchant{
			ID:    "merchant-123",
			Email: "test@example.com",
		}
		c.Set("merchant", merchant)
		c.Next()
	})
	router.GET("/api/v1/merchant/transactions", handler.GetTransactions)

	// Test data
	txHash := "tx-hash-123"
	orderID := "order-123"
	now := time.Now()
	payments := []*model.Payment{
		{
			ID:           "payment-1",
			MerchantID:   "merchant-123",
			AmountVND:    decimal.NewFromInt(2300000),
			AmountCrypto: decimal.NewFromFloat(100),
			Currency:     "USDT",
			Chain:        "solana",
			Status:       "completed",
			TxHash:       sql.NullString{String: txHash, Valid: true},
			OrderID:      sql.NullString{String: orderID, Valid: true},
			CreatedAt:    now,
			UpdatedAt:    now,
			ConfirmedAt:  sql.NullTime{Time: now, Valid: true},
		},
	}

	// Mock expectations
	mockPaymentRepo.On("CountByMerchant", "merchant-123").Return(int64(1), nil)
	mockPaymentRepo.On("ListByMerchant", "merchant-123", 20, 0).Return(payments, nil)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/merchant/transactions?page=1&limit=20", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	data := response.Data.(map[string]interface{})
	transactions := data["transactions"].([]interface{})
	assert.Len(t, transactions, 1)

	transaction := transactions[0].(map[string]interface{})
	assert.Equal(t, payments[0].ID, transaction["payment_id"])
	assert.Equal(t, payments[0].Status, transaction["status"])

	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(1), pagination["current_page"])
	assert.Equal(t, float64(1), pagination["total_pages"])
	assert.Equal(t, float64(1), pagination["total_items"])

	mockPaymentRepo.AssertExpectations(t)
}

func TestGetTransactions_WithPagination(t *testing.T) {
	// Setup
	mockService := new(MockMerchantService)
	mockPaymentRepo := new(MockPaymentRepository)
	handler := NewMerchantHandler(mockService, mockPaymentRepo)

	router := setupTestRouter()
	router.Use(func(c *gin.Context) {
		merchant := &model.Merchant{ID: "merchant-123"}
		c.Set("merchant", merchant)
		c.Next()
	})
	router.GET("/api/v1/merchant/transactions", handler.GetTransactions)

	// Mock expectations for page 2 with 10 items per page
	mockPaymentRepo.On("CountByMerchant", "merchant-123").Return(int64(25), nil)
	mockPaymentRepo.On("ListByMerchant", "merchant-123", 10, 10).Return([]*model.Payment{}, nil)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/merchant/transactions?page=2&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	data := response.Data.(map[string]interface{})
	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(2), pagination["current_page"])
	assert.Equal(t, float64(3), pagination["total_pages"]) // 25 items / 10 per page = 3 pages
	assert.Equal(t, float64(25), pagination["total_items"])
	assert.Equal(t, true, pagination["has_next"])
	assert.Equal(t, true, pagination["has_prev"])

	mockPaymentRepo.AssertExpectations(t)
}

func TestGetTransactions_RepositoryError(t *testing.T) {
	// Setup
	mockService := new(MockMerchantService)
	mockPaymentRepo := new(MockPaymentRepository)
	handler := NewMerchantHandler(mockService, mockPaymentRepo)

	router := setupTestRouter()
	router.Use(func(c *gin.Context) {
		merchant := &model.Merchant{ID: "merchant-123"}
		c.Set("merchant", merchant)
		c.Next()
	})
	router.GET("/api/v1/merchant/transactions", handler.GetTransactions)

	// Mock expectations
	mockPaymentRepo.On("CountByMerchant", "merchant-123").Return(int64(0), errors.New("database error"))

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/merchant/transactions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "QUERY_FAILED", response.Error.Code)

	mockPaymentRepo.AssertExpectations(t)
}

func TestGetProfile_Success(t *testing.T) {
	// Setup
	mockService := new(MockMerchantService)
	mockPaymentRepo := new(MockPaymentRepository)
	handler := NewMerchantHandler(mockService, mockPaymentRepo)

	router := setupTestRouter()
	router.Use(func(c *gin.Context) {
		merchant := &model.Merchant{
			ID:            "merchant-123",
			Email:         "test@example.com",
			BusinessName:  "Test Business",
			OwnerFullName: "John Doe",
			BusinessTaxID: sql.NullString{String: "1234567890", Valid: true},
			PhoneNumber:   sql.NullString{String: "0901234567", Valid: true},
			KYCStatus:     model.KYCStatusApproved,
			Status:        model.MerchantStatusActive,
			CreatedAt:     time.Now(),
		}
		c.Set("merchant", merchant)
		c.Next()
	})
	router.GET("/api/v1/merchant/profile", handler.GetProfile)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/merchant/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	data := response.Data.(map[string]interface{})
	assert.Equal(t, "merchant-123", data["merchant_id"])
	assert.Equal(t, "test@example.com", data["email"])
	assert.Equal(t, "Test Business", data["business_name"])
	assert.Equal(t, "John Doe", data["owner_name"])
	assert.Equal(t, "1234567890", data["tax_id"])
	assert.Equal(t, string(model.KYCStatusApproved), data["kyc_status"])
}
