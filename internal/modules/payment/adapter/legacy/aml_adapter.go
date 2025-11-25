package legacy

import (
	"context"

	complianceservice "github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/service"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
)

type AMLServiceAdapter struct {
	svc complianceservice.AMLService
}

func NewAMLServiceAdapter(svc complianceservice.AMLService) domain.AMLService {
	return &AMLServiceAdapter{svc: svc}
}

func (a *AMLServiceAdapter) ScreenWallet(ctx context.Context, walletAddress string, chain string) error {
	_, err := a.svc.ScreenWalletAddress(ctx, walletAddress, chain)
	return err
}
