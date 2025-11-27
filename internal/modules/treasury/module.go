package treasury

import (
	"database/sql"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/treasury/repository"
	"github.com/sirupsen/logrus"
)

type Module struct {
	WalletRepo    *repository.TreasuryWalletRepository
	OperationRepo *repository.TreasuryOperationRepository
	logger        *logrus.Logger
}

type Config struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

func NewModule(cfg Config) (*Module, error) {
	walletRepo := repository.NewTreasuryWalletRepository(cfg.DB)
	operationRepo := repository.NewTreasuryOperationRepository(cfg.DB)

	cfg.Logger.Info("Treasury module initialized")

	return &Module{
		WalletRepo:    walletRepo,
		OperationRepo: operationRepo,
		logger:        cfg.Logger,
	}, nil
}

func (m *Module) Shutdown() error {
	m.logger.Info("Shutting down treasury module")
	return nil
}
