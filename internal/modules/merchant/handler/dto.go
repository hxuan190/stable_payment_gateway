package handler

import (
	"time"

	"github.com/shopspring/decimal"
)

// RegisterMerchantRequest represents the request body for merchant registration
type RegisterMerchantRequest struct {
	Email                string  `json:"email" binding:"required,email"`
	BusinessName         string  `json:"business_name" binding:"required,min=3,max=200"`
	TaxID                string  `json:"tax_id" binding:"required,min=10,max=20"`
	OwnerName            string  `json:"owner_name" binding:"required,min=3,max=100"`
	PhoneNumber          string  `json:"phone_number" binding:"required,min=10,max=15"`
	BusinessAddress      string  `json:"business_address" binding:"required,min=10,max=500"`
	BankName             string  `json:"bank_name" binding:"required"`
	BankAccountNumber    string  `json:"bank_account_number" binding:"required,min=8,max=20"`
	BankAccountName      string  `json:"bank_account_name" binding:"required"`
	WebhookURL           *string `json:"webhook_url" binding:"omitempty,url"`
	WebsiteURL           *string `json:"website_url" binding:"omitempty,url"`
	BusinessDescription  string  `json:"business_description" binding:"required,min=20,max=1000"`
}

// RegisterMerchantResponse represents the response after merchant registration
type RegisterMerchantResponse struct {
	MerchantID string    `json:"merchant_id"`
	Email      string    `json:"email"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}

// MerchantBalanceResponse represents merchant balance information
type MerchantBalanceResponse struct {
	MerchantID       string          `json:"merchant_id"`
	AvailableVND     decimal.Decimal `json:"available_vnd"`
	PendingVND       decimal.Decimal `json:"pending_vnd"`
	TotalReceivedVND decimal.Decimal `json:"total_received_vnd"`
	TotalPaidOutVND  decimal.Decimal `json:"total_paid_out_vnd"`
	Currency         string          `json:"currency"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// TransactionListRequest represents pagination and filter parameters
type TransactionListRequest struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	Limit     int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Status    string `form:"status" binding:"omitempty,oneof=created pending confirming completed expired failed"`
	StartDate string `form:"start_date" binding:"omitempty"`
	EndDate   string `form:"end_date" binding:"omitempty"`
	OrderBy   string `form:"order_by" binding:"omitempty,oneof=created_at amount_vnd status"`
	Order     string `form:"order" binding:"omitempty,oneof=asc desc"`
}

// TransactionItem represents a single transaction in the list
type TransactionItem struct {
	PaymentID    string          `json:"payment_id"`
	OrderID      *string         `json:"order_id,omitempty"`
	AmountVND    decimal.Decimal `json:"amount_vnd"`
	AmountCrypto decimal.Decimal `json:"amount_crypto"`
	Token        string          `json:"token"`
	Chain        string          `json:"chain"`
	Status       string          `json:"status"`
	TxHash       *string         `json:"tx_hash,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
}

// TransactionListResponse represents paginated transaction list
type TransactionListResponse struct {
	Transactions []TransactionItem          `json:"transactions"`
	Pagination   TransactionPaginationMeta `json:"pagination"`
}

// TransactionPaginationMeta contains pagination metadata for transactions
type TransactionPaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	TotalPages  int   `json:"total_pages"`
	TotalItems  int64 `json:"total_items"`
	Limit       int   `json:"limit"`
	HasNext     bool  `json:"has_next"`
	HasPrev     bool  `json:"has_prev"`
}

// MerchantProfileResponse represents merchant profile information
type MerchantProfileResponse struct {
	MerchantID          string     `json:"merchant_id"`
	Email               string     `json:"email"`
	BusinessName        string     `json:"business_name"`
	TaxID               string     `json:"tax_id"`
	OwnerName           string     `json:"owner_name"`
	PhoneNumber         string     `json:"phone_number"`
	BusinessAddress     string     `json:"business_address"`
	WebhookURL          *string    `json:"webhook_url,omitempty"`
	WebsiteURL          *string    `json:"website_url,omitempty"`
	BusinessDescription string     `json:"business_description"`
	KYCStatus           string     `json:"kyc_status"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"created_at"`
	ApprovedAt          *time.Time `json:"approved_at,omitempty"`
}

// APIKeyResponse represents API key information (only shown once during generation)
type APIKeyResponse struct {
	APIKey    string    `json:"api_key"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
