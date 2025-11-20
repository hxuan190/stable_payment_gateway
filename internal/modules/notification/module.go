package notification

import (
	"github.com/hxuan190/stable_payment_gateway/internal/modules/notification/service"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
	"github.com/sirupsen/logrus"
)

type Module struct {
	Service  *service.NotificationService
	eventBus events.EventBus
	logger   *logrus.Logger
}

type Config struct {
	Service  *service.NotificationService
	EventBus events.EventBus
	Logger   *logrus.Logger
}

func NewModule(cfg Config) (*Module, error) {
	cfg.Logger.Info("Notification module initialized")

	return &Module{
		Service:  cfg.Service,
		eventBus: cfg.EventBus,
		logger:   cfg.Logger,
	}, nil
}

func (m *Module) Shutdown() error {
	m.logger.Info("Shutting down notification module")
	return nil
}
