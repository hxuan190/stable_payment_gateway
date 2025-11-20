package service

import (
	"context"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/interfaces"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type Service interface {
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (*model.Payment, error)
	GetPaymentStatus(ctx context.Context, paymentID string) (*model.Payment, error)
	GetPaymentByReference(ctx context.Context, reference string) (*model.Payment, error)
	ValidatePayment(ctx context.Context, paymentID string) error
	ConfirmPayment(ctx context.Context, req ConfirmPaymentRequest) (*model.Payment, error)
	ExpirePayment(ctx context.Context, paymentID string) error
	FailPayment(ctx context.Context, paymentID, reason string) error
	ListPaymentsByMerchant(ctx context.Context, merchantID string, limit, offset int) ([]*model.Payment, error)
	GetExpiredPayments(ctx context.Context) ([]*model.Payment, error)
}

// ServiceConfig holds configuration for creating the payment service
type ServiceConfig struct {
	Repository        repository.Repository
	MerchantReader    interfaces.MerchantReader
	ComplianceChecker interfaces.ComplianceChecker
	EventBus          events.EventBus
	Logger            *logrus.Logger
}

// NewService creates a new payment service from the modular architecture config
// This is a wrapper around NewPaymentService to match the modular architecture pattern
func NewService(cfg ServiceConfig) Service {
	// Create adapters for interfaces
	merchantAdapter := &merchantReaderAdapter{reader: cfg.MerchantReader}
	complianceAdapter := &complianceCheckerAdapter{checker: cfg.ComplianceChecker}

	// Use a simple exchange rate provider (this should be injected in production)
	exchangeRateProvider := &simpleExchangeRateProvider{}

	// Create payment service config
	paymentConfig := PaymentServiceConfig{
		DefaultChain:    model.ChainSolana,
		DefaultCurrency: "USDT",
		WalletAddress:   "", // Will be set from blockchain config
		FeePercentage:   0.01,
		ExpiryMinutes:   30,
		RedisClient:     nil,
	}

	return NewPaymentService(
		cfg.Repository,
		merchantAdapter,
		exchangeRateProvider,
		complianceAdapter,
		paymentConfig,
		cfg.Logger,
	)
}

// merchantReaderAdapter adapts MerchantReader to MerchantRepository
type merchantReaderAdapter struct {
	reader interfaces.MerchantReader
}

func (a *merchantReaderAdapter) GetByID(id string) (*model.Merchant, error) {
	return a.reader.GetByID(context.Background(), id)
}

func (a *merchantReaderAdapter) Update(merchant *model.Merchant) error {
	// For the payment service, we only need read operations
	// Update operations are not needed in the payment flow
	return nil
}

// complianceCheckerAdapter adapts ComplianceChecker to ComplianceService
type complianceCheckerAdapter struct {
	checker interfaces.ComplianceChecker
}

func (a *complianceCheckerAdapter) CheckMonthlyLimit(ctx context.Context, merchantID string, amountUSD decimal.Decimal) error {
	return a.checker.CheckMonthlyLimit(ctx, merchantID, amountUSD)
}

// simpleExchangeRateProvider provides a simple implementation of exchange rates
// In production, this should be replaced with a real exchange rate service
type simpleExchangeRateProvider struct{}

func (p *simpleExchangeRateProvider) GetUSDTToVND(ctx context.Context) (decimal.Decimal, error) {
	// Hardcoded for now: 1 USDT = 23,000 VND
	return decimal.NewFromInt(23000), nil
}

func (p *simpleExchangeRateProvider) GetUSDCToVND(ctx context.Context) (decimal.Decimal, error) {
	// Hardcoded for now: 1 USDC = 23,000 VND
	return decimal.NewFromInt(23000), nil
}
