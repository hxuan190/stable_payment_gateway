package payout

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/payout/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/payout/service"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
)

type Module struct {
	Service    *service.PayoutService
	Repository *repository.PayoutRepository
	eventBus   events.EventBus
	logger     *logrus.Logger
}

type Config struct {
	DB       *sql.DB
	Cache    *redis.Client
	EventBus events.EventBus
	Logger   *logrus.Logger
}

func NewModule(cfg Config) (*Module, error) {
	repo := repository.NewPayoutRepository(cfg.DB)
	svc := service.NewPayoutService(*repo, cfg.DB)

	cfg.Logger.Info("Payout module initialized")

	return &Module{
		Service:    svc,
		Repository: repo,
		eventBus:   cfg.EventBus,
		logger:     cfg.Logger,
	}, nil
}

func (m *Module) RegisterRoutes(router *gin.RouterGroup) {
	m.logger.Info("Payout module routes registered")
}

func (m *Module) Shutdown() error {
	m.logger.Info("Shutting down payout module")
	return nil
}
