package infrastructure

import (
	"database/sql"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/infrastructure/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/sirupsen/logrus"
)

type Module struct {
	ArchivedRecordRepo           *repository.ArchivedRecordRepository
	NotificationLogRepo          *repository.NotificationLogRepository
	MerchantNotificationPrefRepo *repository.MerchantNotificationPreferenceRepository
	WalletBalanceRepo            *repository.WalletBalanceRepository
	BlockchainTxRepo             *repository.BlockchainTxRepository
	TransactionHashRepo          *repository.TransactionHashRepository
	KYCDocumentRepo              repository.KYCDocumentRepository
	PayoutScheduleRepo           *repository.PayoutScheduleRepository
	ReconciliationRepo           *repository.ReconciliationRepository
	logger                       *logrus.Logger
}

type Config struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

func NewModule(cfg Config) (*Module, error) {
	cfg.Logger.Info("Infrastructure module initialized")

	return &Module{
		ArchivedRecordRepo:           repository.NewArchivedRecordRepository(cfg.DB),
		NotificationLogRepo:          repository.NewNotificationLogRepository(cfg.DB),
		MerchantNotificationPrefRepo: repository.NewMerchantNotificationPreferenceRepository(cfg.DB),
		WalletBalanceRepo:            repository.NewWalletBalanceRepository(cfg.DB),
		BlockchainTxRepo:             repository.NewBlockchainTxRepository(cfg.DB),
		TransactionHashRepo:          repository.NewTransactionHashRepository(cfg.DB),
		KYCDocumentRepo:              repository.NewKYCDocumentRepository(cfg.DB, logger.GetLogger()),
		PayoutScheduleRepo:           repository.NewPayoutScheduleRepository(cfg.DB),
		ReconciliationRepo:           repository.NewReconciliationRepository(cfg.DB),
		logger:                       cfg.Logger,
	}, nil
}

func (m *Module) Shutdown() error {
	m.logger.Info("Shutting down infrastructure module")
	return nil
}
