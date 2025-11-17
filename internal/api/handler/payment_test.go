package handler

import (
	"bytes"
	"context"
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

// MockPaymentService is a mock implementation of PaymentService
type MockPaymentService struct {
	mock.Mock
}

func (m *MockPaymentService) CreatePayment(ctx context.Context, req service.CreatePaymentRequest) (*model.Payment, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payment), args.Error(1)
}

func (m *MockPaymentService) GetPaymentStatus(ctx context.Context, paymentID string) (*model.Payment, error) {
	args := m.Called(ctx, paymentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Payment), args.Error(1)
}

func (m *MockPaymentService) ListPaymentsByMerchant(ctx context.Context, merchantID string, limit, offset int) ([]*model.Payment, error) {
	args := m.Called(ctx, merchantID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Payment), args.Error(1)
}

// Test helpers

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func createTestMerchant() *model.Merchant {
	return &model.Merchant{
		ID:           "merchant-123",
		Email:        "test@example.com",
		BusinessName: "Test Business",
		KYCStatus:    model.KYCStatusApproved,
		Status:       model.MerchantStatusActive,
	}
}

func createTestPayment() *model.Payment {
	return &model.Payment{
		ID:                "payment-123",
		MerchantID:        "merchant-123",
		AmountVND:         decimal.NewFromInt(2300000),
		AmountCrypto:      decimal.NewFromFloat(100.0),
		Currency:          "USDT",
		Chain:             model.ChainSolana,
		ExchangeRate:      decimal.NewFromInt(23000),
		Status:            model.PaymentStatusCreated,
		PaymentReference:  "abcdef123456",
		DestinationWallet: "SolanaWalletAddress123",
		ExpiresAt:         time.Now().Add(30 * time.Minute),
		FeePercentage:     decimal.NewFromFloat(0.01),
		FeeVND:            decimal.NewFromInt(23000),
		NetAmountVND:      decimal.NewFromInt(2277000),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// Test CreatePayment endpoint

func TestCreatePayment_Success(t *testing.T) {
	mockService := new(MockPaymentService)
	handler := NewPaymentHandler(mockService, "http://localhost:8080")

	router := setupTestRouter()
	merchant := createTestMerchant()

	// Set up middleware to inject merchant
	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.POST("/payments", handler.CreatePayment)

	payment := createTestPayment()

	// Mock service response
	mockService.On("CreatePayment", mock.Anything, mock.MatchedBy(func(req service.CreatePaymentRequest) bool {
		return req.MerchantID == merchant.ID &&
			req.AmountVND.Equal(decimal.NewFromInt(2300000)) &&
			req.Currency == "USDT"
	})).Return(payment, nil)

	// Prepare request
	reqBody := dto.CreatePaymentRequest{
		AmountVND:   2300000,
		Currency:    "USDT",
		OrderID:     "order-123",
		Description: "Test payment",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
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

func TestCreatePayment_InvalidAmount(t *testing.T) {
	mockService := new(MockPaymentService)
	handler := NewPaymentHandler(mockService, "http://localhost:8080")

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.POST("/payments", handler.CreatePayment)

	// Prepare request with invalid amount (0)
	reqBody := dto.CreatePaymentRequest{
		AmountVND: 0,
		Currency:  "USDT",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
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

func TestCreatePayment_ServiceError(t *testing.T) {
	mockService := new(MockPaymentService)
	handler := NewPaymentHandler(mockService, "http://localhost:8080")

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.POST("/payments", handler.CreatePayment)

	// Mock service error
	mockService.On("CreatePayment", mock.Anything, mock.Anything).
		Return(nil, service.ErrInvalidAmount)

	// Prepare request
	reqBody := dto.CreatePaymentRequest{
		AmountVND: 2300000,
		Currency:  "USDT",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
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
	assert.Equal(t, "INVALID_AMOUNT", response.Error.Code)

	mockService.AssertExpectations(t)
}

func TestCreatePayment_MerchantNotApproved(t *testing.T) {
	mockService := new(MockPaymentService)
	handler := NewPaymentHandler(mockService, "http://localhost:8080")

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.POST("/payments", handler.CreatePayment)

	// Mock service error for merchant not approved
	mockService.On("CreatePayment", mock.Anything, mock.Anything).
		Return(nil, service.ErrMerchantNotApproved)

	// Prepare request
	reqBody := dto.CreatePaymentRequest{
		AmountVND: 2300000,
		Currency:  "USDT",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusForbidden, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "MERCHANT_NOT_APPROVED", response.Error.Code)

	mockService.AssertExpectations(t)
}

// Test GetPayment endpoint

func TestGetPayment_Success(t *testing.T) {
	mockService := new(MockPaymentService)
	handler := NewPaymentHandler(mockService, "http://localhost:8080")

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.GET("/payments/:id", handler.GetPayment)

	payment := createTestPayment()

	// Mock service response
	mockService.On("GetPaymentStatus", mock.Anything, payment.ID).
		Return(payment, nil)

	// Execute request
	req, _ := http.NewRequest(http.MethodGet, "/payments/"+payment.ID, nil)
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

func TestGetPayment_NotFound(t *testing.T) {
	mockService := new(MockPaymentService)
	handler := NewPaymentHandler(mockService, "http://localhost:8080")

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.GET("/payments/:id", handler.GetPayment)

	// Mock service error
	mockService.On("GetPaymentStatus", mock.Anything, "invalid-payment-id").
		Return(nil, service.ErrPaymentNotFound)

	// Execute request
	req, _ := http.NewRequest(http.MethodGet, "/payments/invalid-payment-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, "PAYMENT_NOT_FOUND", response.Error.Code)

	mockService.AssertExpectations(t)
}

func TestGetPayment_WrongMerchant(t *testing.T) {
	mockService := new(MockPaymentService)
	handler := NewPaymentHandler(mockService, "http://localhost:8080")

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.GET("/payments/:id", handler.GetPayment)

	// Create payment belonging to different merchant
	payment := createTestPayment()
	payment.MerchantID = "different-merchant-id"

	// Mock service response
	mockService.On("GetPaymentStatus", mock.Anything, payment.ID).
		Return(payment, nil)

	// Execute request
	req, _ := http.NewRequest(http.MethodGet, "/payments/"+payment.ID, nil)
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

// Test ListPayments endpoint

func TestListPayments_Success(t *testing.T) {
	mockService := new(MockPaymentService)
	handler := NewPaymentHandler(mockService, "http://localhost:8080")

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.GET("/payments", handler.ListPayments)

	// Create test payments
	payments := []*model.Payment{
		createTestPayment(),
		createTestPayment(),
	}

	// Mock service response
	mockService.On("ListPaymentsByMerchant", mock.Anything, merchant.ID, 20, 0).
		Return(payments, nil)

	// Execute request
	req, _ := http.NewRequest(http.MethodGet, "/payments", nil)
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

func TestListPayments_WithPagination(t *testing.T) {
	mockService := new(MockPaymentService)
	handler := NewPaymentHandler(mockService, "http://localhost:8080")

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.GET("/payments", handler.ListPayments)

	payments := []*model.Payment{createTestPayment()}

	// Mock service response with custom pagination
	mockService.On("ListPaymentsByMerchant", mock.Anything, merchant.ID, 10, 10).
		Return(payments, nil)

	// Execute request with pagination params
	req, _ := http.NewRequest(http.MethodGet, "/payments?page=2&per_page=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	mockService.AssertExpectations(t)
}

func TestListPayments_ServiceError(t *testing.T) {
	mockService := new(MockPaymentService)
	handler := NewPaymentHandler(mockService, "http://localhost:8080")

	router := setupTestRouter()
	merchant := createTestMerchant()

	router.Use(func(c *gin.Context) {
		c.Set("merchant", merchant)
		c.Next()
	})

	router.GET("/payments", handler.ListPayments)

	// Mock service error
	mockService.On("ListPaymentsByMerchant", mock.Anything, merchant.ID, 20, 0).
		Return(nil, errors.New("database error"))

	// Execute request
	req, _ := http.NewRequest(http.MethodGet, "/payments", nil)
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
