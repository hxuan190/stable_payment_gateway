package ledger

import (
	"database/sql"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/ledger/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/ledger/service"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
	"github.com/sirupsen/logrus"
)

type Module struct {
	Service    *service.LedgerService
	Repository *repository.LedgerRepository
	eventBus   events.EventBus
	logger     *logrus.Logger
}

type Config struct {
	DB       *sql.DB
	EventBus events.EventBus
	Logger   *logrus.Logger
}

func NewModule(cfg Config) (*Module, error) {
	repo := repository.NewLedgerRepository(cfg.DB)
	svc := service.NewLedgerService(repo, nil, cfg.DB)

	cfg.Logger.Info("Ledger module initialized")

	return &Module{
		Service:    svc,
		Repository: repo,
		eventBus:   cfg.EventBus,
		logger:     cfg.Logger,
	}, nil
}

func (m *Module) Shutdown() error {
	m.logger.Info("Shutting down ledger module")
	return nil
}
