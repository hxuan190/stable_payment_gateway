package worker

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/hxuan190/stable_payment_gateway/internal/blockchain/solana"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/payment/adapter/legacy"
	paymentrepo "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/adapter/repository"
	paymentservice "github.com/hxuan190/stable_payment_gateway/internal/modules/payment/service"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/cache"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/logger"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
	"github.com/hxuan190/stable_payment_gateway/internal/service"
)

// Server represents the worker server
type Server struct {
	server                *asynq.Server
	mux                   *asynq.ServeMux
	scheduler             *asynq.Scheduler
	db                    *sql.DB
	cache                 cache.Cache
	solanaClient          *solana.Client
	solanaWallet          *solana.Wallet
	paymentService        *paymentservice.PaymentService
	notificationSvc       *service.NotificationService
	reconciliationService *service.ReconciliationService
	merchantRepo          repository.MerchantRepository
	paymentRepo           *legacy.PaymentRepositoryLegacyAdapter
	payoutRepo            repository.PayoutRepository
	walletBalanceRepo     repository.WalletBalanceRepository
}

// ServerConfig holds configuration for the worker server
type ServerConfig struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	DB            *sql.DB
	Cache         cache.Cache
	SolanaClient  *solana.Client
	SolanaWallet  *solana.Wallet
	Concurrency   int
	Queues        map[string]int // Queue name to priority mapping
}

// NewServer creates a new worker server instance
func NewServer(cfg *ServerConfig) *Server {
	redisOpts := asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}

	// Default concurrency if not specified
	concurrency := cfg.Concurrency
	if concurrency == 0 {
		concurrency = 10
	}

	// Default queue priorities if not specified
	queues := cfg.Queues
	if queues == nil {
		queues = map[string]int{
			"webhooks":       5, // Highest priority
			"webhooks_retry": 3,
			"periodic":       4,
			"monitoring":     2,
			"reports":        1, // Lowest priority
		}
	}

	// Create asynq server
	srv := asynq.NewServer(
		redisOpts,
		asynq.Config{
			Concurrency: concurrency,
			Queues:      queues,
			// Retry failed tasks with exponential backoff
			RetryDelayFunc: func(n int, err error, task *asynq.Task) time.Duration {
				return time.Duration(1<<uint(n)) * time.Second
			},
			// Log level
			LogLevel: asynq.InfoLevel,
			// Error handler
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				logger.Error("Task processing failed", err, logger.Fields{
					"task_type": task.Type(),
					"payload":   string(task.Payload()),
				})
			}),
		},
	)

	// Create scheduler for periodic tasks
	scheduler := asynq.NewScheduler(
		redisOpts,
		&asynq.SchedulerOpts{
			LogLevel: asynq.InfoLevel,
		},
	)

	// Initialize repositories
	merchantRepo := repository.NewMerchantRepository(cfg.DB)
	payoutRepo := repository.NewPayoutRepository(cfg.DB)
	ledgerRepo := repository.NewLedgerRepository(cfg.DB)
	balanceRepo := repository.NewBalanceRepository(cfg.DB)
	auditRepo := repository.NewAuditRepository(cfg.DB)
	walletBalanceRepo := repository.NewWalletBalanceRepository(cfg.DB)
	reconciliationRepo := repository.NewReconciliationRepository(cfg.DB)

	// Initialize new payment module
	newPaymentRepo := paymentrepo.NewPostgresPaymentRepository(cfg.DB)
	paymentRepo := legacy.NewPaymentRepositoryLegacyAdapter(newPaymentRepo)
	merchantAdapter := legacy.NewMerchantRepositoryAdapter(merchantRepo)

	// Initialize services
	exchangeRateService := service.NewExchangeRateService(cfg.Cache)
	ledgerService := service.NewLedgerService(ledgerRepo, balanceRepo, cfg.DB)
	notificationService := service.NewNotificationService(merchantRepo, auditRepo)
	paymentService := paymentservice.NewPaymentService(
		newPaymentRepo,
		merchantAdapter,
		nil,
		nil,
		nil,
		paymentservice.PaymentServiceConfig{
			DefaultChain:    "solana",
			DefaultCurrency: "USDT",
			WalletAddress:   "",
			FeePercentage:   0.01,
			ExpiryMinutes:   30,
			RedisClient:     nil,
		},
		logger.GetLogger().Logger,
	)
	reconciliationService := service.NewReconciliationService(
		cfg.DB,
		ledgerService,
		exchangeRateService,
		reconciliationRepo,
		balanceRepo,
		logger.NewLogger(),
	)

	server := &Server{
		server:                srv,
		mux:                   asynq.NewServeMux(),
		scheduler:             scheduler,
		db:                    cfg.DB,
		cache:                 cfg.Cache,
		solanaClient:          cfg.SolanaClient,
		solanaWallet:          cfg.SolanaWallet,
		paymentService:        paymentService,
		notificationSvc:       notificationService,
		reconciliationService: reconciliationService,
		merchantRepo:          merchantRepo,
		paymentRepo:           paymentRepo,
		payoutRepo:            payoutRepo,
		walletBalanceRepo:     walletBalanceRepo,
	}

	// Register handlers
	server.registerHandlers()

	// Schedule periodic tasks
	server.scheduleTasks()

	return server
}

// registerHandlers registers all task handlers
func (s *Server) registerHandlers() {
	// Register webhook delivery handler
	s.mux.HandleFunc(TypeWebhookDelivery, s.handleWebhookDelivery)

	// Register payment expiry handler
	s.mux.HandleFunc(TypePaymentExpiry, s.handlePaymentExpiry)

	// Register balance check handler
	s.mux.HandleFunc(TypeBalanceCheck, s.handleBalanceCheck)

	// Register daily settlement report handler
	s.mux.HandleFunc(TypeDailySettlementReport, s.handleDailySettlementReport)

	// Register daily reconciliation handler
	s.mux.HandleFunc(TypeDailyReconciliation, s.handleDailyReconciliation)

	logger.Info("Worker handlers registered", logger.Fields{
		"handlers": []string{
			TypeWebhookDelivery,
			TypePaymentExpiry,
			TypeBalanceCheck,
			TypeDailySettlementReport,
			TypeDailyReconciliation,
		},
	})
}

// scheduleTasks schedules periodic tasks
func (s *Server) scheduleTasks() {
	// Schedule payment expiry check every 5 minutes
	_, err := s.scheduler.Register(
		"*/5 * * * *", // Every 5 minutes
		asynq.NewTask(TypePaymentExpiry, []byte(`{}`)),
		asynq.Queue("periodic"),
	)
	if err != nil {
		logger.Error("Failed to schedule payment expiry task", err)
	} else {
		logger.Info("Scheduled payment expiry task", logger.Fields{
			"schedule": "every 5 minutes",
		})
	}

	// Schedule balance check every 5 minutes
	_, err = s.scheduler.Register(
		"*/5 * * * *", // Every 5 minutes
		asynq.NewTask(TypeBalanceCheck, []byte(`{}`)),
		asynq.Queue("monitoring"),
	)
	if err != nil {
		logger.Error("Failed to schedule balance check task", err)
	} else {
		logger.Info("Scheduled balance check task", logger.Fields{
			"schedule": "every 5 minutes",
		})
	}

	// Schedule daily settlement report at 8 AM daily
	_, err = s.scheduler.Register(
		"0 8 * * *", // Daily at 8 AM
		asynq.NewTask(TypeDailySettlementReport, []byte(`{}`)),
		asynq.Queue("reports"),
	)
	if err != nil {
		logger.Error("Failed to schedule daily settlement report task", err)
	} else {
		logger.Info("Scheduled daily settlement report task", logger.Fields{
			"schedule": "daily at 8 AM",
		})
	}

	// Schedule daily reconciliation at midnight (verify solvency)
	_, err = s.scheduler.Register(
		"0 0 * * *", // Daily at midnight (00:00)
		asynq.NewTask(TypeDailyReconciliation, []byte(`{}`)),
		asynq.Queue("periodic"),
	)
	if err != nil {
		logger.Error("Failed to schedule daily reconciliation task", err)
	} else {
		logger.Info("Scheduled daily reconciliation task", logger.Fields{
			"schedule": "daily at midnight",
		})
	}
}

// Start starts the worker server and scheduler
func (s *Server) Start() error {
	logger.Info("Starting worker server...")

	// Start scheduler in a goroutine
	go func() {
		if err := s.scheduler.Run(); err != nil {
			logger.Fatal("Scheduler failed", err)
		}
	}()

	logger.Info("Scheduler started")

	// Start worker server (blocking)
	if err := s.server.Run(s.mux); err != nil {
		return fmt.Errorf("worker server failed: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the worker server and scheduler
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Info("Shutting down worker server...")

	// Shutdown scheduler
	s.scheduler.Shutdown()
	logger.Info("Scheduler stopped")

	// Shutdown worker server
	s.server.Shutdown()
	logger.Info("Worker server stopped")

	return nil
}
