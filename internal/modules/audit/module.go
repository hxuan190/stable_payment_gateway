package audit

import (
	"database/sql"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/audit/repository"
	"github.com/sirupsen/logrus"
)

type Module struct {
	Repository *repository.AuditRepository
	logger     *logrus.Logger
}

type Config struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

func NewModule(cfg Config) (*Module, error) {
	repo := repository.NewAuditRepository(cfg.DB)

	cfg.Logger.Info("Audit module initialized")

	return &Module{
		Repository: repo,
		logger:     cfg.Logger,
	}, nil
}

func (m *Module) Shutdown() error {
	m.logger.Info("Shutting down audit module")
	return nil
}
