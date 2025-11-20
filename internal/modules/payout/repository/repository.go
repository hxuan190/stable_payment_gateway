package repository

import "github.com/hxuan190/stable_payment_gateway/internal/model"

type Repository interface {
	Create(payout *model.Payout) error
	GetByID(id string) (*model.Payout, error)
	Update(payout *model.Payout) error
	UpdateStatus(id string, status model.PayoutStatus) error
	ListByMerchant(merchantID string, limit, offset int) ([]*model.Payout, error)
	ListPending() ([]*model.Payout, error)
	Count() (int64, error)
	CountByMerchant(merchantID string) (int64, error)
	Delete(id string) error
}
