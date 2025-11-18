package payment

import (
	"context"
	"time"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/interfaces"
	"github.com/sirupsen/logrus"
)

// Service wraps the existing PaymentService and adds modular capabilities
type Service struct {
	*service.PaymentService
	eventBus events.Publisher
	logger   *logrus.Logger
}

// Ensure Service implements interfaces.PaymentReader
var _ interfaces.PaymentReader = (*Service)(nil)

// NewService creates a payment service with event capabilities
func NewService(paymentSvc *service.PaymentService, eventBus events.Publisher, logger *logrus.Logger) *Service {
	return &Service{
		PaymentService: paymentSvc,
		eventBus:       eventBus,
		logger:         logger,
	}
}

// Implement interfaces.PaymentReader

func (s *Service) GetByID(ctx context.Context, id string) (*interfaces.PaymentInfo, error) {
	payment, err := s.PaymentService.GetPaymentStatus(ctx, id)
	if err != nil {
		return nil, err
	}
	return modelToPaymentInfo(payment), nil
}

func (s *Service) GetByMemo(ctx context.Context, memo string) (*interfaces.PaymentInfo, error) {
	payment, err := s.PaymentService.GetPaymentByReference(ctx, memo)
	if err != nil {
		return nil, err
	}
	return modelToPaymentInfo(payment), nil
}

func (s *Service) GetByTxHash(ctx context.Context, txHash string) (*interfaces.PaymentInfo, error) {
	// This requires adding the method to the existing service
	// For now, return not implemented
	s.logger.Warn("GetByTxHash not yet implemented in legacy service")
	return nil, nil
}

func (s *Service) ListByMerchant(ctx context.Context, merchantID string, limit, offset int) ([]*interfaces.PaymentInfo, error) {
	payments, err := s.PaymentService.ListPaymentsByMerchant(ctx, merchantID, limit, offset)
	if err != nil {
		return nil, err
	}

	result := make([]*interfaces.PaymentInfo, len(payments))
	for i, p := range payments {
		result[i] = modelToPaymentInfo(p)
	}
	return result, nil
}

func (s *Service) IsExpired(ctx context.Context, paymentID string) (bool, error) {
	payment, err := s.PaymentService.GetPaymentStatus(ctx, paymentID)
	if err != nil {
		return false, err
	}
	return payment.IsExpired(), nil
}

// Helper function to convert model to interface type
func modelToPaymentInfo(p *model.Payment) *interfaces.PaymentInfo {
	var confirmedAt *time.Time
	if p.ConfirmedAt.Valid {
		confirmedAt = &p.ConfirmedAt.Time
	}

	return &interfaces.PaymentInfo{
		ID:             p.ID,
		MerchantID:     p.MerchantID,
		AmountVND:      p.AmountVND,
		AmountCrypto:   p.AmountCrypto,
		Chain:          string(p.Chain),
		Token:          p.Currency,
		Status:         string(p.Status),
		WalletAddress:  p.DestinationWallet,
		TxHash:         p.GetTxHash(),
		Memo:           p.PaymentReference,
		ExchangeRate:   p.ExchangeRate,
		ExpiresAt:      p.ExpiresAt,
		ConfirmedAt:    confirmedAt,
		CreatedAt:      p.CreatedAt,
	}
}
