package legacy

import (
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
)

type PaymentRepositoryLegacyAdapter struct {
	newRepo domain.PaymentRepository
}

func NewPaymentRepositoryLegacyAdapter(newRepo domain.PaymentRepository) *PaymentRepositoryLegacyAdapter {
	return &PaymentRepositoryLegacyAdapter{newRepo: newRepo}
}

func (a *PaymentRepositoryLegacyAdapter) Create(payment *model.Payment) error {
	domainPayment := modelToDomainPayment(payment)
	return a.newRepo.Create(domainPayment)
}

func (a *PaymentRepositoryLegacyAdapter) GetByID(id string) (*model.Payment, error) {
	domainPayment, err := a.newRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return domainToModelPayment(domainPayment), nil
}

func (a *PaymentRepositoryLegacyAdapter) GetByTxHash(txHash string) (*model.Payment, error) {
	domainPayment, err := a.newRepo.GetByTxHash(txHash)
	if err != nil {
		return nil, err
	}
	return domainToModelPayment(domainPayment), nil
}

func (a *PaymentRepositoryLegacyAdapter) GetByPaymentReference(reference string) (*model.Payment, error) {
	domainPayment, err := a.newRepo.GetByPaymentReference(reference)
	if err != nil {
		return nil, err
	}
	return domainToModelPayment(domainPayment), nil
}

func (a *PaymentRepositoryLegacyAdapter) UpdateStatus(id string, status model.PaymentStatus) error {
	domainStatus := domain.PaymentStatus(status)
	return a.newRepo.UpdateStatus(id, domainStatus)
}

func (a *PaymentRepositoryLegacyAdapter) Update(payment *model.Payment) error {
	domainPayment := modelToDomainPayment(payment)
	return a.newRepo.Update(domainPayment)
}

func (a *PaymentRepositoryLegacyAdapter) ListByMerchant(merchantID string, limit, offset int) ([]*model.Payment, error) {
	domainPayments, err := a.newRepo.ListByMerchant(merchantID, limit, offset)
	if err != nil {
		return nil, err
	}
	payments := make([]*model.Payment, len(domainPayments))
	for i, dp := range domainPayments {
		payments[i] = domainToModelPayment(dp)
	}
	return payments, nil
}

func (a *PaymentRepositoryLegacyAdapter) GetExpiredPayments() ([]*model.Payment, error) {
	domainPayments, err := a.newRepo.GetExpiredPayments()
	if err != nil {
		return nil, err
	}
	payments := make([]*model.Payment, len(domainPayments))
	for i, dp := range domainPayments {
		payments[i] = domainToModelPayment(dp)
	}
	return payments, nil
}

func (a *PaymentRepositoryLegacyAdapter) CountByAddressAndWindow(fromAddress string, window time.Duration) (int64, error) {
	return a.newRepo.CountByAddressAndWindow(fromAddress, window)
}

func modelToDomainPayment(p *model.Payment) *domain.Payment {
	return &domain.Payment{
		ID:                p.ID,
		MerchantID:        p.MerchantID,
		AmountVND:         p.AmountVND,
		AmountCrypto:      p.AmountCrypto,
		Currency:          p.Currency,
		Chain:             domain.Chain(p.Chain),
		ExchangeRate:      p.ExchangeRate,
		DestinationWallet: p.DestinationWallet,
		PaymentReference:  p.PaymentReference,
		Status:            domain.PaymentStatus(p.Status),
		TxHash:            p.TxHash,
		FromAddress:       p.FromAddress,
		TxConfirmations:   p.TxConfirmations,
		OrderID:           p.OrderID,
		Description:       p.Description,
		CallbackURL:       p.CallbackURL,
		FeeVND:            p.FeeVND,
		NetAmountVND:      p.NetAmountVND,
		ExpiresAt:         p.ExpiresAt,
		ConfirmedAt:       p.ConfirmedAt,
		FailureReason:     p.FailureReason,
		Metadata:          domain.JSONBMap(p.Metadata),
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}

func domainToModelPayment(p *domain.Payment) *model.Payment {
	return &model.Payment{
		ID:                p.ID,
		MerchantID:        p.MerchantID,
		AmountVND:         p.AmountVND,
		AmountCrypto:      p.AmountCrypto,
		Currency:          p.Currency,
		Chain:             model.Chain(p.Chain),
		ExchangeRate:      p.ExchangeRate,
		DestinationWallet: p.DestinationWallet,
		PaymentReference:  p.PaymentReference,
		Status:            model.PaymentStatus(p.Status),
		TxHash:            p.TxHash,
		FromAddress:       p.FromAddress,
		TxConfirmations:   p.TxConfirmations,
		OrderID:           p.OrderID,
		Description:       p.Description,
		CallbackURL:       p.CallbackURL,
		FeeVND:            p.FeeVND,
		NetAmountVND:      p.NetAmountVND,
		ExpiresAt:         p.ExpiresAt,
		ConfirmedAt:       p.ConfirmedAt,
		FailureReason:     p.FailureReason,
		Metadata:          model.JSONBMap(p.Metadata),
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}
