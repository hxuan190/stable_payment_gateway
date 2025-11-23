package legacy

import (
	"github.com/shopspring/decimal"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
)

type MerchantRepositoryAdapter struct {
	repo *repository.MerchantRepository
}

func NewMerchantRepositoryAdapter(repo *repository.MerchantRepository) domain.MerchantRepository {
	return &MerchantRepositoryAdapter{repo: repo}
}

func (a *MerchantRepositoryAdapter) GetMerchantKYCStatus(merchantID string) (string, error) {
	merchant, err := a.repo.GetByID(merchantID)
	if err != nil {
		return "", err
	}
	return string(merchant.KYCStatus), nil
}

func (a *MerchantRepositoryAdapter) UpdateMerchantVolume(merchantID string, amountUSD decimal.Decimal) error {
	merchant, err := a.repo.GetByID(merchantID)
	if err != nil {
		return err
	}
	merchant.TotalVolumeThisMonthUSD = merchant.TotalVolumeThisMonthUSD.Add(amountUSD)
	return a.repo.Update(merchant)
}
