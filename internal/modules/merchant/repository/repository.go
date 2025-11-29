package repository

import (
	"github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/domain"
)

type Repository interface {
	Create(merchant *domain.Merchant) error
	GetByID(id string) (*domain.Merchant, error)
	GetByEmail(email string) (*domain.Merchant, error)
	GetByAPIKey(apiKey string) (*domain.Merchant, error)
	Update(merchant *domain.Merchant) error
	List(limit, offset int) ([]*domain.Merchant, error)
	ListByKYCStatus(status string, limit, offset int) ([]*domain.Merchant, error)
	Count() (int64, error)
	Delete(id string) error
}
