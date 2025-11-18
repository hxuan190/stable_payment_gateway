package payment

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	// Use existing code
	"github.com/hxuan190/stable_payment_gateway/internal/api/handler"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
)

// ModuleV2 wraps existing payment components with modular structure
type ModuleV2 struct {
	PaymentService *service.PaymentService
	PaymentHandler *handler.PaymentHandler
	eventBus       events.EventBus
	logger         *logrus.Logger
}

// ConfigV2 holds configuration for payment module using existing components
type ConfigV2 struct {
	PaymentService *service.PaymentService
	PaymentHandler *handler.PaymentHandler
	EventBus       events.EventBus
	Logger         *logrus.Logger
}

// NewModuleV2 creates a payment module using existing services
func NewModuleV2(cfg ConfigV2) *ModuleV2 {
	if cfg.PaymentService == nil {
		cfg.Logger.Fatal("PaymentService is required")
	}
	if cfg.PaymentHandler == nil {
		cfg.Logger.Fatal("PaymentHandler is required")
	}

	cfg.Logger.Info("Payment module initialized (using existing components)")

	return &ModuleV2{
		PaymentService: cfg.PaymentService,
		PaymentHandler: cfg.PaymentHandler,
		eventBus:       cfg.EventBus,
		logger:         cfg.Logger,
	}
}

// RegisterRoutes registers HTTP routes
func (m *ModuleV2) RegisterRoutes(router *gin.Engine) {
	// Payment routes are already registered in the main setup
	// This is a no-op for V2, kept for interface compatibility
	m.logger.Debug("Payment module routes (managed by existing handler)")
}

// Shutdown gracefully shuts down the module
func (m *ModuleV2) Shutdown() error {
	m.logger.Info("Shutting down payment module")
	return nil
}
