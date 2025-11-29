package service

import (
	"context"

	paymentDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
	payoutDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payout/domain"
	"github.com/shopspring/decimal"
)

type CreatePayoutRequest struct {
	MerchantID  string
	Amount      decimal.Decimal
	Currency    string
	Chain       paymentDomain.Chain
	ToAddress   string
	Description string
}

type Service interface {
	CreatePayout(ctx context.Context, req CreatePayoutRequest) (*payoutDomain.Payout, error)
	GetPayoutByID(ctx context.Context, id string) (*payoutDomain.Payout, error)
	ListPayoutsByMerchant(ctx context.Context, merchantID string, limit, offset int) ([]*payoutDomain.Payout, error)
	ApprovePayout(ctx context.Context, id string, approverID string) error
	RejectPayout(ctx context.Context, id string, reason string) error
	CompletePayout(ctx context.Context, id string, txHash string) error
	FailPayout(ctx context.Context, id string, reason string) error
}
