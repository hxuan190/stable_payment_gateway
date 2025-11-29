package repository

import payoutDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payout/domain"

type Repository interface {
	Create(payout *payoutDomain.Payout) error
	GetByID(id string) (*payoutDomain.Payout, error)
	Update(payout *payoutDomain.Payout) error
	UpdateStatus(id string, status payoutDomain.PayoutStatus) error
	ListByMerchant(merchantID string, limit, offset int) ([]*payoutDomain.Payout, error)
	ListPending() ([]*payoutDomain.Payout, error)
	Count() (int64, error)
	CountByMerchant(merchantID string) (int64, error)
	Delete(id string) error
}
