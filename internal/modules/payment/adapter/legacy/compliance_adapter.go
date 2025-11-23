package legacy

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
)

type ComplianceServiceAdapter struct {
	svc service.ComplianceService
}

func NewComplianceServiceAdapter(svc service.ComplianceService) domain.ComplianceService {
	return &ComplianceServiceAdapter{svc: svc}
}

func (a *ComplianceServiceAdapter) CheckMonthlyLimit(ctx context.Context, merchantID string, amountUSD decimal.Decimal) error {
	return a.svc.CheckMonthlyLimit(ctx, merchantID, amountUSD)
}

func (a *ComplianceServiceAdapter) ValidatePaymentCompliance(ctx context.Context, payment *domain.Payment, merchant interface{}, fromAddress string) error {
	modelPayment := &model.Payment{
		ID:           payment.ID,
		MerchantID:   payment.MerchantID,
		AmountVND:    payment.AmountVND,
		AmountCrypto: payment.AmountCrypto,
		Currency:     payment.Currency,
		Chain:        model.Chain(payment.Chain),
		ExchangeRate: payment.ExchangeRate,
	}
	return a.svc.ValidatePaymentCompliance(ctx, modelPayment, nil, fromAddress)
}
