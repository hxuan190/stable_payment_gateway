package compliance

import (
	"github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/service"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
	"github.com/sirupsen/logrus"
)

type Module struct {
	Service  *service.ComplianceService
	eventBus events.EventBus
	logger   *logrus.Logger
}

type Config struct {
	Service  *service.ComplianceService
	EventBus events.EventBus
	Logger   *logrus.Logger
}

func NewModule(cfg Config) (*Module, error) {
	cfg.Logger.Info("Compliance module initialized")

	return &Module{
		Service:  cfg.Service,
		eventBus: cfg.EventBus,
		logger:   cfg.Logger,
	}, nil
}

func (m *Module) Shutdown() error {
	m.logger.Info("Shutting down compliance module")
	return nil
}
