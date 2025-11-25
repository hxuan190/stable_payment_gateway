package legacy

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	infrastructureservice "github.com/hxuan190/stable_payment_gateway/internal/modules/infrastructure/service"
)

type ExchangeRateHTTPAdapter struct {
	svc *infrastructureservice.ExchangeRateService
}

func NewExchangeRateHTTPAdapter(svc *infrastructureservice.ExchangeRateService) *ExchangeRateHTTPAdapter {
	return &ExchangeRateHTTPAdapter{svc: svc}
}

func (a *ExchangeRateHTTPAdapter) GetExchangeRate(ctx context.Context, currency string) (decimal.Decimal, error) {
	switch currency {
	case "USDT":
		return a.svc.GetUSDTToVND(ctx)
	case "USDC":
		return a.svc.GetUSDCToVND(ctx)
	default:
		return decimal.Zero, fmt.Errorf("unsupported currency: %s", currency)
	}
}
