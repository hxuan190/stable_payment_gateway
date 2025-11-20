package handler

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

// CreatePaymentRequest represents the request to create a new payment
type CreatePaymentRequest struct {
	AmountVND   float64            `json:"amount_vnd" binding:"required,gt=0" validate:"required,gt=0"`
	Currency    string             `json:"currency,omitempty" validate:"omitempty,oneof=USDT USDC BUSD"`
	Chain       string             `json:"chain,omitempty" validate:"omitempty,oneof=solana bsc"`
	OrderID     string             `json:"order_id,omitempty" validate:"omitempty,max=255"`
	Description string             `json:"description,omitempty" validate:"omitempty,max=1000"`
	CallbackURL string             `json:"callback_url,omitempty" validate:"omitempty,url,max=500"`
	TravelRule  *TravelRuleRequest `json:"travel_rule,omitempty"` // Required for transactions > $1000 USD
}

// TravelRuleRequest represents Travel Rule data for high-value transactions (> $1000 USD)
type TravelRuleRequest struct {
	PayerFullName      string `json:"payer_full_name" validate:"required,max=255"`
	PayerWalletAddress string `json:"payer_wallet_address" validate:"required,max=255"`
	PayerCountry       string `json:"payer_country" validate:"required,len=2"` // ISO 3166-1 alpha-2
	PayerIDDocument    string `json:"payer_id_document,omitempty" validate:"omitempty,max=255"`
}

// CreatePaymentResponse represents the response when a payment is created
type CreatePaymentResponse struct {
	PaymentID         string          `json:"payment_id"`
	AmountVND         decimal.Decimal `json:"amount_vnd"`
	AmountCrypto      decimal.Decimal `json:"amount_crypto"`
	Currency          string          `json:"currency"`
	Chain             string          `json:"chain"`
	ExchangeRate      decimal.Decimal `json:"exchange_rate"`
	DestinationWallet string          `json:"destination_wallet"`
	PaymentReference  string          `json:"payment_reference"`
	ExpiresAt         time.Time       `json:"expires_at"`
	FeeVND            decimal.Decimal `json:"fee_vnd"`
	NetAmountVND      decimal.Decimal `json:"net_amount_vnd"`
	Status            string          `json:"status"`
	QRCodeURL         string          `json:"qr_code_url"`
	PaymentURL        string          `json:"payment_url"`
	CreatedAt         time.Time       `json:"created_at"`
}

// GetPaymentResponse represents the response when retrieving payment details
type GetPaymentResponse struct {
	PaymentID         string          `json:"payment_id"`
	MerchantID        string          `json:"merchant_id"`
	AmountVND         decimal.Decimal `json:"amount_vnd"`
	AmountCrypto      decimal.Decimal `json:"amount_crypto"`
	Currency          string          `json:"currency"`
	Chain             string          `json:"chain"`
	ExchangeRate      decimal.Decimal `json:"exchange_rate"`
	DestinationWallet string          `json:"destination_wallet"`
	PaymentReference  string          `json:"payment_reference"`
	Status            string          `json:"status"`

	// Optional fields
	OrderID       *string `json:"order_id,omitempty"`
	Description   *string `json:"description,omitempty"`
	TxHash        *string `json:"tx_hash,omitempty"`
	FromAddress   *string `json:"from_address,omitempty"`
	FailureReason *string `json:"failure_reason,omitempty"`

	// Timing
	ExpiresAt   time.Time  `json:"expires_at"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`

	// Fee information
	FeePercentage decimal.Decimal `json:"fee_percentage"`
	FeeVND        decimal.Decimal `json:"fee_vnd"`
	NetAmountVND  decimal.Decimal `json:"net_amount_vnd"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListPaymentsRequest represents the request to list payments with pagination
type ListPaymentsRequest struct {
	Page    int    `form:"page" binding:"omitempty,min=1" validate:"omitempty,min=1"`
	PerPage int    `form:"per_page" binding:"omitempty,min=1,max=100" validate:"omitempty,min=1,max=100"`
	Status  string `form:"status" binding:"omitempty,oneof=created pending confirming completed expired failed"`
}

// ListPaymentsResponse represents the response when listing payments
type ListPaymentsResponse struct {
	Payments   []PaymentListItem `json:"payments"`
	Pagination *PaginationMeta   `json:"pagination"`
}

// PaymentListItem represents a summary of a payment in a list
type PaymentListItem struct {
	PaymentID    string          `json:"payment_id"`
	AmountVND    decimal.Decimal `json:"amount_vnd"`
	AmountCrypto decimal.Decimal `json:"amount_crypto"`
	Currency     string          `json:"currency"`
	Chain        string          `json:"chain"`
	Status       string          `json:"status"`
	OrderID      *string         `json:"order_id,omitempty"`
	TxHash       *string         `json:"tx_hash,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	ExpiresAt    time.Time       `json:"expires_at"`
	PaidAt       *time.Time      `json:"paid_at,omitempty"`
	ConfirmedAt  *time.Time      `json:"confirmed_at,omitempty"`
}

// PaymentToResponse converts a model.Payment to GetPaymentResponse
func PaymentToResponse(payment *model.Payment) GetPaymentResponse {
	response := GetPaymentResponse{
		PaymentID:         payment.ID,
		MerchantID:        payment.MerchantID,
		AmountVND:         payment.AmountVND,
		AmountCrypto:      payment.AmountCrypto,
		Currency:          payment.Currency,
		Chain:             string(payment.Chain),
		ExchangeRate:      payment.ExchangeRate,
		DestinationWallet: payment.DestinationWallet,
		PaymentReference:  payment.PaymentReference,
		Status:            string(payment.Status),
		ExpiresAt:         payment.ExpiresAt,
		FeePercentage:     payment.FeePercentage,
		FeeVND:            payment.FeeVND,
		NetAmountVND:      payment.NetAmountVND,
		CreatedAt:         payment.CreatedAt,
		UpdatedAt:         payment.UpdatedAt,
	}

	// Handle optional fields
	if payment.OrderID.Valid {
		orderID := payment.OrderID.String
		response.OrderID = &orderID
	}
	if payment.Description.Valid {
		description := payment.Description.String
		response.Description = &description
	}
	if payment.TxHash.Valid {
		txHash := payment.TxHash.String
		response.TxHash = &txHash
	}
	if payment.FromAddress.Valid {
		fromAddress := payment.FromAddress.String
		response.FromAddress = &fromAddress
	}
	if payment.FailureReason.Valid {
		failureReason := payment.FailureReason.String
		response.FailureReason = &failureReason
	}
	if payment.PaidAt.Valid {
		paidAt := payment.PaidAt.Time
		response.PaidAt = &paidAt
	}
	if payment.ConfirmedAt.Valid {
		confirmedAt := payment.ConfirmedAt.Time
		response.ConfirmedAt = &confirmedAt
	}

	return response
}

// PaymentToListItem converts a model.Payment to PaymentListItem
func PaymentToListItem(payment *model.Payment) PaymentListItem {
	item := PaymentListItem{
		PaymentID:    payment.ID,
		AmountVND:    payment.AmountVND,
		AmountCrypto: payment.AmountCrypto,
		Currency:     payment.Currency,
		Chain:        string(payment.Chain),
		Status:       string(payment.Status),
		CreatedAt:    payment.CreatedAt,
		ExpiresAt:    payment.ExpiresAt,
	}

	// Handle optional fields
	if payment.OrderID.Valid {
		orderID := payment.OrderID.String
		item.OrderID = &orderID
	}
	if payment.TxHash.Valid {
		txHash := payment.TxHash.String
		item.TxHash = &txHash
	}
	if payment.PaidAt.Valid {
		paidAt := payment.PaidAt.Time
		item.PaidAt = &paidAt
	}
	if payment.ConfirmedAt.Valid {
		confirmedAt := payment.ConfirmedAt.Time
		item.ConfirmedAt = &confirmedAt
	}

	return item
}

// PaymentStatusResponse represents the public payment status response (no authentication required)
// This DTO is designed for the payer experience layer and excludes merchant-sensitive information
type PaymentStatusResponse struct {
	ID            string          `json:"id"`
	Status        string          `json:"status"`
	AmountCrypto  decimal.Decimal `json:"amount_crypto"`
	AmountVND     decimal.Decimal `json:"amount_vnd"`
	Currency      string          `json:"currency"`
	Chain         string          `json:"chain"`
	WalletAddress string          `json:"wallet_address"`
	PaymentMemo   string          `json:"payment_memo"`
	QRCodeData    string          `json:"qr_code_data"`
	TxHash        *string         `json:"tx_hash,omitempty"`
	Confirmations int             `json:"confirmations"`
	ExpiresAt     time.Time       `json:"expires_at"`
	CreatedAt     time.Time       `json:"created_at"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
}

// PaymentToPublicStatusResponse converts a model.Payment to PaymentStatusResponse (public-safe)
func PaymentToPublicStatusResponse(payment *model.Payment) PaymentStatusResponse {
	response := PaymentStatusResponse{
		ID:            payment.ID,
		Status:        string(payment.Status),
		AmountCrypto:  payment.AmountCrypto,
		AmountVND:     payment.AmountVND,
		Currency:      payment.Currency,
		Chain:         string(payment.Chain),
		WalletAddress: payment.DestinationWallet,
		PaymentMemo:   payment.PaymentReference,
		QRCodeData:    formatQRCodeData(payment),
		Confirmations: 0, // Default to 0, will be updated based on tx status
		ExpiresAt:     payment.ExpiresAt,
		CreatedAt:     payment.CreatedAt,
	}

	// Handle optional fields
	if payment.TxHash.Valid {
		txHash := payment.TxHash.String
		response.TxHash = &txHash
	}
	if payment.ConfirmedAt.Valid {
		confirmedAt := payment.ConfirmedAt.Time
		response.CompletedAt = &confirmedAt
	}

	// Set confirmations based on status
	if payment.Status == model.PaymentStatusCompleted {
		response.Confirmations = 100 // Fully confirmed
	} else if payment.Status == model.PaymentStatusConfirming {
		response.Confirmations = 50 // Partially confirmed
	} else if payment.Status == model.PaymentStatusPending {
		response.Confirmations = 1 // Transaction detected
	}

	return response
}

// formatQRCodeData formats the payment data for QR code generation
// Returns a formatted string that can be used to generate QR codes for different chains
func formatQRCodeData(payment *model.Payment) string {
	switch payment.Chain {
	case model.ChainSolana:
		// Solana Pay format: solana:{address}?amount={amount}&spl-token={mint}&reference={memo}&label={label}
		// For now, return simplified format
		return "solana:" + payment.DestinationWallet + "?amount=" + payment.AmountCrypto.String() + "&reference=" + payment.PaymentReference
	case model.ChainBSC:
		// Ethereum/BSC format
		return "ethereum:" + payment.DestinationWallet + "?value=" + payment.AmountCrypto.String()
	default:
		// Fallback: just return the wallet address
		return payment.DestinationWallet
	}
}
