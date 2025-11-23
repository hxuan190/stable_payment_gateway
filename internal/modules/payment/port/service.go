package port

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
)

// CreatePaymentRequest contains parameters for creating a payment
type CreatePaymentRequest struct {
	MerchantID  string
	AmountVND   decimal.Decimal
	Currency    string       // USDT, USDC, BUSD
	Chain       domain.Chain // solana, bsc
	OrderID     string
	Description string
	CallbackURL string
	// Proof of Ownership (optional, required for unhosted wallets > $1000)
	FromAddress   string // Solana wallet address (Base58)
	Signature     string // Base64-encoded signature proving wallet ownership
	SignedMessage string // The message that was signed (for verification)
}

// ConfirmPaymentRequest contains parameters for confirming a payment
type ConfirmPaymentRequest struct {
	PaymentID     string
	TxHash        string
	FromAddress   string
	ActualAmount  decimal.Decimal
	Confirmations int32
}

// PaymentService defines the interface for payment business logic (Primary Port)
type PaymentService interface {
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (*domain.Payment, error)
	GetPaymentStatus(ctx context.Context, paymentID string) (*domain.Payment, error)
	GetPaymentByReference(ctx context.Context, reference string) (*domain.Payment, error)
	ValidatePayment(ctx context.Context, paymentID string) error
	ConfirmPayment(ctx context.Context, req ConfirmPaymentRequest) (*domain.Payment, error)
	ExpirePayment(ctx context.Context, paymentID string) error
	FailPayment(ctx context.Context, paymentID, reason string) error
	ListPaymentsByMerchant(ctx context.Context, merchantID string, limit, offset int) ([]*domain.Payment, error)
	GetExpiredPayments(ctx context.Context) ([]*domain.Payment, error)
}
