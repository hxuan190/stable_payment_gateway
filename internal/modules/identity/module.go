package identity

import (
	"database/sql"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/identity/repository"
	"github.com/sirupsen/logrus"
)

type Module struct {
	UserRepo                  *repository.UserRepository
	WalletIdentityMappingRepo *repository.WalletIdentityMappingRepository
	logger                    *logrus.Logger
}

type Config struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

func NewModule(cfg Config) (*Module, error) {
	userRepo := repository.NewUserRepository(cfg.DB)
	walletMappingRepo := repository.NewWalletIdentityMappingRepository(cfg.DB)

	cfg.Logger.Info("Identity module initialized")

	return &Module{
		UserRepo:                  userRepo,
		WalletIdentityMappingRepo: walletMappingRepo,
		logger:                    cfg.Logger,
	}, nil
}

func (m *Module) Shutdown() error {
	m.logger.Info("Shutting down identity module")
	return nil
}
