package merchant

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/service"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
)

type Module struct {
	Service    *service.MerchantService
	Repository *repository.MerchantRepository
	eventBus   events.EventBus
	logger     *logrus.Logger
}

type Config struct {
	GormDB   *gorm.DB
	Cache    *redis.Client
	EventBus events.EventBus
	Logger   *logrus.Logger
}

func NewModule(cfg Config) (*Module, error) {
	repo := repository.NewMerchantRepository(cfg.GormDB)

	svc := service.NewMerchantService(*repo, cfg.GormDB)

	cfg.Logger.Info("Merchant module initialized")

	return &Module{
		Service:    svc,
		Repository: repo,
		eventBus:   cfg.EventBus,
		logger:     cfg.Logger,
	}, nil
}

func (m *Module) RegisterRoutes(router *gin.RouterGroup) {
	m.logger.Info("Merchant module routes registered")
}

func (m *Module) Shutdown() error {
	m.logger.Info("Shutting down merchant module")
	return nil
}
