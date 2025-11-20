package handler

import (
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/shopspring/decimal"
)

// RequestPayoutRequest represents the request body for creating a payout request
type RequestPayoutRequest struct {
	AmountVND         float64 `json:"amount_vnd" binding:"required,gt=0" validate:"required,gt=0"`
	BankAccountName   string  `json:"bank_account_name" binding:"required,min=2,max=255"`
	BankAccountNumber string  `json:"bank_account_number" binding:"required,min=8,max=50"`
	BankName          string  `json:"bank_name" binding:"required,min=2,max=100"`
	BankBranch        string  `json:"bank_branch,omitempty" validate:"omitempty,max=100"`
	Notes             string  `json:"notes,omitempty" validate:"omitempty,max=500"`
}

// RequestPayoutResponse represents the response after creating a payout request
type RequestPayoutResponse struct {
	PayoutID          string          `json:"payout_id"`
	MerchantID        string          `json:"merchant_id"`
	AmountVND         decimal.Decimal `json:"amount_vnd"`
	FeeVND            decimal.Decimal `json:"fee_vnd"`
	NetAmountVND      decimal.Decimal `json:"net_amount_vnd"`
	BankAccountName   string          `json:"bank_account_name"`
	BankAccountNumber string          `json:"bank_account_number"`
	BankName          string          `json:"bank_name"`
	BankBranch        *string         `json:"bank_branch,omitempty"`
	Status            string          `json:"status"`
	Notes             *string         `json:"notes,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
}

// GetPayoutResponse represents the response when retrieving payout details
type GetPayoutResponse struct {
	PayoutID          string          `json:"payout_id"`
	MerchantID        string          `json:"merchant_id"`
	AmountVND         decimal.Decimal `json:"amount_vnd"`
	FeeVND            decimal.Decimal `json:"fee_vnd"`
	NetAmountVND      decimal.Decimal `json:"net_amount_vnd"`
	BankAccountName   string          `json:"bank_account_name"`
	BankAccountNumber string          `json:"bank_account_number"`
	BankName          string          `json:"bank_name"`
	BankBranch        *string         `json:"bank_branch,omitempty"`
	Status            string          `json:"status"`

	// Approval information
	RequestedBy     string     `json:"requested_by"`
	ApprovedBy      *string    `json:"approved_by,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`

	// Processing information
	ProcessedBy    *string    `json:"processed_by,omitempty"`
	ProcessedAt    *time.Time `json:"processed_at,omitempty"`
	CompletionDate *time.Time `json:"completion_date,omitempty"`

	// Bank transfer reference
	BankReferenceNumber   *string `json:"bank_reference_number,omitempty"`
	TransactionReceiptURL *string `json:"transaction_receipt_url,omitempty"`

	// Failure information
	FailureReason *string `json:"failure_reason,omitempty"`
	RetryCount    int     `json:"retry_count"`

	// Additional info
	Notes     *string   `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListPayoutsRequest represents the request to list payouts with pagination
type ListPayoutsRequest struct {
	Page    int    `form:"page" binding:"omitempty,min=1" validate:"omitempty,min=1"`
	PerPage int    `form:"per_page" binding:"omitempty,min=1,max=100" validate:"omitempty,min=1,max=100"`
	Status  string `form:"status" binding:"omitempty,oneof=requested approved processing completed rejected failed"`
}

// ListPayoutsResponse represents the response when listing payouts
type ListPayoutsResponse struct {
	Payouts    []PayoutListItem `json:"payouts"`
	Pagination *PaginationMeta  `json:"pagination"`
}

// PayoutListItem represents a summary of a payout in a list
type PayoutListItem struct {
	PayoutID          string          `json:"payout_id"`
	AmountVND         decimal.Decimal `json:"amount_vnd"`
	FeeVND            decimal.Decimal `json:"fee_vnd"`
	NetAmountVND      decimal.Decimal `json:"net_amount_vnd"`
	BankAccountName   string          `json:"bank_account_name"`
	BankAccountNumber string          `json:"bank_account_number"`
	BankName          string          `json:"bank_name"`
	Status            string          `json:"status"`
	CreatedAt         time.Time       `json:"created_at"`
	ApprovedAt        *time.Time      `json:"approved_at,omitempty"`
	CompletionDate    *time.Time      `json:"completion_date,omitempty"`
}

// PayoutToResponse converts a model.Payout to GetPayoutResponse
func PayoutToResponse(payout *model.Payout) GetPayoutResponse {
	response := GetPayoutResponse{
		PayoutID:          payout.ID,
		MerchantID:        payout.MerchantID,
		AmountVND:         payout.AmountVND,
		FeeVND:            payout.FeeVND,
		NetAmountVND:      payout.NetAmountVND,
		BankAccountName:   payout.BankAccountName,
		BankAccountNumber: payout.BankAccountNumber,
		BankName:          payout.BankName,
		Status:            string(payout.Status),
		RequestedBy:       payout.RequestedBy,
		RetryCount:        payout.RetryCount,
		CreatedAt:         payout.CreatedAt,
		UpdatedAt:         payout.UpdatedAt,
	}

	// Handle optional fields
	if payout.BankBranch.Valid {
		bankBranch := payout.BankBranch.String
		response.BankBranch = &bankBranch
	}
	if payout.ApprovedBy.Valid {
		approvedBy := payout.ApprovedBy.String
		response.ApprovedBy = &approvedBy
	}
	if payout.ApprovedAt.Valid {
		approvedAt := payout.ApprovedAt.Time
		response.ApprovedAt = &approvedAt
	}
	if payout.RejectionReason.Valid {
		rejectionReason := payout.RejectionReason.String
		response.RejectionReason = &rejectionReason
	}
	if payout.ProcessedBy.Valid {
		processedBy := payout.ProcessedBy.String
		response.ProcessedBy = &processedBy
	}
	if payout.ProcessedAt.Valid {
		processedAt := payout.ProcessedAt.Time
		response.ProcessedAt = &processedAt
	}
	if payout.CompletionDate.Valid {
		completionDate := payout.CompletionDate.Time
		response.CompletionDate = &completionDate
	}
	if payout.BankReferenceNumber.Valid {
		bankReferenceNumber := payout.BankReferenceNumber.String
		response.BankReferenceNumber = &bankReferenceNumber
	}
	if payout.TransactionReceiptURL.Valid {
		transactionReceiptURL := payout.TransactionReceiptURL.String
		response.TransactionReceiptURL = &transactionReceiptURL
	}
	if payout.FailureReason.Valid {
		failureReason := payout.FailureReason.String
		response.FailureReason = &failureReason
	}
	if payout.Notes.Valid {
		notes := payout.Notes.String
		response.Notes = &notes
	}

	return response
}

// PayoutToListItem converts a model.Payout to PayoutListItem
func PayoutToListItem(payout *model.Payout) PayoutListItem {
	item := PayoutListItem{
		PayoutID:          payout.ID,
		AmountVND:         payout.AmountVND,
		FeeVND:            payout.FeeVND,
		NetAmountVND:      payout.NetAmountVND,
		BankAccountName:   payout.BankAccountName,
		BankAccountNumber: payout.BankAccountNumber,
		BankName:          payout.BankName,
		Status:            string(payout.Status),
		CreatedAt:         payout.CreatedAt,
	}

	// Handle optional fields
	if payout.ApprovedAt.Valid {
		approvedAt := payout.ApprovedAt.Time
		item.ApprovedAt = &approvedAt
	}
	if payout.CompletionDate.Valid {
		completionDate := payout.CompletionDate.Time
		item.CompletionDate = &completionDate
	}

	return item
}
