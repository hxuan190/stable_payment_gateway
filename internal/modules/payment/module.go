package payment

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	paymenthttp "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/adapter/http"
	paymentrepo "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/adapter/repository"
	paymentdomain "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/domain"
	paymentservice "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/service"
	"github.com/hxuan190/stable_payment_gateway/internal/shared/events"
)

// Module encapsulates the payment domain
type Module struct {
	Service    *paymentservice.PaymentService
	Repository paymentdomain.PaymentRepository
	Handler    *paymenthttp.PaymentHandler
	eventBus   events.EventBus
	logger     *logrus.Logger
}

// Config holds configuration for payment module initialization
type Config struct {
	DB          *sql.DB
	RedisClient *redis.Client
	EventBus    events.EventBus
	Logger      *logrus.Logger

	// Dependencies (injected as interfaces)
	MerchantReader       paymentdomain.MerchantRepository
	ExchangeRateProvider paymentdomain.ExchangeRateProvider
	ComplianceChecker    paymentdomain.ComplianceService
	AMLService           paymentdomain.AMLService

	// Service configuration
	DefaultChain    string
	DefaultCurrency string
	WalletAddress   string
	FeePercentage   float64
	ExpiryMinutes   int
	BaseURL         string // For QR code generation
}

// NewModule creates and initializes the payment module
func NewModule(cfg Config) (*Module, error) {
	// Validate required dependencies
	if cfg.DB == nil {
		return nil, ErrMissingDependency("database")
	}
	if cfg.Logger == nil {
		return nil, ErrMissingDependency("logger")
	}
	if cfg.MerchantReader == nil {
		return nil, ErrMissingDependency("merchant reader")
	}
	if cfg.ExchangeRateProvider == nil {
		return nil, ErrMissingDependency("exchange rate provider")
	}

	// Initialize repository (adapter layer)
	repository := paymentrepo.NewPostgresPaymentRepository(cfg.DB)

	// Initialize service (core business logic)
	serviceConfig := paymentservice.PaymentServiceConfig{
		DefaultChain:    paymentdomain.Chain(cfg.DefaultChain),
		DefaultCurrency: cfg.DefaultCurrency,
		WalletAddress:   cfg.WalletAddress,
		FeePercentage:   cfg.FeePercentage,
		ExpiryMinutes:   cfg.ExpiryMinutes,
		RedisClient:     cfg.RedisClient,
	}

	service := paymentservice.NewPaymentService(
		repository,
		cfg.MerchantReader,
		cfg.ExchangeRateProvider,
		cfg.ComplianceChecker,
		cfg.AMLService,
		serviceConfig,
		cfg.Logger,
	)

	// Initialize HTTP handler (adapter layer)
	handler := paymenthttp.NewPaymentHandler(
		service,
		nil, // ComplianceChecker - optional for now
		nil, // ExchangeRateHTTPAdapter - optional for now
		cfg.BaseURL,
	)

	module := &Module{
		Service:    service,
		Repository: repository,
		Handler:    handler,
		eventBus:   cfg.EventBus,
		logger:     cfg.Logger,
	}

	// Setup event subscriptions
	module.setupEventSubscriptions()

	cfg.Logger.Info("Payment module initialized successfully")

	return module, nil
}

// setupEventSubscriptions configures event-driven communication
func (m *Module) setupEventSubscriptions() {
	// Subscribe to blockchain transaction events
	if m.eventBus != nil {
		// TODO: Subscribe to blockchain.transaction_detected
		// TODO: Subscribe to compliance.transaction_approved
		// TODO: Subscribe to compliance.transaction_rejected
	}
}

// RegisterRoutes registers HTTP routes for the payment module
func (m *Module) RegisterRoutes(router *gin.RouterGroup) {
	// Routes are registered in the main server.go file
	// This method exists for future modular route registration
	m.logger.Info("Payment module routes registered")
}

// Shutdown gracefully shuts down the payment module
func (m *Module) Shutdown() error {
	m.logger.Info("Shutting down payment module...")
	// Add any cleanup logic here
	return nil
}

// Helper function for dependency validation errors
func ErrMissingDependency(name string) error {
	return &MissingDependencyError{Dependency: name}
}

type MissingDependencyError struct {
	Dependency string
}

func (e *MissingDependencyError) Error() string {
	return "missing required dependency: " + e.Dependency
}
