package events

import (
	"github.com/hxuan190/stable_payment_gateway/internal/modules/payout/service"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
	"github.com/sirupsen/logrus"
)

type Subscriber struct {
	payoutService service.Service
	logger        *logrus.Logger
}

func NewSubscriber(payoutService service.Service, logger *logrus.Logger) *Subscriber {
	return &Subscriber{
		payoutService: payoutService,
		logger:        logger,
	}
}

func (s *Subscriber) Subscribe(bus events.EventBus) {
	s.logger.Info("Payout module event subscribers registered")
}
