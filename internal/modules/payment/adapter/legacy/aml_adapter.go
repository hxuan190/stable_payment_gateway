package legacy

import (
	"context"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
)

type AMLServiceAdapter struct {
	svc service.AMLService
}

func NewAMLServiceAdapter(svc service.AMLService) domain.AMLService {
	return &AMLServiceAdapter{svc: svc}
}

func (a *AMLServiceAdapter) ScreenWallet(ctx context.Context, walletAddress string, chain string) error {
	_, err := a.svc.ScreenWalletAddress(ctx, walletAddress, chain)
	return err
}
