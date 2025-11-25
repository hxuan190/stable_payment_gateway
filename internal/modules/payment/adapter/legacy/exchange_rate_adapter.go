package legacy

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	infrastructureservice "github.com/hxuan190/stable_payment_gateway/internal/modules/infrastructure/service"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
)

type ExchangeRateServiceAdapter struct {
	svc *infrastructureservice.ExchangeRateService
}

func NewExchangeRateServiceAdapter(svc *infrastructureservice.ExchangeRateService) domain.ExchangeRateProvider {
	return &ExchangeRateServiceAdapter{svc: svc}
}

func (a *ExchangeRateServiceAdapter) GetUSDTToVND(ctx context.Context) (decimal.Decimal, error) {
	return a.svc.GetUSDTToVND(ctx)
}

func (a *ExchangeRateServiceAdapter) GetUSDCToVND(ctx context.Context) (decimal.Decimal, error) {
	return a.svc.GetUSDCToVND(ctx)
}

func (a *ExchangeRateServiceAdapter) GetExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (decimal.Decimal, error) {
	if fromCurrency == "VND" && toCurrency == "USDT" {
		rate, err := a.svc.GetUSDTToVND(ctx)
		if err != nil {
			return decimal.Zero, err
		}
		return rate, nil
	}
	if fromCurrency == "VND" && toCurrency == "USDC" {
		rate, err := a.svc.GetUSDCToVND(ctx)
		if err != nil {
			return decimal.Zero, err
		}
		return rate, nil
	}
	return decimal.Zero, fmt.Errorf("unsupported currency pair: %s/%s", fromCurrency, toCurrency)
}
