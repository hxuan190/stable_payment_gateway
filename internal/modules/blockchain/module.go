package blockchain

import (
	"github.com/hxuan190/stable_payment_gateway/internal/modules/blockchain/bsc"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/blockchain/solana"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
	"github.com/sirupsen/logrus"
)

type Module struct {
	SolanaListener *solana.TransactionListener
	BSCListener    *bsc.TransactionListener
	eventBus       events.EventBus
	logger         *logrus.Logger
}

type Config struct {
	SolanaListener *solana.TransactionListener
	BSCListener    *bsc.TransactionListener
	EventBus       events.EventBus
	Logger         *logrus.Logger
}

func NewModule(cfg Config) (*Module, error) {
	cfg.Logger.Info("Blockchain module initialized")

	return &Module{
		SolanaListener: cfg.SolanaListener,
		BSCListener:    cfg.BSCListener,
		eventBus:       cfg.EventBus,
		logger:         cfg.Logger,
	}, nil
}

func (m *Module) Shutdown() error {
	m.logger.Info("Shutting down blockchain module")
	if m.SolanaListener != nil {
		m.SolanaListener.Stop()
	}
	if m.BSCListener != nil {
		m.BSCListener.Stop()
	}
	return nil
}
