package repository

import (
	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

type Repository interface {
	Create(merchant *model.Merchant) error
	GetByID(id string) (*model.Merchant, error)
	GetByEmail(email string) (*model.Merchant, error)
	GetByAPIKey(apiKey string) (*model.Merchant, error)
	Update(merchant *model.Merchant) error
	UpdateKYCStatus(id string, status model.KYCStatus) error
	List(limit, offset int) ([]*model.Merchant, error)
	Count() (int64, error)
	Delete(id string) error
}
