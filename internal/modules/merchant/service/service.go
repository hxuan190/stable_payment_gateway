package service

import (
	"context"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
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
	RegisterMerchant(ctx context.Context, req RegisterMerchantRequest) (*model.Merchant, error)
	GetMerchantByID(ctx context.Context, id string) (*model.Merchant, error)
	GetMerchantByEmail(ctx context.Context, email string) (*model.Merchant, error)
	GetMerchantByAPIKey(ctx context.Context, apiKey string) (*model.Merchant, error)
	UpdateMerchant(ctx context.Context, id string, req UpdateMerchantRequest) (*model.Merchant, error)
	UpdateKYCStatus(ctx context.Context, id string, status model.KYCStatus, reviewNotes string) error
	ListMerchants(ctx context.Context, limit, offset int) ([]*model.Merchant, error)
	GenerateAPIKey(ctx context.Context, merchantID string) (string, error)
	ValidateAPIKey(ctx context.Context, apiKey string) (*model.Merchant, error)
}
