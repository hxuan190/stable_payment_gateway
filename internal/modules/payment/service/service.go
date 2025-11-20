package service

import (
	"context"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
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
