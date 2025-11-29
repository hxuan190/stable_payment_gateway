package legacy

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/service"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
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
	return a.svc.ValidatePaymentCompliance(ctx, payment, nil, fromAddress)
}
