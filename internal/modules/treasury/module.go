package treasury

import (
	"database/sql"

	"github.com/sirupsen/logrus"
)

type Module struct {
	WalletRepo    *TreasuryWalletRepository
	OperationRepo *TreasuryOperationRepository
	logger        *logrus.Logger
}

type Config struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

func NewModule(cfg Config) (*Module, error) {
	walletRepo := NewTreasuryWalletRepository(cfg.DB)
	operationRepo := NewTreasuryOperationRepository(cfg.DB)

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
