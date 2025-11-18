package payment

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/handler"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/service"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/interfaces"
)

// Module encapsulates the payment domain module
type Module struct {
	Service    service.Service
	Repository repository.Repository
	Handler    *handler.Handler
	eventBus   events.EventBus
	logger     *logrus.Logger
}

// Config holds the configuration for the payment module
type Config struct {
	DB                *sql.DB
	Cache             *redis.Client
	EventBus          events.EventBus
	Logger            *logrus.Logger
	MerchantReader    interfaces.MerchantReader
	ComplianceChecker interfaces.ComplianceChecker
}

// NewModule creates and initializes the payment module
func NewModule(cfg Config) (*Module, error) {
	// Initialize repository layer
	repo := repository.NewPostgresRepository(cfg.DB, cfg.Cache)

	// Initialize service layer
	svc := service.NewService(service.ServiceConfig{
		Repository:        repo,
		MerchantReader:    cfg.MerchantReader,
		ComplianceChecker: cfg.ComplianceChecker,
		EventBus:          cfg.EventBus,
		Logger:            cfg.Logger,
	})

	// Initialize HTTP handler layer
	hdlr := handler.NewHandler(svc, cfg.Logger)

	// Initialize event subscribers (for handling events from other modules)
	// subscriber := events.NewSubscriber(svc, cfg.Logger)
	// subscriber.Subscribe(cfg.EventBus)

	cfg.Logger.Info("Payment module initialized")

	return &Module{
		Service:    svc,
		Repository: repo,
		Handler:    hdlr,
		eventBus:   cfg.EventBus,
		logger:     cfg.Logger,
	}, nil
}

// RegisterRoutes registers HTTP routes for the payment module
func (m *Module) RegisterRoutes(router *gin.RouterGroup) {
	m.Handler.RegisterRoutes(router)
	m.logger.Info("Payment module routes registered")
}

// Shutdown gracefully shuts down the payment module
func (m *Module) Shutdown() error {
	m.logger.Info("Shutting down payment module")
	// Cleanup resources if needed
	return nil
}
