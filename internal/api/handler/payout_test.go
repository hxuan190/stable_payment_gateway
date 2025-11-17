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
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/hxuan190/stable_payment_gateway/internal/api/dto"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
)

// MockPayoutService is a mock implementation of PayoutService
type MockPayoutService struct {
	mock.Mock
}

func (m *MockPayoutService) RequestPayout(input service.RequestPayoutInput) (*model.Payout, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payout), args.Error(1)
}

func (m *MockPayoutService) GetPayoutByID(payoutID string) (*model.Payout, error) {
	args := m.Called(payoutID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payout), args.Error(1)
}

func (m *MockPayoutService) ListPayoutsByMerchant(merchantID string, limit, offset int) ([]*model.Payout, error) {
	args := m.Called(merchantID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Payout), args.Error(1)
}

// Test helpers

func createTestPayout() *model.Payout {
	return &model.Payout{
		ID:                "payout-123",
		MerchantID:        "merchant-123",
		AmountVND:         decimal.NewFromInt(5000000),
		FeeVND:            decimal.NewFromInt(25000),
		NetAmountVND:      decimal.NewFromInt(4975000),
		BankAccountName:   "Test Account",
		BankAccountNumber: "1234567890",
		BankName:          "Test Bank",
		BankBranch:        sql.NullString{String: "Main Branch", Valid: true},
		Status:            model.PayoutStatusRequested,
		RequestedBy:       "merchant-123",
		RetryCount:        0,
		Notes:             sql.NullString{String: "Test payout", Valid: true},
		Metadata:          model.JSONBMap{},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// Test RequestPayout endpoint

func TestRequestPayout_Success(t *testing.T) {
	mockService := new(MockPayoutService)
	handler := NewPayoutHandler(mockService)

	router := setupTestRouter()
	merchant := createTestMerchant()

	// Set up middleware to inject merchant
	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.POST("/payouts", handler.RequestPayout)

	payout := createTestPayout()

	// Mock service response
	mockService.On("RequestPayout", mock.MatchedBy(func(input service.RequestPayoutInput) bool {
		return input.MerchantID == merchant.ID &&
			input.AmountVND.Equal(decimal.NewFromInt(5000000)) &&
			input.BankAccountNumber == "1234567890"
	})).Return(payout, nil)

	// Prepare request
	reqBody := dto.RequestPayoutRequest{
		AmountVND:         5000000,
		BankAccountName:   "Test Account",
		BankAccountNumber: "1234567890",
		BankName:          "Test Bank",
		BankBranch:        "Main Branch",
		Notes:             "Test payout",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/payouts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Data)

	mockService.AssertExpectations(t)
}

func TestRequestPayout_InvalidAmount(t *testing.T) {
	mockService := new(MockPayoutService)
	handler := NewPayoutHandler(mockService)

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.POST("/payouts", handler.RequestPayout)

	// Prepare request with invalid amount (negative)
	reqBody := dto.RequestPayoutRequest{
		AmountVND:         -1000000,
		BankAccountName:   "Test Account",
		BankAccountNumber: "1234567890",
		BankName:          "Test Bank",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/payouts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "INVALID_REQUEST", response.Error.Code)
}

func TestRequestPayout_InsufficientBalance(t *testing.T) {
	mockService := new(MockPayoutService)
	handler := NewPayoutHandler(mockService)

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.POST("/payouts", handler.RequestPayout)

	// Mock service to return insufficient balance error
	mockService.On("RequestPayout", mock.Anything).Return(nil, service.ErrPayoutInsufficientBalance)

	// Prepare request
	reqBody := dto.RequestPayoutRequest{
		AmountVND:         10000000,
		BankAccountName:   "Test Account",
		BankAccountNumber: "1234567890",
		BankName:          "Test Bank",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/payouts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "INSUFFICIENT_BALANCE", response.Error.Code)

	mockService.AssertExpectations(t)
}

func TestRequestPayout_BelowMinimum(t *testing.T) {
	mockService := new(MockPayoutService)
	handler := NewPayoutHandler(mockService)

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.POST("/payouts", handler.RequestPayout)

	// Mock service to return below minimum error
	mockService.On("RequestPayout", mock.Anything).Return(nil, service.ErrPayoutBelowMinimum)

	// Prepare request with amount below minimum
	reqBody := dto.RequestPayoutRequest{
		AmountVND:         500000, // Below 1M VND minimum
		BankAccountName:   "Test Account",
		BankAccountNumber: "1234567890",
		BankName:          "Test Bank",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/payouts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "AMOUNT_BELOW_MINIMUM", response.Error.Code)

	mockService.AssertExpectations(t)
}

func TestRequestPayout_NoAuthentication(t *testing.T) {
	mockService := new(MockPayoutService)
	handler := NewPayoutHandler(mockService)

	router := setupTestRouter()
	router.POST("/payouts", handler.RequestPayout)

	// Prepare request
	reqBody := dto.RequestPayoutRequest{
		AmountVND:         5000000,
		BankAccountName:   "Test Account",
		BankAccountNumber: "1234567890",
		BankName:          "Test Bank",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/payouts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "UNAUTHORIZED", response.Error.Code)
}

// Test GetPayout endpoint

func TestGetPayout_Success(t *testing.T) {
	mockService := new(MockPayoutService)
	handler := NewPayoutHandler(mockService)

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.GET("/payouts/:id", handler.GetPayout)

	payout := createTestPayout()

	// Mock service response
	mockService.On("GetPayoutByID", "payout-123").Return(payout, nil)

	req, _ := http.NewRequest(http.MethodGet, "/payouts/payout-123", nil)

	// Execute request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Data)

	mockService.AssertExpectations(t)
}

func TestGetPayout_NotFound(t *testing.T) {
	mockService := new(MockPayoutService)
	handler := NewPayoutHandler(mockService)

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.GET("/payouts/:id", handler.GetPayout)

	// Mock service to return not found error
	mockService.On("GetPayoutByID", "nonexistent").Return(nil, service.ErrPayoutNotFound)

	req, _ := http.NewRequest(http.MethodGet, "/payouts/nonexistent", nil)

	// Execute request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "PAYOUT_NOT_FOUND", response.Error.Code)

	mockService.AssertExpectations(t)
}

func TestGetPayout_Forbidden(t *testing.T) {
	mockService := new(MockPayoutService)
	handler := NewPayoutHandler(mockService)

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.GET("/payouts/:id", handler.GetPayout)

	payout := createTestPayout()
	payout.MerchantID = "different-merchant" // Different merchant ID

	// Mock service response
	mockService.On("GetPayoutByID", "payout-123").Return(payout, nil)

	req, _ := http.NewRequest(http.MethodGet, "/payouts/payout-123", nil)

	// Execute request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusForbidden, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "FORBIDDEN", response.Error.Code)

	mockService.AssertExpectations(t)
}

// Test ListPayouts endpoint

func TestListPayouts_Success(t *testing.T) {
	mockService := new(MockPayoutService)
	handler := NewPayoutHandler(mockService)

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.GET("/payouts", handler.ListPayouts)

	payouts := []*model.Payout{
		createTestPayout(),
		createTestPayout(),
	}

	// Mock service response
	mockService.On("ListPayoutsByMerchant", merchant.ID, 20, 0).Return(payouts, nil)

	req, _ := http.NewRequest(http.MethodGet, "/payouts", nil)

	// Execute request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Data)

	mockService.AssertExpectations(t)
}

func TestListPayouts_WithPagination(t *testing.T) {
	mockService := new(MockPayoutService)
	handler := NewPayoutHandler(mockService)

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.GET("/payouts", handler.ListPayouts)

	payouts := []*model.Payout{
		createTestPayout(),
	}

	// Mock service response with pagination parameters
	mockService.On("ListPayoutsByMerchant", merchant.ID, 10, 10).Return(payouts, nil)

	req, _ := http.NewRequest(http.MethodGet, "/payouts?page=2&per_page=10", nil)

	// Execute request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Nil(t, response.Error)

	mockService.AssertExpectations(t)
}

func TestListPayouts_ServiceError(t *testing.T) {
	mockService := new(MockPayoutService)
	handler := NewPayoutHandler(mockService)

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.GET("/payouts", handler.ListPayouts)

	// Mock service to return error
	mockService.On("ListPayoutsByMerchant", merchant.ID, 20, 0).Return(nil, errors.New("database error"))

	req, _ := http.NewRequest(http.MethodGet, "/payouts", nil)

	// Execute request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "INTERNAL_ERROR", response.Error.Code)

	mockService.AssertExpectations(t)
}
