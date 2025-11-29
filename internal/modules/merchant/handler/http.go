package handler

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/service"
	paymentdomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
)

// MerchantHandler handles HTTP requests for merchant operations
type MerchantHandler struct {
	merchantService *service.MerchantService
	paymentRepo     paymentdomain.PaymentRepository
}

// NewMerchantHandler creates a new merchant handler instance
func NewMerchantHandler(
	merchantService *service.MerchantService,
	paymentRepo paymentdomain.PaymentRepository,
) *MerchantHandler {
	return &MerchantHandler{
		merchantService: merchantService,
		paymentRepo:     paymentRepo,
	}
}

// RegisterMerchant handles merchant registration
// POST /api/v1/merchant/register
func (h *MerchantHandler) RegisterMerchant(c *gin.Context) {
	var req RegisterMerchantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponseWithDetails(
			"INVALID_REQUEST",
			"Invalid request body",
			fmt.Sprintf("%v", err),
		))
		return
	}

	// Map DTO to service input
	input := service.RegisterMerchantRequest{
		Email:             req.Email,
		BusinessName:      req.BusinessName,
		BusinessTaxID:     req.TaxID,
		OwnerFullName:     req.OwnerName,
		PhoneNumber:       req.PhoneNumber,
		BusinessAddress:   req.BusinessAddress,
		BankAccountName:   req.BankAccountName,
		BankAccountNumber: req.BankAccountNumber,
		BankName:          req.BankName,
	}

	// Register merchant
	merchant, err := h.merchantService.RegisterMerchant(input)
	if err != nil {
		if errors.Is(err, service.ErrMerchantAlreadyExists) {
			c.JSON(http.StatusConflict, ErrorResponse(
				"MERCHANT_ALREADY_EXISTS",
				"A merchant with this email already exists",
			))
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse(
			"REGISTRATION_FAILED",
			"Failed to register merchant",
		))
		return
	}

	// Build response
	response := APIResponse{
		Data: RegisterMerchantResponse{
			MerchantID: merchant.ID,
			Email:      merchant.Email,
			Status:     string(merchant.KYCStatus),
			Message:    "Merchant registration successful. Your account is pending KYC approval. You will receive an email with your API key once approved.",
			CreatedAt:  merchant.CreatedAt,
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusCreated, response)
}

// GetBalance retrieves the merchant's current balance
// GET /api/v1/merchant/balance
func (h *MerchantHandler) GetBalance(c *gin.Context) {
	// Get merchant from context (set by auth middleware)
	merchantInterface, exists := c.Get("merchant")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse(
			"UNAUTHORIZED",
			"Merchant not authenticated",
		))
		return
	}

	merchant, ok := merchantInterface.(*domain.Merchant)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse(
			"INTERNAL_ERROR",
			"Failed to retrieve merchant information",
		))
		return
	}

	// Get balance
	balance, err := h.merchantService.GetMerchantBalance(merchant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(
			"BALANCE_RETRIEVAL_FAILED",
			"Failed to retrieve balance",
		))
		return
	}

	// Build response
	response := APIResponse{
		Data: MerchantBalanceResponse{
			MerchantID:       balance.MerchantID,
			AvailableVND:     balance.AvailableVND,
			PendingVND:       balance.PendingVND,
			TotalReceivedVND: balance.TotalReceivedVND,
			TotalPaidOutVND:  balance.TotalPaidOutVND,
			Currency:         "VND",
			UpdatedAt:        balance.UpdatedAt,
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetTransactions retrieves the merchant's transaction history with pagination
// GET /api/v1/merchant/transactions
func (h *MerchantHandler) GetTransactions(c *gin.Context) {
	// Get merchant from context (set by auth middleware)
	merchantInterface, exists := c.Get("merchant")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse(
			"UNAUTHORIZED",
			"Merchant not authenticated",
		))
		return
	}

	merchant, ok := merchantInterface.(*domain.Merchant)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse(
			"INTERNAL_ERROR",
			"Failed to retrieve merchant information",
		))
		return
	}

	// Parse query parameters
	var req TransactionListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponseWithDetails(
			"INVALID_REQUEST",
			"Invalid query parameters",
			fmt.Sprintf("%v", err),
		))
		return
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20 // Default limit
	}
	if req.OrderBy == "" {
		req.OrderBy = "created_at"
	}
	if req.Order == "" {
		req.Order = "desc"
	}

	// Calculate offset
	offset := (req.Page - 1) * req.Limit

	// Get total count first (for pagination metadata)
	// Note: For MVP, we'll get total from a separate count query
	// In production, this should be optimized
	totalCount, err := h.paymentRepo.CountByMerchant(merchant.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(
			"QUERY_FAILED",
			"Failed to retrieve transaction count",
		))
		return
	}

	// Get payments
	payments, err := h.paymentRepo.ListByMerchant(merchant.ID, req.Limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse(
			"QUERY_FAILED",
			"Failed to retrieve transactions",
		))
		return
	}

	// Map payments to transaction items
	transactions := make([]TransactionItem, 0, len(payments))
	for _, p := range payments {
		item := TransactionItem{
			PaymentID:    p.ID,
			AmountVND:    p.AmountVND,
			AmountCrypto: p.AmountCrypto,
			Token:        p.Currency,
			Chain:        string(p.Chain),
			Status:       string(p.Status),
			CreatedAt:    p.CreatedAt,
			UpdatedAt:    p.UpdatedAt,
		}

		// Add optional fields
		if p.OrderID.Valid {
			orderID := p.OrderID.String
			item.OrderID = &orderID
		}
		if p.TxHash.Valid {
			txHash := p.TxHash.String
			item.TxHash = &txHash
		}
		if p.ConfirmedAt.Valid {
			completedAt := p.ConfirmedAt.Time
			item.CompletedAt = &completedAt
		}

		transactions = append(transactions, item)
	}

	// Calculate pagination metadata
	totalPages := int(math.Ceil(float64(totalCount) / float64(req.Limit)))
	hasNext := req.Page < totalPages
	hasPrev := req.Page > 1

	// Build response
	response := APIResponse{
		Data: TransactionListResponse{
			Transactions: transactions,
			Pagination: TransactionPaginationMeta{
				CurrentPage: req.Page,
				TotalPages:  totalPages,
				TotalItems:  totalCount,
				Limit:       req.Limit,
				HasNext:     hasNext,
				HasPrev:     hasPrev,
			},
		},
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetProfile retrieves the merchant's profile information
// GET /api/v1/merchant/profile
func (h *MerchantHandler) GetProfile(c *gin.Context) {
	// Get merchant from context (set by auth middleware)
	merchantInterface, exists := c.Get("merchant")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse(
			"UNAUTHORIZED",
			"Merchant not authenticated",
		))
		return
	}

	merchant, ok := merchantInterface.(*domain.Merchant)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse(
			"INTERNAL_ERROR",
			"Failed to retrieve merchant information",
		))
		return
	}

	// Build response
	profile := MerchantProfileResponse{
		MerchantID:   merchant.ID,
		Email:        merchant.Email,
		BusinessName: merchant.BusinessName,
		OwnerName:    merchant.OwnerFullName,
		KYCStatus:    string(merchant.KYCStatus),
		Status:       string(merchant.Status),
		CreatedAt:    merchant.CreatedAt,
	}

	// Add optional fields
	if merchant.BusinessTaxID.Valid {
		profile.TaxID = merchant.BusinessTaxID.String
	}
	if merchant.PhoneNumber.Valid {
		profile.PhoneNumber = merchant.PhoneNumber.String
	}
	if merchant.BusinessAddress.Valid {
		profile.BusinessAddress = merchant.BusinessAddress.String
	}
	if merchant.WebhookURL.Valid {
		webhookURL := merchant.WebhookURL.String
		profile.WebhookURL = &webhookURL
	}
	if merchant.KYCApprovedAt.Valid {
		approvedAt := merchant.KYCApprovedAt.Time
		profile.ApprovedAt = &approvedAt
	}

	// Note: We don't include business description and website URL in the model yet
	// These would need to be added to the merchant model
	profile.BusinessDescription = "" // Placeholder
	profile.WebsiteURL = nil         // Placeholder

	response := APIResponse{
		Data:      profile,
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}
