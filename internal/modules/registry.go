package modules

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	// Existing services
	"github.com/hxuan190/stable_payment_gateway/internal/api/handler"
	"github.com/hxuan190/stable_payment_gateway/internal/blockchain/bsc"
	"github.com/hxuan190/stable_payment_gateway/internal/blockchain/solana"
	paymentHandler "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/adapter/http"
	paymentService "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/service"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
)

// ModuleRegistry holds all application modules
// This provides a clean interface for module management and makes it easy
// to extract modules into separate microservices later
type ModuleRegistry struct {
	// Core business modules
	Payment      *PaymentModule
	Merchant     *MerchantModule
	Payout       *PayoutModule
	Blockchain   *BlockchainModule
	Compliance   *ComplianceModule
	Ledger       *LedgerModule
	Notification *NotificationModule

	// Cross-cutting concerns
	eventBus events.EventBus
	logger   *logrus.Logger
}

// PaymentModule encapsulates payment domain
type PaymentModule struct {
	Service *paymentService.PaymentService
	Handler *paymentHandler.PaymentHandler
}

// MerchantModule encapsulates merchant domain
type MerchantModule struct {
	Service *service.MerchantService
	Handler *handler.MerchantHandler
}

// PayoutModule encapsulates payout domain
type PayoutModule struct {
	Service *service.PayoutService
	Handler *handler.PayoutHandler
}

// BlockchainModule encapsulates blockchain operations
type BlockchainModule struct {
	SolanaListener *solana.Listener
	BSCListener    *bsc.Listener
}

// ComplianceModule encapsulates compliance operations
type ComplianceModule struct {
	Service *service.ComplianceService
}

// LedgerModule encapsulates accounting operations
type LedgerModule struct {
	Service *service.LedgerService
}

// NotificationModule encapsulates notification delivery
type NotificationModule struct {
	Service *service.NotificationService
}

// RegistryConfig holds configuration for module registry
type RegistryConfig struct {
	// Services (initialized elsewhere)
	PaymentService      *paymentService.PaymentService
	MerchantService     *service.MerchantService
	PayoutService       *service.PayoutService
	ComplianceService   *service.ComplianceService
	LedgerService       *service.LedgerService
	NotificationService *service.NotificationService

	// Handlers
	PaymentHandler  *paymentHandler.PaymentHandler
	MerchantHandler *handler.MerchantHandler
	PayoutHandler   *handler.PayoutHandler

	// Blockchain
	SolanaListener *solana.Listener
	BSCListener    *bsc.Listener

	// Infrastructure
	EventBus events.EventBus
	Logger   *logrus.Logger
}

// NewRegistry creates a new module registry
func NewRegistry(cfg RegistryConfig) *ModuleRegistry {
	registry := &ModuleRegistry{
		eventBus: cfg.EventBus,
		logger:   cfg.Logger,
	}

	// Initialize modules
	if cfg.PaymentService != nil && cfg.PaymentHandler != nil {
		registry.Payment = &PaymentModule{
			Service: cfg.PaymentService,
			Handler: cfg.PaymentHandler,
		}
		cfg.Logger.Info("Payment module registered")
	}

	if cfg.MerchantService != nil && cfg.MerchantHandler != nil {
		registry.Merchant = &MerchantModule{
			Service: cfg.MerchantService,
			Handler: cfg.MerchantHandler,
		}
		cfg.Logger.Info("Merchant module registered")
	}

	if cfg.PayoutService != nil && cfg.PayoutHandler != nil {
		registry.Payout = &PayoutModule{
			Service: cfg.PayoutService,
			Handler: cfg.PayoutHandler,
		}
		cfg.Logger.Info("Payout module registered")
	}

	if cfg.SolanaListener != nil || cfg.BSCListener != nil {
		registry.Blockchain = &BlockchainModule{
			SolanaListener: cfg.SolanaListener,
			BSCListener:    cfg.BSCListener,
		}
		cfg.Logger.Info("Blockchain module registered")
	}

	if cfg.ComplianceService != nil {
		registry.Compliance = &ComplianceModule{
			Service: cfg.ComplianceService,
		}
		cfg.Logger.Info("Compliance module registered")
	}

	if cfg.LedgerService != nil {
		registry.Ledger = &LedgerModule{
			Service: cfg.LedgerService,
		}
		cfg.Logger.Info("Ledger module registered")
	}

	if cfg.NotificationService != nil {
		registry.Notification = &NotificationModule{
			Service: cfg.NotificationService,
		}
		cfg.Logger.Info("Notification module registered")
	}

	// Setup inter-module event subscriptions
	registry.setupEventSubscriptions()

	cfg.Logger.Info("Module registry initialized successfully")
	return registry
}

// setupEventSubscriptions configures event-driven communication between modules
func (r *ModuleRegistry) setupEventSubscriptions() {
	// When payment is confirmed, notify ledger and notification modules
	r.eventBus.Subscribe("payment.confirmed", func(ctx context.Context, event events.Event) error {
		r.logger.Info("Processing payment.confirmed event")

		// Update ledger (when ledger module supports events)
		if r.Ledger != nil {
			r.logger.Debug("Ledger module will process payment confirmation")
			// TODO: r.Ledger.Service.HandlePaymentConfirmed(ctx, event)
		}

		// Send notification (when notification module supports events)
		if r.Notification != nil {
			r.logger.Debug("Notification module will send webhook")
			// TODO: r.Notification.Service.HandlePaymentConfirmed(ctx, event)
		}

		return nil
	})

	// When merchant KYC is approved, enable payment acceptance
	r.eventBus.Subscribe("merchant.kyc_approved", func(ctx context.Context, event events.Event) error {
		r.logger.Info("Processing merchant.kyc_approved event")
		// Future: Update merchant permissions
		return nil
	})

	// When payout is completed, update ledger
	r.eventBus.Subscribe("payout.completed", func(ctx context.Context, event events.Event) error {
		r.logger.Info("Processing payout.completed event")
		if r.Ledger != nil {
			r.logger.Debug("Ledger module will record payout")
			// TODO: r.Ledger.Service.HandlePayoutCompleted(ctx, event)
		}
		return nil
	})

	r.logger.Info("Event subscriptions configured")
}

// RegisterHTTPRoutes registers all module HTTP routes
func (r *ModuleRegistry) RegisterHTTPRoutes(router *gin.Engine) {
	// Routes are already registered in existing setup
	// This method exists for future modular route registration
	r.logger.Info("HTTP routes registered via module registry")
}

// Shutdown gracefully shuts down all modules
func (r *ModuleRegistry) Shutdown(ctx context.Context) error {
	r.logger.Info("Shutting down all modules...")

	// Shutdown blockchain listeners first (stop accepting new transactions)
	if r.Blockchain != nil {
		if r.Blockchain.SolanaListener != nil {
			r.logger.Info("Stopping Solana listener...")
			r.Blockchain.SolanaListener.Stop()
		}
		if r.Blockchain.BSCListener != nil {
			r.logger.Info("Stopping BSC listener...")
			r.Blockchain.BSCListener.Stop()
		}
	}

	// Shutdown event bus (wait for in-flight events)
	if r.eventBus != nil {
		r.logger.Info("Shutting down event bus...")
		if err := r.eventBus.Shutdown(ctx); err != nil {
			r.logger.WithError(err).Error("Error shutting down event bus")
		}
	}

	r.logger.Info("All modules shut down successfully")
	return nil
}

// GetModuleStatus returns status of all modules
func (r *ModuleRegistry) GetModuleStatus() map[string]bool {
	return map[string]bool{
		"payment":      r.Payment != nil,
		"merchant":     r.Merchant != nil,
		"payout":       r.Payout != nil,
		"blockchain":   r.Blockchain != nil,
		"compliance":   r.Compliance != nil,
		"ledger":       r.Ledger != nil,
		"notification": r.Notification != nil,
	}
}
