package service

import (
	"context"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/domain"
)

type RegisterMerchantRequest struct {
	Email                 string
	BusinessName          string
	BusinessTaxID         string
	BusinessLicenseNumber string
	OwnerFullName         string
	OwnerIDCard           string
	PhoneNumber           string
	BusinessAddress       string
	City                  string
	District              string
	BankAccountName       string
	BankAccountNumber     string
	BankName              string
	BankBranch            string
}

type UpdateMerchantRequest struct {
	BusinessName          string
	BusinessTaxID         string
	BusinessLicenseNumber string
	OwnerFullName         string
	OwnerIDCard           string
	PhoneNumber           string
	BusinessAddress       string
	City                  string
	District              string
	BankAccountName       string
	BankAccountNumber     string
	BankName              string
	BankBranch            string
}

type Service interface {
	RegisterMerchant(ctx context.Context, req RegisterMerchantRequest) (*domain.Merchant, error)
	GetMerchantByID(ctx context.Context, id string) (*domain.Merchant, error)
	GetMerchantByEmail(ctx context.Context, email string) (*domain.Merchant, error)
	GetMerchantByAPIKey(ctx context.Context, apiKey string) (*domain.Merchant, error)
	UpdateMerchant(ctx context.Context, id string, req UpdateMerchantRequest) (*domain.Merchant, error)
	UpdateKYCStatus(ctx context.Context, id string, status domain.KYCStatus, reviewNotes string) error
	ListMerchants(ctx context.Context, limit, offset int) ([]*domain.Merchant, error)
	GenerateAPIKey(ctx context.Context, merchantID string) (string, error)
	ValidateAPIKey(ctx context.Context, apiKey string) (*domain.Merchant, error)
}
