package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// PaymentRepository defines the interface for payment data access
type PaymentRepository interface {
	Create(payment *Payment) error
	GetByID(id string) (*Payment, error)
	GetByTxHash(txHash string) (*Payment, error)
	GetByPaymentReference(reference string) (*Payment, error)
	UpdateStatus(id string, status PaymentStatus) error
	Update(payment *Payment) error
	ListByMerchant(merchantID string, limit, offset int) ([]*Payment, error)
	GetExpiredPayments() ([]*Payment, error)
	GetComplianceExpiredPayments() ([]*Payment, error)
	ListByStatus(status PaymentStatus, limit, offset int) ([]*Payment, error)
	Count() (int64, error)
	CountByMerchant(merchantID string) (int64, error)
	CountByStatus(status PaymentStatus) (int64, error)
	GetTotalVolumeByMerchant(merchantID string) (decimal.Decimal, error)
	CountByAddressAndWindow(fromAddress string, window time.Duration) (int64, error)
	Delete(id string) error
}

// MerchantRepository defines the interface for merchant data access (port for payment module)
type MerchantRepository interface {
	// Note: In a strict Hexagonal Architecture, this should probably return a minimal Merchant struct defined in this module
	// or a generic interface. For now we assume interaction via ID or shared model if absolutely necessary,
	// but ideally, we should define what we need from Merchant here.
	// However, since we are refactoring payment, let's define a local interface for what we need.
	GetMerchantKYCStatus(merchantID string) (string, error)
	UpdateMerchantVolume(merchantID string, amountUSD decimal.Decimal) error
}

// ExchangeRateProvider defines the interface for getting exchange rates
type ExchangeRateProvider interface {
	GetUSDTToVND(ctx context.Context) (decimal.Decimal, error)
	GetUSDCToVND(ctx context.Context) (decimal.Decimal, error)
}

// ComplianceService defines the interface for compliance operations
type ComplianceService interface {
	CheckMonthlyLimit(ctx context.Context, merchantID string, amountUSD decimal.Decimal) error
	ValidatePaymentCompliance(ctx context.Context, payment *Payment, merchant interface{}, fromAddress string) error
}

// AMLService defines the interface for AML operations
type AMLService interface {
	ScreenWallet(ctx context.Context, walletAddress string, chain string) error
}
