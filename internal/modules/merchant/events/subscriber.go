package events

import (
	"github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/service"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
	"github.com/sirupsen/logrus"
)

type Subscriber struct {
	merchantService service.Service
	logger          *logrus.Logger
}

func NewSubscriber(merchantService service.Service, logger *logrus.Logger) *Subscriber {
	return &Subscriber{
		merchantService: merchantService,
		logger:          logger,
	}
}

func (s *Subscriber) Subscribe(bus events.EventBus) {
	s.logger.Info("Merchant module event subscribers registered")
}
