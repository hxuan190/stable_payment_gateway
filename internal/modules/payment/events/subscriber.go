package events

import (
	"context"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/service"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
	"github.com/sirupsen/logrus"
)

type Subscriber struct {
	paymentService service.Service
	logger         *logrus.Logger
}

func NewSubscriber(paymentService service.Service, logger *logrus.Logger) *Subscriber {
	return &Subscriber{
		paymentService: paymentService,
		logger:         logger,
	}
}

func (s *Subscriber) Subscribe(bus events.EventBus) {
	bus.Subscribe("blockchain.transaction_detected", s.HandleTransactionDetected)
	bus.Subscribe("compliance.transaction_approved", s.HandleComplianceApproved)
	bus.Subscribe("compliance.transaction_rejected", s.HandleComplianceRejected)
}

func (s *Subscriber) HandleTransactionDetected(ctx context.Context, event events.Event) error {
	s.logger.Info("Payment module received blockchain.transaction_detected event")
	return nil
}

func (s *Subscriber) HandleComplianceApproved(ctx context.Context, event events.Event) error {
	s.logger.Info("Payment module received compliance.transaction_approved event")
	return nil
}

func (s *Subscriber) HandleComplianceRejected(ctx context.Context, event events.Event) error {
	s.logger.Info("Payment module received compliance.transaction_rejected event")
	return nil
}
