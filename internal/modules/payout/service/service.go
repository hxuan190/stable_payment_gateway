package service

import (
	"context"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/shopspring/decimal"
)

type CreatePayoutRequest struct {
	MerchantID  string
	Amount      decimal.Decimal
	Currency    string
	Chain       model.Chain
	ToAddress   string
	Description string
}

type Service interface {
	CreatePayout(ctx context.Context, req CreatePayoutRequest) (*model.Payout, error)
	GetPayoutByID(ctx context.Context, id string) (*model.Payout, error)
	ListPayoutsByMerchant(ctx context.Context, merchantID string, limit, offset int) ([]*model.Payout, error)
	ApprovePayout(ctx context.Context, id string, approverID string) error
	RejectPayout(ctx context.Context, id string, reason string) error
	CompletePayout(ctx context.Context, id string, txHash string) error
	FailPayout(ctx context.Context, id string, reason string) error
}
